package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"
)

type Service struct {
	provider Provider
	logger   *slog.Logger
}

func NewService(provider Provider, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		provider: provider,
		logger:   logger,
	}
}

func (s *Service) Complete(ctx context.Context, req Request) (*Response, error) {
	if s == nil || s.provider == nil {
		return nil, fmt.Errorf("ai provider unavailable")
	}

	req = applyRequestDefaults(req)
	start := time.Now()

	resp, err := s.provider.Complete(ctx, req)
	if err != nil {
		s.logger.Error("ai completion failed", "error", err, "model", req.Model)
		return nil, err
	}

	if err := decodeStructuredResponse(req.Schema, resp.Content); err != nil {
		s.logger.Error("ai structured decode failed", "error", err, "model", resp.Model)
		return nil, err
	}

	s.logger.Info("ai completion finished", "model", resp.Model, "duration_ms", time.Since(start).Milliseconds())
	return resp, nil
}

func (s *Service) Stream(ctx context.Context, req Request) (<-chan Token, error) {
	if s == nil || s.provider == nil {
		return nil, fmt.Errorf("ai provider unavailable")
	}

	req = applyRequestDefaults(req)
	stream, err := s.provider.Stream(ctx, req)
	if err != nil {
		s.logger.Error("ai stream failed", "error", err, "model", req.Model)
		return nil, err
	}

	if req.Schema == nil {
		return stream, nil
	}

	out := make(chan Token)
	go func() {
		defer close(out)

		var b strings.Builder
		for token := range stream {
			if token.Content != "" {
				b.WriteString(token.Content)
			}
			out <- token
			if token.Error != nil {
				return
			}
			if token.Done {
				if err := decodeStructuredResponse(req.Schema, b.String()); err != nil {
					out <- Token{Error: err, Done: true}
				}
				return
			}
		}
	}()

	return out, nil
}

func applyRequestDefaults(req Request) Request {
	if strings.TrimSpace(req.Model) == "" {
		req.Model = ClaudeHaiku4
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 1024
	}
	return req
}

func decodeStructuredResponse(target any, content string) error {
	if target == nil {
		return nil
	}

	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return fmt.Errorf("structured output target must be a non-nil pointer")
	}

	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode structured output: %w", err)
	}

	return nil
}
