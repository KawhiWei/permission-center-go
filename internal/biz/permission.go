package biz

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type ResourceType string

const (
	ResourceTypeMenu   ResourceType = "menu"
	ResourceTypeButton ResourceType = "button"
)

type Role struct {
	BaseFields
	ID          string
	Application string
	Code        string
	Name        string
	Description string
	Enabled     bool
}

// Resource represents a navigational menu or an operation button. ParentID is nil for roots.
type Resource struct {
	BaseFields
	ID          uuid.UUID
	Application string
	ParentID    *uuid.UUID
	Code        string
	Name        string
	Description string
	Type        ResourceType
	Path        string
	Component   string
	APIPath     string
	HTTPMethod  string
	Icon        string
	Sort        int
	Enabled     bool
}

type ResourceTreeNode struct {
	Resource
	Children []*ResourceTreeNode
}

type RoleRepository interface {
	Create(context.Context, *Role) (*Role, error)
	Get(context.Context, string) (*Role, error)
	ListByApplication(context.Context, string) ([]*Role, error)
	ReplaceResources(context.Context, string, []uuid.UUID) error
	ResourceIDs(context.Context, string) ([]uuid.UUID, error)
}

type ResourceRepository interface {
	Create(context.Context, *Resource) (*Resource, error)
	Get(context.Context, uuid.UUID) (*Resource, error)
	ListByApplication(context.Context, string) ([]*Resource, error)
}

// UserRoleRepository associates the external OIDC subject with application-scoped roles.
// It deliberately has no local users table: the identity authority is NexusAuth.
type UserRoleRepository interface {
	ReplaceRoles(context.Context, string, string, []string) error
	RoleIDs(context.Context, string, string) ([]string, error)
}

type PermissionService struct {
	roles     RoleRepository
	resources ResourceRepository
	userRoles UserRoleRepository
}

func NewPermissionService(roles RoleRepository, resources ResourceRepository, userRoles ...UserRoleRepository) *PermissionService {
	service := &PermissionService{roles: roles, resources: resources}
	if len(userRoles) > 0 {
		service.userRoles = userRoles[0]
	}
	return service
}

func (s *PermissionService) CreateRole(ctx context.Context, application, code, name, description string) (*Role, error) {
	application, err := validateApplication(application)
	if err != nil {
		return nil, err
	}
	code, name, err = validateCodeAndName(code, name)
	if err != nil {
		return nil, err
	}
	return s.roles.Create(ctx, &Role{BaseFields: NewBaseFields(AuditActorFromContext(ctx)), Application: application, Code: code, Name: name, Description: strings.TrimSpace(description), Enabled: true})
}

func (s *PermissionService) ListRoles(ctx context.Context, application string) ([]*Role, error) {
	application, err := validateApplication(application)
	if err != nil {
		return nil, err
	}
	return s.roles.ListByApplication(ctx, application)
}

func (s *PermissionService) CreateResource(ctx context.Context, resource *Resource) (*Resource, error) {
	if resource == nil {
		return nil, fmt.Errorf("%w: resource is required", ErrInvalidArgument)
	}
	application, err := validateApplication(resource.Application)
	if err != nil {
		return nil, err
	}
	resource.Application = application
	code, name, err := validateCodeAndName(resource.Code, resource.Name)
	if err != nil {
		return nil, err
	}
	resource.Code, resource.Name = code, name
	if err := validateResource(resource); err != nil {
		return nil, err
	}
	if resource.ParentID != nil {
		parent, err := s.resources.Get(ctx, *resource.ParentID)
		if err != nil {
			return nil, err
		}
		if parent.Application != resource.Application || parent.Type != ResourceTypeMenu {
			return nil, fmt.Errorf("%w: parent must be a menu in the same application", ErrConflict)
		}
	}
	resource.Enabled = true
	resource.BaseFields = NewBaseFields(AuditActorFromContext(ctx))
	return s.resources.Create(ctx, resource)
}

func (s *PermissionService) ResourceTree(ctx context.Context, application string) ([]*ResourceTreeNode, error) {
	application, err := validateApplication(application)
	if err != nil {
		return nil, err
	}
	resources, err := s.resources.ListByApplication(ctx, application)
	if err != nil {
		return nil, err
	}
	nodes := make(map[uuid.UUID]*ResourceTreeNode, len(resources))
	for _, resource := range resources {
		nodes[resource.ID] = &ResourceTreeNode{Resource: *resource}
	}
	roots := make([]*ResourceTreeNode, 0)
	for _, resource := range resources {
		node := nodes[resource.ID]
		if resource.ParentID == nil {
			roots = append(roots, node)
			continue
		}
		parent, ok := nodes[*resource.ParentID]
		if !ok {
			return nil, fmt.Errorf("%w: resource parent does not belong to application", ErrConflict)
		}
		parent.Children = append(parent.Children, node)
	}
	return roots, nil
}

func (s *PermissionService) GrantRoleResources(ctx context.Context, roleID string, resourceIDs []uuid.UUID) error {
	roleID, err := validateID(roleID, "role_id")
	if err != nil {
		return err
	}
	role, err := s.roles.Get(ctx, roleID)
	if err != nil {
		return err
	}
	seen := make(map[uuid.UUID]struct{}, len(resourceIDs))
	for _, resourceID := range resourceIDs {
		if resourceID == uuid.Nil {
			return fmt.Errorf("%w: resource_id is required", ErrInvalidArgument)
		}
		if _, ok := seen[resourceID]; ok {
			return fmt.Errorf("%w: duplicate resource_id", ErrInvalidArgument)
		}
		seen[resourceID] = struct{}{}
		resource, err := s.resources.Get(ctx, resourceID)
		if err != nil {
			return err
		}
		if resource.Application != role.Application {
			return fmt.Errorf("%w: role and resource must belong to the same application", ErrConflict)
		}
	}
	return s.roles.ReplaceResources(ctx, roleID, resourceIDs)
}

func (s *PermissionService) RoleResources(ctx context.Context, roleID string) ([]uuid.UUID, error) {
	roleID, err := validateID(roleID, "role_id")
	if err != nil {
		return nil, err
	}
	if _, err := s.roles.Get(ctx, roleID); err != nil {
		return nil, err
	}
	return s.roles.ResourceIDs(ctx, roleID)
}

// ReplaceUserRoles atomically replaces one user's roles in an application only.
// The repository verifies the role IDs remain live and belong to that application.
func (s *PermissionService) ReplaceUserRoles(ctx context.Context, subject, application string, roleIDs []string) error {
	if s.userRoles == nil {
		return fmt.Errorf("user role repository is not configured")
	}
	subject, err := validateID(subject, "user_id")
	if err != nil {
		return err
	}
	application, err = validateApplication(application)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(roleIDs))
	for index, roleID := range roleIDs {
		roleID, err = validateID(roleID, "role_id")
		if err != nil {
			return err
		}
		if _, exists := seen[roleID]; exists {
			return fmt.Errorf("%w: duplicate role_id", ErrInvalidArgument)
		}
		seen[roleID] = struct{}{}
		roleIDs[index] = roleID
	}
	return s.userRoles.ReplaceRoles(ctx, subject, application, roleIDs)
}

func (s *PermissionService) UserRoleIDs(ctx context.Context, subject, application string) ([]string, error) {
	if s.userRoles == nil {
		return nil, fmt.Errorf("user role repository is not configured")
	}
	subject, err := validateID(subject, "user_id")
	if err != nil {
		return nil, err
	}
	application, err = validateApplication(application)
	if err != nil {
		return nil, err
	}
	return s.userRoles.RoleIDs(ctx, subject, application)
}

func validateCodeAndName(code, name string) (string, string, error) {
	code, name = strings.TrimSpace(code), strings.TrimSpace(name)
	if code == "" || len(code) > 100 {
		return "", "", fmt.Errorf("%w: code must be 1-100 characters", ErrInvalidArgument)
	}
	if name == "" || len([]rune(name)) > 100 {
		return "", "", fmt.Errorf("%w: name must be 1-100 characters", ErrInvalidArgument)
	}
	return code, name, nil
}

func validateApplication(application string) (string, error) {
	application = strings.TrimSpace(application)
	if application == "" || len(application) > 100 {
		return "", fmt.Errorf("%w: application must be 1-100 characters", ErrInvalidArgument)
	}
	return application, nil
}

func validateID(value, name string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 80 {
		return "", fmt.Errorf("%w: %s must be 1-80 characters", ErrInvalidArgument, name)
	}
	return value, nil
}

func validateResource(resource *Resource) error {
	code, name, err := validateCodeAndName(resource.Code, resource.Name)
	if err != nil {
		return err
	}
	resource.Code, resource.Name = code, name
	if resource.Type != ResourceTypeMenu && resource.Type != ResourceTypeButton {
		return fmt.Errorf("%w: type must be menu or button", ErrInvalidArgument)
	}
	if resource.Type == ResourceTypeButton && resource.ParentID == nil {
		return fmt.Errorf("%w: button must have a menu parent", ErrInvalidArgument)
	}
	if resource.Type == ResourceTypeMenu && resource.APIPath != "" {
		return fmt.Errorf("%w: menu cannot define api_path", ErrInvalidArgument)
	}
	if resource.Type == ResourceTypeButton && strings.TrimSpace(resource.APIPath) == "" {
		return fmt.Errorf("%w: button api_path is required", ErrInvalidArgument)
	}
	return nil
}
