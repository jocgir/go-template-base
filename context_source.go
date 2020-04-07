package template

import "strings"

// ContextSource defines the type of error that could be managed by the error handlers.
type ContextSource byte

const (
	// Field indicates that the context has been created while evaluating field.
	Field ContextSource = 1 << iota
	// Call indicates that the context has been created while evaluating function call.
	Call
	// Print indicates that the context has been created while evaluating object without String() method.
	Print
)

func (s ContextSource) String() string {
	var result []string
	if s == 0 {
		return "None"
	}
	if s&Field != 0 {
		result = append(result, "Field")
	}
	if s&Call != 0 {
		result = append(result, "Call")
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
