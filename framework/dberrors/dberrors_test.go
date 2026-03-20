package dberrors

import (
	"database/sql"
	"fmt"
	"testing"
)

type fakeNotFound struct{}

func (fakeNotFound) Error() string    { return "not found" }
func (fakeNotFound) NotFound() bool   { return true }
func (fakeNotFound) Constraint() bool { return false }

type fakeConstraint struct{}

func (fakeConstraint) Error() string    { return "constraint" }
func (fakeConstraint) NotFound() bool   { return false }
func (fakeConstraint) Constraint() bool { return true }

type fakeOther struct{}

func (fakeOther) Error() string { return "other" }

type NotFoundError struct{}

func (NotFoundError) Error() string { return "named-not-found" }

type ConstraintError struct{}

func (ConstraintError) Error() string { return "named-constraint" }

func TestIsNotFound(t *testing.T) {
	t.Parallel()

	if !IsNotFound(fakeNotFound{}) {
		t.Fatal("expected true for not found error")
	}
	if IsNotFound(fakeConstraint{}) {
		t.Fatal("expected false for non-not-found error")
	}
	if IsNotFound(fakeOther{}) {
		t.Fatal("expected false for plain error")
	}
	if !IsNotFound(sql.ErrNoRows) {
		t.Fatal("expected true for sql.ErrNoRows")
	}
	if !IsNotFound(fmt.Errorf("wrapped: %w", fakeNotFound{})) {
		t.Fatal("expected true for wrapped not found error")
	}
	if !IsNotFound(NotFoundError{}) {
		t.Fatal("expected true for NotFoundError type-name match")
	}
}

func TestIsConstraint(t *testing.T) {
	t.Parallel()

	if !IsConstraint(fakeConstraint{}) {
		t.Fatal("expected true for constraint error")
	}
	if IsConstraint(fakeNotFound{}) {
		t.Fatal("expected false for non-constraint error")
	}
	if IsConstraint(fakeOther{}) {
		t.Fatal("expected false for plain error")
	}
	if !IsConstraint(ConstraintError{}) {
		t.Fatal("expected true for ConstraintError type-name match")
	}
}
