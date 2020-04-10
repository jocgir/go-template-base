package template

import (
	"reflect"
	"sort"
)

// ErrorManagers allows registration of error handlers to manage errors.
// An error handler is a packaged error handler function with preset filters for mode and source.
func (t *Template) ErrorManagers(name string, managers ...*ErrorManager) *Template {
	t.option.addErrorManagers(name, managers)
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

// SafeFuncs allows registering of non standard functions, i.e. functions with no return,
// or that multiple values.
func (t *Template) SafeFuncs(funcMap FuncMap) *Template {
	var replaced FuncMap
	for name, fn := range funcMap {
		f := reflect.ValueOf(fn)
		if f.Kind() != reflect.Func {
			continue
		}
		ft := f.Type()
		if ft.NumOut() == 1 && ft.Out(0) != errorType || ft.NumOut() == 2 && ft.Out(1) == errorType {
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
			result := in.callActualFunc(f)
			return result, in.Error()
		}
	}

	// We register the trap function to handle errors
	t.ErrorManagers("_FailHandler", callFailHandler())
	t.Funcs(FuncMap{
		"trap": func(result interface{}) interface{} { return result },
	})

	if replaced != nil {
		t.ErrorManagers("_NonStandardOutput", InvalidReturnHandlers()...)
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
