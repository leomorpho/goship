package auditlog

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	authUserIDKey       = "auth_user_id"
	contextIPKey        = "audit_log_ip_address"
	contextUserAgentKey = "audit_log_user_agent"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Record(ctx context.Context, action, resourceType, resourceID string, changes any) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("audit log service is not configured")
	}

	payload, err := marshalChanges(changes)
	if err != nil {
		return err
	}

	return s.store.Insert(ctx, Log{
		UserID:       userIDFromContext(ctx),
		Action:       strings.TrimSpace(action),
		ResourceType: strings.TrimSpace(resourceType),
		ResourceID:   strings.TrimSpace(resourceID),
		Changes:      payload,
		IPAddress:    stringValue(ctx, contextIPKey),
		UserAgent:    stringValue(ctx, contextUserAgentKey),
	})
}

func (s *Service) List(ctx context.Context, filters ListFilters) ([]Log, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("audit log service is not configured")
	}
	return s.store.List(ctx, filters)
}

func WithRequestMetadata(ctx context.Context, userID *int64, ipAddress, userAgent string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if userID != nil {
		ctx = context.WithValue(ctx, authUserIDKey, *userID)
	}
	if strings.TrimSpace(ipAddress) != "" {
		ctx = context.WithValue(ctx, contextIPKey, strings.TrimSpace(ipAddress))
	}
	if strings.TrimSpace(userAgent) != "" {
		ctx = context.WithValue(ctx, contextUserAgentKey, strings.TrimSpace(userAgent))
	}
	return ctx
}

func marshalChanges(changes any) (string, error) {
	if changes == nil {
		return "", nil
	}
	data, err := json.Marshal(changes)
	if err != nil {
		return "", fmt.Errorf("marshal audit log changes: %w", err)
	}
	return string(data), nil
}

func userIDFromContext(ctx context.Context) *int64 {
	if ctx == nil {
		return nil
	}
	switch v := ctx.Value(authUserIDKey).(type) {
	case int:
		id := int64(v)
		return &id
	case int64:
		id := v
		return &id
	case int32:
		id := int64(v)
		return &id
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return nil
		}
		return &parsed
	default:
		return nil
	}
}

func stringValue(ctx context.Context, key string) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(key).(string)
	return strings.TrimSpace(value)
}
