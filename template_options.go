package template

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

// MissingMode returns the missing action mode currently set on template.
func (t *Template) MissingMode() MissingAction { return t.option.missingKey.convert() }

// Options returns the option values that are enabled.
func (t *Template) Options() Option { return t.options }

// Option sets options for the template.
// Options can be designed by strings as in OptionDeprecated or they can
// set by providing one of MissingAction value:
//   template.Option(template.Default)
//   template.Option(template.Error)
//   template.Option(template.ZeroValue)
//   template.Option(template.Invalid)
//   // You can still use the previous way of specifying options
//   template.Option("missingkey=zero")
//
// You can also provide other options to enable extended template features:
//   template.Option(template.FunctionsAsMethods)
//   template.Option(template.FunctionsWithContext)
//   template.Option(template.NonStandardResults)
//   template.Option(template.Trap)
//   template.Option(template.Eval)
//
// Many options can be specified at once:
//   template.Option(tenplate.ZeroValue, template.Trap, template.Eval)
//   // Compatible types can also be combined with logical or
//   template.Option(tenplate.ZeroValue, template.Trap | template.Eval)
//   // It is also possible to enable all extended features at once
//   template.Option(tenplate.Default, template.AllOptions)
func (t *Template) Option(options ...interface{}) *Template {
	t.init()
	for _, opt := range options {
		switch opt := opt.(type) {
		case string:
			t.setOption(opt)
		case MissingAction:
			t.option.missingKey = opt.convert()
		case Option:
			t.setTemplateOption(opt)
		}
	}
	return t
}

// Option is used to enable additional options on template.
type Option int

const (
	// FunctionsAsMethods let regular functions to be called as if they are methods of the first argument supplied.
	//
	// It is the indented to act as the opposite of the pipeline mechanism which consider the pipeline argument as
	// the last parameter while Jinja template (Python) consider the piped argument as the first one.
	//
	// Ex:
	//   {{ index $map $key }}   Regular call
	//   {{ $key | index $map }} Piped version
	//   {{ $map.index $key }}   Method version
	FunctionsAsMethods Option = 1 << iota

	// FunctionsWithContext allows user to provide custom function that support custom handling of parameters.
	//
	// The provided function must have the following signature:
	//   func UserFunction(context *template.Context) result
	// The result can be of any type.
	//
	// Then, if the user can call the registered function with any parameters without causing an error. The
	// parameters will have to be interpreted by the user function.
	//
	// Note that this option is automatically enabled if you register a non standard function using the ExtraFuncs
	// method instead of the standard Funcs method.
	FunctionsWithContext

	// PublicFunctions enable recognition of function starting with capital letter even if the original
	// function only exist with lowercase first letter.
	//
	// It is the indented to act as the opposite of the pipeline mechanism which consider the pipeline argument as
	// the last parameter while Jinja template (Python) consider the piped argument as the first one.
	//
	// Ex:
	//   {{ index $map $key }}   Normal function call
	//   {{ Index $map $key }}   The capital version is also working
	PublicFunctions

	// Trap adds the function 'trap' that allows user to catch the return of a failing function call and
	// and continue the template evaluation.
	//
	// {{ if not trap custom_func }}
	// {{ end }}
	Trap

	// Eval option add the function 'eval' to the template available functions. Using that function,
	// a user is able to dynamically add build additional templates during the template execution and
	// execute them.
	//
	// {{ value := "test" }}
	// {{ eval "Hello {{ $value }}" }}
	Eval

	// FlowControl option enables functions to control the execution flow within the template:
	//   {{ break }}    is used within range block to quit the loop
	//   {{ continue }} is used within range block to skip the rest of the block and go to next element
	//   {{ return }}   is used to quit the current template
	FlowControl

	// NonStandardResults enables functions and methods to have no return or more than one returned values.
	// Note that this is simply an alias to FunctionsWithContext and that it is automatically enabled when
	// registering non standard functions with ExtraFuncs method. However, it is required to activate that
	// option for non standard methods returns.
	NonStandardResults = FunctionsWithContext

	// AllOptions enables all available options.
	AllOptions = FlowControl<<2 - 1
)

// List returns an array with all options available.
func (o Option) List() []Option {
	result := make([]Option, 0, reflect.TypeOf(o).Bits())
	for current := Option(1); current <= Option(AllOptions); current <<= 1 {
		if o&current != 0 {
			result = append(result, current)
		}
	}
	return result
}

func (o Option) String() string {
	var result []string
	for _, o := range o.List() {
		var s string
		switch o {
		case FunctionsAsMethods:
			s = "FunctionsAsMethods"
		case FunctionsWithContext:
			s = "FunctionsWithContext"
		case PublicFunctions:
			s = "PublicFunctions"
		case Trap:
			s = "Trap"
		case Eval:
			s = "Eval"
		case FlowControl:
			s = "FlowControl"
		default:
			s = fmt.Sprint(int(o))
		}
		result = append(result, s)
	}
	return strings.Join(result, " ")
}

const (
	// FuncsAsMethodsID is the ID used to register the FunctionsAsMethods handler.
	FuncsAsMethodsID = "^0_FuncsAsMethods"
	// PublicFuncsID is the ID used to register the PublicFunctions handler.
	PublicFuncsID = "^1_PublicFuncs"
	// ContextID is the ID used to register the FunctionsWithContext handler.
	ContextID = "^2_ContextHandlers"
	// CallFailID is the ID used to register the trap handler.
	CallFailID = "^3_CallFailHandler"
)

func (t *Template) setTemplateOption(opt Option) {
	t.options |= opt

	if opt&FunctionsAsMethods != 0 {
		t.ErrorManagers(FuncsAsMethodsID, functionsAsMethods)
	}

	if opt&PublicFunctions != 0 {
		t.ErrorManagers(PublicFuncsID, publicFunctions)
	}

	if opt&FunctionsWithContext != 0 {
		t.ErrorManagers(ContextID, contextManagers...)
	}

	if opt&Trap != 0 {
		t.ErrorManagers(CallFailID, callFailManager).Funcs(FuncMap{
			"trap": func(context *Context) interface{} {
				defer context.Recover()
				args := context.EvalArgs()
				for _, arg := range context.EvalArgs() {
					switch err := arg.(type) {
					case error:
						context.state.push("$error", reflect.ValueOf(err))
						return nil
					}
				}
				return convertResult(args)
			},
		})
	}

	if opt&Eval != 0 {
		t.Funcs(FuncMap{
			"eval": func(context *Context, expressions ...string) (result string, err error) {
				t := context.Template().New("eval")
				data := make(Map)

				if context.dot.IsValid() && context.dot.Type().ConvertibleTo(reflect.TypeOf(data)) {
					iter := context.dot.Convert(reflect.TypeOf(data)).MapRange()
					for iter.Next() {
						data[iter.Key().String()] = iter.Value()
					}
				}

				var init string
				for key, value := range context.Variables() {
					data[key] = value
					init += fmt.Sprintf(`{{- %[1]s := index $ "%[1]s" -}}`, key)
				}
				for _, expr := range expressions {
					var buffer bytes.Buffer
					if t, err = t.Parse(init + expr); err == nil {
						err = t.Execute(&buffer, data)
					}
					if err != nil {
						return result, err
					}
					result += buffer.String()
				}
				return result, nil
			},
		})
	}

	if opt&FlowControl != 0 {
		t.Funcs(FuncMap{
			"break":    func() string { panic(fcBreak) },
			"continue": func() string { panic(fcContinue) },
			"return":   flowReturnValues,
		})
	}
}

var (
	functionsAsMethods = NewErrorManager(func(context *Context) (result interface{}, action ErrorAction) {
		defer context.Recover()
		var invoked bool
		if result, invoked = context.TryCall(context.Match("function")); invoked {
			action = ResultReplaced
		}
		return
	}).Filters(`can't evaluate field (?P<function>.*) in type (?P<receiver>.*)`)

	publicFunctions = NewErrorManager(func(context *Context) (result interface{}, action ErrorAction) {
		defer context.Recover()
		var invoked bool
		name := context.Match("function")
		name = strings.ToLower(name[:1]) + name[1:]
		if result, invoked = context.TryCall(name); invoked {
			action = ResultReplaced
		}
		return
	}).Filters(`can't evaluate field (?P<function>[[:upper:]]\w*) in type .*`)

	contextManagers = ErrorManagers{
		NewErrorManager(func(context *Context) (interface{}, ErrorAction) {
			return context.Call(nil), ResultReplaced
		},
			`can't call method/function "(?P<function>.*)" with (?P<result>\d+) result`).
			OnSources(CallError),
		NewErrorManager(func(context *Context) (result interface{}, action ErrorAction) {
			defer context.Recover()
			if ft := context.fun.Type(); ft.NumIn() > 0 && ft.In(0) == reflect.TypeOf(context) {
				return context.Call(nil), ResultReplaced
			}
			return
		},
			`wrong number of args for (?P<function>.*): want (?P<want>\d+) got (?P<got>\d+)`,
			`can't handle .* for arg of type \*template\.Context`,
			`wrong type for value; expected \*template.Context; got .*`).
			OnSources(CallError)}

	callFailManager = NewErrorManager(func(context *Context) (interface{}, ErrorAction) {
		if context.Trapped() {
			err := context.Error()
			context.ClearError()
			return err, ResultReplaced
		}
		return nil, NoReplace
	},
		"executing \"(?P<template>.*)\" at <(?P<function>.*)>: error calling .*: (?P<error>.*)",
		"executing \"(?P<template>.*)\" at <(?P<function>.*)>: (?P<error>.*)",
		`(?P<error>.+)`,
	).OnSources(CallError)
)
