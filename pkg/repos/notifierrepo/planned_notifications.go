package notifierrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/lastseenonline"
	"github.com/mikestefanello/pagoda/ent/notification"
	"github.com/mikestefanello/pagoda/ent/notificationpermission"
	"github.com/mikestefanello/pagoda/ent/notificationtime"
	"github.com/mikestefanello/pagoda/ent/profile"
	"github.com/mikestefanello/pagoda/ent/user"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/subscriptions"
	"github.com/rs/zerolog/log"
)

type PlannedNotificationsRepo struct {
	orm              *ent.Client
	subscriptionRepo *subscriptions.SubscriptionsRepo
}

func NewPlannedNotificationsRepo(
	orm *ent.Client, subscriptionRepo *subscriptions.SubscriptionsRepo,
) *PlannedNotificationsRepo {

	return &PlannedNotificationsRepo{
		orm:              orm,
		subscriptionRepo: subscriptionRepo,
	}
}

func (p *PlannedNotificationsRepo) CreateNotificationTimeObjects(
	ctx context.Context,
	notifType domain.NotificationType,
	permission domain.NotificationPermissionType,
) error {
	profiles, err := p.orm.NotificationPermission.Query().
		Where(
			notificationpermission.PermissionEQ(notificationpermission.Permission(permission.Value)),
		).
		QueryProfile().
		Select(profile.FieldID).
		WithUser(func(u *ent.UserQuery) {
			u.WithLastSeenAt()
			u.Select(user.FieldID)
		}).
		WithNotificationTimes(
			func(n *ent.NotificationTimeQuery) {
				n.Where(notificationtime.TypeEQ(notificationtime.Type(notifType.Value)))
			},
		).
		All(ctx)
	if err != nil {
		return err
	}

	p.DeleteStaleLastSeenObjects(ctx)

	// Generate missing NotificationTime objects, and update old ones.
	// Currently, we re-evaluate best notif time every day.
	lastRelevantNotificationTimeGeneration := time.Now().Add(time.Hour * 24 * -1)
	for _, profile := range profiles {
		if profile.Edges.NotificationTimes == nil ||
			len(profile.Edges.NotificationTimes) == 0 ||
			profile.Edges.NotificationTimes[0].UpdatedAt.Before(lastRelevantNotificationTimeGeneration) {

			p.UpsertNotificationTime(ctx, profile.ID, notifType)
		}
	}
	return nil
}

func (p *PlannedNotificationsRepo) DeleteStaleLastSeenObjects(ctx context.Context) {
	const TIME_TO_KEEP_DAYS = 30
	deleteBeforeTime := time.Now().Add(time.Hour * 24 * -TIME_TO_KEEP_DAYS)
	// Make sure all users have a NotificationTime
	_, err := p.orm.LastSeenOnline.Delete().
		Where(lastseenonline.SeenAtLTE(deleteBeforeTime)).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).
			Time("deleteBeforeTime", deleteBeforeTime).
			Msg("failed to delete old LastSeenOnline objects")
	}

}

func (p *PlannedNotificationsRepo) UpsertNotificationTime(
	ctx context.Context, profileID int, notificationType domain.NotificationType,
) (int, error) {

	lastSeenTimes, err := p.orm.LastSeenOnline.Query().
		Where(lastseenonline.HasUserWith(user.HasProfileWith(profile.IDEQ(profileID)))).
		All(ctx)
	if err != nil {
		return 0, err
	}

	if len(lastSeenTimes) == 0 {
		return 0, fmt.Errorf("no connection times found for profileID %d", profileID)
	}

	// Create a histogram to count occurrences of each time slot
	const minutesInDay = 24 * 60
	histogram := make([]int, minutesInDay)

	for _, lso := range lastSeenTimes {
		minutesFromMidnight := p.GetMinutesFromMidnight(lso.SeenAt)
		histogram[minutesFromMidnight]++
	}

	// Calculate the peak time over a 20-minute sliding window
	windowSize := 30     // 30 minutes
	incrementJumps := 15 // 15 minutes
	maxSum := 0
	maxSumStartMinute := 0
	maxSumEndMinute := 0
	peakMinute := 0

	// Slide the window over the histogram at 15 minute increments
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

	// Adjust peakMinute to be the middle of the 20-minute window
	peakMinute = int((maxSumStartMinute + maxSumEndMinute) / 2)

	// Update the notification time with the peak minute
	n, err := p.orm.NotificationTime.
		Update().
		Where(
			notificationtime.TypeEQ(notificationtime.Type(notificationType.Value)),
			notificationtime.HasProfileWith(profile.IDEQ(profileID)),
		).
		SetSendMinute(peakMinute).
		Save(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return 0, err
	}

	if n == 0 {
		// No rows were updated, so create a new record
		_, err = p.orm.NotificationTime.
			Create().
			SetSendMinute(peakMinute).
			SetProfileID(profileID).
			SetType(notificationtime.Type(notificationType.Value)).
			Save(ctx)
		if err != nil {
			return 0, err
		}
	}
	return peakMinute, nil
}

// GetMinutesFromMidnight returns the number of minutes from midnight given a timestamp
func (p *PlannedNotificationsRepo) GetMinutesFromMidnight(t time.Time) int {
	return t.Hour()*60 + t.Minute()
}

// GetTimestampFromMinutes returns a timestamp for today at the given minutes from midnight
func (p *PlannedNotificationsRepo) GetTimestampFromMinutes(minutes int) time.Time {
	now := time.Now()
	hour := minutes / 60
	minute := minutes % 60
	return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
}

// ProfileIDsCanGetNotificatiedNow returns the profile IDs who can get notified now for a notification type
func (p *PlannedNotificationsRepo) ProfileIDsCanGetPlannedNotificationNow(
	ctx context.Context, timestamp time.Time, notifType domain.NotificationType, profileIDs *[]int,
) ([]int, error) {

	utcTimestamp := timestamp.UTC()
	prevMidnightTimestamp := time.Date(utcTimestamp.Year(), utcTimestamp.Month(), utcTimestamp.Day(), 0, 0, 0, 0, utcTimestamp.Location())

	timestampMinutesFromMidnight := p.GetMinutesFromMidnight(utcTimestamp)

	query := p.orm.NotificationTime.
		Query().
		Where(
			notificationtime.TypeEQ(notificationtime.Type(notifType.Value)),
			notificationtime.And(
				notificationtime.SendMinuteGTE(0),
				notificationtime.SendMinuteLTE(timestampMinutesFromMidnight),
			),
			// The following prevents any overlap and double notifications.
			notificationtime.Not(
				notificationtime.HasProfileWith(profile.HasNotificationsWith(
					notification.CreatedAtGTE(prevMidnightTimestamp),
					notification.TypeEQ(notification.Type(notifType.Value)),
				)),
			),
		)

	if profileIDs != nil {
		query.Where(
			notificationtime.HasProfileWith(profile.IDIn(*profileIDs...)),
		)
	}

	profiles, err := query.
		QueryProfile().
		Select(profile.FieldID).
		All(ctx)
	if err != nil {
		return []int{}, err
	}

	var profileIDsCanGetNotif []int
	for _, n := range profiles {
		profileIDsCanGetNotif = append(profileIDsCanGetNotif, n.ID)
	}

	return profileIDsCanGetNotif, nil
}
