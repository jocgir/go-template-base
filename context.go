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
	Map     map[string]interface{}
	context = *Context
)

func (c context) Args() []parse.Node                        { return c.args }
func (c context) Call(fun interface{}) interface{}          { return c.callInternal(fun, false) }
func (c context) ClearError()                               { c.SetError(nil) }
func (c context) Dot() reflect.Value                        { return c.dot }
func (c context) Error() error                              { return c.err }
func (c context) Errorf(format string, args ...interface{}) { c.SetError(fmt.Errorf(format, args...)) }
func (c context) ErrorText() string                         { return c.err.Error() }
func (c context) Final() reflect.Value                      { return c.final }
func (c context) Function() reflect.Value                   { return c.fun }
func (c context) Global() reflect.Value                     { return c.state.vars[0].value }
func (c context) Match(name interface{}) string             { return c.matches[fmt.Sprint(name)] }
func (c context) MemberName() string                        { return c.name }
func (c context) Mode() MissingAction                       { return c.state.tmpl.option.missingKey.convert() }
func (c context) Node() parse.Node                          { return c.node }
func (c context) Receiver() reflect.Value                   { return c.receiver }
func (c context) Result() *reflect.Value                    { return c.result }
func (c context) SetError(err error)                        { c.err = err }
func (c context) SetResult(value reflect.Value)             { *c.result = value }
func (c context) Source() ContextSource                     { return c.source }
func (c context) StackLen() int                             { return len(c.state.stack) }
func (c context) StackPeek(n int) *StackCall                { return c.state.peekStack(n) }
func (c context) Template() *Template                       { return c.state.tmpl }
func (c context) Variables() Map                            { return c.state.variables() }

func (c context) handlers() errorHandlers      { return c.Template().errorHandlers }
func (c context) key(key string) ErrorManagers { return c.handlers().managers[key] }
func (c context) keys() []string               { return c.handlers().keys }

func (c context) match(re *regexp.Regexp) bool {
	matches := re.FindStringSubmatch(c.ErrorText())
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

func (c context) EvalArgs() []interface{} {
	result := make([]interface{}, 0, len(c.Args())+1)
	t := reflect.TypeOf(result).Elem()
	for _, arg := range c.Args() {
		result = append(result, c.state.evalArg(c.Dot(), t, arg).Interface())
	}
	if c.Final() != missingVal {
		result = append(result, c.Final().Interface())
	}
	return result
}

func (c context) ArgCount() int {
	if c.Final() != missingVal {
		return len(c.Args()) + 1
	}
	return len(c.Args())
}

func (c context) callInternal(function interface{}, injectSelf bool) interface{} {
	var actualFunc reflect.Value
	switch ft := function.(type) {
	case nil:
		actualFunc = c.fun
	case reflect.Value:
		actualFunc = ft
	default:
		actualFunc = reflect.ValueOf(function)
	}

	args := make([]reflect.Value, 0, c.ArgCount()+1)
	first := 0
	if injectSelf {
		first = 1
		args = append(args, reflect.ValueOf(c))
	}
	typ := actualFunc.Type()
	numIn := typ.NumIn()

	if injectSelf && typ.NumIn() == 1 && !typ.IsVariadic() {
		return c.convertResult(actualFunc.Call(args))
	}

	if typ.IsVariadic() {
		numIn--
	} else if c.ArgCount()+first != typ.NumIn() {
		c.Errorf("wrong number of args for %s: want %d got %d", c.MemberName(), typ.NumIn()-first, c.ArgCount())
		return nil
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
		if i < len(c.Args()) {
			arg = c.state.evalArg(c.Dot(), argType, c.Args()[i])
		} else {
			arg = c.Final()
		}
		args = append(args, arg)
	}
	return c.convertResult(actualFunc.Call(args))
}

func (c context) convertResult(result []reflect.Value) interface{} {
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

func (c context) tryRecover() error {
	action := func() (result ErrorAction) {
		for _, key := range c.keys() {
			for _, handler := range c.key(key) {
				if !handler.CanManage(c) {
					continue
				}
				if value, action := handler.fun(c); action != NoReplace {
					c.SetResult(reflect.ValueOf(value))
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
			newResult[i] = c.state.evalField(c.Dot(), c.MemberName(), c.Node(), c.Args(), c.Final(), c.Result().Index(i)).Interface()
		}
		c.SetResult(reflect.ValueOf(newResult))
	}
	return c.Error()
}
