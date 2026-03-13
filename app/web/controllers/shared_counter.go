package controllers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/sse"
)

var sharedCounterValue atomic.Int64

type sharedCounter struct {
	ctr ui.Controller
}

func NewSharedCounterRoute(ctr ui.Controller) *sharedCounter {
	return &sharedCounter{ctr: ctr}
}

func (r *sharedCounter) Get(ctx echo.Context) error {
	html := fmt.Sprintf(`<!doctype html>
<html><body>
<h1>Shared Counter</h1>
<p>This page demonstrates one write broadcasting to every open tab.</p>
<div id="shared-counter-value"><strong>%d</strong></div>
<button id="increment" type="button">Increment</button>
<script>
const value = document.getElementById("shared-counter-value");
const source = new EventSource("/examples/shared-counter/stream");
source.addEventListener("message", function (event) {
  value.innerHTML = event.data;
});
document.getElementById("increment").addEventListener("click", async function () {
  await fetch("/examples/shared-counter/increment", { method: "POST" });
});
</script>
</body></html>`, sharedCounterValue.Load())

	return ctx.HTML(http.StatusOK, html)
}

func (r *sharedCounter) Stream(ctx echo.Context) error {
	stream, err := sse.New(ctx)
	if err != nil {
		return err
	}
	if r.ctr.Container.SSEHub == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "sse hub is not configured")
	}

	ch, unsubscribe := r.ctr.Container.SSEHub.Subscribe("counter:shared")
	defer unsubscribe()

	if err := stream.SendMessage(renderSharedCounterValue(sharedCounterValue.Load())); err != nil {
		return err
	}

	for {
		select {
		case msg := <-ch:
			if err := stream.SendMessage(msg); err != nil {
				return nil
			}
		case <-ctx.Request().Context().Done():
			return nil
		}
	}
}

func (r *sharedCounter) Increment(ctx echo.Context) error {
	if r.ctr.Container.SSEHub == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "sse hub is not configured")
	}

	value := sharedCounterValue.Add(1)
	if err := r.ctr.Container.SSEHub.PublishHTML("counter:shared", templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, renderSharedCounterValue(value))
		return err
	})); err != nil {
		return err
	}

	return ctx.NoContent(http.StatusAccepted)
}

func renderSharedCounterValue(value int64) string {
	return fmt.Sprintf("<strong>%d</strong>", value)
}
