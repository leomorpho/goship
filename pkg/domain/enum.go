package domain

import (
	"github.com/orsinium-labs/enum"
)

type NotificationType enum.Member[string]

var (
	// Below are left as examples.
	NotificationTypeNewPrivateMessage             = NotificationType{"new_private_message"}
	NotificationTypeConnectionEngagedWithQuestion = NotificationType{"connection_engaged_with_question"}
	NotificationTypeIncrementNumUnseenMessages    = NotificationType{"increment_num_unseen_msg"}
	NotificationTypeDecrementNumUnseenMessages    = NotificationType{"decrement_num_unseen_msg"}
	NotificationTypeUpdateNumNotifications        = NotificationType{"update_num_notifs"}
	NotificationTypePlatformUpdate                = NotificationType{"platform_update"}
	NotificationTypePaymentFailed                 = NotificationType{"payment_failed"}
	NotificationTypeDailyConversationReminder     = NotificationType{"daily_conversation_reminder"}

	NotificationTypes = enum.New(
		NotificationTypeNewPrivateMessage,
		NotificationTypeConnectionEngagedWithQuestion,
		NotificationTypeIncrementNumUnseenMessages,
		NotificationTypeDecrementNumUnseenMessages,
		NotificationTypeUpdateNumNotifications,
		NotificationTypePlatformUpdate,
		NotificationTypePaymentFailed,
		NotificationTypeDailyConversationReminder,
	)
)

// TODO: move to notifierrepo
type NotificationPermissionType enum.Member[string]

var (
	NotificationPermissionDailyReminder     = NotificationPermissionType{"daily_reminder"}
	NotificationPermissionNewFriendActivity = NotificationPermissionType{"partner_activity"}

	NotificationPermissions = enum.New(
		NotificationPermissionDailyReminder,
		NotificationPermissionNewFriendActivity,
	)
)

// TODO: move to notifierrepo
type NotificationPlatform enum.Member[string]

var (
	NotificationPlatformPush    = NotificationPlatform{"push"} // TODO: this should eventually be rename because it's confusing...it's only for PWA
	NotificationPlatformFCMPush = NotificationPlatform{"fcm_push"}
	NotificationPlatformEmail   = NotificationPlatform{"email"}
	NotificationPlatformSMS     = NotificationPlatform{"sms"}

	NotificationPlatforms = enum.New(
		NotificationPlatformPush,
		NotificationPlatformFCMPush,
		NotificationPlatformEmail,
		NotificationPlatformSMS,
	)
)

type ImageSize enum.Member[string]

var (
	ImageSizeThumbnail = ImageSize{"thumbnail"}
	ImageSizePreview   = ImageSize{"preview"}
	ImageSizeFull      = ImageSize{"full"}

	ImageSizes = enum.New(
		ImageSizeThumbnail,
		ImageSizePreview,
		ImageSizeFull,
	)
)

type ImageCategory enum.Member[string]

var (
	ImageCategoryProfilePhoto   = ImageCategory{"profile_photo"}
	ImageCategoryProfileGallery = ImageCategory{"profile_gallery"}

	ImageCategories = enum.New(
		ImageCategoryProfilePhoto,
		ImageCategoryProfileGallery,
	)
)

type ProductType enum.Member[string]

var (
	ProductTypeFree = ProductType{"free"}
	ProductTypePro  = ProductType{"pro"}

	ProductTypes = enum.New(
		ProductTypeFree,
		ProductTypePro,
	)
)

type BottomNavbarItem enum.Member[string]

var (
	BottomNavbarItemHome          = BottomNavbarItem{"home"}
	BottomNavbarItemNotifications = BottomNavbarItem{"notifications"}
	BottomNavbarItemSettings      = BottomNavbarItem{"settings"}
	BottomNavbarItemProfile       = BottomNavbarItem{"profile"}

	BottomNavbarItems = enum.New(
		BottomNavbarItemHome,
		BottomNavbarItemNotifications,
		BottomNavbarItemSettings,
		BottomNavbarItemProfile,
	)
)

type EmailSubscriptionList enum.Member[string]

var (
	EmailNewsletter         = EmailSubscriptionList{"email_newsletter"}
	EmailInitialAnnoucement = EmailSubscriptionList{"launch_announcement"}

	EmailSubscriptionLists = enum.New(
		EmailNewsletter,
		EmailInitialAnnoucement,
	)
)
