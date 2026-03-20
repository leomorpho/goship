package generators

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type MakeFactoryOptions struct {
	Name string
}

type FactoryDeps struct {
	Out io.Writer
	Err io.Writer
	Cwd string
}

func RunMakeFactory(args []string, d FactoryDeps) int {
	if len(args) > 0 {
		switch args[0] {
		case "help", "-h", "--help":
			fmt.Fprintln(d.Out, "usage: ship make:factory <Name>")
			return 0
		}
	}

	opts, err := ParseMakeFactoryArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid make:factory arguments: %v\n", err)
		return 1
	}

	cwd := d.Cwd
	if strings.TrimSpace(cwd) == "" {
		var wdErr error
		cwd, wdErr = os.Getwd()
		if wdErr != nil {
			fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", wdErr)
			return 1
		}
	}

	tokens := splitWords(opts.Name)
	if len(tokens) == 0 {
		fmt.Fprintln(d.Err, "invalid make:factory arguments: usage: ship make:factory <Name>")
		return 1
	}

	pascal := toPascalFromParts(tokens)
	snake := strings.Join(tokens, "_")
	plural := snake + "s"

	target := filepath.Join(cwd, "tests", "factories", snake+"_factory.go")
	if _, err := os.Stat(target); err == nil {
		fmt.Fprintf(d.Err, "refusing to overwrite existing factory file: %s\n", target)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		fmt.Fprintf(d.Err, "failed to create factory directory: %v\n", err)
		return 1
	}

	content := renderFactoryFile(pascal, plural)
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write factory file: %v\n", err)
		return 1
	}

	fmt.Fprintf(d.Out, "Generated factory: %s\n", target)
	return 0
}

func ParseMakeFactoryArgs(args []string) (MakeFactoryOptions, error) {
	opts := MakeFactoryOptions{}
	if len(args) == 0 {
		return opts, errors.New("usage: ship make:factory <Name>")
	}
	opts.Name = strings.TrimSpace(args[0])
	if opts.Name == "" || strings.HasPrefix(opts.Name, "-") {
		return opts, errors.New("usage: ship make:factory <Name>")
	}
	for i := 1; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			return opts, fmt.Errorf("unknown option: %s", args[i])
		}
		return opts, fmt.Errorf("unexpected argument: %s", args[i])
	}
	return opts, nil
}

func renderFactoryFile(pascal, table string) string {
	return fmt.Sprintf(`package factories

import (
	"time"

	"github.com/leomorpho/goship/framework/factory"
)

type %[1]sRecord struct {
	ID        int64     `+"`db:\"id\"`"+`
	CreatedAt time.Time `+"`db:\"created_at\"`"+`
}

func (%[1]sRecord) TableName() string { return %[2]q }

var %[1]s = factory.New(func() %[1]sRecord {
	return %[1]sRecord{
		CreatedAt: time.Now().UTC(),
	}
})
`, pascal, table)
}
