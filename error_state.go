package template

import (
	"fmt"
	"reflect"

	"github.com/jocgir/template/parse"
)

func (s *state) recover(f func(error) error) {
	var err error
	switch rec := recover().(type) {
	case error:
		err = rec
	case nil:
		err = nil
	default:
		err = fmt.Errorf("Panic %v", rec)
	}
	switch err := f(err).(type) {
	case nil:
	case ExecError:
		panic(err)
	default:
		s.errorf(err.Error())
	}
}

func (s *state) newContext(source ContextSource, err error, name string, node parse.Node, args []parse.Node,
	fun, dot, final, receiver reflect.Value, result *reflect.Value) context {
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
	if !s.hasHandlers() || err == nil && isValid(*result) {
		return err
	}
	return s.newContext(source, err, name, node, args, fun, dot, final, receiver, result).tryRecover()
}

func (s *state) format(source ContextSource, node parse.Node, iface interface{}) interface{} {
	if s.hasHandlers() {
		result := reflect.ValueOf(iface)
		if err := s.newContext(source, nil, "", node, nil, nilv, nilv, nilv, result, &result).tryRecover(); err != nil {
			s.errorf(err.Error())
		}
		return result.Interface()
	}
	return iface
}

func (s *state) hasHandlers() bool {
	return len(s.tmpl.option.ehs.handlers) > 0
}

func isValid(value reflect.Value) bool {
	return value.IsValid() && value.CanInterface() && value.Interface() != nil
}
