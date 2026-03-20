package softdelete

import (
	"reflect"
	"strings"
	"time"
)

const Column = "deleted_at"

func ActiveClause() string {
	return Column + " IS NULL"
}

func DeletedClause() string {
	return Column + " IS NOT NULL"
}

func IsDeleted(v any) bool {
	if v == nil {
		return false
	}

	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return false
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return false
	}

	field, ok := findDeletedAtField(rv)
	if !ok {
		return false
	}

	return fieldHasDeletionValue(field)
}

func findDeletedAtField(rv reflect.Value) (reflect.Value, bool) {
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		if fieldType.Anonymous {
			embedded := field
			for embedded.Kind() == reflect.Pointer {
				if embedded.IsNil() {
					return reflect.Value{}, false
				}
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct {
				if nested, ok := findDeletedAtField(embedded); ok {
					return nested, true
				}
			}
		}

		if strings.EqualFold(fieldType.Name, "DeletedAt") {
			return field, true
		}
	}

	return reflect.Value{}, false
}

func fieldHasDeletionValue(field reflect.Value) bool {
	for field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return false
		}
		field = field.Elem()
	}

	if !field.IsValid() {
		return false
	}

	switch field.Kind() {
	case reflect.Struct:
		if t, ok := field.Interface().(time.Time); ok {
			return !t.IsZero()
		}
	case reflect.Bool:
		return field.Bool()
	}

	return false
}
