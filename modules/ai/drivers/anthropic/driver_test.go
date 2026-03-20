package anthropic

import (
	"encoding/json"
	"testing"

	sdkanthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/leomorpho/goship/modules/ai"
	"github.com/stretchr/testify/require"
)

func TestBuildParams_AppliesDefaultsAndStructuredOutputTool(t *testing.T) {
	driver := &AnthropicDriver{defaultModel: ai.ClaudeHaiku4}

	type result struct {
		Status string `json:"status"`
	}

	params := driver.buildParams(ai.Request{
		System:      "You are helpful.",
		Temperature: 0.5,
		Messages: []ai.Message{
			{Role: "user", Content: "Say hello"},
			{Role: "assistant", Content: "Hello"},
		},
		Tools: []ai.Tool{{
			Name:        "lookup_weather",
			Description: "Look up weather",
			InputSchema: struct {
				City string `json:"city"`
			}{},
		}},
		Schema: &result{},
	})

	require.Equal(t, ai.ClaudeHaiku4, string(params.Model))
	require.Equal(t, int64(1024), params.MaxTokens)
	require.Len(t, params.Messages, 2)
	require.Len(t, params.Tools, 2)
	require.NotNil(t, params.ToolChoice.OfTool)
	require.Equal(t, "structured_output", params.ToolChoice.OfTool.Name)
	require.Len(t, params.System, 1)
	require.Equal(t, "You are helpful.", params.System[0].Text)
	require.NotNil(t, params.Temperature)
}

func TestResponseContent_ReturnsStructuredToolJSON(t *testing.T) {
	var message sdkanthropic.Message
	err := json.Unmarshal([]byte(`{
		"id":"msg_1",
		"type":"message",
		"role":"assistant",
		"model":"claude-haiku-4-5-20251001",
		"stop_reason":"end_turn",
		"content":[
			{
				"type":"tool_use",
				"id":"toolu_1",
				"name":"structured_output",
				"input":{"status":"ok","count":2}
			}
		],
		"usage":{"input_tokens":4,"output_tokens":2}
	}`), &message)
	require.NoError(t, err)

	content, err := responseContent(message, true)
	require.NoError(t, err)
	require.JSONEq(t, `{"status":"ok","count":2}`, content)
}

func TestResponseContent_ReturnsTrimmedTextForNonStructuredResponse(t *testing.T) {
	var message sdkanthropic.Message
	err := json.Unmarshal([]byte(`{
		"id":"msg_2",
		"type":"message",
		"role":"assistant",
		"model":"claude-haiku-4-5-20251001",
		"stop_reason":"end_turn",
		"content":[
			{"type":"text","text":" hello "},
			{"type":"text","text":"world "}
		],
		"usage":{"input_tokens":4,"output_tokens":2}
	}`), &message)
	require.NoError(t, err)

	content, err := responseContent(message, false)
	require.NoError(t, err)
	require.Equal(t, "hello world", content)
}
