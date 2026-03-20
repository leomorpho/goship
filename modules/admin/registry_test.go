package admin

import (
	"testing"
	"time"
)

type testPost struct {
	ID        int       `validate:"required"`
	Title     string    `validate:"required"`
	Body      string    `admin:"text"`
	Email     string    `admin:"email"`
	Published bool
	CreatedAt time.Time
	Password  string
}

func TestRegisterPopulatesRegistry(t *testing.T) {
	registry = map[string]AdminResource{}

	Register[testPost](ResourceConfig{
		TableName: "posts",
		ReadOnly:  []string{"ID"},
		Sensitive: []string{"Password"},
	})

	resource, ok := registry["testPost"]
	if !ok {
		t.Fatalf("expected resource to be registered")
	}
	if resource.Name != "testPost" {
		t.Fatalf("resource name = %q, want %q", resource.Name, "testPost")
	}
	if resource.PluralName != "testPosts" {
		t.Fatalf("resource plural name = %q, want %q", resource.PluralName, "testPosts")
	}
	if resource.TableName != "posts" {
		t.Fatalf("resource table = %q, want %q", resource.TableName, "posts")
	}
	if resource.IDField != "ID" {
		t.Fatalf("resource ID field = %q, want %q", resource.IDField, "ID")
	}

	assertField(t, resource.Fields, "ID", FieldTypeReadOnly, true, false)
	assertField(t, resource.Fields, "Title", FieldTypeString, true, false)
	assertField(t, resource.Fields, "Body", FieldTypeText, false, false)
	assertField(t, resource.Fields, "Email", FieldTypeEmail, false, false)
	assertField(t, resource.Fields, "Published", FieldTypeBool, false, false)
	assertField(t, resource.Fields, "CreatedAt", FieldTypeTime, false, false)
	assertField(t, resource.Fields, "Password", FieldTypePassword, false, true)
}

func assertField(t *testing.T, fields []AdminField, name string, typ FieldType, required, sensitive bool) {
	t.Helper()

	for _, field := range fields {
		if field.Name != name {
			continue
		}
		if field.Type != typ {
			t.Fatalf("%s type = %q, want %q", name, field.Type, typ)
		}
		if field.Required != required {
			t.Fatalf("%s required = %v, want %v", name, field.Required, required)
		}
		if field.Sensitive != sensitive {
			t.Fatalf("%s sensitive = %v, want %v", name, field.Sensitive, sensitive)
		}
		return
	}

	t.Fatalf("field %q not found", name)
}
