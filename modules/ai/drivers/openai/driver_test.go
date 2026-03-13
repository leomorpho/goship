package openaidriver

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

func TestOpenAIDriverComplete(t *testing.T) {
	var body openai.ChatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-1",
			"object":  "chat.completion",
			"created": 1,
			"model":   "gpt-4o-mini",
			"choices": []map[string]any{{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": "hello world",
				},
				"finish_reason": "stop",
			}},
			"usage": map[string]any{
				"prompt_tokens":     11,
				"completion_tokens": 3,
				"total_tokens":      14,
			},
		}))
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL + "/v1"
	driver := NewWithConfig(config, ai.GPT4oMini)

	resp, err := driver.Complete(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: "user", Content: "say hi"}},
	})
	require.NoError(t, err)
	require.Equal(t, ai.GPT4oMini, body.Model)
	require.Equal(t, "hello world", resp.Content)
	require.Equal(t, 11, resp.InputTokens)
	require.Equal(t, 3, resp.OutputTokens)
}

func TestOpenAIDriverComplete_StructuredOutputUsesJSONSchema(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"{\"status\":\"ok\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL + "/v1"
	driver := NewWithConfig(config, ai.GPT4oMini)

	type schema struct {
		Status string `json:"status"`
	}

	_, err := driver.Complete(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: "user", Content: "respond"}},
		Schema:   &schema{},
	})
	require.NoError(t, err)

	responseFormat, ok := body["response_format"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "json_schema", responseFormat["type"])
}

func TestOpenAIDriverStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-4o-mini\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hel\"},\"finish_reason\":\"\"}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-4o-mini\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"lo\"},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	config := openai.DefaultConfig("test-key")
	config.BaseURL = server.URL + "/v1"
	driver := NewWithConfig(config, ai.GPT4oMini)

	stream, err := driver.Stream(context.Background(), ai.Request{
		Messages: []ai.Message{{Role: "user", Content: "say hello"}},
	})
	require.NoError(t, err)

	var parts []string
	done := false
	for token := range stream {
		require.NoError(t, token.Error)
		if token.Content != "" {
			parts = append(parts, token.Content)
		}
		if token.Done {
			done = true
		}
	}

	require.Equal(t, []string{"hel", "lo"}, parts)
	require.True(t, done)
}
