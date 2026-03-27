package middleware

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/leomorpho/goship/framework/tests"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogRequestID(t *testing.T) {
	e := echo.New()
	ctx, _ := tests.NewContext(e, "/")
	logger := log.New("test")
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	ctx.SetLogger(logger)

	err := tests.ExecuteMiddleware(ctx, echomw.RequestID())
	require.NoError(t, err)

	err = tests.ExecuteMiddleware(ctx, LogRequestID())
	require.NoError(t, err)

	ctx.Logger().Info("test")

	rID := ctx.Response().Header().Get(echo.HeaderXRequestID)
	require.NotEmpty(t, rID)
	assert.Contains(t, buf.String(), fmt.Sprintf(`"id":"%s"`, rID))
}
