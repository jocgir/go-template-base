package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/divan/num2words"
)

func ExampleTemplate_GetFuncs_default() {
	builtins := New("test").GetBuiltins()
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

func ExampleTemplate_GetFuncs_added() {
	funcs := New("test").
		Funcs(FuncMap{
			"hello": func() string { return "Hello" },
			"world": func() string { return "world" },
		}).
		GetFuncs()
	fmt.Println(strings.Join(funcs, "\n"))

	// Output:
	// hello
	// world
}

func ExampleTemplate_Funcs_empty() {
	defer func() { fmt.Println(recover()) }()

	// Empty function are not supported
	New("test").Funcs(FuncMap{
		"empty": func() {}},
	)

	// Output:
	// can't install method/function "empty" with 0 results
}

func ExampleTemplate_Funcs_multiple_returns() {
	defer func() { fmt.Println(recover()) }()

	// Multiple returns are not supported
	New("test").Funcs(FuncMap{
		"multiple": func() (int, string) { return 0, "Zero" },
	})

	// Output:
	// can't install method/function "multiple" with 2 results
}

func ExampleTemplate_Funcs_error() {
	defer func() { fmt.Println(recover()) }()

	// Multiple returns are not supported
	New("test").Funcs(FuncMap{
		"error": func() error { return fmt.Errorf("bang") },
	})

	// Output:
	// can't install method/function "error" with only error as result
}

func ExampleTemplate_SafeFuncs() {
	// Registering functions with SafeFuncs allow proper management of otherwise
	// considered non valid functions.

	// The default function are replaced by an internal handler that will correctly
	// processes the function output.
	//   If there is no return, an empty string is returned.
	//   If there is more than one return, an array is returned.
	//   If the function only return an error, the error will be processed.
	t := New("test").
		SafeFuncs(FuncMap{
			"empty":    func() {},
			"multiple": func() (int, string) { return 0, "Zero" },
			"error":    func() error { return fmt.Errorf("bang") },
		})
	for _, key := range t.GetFuncs() {
		fun := reflect.TypeOf(t.GetFuncsMap()[key])
		fmt.Println(key, "=", fun)
	}

	var err error
	buffer := new(bytes.Buffer)
	t, _ = t.Parse(`Empty value: "{{empty}}"`)
	err = t.Execute(buffer, nil)
	fmt.Println(err, buffer)

	buffer.Reset()
	t, _ = t.Parse(`Multiple: "{{multiple}}"`)
	err = t.Execute(buffer, nil)
	fmt.Println(err, buffer)

	buffer.Reset()
	t, _ = t.Parse(`Error: "{{error}}"`)
	err = t.Execute(buffer, nil)
	fmt.Println(err, buffer)

	// Output:
	// empty = func(*template.Context) (interface {}, error)
	// error = func(*template.Context) (interface {}, error)
	// multiple = func(*template.Context) (interface {}, error)
	// <nil> Empty value: ""
	// <nil> Multiple: "[0 Zero]"
	// template: test:1:10: executing "test" at <error>: bang Error: "
}

func ExampleTemplate_ErrorManagers() {
	t, err := New("test").
		// We register new functions to return a number, a list and a map
		SafeFuncs(
			FuncMap{
				"number": func() int { return 1234 },
				"list":   func() (int, string) { return 0, "Zero" },
				"map":    func() map[string]interface{} { return map[string]interface{}{"hello": "world"} },
			}).

		// We register an error manager to convert render list into json (map should not be affected)
		ErrorManagers("List as json", NewErrorManager(func(context *Context) (interface{}, ErrorAction) {
			result, err := json.MarshalIndent(context.Result().Interface(), "", "  ")
			if err != nil {
				context.SetError(err)
			}
			return string(result), ResultReplaced
		}).OnSources(Print).OnKinds(reflect.Array, reflect.Slice)).

		// Weird example, but we also convert integer value into its english representation
		// using github.com/divan/num2words package
		ErrorManagers("Number as text", NewErrorManager(func(context *Context) (interface{}, ErrorAction) {
			value := context.Result().Int()
			return num2words.ConvertAnd(int(value)), ResultReplaced
		}).OnSources(Print).OnKinds(reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64)).

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
