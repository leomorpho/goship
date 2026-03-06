package generators

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var ModelNamePattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
var FieldNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type ModelField struct {
	Name string
	Type string
}

type GenerateModelDeps struct {
	Out      io.Writer
	Err      io.Writer
	RunCmd   func(name string, args ...string) int
	HasFile  func(path string) bool
	QueryDir string
}

func RunGenerateModel(args []string, d GenerateModelDeps) int {
	name, fields, force, err := ParseGenerateModelArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "%v\n", err)
		return 1
	}

	queryPath := filepath.Join(d.QueryDir, ModelFileName(name)+".sql")
	if d.HasFile(queryPath) && !force {
		fmt.Fprintf(d.Err, "refusing to overwrite existing model query file %s (use --force)\n", queryPath)
		return 1
	}
	content := RenderModelQueryTemplate(name, fields)
	if content == "" {
		fmt.Fprintln(d.Err, "failed to render model query template")
		return 1
	}
	if d.QueryDir == "" {
		fmt.Fprintln(d.Err, "missing query directory")
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(queryPath), 0o755); err != nil {
		fmt.Fprintf(d.Err, "failed to create query directory: %v\n", err)
		return 1
	}
	if err := os.WriteFile(queryPath, []byte(content), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write model query file: %v\n", err)
		return 1
	}
	fmt.Fprintf(d.Out, "Wrote model query scaffold: %s\n", queryPath)

	tableName := ModelFileName(name) + "s"
	fmt.Fprintln(d.Out, "Next:")
	fmt.Fprintf(d.Out, "- ship db:make create_%s_table\n", tableName)
	fmt.Fprintf(d.Out, "- edit db/migrate/migrations/*_create_%s_table.sql\n", tableName)
	fmt.Fprintln(d.Out, "- ship db:migrate")
	fmt.Fprintln(d.Out, "- ship db:generate")
	return 0
}

func ParseGenerateModelArgs(args []string) (string, []ModelField, bool, error) {
	if len(args) == 0 {
		return "", nil, false, fmt.Errorf("usage: ship make:model <Name> [fields...] [--force]")
	}
	name := strings.TrimSpace(args[0])
	if !ModelNamePattern.MatchString(name) {
		return "", nil, false, fmt.Errorf("invalid model name %q: use PascalCase (e.g. Post, BlogPost)", name)
	}
	fields := make([]ModelField, 0, len(args)-1)
	force := false
	for _, raw := range args[1:] {
		token := strings.TrimSpace(raw)
		if token == "" {
			continue
		}
		if token == "--force" {
			force = true
			continue
		}
		field, err := parseModelField(token)
		if err != nil {
			return "", nil, false, err
		}
		fields = append(fields, field)
	}
	return name, fields, force, nil
}

func parseModelField(token string) (ModelField, error) {
	name, typ, ok := strings.Cut(token, ":")
	if !ok {
		return ModelField{}, fmt.Errorf("invalid field %q: expected name:type", token)
	}
	name = strings.TrimSpace(name)
	typ = strings.ToLower(strings.TrimSpace(typ))
	if !FieldNamePattern.MatchString(name) {
		return ModelField{}, fmt.Errorf("invalid field name %q: use snake_case", name)
	}
	switch typ {
	case "string", "text", "int", "bool", "time", "float", "email", "url":
		return ModelField{Name: name, Type: typ}, nil
	default:
		return ModelField{}, fmt.Errorf("unsupported field type %q for %s", typ, name)
	}
}

func RenderModelQueryTemplate(name string, fields []ModelField) string {
	var b strings.Builder
	tableName := ModelFileName(name) + "s"
	insertName := "Insert" + name
	selectByIDName := "Get" + name + "ByID"

	b.WriteString("-- Model: ")
	b.WriteString(name)
	b.WriteString("\n")
	b.WriteString("-- Table: ")
	b.WriteString(tableName)
	b.WriteString("\n")
	b.WriteString("-- Fields:\n")
	if len(fields) == 0 {
		b.WriteString("-- - id:int\n")
	} else {
		for _, f := range fields {
			b.WriteString("-- - ")
			b.WriteString(f.Name)
			b.WriteString(":")
			b.WriteString(f.Type)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	b.WriteString("-- name: ")
	b.WriteString(insertName)
	b.WriteString(" :one\n")
	b.WriteString("-- TODO: add INSERT columns and values for ")
	b.WriteString(tableName)
	b.WriteString("\n")
	b.WriteString("INSERT INTO ")
	b.WriteString(tableName)
	b.WriteString(" DEFAULT VALUES RETURNING id;\n\n")

	b.WriteString("-- name: ")
	b.WriteString(selectByIDName)
	b.WriteString(" :one\n")
	b.WriteString("SELECT * FROM ")
	b.WriteString(tableName)
	b.WriteString(" WHERE id = ?;\n")

	return b.String()
}

func ModelFileName(name string) string {
	var out []rune
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 {
				out = append(out, '_')
			}
			out = append(out, unicode.ToLower(r))
			continue
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}
