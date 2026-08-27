package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/luck/permission-center-go/internal/auth"
	"github.com/luck/permission-center-go/internal/biz"
)

type Handler struct {
	permissions *biz.PermissionService
	auth        *auth.Service
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
		Application string `json:"application"`
		Code        string `json:"code"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	role, err := h.permissions.CreateRole(h.auditContext(r), request.Application, request.Code, request.Name, request.Description)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, role)
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.permissions.ListRoles(r.Context(), r.URL.Query().Get("application"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": roles})
}

func (h *Handler) CreateResource(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Application string           `json:"application"`
		ParentID    *uuid.UUID       `json:"parent_id"`
		Code        string           `json:"code"`
		Name        string           `json:"name"`
		Description string           `json:"description"`
		Type        biz.ResourceType `json:"type"`
		Path        string           `json:"path"`
		Component   string           `json:"component"`
		APIPath     string           `json:"api_path"`
		HTTPMethod  string           `json:"http_method"`
		Icon        string           `json:"icon"`
		Sort        int              `json:"sort"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	resource, err := h.permissions.CreateResource(h.auditContext(r), &biz.Resource{Application: request.Application, ParentID: request.ParentID, Code: request.Code, Name: request.Name, Description: request.Description, Type: request.Type, Path: request.Path, Component: request.Component, APIPath: request.APIPath, HTTPMethod: request.HTTPMethod, Icon: request.Icon, Sort: request.Sort})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resource)
}

func (h *Handler) ResourceTree(w http.ResponseWriter, r *http.Request) {
	tree, err := h.permissions.ResourceTree(r.Context(), r.URL.Query().Get("application"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": tree})
}

func (h *Handler) GrantRoleResources(w http.ResponseWriter, r *http.Request) {
	roleID := r.PathValue("roleID")
	var request struct {
		ResourceIDs []uuid.UUID `json:"resource_ids"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	if err := h.permissions.GrantRoleResources(h.auditContext(r), roleID, request.ResourceIDs); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RoleResources(w http.ResponseWriter, r *http.Request) {
	ids, err := h.permissions.RoleResources(r.Context(), r.PathValue("roleID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"resource_ids": ids})
}

func (h *Handler) ReplaceUserRoles(w http.ResponseWriter, r *http.Request) {
	var request struct {
		RoleIDs []string `json:"role_ids"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	if err := h.permissions.ReplaceUserRoles(h.auditContext(r), r.PathValue("userID"), r.URL.Query().Get("application"), request.RoleIDs); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UserRoles(w http.ResponseWriter, r *http.Request) {
	roleIDs, err := h.permissions.UserRoleIDs(r.Context(), r.PathValue("userID"), r.URL.Query().Get("application"))
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
