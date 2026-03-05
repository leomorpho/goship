package notifications

import (
	"context"
	"errors"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gofrs/uuid"
	"github.com/leomorpho/goship/framework/domain"
)

type notificationPermissionRecord struct {
	Permission string
	Platform   string
	Token      string
}

type notificationPermissionStorage interface {
	deleteAllPermissions(ctx context.Context, profileID int, platform *domain.NotificationPlatform) error
	listPermissionsByProfileID(ctx context.Context, profileID int) ([]notificationPermissionRecord, error)
	createPermission(ctx context.Context, profileID int, permission domain.NotificationPermissionType, platform domain.NotificationPlatform, token string) error
	deletePermission(ctx context.Context, profileID int, permission domain.NotificationPermissionType, platform *domain.NotificationPlatform, token *string) error
	countPermissionsForPlatform(ctx context.Context, profileID int, platform domain.NotificationPlatform) (int, error)
}

type NotificationPermissionService struct {
	store notificationPermissionStorage
}

func NewNotificationPermissionServiceWithStore(store notificationPermissionStorage) *NotificationPermissionService {
	return &NotificationPermissionService{store: store}
}

func (p *NotificationPermissionService) deleteAllPermissions(
	ctx context.Context, profileID int, platform *domain.NotificationPlatform,
) error {
	return p.store.deleteAllPermissions(ctx, profileID, platform)
}

// GetPermissions returns all permissions specifying which ones the profile has or not.
func (p *NotificationPermissionService) GetPermissions(
	ctx context.Context, profileID int,
) (map[domain.NotificationPermissionType]domain.NotificationPermission, error) {
	records, err := p.store.listPermissionsByProfileID(ctx, profileID)
	if err != nil {
		return nil, err
	}

	userPermsSet := mapset.NewSet[string]()
	tempPermsMap := make(map[string]map[domain.NotificationPlatform]bool)

	for _, rec := range records {
		perm := domain.NotificationPermissions.Parse(rec.Permission)
		platform := domain.NotificationPlatforms.Parse(rec.Platform)
		if perm == nil || platform == nil {
			continue
		}

		userPermsSet.Add(perm.Value)
		if tempPermsMap[perm.Value] == nil {
			tempPermsMap[perm.Value] = make(map[domain.NotificationPlatform]bool)
		}
		tempPermsMap[perm.Value][*platform] = true
	}

	permsMap := make(map[domain.NotificationPermissionType]domain.NotificationPermission)
	for _, perm := range domain.NotificationPermissions.Members() {
		pushPermObj, ok := domain.NotificationPermissionMap[perm]
		if !ok {
			return nil, errors.New("failed to find push permission in NotificationPermissionMap")
		}

		if tempPermsMap[perm.Value] == nil {
			tempPermsMap[perm.Value] = make(map[domain.NotificationPlatform]bool)
		}

		for _, plat := range domain.NotificationPlatforms.Members() {
			if !userPermsSet.Contains(perm.Value) || tempPermsMap[perm.Value][plat] == false {
				tempPermsMap[perm.Value][plat] = false
			}
		}

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
func (p *NotificationPermissionService) CreatePermission(
	ctx context.Context, profileID int, permission domain.NotificationPermissionType, platform *domain.NotificationPlatform,
) error {
	if platform != nil {
		uuidToken, err := uuid.NewV7(uuid.MicrosecondPrecision)
		if err != nil {
			return err
		}
		return p.store.createPermission(ctx, profileID, permission, *platform, uuidToken.String())
	}

	for _, plat := range domain.NotificationPlatforms.Members() {
		uuidToken, err := uuid.NewV7(uuid.MicrosecondPrecision)
		if err != nil {
			return err
		}
		if err := p.store.createPermission(ctx, profileID, permission, plat, uuidToken.String()); err != nil {
			return err
		}
	}

	return nil
}

// DeletePermission deletes a permission type for one or all platforms.
func (p *NotificationPermissionService) DeletePermission(
	ctx context.Context,
	profileID int,
	permission domain.NotificationPermissionType,
	platform *domain.NotificationPlatform,
	token *string,
) error {
	return p.store.deletePermission(ctx, profileID, permission, platform, token)
}

// HasPermissionsForPlatform checks whether a user has any permission for a platform.
func (p *NotificationPermissionService) HasPermissionsForPlatform(
	ctx context.Context,
	profileID int,
	platform domain.NotificationPlatform,
) (bool, error) {
	count, err := p.store.countPermissionsForPlatform(ctx, profileID, platform)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
