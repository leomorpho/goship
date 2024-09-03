package domain

import (
	"database/sql/driver"
	"fmt"

	"github.com/orsinium-labs/enum"
)

type ExperienceType enum.Member[string]

var (
	ExperienceTypeCommittedRelationship = ExperienceType{"committed"}
	ExperienceTypeDating                = ExperienceType{"dating"}

	ExperienceTypes = enum.New(
		ExperienceTypeCommittedRelationship,
		ExperienceTypeDating,
	)
)

type RequestStatus enum.Member[string]

var (
	RequestStatusGranted = RequestStatus{"granted"}
	RequestStatusRefused = RequestStatus{"refused"}
	RequestStatusPending = RequestStatus{"pending"}

	RequestStatuses = enum.New(
		RequestStatusGranted,
		RequestStatusRefused,
		RequestStatusPending,
	)
)

// TODO: update the below enum to use orsinium-labs/enum type enum
type Gender int

const (
	GenderMan Gender = iota
	GenderWoman
	GenderNonBinary
	Genderqueer
	GenderAgender
	GenderTransMale
	GenderTransFemale
	GenderOther
)

var SeededGenders = []Gender{GenderMan, GenderWoman, GenderNonBinary}

// String method to convert the enum to a readable string
func (g Gender) String() string {
	return [...]string{
		"Man", "Woman", "Non-Binary",
	}[g]
}

// GenderFromStr converts a string to its enum representation
func GenderFromStr(str string) (Gender, error) {
	switch str {
	case "Man":
		return GenderMan, nil
	case "Woman":
		return GenderWoman, nil
	case "Non-Binary":
		return GenderNonBinary, nil

	default:
		return 0, fmt.Errorf("invalid gender: %s", str)
	}
}

// Value - to satisfy the driver.Valuer interface
func (g Gender) Value() (driver.Value, error) {
	return g.String(), nil
}

// Scan - to satisfy the sql.Scanner interface
func (g *Gender) Scan(value any) error {
	var s string
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("unsupported type for Gender: %T", value)
	}

	var err error
	*g, err = GenderFromStr(s)
	return err
}

type NotificationType enum.Member[string]

var (
	NotificationTypeNewPrivateMessage             = NotificationType{"new_private_message"}
	NotificationTypeMutualQuestionAnswered        = NotificationType{"mutual_question_answered"}
	NotificationTypeConnectionEngagedWithQuestion = NotificationType{"connection_engaged_with_question"}
	NotificationTypeIncrementNumUnseenMessages    = NotificationType{"increment_num_unseen_msg"}
	NotificationTypeDecrementNumUnseenMessages    = NotificationType{"decrement_num_unseen_msg"}
	NotificationTypeUpdateNumNotifications        = NotificationType{"update_num_notifs"}
	NotificationTypeConnectionRequestAccepted     = NotificationType{"connection_request_accepted"}
	NotificationTypePlatformUpdate                = NotificationType{"platform_update"}
	NotificationTypeConnectionReactedToAnswer     = NotificationType{"connection_reacted_to_answer"}
	NotificationTypeCommittedRelationshipRequest  = NotificationType{"committed_relationship_request"}
	NotificationTypeBreakup                       = NotificationType{"breakup"}
	NotificationTypeContactRequest                = NotificationType{"contact_request"}
	NotificationTypeNewHomeFeedQA                 = NotificationType{"new_homefeed_qa_object"}
	NotificationTypePaymentFailed                 = NotificationType{"payment_failed"}
	NotificationTypeDailyConversationReminder     = NotificationType{"daily_conversation_reminder"}

	NotificationTypes = enum.New(
		NotificationTypeNewPrivateMessage,
		NotificationTypeMutualQuestionAnswered,
		NotificationTypeConnectionEngagedWithQuestion,
		NotificationTypeIncrementNumUnseenMessages,
		NotificationTypeDecrementNumUnseenMessages,
		NotificationTypeUpdateNumNotifications,
		NotificationTypeConnectionRequestAccepted,
		NotificationTypePlatformUpdate,
		NotificationTypeConnectionReactedToAnswer,
		NotificationTypeCommittedRelationshipRequest,
		NotificationTypeBreakup,
		NotificationTypeContactRequest,
		NotificationTypeNewHomeFeedQA,
		NotificationTypePaymentFailed,
		NotificationTypeDailyConversationReminder,
	)
)

// TODO: move to notifierrepo
type NotificationPermissionType enum.Member[string]

var (
	NotificationPermissionDailyReminder   = NotificationPermissionType{"daily_reminder"}
	NotificationPermissionPartnerActivity = NotificationPermissionType{"partner_activity"}

	NotificationPermissions = enum.New(
		NotificationPermissionDailyReminder,
		NotificationPermissionPartnerActivity,
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

// TODO: move to notifierrepo
type ConnectionRequestType enum.Member[string]

var (
	ConnectionRequestCommitted = ConnectionRequestType{"committed"}
	ConnectionRequestDating    = ConnectionRequestType{"dating"}

	ConnectionRequestTypes = enum.New(
		ConnectionRequestCommitted,
		ConnectionRequestDating,
	)
)

type BottomNavbarItem enum.Member[string]

var (
	BottomNavbarItemMeet          = BottomNavbarItem{"meet"}
	BottomNavbarItemHome          = BottomNavbarItem{"home"}
	BottomNavbarItemNotifications = BottomNavbarItem{"notifications"}
	BottomNavbarItemSettings      = BottomNavbarItem{"settings"}
	BottomNavbarItemProfile       = BottomNavbarItem{"profile"}

	BottomNavbarItems = enum.New(
		BottomNavbarItemMeet,
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
