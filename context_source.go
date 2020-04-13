package template

import "strings"

// ContextSource defines the type of error that could be managed by the error handlers.
type ContextSource byte

const (
	// FieldError indicates that the context has been created while evaluating field.
	FieldError ContextSource = 1 << iota
	// CallContext indicates that the context has been created while evaluating function call requiring *Context argument.
	CallContext
	// CallError indicates that the context has been created on error while evaluating function call.
	CallError
	// Print indicates that the context has been created while evaluating object without String() method.
	Print
	// Call indicates that the context has been created while evaluating function call (context or error).
	Call = CallContext | CallError
)

func (s ContextSource) String() string {
	var result []string
	if s == 0 {
		return "None"
	}
	if s&FieldError != 0 {
		result = append(result, "FieldError")
	}
	if s&CallError != 0 {
		result = append(result, "CallError")
	}
	if s&Print != 0 {
		result = append(result, "Print")
	}
	if len(result) == 0 {
		return "Undefined"
	}
	return strings.Join(result, ",")
}

// IsSet check whether or not the source has the specified value set.
func (s ContextSource) IsSet(value ContextSource) bool { return s|value != 0 }
