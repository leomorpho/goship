package notifications

import (
	"context"
	"time"

	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/db/ent/lastseenonline"
	"github.com/leomorpho/goship/db/ent/notification"
	"github.com/leomorpho/goship/db/ent/notificationpermission"
	"github.com/leomorpho/goship/db/ent/notificationtime"
	"github.com/leomorpho/goship/db/ent/profile"
	"github.com/leomorpho/goship/db/ent/user"
	"github.com/leomorpho/goship/framework/domain"
)

type entPlannedNotificationStore struct {
	orm *ent.Client
}

func NewPlannedNotificationsService(
	orm *ent.Client, subscriptionRepo *paidsubscriptions.Service,
) *PlannedNotificationsService {
	return NewPlannedNotificationsServiceWithStore(
		&entPlannedNotificationStore{orm: orm},
		subscriptionRepo,
	)
}

func (s *entPlannedNotificationStore) listProfilesForPermission(
	ctx context.Context, permission domain.NotificationPermissionType, notifType domain.NotificationType,
) ([]plannedNotificationCandidate, error) {
	profiles, err := s.orm.NotificationPermission.Query().
		Where(notificationpermission.PermissionEQ(notificationpermission.Permission(permission.Value))).
		QueryProfile().
		Select(profile.FieldID).
		WithUser(func(u *ent.UserQuery) {
			u.WithLastSeenAt()
			u.Select(user.FieldID)
		}).
		WithNotificationTimes(func(n *ent.NotificationTimeQuery) {
			n.Where(notificationtime.TypeEQ(notificationtime.Type(notifType.Value)))
		}).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]plannedNotificationCandidate, 0, len(profiles))
	for _, p := range profiles {
		candidate := plannedNotificationCandidate{ProfileID: p.ID}
		if p.Edges.NotificationTimes != nil && len(p.Edges.NotificationTimes) > 0 {
			updatedAt := p.Edges.NotificationTimes[0].UpdatedAt
			candidate.NotificationTimeUpdatedAt = &updatedAt
		}
		out = append(out, candidate)
	}
	return out, nil
}

func (s *entPlannedNotificationStore) deleteStaleLastSeenBefore(
	ctx context.Context, deleteBeforeTime time.Time,
) error {
	_, err := s.orm.LastSeenOnline.Delete().
		Where(lastseenonline.SeenAtLTE(deleteBeforeTime)).
		Exec(ctx)
	return err
}

func (s *entPlannedNotificationStore) listLastSeenForProfile(
	ctx context.Context, profileID int,
) ([]time.Time, error) {
	lastSeenTimes, err := s.orm.LastSeenOnline.Query().
		Where(lastseenonline.HasUserWith(user.HasProfileWith(profile.IDEQ(profileID)))).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]time.Time, 0, len(lastSeenTimes))
	for _, lso := range lastSeenTimes {
		out = append(out, lso.SeenAt)
	}
	return out, nil
}

func (s *entPlannedNotificationStore) upsertNotificationTime(
	ctx context.Context, profileID int, notificationType domain.NotificationType, sendMinute int,
) error {
	n, err := s.orm.NotificationTime.
		Update().
		Where(
			notificationtime.TypeEQ(notificationtime.Type(notificationType.Value)),
			notificationtime.HasProfileWith(profile.IDEQ(profileID)),
		).
		SetSendMinute(sendMinute).
		Save(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}
	if n == 0 {
		_, err = s.orm.NotificationTime.
			Create().
			SetSendMinute(sendMinute).
			SetProfileID(profileID).
			SetType(notificationtime.Type(notificationType.Value)).
			Save(ctx)
		return err
	}
	return nil
}

func (s *entPlannedNotificationStore) listProfileIDsCanGetPlannedNotificationNow(
	ctx context.Context, notifType domain.NotificationType, prevMidnightTimestamp time.Time, timestampMinutesFromMidnight int, profileIDs *[]int,
) ([]int, error) {
	query := s.orm.NotificationTime.
		Query().
		Where(
			notificationtime.TypeEQ(notificationtime.Type(notifType.Value)),
			notificationtime.And(
				notificationtime.SendMinuteGTE(0),
				notificationtime.SendMinuteLTE(timestampMinutesFromMidnight),
			),
			notificationtime.Not(
				notificationtime.HasProfileWith(profile.HasNotificationsWith(
					notification.CreatedAtGTE(prevMidnightTimestamp),
					notification.TypeEQ(notification.Type(notifType.Value)),
				)),
			),
		)

	if profileIDs != nil {
		query.Where(notificationtime.HasProfileWith(profile.IDIn(*profileIDs...)))
	}

	profiles, err := query.QueryProfile().
		Select(profile.FieldID).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]int, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, p.ID)
	}
	return out, nil
}
