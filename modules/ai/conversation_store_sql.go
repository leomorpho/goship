package ai

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type ConversationSQLStore struct {
	db      *sql.DB
	dialect string
}

func NewConversationSQLStore(db *sql.DB, dialect string) *ConversationSQLStore {
	return &ConversationSQLStore{db: db, dialect: dialect}
}

func (s *ConversationSQLStore) CreateConversation(ctx context.Context, userID int64, model string, title string) (Conversation, error) {
	now := time.Now().UTC()
	normalizedModel := strings.TrimSpace(model)
	if normalizedModel == "" {
		normalizedModel = ClaudeHaiku4
	}

	if isPostgresDialect(s.dialect) {
		query := `INSERT INTO ai_conversations (user_id, model, title, created_at, updated_at)
VALUES (` + sqlPlaceholder(s.dialect, 1) + `, ` + sqlPlaceholder(s.dialect, 2) + `, ` + sqlPlaceholder(s.dialect, 3) + `, ` + sqlPlaceholder(s.dialect, 4) + `, ` + sqlPlaceholder(s.dialect, 5) + `)
RETURNING id`
		var id int64
		if err := s.db.QueryRowContext(ctx, query, userID, normalizedModel, strings.TrimSpace(title), now, now).Scan(&id); err != nil {
			return Conversation{}, err
		}
		return Conversation{
			ID:        id,
			UserID:    userID,
			Model:     normalizedModel,
			Title:     strings.TrimSpace(title),
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}

	query := `INSERT INTO ai_conversations (user_id, model, title, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)`
	result, err := s.db.ExecContext(ctx, query, userID, normalizedModel, strings.TrimSpace(title), now, now)
	if err != nil {
		return Conversation{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Conversation{}, err
	}
	return Conversation{
		ID:        id,
		UserID:    userID,
		Model:     normalizedModel,
		Title:     strings.TrimSpace(title),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *ConversationSQLStore) GetConversation(ctx context.Context, conversationID int64) (Conversation, error) {
	query := `SELECT id, user_id, model, COALESCE(title, ''), created_at, updated_at
FROM ai_conversations WHERE id = ` + sqlPlaceholder(s.dialect, 1) + ` LIMIT 1`
	var conversation Conversation
	if err := s.db.QueryRowContext(ctx, query, conversationID).Scan(
		&conversation.ID,
		&conversation.UserID,
		&conversation.Model,
		&conversation.Title,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	); err != nil {
		return Conversation{}, err
	}
	return conversation, nil
}

func (s *ConversationSQLStore) ListConversations(ctx context.Context, userID int64) ([]Conversation, error) {
	query := `SELECT id, user_id, model, COALESCE(title, ''), created_at, updated_at
FROM ai_conversations WHERE user_id = ` + sqlPlaceholder(s.dialect, 1) + `
ORDER BY updated_at DESC, id DESC`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var conversation Conversation
		if err := rows.Scan(
			&conversation.ID,
			&conversation.UserID,
			&conversation.Model,
			&conversation.Title,
			&conversation.CreatedAt,
			&conversation.UpdatedAt,
		); err != nil {
			return nil, err
		}
		conversations = append(conversations, conversation)
	}
	return conversations, rows.Err()
}

func (s *ConversationSQLStore) ListMessages(ctx context.Context, conversationID int64) ([]ConversationMessage, error) {
	query := `SELECT id, conversation_id, role, content, COALESCE(input_tokens, 0), COALESCE(output_tokens, 0), COALESCE(model, ''), created_at
FROM ai_messages WHERE conversation_id = ` + sqlPlaceholder(s.dialect, 1) + `
ORDER BY created_at ASC, id ASC`
	rows, err := s.db.QueryContext(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ConversationMessage
	for rows.Next() {
		var message ConversationMessage
		if err := rows.Scan(
			&message.ID,
			&message.ConversationID,
			&message.Role,
			&message.Content,
			&message.InputTokens,
			&message.OutputTokens,
			&message.Model,
			&message.CreatedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func (s *ConversationSQLStore) AppendMessage(ctx context.Context, message ConversationMessage) (ConversationMessage, error) {
	now := time.Now().UTC()
	if isPostgresDialect(s.dialect) {
		query := `INSERT INTO ai_messages (conversation_id, role, content, input_tokens, output_tokens, model, created_at)
VALUES (` + sqlPlaceholder(s.dialect, 1) + `, ` + sqlPlaceholder(s.dialect, 2) + `, ` + sqlPlaceholder(s.dialect, 3) + `, ` + sqlPlaceholder(s.dialect, 4) + `, ` + sqlPlaceholder(s.dialect, 5) + `, ` + sqlPlaceholder(s.dialect, 6) + `, ` + sqlPlaceholder(s.dialect, 7) + `)
RETURNING id`
		if err := s.db.QueryRowContext(ctx, query,
			message.ConversationID,
			message.Role,
			message.Content,
			nullIfZero(message.InputTokens),
			nullIfZero(message.OutputTokens),
			nullIfEmpty(message.Model),
			now,
		).Scan(&message.ID); err != nil {
			return ConversationMessage{}, err
		}
	} else {
		query := `INSERT INTO ai_messages (conversation_id, role, content, input_tokens, output_tokens, model, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`
		result, err := s.db.ExecContext(ctx, query,
			message.ConversationID,
			message.Role,
			message.Content,
			nullIfZero(message.InputTokens),
			nullIfZero(message.OutputTokens),
			nullIfEmpty(message.Model),
			now,
		)
		if err != nil {
			return ConversationMessage{}, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return ConversationMessage{}, err
		}
		message.ID = id
	}

	updateConversationQuery := `UPDATE ai_conversations SET updated_at = ` + sqlPlaceholder(s.dialect, 1) + ` WHERE id = ` + sqlPlaceholder(s.dialect, 2)
	if _, err := s.db.ExecContext(ctx, updateConversationQuery, now, message.ConversationID); err != nil {
		return ConversationMessage{}, err
	}

	message.CreatedAt = now
	return message, nil
}

func (s *ConversationSQLStore) UpdateConversationTitle(ctx context.Context, conversationID int64, title string) error {
	query := `UPDATE ai_conversations SET title = ` + sqlPlaceholder(s.dialect, 1) + `, updated_at = ` + sqlPlaceholder(s.dialect, 2) + ` WHERE id = ` + sqlPlaceholder(s.dialect, 3)
	_, err := s.db.ExecContext(ctx, query, strings.TrimSpace(title), time.Now().UTC(), conversationID)
	return err
}

func sqlPlaceholder(dialect string, index int) string {
	if isPostgresDialect(dialect) {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func isPostgresDialect(dialect string) bool {
	normalized := strings.ToLower(strings.TrimSpace(dialect))
	return normalized == "postgres" || normalized == "postgresql" || normalized == "pgx"
}

func nullIfZero(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
