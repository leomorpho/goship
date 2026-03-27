package profiles

import (
	"testing"
	"time"
)

func TestCalculateAgeAt_BirthdayBoundaries(t *testing.T) {
	birthdate := time.Date(2000, time.March, 27, 12, 0, 0, 0, time.UTC)

	beforeBirthday := time.Date(2026, time.March, 26, 23, 59, 0, 0, time.UTC)
	onBirthday := time.Date(2026, time.March, 27, 0, 1, 0, 0, time.UTC)
	afterBirthday := time.Date(2026, time.March, 28, 0, 0, 0, 0, time.UTC)

	if got := calculateAgeAt(birthdate, beforeBirthday); got != 25 {
		t.Fatalf("age before birthday = %d, want 25", got)
	}
	if got := calculateAgeAt(birthdate, onBirthday); got != 26 {
		t.Fatalf("age on birthday = %d, want 26", got)
	}
	if got := calculateAgeAt(birthdate, afterBirthday); got != 26 {
		t.Fatalf("age after birthday = %d, want 26", got)
	}
}
