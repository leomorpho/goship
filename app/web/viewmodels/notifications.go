package viewmodels

import "github.com/leomorpho/goship/framework/domain"

type NormalNotificationsPageData struct {
	Notifications []*domain.Notification
	NextPageURL   string
}
