package template

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActionConversion(t *testing.T) {
	assert.Equal(t, mapInvalid, Invalid.convert())
	assert.Equal(t, mapZeroValue, ZeroValue.convert())
	assert.Equal(t, mapError, Error.convert())
	assert.Equal(t, mapInvalid, (Error + 1).convert())
	assert.Equal(t, mapInvalid, AllActions.convert())
	assert.Equal(t, mapInvalid, MissingAction(0).convert())

	assert.Equal(t, Invalid, mapInvalid.convert())
	assert.Equal(t, ZeroValue, mapZeroValue.convert())
	assert.Equal(t, Error, mapError.convert())
	assert.Equal(t, missingKeyAction(0), Invalid.convert())
	assert.Panics(t, func() { _ = missingKeyAction(3).convert() })
}

func TestErrorHandling(t *testing.T) {
	err := fmt.Errorf
	type results map[MissingAction]interface{}
	type handlers []*ErrorManager

	defaultHandler := func(context *ErrorContext) (interface{}, ErrorAction) {
		if context.Name == "default" {
			context.Err = nil
			return "Zero", ResultReplaced
		}
		return nil, NoReplace
	}

	privateHandler := func(context *ErrorContext) (interface{}, ErrorAction) {
		if context.Receiver.Kind() == reflect.Struct && context.Name == "private" {
			context.Err = nil
			return "Zero", ResultReplaced
		}
		return nil, NoReplace
	}

	allHandlers := handlers{
		{defaultHandler, AllErrors, ZeroValue},
		{privateHandler, FieldError, ZeroValue},
	}
	tests := []struct {
		name, input string
		data        interface{}
		result      interface{}
		handlers    handlers
		wanted      results
		funcs       FuncMap
	}{
		{
			name:   "Missing",
			input:  "{{.missing}}",
			result: NoValue,
			wanted: results{
				Error: err(`template: t:1:2: executing "t" at <.missing>: nil data; no entry for key "missing"`),
			},
		},
		{
			name:   "Nil variable",
			input:  "{{.map.missing}}",
			data:   map[string]interface{}{"map": nil},
			result: err(`template: t:1:6: executing "t" at <.map.missing>: nil pointer evaluating interface {}.missing`),
		},
		{
			name:   "Missing interface{}",
			input:  "{{.map.missing}}",
			data:   map[string]interface{}{"map": map[string]interface{}{}},
			result: NoValue,
			wanted: results{
				Error: err(`template: t:1:6: executing "t" at <.map.missing>: map has no entry for key "missing"`),
			},
		},
		{
			name:  "Missing string",
			input: "{{.map.empty}}",
			data:  map[string]interface{}{"map": map[string]string{}},
			wanted: results{
				Invalid:   NoValue,
				Error:     err(`template: t:1:6: executing "t" at <.map.empty>: map has no entry for key "empty"`),
				ZeroValue: "",
			},
		},
		{
			name:  "Missing int",
			input: "{{.map.zero}}",
			data:  map[string]interface{}{"map": map[string]int{}},
			wanted: results{
				Invalid:   NoValue,
				Error:     err(`template: t:1:6: executing "t" at <.map.zero>: map has no entry for key "zero"`),
				ZeroValue: "0",
			},
		},
		{
			name:  "Missing bool",
			input: "{{.map.bool}}",
			data:  map[string]interface{}{"map": map[string]bool{}},
			wanted: results{
				Invalid:   NoValue,
				Error:     err(`template: t:1:6: executing "t" at <.map.bool>: map has no entry for key "bool"`),
				ZeroValue: "false",
			},
		},
		{
			name:     "Missing key with handler",
			input:    "{{.map.default}}",
			data:     map[string]interface{}{"map": map[string]interface{}{}},
			handlers: allHandlers,
			wanted: results{
				Invalid:   NoValue,
				Error:     err(`template: t:1:6: executing "t" at <.map.default>: map has no entry for key "default"`),
				ZeroValue: "Zero",
			},
		},
		{
			name:     "Missing field with handler",
			input:    "{{.Value.default}}",
			data:     struct{ Value struct{} }{},
			handlers: allHandlers,
			result:   err(`template: t:1:8: executing "t" at <.Value.default>: can't evaluate field default in type struct {}`),
			wanted:   results{ZeroValue: "Zero"},
		},
		{
			name:     "Accessing private field with handler",
			input:    "{{.private}}",
			data:     struct{ private int }{},
			handlers: allHandlers,
			result:   err(`template: t:1:2: executing "t" at <.private>: private is an unexported field of struct type struct { private int }`),
			wanted:   results{ZeroValue: "Zero"},
		},
		{
			name:     "Accessing field with arguments",
			input:    "{{.Private 1}}",
			data:     struct{ Private int }{},
			handlers: allHandlers,
			result:   err(`template: t:1:2: executing "t" at <.Private>: Private has arguments but cannot be invoked as function`),
		},
		// Testing methods
		{
			name:     "Data with method",
			input:    "{{.Upper}}",
			data:     &dataWithMethod{"Hello"},
			handlers: allHandlers,
			result:   "HELLO",
		},
		{
			name:     "Data with method and bad parameter",
			input:    "{{.Lower 1}}",
			data:     &dataWithMethod{"Hello"},
			handlers: allHandlers,
			result:   err(`template: t:1:2: executing "t" at <.Lower>: wrong number of args for Lower: want 0 got 1`),
		},
		{
			name:     "Calling method with no return",
			input:    "{{.NoReturn}}",
			data:     &dataWithMethod{},
			handlers: handlers{InvalidReturnHandler()},
			result:   "",
		},
		{
			name:     "Calling variadic method with no return",
			input:    "{{.VariadicNoReturn 0 1}}",
			data:     &dataWithMethod{},
			handlers: handlers{InvalidReturnHandler()},
			result:   "",
		},
		{
			name:     "Calling error method",
			input:    "{{.Error}}",
			data:     &dataWithMethod{},
			handlers: handlers{InvalidReturnHandler()},
			result:   err(`template: t:1:2: executing "t" at <.Error>: bang`),
		},
		{
			name:     "Calling method with 2 return values",
			input:    "{{.Tuple}}",
			data:     &dataWithMethod{},
			handlers: handlers{InvalidReturnHandler()},
			result:   "[2 two]",
		},
		{
			name:     "Calling method with 3 return values and no error",
			input:    "{{.Tuple4 ``}}",
			data:     &dataWithMethod{},
			handlers: handlers{InvalidReturnHandler()},
			result:   "[4 four true]",
		},
		{
			name:     "Calling method with 3 return values and error",
			input:    `{{.Tuple4 "bang"}}`,
			data:     &dataWithMethod{},
			handlers: handlers{InvalidReturnHandler()},
			result:   err(`template: t:1:10: executing "t" at <"bang">: bang`),
		},
		// Testing functions
		{
			name:     "Calling function with only error as return",
			input:    `{{error}}`,
			handlers: handlers{InvalidReturnHandler()},
			result:   err(`template: t:1:2: executing "t" at <error>: boom`),
			funcs:    FuncMap{"error": func() error { return fmt.Errorf("boom") }},
		},
		{
			name:     "Calling function with no return",
			input:    `{{noReturn}}`,
			handlers: handlers{InvalidReturnHandler()},
			result:   "",
			funcs:    FuncMap{"noReturn": func() {}},
		},
		{
			name:     "Calling function with no return (and undesired argument)",
			input:    `{{noReturn 0}}`,
			handlers: handlers{InvalidReturnHandler()},
			result:   err(`template: t:1:11: executing "t" at <0>: wrong number of args for noReturn: want 0 got 1`),
			funcs:    FuncMap{"noReturn": func() {}},
		},
		{
			name:     "Calling function with no return (and undesired arguments)",
			input:    `{{noReturn 0 1}}`,
			handlers: handlers{InvalidReturnHandler()},
			result:   err(`template: t:1:2: executing "t" at <noReturn>: wrong number of args for noReturn: want 0 got 2`),
			funcs:    FuncMap{"noReturn": func() {}},
		},
		{
			name:     "Calling variadic function with no return with argument",
			input:    `{{noReturn 0}}`,
			handlers: handlers{InvalidReturnHandler()},
			result:   "",
			funcs:    FuncMap{"noReturn": func(...int) {}},
		},
		{
			name:     "Calling variadic function with no return with arguments",
			input:    `{{noReturn 0 1}}`,
			handlers: handlers{InvalidReturnHandler()},
			result:   "",
			funcs:    FuncMap{"noReturn": func(...int) {}},
		},
		{
			name:     "Calling variadic function with no return and bad arguments",
			input:    `{{noReturn 0 1}}`,
			handlers: handlers{InvalidReturnHandler()},
			result:   err(`template: t:1:11: executing "t" at <0>: expected string; found 0`),
			funcs:    FuncMap{"noReturn": func(string, ...int) {}},
		},
		{
			name:     "Calling variadic function with no return and bad arguments2",
			input:    `{{noReturn 0 "1"}}`,
			handlers: handlers{InvalidReturnHandler()},
			result:   err(`template: t:1:13: executing "t" at <"1">: expected integer; found "1"`),
			funcs:    FuncMap{"noReturn": func(...int) {}},
		},
		{
			name:     "Calling function with 2 returns",
			input:    `{{two}}`,
			handlers: handlers{InvalidReturnHandler()},
			result:   "[0 zero]",
			funcs:    FuncMap{"two": func() (int, string) { return 0, "zero" }},
		},
		// Multiple values
		{
			name:  "Testing array",
			input: `{{$v := repeat "test" 5}}{{$v.value}}{{$v.Append ".txt"}}`,
			handlers: handlers{
				&ErrorManager{
					func(context *ErrorContext) (interface{}, ErrorAction) {
						if strings.Contains(context.Err.Error(), `can't evaluate field value in type []template.dataWithMethod`) {
							context.Err = nil
							return context.Receiver.Interface(), ResultAsArray
						}
						if strings.Contains(context.Err.Error(), `can't evaluate field Append in type []template.dataWithMethod`) {
							context.Err = nil
							return context.Receiver.Interface(), ResultAsArray
						}
						if strings.Contains(context.Err.Error(), `value is an unexported field of struct type template.dataWithMethod`) {
							context.Err = nil
							return context.Receiver.Interface().(dataWithMethod).value, ResultReplaced
						}
						return nil, NoReplace
					}, AllErrors, Invalid,
				},
			},
			funcs: FuncMap{
				"repeat": func(s string, count int) []dataWithMethod {
					result := make([]dataWithMethod, count)
					for i := range result {
						result[i] = dataWithMethod{fmt.Sprint(s, i+1)}
					}
					return result
				},
			},
			result: err(`template: t:1:29: executing "t" at <$v.value>: can't evaluate field value in type []template.dataWithMethod`),
			wanted: results{Invalid: "[test1 test2 test3 test4 test5][test1.txt test2.txt test3.txt test4.txt test5.txt]"}},
	}
	var filter string // = "Calling function with only error as return"
	for _, tc := range tests {
		if filter != "" && filter != tc.name {
			continue
		}
		for _, option := range []MissingAction{Invalid, ZeroValue, Error} {
			t.Run(fmt.Sprintf("%s:%s", tc.name, option), func(t *testing.T) {
				tmpl, err := New("t").ErrorManagers(tc.name, tc.handlers...).SafeFuncs(tc.funcs).Parse(tc.input)
				if err != nil {
					t.Fatalf("parse error: %s", err)
				}

				tmpl.option.missingKey = option.convert()
				buffer := new(bytes.Buffer)
				err = tmpl.Execute(buffer, tc.data)

				result := tc.result
				if tc.wanted != nil {
					if value, isSet := tc.wanted[option]; isSet {
						result = value
					}
				}

				switch expected := result.(type) {
				case error:
					assert.EqualError(t, err, expected.Error())
					assert.Equal(t, "", buffer.String())
				case string:
					assert.NoError(t, err)
					assert.Equal(t, expected, buffer.String())
				default:
					assert.Failf(t, "Unexpected", "result type %T", expected)
				}
			})
		}
	}
}

type dataWithMethod struct{ value string }

func (d *dataWithMethod) Len() int                { return len(d.value) }
func (d *dataWithMethod) Upper() string           { return strings.ToUpper(d.value) }
func (d *dataWithMethod) Lower() string           { return strings.ToLower(d.value) }
func (d *dataWithMethod) Append(s string) string  { return d.value + s }
func (d *dataWithMethod) NoReturn()               {}
func (d *dataWithMethod) Error() error            { return fmt.Errorf("bang") }
func (d *dataWithMethod) VariadicNoReturn(...int) {}
func (d *dataWithMethod) Tuple() (int, string)    { return 2, "two" }
func (d *dataWithMethod) Tuple4(format string, args ...interface{}) (int, string, bool, error) {
	var err error
	if format != "" {
		err = fmt.Errorf(format, args...)
	}
	return 4, "four", true, err
}
