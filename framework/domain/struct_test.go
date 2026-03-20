package domain

import (
	"testing"
	"time"
)

func TestProfileAccessorsAreNilSafe(t *testing.T) {
	var nilProfile *Profile
	if nilProfile.HasProfileImage() {
		t.Fatal("nil profile should not report a profile image")
	}
	if got := nilProfile.ProfileImageThumbnailURL(); got != "" {
		t.Fatalf("thumbnail url = %q, want empty", got)
	}
	if got := nilProfile.ContactPhoneNumber(); got != "" {
		t.Fatalf("contact phone = %q, want empty", got)
	}

	international := "+1 555 111 2222"
	profile := &Profile{
		PhoneNumberE164:          "+15551112222",
		PhoneNumberInternational: &international,
		ProfileImage: &Photo{
			ThumbnailURL: "https://example.com/thumb.jpg",
		},
	}

	if !profile.HasProfileImage() {
		t.Fatal("profile should report a profile image")
	}
	if got := profile.ProfileImageThumbnailURL(); got != "https://example.com/thumb.jpg" {
		t.Fatalf("thumbnail url = %q, want populated value", got)
	}
	if got := profile.ContactPhoneNumber(); got != international {
		t.Fatalf("contact phone = %q, want international format", got)
	}

	profile.PhoneNumberInternational = nil
	if got := profile.ContactPhoneNumber(); got != profile.PhoneNumberE164 {
		t.Fatalf("contact phone fallback = %q, want %q", got, profile.PhoneNumberE164)
	}
}

func TestQuestionAndSeenAtAccessorsAreNilSafe(t *testing.T) {
	var nilQuestion *Question
	if nilQuestion.HasVotedAt() {
		t.Fatal("nil question should not report a vote timestamp")
	}
	if got := nilQuestion.VotedAtOrZero(); !got.IsZero() {
		t.Fatalf("vote timestamp = %v, want zero", got)
	}

	var nilAnswer *Answer
	if nilAnswer.HasSeenAt() {
		t.Fatal("nil answer should not report seen-at")
	}
	if got := nilAnswer.SeenAtOrZero(); !got.IsZero() {
		t.Fatalf("answer seen-at = %v, want zero", got)
	}

	var nilMessage *PrivateMessage
	if nilMessage.HasSeenAt() {
		t.Fatal("nil private message should not report seen-at")
	}
	if got := nilMessage.SeenAtOrZero(); !got.IsZero() {
		t.Fatalf("message seen-at = %v, want zero", got)
	}

	now := time.Now().UTC()
	question := &Question{VotedAt: &now}
	answer := &Answer{SeenAt: &now}
	message := &PrivateMessage{SeenAt: &now}

	if !question.HasVotedAt() || !question.VotedAtOrZero().Equal(now) {
		t.Fatal("question accessors should return the populated timestamp")
	}
	if !answer.HasSeenAt() || !answer.SeenAtOrZero().Equal(now) {
		t.Fatal("answer accessors should return the populated timestamp")
	}
	if !message.HasSeenAt() || !message.SeenAtOrZero().Equal(now) {
		t.Fatal("message accessors should return the populated timestamp")
	}
}

func TestAuthorAccessorsAreNilSafe(t *testing.T) {
	var nilAuthor *Author
	if nilAuthor.HasProfileImage() {
		t.Fatal("nil author should not report a profile image")
	}
	if got := nilAuthor.ProfileImageThumbnailURL(); got != "" {
		t.Fatalf("author thumbnail url = %q, want empty", got)
	}

	author := &Author{ProfileImage: &Photo{ThumbnailURL: "https://example.com/author-thumb.jpg"}}
	if !author.HasProfileImage() {
		t.Fatal("author should report a profile image")
	}
	if got := author.ProfileImageThumbnailURL(); got != "https://example.com/author-thumb.jpg" {
		t.Fatalf("author thumbnail url = %q, want populated value", got)
	}
}
