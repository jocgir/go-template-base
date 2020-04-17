package template

import (
	"fmt"
	"reflect"

	"github.com/jocgir/template/parse"
)

// StackCall returns information about a stack element.
type StackCall struct {
	Name     string
	Function reflect.Type
}

func (s *state) recover(f func(error) error) { s.recovered(recover(), f) }

func (s *state) recovered(rec interface{}, f func(error) error) {
	var err = asError(rec)
	if f != nil {
		err = f(err)
	}
	switch err := err.(type) {
	case nil:
	case ExecError, flowControl:
		panic(err)
	default:
		s.errorf(err.Error())
	}
}

func (s *state) newContext(source ContextSource, err error, name string, node parse.Node, args []parse.Node,
	fun, dot, final, receiver reflect.Value, result *reflect.Value) *Context {
	return &Context{
		source:   source,
		state:    s,
		err:      err,
		name:     name,
		node:     node,
		args:     args,
		result:   result,
		fun:      fun,
		dot:      dot,
		final:    final,
		receiver: receiver,
	}
}

func (s *state) result(source ContextSource, err error, name string, node parse.Node, args []parse.Node,
	fun, dot, final, receiver reflect.Value, result *reflect.Value) error {
	if !s.hasErrorManagers() || err == nil && isValid(*result) {
		return err
	}
	return s.newContext(source, err, name, node, args, fun, dot, final, receiver, result).tryRecover()
}

func (s *state) invokeWithContext(name string, node parse.Node, args []parse.Node,
	fun, dot, final, receiver reflect.Value, result *reflect.Value) {
	context := s.newContext(CallContext, nil, name, node, args, fun, dot, final, receiver, result)
	defer s.recover(nil)
	*result = reflect.ValueOf(context.Call(nil))
	if err := context.Error(); err != nil {
		panic(err)
	}
}

func (s *state) format(source ContextSource, node parse.Node, iface interface{}) interface{} {
	if s.hasErrorManagers() {
		result := reflect.ValueOf(iface)
		if err := s.newContext(source, nil, "", node, nil, nilv, nilv, nilv, result, &result).tryRecover(); err != nil {
			s.errorf(err.Error())
		}
		return result.Interface()
	}
	return iface
}

func (s *state) hasErrorManagers() bool     { return len(s.tmpl.errorHandlers.managers) > 0 }
func (s *state) peekStack(n int) *StackCall { return s.stack[len(s.stack)-n-1] }
func (s *state) errorHandled(err error) bool {
	if _, isFlowControl := err.(flowControl); isFlowControl {
		return true
	}
	return len(s.stack) > 1 && s.peekStack(1).Name == "trap"
}

func (s *state) variables() Map {
	result := make(Map, len(s.vars)-1)
	for _, v := range s.vars[1:] {
		result[v.name] = v.value
	}
	return result
}

func (s *state) pushStack(name string, fun reflect.Type) {
	s.stack = append(s.stack, &StackCall{name, fun})
}

func (s *state) popStack() (result *StackCall) {
	last := len(s.stack) - 1
	result = s.stack[last]
	s.stack = s.stack[:last]
	return
}

func isValid(value reflect.Value) bool {
	return value.IsValid() && value.CanInterface() && value.Interface() != nil
}

func asError(err interface{}) error {
	switch err := err.(type) {
	case error:
		return err
	case nil:
		return nil
	default:
		return fmt.Errorf("Panic %v", err)
	}
}
