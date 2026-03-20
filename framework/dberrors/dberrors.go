package dberrors

import (
	"database/sql"
	"errors"
	"reflect"
)

type notFoundError interface {
	error
	NotFound() bool
}

type constraintError interface {
	error
	Constraint() bool
}

func IsNotFound(err error) bool {
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	if hasErrorTypeName(err, "NotFoundError") {
		return true
	}
	var target notFoundError
	return errors.As(err, &target) && target.NotFound()
}

func IsConstraint(err error) bool {
	if hasErrorTypeName(err, "ConstraintError") {
		return true
	}
	var target constraintError
	return errors.As(err, &target) && target.Constraint()
}

func hasErrorTypeName(err error, typeName string) bool {
	if err == nil {
		return false
	}
	for _, candidate := range flattenErrors(err) {
		if candidate == nil {
			continue
		}
		t := reflect.TypeOf(candidate)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Name() == typeName {
			return true
		}
	}
	return false
}

func flattenErrors(err error) []error {
	out := make([]error, 0, 4)
	queue := []error{err}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == nil {
			continue
		}
		out = append(out, current)
		if single, ok := current.(interface{ Unwrap() error }); ok {
			if next := single.Unwrap(); next != nil {
				queue = append(queue, next)
			}
		}
		if multi, ok := current.(interface{ Unwrap() []error }); ok {
			queue = append(queue, multi.Unwrap()...)
		}
	}
	return out
}
