package sse

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestNewSetsSSEHeaders(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest("GET", "/stream", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	stream, err := New(ctx)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if stream == nil {
		t.Fatalf("New() returned nil stream")
	}
	if got := rec.Header().Get(echo.HeaderContentType); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := rec.Header().Get("X-Accel-Buffering"); got != "no" {
		t.Fatalf("X-Accel-Buffering = %q", got)
	}
}

func TestStreamSendFormatsMessage(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest("GET", "/stream", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	stream, err := New(ctx)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := stream.Send("message", "<div>hello</div>"); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if got := rec.Body.String(); got != "event: message\ndata: <div>hello</div>\n\n" {
		t.Fatalf("body = %q", got)
	}
	if !rec.Flushed {
		t.Fatalf("expected response to flush")
	}
}

func TestStreamWaitReturnsOnCancel(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest("GET", "/stream", nil)
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	stream, err := New(ctx)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	done := make(chan struct{})
	go func() {
		stream.Wait()
		close(done)
	}()

	cancelCtx, cancel := context.WithCancel(req.Context())
	ctx.SetRequest(req.WithContext(cancelCtx))
	stream.ctx = cancelCtx
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Wait() did not return after cancellation")
	}
}

func TestStreamSendReturnsContextError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest("GET", "/stream", nil)
	cancelCtx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(cancelCtx)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	stream, err := New(ctx)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := stream.SendMessage("ignored"); !errors.Is(err, context.Canceled) {
		t.Fatalf("SendMessage() error = %v, want context.Canceled", err)
	}
}
