package uxflashmessages

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/leomorpho/goship/framework/testkit"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

func TestMsg(t *testing.T) {
	e := echo.New()
	ctx, _ := tests.NewContext(e, "/")
	tests.InitSession(ctx)

	assertMsg := func(typ Type, message string) {
		ret := Get(ctx, typ)
		require.Len(t, ret, 1)
		assert.Equal(t, message, ret[0])
		ret = Get(ctx, typ)
		require.Len(t, ret, 0)
	}

	text := "aaa"
	Success(ctx, text)
	assertMsg(TypeSuccess, text)

	text = "bbb"
	Info(ctx, text)
	assertMsg(TypeInfo, text)

	text = "ccc"
	Danger(ctx, text)
	assertMsg(TypeDanger, text)

	text = "ddd"
	Warning(ctx, text)
	assertMsg(TypeWarning, text)

	text = "eee"
	Set(ctx, TypeSuccess, text)
	assertMsg(TypeSuccess, text)
}

type failingStore struct{}

func (failingStore) Get(_ *http.Request, _ string) (*sessions.Session, error) {
	return nil, fmt.Errorf("store down")
}

func (failingStore) New(_ *http.Request, name string) (*sessions.Session, error) {
	return sessions.NewSession(failingStore{}, name), nil
}

func (failingStore) Save(_ *http.Request, _ http.ResponseWriter, _ *sessions.Session) error {
	return fmt.Errorf("save failed")
}

func TestMsg_SessionFailurePolicy(t *testing.T) {
	e := echo.New()
	ctx, _ := tests.NewContext(e, "/")

	var buf bytes.Buffer
	logger := log.New("uxflashmessages-test")
	logger.SetOutput(&buf)
	ctx.SetLogger(logger)

	err := tests.ExecuteMiddleware(ctx, session.Middleware(failingStore{}))
	require.NoError(t, err)

	_ = Get(ctx, TypeInfo)
	assert.Contains(t, buf.String(), "cannot load flash message session for read")

	buf.Reset()
	Set(ctx, TypeInfo, "message")
	assert.Contains(t, buf.String(), "cannot load flash message session for write")
}
