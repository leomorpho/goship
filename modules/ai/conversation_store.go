package ai

import (
	"context"
	"time"
)

type Conversation struct {
	ID        int64
	UserID    int64
	Model     string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ConversationMessage struct {
	ID             int64
	ConversationID int64
	Role           string
	Content        string
	InputTokens    int
	OutputTokens   int
	Model          string
	CreatedAt      time.Time
}

type ConversationStore interface {
	CreateConversation(ctx context.Context, userID int64, model string, title string) (Conversation, error)
	GetConversation(ctx context.Context, conversationID int64) (Conversation, error)
	ListConversations(ctx context.Context, userID int64) ([]Conversation, error)
	ListMessages(ctx context.Context, conversationID int64) ([]ConversationMessage, error)
	AppendMessage(ctx context.Context, message ConversationMessage) (ConversationMessage, error)
	UpdateConversationTitle(ctx context.Context, conversationID int64, title string) error
}
