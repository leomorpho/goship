package factory

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

type tableNamer interface {
	TableName() string
}

type Factory[T any] struct {
	defaults   func() T
	afterBuild []func(*T)
}

func New[T any](defaults func() T) *Factory[T] {
	return &Factory[T]{defaults: defaults}
}

func (f *Factory[T]) AfterBuild(fn func(*T)) *Factory[T] {
	if fn != nil {
		f.afterBuild = append(f.afterBuild, fn)
	}
	return f
}

func (f *Factory[T]) Build(overrides ...func(*T)) T {
	v := f.defaults()
	for _, fn := range overrides {
		if fn != nil {
			fn(&v)
		}
	}
	for _, fn := range f.afterBuild {
		fn(&v)
	}
	return v
}

func (f *Factory[T]) Create(t testing.TB, db *sql.DB, overrides ...func(*T)) T {
	t.Helper()
	v := f.Build(overrides...)
	if err := insertStruct(db, &v); err != nil {
		t.Fatalf("factory create failed: %v", err)
	}
	return v
}

var seq atomic.Int64

func Sequence(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, seq.Add(1))
}

func insertStruct[T any](db *sql.DB, ptr *T) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	rv := reflect.ValueOf(ptr)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("expected non-nil pointer value")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct value, got %s", rv.Kind())
	}

	rt := rv.Type()
	table := tableNameForValue(rv)

	columns := make([]string, 0, rt.NumField())
	args := make([]any, 0, rt.NumField())
	idField := -1

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		if sf.Tag.Get("factory") == "-" {
			continue
		}
		col := columnName(sf)
		fv := rv.Field(i)

		if strings.EqualFold(sf.Name, "ID") && isZeroValue(fv) {
			idField = i
			continue
		}

		columns = append(columns, col)
		args = append(args, fv.Interface())
	}

	if len(columns) == 0 {
		return fmt.Errorf("no insertable fields found for %s", rt.Name())
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(columns)), ",")
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		placeholders,
	)
	res, err := db.Exec(query, args...)
	if err != nil {
		return err
	}

	if idField >= 0 {
		id, idErr := res.LastInsertId()
		if idErr == nil {
			setIDField(rv.Field(idField), id)
		}
	}

	return nil
}

func tableNameForValue(v reflect.Value) string {
	if v.CanInterface() {
		if namer, ok := v.Interface().(tableNamer); ok {
			return namer.TableName()
		}
	}
	if v.CanAddr() && v.Addr().CanInterface() {
		if namer, ok := v.Addr().Interface().(tableNamer); ok {
			return namer.TableName()
		}
	}
	return toSnake(v.Type().Name()) + "s"
}

func columnName(field reflect.StructField) string {
	dbTag := strings.TrimSpace(field.Tag.Get("db"))
	if dbTag != "" && dbTag != "-" {
		return strings.Split(dbTag, ",")[0]
	}
	return toSnake(field.Name)
}

func toSnake(name string) string {
	if name == "" {
		return ""
	}
	var b strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isZeroValue(v reflect.Value) bool {
	return v.IsZero()
}

func setIDField(v reflect.Value, id int64) {
	if !v.CanSet() {
		return
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(id)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if id >= 0 {
			v.SetUint(uint64(id))
		}
	}
}
