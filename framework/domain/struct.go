package domain

import (
	"time"
)

type Question struct {
	ID            int                  `json:"id"`
	Content       string               `json:"content"`
	Description   string               `json:"description"`
	IsQuiz        bool                 `json:"is_quiz"`
	QuizQuestions map[int]QuizQuestion `json:"questions"`

	Answer      Answer `json:"answers"`
	OtherAnswer Answer `json:"self_answer"` // Currently, this is used only for quiz results, to show self and other side by side.
	VotingCount int    `json:"voting_count"`
	// TODO: not high priority, but Liked/Disliked can be combined in one var. This was
	// done as a way to get a feature in quickly.
	Liked        bool `json:"liked"`
	Disliked     bool `json:"disliked"`
	VotedAt      *time.Time
	HasSelfDraft bool
}

type QuestionCategory struct {
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

type QuizQuestion struct {
	Order     int    `json:"order"`
	Content   string `json:"content"`
	Type      string `json:"type"`
	AnswerMin string `json:"answer_min"`
	AnswerMax string `json:"answer_max"`
}

type Quiz struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Tags        []string       `json:"tags"`
	ExternalID  string         `json:"external_id"`
	Questions   []QuizQuestion `json:"questions"`
}

// TODO: deprecate in favor of Profile
type Author struct {
	UserId       int    `json:"user_id"`
	ProfileID    int    `json:"dating_profile_id"`
	ProfileImage *Photo `json:"profileImage"`
	Name         string `json:"name"`
}

type Answer struct {
	ID                      int                  `json:"id"`
	CreatedAt               time.Time            `json:"created_at"`
	PublishedAt             time.Time            `json:"published_at"`
	IsPublished             bool                 `json:"isPublished"`
	Content                 string               `json:"content"`
	IsQuiz                  bool                 `json:"is_quizz"`
	QuizResults             []QuizQuestionAnswer `json:"quizz_results"`
	Author                  Author               `json:"author"`
	SeenAt                  *time.Time           `json:"seen_at"`
	AggregatedEmojiReaction []AggregatedEmojiReaction
	Rating                  AnswerRating
}

type AnswerRating struct {
	Effort       int `json:"effort"`
	Clarity      int `json:"clarity"`
	Truthfulness int `json:"truthfulness"`
}

type QuizQuestionAnswer struct {
	Order       int    `json:"order"`
	AnswerValue int    `json:"answer_value"`
	AnswerText  string `json:"answer_text"`
}

type Photo struct {
	ID int `json:"id"`
	// TODO: below field name is misleading. It's really the URL. Photo
	// is a non-descript name. Change it for a better naming alternative.
	Photo           string `json:"photo"` // URL -> DEPRECATE...DO NOT USE ANYMORE!
	ThumbnailURL    string `json:"thumbnail_url"`
	ThumbnailWidth  int    `json:"thumbnail_w"`
	ThumbnailHeight int    `json:"thumbnail_h"`
	PreviewURL      string `json:"preview_url"`
	PreviewWidth    int    `json:"preview_w"`
	PreviewHeight   int    `json:"preview_h"`
	FullURL         string `json:"full_url"`
	FullWidth       int    `json:"full_w"`
	FullHeight      int    `json:"full_h"`
	Alt             string `json:"alt"`
}

type PrivateMessage struct {
	ID          int        `json:"id"`
	PublishedAt time.Time  `json:"published_at"`
	Content     string     `json:"content"`
	Sender      Profile    `json:"author"`
	Recipient   Profile    `json:"recipient"`
	SeenAt      *time.Time `json:"seen_at"`
}

type Profile struct {
	ID                       int     `json:"id"`
	Name                     string  `json:"name"`
	Age                      int     `json:"age"`
	Bio                      string  `json:"bio"`
	PhoneNumberE164          string  `json:"phone_number_e164"`
	PhoneNumberInternational *string `json:"phone_number_international"`
	CountryCode              string  `json:"country_code"`
	ProfileImage             *Photo  `json:"profileImage"`
	Photos                   []Photo `json:"photos"`
}

type EmailSubscription struct {
	ID               int     `json:"id"`
	Email            string  `json:"email"`
	Verified         bool    `json:"verified"`
	ConfirmationCode string  `json:"confirmation_code"`
	Lat              float64 `json:"latitude"`
	Lon              float64 `json:"longitude"`
}

type Notification struct {
	ID        int                    `json:"id"`
	Type      NotificationType       `json:"type"`
	Title     string                 `json:"title"`
	Text      string                 `json:"text"`
	Link      string                 `json:"link,omitempty"` // omitempty will not serialize the field if it's empty
	Data      map[string]interface{} `json:"data,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	Read      bool                   `json:"read"`
	ReadAt    time.Time              `json:"read_at"`
	ProfileID int                    `json:"profileId"`
	// ProfileIDWhoCausedNotif is who caused the notification, if available.
	ProfileIDWhoCausedNotif int `json:"profile_id_who_caused_notif"`
	// ResourceIDTiedToNotif is what the notification is about, if it is a reaction to the creation
	// of a specific resource, say an answerID.
	ResourceIDTiedToNotif int `json:"resource_id_tied_to_notif"`
	// When shown in the notification center, this has the text on the button to navigate to the linked ressource
	ButtonText string
	// ReadInNotificationsCenter determines whether the notification will be marked as read when seen in the notification
	// center, or whether the user needs to click on it to mark it as read.
	ReadInNotificationsCenter bool
}

type Invitation struct {
	ID               int       `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	InviteeName      string    `json:"invitee_name"`
	ConfirmationCode string    `json:"confirmation_code"`
}

// AggregatedEmojiReaction aggregates all the emoji reaction for a specific emoji root,
// for example :+1: can link all variations on skin tones.
type AggregatedEmojiReaction struct {
	Count           int
	RootEmoji       Emoji
	EmojiVariations []AnswerReaction
}

type AnswerReaction struct {
	Emoji   Emoji
	Profile Profile
}

type Emoji struct {
	ID          int
	UnifiedCode string
	ShortCode   string
}

type NotificationPermission struct {
	// Title
	Title         string                           `json:"title"`
	Subtitle      string                           `json:"subtitle"`
	Permission    string                           `json:"permission"`
	PlatformsList []NotificationPermissionPlatform `json:"platforms_list"`
}

type NotificationPermissionPlatform struct {
	Platform string `json:"platform"`
	Granted  bool   `json:"granted"`
}
