package viewmodels

import (
	"time"

	"github.com/leomorpho/goship/framework/domain"
)

type NotificationItem struct {
	ID                        int
	Title                     string
	Text                      string
	ButtonText                string
	Link                      string
	CreatedAt                 time.Time
	Read                      bool
	ReadInNotificationsCenter bool
}

type NormalNotificationsPageData struct {
	Notifications []NotificationItem
	NextPageURL   string
}

func NotificationItemsFromDomain(items []*domain.Notification) []NotificationItem {
	if len(items) == 0 {
		return []NotificationItem{}
	}

	out := make([]NotificationItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, NotificationItem{
			ID:                        item.ID,
			Title:                     item.Title,
			Text:                      item.Text,
			ButtonText:                item.ButtonText,
			Link:                      item.Link,
			CreatedAt:                 item.CreatedAt,
			Read:                      item.Read,
			ReadInNotificationsCenter: item.ReadInNotificationsCenter,
		})
	}
	return out
}
