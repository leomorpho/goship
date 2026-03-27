package notifications

import (
	"strings"

	"github.com/leomorpho/goship/framework/domain"
	"github.com/orsinium-labs/enum"
)

type PermissionType enum.Member[string]

var (
	PermissionDailyReminder     = PermissionType{"daily_reminder"}
	PermissionNewFriendActivity = PermissionType{"partner_activity"}

	Permissions = enum.New(
		PermissionDailyReminder,
		PermissionNewFriendActivity,
	)
)

type Platform enum.Member[string]

const legacyPlatformPushValue = "push"

var (
	PlatformPWAPush = Platform{"pwa_push"}
	PlatformFCMPush = Platform{"fcm_push"}
	PlatformEmail   = Platform{"email"}
	PlatformSMS     = Platform{"sms"}

	Platforms = enum.New(
		PlatformPWAPush,
		PlatformFCMPush,
		PlatformEmail,
		PlatformSMS,
	)
)

func ParsePlatform(raw string) *Platform {
	normalized := strings.TrimSpace(raw)
	if normalized == legacyPlatformPushValue {
		normalized = PlatformPWAPush.Value
	}
	return Platforms.Parse(normalized)
}

var PermissionMap = map[PermissionType]domain.NotificationPermission{
	PermissionDailyReminder: {
		Title:      "Daily conversation",
		Subtitle:   "A reminder to not miss today's question, sent at most once a day.",
		Permission: PermissionDailyReminder.Value,
	},
	PermissionNewFriendActivity: {
		Title:      "Partner activity",
		Subtitle:   "Answers you missed, sent at most once a day.",
		Permission: PermissionNewFriendActivity.Value,
	},
}
