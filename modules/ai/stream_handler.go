package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func StreamCompletion(ctx context.Context, w http.ResponseWriter, req Request, provider Provider) error {
	if provider == nil {
		return errors.New("ai provider unavailable")
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		return errors.New("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	tokens, err := provider.Stream(ctx, req)
	if err != nil {
		return err
	}

	for token := range tokens {
		if token.Error != nil {
			if writeErr := writeSSEEvent(w, "error", token.Error.Error()); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return token.Error
		}
		if token.Content != "" {
			if writeErr := writeSSEEvent(w, "message", token.Content); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
		}
		if token.Done {
			if writeErr := writeSSEEvent(w, "done", "complete"); writeErr != nil {
				return writeErr
			}
			flusher.Flush()
			return nil
		}
	}

	if err := writeSSEEvent(w, "done", "complete"); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func writeSSEEvent(w http.ResponseWriter, event string, data string) error {
	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return err
	}
	for _, line := range strings.Split(data, "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, "\n")
	return err
}
