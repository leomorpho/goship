package domain

var NotificationCenterButtonText = map[NotificationType]string{
	NotificationTypeConnectionEngagedWithQuestion: "Answer",
}

// DeleteOnceReadNotificationTypesMap is a map of notification types th;oiSJDfiujladijrgoizdikrjgat can be deleted once seen.
// Note that the boolean doesn't matter, this is just a lazy way of creating a set in Go.
var DeleteOnceReadNotificationTypesMap = map[NotificationType]bool{
	NotificationTypeDailyConversationReminder: true,
}
