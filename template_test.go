package template_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/divan/num2words"
	"github.com/jocgir/template"
)

func ExampleTemplate_GetBuiltins() {
	builtins := template.New("test").GetBuiltins()
	fmt.Println(strings.Join(builtins, "\n"))

	// Output:
	// and
	// call
	// eq
	// ge
	// gt
	// html
	// index
	// js
	// le
	// len
	// lt
	// ne
	// not
	// or
	// print
	// printf
	// println
	// slice
	// urlquery
}

func ExampleTemplate_GetFuncs() {
	funcs := template.New("test").
		Funcs(template.FuncMap{
			"hello": func() string { return "Hello" },
			"world": func() string { return "world" },
		}).
		GetFuncs()
	fmt.Println(strings.Join(funcs, "\n"))

	// Output:
	// hello
	// world
}

func ExampleTemplate_Option_methods() {
	// Let's say we have the following object as context:
	//   type MyObject struct{}
	//   func (o *MyObject) NoReturn()         {}
	//   func (o *MyObject) Error() error      { return fmt.Errorf("bang") }
	//   func (o *MyObject) Tuple() (int, int) { return 1, 2 }

	var (
		withExtraFuncs bool

		test = func(code string) {
			var (
				buffer = new(bytes.Buffer)
				t      = template.New("test")
				result string
			)

			defer func() {
				if rec := recover(); rec != nil {
					result = rec.(error).Error()
				}
				fmt.Printf("  %s = %q\n", code, result)
			}()

			if withExtraFuncs {
				// Calling ExtraFuncs with or without custom functions registers
				// special functions/methods error handling and also add trap and eval
				// functions.
				t.Option(template.NonStandardResults, template.Trap)
			}
			tt := template.Must(t.Parse(code))
			if err := tt.Execute(buffer, new(MyObject)); err == nil {
				result = buffer.String()
			} else {
				result = err.Error()
			}
		}
	)

	for _, mode := range []string{"Without", "With"} {
		withExtraFuncs = mode == "With"
		fmt.Printf("\n%s ExtraFuncs:\n", mode)
		test(`{{ .NoReturn }}`)
		test(`{{ .Tuple }}`)
		test(`{{ .Error }}`)
		test(`{{ trap .Error }}`)
	}

	// Output:
	// Without ExtraFuncs:
	//   {{ .NoReturn }} = "template: test:1:3: executing \"test\" at <.NoReturn>: can't call method/function \"NoReturn\" with 0 results"
	//   {{ .Tuple }} = "template: test:1:3: executing \"test\" at <.Tuple>: can't call method/function \"Tuple\" with 2 results"
	//   {{ .Error }} = "template: test:1:3: executing \"test\" at <.Error>: can't call method/function \"Error\" with 1 results"
	//   {{ trap .Error }} = "template: test:1: function \"trap\" not defined"
	//
	// With ExtraFuncs:
	//   {{ .NoReturn }} = ""
	//   {{ .Tuple }} = "[1 2]"
	//   {{ .Error }} = "template: test:1:3: executing \"test\" at <.Error>: bang"
	//   {{ trap .Error }} = "bang"
}

type MyObject struct{}

func (o *MyObject) NoReturn()         {}
func (o *MyObject) Error() error      { return fmt.Errorf("bang") }
func (o *MyObject) Tuple() (int, int) { return 1, 2 }

func ExampleTemplate_ExtraFuncs_functions() {
	var (
		usingExtra bool

		test = func(name, code string, fun interface{}) {
			var (
				buffer = new(bytes.Buffer)
				t      = template.New("test").Option(template.Trap)
				funcs  = template.FuncMap{name: fun}
				result string
			)

			defer func() {
				if rec := recover(); rec != nil {
					result = rec.(error).Error()
				}
				fmt.Printf("  %s = %q\n", code, result)
			}()

			if usingExtra {
				t.ExtraFuncs(funcs)
			} else {
				t.Funcs(funcs)
			}

			t = template.Must(t.Parse(code))
			if err := t.Execute(buffer, nil); err == nil {
				result = buffer.String()
			} else {
				result = err.Error()
			}
		}
	)

	for _, mode := range []string{"Funcs", "ExtraFuncs"} {
		usingExtra = mode == "ExtraFuncs"
		fmt.Printf("\nWith %s:\n", mode)
		test("empty", `{{empty}}`, func() {})
		test("tuple", `{{tuple}}`, func() (int, int) { return 1, 2 })
		test("error", `{{error}}`, func() error { return fmt.Errorf("bang") })
		test("trapped", `{{trap trapped}}`, func() (string, error) { panic("boom") })
	}

	// Output:
	// With Funcs:
	//   {{empty}} = "can't install method/function \"empty\" with 0 results"
	//   {{tuple}} = "can't install method/function \"tuple\" with 2 results"
	//   {{error}} = "can't install method/function \"error\" with only error as result"
	//   {{trap trapped}} = "boom"
	//
	// With ExtraFuncs:
	//   {{empty}} = ""
	//   {{tuple}} = "[1 2]"
	//   {{error}} = "template: test:1:2: executing \"test\" at <error>: bang"
	//   {{trap trapped}} = "boom"
}

func ExampleTemplate_ErrorManagers_format() {
	t, err := template.New("test").
		// We register new functions to return a number, a list and a map
		ExtraFuncs(
			template.FuncMap{
				"number": func() int { return 1234 },
				"list":   func() (int, string) { return 0, "Zero" },
				"map":    func() template.Map { return template.Map{"hello": "world"} },
			}).

		// We register an error manager to convert render list into json (map should not be affected)
		ErrorManagers("List as json", template.NewErrorManager(
			func(context *template.Context) (interface{}, template.ErrorAction) {
				result, err := json.MarshalIndent(context.Result().Interface(), "", "  ")
				if err != nil {
					context.SetError(err)
				}
				return string(result), template.ResultReplaced
			}).OnSources(template.Print).OnKinds(reflect.Array, reflect.Slice)).

		// Weird example, but we also convert integer value into its english representation
		// using github.com/divan/num2words package
		ErrorManagers("Number as text", template.NewErrorManager(
			func(context *template.Context) (interface{}, template.ErrorAction) {
				value := context.Result().Int()
				return num2words.ConvertAnd(int(value)), template.ResultReplaced
			}).OnSources(template.Print).OnKinds(reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64)).

		// Here is the template text to process
		Parse("Number = {{number}}\nList = {{list}}\nMap = {{map}}")
	if err != nil {
		panic(err)
	}

	buffer := new(bytes.Buffer)
	t.Execute(buffer, nil)
	fmt.Println(buffer)
	// Output:
	// Number = one thousand two hundred and thirty-four
	// List = [
	//   0,
	//   "Zero"
	// ]
	// Map = map[hello:world]
}

func ExampleTemplate_ErrorManagers_bad_parameters() {
	var (
		withErrorManager bool

		add  = func(a, b int) int { return a + b }
		base = template.New("test").Funcs(template.FuncMap{"add": add})
		test = func(code string) {
			t := template.Must(base.Clone())
			if withErrorManager {
				// We register an handler to add arguments if they are not integer or there are
				// more or less arguments than expected
				t.ErrorManagers("Add other types", template.NewErrorManager(
					func(context *template.Context) (result interface{}, action template.ErrorAction) {
						context.ClearError()
						args := context.EvalArgs()

						defer func() {
							if recover() != nil {
								// In case of error, we simply concat the string representation of args.
								result = fmt.Sprint(args...)
								action = template.ResultReplaced
							}
						}()

						// We try to add all arguments as float64
						var value float64
						for _, arg := range args {
							v, err := strconv.ParseFloat(fmt.Sprint(arg), 64)
							if err != nil {
								panic(err)
							}
							value += v
						}
						return value, template.ResultReplaced
					}).
					OnMembers("add").
					Filters(`(?P<template>.*):(?P<line>\d+):(?P<column>\d+): executing .*: (?P<error>.*)$`),
				)
			}
			var buffer = new(bytes.Buffer)
			err := template.Must(t.Parse(code)).Execute(buffer, nil)
			result := buffer.String()
			if err != nil {
				result = err.Error()
			}
			fmt.Printf("  %s = %q\n", code, result)
		}
	)

	for _, mode := range []string{"Without", "With"} {
		withErrorManager = mode == "With"
		fmt.Printf("\n%s Error Manager:\n", mode)
		test(`{{ add 2 3 }}`)
		test(`{{ add 2.0 3.0 }}`)
		test(`{{ add }}`)
		test(`{{ add 5 }}`)
		test(`{{ add 1 2 3 }}`)
		test(`{{ add 1.2 3.4 }}`)
		test(`{{ add "a" "b" "c" "d" }}`)
		test(`{{ "suffix" | add "prefix" 0 1 }}`)
	}

	// Output:
	// 	Without Error Manager:
	//   {{ add 2 3 }} = "5"
	//   {{ add 2.0 3.0 }} = "5"
	//   {{ add }} = "template: test:1:3: executing \"test\" at <add>: wrong number of args for add: want 2 got 0"
	//   {{ add 5 }} = "template: test:1:3: executing \"test\" at <add>: wrong number of args for add: want 2 got 1"
	//   {{ add 1 2 3 }} = "template: test:1:3: executing \"test\" at <add>: wrong number of args for add: want 2 got 3"
	//   {{ add 1.2 3.4 }} = "template: test:1:7: executing \"test\" at <1.2>: expected integer; found 1.2"
	//   {{ add "a" "b" "c" "d" }} = "template: test:1:3: executing \"test\" at <add>: wrong number of args for add: want 2 got 4"
	//   {{ "suffix" | add "prefix" 0 1 }} = "template: test:1:14: executing \"test\" at <add>: wrong number of args for add: want 2 got 4"
	//
	// With Error Manager:
	//   {{ add 2 3 }} = "5"
	//   {{ add 2.0 3.0 }} = "5"
	//   {{ add }} = "0"
	//   {{ add 5 }} = "5"
	//   {{ add 1 2 3 }} = "6"
	//   {{ add 1.2 3.4 }} = "4.6"
	//   {{ add "a" "b" "c" "d" }} = "abcd"
	//   {{ "suffix" | add "prefix" 0 1 }} = "prefix0 1suffix"
}

func ExampleTemplate_Funcs_context() {
	var (
		t = template.New("test")

		// Adding a custom function that directly handle the *template.Context greatly simplifies
		// the code and avoid having to handle errors. The custom function is then responsible to
		// evaluate the supplied arguments.
		sum = func(context *template.Context) interface{} {
			var (
				value float64
				args  = context.EvalArgs()
			)
			for _, arg := range args {
				v, err := strconv.ParseFloat(fmt.Sprint(arg), 64)
				if err != nil {
					return fmt.Sprint(args...)
				}
				value += v
			}
			return value
		}

		test = func(code string) {
			var (
				buffer = new(bytes.Buffer)
				result string
			)

			if err := template.Must(t.Parse(code)).Execute(buffer, nil); err == nil {
				result = buffer.String()
			} else {
				result = err.Error()
			}
			fmt.Printf("%s = %q\n", code, result)
		}
	)

	// There is no need to call ExtraFuncs when the custom function already handle *template.Context
	t.Funcs(template.FuncMap{"sum": sum})

	test(`{{ sum 2 3 }}`)
	test(`{{ sum 2.0 3.0 }}`)
	test(`{{ sum }}`)
	test(`{{ sum 5 }}`)
	test(`{{ sum 1 2 3 }}`)
	test(`{{ sum 1.2 3.4 }}`)
	test(`{{ sum "a" "b" "c" "d" }}`)
	test(`{{ "suffix" | sum "prefix" 0 1 }}`)

	// Output:
	// {{ sum 2 3 }} = "5"
	// {{ sum 2.0 3.0 }} = "5"
	// {{ sum }} = "0"
	// {{ sum 5 }} = "5"
	// {{ sum 1 2 3 }} = "6"
	// {{ sum 1.2 3.4 }} = "4.6"
	// {{ sum "a" "b" "c" "d" }} = "abcd"
	// {{ "suffix" | sum "prefix" 0 1 }} = "prefix0 1suffix"
}

func ExampleTemplate_Option_functions_as_methods() {
	var (
		t = template.New("test")

		// Remove all strings representation of arguments from the first argument
		remove = func(context *template.Context) interface{} {
			var (
				args   = context.EvalArgs()
				result = fmt.Sprint(args[0])
			)
			for _, substr := range args[1:] {
				result = strings.ReplaceAll(result, fmt.Sprint(substr), "")
			}
			return result
		}

		substract = func(context *template.Context) interface{} {
			var (
				value float64
				args  = context.EvalArgs()
			)
			for i, arg := range args {
				v, err := strconv.ParseFloat(fmt.Sprint(arg), 64)
				if err != nil {
					// If one onf the value is not numeric, we consider delegate the
					// processing to the remove function
					return context.Call("remove")
				}
				if i > 0 {
					v = -v
				}
				value += v
			}
			return value
		}

		test = func(code string) {
			var (
				buffer = new(bytes.Buffer)
				result string
			)

			if err := template.Must(t.Parse(code)).Execute(buffer, nil); err == nil {
				result = buffer.String()
			} else {
				result = err.Error()
			}
			fmt.Printf("  %s = %q\n", code, result)
		}
	)

	// There is no need to call ExtraFuncs when the custom function already handle *template.Context
	t.Funcs(template.FuncMap{
		"remove":    remove,
		"substract": substract,
	}).Option(template.FunctionsAsMethods)

	t.Option()
	for _, mode := range []string{"registered", "unregistered"} {
		fmt.Println("\nMethods as Functions is", mode)
		if mode == "unregistered" {
			// We deregister the error manager that handle the method as function mechanism
			t.ErrorManagers(template.FuncsAsMethodsID)
		}

		test(`{{ (2).substract 3 }}`)
		test(`{{ (2.0).substract 3.0 }}`)
		test(`{{ (5).substract }}`)
		test(`{{ (1).substract 2 3 }}`)
		test(`{{ (1.2).substract 3.4 }}`)
		test(`{{ ((1.2).substract 3.4).substract 5.6 }}`)
		test(`{{ ("Hello").substract "a" "e" "i" "o" "u" }}`)
		test(`{{ "!" | ("Hello World!").substract "ll" }}`)
		test(`Not working: {{ (2).add 3 }}`)
	}

	// Output:
	// Methods as Functions is registered
	//   {{ (2).substract 3 }} = "-1"
	//   {{ (2.0).substract 3.0 }} = "-1"
	//   {{ (5).substract }} = "5"
	//   {{ (1).substract 2 3 }} = "-4"
	//   {{ (1.2).substract 3.4 }} = "-2.2"
	//   {{ ((1.2).substract 3.4).substract 5.6 }} = "-7.8"
	//   {{ ("Hello").substract "a" "e" "i" "o" "u" }} = "Hll"
	//   {{ "!" | ("Hello World!").substract "ll" }} = "Heo World"
	//   Not working: {{ (2).add 3 }} = "template: test:1:17: executing \"test\" at <2>: can't evaluate field add in type int"
	//
	// Methods as Functions is unregistered
	//   {{ (2).substract 3 }} = "template: test:1:4: executing \"test\" at <2>: can't evaluate field substract in type int"
	//   {{ (2.0).substract 3.0 }} = "template: test:1:4: executing \"test\" at <2.0>: can't evaluate field substract in type float64"
	//   {{ (5).substract }} = "template: test:1:4: executing \"test\" at <5>: can't evaluate field substract in type int"
	//   {{ (1).substract 2 3 }} = "template: test:1:4: executing \"test\" at <1>: can't evaluate field substract in type int"
	//   {{ (1.2).substract 3.4 }} = "template: test:1:4: executing \"test\" at <1.2>: can't evaluate field substract in type float64"
	//   {{ ((1.2).substract 3.4).substract 5.6 }} = "template: test:1:5: executing \"test\" at <1.2>: can't evaluate field substract in type float64"
	//   {{ ("Hello").substract "a" "e" "i" "o" "u" }} = "template: test:1:4: executing \"test\" at <\"Hello\">: can't evaluate field substract in type string"
	//   {{ "!" | ("Hello World!").substract "ll" }} = "template: test:1:10: executing \"test\" at <\"Hello World!\">: can't evaluate field substract in type string"
	//   Not working: {{ (2).add 3 }} = "template: test:1:17: executing \"test\" at <2>: can't evaluate field add in type int"
}
