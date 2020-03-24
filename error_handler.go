package template

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"sync"
)

// ErrorHandler allows registration of function that could handle templage evaluation errors.
// This give the caller opportunity to recover errors and/or add custom values other that that default actions.
func (t *Template) ErrorHandler(name string, source ErrorSource, mode MissingAction, handler ErrorHandler) *Template {
	return t.ErrorManagers(name, &ErrorManager{handler, source, mode})
}

// ErrorManagers allows registration of error handlers to manage errors.
// An error handler is a packaged error handler function with preset filters for mode and source.
func (t *Template) ErrorManagers(name string, managers ...*ErrorManager) *Template {
	t.option.addErrorManagers(name, managers)
	return t
}

// GetFuncs returns the list of function added to the template.
func (t *Template) GetFuncs(builtin bool) FuncMap {
	if builtin {
		return builtins
	}
	return t.parseFuncs
}

// Template returns the current template associated to the error context.
func (c *ErrorContext) Template() *Template { return c.state.tmpl }
func (c *ErrorContext) option() option      { return c.Template().option }
func (c *ErrorContext) ehs() errorHandlers  { return c.option().ehs }
func (c *ErrorContext) keys() []string      { return c.ehs().keys }

func (a missingKeyAction) convert() MissingAction {
	return []MissingAction{Invalid, ZeroValue, Error}[a]
}

func (a MissingAction) convert() missingKeyAction {
	if a < Invalid || a > Error {
		return mapInvalid
	}
	return []missingKeyAction{mapInvalid, mapZeroValue, mapInvalid, mapError}[a-1]
}

func (c *ErrorContext) tryRecoverNonStandardReturn() (interface{}, error) {
	ft := c.Function.Type()
	if ft.NumIn() == 1 && ft.In(0) == reflect.TypeOf(c) {
		return convertResult(c.Function.Call([]reflect.Value{reflect.ValueOf(c)}))
	}
	return nil, nil
}

func (c *ErrorContext) callActualFunc(fun reflect.Value) (interface{}, error) {
	args := make([]reflect.Value, 0, len(c.Args))
	typ := fun.Type()
	numIn := typ.NumIn()
	if typ.IsVariadic() {
		numIn--
	} else if len(c.Args) != typ.NumIn() {
		return nil, fmt.Errorf("wrong number of args for %s: want %d got %d", c.Name, typ.NumIn(), len(c.Args))
	}

	var argType reflect.Type
	for i := range c.Args {
		if i <= numIn {
			argType = typ.In(i)
		}
		if typ.IsVariadic() && i == numIn {
			argType = argType.Elem()
		}
		args = append(args, c.state.evalArg(c.Dot, argType, c.Args[i]))
	}
	return convertResult(fun.Call(args))
}

func convertResult(result []reflect.Value) (interface{}, error) {
	if len(result) == 0 {
		return "", nil
	}
	var lenResult = len(result)
	var err error
	if result[len(result)-1].Type() == errorType {
		lenResult--
		err, _ = result[lenResult].Interface().(error)
	}
	array := make([]interface{}, lenResult)
	for i := range array {
		array[i] = result[i].Interface()
	}
	if len(array) == 1 {
		return array[0], err
	}
	return array, err
}

func (o *option) addErrorManagers(name string, managers []*ErrorManager) {
	if o.ehs.handlers == nil {
		o.ehs.handlers = make(map[string][]*ErrorManager)
	}
	if len(managers) == 0 {
		delete(o.ehs.handlers, name)
	} else {
		o.ehs.handlers[name] = managers
	}

	o.ehs.keys = make([]string, 0, len(o.ehs.handlers))
	for key := range o.ehs.handlers {
		o.ehs.keys = append(o.ehs.keys, key)
	}
	sort.Strings(o.ehs.keys)
}

// CanManage returns true if the error manager can handle the kind of error.
func (h *ErrorManager) CanManage(source ErrorSource, mode MissingAction) bool {
	return source&h.Source > 0 && mode&h.Mode > 0
}

func (c *ErrorContext) invoke() (result reflect.Value, action ErrorAction) {
	for _, key := range c.keys() {
		for _, handler := range c.ehs().handlers[key] {
			if !handler.CanManage(c.Source, c.option().missingKey.convert()) {
				continue
			}
			if value, missedAction := handler.Function(c); missedAction != NoReplace {
				return reflect.ValueOf(value), missedAction
			}
		}
	}
	return
}

// SafeFuncs allows registering of non standard functions, i.e. functions with no return,
// or that multiple values.
func (t *Template) SafeFuncs(funcMap FuncMap) *Template {
	for name, fn := range funcMap {
		f := reflect.ValueOf(fn)
		if f.Kind() != reflect.Func {
			continue
		}
		ft := f.Type()
		if ft.NumOut() == 1 && ft.Out(0) != errorType || ft.NumOut() == 2 && ft.Out(1) == errorType {
			continue
		}
		funcMap[name] = func(in *ErrorContext) (interface{}, error) {
			return in.callActualFunc(f)
		}
	}
	return t.Funcs(funcMap)
}

func (s *state) tryRecoverError(rec interface{}, context *ErrorContext) reflect.Value {
	if len(s.tmpl.option.ehs.keys) > 0 {
		if rec == nil && context.Result.IsValid() && context.Result.CanInterface() && context.Result.Interface() != nil {
			return context.Result
		}
		context.state = s
		context.Err, _ = rec.(error)
		result, action := context.invoke()
		if action != NoReplace && context.Err != nil {
			s.errorf(context.Err.Error())
		}
		switch action {
		case ResultReplaced:
			return result
		case ResultAsArray:
			newResult := make([]interface{}, result.Len())
			for i := 0; i < result.Len(); i++ {
				newResult[i] = s.evalField(context.Dot, context.Name, context.Node, context.Args, context.Final, result.Index(i)).Interface()
			}
			return reflect.ValueOf(newResult)
		}
	}
	if rec != nil {
		panic(rec)
	}
	return context.Result
}

// InvalidReturnHandler returns an handler that handle function and methods not returning any result.
func InvalidReturnHandler() *ErrorManager {
	return &ErrorManager{
		Source: CallError,
		Mode:   AllActions,
		Function: func(context *ErrorContext) (interface{}, ErrorAction) {
			initRegex()
			if reCall.MatchString(context.Err.Error()) {
				var result interface{}
				result, context.Err = context.callActualFunc(context.Function)
				return result, ResultReplaced
			}
			text := context.Err.Error()
			if reArgs.MatchString(text) || reArgType.MatchString(text) {
				if result, err := context.tryRecoverNonStandardReturn(); result != nil || err != nil {
					context.Err = err
					return result, ResultReplaced
				}
			}
			return nil, NoReplace
		},
	}
}

func initRegex() {
	initReturn.Do(func() {
		reCall = regexp.MustCompile(`can't call method/function ".*" with \d+ result`)
		reArgs = regexp.MustCompile(`wrong number of args for .*: want 1 got \d+`)
		reArgType = regexp.MustCompile(`can't handle .* for arg of type \*template\.ErrorContext`)
	})
}

var (
	initReturn                sync.Once
	reCall, reArgs, reArgType *regexp.Regexp
)
