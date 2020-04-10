package template_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/divan/num2words"
	"github.com/jocgir/template"
)

func ExampleTemplate_GetFuncs_default() {
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

func ExampleTemplate_GetFuncs_added() {
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

func ExampleTemplate_Funcs_empty() {
	defer func() { fmt.Println(recover()) }()

	// Empty function are not supported
	template.New("test").Funcs(template.FuncMap{
		"empty": func() {}},
	)

	// Output:
	// can't install method/function "empty" with 0 results
}

func ExampleTemplate_Funcs_multiple_returns() {
	defer func() { fmt.Println(recover()) }()

	// Multiple returns are not supported
	template.New("test").Funcs(template.FuncMap{
		"multiple": func() (int, string) { return 0, "Zero" },
	})

	// Output:
	// can't install method/function "multiple" with 2 results
}

func ExampleTemplate_Funcs_error() {
	defer func() { fmt.Println(recover()) }()

	// Multiple returns are not supported
	template.New("test").Funcs(template.FuncMap{
		"error": func() error { return fmt.Errorf("bang") },
	})

	// Output:
	// can't install method/function "error" with only error as result
}

func ExampleTemplate_ExtraFuncs_methods() {
	// Let's say we have the following object as context:
	//   type MyObject struct{}
	//   func (o *MyObject) NoReturn()         {}
	//   func (o *MyObject) Error() error      { return fmt.Errorf("bang") }
	//   func (o *MyObject) Tuple() (int, int) { return 1, 2 }
	test := func(name, code string, withExtraFuncs bool) {
		var (
			buffer = new(bytes.Buffer)
			t      = template.New(name)
			err    error
		)

		defer func() {
			if rec := recover(); rec != nil {
				err = rec.(error)
			}
			fmt.Printf("  %-12s: Err=%v Result=%q\n", t.Name(), err, buffer.String())
		}()

		if withExtraFuncs {
			// Calling ExtraFuncs with or without custom functions registers
			// special functions/methods error handling and also add trap and eval
			// functions.
			t.ExtraFuncs(nil)
		}
		tt, err := t.Parse(code)
		if err == nil {
			err = tt.Execute(buffer, new(MyObject))
		}
	}

	for _, mode := range []string{"Without", "With"} {
		fmt.Printf("\n%s ExtraFuncs:\n", mode)
		extra := mode == "With"
		test("Empty method", `{{.NoReturn}}`, extra)
		test("Tuple method", `{{.Tuple}}`, extra)
		test("Error method", `{{.Error}}`, extra)
		test("Trap error", `{{trap .Error}}`, extra)
	}

	// Output:
	// Without ExtraFuncs:
	//   Empty method: Err=template: Empty method:1:2: executing "Empty method" at <.NoReturn>: can't call method/function "NoReturn" with 0 results Result=""
	//   Tuple method: Err=template: Tuple method:1:2: executing "Tuple method" at <.Tuple>: can't call method/function "Tuple" with 2 results Result=""
	//   Error method: Err=template: Error method:1:2: executing "Error method" at <.Error>: can't call method/function "Error" with 1 results Result=""
	//   Trap error  : Err=template: Trap error:1: function "trap" not defined Result=""
	//
	// With ExtraFuncs:
	//   Empty method: Err=<nil> Result=""
	//   Tuple method: Err=<nil> Result="[1 2]"
	//   Error method: Err=template: Error method:1:2: executing "Error method" at <.Error>: bang Result=""
	//   Trap error  : Err=<nil> Result="bang"
}

type MyObject struct{}

func (o *MyObject) NoReturn()         {}
func (o *MyObject) Error() error      { return fmt.Errorf("bang") }
func (o *MyObject) Tuple() (int, int) { return 1, 2 }

func ExampleTemplate_ExtraFuncs_functions() {
	// Registering functions with SafeFuncs allow proper management of otherwise
	// considered non valid functions.

	// The default function are replaced by an internal handler that will correctly
	// processes the function output.
	//   If there is no return, an empty string is returned.
	//   If there is more than one return, an array is returned.
	//   If the function only return an error, the error will be processed.
	t := template.New("test").
		ExtraFuncs(template.FuncMap{
			"empty": func() {},
			"tuple": func() (int, int) { return 1, 2 },
			"error": func() error { return fmt.Errorf("bang") },
		})

	// We first print the function prototypes
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
	t, _ = t.Parse(`Tuple: "{{tuple}}"`)
	err = t.Execute(buffer, nil)
	fmt.Println(err, buffer)

	buffer.Reset()
	t, _ = t.Parse(`Error: "{{error}}"`)
	err = t.Execute(buffer, nil)
	fmt.Println(err, buffer)

	// Output:
	// empty = func(*template.Context) (interface {}, error)
	// error = func(*template.Context) (interface {}, error)
	// eval = func(*template.Context, ...string) (string, error)
	// trap = func(interface {}) interface {}
	// tuple = func(*template.Context) (interface {}, error)
	// <nil> Empty value: ""
	// <nil> Tuple: "[1 2]"
	// template: test:1:10: executing "test" at <error>: bang Error: "
}

func ExampleTemplate_ErrorManagers() {
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
