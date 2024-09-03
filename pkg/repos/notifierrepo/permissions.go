package notifierrepo

import (
	"context"
	"errors"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gofrs/uuid"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/notificationpermission"
	"github.com/mikestefanello/pagoda/ent/profile"
	"github.com/mikestefanello/pagoda/pkg/domain"
)

type NotificationSendPermissionRepo struct {
	orm *ent.Client
}

func NewNotificationSendPermissionRepo(orm *ent.Client) *NotificationSendPermissionRepo {
	return &NotificationSendPermissionRepo{
		orm: orm,
	}
}

func (p *NotificationSendPermissionRepo) deleteAllPermissions(
	ctx context.Context, profileID int, platform *domain.NotificationPlatform) error {

	query := p.orm.NotificationPermission.
		Delete().
		Where(
			notificationpermission.HasProfileWith(profile.IDEQ(profileID)),
		)

	if platform != nil {
		query.Where(
			notificationpermission.PlatformEQ(notificationpermission.Platform(platform.Value)),
		)
	}

	_, err := query.
		Exec(ctx)

	return err
}

// GetPermissions returns all permissions specifying which ones the profile has or not.
func (p *NotificationSendPermissionRepo) GetPermissions(
	ctx context.Context, profileID int,
) (map[domain.NotificationPermissionType]domain.NotificationPermission, error) {

	entPerms, err := p.orm.NotificationPermission.
		Query().
		Where(
			notificationpermission.HasProfileWith(profile.IDEQ(profileID)),
		).
		All(ctx)

	if err != nil {
		return nil, err
	}

	userPermsSet := mapset.NewSet[string]()
	tempPermsMap := make(map[string]map[domain.NotificationPlatform]bool)

	// Create a temporary map of permissions, turning on the ones the user has.
	for _, entPerm := range entPerms {
		perm := *domain.NotificationPermissions.Parse(entPerm.Permission.String())
		userPermsSet.Add(perm.Value)

		platform := *domain.NotificationPlatforms.Parse(entPerm.Platform.String())

		// Initialize the platform map if it's nil
		if tempPermsMap[perm.Value] == nil {
			tempPermsMap[perm.Value] = make(map[domain.NotificationPlatform]bool)
		}

		tempPermsMap[perm.Value][platform] = true
	}

	permsMap := make(map[domain.NotificationPermissionType]domain.NotificationPermission)

	// Iterate through all permissions and platforms to ensure all are represented
	for _, perm := range domain.NotificationPermissions.Members() {
		pushPermObj, ok := domain.NotificationPermissionMap[perm]
		if !ok {
			return nil, errors.New("failed to find push permission in NotificationPermissionMap")
		}

		// Initialize the platform map if it's nil
		if tempPermsMap[perm.Value] == nil {
			tempPermsMap[perm.Value] = make(map[domain.NotificationPlatform]bool)
		}

		for _, plat := range domain.NotificationPlatforms.Members() {
			if !userPermsSet.Contains(perm.Value) || tempPermsMap[perm.Value][plat] == false {
				tempPermsMap[perm.Value][plat] = false
			}
		}

		// Convert the temporary map to PlatformsList
		pushPermObj.PlatformsList = []domain.NotificationPermissionPlatform{}
		if platforms, ok := tempPermsMap[pushPermObj.Permission]; ok {
			for platform, granted := range platforms {
				pushPermObj.PlatformsList = append(pushPermObj.PlatformsList, domain.NotificationPermissionPlatform{
					Platform: platform.Value,
					Granted:  granted,
				})
			}
		}

		permsMap[perm] = pushPermObj
	}

	return permsMap, nil
}

// CreatePermission create a permission type for one or all platforms.
func (p *NotificationSendPermissionRepo) CreatePermission(
	ctx context.Context, profileID int, permission domain.NotificationPermissionType, platform *domain.NotificationPlatform,
) (err error) {
	if platform != nil {
		uuidToken, err := uuid.NewV7(uuid.MicrosecondPrecision)
		if err != nil {
			return err
		}

		_, err = p.orm.NotificationPermission.
			Create().
			SetPermission(notificationpermission.Permission(permission.Value)).
			SetPlatform(notificationpermission.Platform(platform.Value)).
			SetProfileID(profileID).
			SetToken(uuidToken.String()).
			Save(ctx)

	} else {
		for _, plat := range domain.NotificationPlatforms.Members() {
			uuidToken, err := uuid.NewV7(uuid.MicrosecondPrecision)
			if err != nil {
				return err
			}

			_, err = p.orm.NotificationPermission.
				Create().
				SetPermission(notificationpermission.Permission(permission.Value)).
				SetPlatform(notificationpermission.Platform(plat.Value)).
				SetProfileID(profileID).
				SetToken(uuidToken.String()).
				Save(ctx)
			if err != nil {
				return err
			}
		}
	}

	return err
}

// DeletePermission deletes a permission type for one or all platforms.
func (p *NotificationSendPermissionRepo) DeletePermission(
	ctx context.Context,
	profileID int,
	permission domain.NotificationPermissionType,
	platform *domain.NotificationPlatform,
	token *string,
) (err error) {
	query := p.orm.NotificationPermission.
		Query().
		Where(
			notificationpermission.HasProfileWith(profile.IDEQ(profileID)),
			notificationpermission.PermissionEQ(notificationpermission.Permission(permission.Value)),
		)
	if token != nil {
		query.Where(
			notificationpermission.TokenEQ(*token),
		)
	}

	if platform != nil {
		query.Where(
			notificationpermission.PlatformEQ(notificationpermission.Platform(platform.Value)),
		)
	}
	id, err := query.OnlyID(ctx)
	if err != nil {
		return err
	}

	return p.orm.NotificationPermission.DeleteOneID(id).Exec(ctx)
}

// HasPermission checks whether a user still has notification permission for a specific notification platform.
func (p *NotificationSendPermissionRepo) HasPermissionsForPlatform(
	ctx context.Context,
	profileID int,
	platform domain.NotificationPlatform,
) (bool, error) {
	count, err := p.orm.NotificationPermission.
		Query().
		Where(
			notificationpermission.HasProfileWith(profile.IDEQ(profileID)),
			notificationpermission.PlatformEQ(notificationpermission.Platform(platform.Value)),
		).
		Count(ctx)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
