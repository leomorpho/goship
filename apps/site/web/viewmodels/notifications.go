package viewmodels

import "github.com/leomorpho/goship/pkg/domain"

type NormalNotificationsPageData struct {
	Notifications []*domain.Notification
	NextPageURL   string
}
