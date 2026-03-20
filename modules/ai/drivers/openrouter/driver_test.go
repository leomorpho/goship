package openrouterdriver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leomorpho/goship/modules/ai"
	openai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/require"
)

func TestOpenRouterDriverComplete_UsesOpenRouterHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/chat/completions", r.URL.Path)
		require.Equal(t, "https://example.test", r.Header.Get("HTTP-Referer"))
		require.Equal(t, "Example App", r.Header.Get("X-Title"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"anthropic/claude-haiku-4-5-20251001","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1,"total_tokens":3}}`)
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL + "/api/v1"
	config.HTTPClient = &headerClient{
		base: http.DefaultClient,
		headers: map[string]string{
			"HTTP-Referer": "https://example.test",
			"X-Title":      "Example App",
		},
	}

	driver := NewWithConfig(config, ai.ORClaudeHaiku4)
	resp, err := driver.Complete(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: "user", Content: "ping"}},
	})
	require.NoError(t, err)
	require.Equal(t, "ok", resp.Content)
	require.Equal(t, ai.ORClaudeHaiku4, resp.Model)
}

func TestOpenRouterDriverStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"anthropic/claude-haiku-4-5-20251001\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"o\"},\"finish_reason\":\"\"}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"anthropic/claude-haiku-4-5-20251001\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"k\"},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL + "/api/v1"
	config.HTTPClient = &headerClient{base: http.DefaultClient, headers: map[string]string{}}

	driver := NewWithConfig(config, ai.ORClaudeHaiku4)
	stream, err := driver.Stream(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: "user", Content: "ping"}},
	})
	require.NoError(t, err)

	var got []string
	for token := range stream {
		require.NoError(t, token.Error)
		if token.Content != "" {
			got = append(got, token.Content)
		}
	}

	require.Equal(t, []string{"o", "k"}, got)
}

func TestOpenRouterDriverRequestBodyUsesProvidedModel(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"openai/gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL + "/api/v1"
	config.HTTPClient = &headerClient{base: http.DefaultClient, headers: map[string]string{}}

	driver := NewWithConfig(config, ai.ORGPTo4Mini)
	_, err := driver.Complete(context.Background(), ai.Request{
		Model:    ai.ORGPTo4Mini,
		Messages: []ai.Message{{Role: "user", Content: "ping"}},
	})
	require.NoError(t, err)
	require.Equal(t, ai.ORGPTo4Mini, body["model"])
}

func TestOpenRouterDriverOmitsAttributionHeadersWhenBlankAndUsesDefaultModel(t *testing.T) {
	var (
		body        map[string]any
		refererSeen string
		titleSeen   string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refererSeen = r.Header.Get("HTTP-Referer")
		titleSeen = r.Header.Get("X-Title")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"anthropic/claude-haiku-4-5-20251001","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL + "/api/v1"
	driver := New("test-key", "", " ", " ")
	driver.Client = openai.NewClientWithConfig(config)

	_, err := driver.Complete(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: "user", Content: "ping"}},
	})
	require.NoError(t, err)
	require.Equal(t, "", refererSeen)
	require.Equal(t, "", titleSeen)
	require.Equal(t, ai.ORClaudeHaiku4, body["model"])
}
