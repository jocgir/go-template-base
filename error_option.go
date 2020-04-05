package template

import (
	"sort"
)

func (o *option) addErrorManagers(name string, managers []*ErrorManager) {
	if o.ehs.handlers == nil {
		o.ehs.handlers = make(map[string][]*ErrorManager)
	}
	if len(managers) == 0 {
		delete(o.ehs.handlers, name)
	} else {
		o.ehs.handlers[name] = managers
	}

	o.ehs.keys = make([]string, 0, len(o.ehs.handlers))
	for key := range o.ehs.handlers {
		o.ehs.keys = append(o.ehs.keys, key)
	}
	sort.Strings(o.ehs.keys)
}
