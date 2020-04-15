package template

import (
	"reflect"
	"regexp"
)

// ErrorHandler represents the function type used to try to recover missing key during the template evaluation.
type ErrorHandler func(context *Context) (interface{}, ErrorAction)

// NoValue is the rendered string representation of invalid value if missingkey is set to invalid or left to default.
const NoValue = "<no value>"

// ErrorManagers represents a list of ErrorManager.
type ErrorManagers []*ErrorManager

// NewErrorManager creates an ErrorManager object.
func NewErrorManager(handler ErrorHandler, filters ...string) *ErrorManager {
	return (&ErrorManager{fun: handler}).Filters(filters...)
}

// ErrorManager represents a pre-packaged ErrorHandler function.
type ErrorManager struct {
	fun     ErrorHandler
	source  ContextSource
	mode    MissingAction
	members []string
	filters []*regexp.Regexp
	kinds   []reflect.Kind
}

// OnSources indicates the error source handled by this manager.
func (em *ErrorManager) OnSources(sources ...ContextSource) *ErrorManager {
	for _, source := range sources {
		em.source |= source
	}
	return em
}

// OnActions indicates the error action mode handled by this manager.
func (em *ErrorManager) OnActions(modes ...MissingAction) *ErrorManager {
	for _, mode := range modes {
		em.mode |= mode
	}
	return em
}

// Filters indicates what errors pattern are processed by this manager.
// filters must be valid regular expressions.
// If the filter contains subexpression such as (?P<name>.*), the name will be
// available through context.Match("name"). If is also possible to access the match
// by calling context(n) where:
//   0 the whole match
//   1 the first matching group and so on
func (em *ErrorManager) Filters(filters ...string) *ErrorManager {
	for _, filter := range filters {
		em.filters = append(em.filters, regexp.MustCompile(filter))
	}
	return em
}

// OnMembers indicates the faulty members handled by this manager.
func (em *ErrorManager) OnMembers(members ...string) *ErrorManager {
	em.members = append(em.members, members...)
	return em
}

// OnKinds indicates the faulty receiver kind handled by this manager.
func (em *ErrorManager) OnKinds(kinds ...reflect.Kind) *ErrorManager {
	em.kinds = append(em.kinds, kinds...)
	return em
}

// CanManage returns true if the error manager can handle the kind of error.
func (em *ErrorManager) CanManage(context *Context) bool {
	if em.source != 0 && !em.source.IsSet(context.source) || em.mode != 0 && !em.mode.IsSet(context.Template().MissingMode()) {
		return false
	}
	if len(em.members) > 0 {
		match := false
		for i := 0; !match && i < len(em.members); i++ {
			match = em.members[i] == context.MemberName()
		}
		if !match {
			return false
		}
	}
	if len(em.kinds) > 0 {
		match := false
		for i := 0; !match && i < len(em.kinds); i++ {
			match = em.kinds[i] == context.Receiver().Kind()
		}
		if !match {
			return false
		}
	}
	if context.Error() != nil {
		for _, re := range em.filters {
			if context.match(re) {
				return true
			}
		}
	}
	return len(em.filters) == 0
}
