package template

// ErrorHandler represents the function type used to try to recover missing key during the template evaluation.
type ErrorHandler func(context *Context) (interface{}, ErrorAction)

// NoValue is the rendered string representation of invalid value if missingkey is set to invalid or left to default.
const NoValue = "<no value>"

type errorHandlers struct {
	handlers map[string][]*ErrorManager
	keys     []string
}

// InvalidReturnHandlers returns handlers that handle function and methods not returning any result.
func InvalidReturnHandlers() ErrorManagers {
	return ErrorManagers{
		NewErrorManager(
			func(context *Context) (interface{}, ErrorAction) {
				return context.callActualFunc(context.fun), ResultReplaced
			},
			`can't call method/function ".*" with \d+ result`).
			OnSources(Call),
		NewErrorManager(
			func(context *Context) (interface{}, ErrorAction) {
				if result, ok := context.tryRecoverNonStandardReturn(); ok {
					return result, ResultReplaced
				}
				return nil, NoReplace
			},
			`wrong number of args for .*: want 1 got \d+`,
			`can't handle .* for arg of type \*template\.Context`,
			`wrong type for value; expected \*template.Context; got .*`).
			OnSources(Call),
	}
}
