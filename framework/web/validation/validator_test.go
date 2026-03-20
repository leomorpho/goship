package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidator(t *testing.T) {
	type example struct {
		Value string `validate:"required"`
	}
	v := NewValidator()
	e := example{}
	err := v.Validate(e)
	assert.Error(t, err)
	e.Value = "a"
	err = v.Validate(e)
	assert.NoError(t, err)
}
