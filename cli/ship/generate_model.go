package ship

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var modelNamePattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
var fieldNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type modelField struct {
	Name string
	Type string
}

func (c CLI) runGenerateModel(args []string) int {
	name, fields, force, err := parseGenerateModelArgs(args)
	if err != nil {
		fmt.Fprintf(c.Err, "%v\n", err)
		return 1
	}

	if len(fields) == 0 {
		if code := c.runCmd("go", "run", "-mod=mod", "entgo.io/ent/cmd/ent", "new", name); code != 0 {
			return code
		}
	} else {
		schemaPath := filepath.Join("ent", "schema", modelFileName(name)+".go")
		if hasFile(schemaPath) && !force {
			fmt.Fprintf(c.Err, "refusing to overwrite existing schema %s (use --force)\n", schemaPath)
			return 1
		}
		content, renderErr := renderEntSchema(name, fields)
		if renderErr != nil {
			fmt.Fprintf(c.Err, "failed to render schema: %v\n", renderErr)
			return 1
		}
		if err := os.MkdirAll(filepath.Dir(schemaPath), 0o755); err != nil {
			fmt.Fprintf(c.Err, "failed to create schema directory: %v\n", err)
			return 1
		}
		if err := os.WriteFile(schemaPath, []byte(content), 0o644); err != nil {
			fmt.Fprintf(c.Err, "failed to write schema file: %v\n", err)
			return 1
		}
		fmt.Fprintf(c.Out, "Wrote schema: %s\n", schemaPath)
	}

	if code := c.runCmd("go", "run", "-mod=mod", "entgo.io/ent/cmd/ent", "generate", "--feature", "sql/upsert,sql/execquery", "./ent/schema"); code != 0 {
		return code
	}

	fmt.Fprintln(c.Out, "Next:")
	fmt.Fprintf(c.Out, "- ship db:make add_%ss\n", modelFileName(name))
	fmt.Fprintln(c.Out, "- ship db:migrate")
	return 0
}

func parseGenerateModelArgs(args []string) (string, []modelField, bool, error) {
	if len(args) == 0 {
		return "", nil, false, fmt.Errorf("usage: ship make:model <Name> [fields...] [--force]")
	}
	name := strings.TrimSpace(args[0])
	if !modelNamePattern.MatchString(name) {
		return "", nil, false, fmt.Errorf("invalid model name %q: use PascalCase (e.g. Post, BlogPost)", name)
	}
	fields := make([]modelField, 0, len(args)-1)
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

func parseModelField(token string) (modelField, error) {
	name, typ, ok := strings.Cut(token, ":")
	if !ok {
		return modelField{}, fmt.Errorf("invalid field %q: expected name:type", token)
	}
	name = strings.TrimSpace(name)
	typ = strings.ToLower(strings.TrimSpace(typ))
	if !fieldNamePattern.MatchString(name) {
		return modelField{}, fmt.Errorf("invalid field name %q: use snake_case", name)
	}
	switch typ {
	case "string", "text", "int", "bool", "time", "float", "email", "url":
		return modelField{Name: name, Type: typ}, nil
	default:
		return modelField{}, fmt.Errorf("unsupported field type %q for %s", typ, name)
	}
}

func renderEntSchema(name string, fields []modelField) (string, error) {
	var b strings.Builder
	b.WriteString("package schema\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"entgo.io/ent\"\n")
	b.WriteString("\t\"entgo.io/ent/schema/field\"\n")
	b.WriteString(")\n\n")
	b.WriteString("// ")
	b.WriteString(name)
	b.WriteString(" holds the schema definition for the ")
	b.WriteString(name)
	b.WriteString(" entity.\n")
	b.WriteString("type ")
	b.WriteString(name)
	b.WriteString(" struct {\n\tent.Schema\n}\n\n")
	b.WriteString("// Fields of the ")
	b.WriteString(name)
	b.WriteString(".\n")
	b.WriteString("func (")
	b.WriteString(name)
	b.WriteString(") Fields() []ent.Field {\n")
	b.WriteString("\treturn []ent.Field{\n")
	for _, f := range fields {
		b.WriteString("\t\t")
		b.WriteString(renderFieldCall(f))
		b.WriteString(",\n")
	}
	b.WriteString("\t}\n")
	b.WriteString("}\n")

	out, err := format.Source([]byte(b.String()))
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func renderFieldCall(f modelField) string {
	switch f.Type {
	case "text":
		return fmt.Sprintf("field.Text(%q)", f.Name)
	case "int":
		return fmt.Sprintf("field.Int(%q)", f.Name)
	case "bool":
		return fmt.Sprintf("field.Bool(%q)", f.Name)
	case "time":
		return fmt.Sprintf("field.Time(%q)", f.Name)
	case "float":
		return fmt.Sprintf("field.Float(%q)", f.Name)
	default:
		return fmt.Sprintf("field.String(%q)", f.Name)
	}
}

func modelFileName(name string) string {
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
