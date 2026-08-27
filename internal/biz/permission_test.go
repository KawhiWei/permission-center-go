package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type memoryResourceRepo struct{ resources map[uuid.UUID]*Resource }

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

func (r *memoryResourceRepo) Create(_ context.Context, resource *Resource) (*Resource, error) {
	resource.ID = uuid.New()
	r.resources[resource.ID] = resource
	return resource, nil
}
func (r *memoryResourceRepo) Get(_ context.Context, id uuid.UUID) (*Resource, error) {
	value, ok := r.resources[id]
	if !ok {
		return nil, ErrNotFound
	}
	return value, nil
}
func (r *memoryResourceRepo) ListByApplication(_ context.Context, application string) ([]*Resource, error) {
	values := []*Resource{}
	for _, value := range r.resources {
		if value.Application == application {
			values = append(values, value)
		}
	}
	return values, nil
}

func TestCreateResourceRejectsButtonWithoutMenuParent(t *testing.T) {
	ctx := context.Background()
	resources := &memoryResourceRepo{resources: map[uuid.UUID]*Resource{}}
	service := NewPermissionService(nil, resources)
	if _, err := service.CreateResource(ctx, &Resource{Application: "ops", Code: "create", Name: "Create", Type: ResourceTypeButton, APIPath: "/v1/items"}); err == nil {
		t.Fatal("expected invalid root button")
	}
	menu, err := service.CreateResource(ctx, &Resource{Application: "ops", Code: "items", Name: "Items", Type: ResourceTypeMenu, Path: "/items"})
	if err != nil {
		t.Fatal(err)
	}
	button, err := service.CreateResource(ctx, &Resource{Application: "ops", ParentID: &menu.ID, Code: "items:create", Name: "Create", Type: ResourceTypeButton, APIPath: "/v1/items", HTTPMethod: "POST"})
	if err != nil {
		t.Fatal(err)
	}
	if button.ParentID == nil || *button.ParentID != menu.ID {
		t.Fatalf("button parent = %v", button.ParentID)
	}
}

func TestResourceTreeBuildsHierarchy(t *testing.T) {
	rootID, childID := uuid.New(), uuid.New()
	resources := &memoryResourceRepo{resources: map[uuid.UUID]*Resource{
		rootID:  {ID: rootID, Application: "admin", Type: ResourceTypeMenu, Sort: 1},
		childID: {ID: childID, Application: "admin", ParentID: &rootID, Type: ResourceTypeButton, Sort: 2},
	}}
	service := NewPermissionService(nil, resources)
	tree, err := service.ResourceTree(context.Background(), "admin")
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
