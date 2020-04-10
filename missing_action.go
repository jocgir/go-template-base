package template

import "strings"

// MissingAction is the public representation of the private missingKeyAction
type MissingAction uint8

const (
	// Invalid is used to return invalid reflect.Value when undefined.
	Invalid MissingAction = 1 << iota
	// ZeroValue indicates to return the zero value for the map element when undefined.
	ZeroValue
	// Error indicates that missing elements should be considered as error.
	Error
)

func (a MissingAction) String() string {
	var result []string
	if a == 0 {
		return "None"
	}
	if a&Invalid != 0 {
		result = append(result, "Invalid")
	}
	if a&ZeroValue != 0 {
		result = append(result, "ZeroValue")
	}
	if a&Error != 0 {
		result = append(result, "Error")
	}
	if len(result) == 0 {
		return "Undefined"
	}
	return strings.Join(result, ",")
}

// IsSet check whether or not the action has the specified value set.
func (a MissingAction) IsSet(value MissingAction) bool { return a|value != 0 }

func (a MissingAction) convert() missingKeyAction {
	if a < Invalid || a > Error {
		return mapInvalid
	}
	return []missingKeyAction{mapInvalid, mapZeroValue, mapInvalid, mapError}[a-1]
}

// Functions to easily convert missingKeyAction to public MissingAction.
func (a missingKeyAction) convert() MissingAction {
	return []MissingAction{Invalid, ZeroValue, Error}[a]
}
func (a missingKeyAction) String() string { return a.convert().String() }
