package types

import "github.com/mikestefanello/pagoda/pkg/domain"

type NormalNotificationsPageData struct {
	Notifications []*domain.Notification
	NextPageURL   string
}
