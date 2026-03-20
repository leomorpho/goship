package ai

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockProvider struct {
	complete func(context.Context, Request) (*Response, error)
	stream   func(context.Context, Request) (<-chan Token, error)
}

func (m mockProvider) Complete(ctx context.Context, req Request) (*Response, error) {
	return m.complete(ctx, req)
}

func (m mockProvider) Stream(ctx context.Context, req Request) (<-chan Token, error) {
	return m.stream(ctx, req)
}

func TestServiceComplete_DefaultsModelAndMaxTokens(t *testing.T) {
	var seen Request
	svc := NewService(mockProvider{
		complete: func(_ context.Context, req Request) (*Response, error) {
			seen = req
			return &Response{Content: "ok", Model: req.Model}, nil
		},
		stream: func(context.Context, Request) (<-chan Token, error) {
			return nil, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	resp, err := svc.Complete(context.Background(), Request{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	require.NoError(t, err)
	require.Equal(t, ClaudeHaiku4, seen.Model)
	require.Equal(t, 1024, seen.MaxTokens)
	require.Equal(t, ClaudeHaiku4, resp.Model)
}

func TestServiceComplete_DecodesStructuredOutput(t *testing.T) {
	type structured struct {
		Answer string `json:"Answer"`
		Count  int    `json:"Count"`
	}

	var result structured
	svc := NewService(mockProvider{
		complete: func(context.Context, Request) (*Response, error) {
			return &Response{
				Content: `{"Answer":"hi","Count":2}`,
				Model:   ClaudeHaiku4,
			}, nil
		},
		stream: func(context.Context, Request) (<-chan Token, error) {
			return nil, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	resp, err := svc.Complete(context.Background(), Request{
		Messages: []Message{{Role: "user", Content: "hello"}},
		Schema:   &result,
	})
	require.NoError(t, err)
	require.Equal(t, `{"Answer":"hi","Count":2}`, resp.Content)
	require.Equal(t, structured{Answer: "hi", Count: 2}, result)
}

func TestServiceStream_DecodesStructuredOutput(t *testing.T) {
	type structured struct {
		Status string `json:"Status"`
	}

	tokens := make(chan Token, 3)
	tokens <- Token{Content: `{"Status":"`}
	tokens <- Token{Content: `ready"}`}
	tokens <- Token{Done: true}
	close(tokens)

	var result structured
	svc := NewService(mockProvider{
		complete: func(context.Context, Request) (*Response, error) {
			return nil, nil
		},
		stream: func(context.Context, Request) (<-chan Token, error) {
			return tokens, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	stream, err := svc.Stream(context.Background(), Request{
		Messages: []Message{{Role: "user", Content: "hello"}},
		Schema:   &result,
	})
	require.NoError(t, err)

	var got []Token
	for token := range stream {
		got = append(got, token)
	}

	require.Len(t, got, 3)
	require.True(t, got[2].Done)
	require.NoError(t, got[2].Error)
	require.Equal(t, structured{Status: "ready"}, result)
}

func TestServiceComplete_PropagatesProviderError(t *testing.T) {
	expected := errors.New("boom")
	svc := NewService(mockProvider{
		complete: func(context.Context, Request) (*Response, error) {
			return nil, expected
		},
		stream: func(context.Context, Request) (<-chan Token, error) {
			return nil, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	_, err := svc.Complete(context.Background(), Request{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	require.ErrorIs(t, err, expected)
}
