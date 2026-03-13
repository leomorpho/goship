package ai

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/stretchr/testify/require"
)

func TestConversationServiceSendMessagePersistsHistory(t *testing.T) {
	db := newConversationTestDB(t)
	store := NewConversationSQLStore(db, "sqlite")
	provider := mockProvider{
		complete: func(_ context.Context, req Request) (*Response, error) {
			require.Len(t, req.Messages, 1)
			require.Equal(t, "user", req.Messages[0].Role)
			require.Equal(t, "Hello from the user", req.Messages[0].Content)
			return &Response{
				Content:      "Hello from the assistant",
				InputTokens:  12,
				OutputTokens: 5,
				Model:        ClaudeHaiku4,
			}, nil
		},
		stream: func(context.Context, Request) (<-chan Token, error) {
			return nil, nil
		},
	}

	service := NewConversationService(store, provider)
	conversation, err := service.CreateConversation(context.Background(), 1, ClaudeHaiku4, "")
	require.NoError(t, err)

	stream, err := service.SendMessage(context.Background(), conversation.ID, "Hello from the user")
	require.NoError(t, err)

	var streamed strings.Builder
	for token := range stream {
		require.NoError(t, token.Error)
		if token.Content != "" {
			streamed.WriteString(token.Content)
		}
	}
	require.Equal(t, "Hello from the assistant", streamed.String())

	history, err := service.GetHistory(context.Background(), conversation.ID)
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Equal(t, "user", history[0].Role)
	require.Equal(t, "Hello from the user", history[0].Content)
	require.Equal(t, "assistant", history[1].Role)
	require.Equal(t, "Hello from the assistant", history[1].Content)
	require.Equal(t, 12, history[1].InputTokens)
	require.Equal(t, 5, history[1].OutputTokens)
	require.Equal(t, ClaudeHaiku4, history[1].Model)

	conversations, err := service.ListConversations(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, conversations, 1)
	require.Equal(t, "Hello from the user", conversations[0].Title)
}

func TestConversationSQLStoreListMessagesOrdersByCreatedAt(t *testing.T) {
	db := newConversationTestDB(t)
	store := NewConversationSQLStore(db, "sqlite")

	conversation, err := store.CreateConversation(context.Background(), 1, ClaudeHaiku4, "Test")
	require.NoError(t, err)

	first, err := store.AppendMessage(context.Background(), ConversationMessage{
		ConversationID: conversation.ID,
		Role:           "user",
		Content:        "one",
	})
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	second, err := store.AppendMessage(context.Background(), ConversationMessage{
		ConversationID: conversation.ID,
		Role:           "assistant",
		Content:        "two",
	})
	require.NoError(t, err)

	messages, err := store.ListMessages(context.Background(), conversation.ID)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, first.ID, messages[0].ID)
	require.Equal(t, second.ID, messages[1].ID)
}

func newConversationTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:?_journal=WAL&_timeout=5000&_fk=true")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	schema := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL
		)`,
		`CREATE TABLE ai_conversations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			model TEXT NOT NULL,
			title TEXT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE ai_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id INTEGER NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			input_tokens INTEGER,
			output_tokens INTEGER,
			model TEXT,
			created_at DATETIME NOT NULL
		)`,
		`CREATE INDEX idx_ai_messages_conversation ON ai_messages(conversation_id, created_at)`,
		`INSERT INTO users (id, email) VALUES (1, 'test@example.com')`,
	}

	for _, stmt := range schema {
		_, err := db.Exec(stmt)
		require.NoError(t, err)
	}

	return db
}
