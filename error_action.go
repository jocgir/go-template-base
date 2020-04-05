package template

// ErrorAction defines the action done by external handler when managing missing key.
type ErrorAction uint8

const (
	// NoReplace is returned if the external handler has not been able to fix the missing key.
	NoReplace ErrorAction = iota
	// ResultReplaced is returned if the external handler returned a valid replacement for the missing key.
	ResultReplaced
	// ResultAsArray is returned if the external handler returned an array on which we should apply the missing key.
	ResultAsArray
)

func (a ErrorAction) String() string {
	switch a {
	case NoReplace:
		return "NoReplace"
	case ResultReplaced:
		return "Replaced"
	case ResultAsArray:
		return "AsArray"
	}
	return "Undefined"
}
