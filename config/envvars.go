package config

import (
	"bufio"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

type EnvVar struct {
	Name     string   `json:"name"`
	Aliases  []string `json:"aliases,omitempty"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Default  string   `json:"default,omitempty"`
}

func EnvVars() ([]EnvVar, error) {
	cfg := defaultConfig()
	typ := reflect.TypeOf(cfg)
	val := reflect.ValueOf(cfg)

	vars := make([]EnvVar, 0)
	if err := collectEnvVars(typ, val, &vars); err != nil {
		return nil, err
	}

	sort.Slice(vars, func(i, j int) bool {
		return vars[i].Name < vars[j].Name
	})
	return vars, nil
}

func MissingRequiredEnv(start string) ([]EnvVar, error) {
	vars, err := EnvVars()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for _, raw := range os.Environ() {
		name, _, ok := strings.Cut(raw, "=")
		if ok && strings.TrimSpace(name) != "" {
			seen[strings.TrimSpace(name)] = struct{}{}
		}
	}

	if path, ok := findDotEnv(start); ok {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "export ") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
			}
			name, _, ok := strings.Cut(line, "=")
			if ok && strings.TrimSpace(name) != "" {
				seen[strings.TrimSpace(name)] = struct{}{}
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	missing := make([]EnvVar, 0)
	for _, item := range vars {
		if !item.Required {
			continue
		}
		if _, ok := seen[item.Name]; ok {
			continue
		}
		found := false
		for _, alias := range item.Aliases {
			if _, ok := seen[alias]; ok {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, item)
		}
	}
	return missing, nil
}

func findDotEnvFromWD() (string, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}
	return findDotEnv(wd)
}

func findDotEnv(start string) (string, bool) {
	dir := filepath.Clean(start)
	for {
		path := filepath.Join(dir, ".env")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func collectEnvVars(typ reflect.Type, val reflect.Value, vars *[]EnvVar) error {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)
		tag := strings.TrimSpace(field.Tag.Get("env"))

		if fieldValue.Kind() == reflect.Struct && field.Type != reflect.TypeOf(time.Duration(0)) && tag == "" {
			if err := collectEnvVars(field.Type, fieldValue, vars); err != nil {
				return err
			}
			continue
		}

		if tag == "" || tag == "-" {
			continue
		}

		envList := splitEnvTag(tag)
		if len(envList) == 0 {
			continue
		}

		item := EnvVar{
			Name:     envList[0],
			Type:     formatEnvType(field.Type),
			Required: strings.EqualFold(field.Tag.Get("env-required"), "true"),
		}
		if len(envList) > 1 {
			item.Aliases = envList[1:]
		}
		if def := formatDefaultValue(fieldValue); def != "" {
			item.Default = def
		}

		*vars = append(*vars, item)
	}
	return nil
}

func splitEnvTag(tag string) []string {
	parts := strings.Split(tag, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && part != "-" {
			out = append(out, part)
		}
	}
	return out
}

func formatEnvType(typ reflect.Type) string {
	if typ == reflect.TypeOf(time.Duration(0)) {
		return "duration"
	}
	switch typ.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "int"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "uint"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return "bytes"
		}
		return "slice"
	default:
		return typ.Kind().String()
	}
}

func formatDefaultValue(v reflect.Value) string {
	if !v.IsValid() || v.IsZero() {
		return ""
	}
	if v.Type() == reflect.TypeOf(time.Duration(0)) {
		return v.Interface().(time.Duration).String()
	}
	return strings.TrimSpace(strings.Trim(vToString(v), "\""))
}

func vToString(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	default:
		return ""
	}
}
