package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/luck/permission-center-go/internal/auth"
	"github.com/luck/permission-center-go/internal/biz"
	"github.com/luck/permission-center-go/internal/config"
)

func TestSwaggerUIAndOpenAPIDocument(t *testing.T) {
	server := NewServer(NewHandler(nil))

	redirect := httptest.NewRecorder()
	server.ServeHTTP(redirect, httptest.NewRequest(http.MethodGet, "/swagger", nil))
	if redirect.Code != http.StatusMovedPermanently || redirect.Header().Get("Location") != "/swagger/index.html" {
		t.Fatalf("swagger redirect = %d location=%q", redirect.Code, redirect.Header().Get("Location"))
	}

	ui := httptest.NewRecorder()
	server.ServeHTTP(ui, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	if ui.Code != http.StatusOK || !strings.Contains(ui.Body.String(), "Swagger UI") {
		t.Fatalf("swagger UI = %d body=%s", ui.Code, ui.Body.String())
	}

	documentResponse := httptest.NewRecorder()
	server.ServeHTTP(documentResponse, httptest.NewRequest(http.MethodGet, "/swagger/openapi.json", nil))
	if documentResponse.Code != http.StatusOK {
		t.Fatalf("openapi document = %d body=%s", documentResponse.Code, documentResponse.Body.String())
	}
	var document map[string]any
	if err := json.Unmarshal(documentResponse.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode OpenAPI document: %v", err)
	}
	if document["openapi"] != "3.0.3" {
		t.Fatalf("openapi version = %#v", document["openapi"])
	}
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI paths missing from document: %#v", document["paths"])
	}
	for path := range paths {
		if path == "/v1/authorization/decisions" || path == "/v1/pdp/decisions" || path == "/v1/applications" {
			t.Fatalf("legacy path present in OpenAPI document: %s", path)
		}
	}
	for _, operation := range []struct {
		path   string
		method string
	}{
		{path: "/v1/roles/{roleID}", method: "put"},
		{path: "/v1/roles/{roleID}", method: "delete"},
		{path: "/v1/menus/{id}", method: "put"},
		{path: "/v1/menus/{id}", method: "delete"},
		{path: "/v1/authorization/api-endpoints", method: "post"},
		{path: "/v1/authorization/api-endpoints", method: "get"},
		{path: "/v1/authorization/api-endpoints/import-swagger", method: "post"},
		{path: "/v1/authorization/api-endpoints/{id}", method: "get"},
		{path: "/v1/authorization/api-endpoints/{id}", method: "put"},
		{path: "/v1/authorization/api-endpoints/{id}", method: "delete"},
	} {
		pathItem, ok := paths[operation.path].(map[string]any)
		if !ok || pathItem[operation.method] == nil {
			t.Fatalf("OpenAPI operation missing: %s %s", strings.ToUpper(operation.method), operation.path)
		}
	}
}

type testRoleRepo struct {
	values  map[string]*biz.Role
	menuIDs []uuid.UUID
}

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
func (r *testRoleRepo) Update(_ context.Context, value *biz.Role) (*biz.Role, error) {
	if _, ok := r.values[value.ID]; !ok {
		return nil, biz.ErrNotFound
	}
	r.values[value.ID] = value
	return value, nil
}
func (r *testRoleRepo) SoftDelete(_ context.Context, id string) error {
	value, ok := r.values[id]
	if !ok {
		return biz.ErrNotFound
	}
	value.IsDeleted = true
	value.Enabled = false
	return nil
}
func (*testRoleRepo) ListByServiceResource(context.Context, string) ([]*biz.Role, error) {
	return nil, nil
}
func (r *testRoleRepo) ReplaceMenus(_ context.Context, _ string, menuIDs []uuid.UUID) error {
	r.menuIDs = append([]uuid.UUID(nil), menuIDs...)
	return nil
}
func (r *testRoleRepo) MenuIDs(context.Context, string) ([]uuid.UUID, error) {
	return append([]uuid.UUID(nil), r.menuIDs...), nil
}

type testMenuRepo struct {
	values map[uuid.UUID]*biz.Menu
}

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
	return append([]string(nil), r.roleIDs...), nil
}

func (r *testMenuRepo) Create(_ context.Context, menu *biz.Menu) (*biz.Menu, error) {
	if r.values == nil {
		r.values = map[uuid.UUID]*biz.Menu{}
	}
	if menu.ID == uuid.Nil {
		menu.ID = uuid.New()
	}
	r.values[menu.ID] = menu
	return menu, nil
}
func (r *testMenuRepo) Get(_ context.Context, id uuid.UUID) (*biz.Menu, error) {
	menu, ok := r.values[id]
	if !ok {
		return nil, biz.ErrNotFound
	}
	return menu, nil
}
func (r *testMenuRepo) Update(_ context.Context, menu *biz.Menu) (*biz.Menu, error) {
	if _, ok := r.values[menu.ID]; !ok {
		return nil, biz.ErrNotFound
	}
	r.values[menu.ID] = menu
	return menu, nil
}
func (r *testMenuRepo) SoftDelete(_ context.Context, id uuid.UUID) error {
	menu, ok := r.values[id]
	if !ok {
		return biz.ErrNotFound
	}
	menu.IsDeleted = true
	menu.Enabled = false
	return nil
}
func (r *testMenuRepo) ListByServiceResource(_ context.Context, serviceResource string) ([]*biz.Menu, error) {
	menus := make([]*biz.Menu, 0)
	for _, menu := range r.values {
		if menu.ServiceResource == serviceResource {
			menus = append(menus, menu)
		}
	}
	return menus, nil
}

func TestCreateRole(t *testing.T) {
	roles := &testRoleRepo{values: map[string]*biz.Role{}}
	server := NewServer(NewHandler(biz.NewPermissionService(roles, &testMenuRepo{})))
	req := httptest.NewRequest(http.MethodPost, "/v1/roles", strings.NewReader(`{"service_resource":"admin","code":"operator","name":"Operator"}`))
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
	server := NewServer(NewHandler(biz.NewPermissionService(roles, &testMenuRepo{})))
	request := httptest.NewRequest(http.MethodPost, "/v1/roles", strings.NewReader(`{"service_resource":"admin","code":"operator","name":"Operator","other":true}`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestMenuEndpointsUseMenuNaming(t *testing.T) {
	menuID := uuid.New()
	roles := &testRoleRepo{values: map[string]*biz.Role{
		"role-1": {ID: "role-1", ServiceResource: "admin", Enabled: true},
	}}
	menus := &testMenuRepo{values: map[uuid.UUID]*biz.Menu{
		menuID: {ID: menuID, ServiceResource: "admin", Code: "items", Name: "Items", Type: biz.MenuTypeMenu, Enabled: true},
	}}
	server := NewServer(NewHandler(biz.NewPermissionService(roles, menus)))

	createRequest := httptest.NewRequest(http.MethodPost, "/v1/menus", strings.NewReader(`{"service_resource":"admin","code":"settings","name":"Settings","type":"menu","path":"/settings"}`))
	createResponse := httptest.NewRecorder()
	server.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create menu status = %d body=%s", createResponse.Code, createResponse.Body.String())
	}

	grantRequest := httptest.NewRequest(http.MethodPut, "/v1/roles/role-1/menus", strings.NewReader(`{"menu_ids":["`+menuID.String()+`"]}`))
	grantResponse := httptest.NewRecorder()
	server.ServeHTTP(grantResponse, grantRequest)
	if grantResponse.Code != http.StatusOK {
		t.Fatalf("grant menus status = %d body=%s", grantResponse.Code, grantResponse.Body.String())
	}
	if len(roles.menuIDs) != 1 || roles.menuIDs[0] != menuID {
		t.Fatalf("granted menu IDs = %#v", roles.menuIDs)
	}

	roleMenusRequest := httptest.NewRequest(http.MethodGet, "/v1/roles/role-1/menus", nil)
	roleMenusResponse := httptest.NewRecorder()
	server.ServeHTTP(roleMenusResponse, roleMenusRequest)
	if roleMenusResponse.Code != http.StatusOK {
		t.Fatalf("role menus status = %d body=%s", roleMenusResponse.Code, roleMenusResponse.Body.String())
	}
	if !strings.Contains(roleMenusResponse.Body.String(), `"menu_ids"`) || strings.Contains(roleMenusResponse.Body.String(), `"resource_ids"`) {
		t.Fatalf("role menus body = %s", roleMenusResponse.Body.String())
	}

	treeRequest := httptest.NewRequest(http.MethodGet, "/v1/menus/tree?service_resource=admin", nil)
	treeResponse := httptest.NewRecorder()
	server.ServeHTTP(treeResponse, treeRequest)
	if treeResponse.Code != http.StatusOK {
		t.Fatalf("menu tree status = %d body=%s", treeResponse.Code, treeResponse.Body.String())
	}
	if !strings.Contains(treeResponse.Body.String(), `"type":"menu"`) ||
		!strings.Contains(treeResponse.Body.String(), `"children"`) ||
		strings.Contains(treeResponse.Body.String(), `"Type"`) ||
		strings.Contains(treeResponse.Body.String(), `"Children"`) {
		t.Fatalf("menu tree body = %s", treeResponse.Body.String())
	}

	legacyRequest := httptest.NewRequest(http.MethodGet, "/v1/resources/tree?service_resource=admin", nil)
	legacyResponse := httptest.NewRecorder()
	server.ServeHTTP(legacyResponse, legacyRequest)
	if legacyResponse.Code != http.StatusNotFound {
		t.Fatalf("legacy resource route status = %d", legacyResponse.Code)
	}
}

func TestReplaceUserRoles(t *testing.T) {
	roles := &testRoleRepo{values: map[string]*biz.Role{}}
	userRoles := &testUserRoleRepo{}
	server := NewServer(NewHandler(biz.NewPermissionService(roles, &testMenuRepo{}, userRoles)))
	request := httptest.NewRequest(http.MethodPut, "/v1/users/nexus-user-1/roles?service_resource=admin", strings.NewReader(`{"role_ids":["operator"]}`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if userRoles.subject != "nexus-user-1" || userRoles.app != "admin" || len(userRoles.roleIDs) != 1 {
		t.Fatalf("assignment = %#v", userRoles)
	}
}

func TestRoleUpdateAndDeleteEndpoints(t *testing.T) {
	roles := &testRoleRepo{values: map[string]*biz.Role{
		"role-1": {ID: "role-1", ServiceResource: "admin", Code: "old", Name: "Old", Enabled: true},
	}}
	server := NewServer(NewHandler(biz.NewPermissionService(roles, &testMenuRepo{})))

	updateRequest := httptest.NewRequest(http.MethodPut, "/v1/roles/role-1", strings.NewReader(`{"code":"operator","name":"Operator","description":"Operators","enabled":true}`))
	updateResponse := httptest.NewRecorder()
	server.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update role status = %d body=%s", updateResponse.Code, updateResponse.Body.String())
	}
	role := roles.values["role-1"]
	if role.Code != "operator" || role.Name != "Operator" || role.Description != "Operators" || !role.Enabled {
		t.Fatalf("updated role = %#v", role)
	}

	deleteResponse := httptest.NewRecorder()
	server.ServeHTTP(deleteResponse, httptest.NewRequest(http.MethodDelete, "/v1/roles/role-1", nil))
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete role status = %d body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}
	if !role.IsDeleted || role.Enabled {
		t.Fatalf("deleted role = %#v", role)
	}
}

func TestMenuUpdateAndDeleteEndpoints(t *testing.T) {
	rootID, menuID := uuid.New(), uuid.New()
	root := &biz.Menu{ID: rootID, ServiceResource: "admin", Code: "items", Name: "Items", Type: biz.MenuTypeMenu, Enabled: true}
	menu := &biz.Menu{ID: menuID, ServiceResource: "admin", ParentID: &rootID, Code: "create", Name: "Create", Type: biz.MenuTypeButton, APIPath: "/v1/items", HTTPMethod: http.MethodPost, Enabled: true}
	menus := &testMenuRepo{values: map[uuid.UUID]*biz.Menu{rootID: root, menuID: menu}}
	server := NewServer(NewHandler(biz.NewPermissionService(&testRoleRepo{values: map[string]*biz.Role{}}, menus)))

	updateRequest := httptest.NewRequest(http.MethodPut, "/v1/menus/"+menuID.String(), strings.NewReader(`{"code":"update","name":"Update","description":"Update items","path":"/items/update","component":"UpdatePage","api_path":"/v1/items/update","http_method":"PATCH","icon":"edit","sort":8,"enabled":true}`))
	updateResponse := httptest.NewRecorder()
	server.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update menu status = %d body=%s", updateResponse.Code, updateResponse.Body.String())
	}
	updatedMenu := menus.values[menuID]
	if updatedMenu.Code != "update" || updatedMenu.Name != "Update" || updatedMenu.Description != "Update items" || updatedMenu.Path != "/items/update" ||
		updatedMenu.Component != "UpdatePage" || updatedMenu.APIPath != "/v1/items/update" || updatedMenu.HTTPMethod != "PATCH" || updatedMenu.Icon != "edit" || updatedMenu.Sort != 8 || !updatedMenu.Enabled {
		t.Fatalf("updated menu = %#v", updatedMenu)
	}

	deleteResponse := httptest.NewRecorder()
	server.ServeHTTP(deleteResponse, httptest.NewRequest(http.MethodDelete, "/v1/menus/"+menuID.String(), nil))
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("delete menu status = %d body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}
	if !updatedMenu.IsDeleted || updatedMenu.Enabled {
		t.Fatalf("deleted menu = %#v", updatedMenu)
	}

	invalidResponse := httptest.NewRecorder()
	server.ServeHTTP(invalidResponse, httptest.NewRequest(http.MethodDelete, "/v1/menus/not-a-uuid", nil))
	if invalidResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid menu ID status = %d body=%s", invalidResponse.Code, invalidResponse.Body.String())
	}
}

func TestLegacyRoutesAreNotFound(t *testing.T) {
	server := NewServer(NewHandler(nil))
	for _, legacyPath := range []string{
		"/v1/authorization/decisions",
		"/v1/pdp/decisions",
		"/v1/applications",
	} {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, legacyPath, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("legacy route %s status = %d", legacyPath, response.Code)
		}
	}
}

func TestDevelopmentModeInjectsAuditActor(t *testing.T) {
	roles := &testRoleRepo{values: map[string]*biz.Role{}}
	authenticator, err := auth.New(context.Background(), config.OIDCConfig{Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(NewHandler(biz.NewPermissionService(roles, &testMenuRepo{}), authenticator))
	request := httptest.NewRequest(http.MethodPost, "/v1/roles", strings.NewReader(`{"service_resource":"admin","code":"operator","name":"Operator"}`))
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
