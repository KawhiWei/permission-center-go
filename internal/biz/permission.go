package biz

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type MenuType string

const (
	MenuTypeMenu   MenuType = "menu"
	MenuTypeButton MenuType = "button"
)

type Role struct {
	BaseFields
	ID              string `json:"id"`
	ServiceResource string `json:"service_resource"`
	// Application is retained as a wire-compatible alias for existing clients.
	Application string `json:"application,omitempty"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// Menu represents a navigational menu or an operation button. ParentID is nil for roots.
type Menu struct {
	BaseFields
	ID              uuid.UUID `json:"id"`
	ServiceResource string    `json:"service_resource"`
	// Application is retained as a wire-compatible alias for existing clients.
	Application string     `json:"application,omitempty"`
	ParentID    *uuid.UUID `json:"parent_id"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Type        MenuType   `json:"type"`
	Path        string     `json:"path"`
	Component   string     `json:"component"`
	APIPath     string     `json:"api_path"`
	HTTPMethod  string     `json:"http_method"`
	Icon        string     `json:"icon"`
	Sort        int        `json:"sort"`
	Enabled     bool       `json:"enabled"`
}

type MenuTreeNode struct {
	Menu
	Children []*MenuTreeNode `json:"children"`
}

type RoleRepository interface {
	Create(context.Context, *Role) (*Role, error)
	Get(context.Context, string) (*Role, error)
	ListByApplication(context.Context, string) ([]*Role, error)
	ReplaceMenus(context.Context, string, []uuid.UUID) error
	MenuIDs(context.Context, string) ([]uuid.UUID, error)
}

type MenuRepository interface {
	Create(context.Context, *Menu) (*Menu, error)
	Get(context.Context, uuid.UUID) (*Menu, error)
	ListByApplication(context.Context, string) ([]*Menu, error)
}

// UserRoleRepository associates the external OIDC subject with application-scoped roles.
// It deliberately has no local users table: the identity authority is NexusAuth.
type UserRoleRepository interface {
	ReplaceRoles(context.Context, string, string, []string) error
	RoleIDs(context.Context, string, string) ([]string, error)
}

type PermissionService struct {
	roles            RoleRepository
	menus            MenuRepository
	userRoles        UserRoleRepository
	applications     ApplicationRepository
	serviceResources ServiceResourceCatalog
}

// WithApplicationRepository enables application existence checks without
// breaking callers that intentionally use an externally managed catalog.
func (s *PermissionService) WithApplicationRepository(applications ApplicationRepository) *PermissionService {
	s.applications = applications
	return s
}

// WithServiceResourceCatalog makes service-resource metadata the source of
// truth for scope validation. The old application repository remains a
// fallback for callers that have not opted into the new catalog yet.
func (s *PermissionService) WithServiceResourceCatalog(catalog ServiceResourceCatalog) *PermissionService {
	s.serviceResources = catalog
	return s
}

func (s *PermissionService) ensureApplication(ctx context.Context, application string) error {
	return s.ensureServiceResource(ctx, application)
}

func (s *PermissionService) ensureServiceResource(ctx context.Context, serviceResource string) error {
	if s.serviceResources != nil {
		value, err := s.serviceResources.Get(ctx, serviceResource)
		if err != nil {
			return err
		}
		if !value.IsActive {
			return fmt.Errorf("%w: service resource is disabled", ErrConflict)
		}
		return nil
	}
	if s.applications == nil {
		return nil
	}
	value, err := s.applications.Get(ctx, serviceResource)
	if err != nil {
		return err
	}
	if !value.Enabled {
		return fmt.Errorf("%w: application is disabled", ErrConflict)
	}
	return nil
}

func NewPermissionService(roles RoleRepository, menus MenuRepository, userRoles ...UserRoleRepository) *PermissionService {
	service := &PermissionService{roles: roles, menus: menus}
	if len(userRoles) > 0 {
		service.userRoles = userRoles[0]
	}
	return service
}

func (s *PermissionService) CreateRole(ctx context.Context, application, code, name, description string) (*Role, error) {
	serviceResource, err := validateServiceResource(application)
	if err != nil {
		return nil, err
	}
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	code, name, err = validateCodeAndName(code, name)
	if err != nil {
		return nil, err
	}
	return s.roles.Create(ctx, &Role{BaseFields: NewBaseFields(AuditActorFromContext(ctx)), ServiceResource: serviceResource, Application: serviceResource, Code: code, Name: name, Description: strings.TrimSpace(description), Enabled: true})
}

func (s *PermissionService) ListRoles(ctx context.Context, application string) ([]*Role, error) {
	serviceResource, err := validateServiceResource(application)
	if err != nil {
		return nil, err
	}
	return s.roles.ListByApplication(ctx, serviceResource)
}

func (s *PermissionService) CreateMenu(ctx context.Context, menu *Menu) (*Menu, error) {
	if menu == nil {
		return nil, fmt.Errorf("%w: menu is required", ErrInvalidArgument)
	}
	serviceResource, err := setServiceResourceScope(menu.ServiceResource, menu.Application)
	if err != nil {
		return nil, err
	}
	menu.ServiceResource, menu.Application = serviceResource, serviceResource
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	code, name, err := validateCodeAndName(menu.Code, menu.Name)
	if err != nil {
		return nil, err
	}
	menu.Code, menu.Name = code, name
	if err := validateMenu(menu); err != nil {
		return nil, err
	}
	if menu.ParentID != nil {
		parent, err := s.menus.Get(ctx, *menu.ParentID)
		if err != nil {
			return nil, err
		}
		if serviceResourceValue(parent.ServiceResource, parent.Application) != serviceResource || parent.Type != MenuTypeMenu {
			return nil, fmt.Errorf("%w: parent must be a menu in the same application", ErrConflict)
		}
	}
	menu.Enabled = true
	menu.BaseFields = NewBaseFields(AuditActorFromContext(ctx))
	return s.menus.Create(ctx, menu)
}

func (s *PermissionService) MenuTree(ctx context.Context, application string) ([]*MenuTreeNode, error) {
	serviceResource, err := validateServiceResource(application)
	if err != nil {
		return nil, err
	}
	menus, err := s.menus.ListByApplication(ctx, serviceResource)
	if err != nil {
		return nil, err
	}
	nodes := make(map[uuid.UUID]*MenuTreeNode, len(menus))
	for _, menu := range menus {
		nodes[menu.ID] = &MenuTreeNode{Menu: *menu}
	}
	roots := make([]*MenuTreeNode, 0)
	for _, menu := range menus {
		node := nodes[menu.ID]
		if menu.ParentID == nil {
			roots = append(roots, node)
			continue
		}
		parent, ok := nodes[*menu.ParentID]
		if !ok {
			return nil, fmt.Errorf("%w: menu parent does not belong to application", ErrConflict)
		}
		parent.Children = append(parent.Children, node)
	}
	return roots, nil
}

func (s *PermissionService) GrantRoleMenus(ctx context.Context, roleID string, menuIDs []uuid.UUID) error {
	roleID, err := validateID(roleID, "role_id")
	if err != nil {
		return err
	}
	role, err := s.roles.Get(ctx, roleID)
	if err != nil {
		return err
	}
	seen := make(map[uuid.UUID]struct{}, len(menuIDs))
	for _, menuID := range menuIDs {
		if menuID == uuid.Nil {
			return fmt.Errorf("%w: menu_id is required", ErrInvalidArgument)
		}
		if _, ok := seen[menuID]; ok {
			return fmt.Errorf("%w: duplicate menu_id", ErrInvalidArgument)
		}
		seen[menuID] = struct{}{}
		menu, err := s.menus.Get(ctx, menuID)
		if err != nil {
			return err
		}
		if serviceResourceValue(menu.ServiceResource, menu.Application) != serviceResourceValue(role.ServiceResource, role.Application) {
			return fmt.Errorf("%w: role and menu must belong to the same application", ErrConflict)
		}
	}
	return s.roles.ReplaceMenus(ctx, roleID, menuIDs)
}

func (s *PermissionService) RoleMenus(ctx context.Context, roleID string) ([]uuid.UUID, error) {
	roleID, err := validateID(roleID, "role_id")
	if err != nil {
		return nil, err
	}
	if _, err := s.roles.Get(ctx, roleID); err != nil {
		return nil, err
	}
	return s.roles.MenuIDs(ctx, roleID)
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
	application, err = validateServiceResource(application)
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
	application, err = validateServiceResource(application)
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
	return validateServiceResource(application)
}

func validateID(value, name string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 80 {
		return "", fmt.Errorf("%w: %s must be 1-80 characters", ErrInvalidArgument, name)
	}
	return value, nil
}

func validateMenu(menu *Menu) error {
	code, name, err := validateCodeAndName(menu.Code, menu.Name)
	if err != nil {
		return err
	}
	menu.Code, menu.Name = code, name
	if menu.Type != MenuTypeMenu && menu.Type != MenuTypeButton {
		return fmt.Errorf("%w: type must be menu or button", ErrInvalidArgument)
	}
	if menu.Type == MenuTypeButton && menu.ParentID == nil {
		return fmt.Errorf("%w: button must have a menu parent", ErrInvalidArgument)
	}
	if menu.Type == MenuTypeMenu && menu.APIPath != "" {
		return fmt.Errorf("%w: menu cannot define api_path", ErrInvalidArgument)
	}
	if menu.Type == MenuTypeButton && strings.TrimSpace(menu.APIPath) == "" {
		return fmt.Errorf("%w: button api_path is required", ErrInvalidArgument)
	}
	return nil
}
