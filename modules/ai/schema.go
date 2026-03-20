package ai

import (
	"encoding/json"
	"reflect"
	"strings"
)

const structuredOutputToolName = "structured_output"

func toolSchema(input any) map[string]any {
	if schema, ok := input.(map[string]any); ok {
		return schema
	}

	value := reflect.ValueOf(input)
	if !value.IsValid() {
		return map[string]any{"type": "object"}
	}

	return jsonSchemaForType(value.Type())
}

func jsonSchemaForType(typ reflect.Type) map[string]any {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Struct:
		properties := map[string]any{}
		required := make([]string, 0, typ.NumField())
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}

			name, omitEmpty := jsonFieldName(field)
			if name == "" {
				continue
			}

			properties[name] = jsonSchemaForType(field.Type)
			if !omitEmpty {
				required = append(required, name)
			}
		}

		schema := map[string]any{
			"type":       "object",
			"properties": properties,
		}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": jsonSchemaForType(typ.Elem()),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": true,
		}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.String:
		return map[string]any{"type": "string"}
	default:
		return map[string]any{"type": "string"}
	}
}

func jsonFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false
	}
	if tag == "" {
		return field.Name, false
	}

	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		name = field.Name
	}

	omitEmpty := false
	for _, part := range parts[1:] {
		if strings.TrimSpace(part) == "omitempty" {
			omitEmpty = true
			break
		}
	}

	return name, omitEmpty
}

func marshalStructuredContent(content any) string {
	b, err := json.Marshal(content)
	if err != nil {
		return ""
	}
	return string(b)
}
