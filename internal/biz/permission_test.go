package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type memoryMenuRepo struct{ menus map[uuid.UUID]*Menu }

type memoryUserRoleRepo struct {
	subject     string
	application string
	roleIDs     []string
}

func (r *memoryUserRoleRepo) ReplaceRoles(_ context.Context, subject, application string, roleIDs []string) error {
	r.subject, r.application = subject, application
	r.roleIDs = append([]string(nil), roleIDs...)
	return nil
}

func (r *memoryUserRoleRepo) RoleIDs(_ context.Context, _, _ string) ([]string, error) {
	return append([]string(nil), r.roleIDs...), nil
}

func (r *memoryMenuRepo) Create(_ context.Context, menu *Menu) (*Menu, error) {
	menu.ID = uuid.New()
	r.menus[menu.ID] = menu
	return menu, nil
}
func (r *memoryMenuRepo) Get(_ context.Context, id uuid.UUID) (*Menu, error) {
	value, ok := r.menus[id]
	if !ok {
		return nil, ErrNotFound
	}
	return value, nil
}
func (r *memoryMenuRepo) ListByApplication(_ context.Context, application string) ([]*Menu, error) {
	values := []*Menu{}
	for _, value := range r.menus {
		if value.Application == application {
			values = append(values, value)
		}
	}
	return values, nil
}

func TestCreateMenuRejectsButtonWithoutMenuParent(t *testing.T) {
	ctx := context.Background()
	menus := &memoryMenuRepo{menus: map[uuid.UUID]*Menu{}}
	service := NewPermissionService(nil, menus)
	if _, err := service.CreateMenu(ctx, &Menu{Application: "ops", Code: "create", Name: "Create", Type: MenuTypeButton, APIPath: "/v1/items"}); err == nil {
		t.Fatal("expected invalid root button")
	}
	menu, err := service.CreateMenu(ctx, &Menu{Application: "ops", Code: "items", Name: "Items", Type: MenuTypeMenu, Path: "/items"})
	if err != nil {
		t.Fatal(err)
	}
	button, err := service.CreateMenu(ctx, &Menu{Application: "ops", ParentID: &menu.ID, Code: "items:create", Name: "Create", Type: MenuTypeButton, APIPath: "/v1/items", HTTPMethod: "POST"})
	if err != nil {
		t.Fatal(err)
	}
	if button.ParentID == nil || *button.ParentID != menu.ID {
		t.Fatalf("button parent = %v", button.ParentID)
	}
}

func TestMenuTreeBuildsHierarchy(t *testing.T) {
	rootID, childID := uuid.New(), uuid.New()
	menus := &memoryMenuRepo{menus: map[uuid.UUID]*Menu{
		rootID:  {ID: rootID, Application: "admin", Type: MenuTypeMenu, Sort: 1},
		childID: {ID: childID, Application: "admin", ParentID: &rootID, Type: MenuTypeButton, Sort: 2},
	}}
	service := NewPermissionService(nil, menus)
	tree, err := service.MenuTree(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) != 1 || len(tree[0].Children) != 1 || tree[0].Children[0].ID != childID {
		t.Fatalf("tree = %#v", tree)
	}
}

func TestReplaceUserRolesValidatesAndPreservesApplicationScope(t *testing.T) {
	userRoles := &memoryUserRoleRepo{}
	service := NewPermissionService(nil, nil, userRoles)
	if err := service.ReplaceUserRoles(context.Background(), " nexus-user-1 ", "admin", []string{"operator", "auditor"}); err != nil {
		t.Fatal(err)
	}
	if userRoles.subject != "nexus-user-1" || userRoles.application != "admin" {
		t.Fatalf("stored scope = %q / %q", userRoles.subject, userRoles.application)
	}
	if err := service.ReplaceUserRoles(context.Background(), "nexus-user-1", "admin", []string{"operator", "operator"}); err == nil {
		t.Fatal("expected duplicate role validation error")
	}
}

func TestAuditActorDefaultsAndNormalization(t *testing.T) {
	if actor := AuditActorFromContext(context.Background()); actor.ID != "system" || actor.Name != "system" {
		t.Fatalf("default actor = %#v", actor)
	}
	ctx := WithAuditActor(context.Background(), AuditActor{ID: " user-1 ", Name: " Alice "})
	actor := AuditActorFromContext(ctx)
	if actor.ID != "user-1" || actor.Name != "Alice" {
		t.Fatalf("normalized actor = %#v", actor)
	}
	base := NewBaseFields(actor)
	if base.CreatedByID != actor.ID || base.UpdatedByID != actor.ID || base.IsDeleted {
		t.Fatalf("base fields = %#v", base)
	}
}
