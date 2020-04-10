package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActionToNative(t *testing.T) {
	assert.Equal(t, mapInvalid, Invalid.convert())
	assert.Equal(t, mapZeroValue, ZeroValue.convert())
	assert.Equal(t, mapError, Error.convert())
	assert.Equal(t, mapInvalid, (Error + 1).convert())
	assert.Equal(t, mapInvalid, MissingAction(0).convert())
}

func TestActionFromNative(t *testing.T) {
	assert.Equal(t, Invalid, mapInvalid.convert())
	assert.Equal(t, ZeroValue, mapZeroValue.convert())
	assert.Equal(t, Error, mapError.convert())
	assert.Equal(t, missingKeyAction(0), Invalid.convert())
	assert.Panics(t, func() { _ = missingKeyAction(3).convert() })
}
