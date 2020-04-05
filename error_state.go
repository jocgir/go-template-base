package template

import "reflect"

func (s *state) tryRecoverError(rec interface{}, ec *ErrorContext) reflect.Value {
	if len(s.tmpl.option.ehs.keys) > 0 {
		if rec == nil && ec.result.IsValid() && ec.result.CanInterface() && ec.result.Interface() != nil {
			return ec.result
		}
		ec.state = s
		ec.err, _ = rec.(error)
		if result, action := ec.invoke(); action != NoReplace {
			if ec.err != nil {
				s.errorf(ec.err.Error())
			}
			switch action {
			case ResultReplaced:
				return result
			case ResultAsArray:
				newResult := make([]interface{}, result.Len())
				for i := 0; i < result.Len(); i++ {
					newResult[i] = s.evalField(ec.dot, ec.name, ec.node, ec.args, ec.final, result.Index(i)).Interface()
				}
				return reflect.ValueOf(newResult)
			}
		}
	}
	if rec != nil {
		panic(rec)
	}
	return ec.result
}
