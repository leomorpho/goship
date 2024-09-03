package notifierrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/notification"
	"github.com/mikestefanello/pagoda/ent/profile"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/rs/zerolog/log"
)

// NotificationStorage defines the interface for storage operations on notifications.
type NotificationStorage interface {
	CreateNotification(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	GetNotificationsByProfileID(ctx context.Context, profileID int, onlyUnread bool, beforeTimestamp *time.Time, pageSize *int) ([]*domain.Notification, error)
	MarkNotificationAsRead(ctx context.Context, notificationID int, profileID *int) error
	MarkAllNotificationAsRead(ctx context.Context, profileID int) error
	MarkNotificationAsUnread(ctx context.Context, notificationID int, profileID *int) error
	DeleteNotification(ctx context.Context, notificationID int, profileID *int) error
	HasNotificationForResourceAndPerson(ctx context.Context, notifType domain.NotificationType, profileIDWhoCausedNotif, resourceID *int, maxAge time.Duration) (exists bool, err error)
}

type NotificationStorageRepo struct {
	NotificationStorage
	orm *ent.Client
}

// ConvertEntToDomain converts an Ent Notification to a domain Notification.
func ConvertEntToDomain(e *ent.Notification) *domain.Notification {
	if e.Edges.Profile == nil {
		log.Fatal().Str("missing edge", "Profile edge should be set")
	}
	notification := &domain.Notification{
		ID:        e.ID,
		Type:      *domain.NotificationTypes.Parse(string(e.Type)),
		Title:     e.Title,
		Text:      e.Text,
		CreatedAt: e.CreatedAt,
		Read:      e.Read,
		ProfileID: e.Edges.Profile.ID,
	}
	if e.ReadInNotificationsCenter != nil {
		notification.ReadInNotificationsCenter = *e.ReadInNotificationsCenter
	}

	if e.Link != nil {
		notification.Link = *e.Link
	}
	if e.ReadAt != nil {
		notification.ReadAt = *e.ReadAt
	}
	return notification
}

func NewNotificationStorageRepo(orm *ent.Client) *NotificationStorageRepo {
	return &NotificationStorageRepo{
		orm: orm,
	}
}

// CreateNotification creates a new notification.
func (r *NotificationStorageRepo) CreateNotification(ctx context.Context, n domain.Notification) (*domain.Notification, error) {
	created, err := r.orm.Notification.
		Create().
		SetType(notification.Type(n.Type.Value)).
		SetTitle(n.Title).
		SetText(n.Text).
		SetNillableLink(&n.Link).
		SetRead(false).
		SetProfileID(n.ProfileID).
		SetProfileIDWhoCausedNotification(n.ProfileIDWhoCausedNotif).
		SetResourceIDTiedToNotif(n.ResourceIDTiedToNotif).
		SetReadInNotificationsCenter(n.ReadInNotificationsCenter).
		Save(ctx)

	if err != nil {
		log.Error().AnErr("NewNotificationStorageRepo", err).Msg("Error creating a notification in DB")
		return nil, err
	}

	// Load the Profile edge for the created notification
	created, err = r.orm.Notification.
		Query().
		Where(notification.IDEQ(created.ID)).
		WithProfile(). // This loads the Profile edge
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return ConvertEntToDomain(created), nil
}

// HasNotificationForResourceAndPerson returns true if there is an existing
// notification for a resource
func (r *NotificationStorageRepo) HasNotificationForResourceAndPerson(
	ctx context.Context, notifType domain.NotificationType, profileIDWhoCausedNotif, resourceID *int, maxAge time.Duration,
) (exists bool, err error) {
	cutoff := time.Now().Add(-maxAge) // Calculate the cutoff timestamp

	query := r.orm.Notification.
		Query().
		Where(
			notification.TypeEQ(notification.Type(notifType.Value)),
			notification.CreatedAtGT(cutoff), // Only include notifications newer than the cutoff
		)

	if profileIDWhoCausedNotif != nil {
		query.Where(
			notification.ProfileIDWhoCausedNotification(*profileIDWhoCausedNotif),
		)
	}
	if resourceID != nil {
		query.Where(
			notification.ResourceIDTiedToNotifEQ(*resourceID),
		)
	}
	return query.Exist(ctx)
}

// GetNotificationsByProfileID retrieves all notifications for a given profile ID.
// If onlyUnread is true, it fetches only those notifications that haven't been read.
func (r *NotificationStorageRepo) GetNotificationsByProfileID(
	ctx context.Context, profileID int, onlyUnread bool, beforeTimestamp *time.Time, pageSize *int,
) ([]*domain.Notification, error) {
	query := r.orm.Notification.
		Query().
		Where(
			notification.HasProfileWith(profile.IDEQ(profileID)),
		).WithProfile() // TODO: optimize to only get profile

	if onlyUnread {
		query = query.Where(notification.ReadEQ(false))
	}
	if beforeTimestamp != nil {
		query.Where(
			notification.CreatedAtLT(*beforeTimestamp),
		)
	}

	query.Order(ent.Desc(notification.FieldCreatedAt))

	if pageSize != nil {
		query.Limit(*pageSize)
	}

	entNotifs, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	var domainNotifs []*domain.Notification
	for _, entNotif := range entNotifs {
		domainNotifs = append(domainNotifs, ConvertEntToDomain(entNotif))
	}

	return domainNotifs, nil
}

// MarkNotificationAsRead updates the specified notification's read status to true.
func (r *NotificationStorageRepo) MarkNotificationAsRead(
	ctx context.Context, notificationID int, profileID *int,
) error {
	// If profileID is provided, check that the notification belongs to this profile
	if profileID != nil {
		if err := r.checkNotificationBelongsToProfile(ctx, *profileID, notificationID); err != nil {
			return err
		}
	}
	notif, err := r.orm.Notification.Query().
		Where(
			notification.IDEQ(notificationID),
		).
		Only(ctx)
	if err != nil {
		return err
	}

	deletable := domain.DeleteOnceReadNotificationTypesMap[*domain.NotificationTypes.Parse(notif.Type.String())]
	if deletable {
		return r.orm.Notification.
			DeleteOneID(notificationID).
			Exec(ctx)
	} else {
		// Update the notification in the database
		_, err := r.orm.Notification.
			UpdateOneID(notificationID).
			SetRead(true).
			SetReadAt(time.Now().UTC()).
			Save(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

// MarkAllNotificationAsRead updates all notifications' read status to true for a given profile ID.
func (r *NotificationStorageRepo) MarkAllNotificationAsRead(
	ctx context.Context, profileID int,
) error {
	// Perform the update in the database
	_, err := r.orm.Notification.
		Update().
		Where(notification.HasProfileWith(profile.IDEQ(profileID))).
		SetRead(true).
		SetReadAt(time.Now().UTC()).
		Save(ctx)

	if err != nil {
		log.Error().AnErr("MarkAllNotificationAsRead", err).Msg("Error marking all notifications as read")
	}
	return err
}

// MarkNotificationAsRead updates the specified notification's read status to true.
func (r *NotificationStorageRepo) MarkNotificationAsUnread(
	ctx context.Context, notificationID int, profileID *int,
) error {
	// If profileID is provided, check that the notification belongs to this profile
	if profileID != nil {
		if err := r.checkNotificationBelongsToProfile(ctx, *profileID, notificationID); err != nil {
			return err
		}
	}

	// Update the notification in the database
	_, err := r.orm.Notification.
		UpdateOneID(notificationID).
		SetRead(false).
		SetReadAt(time.Time{}).
		Save(ctx)
	return err
}

// DeleteNotification deletes a notification by its ID.
func (r *NotificationStorageRepo) DeleteNotification(
	ctx context.Context, notificationID int, profileID *int,
) error {
	// If profileID is provided, check that the notification belongs to this profile
	if profileID != nil {
		if err := r.checkNotificationBelongsToProfile(ctx, *profileID, notificationID); err != nil {
			return err
		}
	}

	// Perform the deletion in the database
	err := r.orm.Notification.
		DeleteOneID(notificationID).
		Exec(ctx)

	if err != nil {
		log.Error().AnErr("DeleteNotification", err).Msg("Error deleting a notification")
	}
	return err
}

func (r NotificationStorageRepo) checkNotificationBelongsToProfile(
	ctx context.Context, profileID, notificationID int,
) error {
	count, err := r.orm.Notification.
		Query().
		Where(notification.IDEQ(notificationID)).
		Where(notification.HasProfileWith(profile.IDEQ(profileID))).
		Count(ctx)

	if err != nil {
		log.Error().AnErr("DeleteNotification", err).Msg("Error querying the notification")
		return err
	}

	// If the notification does not belong to the profile, return an error
	if count == 0 {
		return fmt.Errorf("notification does not belong to the provided profile")
	}
	return nil
}
