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

var nilv = reflect.Value{}

type context = *Context

func (c context) Args() []parse.Node                        { return c.args }
func (c context) Call(args []reflect.Value) []reflect.Value { return c.fun.Call(args) }
func (c context) ClearError()                               { c.SetError(nil) }
func (c context) Dot() reflect.Value                        { return c.dot }
func (c context) Error() error                              { return c.err }
func (c context) Errorf(format string, args ...interface{}) { c.SetError(fmt.Errorf(format, args...)) }
func (c context) ErrorText() string                         { return c.err.Error() }
func (c context) Final() reflect.Value                      { return c.final }
func (c context) Function() reflect.Type                    { return c.fun.Type() }
func (c context) MemberName() string                        { return c.name }
func (c context) Node() parse.Node                          { return c.node }
func (c context) StackLen() int                             { return len(c.state.stack) }
func (c context) StackPeek(n int) *StackCall                { return c.state.peekStack(n) }
func (c context) Receiver() reflect.Value                   { return c.receiver }
func (c context) Result() *reflect.Value                    { return c.result }
func (c context) SetError(err error)                        { c.err = err }
func (c context) SetResult(value reflect.Value)             { *c.result = value }
func (c context) Source() ContextSource                     { return c.source }
func (c context) Template() *Template                       { return c.state.tmpl }
func (c context) Match(name interface{}) string             { return c.matches[fmt.Sprint(name)] }

func (c context) ehs() errorHandlers { return c.option().ehs }
func (c context) keys() []string     { return c.ehs().keys }
func (c context) option() option     { return c.Template().option }

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

func (c context) ArgCount() int {
	if c.Final() != missingVal {
		return len(c.Args()) + 1
	}
	return len(c.Args())
}

func (c context) tryRecoverNonStandardReturn() (interface{}, bool) {
	if ft := c.Function(); ft.NumIn() == 1 && ft.In(0) == reflect.TypeOf(c) {
		return c.convertResult(c.Call([]reflect.Value{reflect.ValueOf(c)})), true
	}
	return nil, false
}

func (c context) callActualFunc(fun reflect.Value) interface{} {
	args := make([]reflect.Value, 0, c.ArgCount())
	typ := fun.Type()
	numIn := typ.NumIn()

	if typ.IsVariadic() {
		numIn--
	} else if c.ArgCount() != typ.NumIn() {
		c.Errorf("wrong number of args for %s: want %d got %d", c.MemberName(), typ.NumIn(), c.ArgCount())
		return nil
	}

	var argType reflect.Type
	for i := 0; i < c.ArgCount(); i++ {
		if i <= numIn {
			argType = typ.In(i)
		}
		if typ.IsVariadic() && i == numIn {
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
	return c.convertResult(fun.Call(args))
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
	action := func() ErrorAction {
		for _, key := range c.keys() {
			for _, handler := range c.ehs().handlers[key] {
				if !handler.CanManage(c) {
					continue
				}
				if value, action := handler.fun(c); action != NoReplace {
					c.SetResult(reflect.ValueOf(value))
					return action
				}
			}
		}
		return NoReplace
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
