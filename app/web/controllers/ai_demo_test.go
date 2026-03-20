package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/app/web/ui"
	aimodule "github.com/leomorpho/goship/modules/ai"
)

type fakeAIProvider struct {
	stream func(context.Context, aimodule.Request) (<-chan aimodule.Token, error)
}

func (f fakeAIProvider) Complete(context.Context, aimodule.Request) (*aimodule.Response, error) {
	return nil, nil
}

func (f fakeAIProvider) Stream(ctx context.Context, req aimodule.Request) (<-chan aimodule.Token, error) {
	return f.stream(ctx, req)
}

func TestAIDemoStream_WritesSSETokenStreamAndDoneEvent(t *testing.T) {
	tokens := make(chan aimodule.Token, 3)
	tokens <- aimodule.Token{Content: "hel"}
	tokens <- aimodule.Token{Content: "lo"}
	tokens <- aimodule.Token{Done: true}
	close(tokens)

	container := &foundation.Container{
		AI: aimodule.NewService(fakeAIProvider{
			stream: func(context.Context, aimodule.Request) (<-chan aimodule.Token, error) {
				return tokens, nil
			},
		}, nil),
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/auth/ai-demo/stream?prompt=hello", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	route := NewAIDemoRoute(ui.NewController(container))
	if err := route.Stream(ctx); err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected text/event-stream content type, got %q", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	if body == "" {
		t.Fatal("expected SSE response body")
	}
	if want := "event: message"; !strings.Contains(body, want) {
		t.Fatalf("expected body to contain %q, got %q", want, body)
	}
	if want := "data: hel"; !strings.Contains(body, want) {
		t.Fatalf("expected body to contain %q, got %q", want, body)
	}
	if want := "data: lo"; !strings.Contains(body, want) {
		t.Fatalf("expected body to contain %q, got %q", want, body)
	}
	if want := "event: done"; !strings.Contains(body, want) {
		t.Fatalf("expected body to contain %q, got %q", want, body)
	}
}
