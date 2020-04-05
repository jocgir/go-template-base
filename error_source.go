package template

import "strings"

// ErrorSource defines the type of error that could be managed by the error handlers.
type ErrorSource byte

const (
	// FieldError represent an error that occurred in evalField.
	FieldError ErrorSource = 1 << iota
	// CallError an error that occurred in evalCall.
	CallError
)

func (es ErrorSource) String() string {
	var result []string
	if es == 0 {
		return "None"
	}
	if es&FieldError != 0 {
		result = append(result, "Field")
	}
	if es&CallError != 0 {
		result = append(result, "Call")
	}
	if len(result) == 0 {
		return "Undefined"
	}
	return strings.Join(result, ",")
}

// IsSet check whether or not the source has the specified value set.
func (es ErrorSource) IsSet(value ErrorSource) bool { return es|value != 0 }
