package template

import (
	"reflect"

	"github.com/jocgir/template/parse"
)

// ErrorHandler represents the function type used to try to recover missing key during the template evaluation.
type ErrorHandler func(context *ErrorContext) (interface{}, ErrorAction)

// ErrorAction defines the action done by external handler when managing missing key.
type ErrorAction uint8

//go:generate stringer -type=ErrorAction -output generated_actions.go
const (
	// NoReplace is returned if the external handler has not been able to fix the missing key.
	NoReplace ErrorAction = iota
	// ResultReplaced is returned if the external handler returned a valid replacement for the missing key.
	ResultReplaced
	// ResultAsArray is returned if the external handler returned an array on which we should apply the missing key.
	ResultAsArray
)

// MissingAction is the public representation of the private missingKeyAction
type MissingAction uint8

//go:generate stringer -type=MissingAction -output generated_options.go
const (
	Invalid    MissingAction = 1 << iota // Return an invalid reflect.Value.
	ZeroValue                            // Return the zero value for the map element.
	Error                                // Error out
	AllActions = ^MissingAction(0)
)

// ErrorSource defines the type of error that could be managed by the error handlers.
type ErrorSource byte

//go:generate stringer -type=ErrorSource -output generated_errors.go
const (
	// FieldError represent an error that occurred in evalField.
	FieldError ErrorSource = 1 << iota
	// CallError an error that occurred in evalCall.
	CallError
	// AllErrors include all error sources.
	AllErrors = ^ErrorSource(0)
)

// NoValue is the rendered string representation of invalid value if missingkey is set to invalid or left to default.
const NoValue = "<no value>"

// ErrorContext gives the current context error information to the error handler.
type ErrorContext struct {
	state                                  *state
	Source                                 ErrorSource
	Err                                    error
	Name                                   string
	Node                                   parse.Node
	Args                                   []parse.Node
	Function, Result, Dot, Final, Receiver reflect.Value
}

// ErrorManager represents a pre-packaged ErrorHandler function.
type ErrorManager struct {
	Function ErrorHandler
	Source   ErrorSource
	Mode     MissingAction
}

type errorHandlers struct {
	handlers map[string][]*ErrorManager
	keys     []string
}
