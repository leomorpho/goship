package generators

import (
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type PolicyDeps struct {
	Out     io.Writer
	Err     io.Writer
	HasFile func(path string) bool
}

func RunMakePolicy(args []string, d PolicyDeps) int {
	name, force, err := parseMakePolicyArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "%v\n", err)
		return 1
	}
	fileName := ModelFileName(name)
	target := filepath.Join("app", "policies", fileName+".go")
	if d.HasFile != nil && d.HasFile(target) && !force {
		fmt.Fprintf(d.Err, "refusing to overwrite existing policy file %s (use --force)\n", target)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		fmt.Fprintf(d.Err, "failed to create policies directory: %v\n", err)
		return 1
	}
	content, err := renderPolicyTemplate(name)
	if err != nil {
		fmt.Fprintf(d.Err, "failed to render policy file: %v\n", err)
		return 1
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write policy file: %v\n", err)
		return 1
	}
	writeGeneratorReport(d.Out, "policy", false, []string{target}, nil, nil, []string{
		"Wire the generated policy into route/resource/admin checks as needed.",
	})
	return 0
}

func parseMakePolicyArgs(args []string) (string, bool, error) {
	if len(args) == 0 {
		return "", false, fmt.Errorf("usage: ship make:policy <Name> [--force]")
	}
	name := strings.TrimSpace(args[0])
	if !ModelNamePattern.MatchString(name) {
		return "", false, fmt.Errorf("invalid policy name %q: use PascalCase (e.g. AdminDashboard)", name)
	}
	force := false
	for _, arg := range args[1:] {
		switch strings.TrimSpace(arg) {
		case "--force":
			force = true
		default:
			return "", false, fmt.Errorf("usage: ship make:policy <Name> [--force]")
		}
	}
	return name, force, nil
}

func renderPolicyTemplate(name string) (string, error) {
	source := generatedGoFileHeader("policy", ModelFileName(name)) + fmt.Sprintf(`package policies

type %[1]sPolicy struct{}

func New%[1]sPolicy() %[1]sPolicy {
	return %[1]sPolicy{}
}

func (p %[1]sPolicy) Allows(actor PolicyActor) bool {
	return actor.IsAdmin
}
`, name)
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}
