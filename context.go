package template

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/jocgir/template/parse"
)

// Context gives the current context error information to the error handler.
type Context struct {
	state                     *state
	source                    ContextSource
	err                       error
	name                      string
	node                      parse.Node
	args                      []parse.Node
	fun, dot, final, receiver reflect.Value
	result                    *reflect.Value
	matches                   map[string]string
}

var (
	nilv        = reflect.Value{}
	contextType = reflect.TypeOf(&Context{})
)

type (
	// Map represent a generic map with strings as keys.
	Map map[string]interface{}
)

// ClearError is used by error managers to indicate that an error condition has been solved.
func (c *Context) ClearError() { c.SetError(nil) }

// Current returns the current context designated by dot {{ . }}.
func (c *Context) Current() reflect.Value { return c.dot }

// PipelineArg returns the argument supplied through pipeline.
func (c *Context) PipelineArg() reflect.Value { return c.final }

// Global returns the global context designated by string {{ $ }}.
func (c *Context) Global() reflect.Value { return c.state.vars[0].value }

// Match returns the designated group in error matching regex.
// It is under the user responsability to supply a valid regular expression to match the error text.
// Within that expression, user can defines anonymous subexpression group using (.*) or
// named subexpression (?P<name>.*).
// Context.Match(0) will return the whole match.
// Context.Match(n) will return the nth submatch group.
// Context.Match("name") will return the named submatch group.
func (c *Context) Match(name interface{}) string { return c.matches[fmt.Sprint(name)] }

// MemberName returns the faulting member that created the context, either field name or method name.
func (c *Context) MemberName() string { return c.name }

// Node returns the current node that's being processed by go template.
func (c *Context) Node() parse.Node { return c.node }

// Receiver returns the current object receiver (i.e. the object on witch a field or a method is called upon).
func (c *Context) Receiver() reflect.Value { return c.receiver }

// Recover allows user writing function dealing with context to ensure that any unmanaged error will be
// handled properly. Simply add the following call at the begining of your function:
//   func(context *Context) result {
//     defer context.Recover()
//     ...
//   }
func (c *Context) Recover() { c.state.recovered(recover(), nil) }

// Result returns the current result value that will be returned by the context.
func (c *Context) Result() *reflect.Value { return c.result }

// Error returns the current error condition associated with the context.
func (c *Context) Error() error { return c.err }

// Errorf allows handlers to set the current error state by formatting the error message.
func (c *Context) Errorf(format string, args ...interface{}) { c.SetError(fmt.Errorf(format, args...)) }

// SetError allows handlers to set the current error state.
func (c *Context) SetError(err error) { c.err = err }

// StackLen returns the current stack length.
func (c *Context) StackLen() int { return len(c.state.stack) }

// StackPeek returns the nth value in the template calling stack (0 meaning the current function).
func (c *Context) StackPeek(n int) *StackCall { return c.state.peekStack(n) }

// Template returns the current template being evaluated.
func (c *Context) Template() *Template { return c.state.tmpl }

// Variables returns the current variable values available through {{ $variable }}.
func (c *Context) Variables() Map { return c.state.variables() }

// ArgCount return the total number of argument supplied to the context including
// the piped argument if there is.
func (c *Context) ArgCount() int {
	if c.PipelineArg() != missingVal {
		return len(c.args) + 1
	}
	return len(c.args)
}

// EvalArgs returns an []interface{} from the supplied arguments.
// If there is a piped argument, it will be added at the end.
// If there is a receiver, it will be inserted as the first argument.
func (c *Context) EvalArgs() []interface{} {
	result := make([]interface{}, 0, len(c.args)+1)
	t := reflect.TypeOf(result).Elem()
	for i, arg := range c.args {
		var value interface{}
		if i == 0 && c.Receiver().IsValid() {
			value = c.Receiver().Interface()
		} else {
			value = c.state.evalArg(c.dot, t, arg).Interface()
		}
		result = append(result, value)
	}
	if c.PipelineArg().IsValid() && c.PipelineArg() != missingVal {
		result = append(result, c.PipelineArg().Interface())
	}
	return result
}

// Call invokes the supplied function with the arguments supplied in the context.
// If function is nil, the function attached to the context will be called.
// The function an be either a reflect.Value, a function prototype or the name of a registered function.
func (c *Context) Call(function interface{}) interface{} {
	if result, called := c.TryCall(function); !called {
		panic(fmt.Errorf("Unable to find %v", function))
	} else {
		return result
	}
}

// TryCall tries to invokes the supplied function with the arguments supplied in the context.
// If it is not possible to invoke the function, a false value will be returned as second return value.
// If the function exist, the second value will be true even if the call fails. Check Error() method to
// get the result of the call.
func (c *Context) TryCall(function interface{}) (interface{}, bool) {
	var actualFunc reflect.Value
	switch ft := function.(type) {
	case nil:
		actualFunc = c.fun
	case reflect.Value:
		actualFunc = ft
	case string:
		actualFunc = reflect.ValueOf(c.Template().GetBuiltinsMap()[ft])
		if !actualFunc.IsValid() {
			actualFunc = reflect.ValueOf(c.Template().GetFuncsMap()[ft])
			if !actualFunc.IsValid() {
				return nil, false
			}
		}
	default:
		actualFunc = reflect.ValueOf(function)
	}

	typ := actualFunc.Type()
	args := make([]reflect.Value, 0, c.ArgCount()+1)
	first := 0
	injectSelf := typ.NumIn() > 0 && typ.In(0) == reflect.TypeOf(c)
	if injectSelf {
		first = 1
		args = append(args, reflect.ValueOf(c))
	}
	numIn := typ.NumIn()

	if injectSelf && typ.NumIn() == 1 && !typ.IsVariadic() {
		return c.convertResult(actualFunc.Call(args)), true
	}

	if typ.IsVariadic() {
		numIn--
	} else if c.ArgCount()+first != typ.NumIn() {
		c.Errorf("wrong number of args for %s: want %d got %d", c.MemberName(), typ.NumIn()-first, c.ArgCount())
		return nil, true
	}

	var argType reflect.Type
	for i := 0; i < c.ArgCount(); i++ {
		if i+first <= numIn {
			argType = typ.In(i + first)
		}
		if typ.IsVariadic() && i+first == numIn {
			argType = argType.Elem()
		}
		var arg reflect.Value
		if i < len(c.args) {
			if i+first == 0 && c.Receiver().IsValid() {
				// If there is a receiver, we use it as the first argument
				arg = c.Receiver()
				if !arg.Type().AssignableTo(argType) && arg.Type().ConvertibleTo(argType) {
					// We ensure that the type is compatible with the desired type and there
					// is no data loss (or transformation) during the conversion
					if na := arg.Convert(argType); fmt.Sprint(na) == fmt.Sprint(arg) {
						arg = na
					}
				}
			} else {
				arg = c.state.evalArg(c.dot, argType, c.args[i])
			}
		} else {
			arg = c.PipelineArg()
		}
		args = append(args, arg)
	}
	return c.convertResult(actualFunc.Call(args)), true
}

func (c *Context) convertResult(result []reflect.Value) interface{} {
	if len(result) == 0 {
		c.ClearError()
		return ""
	}
	var lenResult = len(result)
	if result[len(result)-1].Type() == errorType {
		lenResult--
		c.err, _ = result[lenResult].Interface().(error)
	} else {
		c.ClearError()
	}
	array := make([]interface{}, lenResult)
	for i := range array {
		array[i] = result[i].Interface()
	}
	if len(array) == 1 {
		return array[0]
	}
	return array
}

func (c *Context) tryRecover() error {
	action := func() (result ErrorAction) {
		for _, key := range c.keys() {
			for _, handler := range c.key(key) {
				if !handler.CanManage(c) {
					continue
				}
				if value, action := handler.fun(c); action != NoReplace {
					*c.result = reflect.ValueOf(value)
					result = action
					if c.Error() == nil {
						return
					}
				}
			}
		}
		return
	}()

	if action == ResultAsArray {
		newResult := make([]interface{}, c.Result().Len())
		for i := 0; i < c.Result().Len(); i++ {
			newResult[i] = c.state.evalField(c.dot, c.MemberName(), c.Node(), c.args, c.PipelineArg(), c.Result().Index(i)).Interface()
		}
		*c.result = reflect.ValueOf(newResult)
	}
	return c.Error()
}

func (c *Context) handlers() errorHandlers      { return c.Template().errorHandlers }
func (c *Context) key(key string) ErrorManagers { return c.handlers().managers[key] }
func (c *Context) keys() []string               { return c.handlers().keys }

func (c *Context) match(re *regexp.Regexp) bool {
	matches := re.FindStringSubmatch(c.err.Error())
	if len(matches) == 0 {
		return false
	}
	c.matches = make(map[string]string, len(matches))
	subexp := re.SubexpNames()
	for i, match := range matches {
		c.matches[fmt.Sprint(i)] = match
		if subexp[i] != "" {
			c.matches[subexp[i]] = match
		}
	}
	return true
}

type errorHandlers struct {
	managers map[string]ErrorManagers
	keys     []string
}
