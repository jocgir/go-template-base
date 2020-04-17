package template

import (
	"bytes"
	"reflect"
	"sort"
)

// MustExecute parses and execute the provided template code.
// It panics if any error occur during the parsing or the execution.
func (t *Template) MustExecute(text string, data interface{}) string {
	var (
		b    = new(bytes.Buffer)
		tmpl = Must(t.Parse(text))
	)
	if err := tmpl.Execute(b, data); err != nil {
		panic(err)
	}
	return b.String()
}

// ErrorManagers allows registration of error handlers to manage errors.
// An error handler is a packaged error handler function with preset filters for mode and source.
//
// Is is possible to deregister a previously added error manager by simply calling this method
// with the id of the manager to remove without managers.
//   t.ErrorManagers("id to remove")
func (t *Template) ErrorManagers(name string, managers ...*ErrorManager) *Template {
	if t.errorHandlers.managers == nil {
		t.errorHandlers.managers = make(map[string]ErrorManagers)
	}
	if len(managers) == 0 {
		delete(t.errorHandlers.managers, name)
	} else {
		t.errorHandlers.managers[name] = managers
	}

	t.errorHandlers.keys = make([]string, 0, len(t.errorHandlers.managers))
	for key := range t.errorHandlers.managers {
		t.errorHandlers.keys = append(t.errorHandlers.keys, key)
	}
	sort.Strings(t.errorHandlers.keys)
	return t
}

// GetBuiltins returns the sorted list of builtin functions name added to the template.
func (t *Template) GetBuiltins() []string { return getSortedName(t.GetBuiltinsMap()) }

// GetBuiltinsMap returns the list of builtin functions added to the template.
func (t *Template) GetBuiltinsMap() FuncMap { return builtins }

// GetFuncs returns the sorted list of function names added to the template.
func (t *Template) GetFuncs() []string { return getSortedName(t.GetFuncsMap()) }

// GetFuncsMap returns the list of function added to the template.
func (t *Template) GetFuncsMap() FuncMap { return t.parseFuncs }

// ExtraFuncs allows registering of non standard functions, i.e. functions with no return,
// or that returns multiple values.
//
// Using this method to register your functions also handle calling object methods that are
// either not returning values or return more that one value.
//
// Functions with no return will be modified to return an empty string.
// Functions with more than one return value will be modified to return an array of interface{}.
// Functions with just an error value will be modified to return an empty string if there is no error.
func (t *Template) ExtraFuncs(funcMap FuncMap) *Template {
	var replaced FuncMap
	for name := range funcMap {
		fn := funcMap[name]
		if reflect.ValueOf(fn).Kind() != reflect.Func {
			continue
		}

		if ft := reflect.TypeOf(fn); ft.NumOut() == 1 && ft.Out(0) != errorType || ft.NumOut() == 2 && ft.Out(1) == errorType {
			continue
		}
		if replaced == nil {
			// We duplicate the supplied func map to not alter the original
			replaced = make(FuncMap, len(funcMap))
			for key := range funcMap {
				replaced[key] = funcMap[key]
			}
		}
		replaced[name] = func(in *Context) (interface{}, error) {
			return in.Call(fn), in.Error()
		}
	}

	if replaced != nil {
		t.Option(NonStandardResults)
		return t.Funcs(replaced)
	}
	return t.Funcs(funcMap)
}

func getSortedName(funcs FuncMap) []string {
	list := make([]string, 0, len(funcs))
	for name := range funcs {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}
