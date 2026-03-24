package ui_test

import (
	"testing"

	"github.com/leomorpho/goship/framework/tests"
	"github.com/leomorpho/goship/framework/web/ui"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormSubmission(t *testing.T) {
	type formTest struct {
		Name       string `validate:"required"`
		Email      string `validate:"required,email"`
		Submission ui.FormSubmission
	}

	ctx, _ := tests.NewContext(c.Web, "/")
	form := formTest{
		Name:  "",
		Email: "a@a.com",
	}
	err := form.Submission.Process(ctx, form)
	assert.NoError(t, err)

	assert.True(t, form.Submission.HasErrors())
	assert.True(t, form.Submission.FieldHasErrors("Name"))
	assert.False(t, form.Submission.FieldHasErrors("Email"))
	require.Len(t, form.Submission.GetFieldErrors("Name"), 1)
	assert.Len(t, form.Submission.GetFieldErrors("Email"), 0)
	assert.Equal(t, "This field is required.", form.Submission.GetFieldErrors("Name")[0])
	assert.Equal(t, "gs-field-error", form.Submission.GetFieldStatusClass("Name"))
	assert.Equal(t, "gs-field-success", form.Submission.GetFieldStatusClass("Email"))
	assert.Equal(t, "gs-field-input gs-field-error", form.Submission.GetFieldInputClass("Name"))
	assert.Equal(t, "gs-field-input gs-field-success", form.Submission.GetFieldInputClass("Email"))
	assert.Equal(t, "gs-field-hint gs-color-danger", form.Submission.GetFieldHintClass("Name"))
	assert.Equal(t, "gs-field-hint gs-color-muted", form.Submission.GetFieldHintClass("Email"))
	assert.False(t, form.Submission.IsDone())
}
