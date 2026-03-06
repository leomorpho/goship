package ship

import (
	"os"

	cmd "github.com/leomorpho/goship/tools/cli/ship/internal/commands"
	rt "github.com/leomorpho/goship/tools/cli/ship/internal/runtime"
)

func findGoModule(start string) (string, string, error) {
	return rt.FindGoModule(start)
}

func hasFile(path string) bool {
	return rt.HasFile(path)
}

func hasMakefile() bool {
	return rt.HasMakefile()
}

func relocateTemplGenerated(rootPath string) error {
	return rt.RelocateTemplGenerated(rootPath)
}

func resolveComposeCommand() ([]string, error) {
	return rt.ResolveComposeCommand()
}

func resolveComposeCommandWith(lookPath func(string) (string, error), dockerComposeVersion func() error) ([]string, error) {
	return rt.ResolveComposeCommandWith(lookPath, dockerComposeVersion)
}

func (c CLI) resolveDBURL() (string, error) {
	if c.ResolveDBURL != nil {
		return c.ResolveDBURL()
	}
	return resolveAtlasDBURL()
}

func resolveAtlasDBURL() (string, error) {
	return rt.ResolveDBURL()
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isLocalDBURL(dbURL string) bool {
	return cmd.IsLocalDBURL(dbURL)
}
