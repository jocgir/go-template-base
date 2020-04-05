package template

import (
	"fmt"
	"reflect"

	"github.com/jocgir/template/parse"
)

// ErrorContext gives the current context error information to the error handler.
type ErrorContext struct {
	state                             *state
	source                            ErrorSource
	err                               error
	name                              string
	node                              parse.Node
	args                              []parse.Node
	fun, result, dot, final, receiver reflect.Value
}

type errorContext = *ErrorContext

func (c errorContext) Template() *Template     { return c.state.tmpl }
func (c errorContext) Source() ErrorSource     { return c.source }
func (c errorContext) Error() error            { return c.err }
func (c errorContext) MemberName() string      { return c.name }
func (c errorContext) Node() parse.Node        { return c.node }
func (c errorContext) Args() []parse.Node      { return c.args }
func (c errorContext) Function() reflect.Type  { return c.fun.Type() }
func (c errorContext) Result() reflect.Value   { return c.result }
func (c errorContext) Dot() reflect.Value      { return c.dot }
func (c errorContext) Final() reflect.Value    { return c.final }
func (c errorContext) Receiver() reflect.Value { return c.receiver }
func (c errorContext) ClearError()             { c.err = nil }
func (c errorContext) option() option          { return c.Template().option }
func (c errorContext) ehs() errorHandlers      { return c.option().ehs }
func (c errorContext) keys() []string          { return c.ehs().keys }

func (c errorContext) Errorf(format string, args ...interface{}) {
	c.err = fmt.Errorf(format, args...)
}

func (c errorContext) invoke() (result reflect.Value, action ErrorAction) {
	for _, key := range c.keys() {
		for _, handler := range c.ehs().handlers[key] {
			if !handler.CanManage(c) {
				continue
			}
			if value, missedAction := handler.fun(c); missedAction != NoReplace {
				return reflect.ValueOf(value), missedAction
			}
		}
	}
	return
}

func (c errorContext) tryRecoverNonStandardReturn() (interface{}, bool) {
	ft := c.fun.Type()
	if ft.NumIn() == 1 && ft.In(0) == reflect.TypeOf(c) {
		return c.convertResult(c.fun.Call([]reflect.Value{reflect.ValueOf(c)})), true
	}
	return nil, false
}

func (c errorContext) callActualFunc(fun reflect.Value) interface{} {
	args := make([]reflect.Value, 0, len(c.args))
	typ := fun.Type()
	numIn := typ.NumIn()
	if typ.IsVariadic() {
		numIn--
	} else if len(c.args) != typ.NumIn() {
		c.Errorf("wrong number of args for %s: want %d got %d", c.name, typ.NumIn(), len(c.args))
		return nil
	}

	var argType reflect.Type
	for i := range c.args {
		if i <= numIn {
			argType = typ.In(i)
		}
		if typ.IsVariadic() && i == numIn {
			argType = argType.Elem()
		}
		args = append(args, c.state.evalArg(c.dot, argType, c.args[i]))
	}
	return c.convertResult(fun.Call(args))
}

func (c errorContext) convertResult(result []reflect.Value) interface{} {
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
