package template

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
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
	// empty = func(*template.ErrorContext) (interface {}, error)
	// error = func(*template.ErrorContext) (interface {}, error)
	// multiple = func(*template.ErrorContext) (interface {}, error)
	// <nil> Empty value: ""
	// <nil> Multiple: "[0 Zero]"
	// template: test:1:10: executing "test" at <error>: bang Error: "
}
