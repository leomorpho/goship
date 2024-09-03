package domain

import "time"

const FriendEngagedWithQuestionNotificationText = "🤔💭 Your partner answered a new question. Answer it too to see their thoughts on it!"
const FriendEngagedWithQuizNotificationText = "🤔💭 Your partner answered a new quiz. Answer it too to see their answers!"

const DefaultGender = GenderWoman
const DefaultMinInterestedAge = 30
const DefaultMaxInterestedAge = 30
const DefaultBio = "Hello, there! 🌟"

var DefaultBirthdate = time.Date(1898, time.January, 6, 0, 0, 0, 0, time.UTC)

const DefaultLatitude = 48.8566
const DefaultLongitude = 2.3522
const DefaultRadius = 50000 // meters -> 50km

// Define the sizes
var ImageSizeEnumToSizeMap = map[ImageSize]int{
	ImageSizeThumbnail: 150,  // max 150 pixels wide or tall
	ImageSizePreview:   800,  // max 800 pixels wide or tall
	ImageSizeFull:      1600, // max 1600 pixels wide or tall
}

const FreePlanNumAnswersPerDay = 1

const NOTIFICATION_TYPE = "permission"

const COMMITTED_RELATIONSHIP_INVITATION = "committed_relationship_invitation_placeholder"
