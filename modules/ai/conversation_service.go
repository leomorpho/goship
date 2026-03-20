package ai

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

type ConversationService struct {
	store    ConversationStore
	provider Provider
}

func NewConversationService(store ConversationStore, provider Provider) *ConversationService {
	return &ConversationService{
		store:    store,
		provider: provider,
	}
}

func (s *ConversationService) CreateConversation(ctx context.Context, userID int64, model string, title string) (Conversation, error) {
	if s == nil || s.store == nil {
		return Conversation{}, fmt.Errorf("conversation store unavailable")
	}
	if strings.TrimSpace(model) == "" {
		model = ClaudeHaiku4
	}
	return s.store.CreateConversation(ctx, userID, model, title)
}

func (s *ConversationService) ListConversations(ctx context.Context, userID int64) ([]Conversation, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("conversation store unavailable")
	}
	return s.store.ListConversations(ctx, userID)
}

func (s *ConversationService) GetHistory(ctx context.Context, conversationID int64) ([]ConversationMessage, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("conversation store unavailable")
	}
	return s.store.ListMessages(ctx, conversationID)
}

func (s *ConversationService) SendMessage(ctx context.Context, conversationID int64, userMessage string) (<-chan Token, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("conversation store unavailable")
	}
	if s.provider == nil {
		return nil, fmt.Errorf("ai provider unavailable")
	}

	conversation, err := s.store.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	userMessage = strings.TrimSpace(userMessage)
	if userMessage == "" {
		return nil, fmt.Errorf("user message is required")
	}

	if _, err := s.store.AppendMessage(ctx, ConversationMessage{
		ConversationID: conversationID,
		Role:           "user",
		Content:        userMessage,
	}); err != nil {
		return nil, err
	}

	history, err := s.store.ListMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	requestMessages := make([]Message, 0, len(history))
	for _, message := range history {
		requestMessages = append(requestMessages, Message{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	out := make(chan Token)
	go func() {
		defer close(out)

		response, completeErr := s.provider.Complete(ctx, Request{
			Model:    conversation.Model,
			Messages: requestMessages,
		})
		if completeErr != nil {
			out <- Token{Error: completeErr, Done: true}
			return
		}

		if strings.TrimSpace(conversation.Title) == "" {
			_ = s.store.UpdateConversationTitle(ctx, conversationID, generateConversationTitle(userMessage))
		}

		for _, chunk := range chunkContent(response.Content, 24) {
			out <- Token{Content: chunk}
		}

		if _, storeErr := s.store.AppendMessage(ctx, ConversationMessage{
			ConversationID: conversationID,
			Role:           "assistant",
			Content:        response.Content,
			InputTokens:    response.InputTokens,
			OutputTokens:   response.OutputTokens,
			Model:          response.Model,
		}); storeErr != nil {
			out <- Token{Error: storeErr, Done: true}
			return
		}

		out <- Token{Done: true}
	}()

	return out, nil
}

func generateConversationTitle(input string) string {
	title := strings.TrimSpace(input)
	if utf8.RuneCountInString(title) <= 48 {
		return title
	}

	runes := []rune(title)
	return strings.TrimSpace(string(runes[:48])) + "..."
}

func chunkContent(content string, chunkSize int) []string {
	if chunkSize <= 0 || utf8.RuneCountInString(content) <= chunkSize {
		return []string{content}
	}

	runes := []rune(content)
	chunks := make([]string, 0, (len(runes)/chunkSize)+1)
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}
