package permissions

import (
	"fmt"

	pgadapter "github.com/casbin/casbin-pg-adapter"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/orsinium-labs/enum"
)

/*
Permissions are linked to the following tuple: (tenant, subject, object, action)

- TENANT: meant to support SAAS with multiple different groups of logically isolated users,
like different companies. Set to a constant if you have no tenants.

- SUBJECT: some entity.

- OBJECT: the thing that has the permission.

- ACTION: what action is allowed on the object by a subject and tenant.
*/
type (
	TenantID  string
	SubjectID string

	PermissionObject enum.Member[string]
	PermissionAction enum.Member[string]
)

var (
	QuestionPolicy = PermissionObject{"question"}

	ObjectsWithPolicies = enum.New(
		QuestionPolicy,
	)

	ReadAction    = PermissionAction{"read"}
	CreateAction  = PermissionAction{"create"}
	PublishAction = PermissionAction{"publish"}

	ActionPolicies = enum.New(
		ReadAction,
		CreateAction,
		PublishAction,
	)
)

// CasbinAdapter defines the interface for loading, saving, and managing policies
type CasbinAdapter interface {
	persist.Adapter
}

func NewPostgresCasbinAdapter(dsn string) (CasbinAdapter, error) {
	return pgadapter.NewAdapter(dsn)
}

// PermissionClient struct to interact with Casbin
type PermissionClient struct {
	enforcer            *casbin.Enforcer
	useFilteredPolicies bool
}

// NewPermissionClient creates a new PermissionClient
func NewPermissionClient(
	modelStr string,
	adapter CasbinAdapter,
	useFilteredPolicies bool,
) (*PermissionClient, error) {

	m, err := model.NewModelFromString(modelStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create model from string: %w", err)
	}

	// Initialize the Casbin enforcer using the adapter
	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Casbin enforcer: %w", err)
	}

	return &PermissionClient{
		enforcer:            enforcer,
		useFilteredPolicies: useFilteredPolicies,
	}, nil
}

// LoadTenantPolicies loads policies for a specific tenant.
func (s *PermissionClient) LoadTenantPolicies(tenantID string) error {
	if s.useFilteredPolicies {
		filter := &pgadapter.Filter{
			P: []string{tenantID}, // Assuming the tenant_id is the first field in the policy
			G: []string{tenantID}, // Same assumption for the grouping policies
		}
		return s.enforcer.LoadFilteredPolicy(filter)
	}

	// For adapters not supporting filtered policies or when filtered policies are not required
	return s.enforcer.LoadPolicy()
}

// EnsureTenantPolicyLoaded ensures the policy for a given tenant is loaded.
func (s *PermissionClient) EnsureTenantPolicyLoaded(tenantID string) error {

	// Load policy for the tenant
	if err := s.LoadTenantPolicies(tenantID); err != nil {
		return err // Handle error from policy loading
	}
	return nil
}

// CheckPermission checks if a user has permission to perform an action on an object
func (s *PermissionClient) CheckPermission(tenantID, sub, obj, act string) (bool, error) {
	if err := s.EnsureTenantPolicyLoaded(tenantID); err != nil {
		return false, err
	}

	// Proceed with enforcing and update the cache
	allowed, err := s.enforcer.Enforce(sub, tenantID, obj, act)
	if err != nil {
		return false, err
	}
	return allowed, err
}

// AddPermission adds a new permission to the policy
func (s *PermissionClient) AddPolicy(tenantID, sub, obj, act string) (bool, error) {
	if err := s.EnsureTenantPolicyLoaded(tenantID); err != nil {
		return false, err
	}

	// Attempt to add the policy
	added, err := s.enforcer.AddPolicy(sub, tenantID, obj, act)
	if err != nil {
		return false, err
	}
	if added {
		err = s.enforcer.SavePolicy()
		if err != nil {
			return false, err
		}

	}
	return added, err
}

func (s *PermissionClient) AddGroupingPolicy(tenantID, sub, group string) (bool, error) {
	if err := s.EnsureTenantPolicyLoaded(tenantID); err != nil {
		return false, err
	}

	// TODO: I suspect there may be stale permissions until the cache
	// gets dropped. Not sure how to cleanly invalidate cached permissions for a specific
	// tenant or subject when adding roles like that. It might not be worth it,
	// and waiting for the cache to become stale might be easiest (by keeping it
	// to 20-30min or so).
	added, err := s.enforcer.AddGroupingPolicy(sub, group, tenantID)
	if err != nil {
		return false, err
	}
	if added {
		err = s.enforcer.SavePolicy()
		if err != nil {
			return false, err
		}

	}

	return added, nil

}

// RemovePolicy removes a permission from the policy
func (s *PermissionClient) RemovePolicy(tenantID, sub, obj, act string) (bool, error) {
	if err := s.EnsureTenantPolicyLoaded(tenantID); err != nil {
		return false, err
	}

	// Attempt to remove the policy
	removed, err := s.enforcer.RemovePolicy(sub, tenantID, obj, act)
	if err != nil {
		return false, err
	}
	if removed {
		err = s.enforcer.SavePolicy()
		if err != nil {
			return false, err
		}
	}
	return removed, err
}

func (s *PermissionClient) RemoveGroupingPolicy(tenantID, sub, group string) (bool, error) {
	if err := s.EnsureTenantPolicyLoaded(tenantID); err != nil {
		return false, err
	}

	// TODO: I suspect there may be stale permissions until the cache
	// gets dropped. Not sure how to cleanly invalidate cached permissions for a specific
	// tenant or subject when adding roles like that. It might not be worth it,
	// and waiting for the cache to become stale might be easiest (by keeping it
	// to 20-30min or so).
	added, err := s.enforcer.RemoveGroupingPolicy(sub, group, tenantID)
	if err != nil {
		return false, err
	}
	if added {
		err = s.enforcer.SavePolicy()
		if err != nil {
			return false, err
		}

	}

	return added, nil
}
