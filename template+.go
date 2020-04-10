package template

import (
	"reflect"
	"sort"
)

// ErrorManagers allows registration of error handlers to manage errors.
// An error handler is a packaged error handler function with preset filters for mode and source.
func (t *Template) ErrorManagers(name string, managers ...*ErrorManager) *Template {
	if t.option.errorHandlers.managers == nil {
		t.option.errorHandlers.managers = make(map[string]ErrorManagers)
	}
	if len(managers) == 0 {
		delete(t.option.errorHandlers.managers, name)
	} else {
		t.option.errorHandlers.managers[name] = managers
	}

	t.option.errorHandlers.keys = make([]string, 0, len(t.option.errorHandlers.managers))
	for key := range t.option.errorHandlers.managers {
		t.option.errorHandlers.keys = append(t.option.errorHandlers.keys, key)
	}
	sort.Strings(t.option.errorHandlers.keys)
	return t
}

// GetBuiltins returns the sorted list of builtin functions name added to the template.
func (t *Template) GetBuiltins() []string { return getSortedName(t.GetBuiltinsMap()) }

// GetBuiltinsMap returns the list of builtin functions added to the template.
func (t *Template) GetBuiltinsMap() FuncMap { return builtins }

// GetFuncs returns the sorted list of function names added to the template.
func (t *Template) GetFuncs() []string { return getSortedName(t.GetFuncsMap()) }

// GetFuncsMap returns the list of function added to the template.
func (t *Template) GetFuncsMap() FuncMap { return t.parseFuncs }

// ExtraFuncs allows registering of non standard functions, i.e. functions with no return,
// or that returns multiple values.
//
// Using this method to register your functions also handle calling object methods that are
// either not returning values or return more that one value.
//
// It will also register two more functions:
//   eval: Allows dynamic evaluation of the supplied strings as template.
//   trap: Will catch any error and return it as an error object instead of stopping the template processing.
//
// Functions with no return will be modified to return an empty string.
// Functions with more than one return value will be modified to return an array of interface{}.
// Functions with just an error value will be modified to return an empty string if there is no error.
func (t *Template) ExtraFuncs(funcMap FuncMap) *Template {
	var replaced FuncMap
	for name := range funcMap {
		fn := funcMap[name]
		if reflect.ValueOf(fn).Kind() != reflect.Func {
			continue
		}

		if ft := reflect.TypeOf(fn); ft.NumOut() == 1 && ft.Out(0) != errorType || ft.NumOut() == 2 && ft.Out(1) == errorType {
			continue
		}
		if replaced == nil {
			// We duplicate the supplied func map to not alter the original
			replaced = make(FuncMap, len(funcMap))
			for key := range funcMap {
				replaced[key] = funcMap[key]
			}
		}
		replaced[name] = func(in *Context) (interface{}, error) {
			result := in.Call(fn)
			return result, in.Error()
		}
	}

	if t.GetFuncsMap()["trap"] == nil {
		// We register the trap function to handle errors
		t.ErrorManagers("1_ContextHandlers", contextHandlers()...)
		t.ErrorManagers("2_CallFailHandler", callFailHandler())
		t.Funcs(FuncMap{
			"eval": eval,
			"trap": func(result interface{}) interface{} { return result },
		})
	}

	if replaced != nil {
		return t.Funcs(replaced)
	}
	return t.Funcs(funcMap)
}

func getSortedName(funcs FuncMap) []string {
	list := make([]string, 0, len(funcs))
	for name := range funcs {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

func contextHandlers() ErrorManagers {
	return ErrorManagers{
		NewErrorManager(
			func(context *Context) (interface{}, ErrorAction) {
				return context.Call(nil), ResultReplaced
			},
			`can't call method/function "(?P<function>.*)" with (?P<result>\d+) result`).
			OnSources(Call),
		NewErrorManager(
			func(context *Context) (interface{}, ErrorAction) {
				if ft := context.Function().Type(); ft.NumIn() > 0 && ft.In(0) == reflect.TypeOf(context) {
					return context.callInternal(nil, true), ResultReplaced
				}
				return nil, NoReplace
			},
			`wrong number of args for (?P<function>.*): want (?P<want>\d+) got (?P<got>\d+)`,
			`can't handle .* for arg of type \*template\.Context`,
			`wrong type for value; expected \*template.Context; got .*`).
			OnSources(Call),
	}
}

func callFailHandler() *ErrorManager {
	return NewErrorManager(
		func(context *Context) (interface{}, ErrorAction) {
			if context.StackLen() > 1 && context.StackPeek(1).Name == "trap" {
				context.ClearError()
				return context.Match("error"), ResultReplaced
			}
			return nil, NoReplace
		},
		`error calling (?P<function>.*): (?P<error>.*)`,
		`(?P<error>.*)`,
	).OnSources(Call)
}

type errorHandlers struct {
	managers map[string]ErrorManagers
	keys     []string
}
