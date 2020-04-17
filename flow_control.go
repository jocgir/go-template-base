package template

import (
	"fmt"
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
