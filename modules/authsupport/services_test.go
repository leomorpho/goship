package authsupport_test

import (
	"os"
	"testing"

	"github.com/leomorpho/goship/config"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	"github.com/leomorpho/goship/framework/testkit"

	"github.com/labstack/echo/v4"
)

var (
	c   *frameworkbootstrap.Container
	ctx echo.Context
	usr *tests.UserRecord
)

func TestMain(m *testing.M) {
	// Set the environment to test
	config.SwitchEnvironment(config.EnvTest)

	// Create a new container
	c = frameworkbootstrap.NewContainer(nil)

	// Create a web context
	ctx, _ = tests.NewContext(c.Web, "/")
	tests.InitSession(ctx)

	// Create a test user
	var err error
	if usr, err = tests.CreateRandomUserDB(c.Database); err != nil {
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
