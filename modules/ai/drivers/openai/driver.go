package openaidriver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/leomorpho/goship/modules/ai"
	openai "github.com/sashabaranov/go-openai"
)

type OpenAIDriver struct {
	Client       *openai.Client
	DefaultModel string
}

func New(apiKey string, defaultModel string) *OpenAIDriver {
	config := openai.DefaultConfig(apiKey)
	return NewWithConfig(config, defaultModel)
}

func NewWithConfig(config openai.ClientConfig, defaultModel string) *OpenAIDriver {
	if strings.TrimSpace(defaultModel) == "" {
		defaultModel = ai.GPT4oMini
	}

	return &OpenAIDriver{
		Client:       openai.NewClientWithConfig(config),
		DefaultModel: defaultModel,
	}
}

func (d *OpenAIDriver) Complete(ctx context.Context, req ai.Request) (*ai.Response, error) {
	request, err := d.buildRequest(req, false)
	if err != nil {
		return nil, err
	}

	response, err := d.Client.CreateChatCompletion(ctx, request)
	if err != nil {
		return nil, err
	}
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("openai completion returned no choices")
	}

	content := responseContent(response.Choices[0].Message)
	return &ai.Response{
		Content:      content,
		InputTokens:  response.Usage.PromptTokens,
		OutputTokens: response.Usage.CompletionTokens,
		Model:        response.Model,
		FinishReason: string(response.Choices[0].FinishReason),
	}, nil
}

func (d *OpenAIDriver) Stream(ctx context.Context, req ai.Request) (<-chan ai.Token, error) {
	request, err := d.buildRequest(req, true)
	if err != nil {
		return nil, err
	}

	stream, err := d.Client.CreateChatCompletionStream(ctx, request)
	if err != nil {
		return nil, err
	}

	out := make(chan ai.Token)
	go func() {
		defer close(out)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err == io.EOF {
				out <- ai.Token{Done: true}
				return
			}
			if err != nil {
				out <- ai.Token{Error: err, Done: true}
				return
			}

			for _, choice := range response.Choices {
				if content := choice.Delta.Content; content != "" {
					out <- ai.Token{Content: content}
				}
				for _, toolCall := range choice.Delta.ToolCalls {
					if toolCall.Function.Arguments != "" {
						out <- ai.Token{Content: toolCall.Function.Arguments}
					}
				}
			}
		}
	}()

	return out, nil
}

func (d *OpenAIDriver) buildRequest(req ai.Request, stream bool) (openai.ChatCompletionRequest, error) {
	req = applyOpenAIDefaults(req, d.DefaultModel)

	request := openai.ChatCompletionRequest{
		Model:               req.Model,
		Messages:            make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1),
		MaxTokens:           req.MaxTokens,
		MaxCompletionTokens: req.MaxTokens,
		Temperature:         req.Temperature,
		Stream:              stream,
	}

	if stream {
		request.StreamOptions = &openai.StreamOptions{IncludeUsage: true}
	}

	if strings.TrimSpace(req.System) != "" {
		request.Messages = append(request.Messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		role := openai.ChatMessageRoleUser
		switch strings.ToLower(strings.TrimSpace(msg.Role)) {
		case "assistant":
			role = openai.ChatMessageRoleAssistant
		case "system":
			role = openai.ChatMessageRoleSystem
		}

		request.Messages = append(request.Messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	if len(req.Tools) > 0 {
		request.Tools = make([]openai.Tool, 0, len(req.Tools))
		for _, tool := range req.Tools {
			request.Tools = append(request.Tools, openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  ai.ToolSchema(tool.InputSchema),
					Strict:      true,
				},
			})
		}
	}

	if req.Schema != nil {
		schemaBytes, err := json.Marshal(ai.ToolSchema(req.Schema))
		if err != nil {
			return openai.ChatCompletionRequest{}, err
		}
		request.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   "structured_output",
				Schema: json.RawMessage(schemaBytes),
				Strict: true,
			},
		}
	}

	return request, nil
}

func applyOpenAIDefaults(req ai.Request, defaultModel string) ai.Request {
	if strings.TrimSpace(req.Model) == "" {
		req.Model = defaultModel
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 1024
	}
	return req
}

func responseContent(message openai.ChatCompletionMessage) string {
	if message.Content != "" {
		return strings.TrimSpace(message.Content)
	}
	if len(message.ToolCalls) > 0 {
		return strings.TrimSpace(message.ToolCalls[0].Function.Arguments)
	}
	if message.FunctionCall != nil {
		return strings.TrimSpace(message.FunctionCall.Arguments)
	}
	return ""
}
