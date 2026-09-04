package httpserver

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/luck/permission-center-go/internal/biz"
)

func (h *Handler) WithPDP(pdp *biz.PDPService) *Handler {
	h.pdp = pdp
	return h
}

func (h *Handler) CreateAuthorizationResource(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	var request authorizationResourceRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	value, err := h.pdp.CreateResource(h.auditContext(r), request.resource())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (h *Handler) ListAuthorizationResources(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	values, err := h.pdp.ListResources(r.Context(), r.URL.Query().Get("application"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (h *Handler) GetAuthorizationResource(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "resource_id")
	if err != nil {
		writeError(w, err)
		return
	}
	value, err := h.pdp.GetResource(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateAuthorizationResource(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "resource_id")
	if err != nil {
		writeError(w, err)
		return
	}
	var request authorizationResourceRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	existing, err := h.pdp.GetResource(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	value := request.resource()
	value.ID = id
	if request.Application == "" {
		value.Application = existing.Application
	}
	if request.Type == "" && request.ResourceType == "" {
		value.Type = existing.Type
	}
	if request.Enabled == nil {
		value.Enabled = existing.Enabled
	}
	value, err = h.pdp.UpdateResource(h.auditContext(r), value)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteAuthorizationResource(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "resource_id")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.pdp.DeleteResource(h.auditContext(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *Handler) CreateAuthorizationAction(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	var request authorizationActionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	value, err := h.pdp.CreateAction(h.auditContext(r), request.action())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (h *Handler) ListAuthorizationActions(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	values, err := h.pdp.ListActions(r.Context(), r.URL.Query().Get("application"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (h *Handler) GetAuthorizationAction(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "action_id")
	if err != nil {
		writeError(w, err)
		return
	}
	value, err := h.pdp.GetAction(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateAuthorizationAction(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "action_id")
	if err != nil {
		writeError(w, err)
		return
	}
	var request authorizationActionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	existing, err := h.pdp.GetAction(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	value := request.action()
	value.ID = id
	if request.Application == "" {
		value.Application = existing.Application
	}
	if request.Enabled == nil {
		value.Enabled = existing.Enabled
	}
	if value.Code == "" {
		value.Code = existing.Code
	}
	if value.Name == "" {
		value.Name = existing.Name
	}
	value, err = h.pdp.UpdateAction(h.auditContext(r), value)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteAuthorizationAction(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "action_id")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.pdp.DeleteAction(h.auditContext(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *Handler) CreateAuthorizationAPIEndpoint(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	var request authorizationAPIEndpointRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	value, err := request.endpoint(uuid.Nil)
	if err == nil {
		value, err = h.pdp.CreateAPIEndpoint(h.auditContext(r), value)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (h *Handler) ListAuthorizationAPIEndpoints(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	values, err := h.pdp.ListAPIEndpoints(r.Context(), r.URL.Query().Get("application"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (h *Handler) GetAuthorizationAPIEndpoint(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "endpoint_id")
	if err != nil {
		writeError(w, err)
		return
	}
	value, err := h.pdp.GetAPIEndpoint(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateAuthorizationAPIEndpoint(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "endpoint_id")
	if err != nil {
		writeError(w, err)
		return
	}
	var request authorizationAPIEndpointRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	existing, err := h.pdp.GetAPIEndpoint(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	value, err := request.endpoint(id)
	if err != nil {
		writeError(w, err)
		return
	}
	if request.Application == "" {
		value.Application = existing.Application
	}
	if request.ResourceID == "" {
		value.ResourceID = existing.ResourceID
	}
	if request.ActionID == "" {
		value.ActionID = existing.ActionID
	}
	if request.EnforcementMode == "" {
		value.EnforcementMode = existing.EnforcementMode
	}
	if request.Enabled == nil {
		value.Enabled = existing.Enabled
	}
	if value.ServiceCode == "" {
		value.ServiceCode = existing.ServiceCode
	}
	if value.Method == "" {
		value.Method = existing.Method
	}
	if value.PathTemplate == "" {
		value.PathTemplate = existing.PathTemplate
	}
	value, err = h.pdp.UpdateAPIEndpoint(h.auditContext(r), value)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteAuthorizationAPIEndpoint(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "endpoint_id")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.pdp.DeleteAPIEndpoint(h.auditContext(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *Handler) CreateAuthorizationPolicy(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	var request authorizationPolicyRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	value, err := h.pdp.CreatePolicy(h.auditContext(r), request.policy())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (h *Handler) ListAuthorizationPolicies(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	values, err := h.pdp.ListPolicies(r.Context(), r.URL.Query().Get("application"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (h *Handler) GetAuthorizationPolicy(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "policy_id")
	if err != nil {
		writeError(w, err)
		return
	}
	value, err := h.pdp.GetPolicy(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *Handler) UpdateAuthorizationPolicy(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "policy_id")
	if err != nil {
		writeError(w, err)
		return
	}
	var request authorizationPolicyRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	existing, err := h.pdp.GetPolicy(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	value := request.policy()
	value.ID = id
	if request.Application == "" {
		value.Application = existing.Application
	}
	if value.Code == "" {
		value.Code = existing.Code
	}
	if value.Name == "" {
		value.Name = existing.Name
	}
	if value.Effect == "" {
		value.Effect = existing.Effect
	}
	if request.Priority == nil {
		value.Priority = existing.Priority
	}
	if request.ResourceCodes == nil && request.ResourceSelector == nil {
		value.ResourceCodes = existing.ResourceCodes
	}
	if request.ActionCodes == nil && request.ActionSelector == nil {
		value.ActionCodes = existing.ActionCodes
	}
	if request.Enabled == nil {
		value.Enabled = existing.Enabled
	}
	value, err = h.pdp.UpdatePolicy(h.auditContext(r), value)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteAuthorizationPolicy(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "policy_id")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.pdp.DeletePolicy(h.auditContext(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *Handler) GetAuthorizationPolicyBindings(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "policy_id")
	if err != nil {
		writeError(w, err)
		return
	}
	values, err := h.pdp.PolicyBindings(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (h *Handler) ReplaceAuthorizationPolicyBindings(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	id, err := parsePDPUUID(r.PathValue("id"), "policy_id")
	if err != nil {
		writeError(w, err)
		return
	}
	var request authorizationPolicyBindingsRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	values := request.values()
	if err := h.pdp.ReplacePolicyBindings(h.auditContext(r), id, values); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

func (h *Handler) DecidePDP(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	var payload authorizationDecisionRequest
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, err)
		return
	}
	decision, err := h.pdp.Decide(r.Context(), payload.normalize())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, decision)
}

func (h *Handler) SimulateAuthorizationPolicy(w http.ResponseWriter, r *http.Request) {
	if h.pdp == nil {
		writeError(w, biz.ErrNotFound)
		return
	}
	idText := r.PathValue("id")
	if strings.HasSuffix(idText, ":simulate") {
		idText = strings.TrimSuffix(idText, ":simulate")
	}
	id, err := parsePDPUUID(idText, "policy_id")
	if err != nil {
		writeError(w, err)
		return
	}
	if _, err := h.pdp.GetPolicy(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	var payload authorizationDecisionRequest
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, err)
		return
	}
	decision, err := h.pdp.Simulate(r.Context(), payload.normalize())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, decision)
}

type authorizationResourceRequest struct {
	Application  string           `json:"application"`
	Code         string           `json:"code"`
	Type         biz.ResourceType `json:"type"`
	ResourceType biz.ResourceType `json:"resource_type"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Matcher      string           `json:"matcher"`
	Enabled      *bool            `json:"enabled"`
}

func (r authorizationResourceRequest) resource() *biz.AuthorizationResource {
	typeValue := r.Type
	if typeValue == "" {
		typeValue = r.ResourceType
	}
	enabled := true
	if r.Enabled != nil {
		enabled = *r.Enabled
	}
	return &biz.AuthorizationResource{Application: r.Application, Code: r.Code, Type: typeValue, Name: r.Name, Description: r.Description, Matcher: r.Matcher, Enabled: enabled}
}

type authorizationActionRequest struct {
	Application string `json:"application"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"`
}

func (r authorizationActionRequest) action() *biz.AuthorizationAction {
	enabled := true
	if r.Enabled != nil {
		enabled = *r.Enabled
	}
	return &biz.AuthorizationAction{Application: r.Application, Code: r.Code, Name: r.Name, Description: r.Description, Enabled: enabled}
}

type authorizationAPIEndpointRequest struct {
	Application     string              `json:"application"`
	ServiceCode     string              `json:"service_code"`
	Method          string              `json:"method"`
	PathTemplate    string              `json:"path_template"`
	ResourceID      string              `json:"resource_id"`
	ActionID        string              `json:"action_id"`
	EnforcementMode biz.EnforcementMode `json:"enforcement_mode"`
	Enabled         *bool               `json:"enabled"`
}

func (r authorizationAPIEndpointRequest) endpoint(id uuid.UUID) (*biz.AuthorizationAPIEndpoint, error) {
	resourceID, err := parsePDPUUIDOptional(r.ResourceID, "resource_id")
	if err != nil {
		return nil, err
	}
	actionID, err := parsePDPUUIDOptional(r.ActionID, "action_id")
	if err != nil {
		return nil, err
	}
	enabled := true
	if r.Enabled != nil {
		enabled = *r.Enabled
	}
	return &biz.AuthorizationAPIEndpoint{ID: id, Application: r.Application, ServiceCode: r.ServiceCode, Method: r.Method, PathTemplate: r.PathTemplate, ResourceID: resourceID, ActionID: actionID, EnforcementMode: r.EnforcementMode, Enabled: enabled}, nil
}

type authorizationPolicyRequest struct {
	Application      string           `json:"application"`
	Code             string           `json:"code"`
	Name             string           `json:"name"`
	Description      string           `json:"description"`
	Effect           biz.PolicyEffect `json:"effect"`
	Priority         *int             `json:"priority"`
	ResourceCodes    []string         `json:"resource_codes"`
	ActionCodes      []string         `json:"action_codes"`
	ResourceSelector []string         `json:"resource_selector"`
	ActionSelector   []string         `json:"action_selector"`
	Enabled          *bool            `json:"enabled"`
}

func (r authorizationPolicyRequest) policy() *biz.AuthorizationPolicy {
	resourceCodes := r.ResourceCodes
	if resourceCodes == nil {
		resourceCodes = r.ResourceSelector
	}
	actionCodes := r.ActionCodes
	if actionCodes == nil {
		actionCodes = r.ActionSelector
	}
	priority := 0
	if r.Priority != nil {
		priority = *r.Priority
	}
	enabled := true
	if r.Enabled != nil {
		enabled = *r.Enabled
	}
	return &biz.AuthorizationPolicy{Application: r.Application, Code: r.Code, Name: r.Name, Description: r.Description, Effect: r.Effect, Priority: priority, ResourceCodes: resourceCodes, ActionCodes: actionCodes, Enabled: enabled}
}

type authorizationPolicyBindingsRequest struct {
	Bindings []authorizationPolicyBindingInput `json:"bindings"`
	Items    []authorizationPolicyBindingInput `json:"items"`
}

type authorizationPolicyBindingInput struct {
	SubjectType  biz.SubjectType `json:"subject_type"`
	SubjectValue string          `json:"subject_value"`
	Enabled      *bool           `json:"enabled"`
}

func (r authorizationPolicyBindingsRequest) values() []biz.AuthorizationPolicyBinding {
	items := r.Bindings
	if items == nil {
		items = r.Items
	}
	values := make([]biz.AuthorizationPolicyBinding, 0, len(items))
	for _, item := range items {
		enabled := true
		if item.Enabled != nil {
			enabled = *item.Enabled
		}
		values = append(values, biz.AuthorizationPolicyBinding{SubjectType: item.SubjectType, SubjectValue: item.SubjectValue, Enabled: enabled})
	}
	return values
}

type authorizationDecisionRequest struct {
	Application  string            `json:"application"`
	SubjectID    string            `json:"subject_id"`
	ResourceCode string            `json:"resource_code"`
	ResourceType biz.ResourceType  `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	Action       string            `json:"action"`
	ActionCode   string            `json:"action_code"`
	ServiceCode  string            `json:"service_code"`
	Method       string            `json:"method"`
	PathTemplate string            `json:"path_template"`
	Subject      *decisionSubject  `json:"subject"`
	Resource     *decisionResource `json:"resource"`
	Request      *decisionHTTP     `json:"request"`
	Environment  map[string]any    `json:"environment"`
}

type decisionSubject struct {
	ID         string         `json:"id"`
	SubjectID  string         `json:"subject_id"`
	Claims     map[string]any `json:"claims"`
	Attributes map[string]any `json:"attributes"`
}

type decisionResource struct {
	Code       string           `json:"code"`
	Type       biz.ResourceType `json:"type"`
	ResourceID string           `json:"id"`
	Attributes map[string]any   `json:"attributes"`
}

type decisionHTTP struct {
	Method       string `json:"method"`
	PathTemplate string `json:"pathTemplate"`
	Path         string `json:"path"`
	RequestID    string `json:"requestId"`
}

func (r authorizationDecisionRequest) normalize() biz.DecisionRequest {
	result := biz.DecisionRequest{Application: r.Application, SubjectID: r.SubjectID, ResourceCode: r.ResourceCode, ResourceType: r.ResourceType, ResourceID: r.ResourceID, Action: r.Action, ServiceCode: r.ServiceCode, Method: r.Method, PathTemplate: r.PathTemplate}
	if result.Action == "" {
		result.Action = r.ActionCode
	}
	if r.Subject != nil {
		if result.SubjectID == "" {
			result.SubjectID = r.Subject.ID
			if result.SubjectID == "" {
				result.SubjectID = r.Subject.SubjectID
			}
		}
	}
	if r.Resource != nil {
		if result.ResourceCode == "" {
			result.ResourceCode = r.Resource.Code
		}
		if result.ResourceType == "" {
			result.ResourceType = r.Resource.Type
		}
		if result.ResourceID == "" {
			result.ResourceID = r.Resource.ResourceID
		}
	}
	if r.Request != nil {
		if result.Method == "" {
			result.Method = r.Request.Method
		}
		if result.PathTemplate == "" {
			result.PathTemplate = r.Request.PathTemplate
			if result.PathTemplate == "" {
				result.PathTemplate = r.Request.Path
			}
		}
	}
	return result
}

func parsePDPUUID(value, name string) (uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return uuid.Nil, fmt.Errorf("%w: %s is required", biz.ErrInvalidArgument, name)
	}
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, fmt.Errorf("%w: %s must be a valid UUID", biz.ErrInvalidArgument, name)
	}
	return id, nil
}

func parsePDPUUIDOptional(value, name string) (uuid.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return uuid.Nil, nil
	}
	return parsePDPUUID(value, name)
}
