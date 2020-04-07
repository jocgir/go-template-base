package template

import (
	"reflect"
	"regexp"
)

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
// filters can be regular expression, wildcard expression or part of a string.
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
	mode := context.option().missingKey.convert()
	if em.source != 0 && context.Source()&em.source == 0 || em.mode != 0 && mode&em.mode == 0 {
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
			if re.MatchString(context.ErrorText()) {
				return true
			}
		}
	}
	return len(em.filters) == 0
}
