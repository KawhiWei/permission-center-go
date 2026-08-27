package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/luck/permission-center-go/internal/auth"
	"github.com/luck/permission-center-go/internal/biz"
	"github.com/luck/permission-center-go/internal/config"
)

type testRoleRepo struct{ values map[string]*biz.Role }

func (r *testRoleRepo) Create(_ context.Context, value *biz.Role) (*biz.Role, error) {
	value.ID = "role-test"
	r.values[value.ID] = value
	return value, nil
}
func (r *testRoleRepo) Get(_ context.Context, id string) (*biz.Role, error) {
	value, ok := r.values[id]
	if !ok {
		return nil, biz.ErrNotFound
	}
	return value, nil
}
func (*testRoleRepo) ListByApplication(context.Context, string) ([]*biz.Role, error) { return nil, nil }
func (*testRoleRepo) ReplaceResources(context.Context, string, []uuid.UUID) error    { return nil }
func (*testRoleRepo) ResourceIDs(context.Context, string) ([]uuid.UUID, error)       { return nil, nil }

type testResourceRepo struct{}

type testUserRoleRepo struct {
	subject string
	app     string
	roleIDs []string
}

func (r *testUserRoleRepo) ReplaceRoles(_ context.Context, subject, app string, roleIDs []string) error {
	r.subject, r.app, r.roleIDs = subject, app, append([]string(nil), roleIDs...)
	return nil
}
func (r *testUserRoleRepo) RoleIDs(context.Context, string, string) ([]string, error) {
	return nil, nil
}

func (*testResourceRepo) Create(context.Context, *biz.Resource) (*biz.Resource, error) {
	return nil, nil
}
func (*testResourceRepo) Get(context.Context, uuid.UUID) (*biz.Resource, error) {
	return nil, biz.ErrNotFound
}
func (*testResourceRepo) ListByApplication(context.Context, string) ([]*biz.Resource, error) {
	return nil, nil
}

func TestCreateRole(t *testing.T) {
	roles := &testRoleRepo{values: map[string]*biz.Role{}}
	server := NewServer(NewHandler(biz.NewPermissionService(roles, &testResourceRepo{})))
	req := httptest.NewRequest(http.MethodPost, "/v1/roles", strings.NewReader(`{"application":"admin","code":"operator","name":"Operator"}`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, req)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if len(roles.values) != 1 {
		t.Fatalf("role count = %d", len(roles.values))
	}
	for _, role := range roles.values {
		if role.CreatedByID != "system" || role.UpdatedByID != "system" {
			t.Fatalf("audit fields = %#v", role.BaseFields)
		}
	}
}

func TestRejectsUnknownJSONFields(t *testing.T) {
	roles := &testRoleRepo{values: map[string]*biz.Role{}}
	server := NewServer(NewHandler(biz.NewPermissionService(roles, &testResourceRepo{})))
	request := httptest.NewRequest(http.MethodPost, "/v1/roles", strings.NewReader(`{"application":"admin","code":"operator","name":"Operator","other":true}`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestReplaceUserRoles(t *testing.T) {
	roles := &testRoleRepo{values: map[string]*biz.Role{}}
	userRoles := &testUserRoleRepo{}
	server := NewServer(NewHandler(biz.NewPermissionService(roles, &testResourceRepo{}, userRoles)))
	request := httptest.NewRequest(http.MethodPut, "/v1/users/nexus-user-1/roles?application=admin", strings.NewReader(`{"role_ids":["operator"]}`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if userRoles.subject != "nexus-user-1" || userRoles.app != "admin" || len(userRoles.roleIDs) != 1 {
		t.Fatalf("assignment = %#v", userRoles)
	}
}

func TestDevelopmentModeInjectsAuditActor(t *testing.T) {
	roles := &testRoleRepo{values: map[string]*biz.Role{}}
	authenticator, err := auth.New(context.Background(), config.OIDCConfig{Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(NewHandler(biz.NewPermissionService(roles, &testResourceRepo{}), authenticator))
	request := httptest.NewRequest(http.MethodPost, "/v1/roles", strings.NewReader(`{"application":"admin","code":"operator","name":"Operator"}`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	for _, role := range roles.values {
		if role.CreatedByID != "development" || role.CreatedByName != "开发模式" {
			t.Fatalf("audit fields = %#v", role.BaseFields)
		}
	}
}
