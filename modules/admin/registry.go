package admin

import (
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
)

var registry = map[string]AdminResource{}

type ResourceConfig struct {
	TableName  string
	ListFields []string
	ReadOnly   []string
	Sensitive  []string
}

func Register[T any](cfg ResourceConfig) {
	resourceType := reflect.TypeOf(*new(T))
	if resourceType.Kind() == reflect.Ptr {
		resourceType = resourceType.Elem()
	}
	if resourceType.Kind() != reflect.Struct {
		return
	}

	readOnly := makeSet(cfg.ReadOnly)
	sensitive := makeSet(cfg.Sensitive)

	fields := make([]AdminField, 0, resourceType.NumField())
	idField := ""

	for i := 0; i < resourceType.NumField(); i++ {
		sf := resourceType.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}

		isSensitive := sensitive[sf.Name]
		isReadOnly := readOnly[sf.Name]
		fieldType := deriveFieldType(sf, isSensitive, isReadOnly)

		if idField == "" && strings.EqualFold(sf.Name, "id") {
			idField = sf.Name
		}

		fields = append(fields, AdminField{
			Name:      sf.Name,
			Label:     humanizeFieldName(sf.Name),
			Type:      fieldType,
			Required:  isRequiredField(sf),
			Sensitive: isSensitive,
		})
	}

	resource := AdminResource{
		Name:       resourceType.Name(),
		PluralName: pluralize(resourceType.Name()),
		TableName:  strings.TrimSpace(cfg.TableName),
		Fields:     fields,
		IDField:    idField,
	}
	if resource.TableName == "" {
		resource.TableName = strings.ToLower(resource.PluralName)
	}

	registry[resource.Name] = resource
}

func RegisteredResources() []AdminResource {
	out := make([]AdminResource, 0, len(registry))
	for _, resource := range registry {
		out = append(out, resource)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func FindResourceByPluralName(name string) (AdminResource, bool) {
	want := strings.ToLower(strings.TrimSpace(name))
	for _, resource := range registry {
		if strings.ToLower(resource.PluralName) == want {
			return resource, true
		}
	}
	return AdminResource{}, false
}

func deriveFieldType(sf reflect.StructField, sensitive, readOnly bool) FieldType {
	if sensitive {
		return FieldTypePassword
	}
	if readOnly {
		return FieldTypeReadOnly
	}

	adminTag := sf.Tag.Get("admin")
	switch sf.Type.Kind() {
	case reflect.String:
		switch {
		case hasTag(adminTag, "email"):
			return FieldTypeEmail
		case hasTag(adminTag, "text"):
			return FieldTypeText
		default:
			return FieldTypeString
		}
	case reflect.Bool:
		return FieldTypeBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return FieldTypeInt
	case reflect.Struct:
		if sf.Type == reflect.TypeOf(time.Time{}) {
			return FieldTypeTime
		}
	}

	return FieldTypeString
}

func hasTag(tag, want string) bool {
	for _, part := range strings.Split(tag, ",") {
		if strings.TrimSpace(strings.ToLower(part)) == want {
			return true
		}
	}
	return false
}

func isRequiredField(sf reflect.StructField) bool {
	if sf.Type.Kind() == reflect.Ptr {
		return false
	}
	validateTag := sf.Tag.Get("validate")
	return hasTag(validateTag, "required")
}

func makeSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		key := strings.TrimSpace(value)
		if key == "" {
			continue
		}
		out[key] = true
	}
	return out
}

func humanizeFieldName(name string) string {
	if name == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(name) + 4)
	runes := []rune(name)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func pluralize(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if strings.HasSuffix(name, "s") {
		return name
	}
	return name + "s"
}
