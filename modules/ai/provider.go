package ai

import "context"

type Provider interface {
	Complete(ctx context.Context, req Request) (*Response, error)
	Stream(ctx context.Context, req Request) (<-chan Token, error)
}

type Request struct {
	Model       string
	System      string
	Messages    []Message
	MaxTokens   int
	Temperature float32
	Schema      any
	Tools       []Tool
}

type Message struct {
	Role    string
	Content string
}

type Tool struct {
	Name        string
	Description string
	InputSchema any
}

type Response struct {
	Content      string
	InputTokens  int
	OutputTokens int
	Model        string
	FinishReason string
}

type Token struct {
	Content string
	Done    bool
	Error   error
}
