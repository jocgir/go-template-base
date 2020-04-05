package template

import (
	"reflect"
)

// ErrorManagers allows registration of error handlers to manage errors.
// An error handler is a packaged error handler function with preset filters for mode and source.
func (t *Template) ErrorManagers(name string, managers ...*ErrorManager) *Template {
	t.option.addErrorManagers(name, managers)
	return t
}

// GetFuncs returns the list of function added to the template.
func (t *Template) GetFuncs(builtin bool) FuncMap {
	if builtin {
		return builtins
	}
	return t.parseFuncs
}

// SafeFuncs allows registering of non standard functions, i.e. functions with no return,
// or that multiple values.
func (t *Template) SafeFuncs(funcMap FuncMap) *Template {
	for name, fn := range funcMap {
		f := reflect.ValueOf(fn)
		if f.Kind() != reflect.Func {
			continue
		}
		ft := f.Type()
		if ft.NumOut() == 1 && ft.Out(0) != errorType || ft.NumOut() == 2 && ft.Out(1) == errorType {
			continue
		}
		funcMap[name] = func(in *ErrorContext) (interface{}, error) {
			result := in.callActualFunc(f)
			return result, in.Error()
		}
	}
	return t.Funcs(funcMap)
}
