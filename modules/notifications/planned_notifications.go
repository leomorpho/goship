package notifications

import (
	"context"
	"fmt"
	"time"

	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/rs/zerolog/log"
)

type plannedNotificationCandidate struct {
	ProfileID                 int
	NotificationTimeUpdatedAt *time.Time
}

type plannedNotificationStorage interface {
	listProfilesForPermission(ctx context.Context, permission domain.NotificationPermissionType, notifType domain.NotificationType) ([]plannedNotificationCandidate, error)
	deleteStaleLastSeenBefore(ctx context.Context, deleteBeforeTime time.Time) error
	listLastSeenForProfile(ctx context.Context, profileID int) ([]time.Time, error)
	upsertNotificationTime(ctx context.Context, profileID int, notificationType domain.NotificationType, sendMinute int) error
	listProfileIDsCanGetPlannedNotificationNow(
		ctx context.Context, notifType domain.NotificationType, prevMidnightTimestamp time.Time, timestampMinutesFromMidnight int, profileIDs *[]int,
	) ([]int, error)
}

type PlannedNotificationsService struct {
	store            plannedNotificationStorage
	subscriptionRepo *paidsubscriptions.Service
}

func NewPlannedNotificationsServiceWithStore(
	store plannedNotificationStorage, subscriptionRepo *paidsubscriptions.Service,
) *PlannedNotificationsService {
	return &PlannedNotificationsService{
		store:            store,
		subscriptionRepo: subscriptionRepo,
	}
}

func (p *PlannedNotificationsService) CreateNotificationTimeObjects(
	ctx context.Context,
	notifType domain.NotificationType,
	permission domain.NotificationPermissionType,
) error {
	profiles, err := p.store.listProfilesForPermission(ctx, permission, notifType)
	if err != nil {
		return err
	}

	p.DeleteStaleLastSeenObjects(ctx)

	// Generate missing NotificationTime objects, and update old ones.
	// Currently, we re-evaluate best notif time every day.
	lastRelevantNotificationTimeGeneration := time.Now().Add(time.Hour * 24 * -1)
	for _, profile := range profiles {
		if profile.NotificationTimeUpdatedAt == nil || profile.NotificationTimeUpdatedAt.Before(lastRelevantNotificationTimeGeneration) {
			if _, err := p.UpsertNotificationTime(ctx, profile.ProfileID, notifType); err != nil {
				log.Debug().
					Err(err).
					Int("profileID", profile.ProfileID).
					Str("notificationType", notifType.Value).
					Msg("skipping notification time upsert for profile")
			}
		}
	}
	return nil
}

func (p *PlannedNotificationsService) DeleteStaleLastSeenObjects(ctx context.Context) {
	const timeToKeepDays = 30
	deleteBeforeTime := time.Now().Add(time.Hour * 24 * -timeToKeepDays)
	if err := p.store.deleteStaleLastSeenBefore(ctx, deleteBeforeTime); err != nil {
		log.Error().Err(err).
			Time("deleteBeforeTime", deleteBeforeTime).
			Msg("failed to delete old LastSeenOnline objects")
	}
}

func (p *PlannedNotificationsService) UpsertNotificationTime(
	ctx context.Context, profileID int, notificationType domain.NotificationType,
) (int, error) {
	lastSeenTimes, err := p.store.listLastSeenForProfile(ctx, profileID)
	if err != nil {
		return 0, err
	}
	if len(lastSeenTimes) == 0 {
		return 0, fmt.Errorf("no connection times found for profileID %d", profileID)
	}

	// Create a histogram to count occurrences of each time slot.
	const minutesInDay = 24 * 60
	histogram := make([]int, minutesInDay)
	for _, lso := range lastSeenTimes {
		minutesFromMidnight := p.GetMinutesFromMidnight(lso)
		histogram[minutesFromMidnight]++
	}

	// Calculate the peak time over a 30-minute sliding window with 15-minute jumps.
	windowSize := 30
	incrementJumps := 15
	maxSum := 0
	maxSumStartMinute := 0
	maxSumEndMinute := 0
	for i := 0; i < minutesInDay; i += incrementJumps {
		currentSum := 0
		if i+windowSize > minutesInDay {
			continue
		}
		for j := i; j < i+windowSize; j++ {
			currentSum += histogram[j]
		}
		if maxSum < currentSum {
			maxSum = currentSum
			maxSumStartMinute = i
			maxSumEndMinute = i + windowSize
		}
	}
	peakMinute := int((maxSumStartMinute + maxSumEndMinute) / 2)

	if err := p.store.upsertNotificationTime(ctx, profileID, notificationType, peakMinute); err != nil {
		return 0, err
	}
	return peakMinute, nil
}

// GetMinutesFromMidnight returns the number of minutes from midnight given a timestamp.
func (p *PlannedNotificationsService) GetMinutesFromMidnight(t time.Time) int {
	return t.Hour()*60 + t.Minute()
}

// GetTimestampFromMinutes returns a timestamp for today at the given minutes from midnight.
func (p *PlannedNotificationsService) GetTimestampFromMinutes(minutes int) time.Time {
	now := time.Now()
	hour := minutes / 60
	minute := minutes % 60
	return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
}

// ProfileIDsCanGetPlannedNotificationNow returns the profile IDs who can get notified now for a notification type.
func (p *PlannedNotificationsService) ProfileIDsCanGetPlannedNotificationNow(
	ctx context.Context, timestamp time.Time, notifType domain.NotificationType, profileIDs *[]int,
) ([]int, error) {
	utcTimestamp := timestamp.UTC()
	prevMidnightTimestamp := time.Date(utcTimestamp.Year(), utcTimestamp.Month(), utcTimestamp.Day(), 0, 0, 0, 0, utcTimestamp.Location())
	timestampMinutesFromMidnight := p.GetMinutesFromMidnight(utcTimestamp)

	return p.store.listProfileIDsCanGetPlannedNotificationNow(
		ctx,
		notifType,
		prevMidnightTimestamp,
		timestampMinutesFromMidnight,
		profileIDs,
	)
}
