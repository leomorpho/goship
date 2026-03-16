package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	frameworklogging "github.com/leomorpho/goship/framework/logging"
)

func TestRecoverPanics_LogsAndKeepsServerAlive(t *testing.T) {
	logOut := &bytes.Buffer{}
	logger := frameworklogging.NewEchoLogger(slog.New(slog.NewJSONHandler(logOut, nil)))

	e := echo.New()
	e.Logger = logger
	e.Use(RecoverPanics(logger))
	e.GET("/panic", func(c echo.Context) error {
		panic("boom")
	})
	e.GET("/ok", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	panicReq := httptest.NewRequest(http.MethodGet, "/panic", nil)
	panicRec := httptest.NewRecorder()
	e.ServeHTTP(panicRec, panicReq)
	if panicRec.Code != http.StatusInternalServerError {
		t.Fatalf("panic status = %d, want 500", panicRec.Code)
	}

	okReq := httptest.NewRequest(http.MethodGet, "/ok", nil)
	okRec := httptest.NewRecorder()
	e.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("ok status = %d, want 200", okRec.Code)
	}
	if strings.TrimSpace(okRec.Body.String()) != "ok" {
		t.Fatalf("ok body = %q, want ok", okRec.Body.String())
	}

	logText := logOut.String()
	if !strings.Contains(logText, "\"error\":\"boom\"") {
		t.Fatalf("log output = %q, want panic error field", logText)
	}
	if !strings.Contains(logText, "\"stack\":") {
		t.Fatalf("log output = %q, want stack field", logText)
	}
	if !strings.Contains(logText, "\"path\":\"/panic\"") {
		t.Fatalf("log output = %q, want request path field", logText)
	}
}
