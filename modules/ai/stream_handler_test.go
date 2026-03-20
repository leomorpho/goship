package ai

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStreamCompletionWritesSSEEvents(t *testing.T) {
	provider := mockProvider{
		complete: func(context.Context, Request) (*Response, error) {
			return nil, nil
		},
		stream: func(context.Context, Request) (<-chan Token, error) {
			ch := make(chan Token, 3)
			ch <- Token{Content: "hel"}
			ch <- Token{Content: "lo"}
			ch <- Token{Done: true}
			close(ch)
			return ch, nil
		},
	}

	rec := httptest.NewRecorder()
	err := StreamCompletion(context.Background(), rec, Request{
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, provider)
	require.NoError(t, err)
	require.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "event: message")
	require.Contains(t, rec.Body.String(), "data: hel")
	require.Contains(t, rec.Body.String(), "data: lo")
	require.Contains(t, rec.Body.String(), "event: done")
	require.Contains(t, rec.Body.String(), "data: complete")
}

func TestStreamCompletionReturnsProviderError(t *testing.T) {
	expected := streamTestError("stream broke")
	provider := mockProvider{
		complete: func(context.Context, Request) (*Response, error) {
			return nil, nil
		},
		stream: func(context.Context, Request) (<-chan Token, error) {
			ch := make(chan Token, 1)
			ch <- Token{Error: expected, Done: true}
			close(ch)
			return ch, nil
		},
	}

	rec := httptest.NewRecorder()
	err := StreamCompletion(context.Background(), rec, Request{
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, provider)
	require.ErrorIs(t, err, expected)
	require.Contains(t, rec.Body.String(), "event: error")
	require.Contains(t, rec.Body.String(), "stream broke")
}

func TestStreamCompletionReturnsErrorWhenProviderUnavailable(t *testing.T) {
	rec := httptest.NewRecorder()
	err := StreamCompletion(context.Background(), rec, Request{
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, nil)
	require.EqualError(t, err, "ai provider unavailable")
}

func TestStreamCompletionWritesDoneWhenStreamClosesWithoutDoneToken(t *testing.T) {
	provider := mockProvider{
		complete: func(context.Context, Request) (*Response, error) {
			return nil, nil
		},
		stream: func(context.Context, Request) (<-chan Token, error) {
			ch := make(chan Token, 2)
			ch <- Token{Content: "partial"}
			close(ch)
			return ch, nil
		},
	}

	rec := httptest.NewRecorder()
	err := StreamCompletion(context.Background(), rec, Request{
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, provider)
	require.NoError(t, err)
	require.Contains(t, rec.Body.String(), "data: partial")
	require.Contains(t, rec.Body.String(), "event: done")
}

type streamTestError string

func (e streamTestError) Error() string {
	return string(e)
}
