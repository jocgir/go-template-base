package template

import (
	"bytes"
	"fmt"
	"reflect"
)

// MissingMode returns the missing action mode currently set on template.
func (t *Template) MissingMode() MissingAction { return t.option.missingKey.convert() }

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

	// NonStandardResults enables functions and methods to have no return or more than one returned values.
	// Note that this is simply an alias to FunctionsWithContext and that it is automatically enabled when
	// registering non standard functions with ExtraFuncs method. However, it is required to activate that
	// option for non standard methods returns.
	NonStandardResults = FunctionsWithContext

	// AllOptions enables all available options.
	AllOptions = ^Option(0)
)

const (
	// FuncsAsMethodsID is the ID used to register the FunctionsAsMethods handler.
	FuncsAsMethodsID = "^0_FuncsAsMethods"
	// ContextID is the ID used to register the FunctionsWithContext handler.
	ContextID = "^1_ContextHandlers"
	// CallFailID is the ID used to register the trap handler.
	CallFailID = "^2_CallFailHandler"
)

func (t *Template) setTemplateOption(opt Option) {
	if opt&FunctionsAsMethods != 0 {
		t.ErrorManagers(FuncsAsMethodsID, NewErrorManager(func(context *Context) (result interface{}, action ErrorAction) {
			defer context.Recover()
			var invoked bool
			if result, invoked = context.TryCall(context.Match("function")); invoked {
				action = ResultReplaced
			}
			return
		}).Filters(`can't evaluate field (?P<function>.*) in type (?P<receiver>.*)`))
	}

	if opt&FunctionsWithContext != 0 {
		t.ErrorManagers(ContextID,
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
				OnSources(CallError),
		)
	}

	if opt&Trap != 0 {
		t.ErrorManagers(CallFailID,
			NewErrorManager(func(context *Context) (interface{}, ErrorAction) {
				if context.StackLen() > 1 && context.StackPeek(1).Name == "trap" {
					context.ClearError()
					return context.Match("error"), ResultReplaced
				}
				return nil, NoReplace
			},
				"executing \"(?P<template>.*)\" at <(?P<function>.*)>: error calling .*: (?P<error>.*)",
				"executing \"(?P<template>.*)\" at <(?P<function>.*)>: (?P<error>.*)",
				`(?P<error>.+)`,
			).OnSources(CallError),
		).Funcs(FuncMap{
			"trap": func(result interface{}) interface{} { return result },
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
}
