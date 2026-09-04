package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/luck/permission-center-go/internal/auth"
	"github.com/luck/permission-center-go/internal/biz"
)

type Handler struct {
	permissions      *biz.PermissionService
	applications     *biz.ApplicationService
	serviceResources biz.ServiceResourceCatalog
	pdp              *biz.PDPService
	auth             *auth.Service
}

func (h *Handler) WithApplications(applications *biz.ApplicationService) *Handler {
	h.applications = applications
	return h
}

func (h *Handler) WithServiceResourceCatalog(catalog biz.ServiceResourceCatalog) *Handler {
	h.serviceResources = catalog
	return h
}

func (h *Handler) ListServiceResources(w http.ResponseWriter, r *http.Request) {
	if h.serviceResources == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	values, err := h.serviceResources.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (h *Handler) GetServiceResource(w http.ResponseWriter, r *http.Request) {
	if h.serviceResources == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	value, err := h.serviceResources.Get(r.Context(), r.PathValue("name"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *Handler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Application string `json:"application"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSON(r, &q); err != nil {
		writeError(w, err)
		return
	}
	if h.applications == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	value, err := h.applications.Create(h.auditContext(r), q.Application, q.Name, q.Description)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
func (h *Handler) ListApplications(w http.ResponseWriter, r *http.Request) {
	if h.applications == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	values, err := h.applications.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values})
}
func (h *Handler) GetApplication(w http.ResponseWriter, r *http.Request) {
	if h.applications == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	value, err := h.applications.Get(r.Context(), r.PathValue("application"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}
func (h *Handler) UpdateApplication(w http.ResponseWriter, r *http.Request) {
	var q struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
	}
	if err := decodeJSON(r, &q); err != nil {
		writeError(w, err)
		return
	}
	if h.applications == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	value, err := h.applications.Update(h.auditContext(r), r.PathValue("application"), q.Name, q.Description, q.Enabled)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}
func (h *Handler) DeleteApplication(w http.ResponseWriter, r *http.Request) {
	if h.applications == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	if err := h.applications.Delete(h.auditContext(r), r.PathValue("application")); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func NewHandler(permissions *biz.PermissionService, authenticators ...*auth.Service) *Handler {
	var authenticator *auth.Service
	if len(authenticators) > 0 {
		authenticator = authenticators[0]
	}
	return &Handler{permissions: permissions, auth: authenticator}
}

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ServiceResource string `json:"service_resource"`
		Application     string `json:"application"`
		Code            string `json:"code"`
		Name            string `json:"name"`
		Description     string `json:"description"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	role, err := h.permissions.CreateRole(h.auditContext(r), bizScope(request.ServiceResource, request.Application), request.Code, request.Name, request.Description)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, role)
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.permissions.ListRoles(r.Context(), queryServiceResource(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": roles})
}

func (h *Handler) CreateMenu(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ServiceResource string       `json:"service_resource"`
		Application     string       `json:"application"`
		ParentID        *uuid.UUID   `json:"parent_id"`
		Code            string       `json:"code"`
		Name            string       `json:"name"`
		Description     string       `json:"description"`
		Type            biz.MenuType `json:"type"`
		Path            string       `json:"path"`
		Component       string       `json:"component"`
		APIPath         string       `json:"api_path"`
		HTTPMethod      string       `json:"http_method"`
		Icon            string       `json:"icon"`
		Sort            int          `json:"sort"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	menu, err := h.permissions.CreateMenu(h.auditContext(r), &biz.Menu{ServiceResource: request.ServiceResource, Application: request.Application, ParentID: request.ParentID, Code: request.Code, Name: request.Name, Description: request.Description, Type: request.Type, Path: request.Path, Component: request.Component, APIPath: request.APIPath, HTTPMethod: request.HTTPMethod, Icon: request.Icon, Sort: request.Sort})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, menu)
}

func (h *Handler) MenuTree(w http.ResponseWriter, r *http.Request) {
	tree, err := h.permissions.MenuTree(r.Context(), queryServiceResource(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": tree})
}

func (h *Handler) GrantRoleMenus(w http.ResponseWriter, r *http.Request) {
	roleID := r.PathValue("roleID")
	var request struct {
		MenuIDs []uuid.UUID `json:"menu_ids"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	if err := h.permissions.GrantRoleMenus(h.auditContext(r), roleID, request.MenuIDs); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *Handler) RoleMenus(w http.ResponseWriter, r *http.Request) {
	ids, err := h.permissions.RoleMenus(r.Context(), r.PathValue("roleID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"menu_ids": ids})
}

func (h *Handler) ReplaceUserRoles(w http.ResponseWriter, r *http.Request) {
	var request struct {
		RoleIDs []string `json:"role_ids"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	if err := h.permissions.ReplaceUserRoles(h.auditContext(r), r.PathValue("userID"), queryServiceResource(r), request.RoleIDs); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *Handler) UserRoles(w http.ResponseWriter, r *http.Request) {
	roleIDs, err := h.permissions.UserRoleIDs(r.Context(), r.PathValue("userID"), queryServiceResource(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"role_ids": roleIDs})
}

func (h *Handler) auditContext(r *http.Request) context.Context {
	actor := biz.AuditActor{ID: "system", Name: "system"}
	if h.auth != nil {
		if h.auth.Enabled() {
			if user, ok := h.auth.Me(r); ok {
				actor = biz.AuditActor{ID: user.Subject, Name: user.Name}
			}
		} else {
			actor = biz.AuditActor{ID: "development", Name: "开发模式"}
		}
	}
	return biz.WithAuditActor(r.Context(), actor)
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: invalid JSON body", biz.ErrInvalidArgument)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("%w: request body must contain one JSON value", biz.ErrInvalidArgument)
	}
	return nil
}

func queryServiceResource(r *http.Request) string {
	if value := strings.TrimSpace(r.URL.Query().Get("service_resource")); value != "" {
		return value
	}
	return strings.TrimSpace(r.URL.Query().Get("application"))
}

func bizScope(serviceResource, application string) string {
	if value := strings.TrimSpace(serviceResource); value != "" {
		return value
	}
	return strings.TrimSpace(application)
}
