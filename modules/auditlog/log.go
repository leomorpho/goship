package auditlog

import "time"

type Log struct {
	ID           int64
	UserID       *int64
	Action       string
	ResourceType string
	ResourceID   string
	Changes      string
	IPAddress    string
	UserAgent    string
	CreatedAt    time.Time
}

type ListFilters struct {
	UserID       *int64
	Action       string
	ResourceType string
	ResourceID   string
	Limit        int
}
