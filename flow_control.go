package template

import (
	"bytes"
	"fmt"
	"reflect"
)

type flowControl byte

const (
	fcFlow flowControl = iota
	fcBreak
	fcContinue
	fcReturn
)

func (fc flowControl) Error() string { return fc.String() }
func (fc flowControl) String() string {
	switch fc {
	case fcFlow:
		return "flow"
	case fcBreak:
		return "break"
	case fcContinue:
		return "continue"
	case fcReturn:
		return "return"
	default:
		panic(fmt.Errorf("Undefined flow controle type %d", fc))
	}
}

func flow(action func()) (result flowControl) {
	defer func() {
		switch rec := recover().(type) {
		case nil:
		case flowControl:
			if rec == fcReturn {
				panic(rec)
			}
			result = rec
		default:
			panic(rec)
		}

	}()
	action()
	return
}

func flowReturnValues(context *Context) interface{} {
	args := context.EvalArgs()
	if len(args) > 0 {
		if buffer, isBuffer := context.state.wr.(*bytes.Buffer); isBuffer {
			buffer.Reset()
		}
		value := reflect.ValueOf(convertResult(args))
		context.state.printValue(context.Template().Root, value)
	}
	panic(fcReturn)
}

func convertResult(result []interface{}) interface{} {
	switch len(result) {
	case 0:
		return ""
	case 1:
		return result[0]
	default:
		return result
	}
}
