package httpserver

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/luck/permission-center-go/internal/biz"
	"github.com/luck/permission-center-go/internal/policy/model"
)

// This package mirrors the production HTTP package for black-box route tests.
func (h *Handler) WithAuthorizationPolicyService(service *biz.AuthorizationPolicyService) *Handler {
	h.policies = service
	return h
}

type policyRequest struct {
	ServiceResource   string                  `json:"service_resource"`
	Code              string                  `json:"code"`
	Name              string                  `json:"name"`
	Description       string                  `json:"description"`
	Effect            model.Effect            `json:"effect"`
	AuthorizationType model.AuthorizationType `json:"authorization_type"`
	Priority          int                     `json:"priority"`
	Condition         *model.Condition        `json:"condition"`
	RoleIDs           []string                `json:"role_ids"`
	EndpointIDs       []uuid.UUID             `json:"endpoint_ids"`
}

func (r policyRequest) value(id uuid.UUID) *biz.AuthorizationPolicy {
	return &biz.AuthorizationPolicy{ID: id, ServiceResource: r.ServiceResource, Code: r.Code, Name: r.Name, Description: r.Description, Effect: r.Effect, AuthorizationType: r.AuthorizationType, Priority: r.Priority, Condition: r.Condition, RoleIDs: r.RoleIDs, EndpointIDs: r.EndpointIDs}
}
func (h *Handler) CreateAuthorizationPolicy(w http.ResponseWriter, r *http.Request) {
	if h.policies == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	var request policyRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	value, err := h.policies.Create(h.auditContext(r), request.value(uuid.Nil))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
func (h *Handler) ListAuthorizationPolicies(w http.ResponseWriter, r *http.Request) {
	if h.policies == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	values, err := h.policies.List(r.Context(), queryServiceResource(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values})
}
func (h *Handler) GetAuthorizationPolicy(w http.ResponseWriter, r *http.Request) {
	if h.policies == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePolicyUUID(r.PathValue("id"))
	if err == nil {
		var value *biz.AuthorizationPolicy
		value, err = h.policies.Get(r.Context(), id)
		if err == nil {
			writeJSON(w, http.StatusOK, value)
			return
		}
	}
	writeError(w, err)
}
func (h *Handler) UpdateAuthorizationPolicy(w http.ResponseWriter, r *http.Request) {
	if h.policies == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePolicyUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	var request policyRequest
	if err = decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	value, err := h.policies.Update(h.auditContext(r), request.value(id))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}
func (h *Handler) DeleteAuthorizationPolicy(w http.ResponseWriter, r *http.Request) {
	if h.policies == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePolicyUUID(r.PathValue("id"))
	if err == nil {
		err = h.policies.Delete(h.auditContext(r), id)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}
func (h *Handler) PublishAuthorizationPolicy(w http.ResponseWriter, r *http.Request) {
	if h.policies == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePolicyUUID(r.PathValue("id"))
	if err == nil {
		var value *biz.AuthorizationPolicy
		value, err = h.policies.Publish(h.auditContext(r), id)
		if err == nil {
			writeJSON(w, http.StatusOK, value)
			return
		}
	}
	writeError(w, err)
}
func (h *Handler) RollbackAuthorizationPolicy(w http.ResponseWriter, r *http.Request) {
	if h.policies == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePolicyUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	var request struct {
		Version int `json:"version"`
	}
	if err = decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	value, err := h.policies.Rollback(h.auditContext(r), id, request.Version)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}
func (h *Handler) DecideAuthorizationPolicy(w http.ResponseWriter, r *http.Request) {
	credential := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if h.pdpServiceCredential == "" || subtle.ConstantTimeCompare([]byte(credential), []byte(h.pdpServiceCredential)) != 1 {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	h.decideAuthorizationPolicy(w, r, true)
}
func (h *Handler) SimulateAuthorizationPolicy(w http.ResponseWriter, r *http.Request) {
	h.decideAuthorizationPolicy(w, r, false)
}
func (h *Handler) decideAuthorizationPolicy(w http.ResponseWriter, r *http.Request, rawResponse bool) {
	if h.policies == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	var request model.Input
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.policies.Decide(r.Context(), request)
	if err != nil {
		writeError(w, err)
		return
	}
	if rawResponse {
		writeRawJSON(w, http.StatusOK, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (h *Handler) ListPDPDecisionLogs(w http.ResponseWriter, r *http.Request) {
	if h.policies == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	var id uuid.UUID
	var err error
	if value := strings.TrimSpace(r.URL.Query().Get("endpoint_id")); value != "" {
		id, err = parsePolicyUUID(value)
	}
	if err == nil {
		var values []biz.DecisionLog
		values, err = h.policies.DecisionLogs(r.Context(), queryServiceResource(r), id)
		if err == nil {
			writeJSON(w, http.StatusOK, map[string]any{"items": values})
			return
		}
	}
	writeError(w, err)
}
func parsePolicyUUID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, fmt.Errorf("%w: policy id must be a UUID", biz.ErrInvalidArgument)
	}
	return id, nil
}
