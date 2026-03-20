package anthropic

import (
	"context"
	"fmt"
	"strings"

	sdkanthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/leomorpho/goship/modules/ai"
)

type AnthropicDriver struct {
	client       sdkanthropic.Client
	defaultModel string
}

func New(apiKey string, defaultModel string, opts ...option.RequestOption) *AnthropicDriver {
	if strings.TrimSpace(defaultModel) == "" {
		defaultModel = ai.ClaudeHaiku4
	}

	opts = append(opts, option.WithAPIKey(apiKey))

	return &AnthropicDriver{
		client:       sdkanthropic.NewClient(opts...),
		defaultModel: defaultModel,
	}
}

func (d *AnthropicDriver) Complete(ctx context.Context, req ai.Request) (*ai.Response, error) {
	params := d.buildParams(req)
	message, err := d.client.Messages.New(ctx, params)
	if err != nil {
		return nil, err
	}

	content, err := responseContent(*message, req.Schema != nil)
	if err != nil {
		return nil, err
	}

	return &ai.Response{
		Content:      content,
		InputTokens:  int(message.Usage.InputTokens),
		OutputTokens: int(message.Usage.OutputTokens),
		Model:        string(message.Model),
		FinishReason: string(message.StopReason),
	}, nil
}

func (d *AnthropicDriver) Stream(ctx context.Context, req ai.Request) (<-chan ai.Token, error) {
	params := d.buildParams(req)
	stream := d.client.Messages.NewStreaming(ctx, params)

	out := make(chan ai.Token)
	go func() {
		defer close(out)

		var message sdkanthropic.Message
		for stream.Next() {
			event := stream.Current()
			if err := message.Accumulate(event); err != nil {
				out <- ai.Token{Error: err, Done: true}
				return
			}

			switch variant := event.AsAny().(type) {
			case sdkanthropic.ContentBlockDeltaEvent:
				switch delta := variant.Delta.AsAny().(type) {
				case sdkanthropic.TextDelta:
					out <- ai.Token{Content: delta.Text}
				case sdkanthropic.InputJSONDelta:
					out <- ai.Token{Content: delta.PartialJSON}
				}
			}
		}

		if err := stream.Err(); err != nil {
			out <- ai.Token{Error: err, Done: true}
			return
		}

		if _, err := responseContent(message, req.Schema != nil); err != nil {
			out <- ai.Token{Error: err, Done: true}
			return
		}

		out <- ai.Token{Done: true}
	}()

	return out, nil
}

func (d *AnthropicDriver) buildParams(req ai.Request) sdkanthropic.MessageNewParams {
	req = applyDriverDefaults(req, d.defaultModel)

	params := sdkanthropic.MessageNewParams{
		MaxTokens: int64(req.MaxTokens),
		Messages:  make([]sdkanthropic.MessageParam, 0, len(req.Messages)),
		Model:     sdkanthropic.Model(req.Model),
	}

	if req.System != "" {
		params.System = []sdkanthropic.TextBlockParam{{Text: req.System}}
	}
	if req.Temperature > 0 {
		params.Temperature = sdkanthropic.Float(float64(req.Temperature))
	}

	for _, msg := range req.Messages {
		switch strings.ToLower(strings.TrimSpace(msg.Role)) {
		case "assistant":
			params.Messages = append(params.Messages, sdkanthropic.NewAssistantMessage(sdkanthropic.NewTextBlock(msg.Content)))
		default:
			params.Messages = append(params.Messages, sdkanthropic.NewUserMessage(sdkanthropic.NewTextBlock(msg.Content)))
		}
	}

	tools := make([]sdkanthropic.ToolUnionParam, 0, len(req.Tools)+1)
	for _, tool := range req.Tools {
		tools = append(tools, sdkanthropic.ToolUnionParam{
			OfTool: &sdkanthropic.ToolParam{
				Name:        tool.Name,
				Description: sdkanthropic.String(tool.Description),
				InputSchema: toToolInputSchema(tool.InputSchema),
			},
		})
	}

	if req.Schema != nil {
		tools = append(tools, sdkanthropic.ToolUnionParam{
			OfTool: &sdkanthropic.ToolParam{
				Name:        "structured_output",
				Description: sdkanthropic.String("Return the final answer as JSON that matches the provided schema."),
				InputSchema: toToolInputSchema(req.Schema),
				Strict:      sdkanthropic.Bool(true),
			},
		})
		params.ToolChoice = sdkanthropic.ToolChoiceParamOfTool("structured_output")
	}

	if len(tools) > 0 {
		params.Tools = tools
	}

	return params
}

func applyDriverDefaults(req ai.Request, defaultModel string) ai.Request {
	if strings.TrimSpace(req.Model) == "" {
		req.Model = defaultModel
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 1024
	}
	return req
}

func toToolInputSchema(input any) sdkanthropic.ToolInputSchemaParam {
	schema := ai.ToolSchema(input)
	properties, _ := schema["properties"]
	required, _ := schema["required"].([]string)

	result := sdkanthropic.ToolInputSchemaParam{
		Properties: properties,
		Required:   required,
	}
	if extra, ok := schema["additionalProperties"]; ok {
		result.ExtraFields = map[string]any{"additionalProperties": extra}
	}
	return result
}

func responseContent(message sdkanthropic.Message, structured bool) (string, error) {
	var text strings.Builder

	for _, block := range message.Content {
		switch variant := block.AsAny().(type) {
		case sdkanthropic.TextBlock:
			text.WriteString(variant.Text)
		case sdkanthropic.ToolUseBlock:
			if structured && variant.Name == "structured_output" {
				return string(variant.Input), nil
			}
		}
	}

	if structured {
		if content := strings.TrimSpace(text.String()); content != "" {
			return content, nil
		}
		return "", fmt.Errorf("structured output tool response missing")
	}

	return strings.TrimSpace(text.String()), nil
}
