package softdelete

import (
	"testing"
	"time"
)

func TestClauses(t *testing.T) {
	if got := ActiveClause(); got != "deleted_at IS NULL" {
		t.Fatalf("ActiveClause() = %q", got)
	}
	if got := DeletedClause(); got != "deleted_at IS NOT NULL" {
		t.Fatalf("DeletedClause() = %q", got)
	}
}

func TestIsDeleted(t *testing.T) {
	now := time.Now().UTC()

	type embedded struct {
		DeletedAt *time.Time
	}
	type direct struct {
		DeletedAt *time.Time
	}
	type valueField struct {
		DeletedAt time.Time
	}
	type nested struct {
		embedded
	}

	tests := []struct {
		name string
		in   any
		want bool
	}{
		{name: "nil", in: nil, want: false},
		{name: "missing field", in: struct{ ID int }{ID: 1}, want: false},
		{name: "nil pointer field", in: direct{}, want: false},
		{name: "non-nil pointer field", in: direct{DeletedAt: &now}, want: true},
		{name: "pointer to struct", in: &direct{DeletedAt: &now}, want: true},
		{name: "zero time value", in: valueField{}, want: false},
		{name: "non-zero time value", in: valueField{DeletedAt: now}, want: true},
		{name: "embedded field", in: nested{embedded: embedded{DeletedAt: &now}}, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsDeleted(tc.in); got != tc.want {
				t.Fatalf("IsDeleted(%T) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
