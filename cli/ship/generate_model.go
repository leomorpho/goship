package ship

import (
	"fmt"
	"regexp"
	"strings"
)

var modelNamePattern = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)

func (c CLI) runGenerateModel(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(c.Err, "usage: ship generate model <Name> [fields...]")
		return 1
	}

	name := strings.TrimSpace(args[0])
	if !modelNamePattern.MatchString(name) {
		fmt.Fprintf(c.Err, "invalid model name %q: use PascalCase (e.g. Post, BlogPost)\n", name)
		return 1
	}

	if len(args) > 1 {
		fmt.Fprintf(c.Err, "note: field scaffolding is not yet implemented; create fields in ent/schema/%s.go\n", strings.ToLower(name))
	}

	if code := c.runCmd("go", "run", "-mod=mod", "entgo.io/ent/cmd/ent", "new", name); code != 0 {
		return code
	}

	return c.runCmd("go", "run", "-mod=mod", "entgo.io/ent/cmd/ent", "generate", "--feature", "sql/upsert,sql/execquery", "./ent/schema")
}
