package notifications

import (
	"context"

	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/db/ent/notificationpermission"
	"github.com/leomorpho/goship/db/ent/profile"
	"github.com/leomorpho/goship/framework/domain"
)

type entNotificationPermissionStore struct {
	orm *ent.Client
}

func NewNotificationPermissionService(orm *ent.Client) *NotificationPermissionService {
	return NewNotificationPermissionServiceWithStore(newEntNotificationPermissionStore(orm))
}

func newEntNotificationPermissionStore(orm *ent.Client) *entNotificationPermissionStore {
	return &entNotificationPermissionStore{orm: orm}
}

func (s *entNotificationPermissionStore) deleteAllPermissions(
	ctx context.Context, profileID int, platform *domain.NotificationPlatform,
) error {
	query := s.orm.NotificationPermission.
		Delete().
		Where(notificationpermission.HasProfileWith(profile.IDEQ(profileID)))

	if platform != nil {
		query.Where(notificationpermission.PlatformEQ(notificationpermission.Platform(platform.Value)))
	}

	_, err := query.Exec(ctx)
	return err
}

func (s *entNotificationPermissionStore) listPermissionsByProfileID(
	ctx context.Context, profileID int,
) ([]notificationPermissionRecord, error) {
	entPerms, err := s.orm.NotificationPermission.
		Query().
		Where(notificationpermission.HasProfileWith(profile.IDEQ(profileID))).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]notificationPermissionRecord, 0, len(entPerms))
	for _, perm := range entPerms {
		out = append(out, notificationPermissionRecord{
			Permission: perm.Permission.String(),
			Platform:   perm.Platform.String(),
			Token:      perm.Token,
		})
	}
	return out, nil
}

func (s *entNotificationPermissionStore) createPermission(
	ctx context.Context, profileID int, permission domain.NotificationPermissionType, platform domain.NotificationPlatform, token string,
) error {
	_, err := s.orm.NotificationPermission.
		Create().
		SetPermission(notificationpermission.Permission(permission.Value)).
		SetPlatform(notificationpermission.Platform(platform.Value)).
		SetProfileID(profileID).
		SetToken(token).
		Save(ctx)
	return err
}

func (s *entNotificationPermissionStore) deletePermission(
	ctx context.Context,
	profileID int,
	permission domain.NotificationPermissionType,
	platform *domain.NotificationPlatform,
	token *string,
) error {
	query := s.orm.NotificationPermission.
		Query().
		Where(
			notificationpermission.HasProfileWith(profile.IDEQ(profileID)),
			notificationpermission.PermissionEQ(notificationpermission.Permission(permission.Value)),
		)
	if token != nil {
		query.Where(notificationpermission.TokenEQ(*token))
	}
	if platform != nil {
		query.Where(notificationpermission.PlatformEQ(notificationpermission.Platform(platform.Value)))
	}

	id, err := query.OnlyID(ctx)
	if err != nil {
		return err
	}
	return s.orm.NotificationPermission.DeleteOneID(id).Exec(ctx)
}

func (s *entNotificationPermissionStore) countPermissionsForPlatform(
	ctx context.Context, profileID int, platform domain.NotificationPlatform,
) (int, error) {
	return s.orm.NotificationPermission.
		Query().
		Where(
			notificationpermission.HasProfileWith(profile.IDEQ(profileID)),
			notificationpermission.PlatformEQ(notificationpermission.Platform(platform.Value)),
		).
		Count(ctx)
}
