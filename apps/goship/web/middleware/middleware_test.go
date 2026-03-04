package middleware

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leomorpho/goship/apps/goship/foundation"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/ent"
	"github.com/leomorpho/goship/pkg/tests"
)

var (
	c   *foundation.Container
	usr *ent.User
)

func TestMain(m *testing.M) {
	if err := chdirRepoRoot(); err != nil {
		panic(err)
	}

	// Set the environment to test
	config.SwitchEnvironment(config.EnvTest)

	// Create a new container
	c = foundation.NewContainer()

	// Create a user
	var err error
	if usr, err = tests.CreateRandomUser(c.ORM); err != nil {
		panic(err)
	}

	// Run tests
	exitVal := m.Run()

	// Shutdown the container
	if err = c.Shutdown(); err != nil {
		panic(err)
	}

	os.Exit(exitVal)
}

func chdirRepoRoot() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return os.Chdir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return os.ErrNotExist
		}
		dir = parent
	}
}
