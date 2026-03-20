package sse

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type Stream struct {
	w       http.ResponseWriter
	flusher http.Flusher
	ctx     context.Context
}

func New(c echo.Context) (*Stream, error) {
	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming unsupported")
	}

	header := c.Response().Header()
	header.Set(echo.HeaderContentType, "text/event-stream")
	header.Set(echo.HeaderCacheControl, "no-cache")
	header.Set(echo.HeaderConnection, "keep-alive")
	header.Set("X-Accel-Buffering", "no")

	return &Stream{
		w:       c.Response().Writer,
		flusher: flusher,
		ctx:     c.Request().Context(),
	}, nil
}

func (s *Stream) Send(event, data string) error {
	if s == nil {
		return errors.New("stream is nil")
	}
	if err := s.ctx.Err(); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}

func (s *Stream) SendMessage(data string) error {
	return s.Send("message", data)
}

func (s *Stream) Wait() {
	if s == nil || s.ctx == nil {
		return
	}
	<-s.ctx.Done()
}

// Proxy forwards messages from the channel to the stream until the channel is closed
// or the client disconnects.
func (s *Stream) Proxy(ch <-chan string) error {
	if s == nil || s.ctx == nil {
		return errors.New("stream is nil")
	}
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			if err := s.SendMessage(msg); err != nil {
				return err
			}
		case <-s.ctx.Done():
			return nil // client disconnected
		}
	}
}
