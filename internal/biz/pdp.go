package biz

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// ResourceType identifies the two resource registrations supported by the
// Phase A PDP. API resources protect a route entry point; entity resources
// protect business records after the owning service has loaded their
// attributes.
type ResourceType string

const (
	ResourceTypeAPI    ResourceType = "api"
	ResourceTypeEntity ResourceType = "entity"
)

// PolicyEffect controls the result of a matching policy. Deny always wins over
// allow, regardless of priority; priority is retained for deterministic audit
// and management display ordering.
type PolicyEffect string

const (
	PolicyEffectAllow PolicyEffect = "allow"
	PolicyEffectDeny  PolicyEffect = "deny"
)

type SubjectType string

const (
	SubjectTypeRole    SubjectType = "role"
	SubjectTypeSubject SubjectType = "subject"
)

type EnforcementMode string

const (
	EnforcementModeEnforce  EnforcementMode = "enforce"
	EnforcementModeAudit    EnforcementMode = "audit"
	EnforcementModeDisabled EnforcementMode = "disabled"
)

type AuthorizationResource struct {
	BaseFields
	ID              uuid.UUID `json:"id"`
	ServiceResource string    `json:"service_resource"`
	// Application is retained as a wire-compatible alias for existing clients.
	Application string       `json:"application,omitempty"`
	Code        string       `json:"code"`
	Type        ResourceType `json:"type"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Matcher     string       `json:"matcher,omitempty"`
	Enabled     bool         `json:"enabled"`
}

// Resource is a short alias useful to PEP callers that do not need the
// persistence-oriented authorization prefix.
type Resource = AuthorizationResource

type AuthorizationAction struct {
	BaseFields
	ID              uuid.UUID `json:"id"`
	ServiceResource string    `json:"service_resource"`
	// Application is retained as a wire-compatible alias for existing clients.
	Application string `json:"application,omitempty"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

type Action = AuthorizationAction

type AuthorizationAPIEndpoint struct {
	BaseFields
	ID              uuid.UUID `json:"id"`
	ServiceResource string    `json:"service_resource"`
	// Application is retained as a wire-compatible alias for existing clients.
	Application     string          `json:"application,omitempty"`
	ServiceCode     string          `json:"service_code"`
	Method          string          `json:"method"`
	PathTemplate    string          `json:"path_template"`
	ResourceID      uuid.UUID       `json:"resource_id"`
	ActionID        uuid.UUID       `json:"action_id"`
	EnforcementMode EnforcementMode `json:"enforcement_mode"`
	Enabled         bool            `json:"enabled"`
}

type APIEndpoint = AuthorizationAPIEndpoint

type AuthorizationPolicy struct {
	BaseFields
	ID              uuid.UUID `json:"id"`
	ServiceResource string    `json:"service_resource"`
	// Application is retained as a wire-compatible alias for existing clients.
	Application   string       `json:"application,omitempty"`
	Code          string       `json:"code"`
	Name          string       `json:"name"`
	Description   string       `json:"description"`
	Effect        PolicyEffect `json:"effect"`
	Priority      int          `json:"priority"`
	ResourceCodes []string     `json:"resource_codes"`
	ActionCodes   []string     `json:"action_codes"`
	Enabled       bool         `json:"enabled"`
}

type Policy = AuthorizationPolicy

type AuthorizationPolicyBinding struct {
	BaseFields
	PolicyID     uuid.UUID   `json:"policy_id"`
	SubjectType  SubjectType `json:"subject_type"`
	SubjectValue string      `json:"subject_value"`
	Enabled      bool        `json:"enabled"`
}

type PolicyBinding = AuthorizationPolicyBinding

// DecisionRequest is intentionally flat at the Go API boundary. The HTTP
// adapter also accepts the nested subject/resource/request shape documented for
// service-to-service callers and normalizes it to this type.
type DecisionRequest struct {
	ServiceResource string `json:"service_resource"`
	// Application remains accepted by old PEPs as an alias for service_resource.
	Application  string       `json:"application,omitempty"`
	SubjectID    string       `json:"subject_id"`
	ResourceCode string       `json:"resource_code"`
	ResourceType ResourceType `json:"resource_type"`
	ResourceID   string       `json:"resource_id"`
	Action       string       `json:"action"`
	ServiceCode  string       `json:"service_code"`
	Method       string       `json:"method"`
	PathTemplate string       `json:"path_template"`
}

type Decision struct {
	Allow            bool        `json:"allow"`
	ReasonCode       string      `json:"reasonCode"`
	MatchedPolicyIDs []uuid.UUID `json:"matchedPolicyIds"`
}

// PDPRepository is the persistence contract for authorization configuration
// and decision lookups. It keeps the business service independently testable
// while the concrete implementation remains PostgreSQL-specific.
type PDPRepository interface {
	CreateResource(context.Context, *AuthorizationResource) (*AuthorizationResource, error)
	GetResource(context.Context, uuid.UUID) (*AuthorizationResource, error)
	GetResourceByCode(context.Context, string, string) (*AuthorizationResource, error)
	ListResources(context.Context, string) ([]*AuthorizationResource, error)
	UpdateResource(context.Context, *AuthorizationResource) (*AuthorizationResource, error)
	SoftDeleteResource(context.Context, uuid.UUID) error

	CreateAction(context.Context, *AuthorizationAction) (*AuthorizationAction, error)
	GetAction(context.Context, uuid.UUID) (*AuthorizationAction, error)
	GetActionByCode(context.Context, string, string) (*AuthorizationAction, error)
	ListActions(context.Context, string) ([]*AuthorizationAction, error)
	UpdateAction(context.Context, *AuthorizationAction) (*AuthorizationAction, error)
	SoftDeleteAction(context.Context, uuid.UUID) error

	CreateAPIEndpoint(context.Context, *AuthorizationAPIEndpoint) (*AuthorizationAPIEndpoint, error)
	GetAPIEndpoint(context.Context, uuid.UUID) (*AuthorizationAPIEndpoint, error)
	GetAPIEndpointByRoute(context.Context, string, string, string, string) (*AuthorizationAPIEndpoint, error)
	ListAPIEndpoints(context.Context, string) ([]*AuthorizationAPIEndpoint, error)
	UpdateAPIEndpoint(context.Context, *AuthorizationAPIEndpoint) (*AuthorizationAPIEndpoint, error)
	SoftDeleteAPIEndpoint(context.Context, uuid.UUID) error

	CreatePolicy(context.Context, *AuthorizationPolicy) (*AuthorizationPolicy, error)
	GetPolicy(context.Context, uuid.UUID) (*AuthorizationPolicy, error)
	ListPolicies(context.Context, string) ([]*AuthorizationPolicy, error)
	UpdatePolicy(context.Context, *AuthorizationPolicy) (*AuthorizationPolicy, error)
	SoftDeletePolicy(context.Context, uuid.UUID) error

	ReplacePolicyBindings(context.Context, uuid.UUID, []AuthorizationPolicyBinding) error
	ListPolicyBindings(context.Context, uuid.UUID) ([]*AuthorizationPolicyBinding, error)
	ListMatchingPolicies(context.Context, string, string, []string) ([]*AuthorizationPolicy, error)
}

// PDPService owns validation and the Phase A decision algorithm. A role
// repository is optional for unit tests and is required for role-bound policy
// evaluation in a running application.
type PDPService struct {
	repository       PDPRepository
	userRoles        UserRoleRepository
	roles            RoleRepository
	applications     ApplicationRepository
	serviceResources ServiceResourceCatalog
}

func NewPDPService(repository PDPRepository, userRoles UserRoleRepository, roles ...RoleRepository) *PDPService {
	service := &PDPService{repository: repository, userRoles: userRoles}
	if len(roles) > 0 {
		service.roles = roles[0]
	}
	return service
}

func (s *PDPService) WithApplicationRepository(applications ApplicationRepository) *PDPService {
	s.applications = applications
	return s
}

// WithServiceResourceCatalog makes service-resource metadata the source of
// truth for scope validation. The application repository remains a fallback
// for compatibility with callers that have not migrated their wiring.
func (s *PDPService) WithServiceResourceCatalog(catalog ServiceResourceCatalog) *PDPService {
	s.serviceResources = catalog
	return s
}

func (s *PDPService) ensureReady() error {
	if s == nil || s.repository == nil {
		return fmt.Errorf("pdp repository is not configured")
	}
	return nil
}

func (s *PDPService) ensureApplication(ctx context.Context, application string) error {
	return s.ensureServiceResource(ctx, application)
}

func (s *PDPService) ensureServiceResource(ctx context.Context, serviceResource string) error {
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

func (s *PDPService) CreateResource(ctx context.Context, value *AuthorizationResource) (*AuthorizationResource, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("%w: resource is required", ErrInvalidArgument)
	}
	serviceResource, err := setServiceResourceScope(value.ServiceResource, value.Application)
	if err != nil {
		return nil, err
	}
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	value.ServiceResource, value.Application = serviceResource, serviceResource
	if err := validateAuthorizationResource(value); err != nil {
		return nil, err
	}
	value.BaseFields = NewBaseFields(AuditActorFromContext(ctx))
	return s.repository.CreateResource(ctx, value)
}

func (s *PDPService) GetResource(ctx context.Context, id uuid.UUID) (*AuthorizationResource, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: resource id is required", ErrInvalidArgument)
	}
	return s.repository.GetResource(ctx, id)
}

func (s *PDPService) ListResources(ctx context.Context, application string) ([]*AuthorizationResource, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	serviceResource, err := validateServiceResource(application)
	if err != nil {
		return nil, err
	}
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	return s.repository.ListResources(ctx, serviceResource)
}

func (s *PDPService) UpdateResource(ctx context.Context, value *AuthorizationResource) (*AuthorizationResource, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if value == nil || value.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: resource id is required", ErrInvalidArgument)
	}
	existing, err := s.repository.GetResource(ctx, value.ID)
	if err != nil {
		return nil, err
	}
	if serviceResourceValue(value.ServiceResource, value.Application) == "" {
		value.ServiceResource = serviceResourceValue(existing.ServiceResource, existing.Application)
	}
	serviceResource, err := setServiceResourceScope(value.ServiceResource, value.Application)
	if err != nil {
		return nil, err
	}
	if serviceResource != serviceResourceValue(existing.ServiceResource, existing.Application) {
		return nil, fmt.Errorf("%w: resource application cannot be changed", ErrConflict)
	}
	value.ServiceResource, value.Application = serviceResource, serviceResource
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	if err := validateAuthorizationResource(value); err != nil {
		return nil, err
	}
	value.BaseFields = NewBaseFields(AuditActorFromContext(ctx))
	return s.repository.UpdateResource(ctx, value)
}

func (s *PDPService) DeleteResource(ctx context.Context, id uuid.UUID) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if id == uuid.Nil {
		return fmt.Errorf("%w: resource id is required", ErrInvalidArgument)
	}
	return s.repository.SoftDeleteResource(ctx, id)
}

func (s *PDPService) CreateAction(ctx context.Context, value *AuthorizationAction) (*AuthorizationAction, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("%w: action is required", ErrInvalidArgument)
	}
	serviceResource, err := setServiceResourceScope(value.ServiceResource, value.Application)
	if err != nil {
		return nil, err
	}
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	value.ServiceResource, value.Application = serviceResource, serviceResource
	if err := validateAuthorizationAction(value); err != nil {
		return nil, err
	}
	value.BaseFields = NewBaseFields(AuditActorFromContext(ctx))
	return s.repository.CreateAction(ctx, value)
}

func (s *PDPService) GetAction(ctx context.Context, id uuid.UUID) (*AuthorizationAction, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: action id is required", ErrInvalidArgument)
	}
	return s.repository.GetAction(ctx, id)
}

func (s *PDPService) ListActions(ctx context.Context, application string) ([]*AuthorizationAction, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	serviceResource, err := validateServiceResource(application)
	if err != nil {
		return nil, err
	}
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	return s.repository.ListActions(ctx, serviceResource)
}

func (s *PDPService) UpdateAction(ctx context.Context, value *AuthorizationAction) (*AuthorizationAction, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if value == nil || value.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: action id is required", ErrInvalidArgument)
	}
	existing, err := s.repository.GetAction(ctx, value.ID)
	if err != nil {
		return nil, err
	}
	if serviceResourceValue(value.ServiceResource, value.Application) == "" {
		value.ServiceResource = serviceResourceValue(existing.ServiceResource, existing.Application)
	}
	serviceResource, err := setServiceResourceScope(value.ServiceResource, value.Application)
	if err != nil {
		return nil, err
	}
	if serviceResource != serviceResourceValue(existing.ServiceResource, existing.Application) {
		return nil, fmt.Errorf("%w: action application cannot be changed", ErrConflict)
	}
	value.ServiceResource, value.Application = serviceResource, serviceResource
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	if err := validateAuthorizationAction(value); err != nil {
		return nil, err
	}
	value.BaseFields = NewBaseFields(AuditActorFromContext(ctx))
	return s.repository.UpdateAction(ctx, value)
}

func (s *PDPService) DeleteAction(ctx context.Context, id uuid.UUID) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if id == uuid.Nil {
		return fmt.Errorf("%w: action id is required", ErrInvalidArgument)
	}
	return s.repository.SoftDeleteAction(ctx, id)
}

func (s *PDPService) CreateAPIEndpoint(ctx context.Context, value *AuthorizationAPIEndpoint) (*AuthorizationAPIEndpoint, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("%w: api endpoint is required", ErrInvalidArgument)
	}
	serviceResource, err := setServiceResourceScope(value.ServiceResource, value.Application)
	if err != nil {
		return nil, err
	}
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	value.ServiceResource, value.Application = serviceResource, serviceResource
	if err := s.validateAPIEndpoint(ctx, value); err != nil {
		return nil, err
	}
	value.BaseFields = NewBaseFields(AuditActorFromContext(ctx))
	return s.repository.CreateAPIEndpoint(ctx, value)
}

func (s *PDPService) GetAPIEndpoint(ctx context.Context, id uuid.UUID) (*AuthorizationAPIEndpoint, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: api endpoint id is required", ErrInvalidArgument)
	}
	return s.repository.GetAPIEndpoint(ctx, id)
}

func (s *PDPService) ListAPIEndpoints(ctx context.Context, application string) ([]*AuthorizationAPIEndpoint, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	serviceResource, err := validateServiceResource(application)
	if err != nil {
		return nil, err
	}
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	return s.repository.ListAPIEndpoints(ctx, serviceResource)
}

func (s *PDPService) UpdateAPIEndpoint(ctx context.Context, value *AuthorizationAPIEndpoint) (*AuthorizationAPIEndpoint, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if value == nil || value.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: api endpoint id is required", ErrInvalidArgument)
	}
	existing, err := s.repository.GetAPIEndpoint(ctx, value.ID)
	if err != nil {
		return nil, err
	}
	if serviceResourceValue(value.ServiceResource, value.Application) == "" {
		value.ServiceResource = serviceResourceValue(existing.ServiceResource, existing.Application)
	}
	serviceResource, err := setServiceResourceScope(value.ServiceResource, value.Application)
	if err != nil {
		return nil, err
	}
	if serviceResource != serviceResourceValue(existing.ServiceResource, existing.Application) {
		return nil, fmt.Errorf("%w: api endpoint application cannot be changed", ErrConflict)
	}
	value.ServiceResource, value.Application = serviceResource, serviceResource
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	if err := s.validateAPIEndpoint(ctx, value); err != nil {
		return nil, err
	}
	value.BaseFields = NewBaseFields(AuditActorFromContext(ctx))
	return s.repository.UpdateAPIEndpoint(ctx, value)
}

func (s *PDPService) DeleteAPIEndpoint(ctx context.Context, id uuid.UUID) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if id == uuid.Nil {
		return fmt.Errorf("%w: api endpoint id is required", ErrInvalidArgument)
	}
	return s.repository.SoftDeleteAPIEndpoint(ctx, id)
}

func (s *PDPService) CreatePolicy(ctx context.Context, value *AuthorizationPolicy) (*AuthorizationPolicy, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("%w: policy is required", ErrInvalidArgument)
	}
	serviceResource, err := setServiceResourceScope(value.ServiceResource, value.Application)
	if err != nil {
		return nil, err
	}
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	value.ServiceResource, value.Application = serviceResource, serviceResource
	if err := validateAuthorizationPolicy(value); err != nil {
		return nil, err
	}
	value.BaseFields = NewBaseFields(AuditActorFromContext(ctx))
	return s.repository.CreatePolicy(ctx, value)
}

func (s *PDPService) GetPolicy(ctx context.Context, id uuid.UUID) (*AuthorizationPolicy, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: policy id is required", ErrInvalidArgument)
	}
	return s.repository.GetPolicy(ctx, id)
}

func (s *PDPService) ListPolicies(ctx context.Context, application string) ([]*AuthorizationPolicy, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	serviceResource, err := validateServiceResource(application)
	if err != nil {
		return nil, err
	}
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	return s.repository.ListPolicies(ctx, serviceResource)
}

func (s *PDPService) UpdatePolicy(ctx context.Context, value *AuthorizationPolicy) (*AuthorizationPolicy, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	if value == nil || value.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: policy id is required", ErrInvalidArgument)
	}
	existing, err := s.repository.GetPolicy(ctx, value.ID)
	if err != nil {
		return nil, err
	}
	if serviceResourceValue(value.ServiceResource, value.Application) == "" {
		value.ServiceResource = serviceResourceValue(existing.ServiceResource, existing.Application)
	}
	serviceResource, err := setServiceResourceScope(value.ServiceResource, value.Application)
	if err != nil {
		return nil, err
	}
	if serviceResource != serviceResourceValue(existing.ServiceResource, existing.Application) {
		return nil, fmt.Errorf("%w: policy application cannot be changed", ErrConflict)
	}
	value.ServiceResource, value.Application = serviceResource, serviceResource
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	if err := validateAuthorizationPolicy(value); err != nil {
		return nil, err
	}
	value.BaseFields = NewBaseFields(AuditActorFromContext(ctx))
	return s.repository.UpdatePolicy(ctx, value)
}

func (s *PDPService) DeletePolicy(ctx context.Context, id uuid.UUID) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	if id == uuid.Nil {
		return fmt.Errorf("%w: policy id is required", ErrInvalidArgument)
	}
	return s.repository.SoftDeletePolicy(ctx, id)
}

func (s *PDPService) ReplacePolicyBindings(ctx context.Context, policyID uuid.UUID, values []AuthorizationPolicyBinding) error {
	if err := s.ensureReady(); err != nil {
		return err
	}
	policy, err := s.GetPolicy(ctx, policyID)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(values))
	for index := range values {
		value := &values[index]
		value.PolicyID = policyID
		value.SubjectValue = strings.TrimSpace(value.SubjectValue)
		if value.SubjectValue == "" || len(value.SubjectValue) > 80 {
			return fmt.Errorf("%w: subject_value must be 1-80 characters", ErrInvalidArgument)
		}
		if value.SubjectType != SubjectTypeRole && value.SubjectType != SubjectTypeSubject {
			return fmt.Errorf("%w: subject_type must be role or subject", ErrInvalidArgument)
		}
		key := string(value.SubjectType) + "\x00" + value.SubjectValue
		if _, ok := seen[key]; ok {
			return fmt.Errorf("%w: duplicate policy binding", ErrInvalidArgument)
		}
		seen[key] = struct{}{}
		if value.SubjectType == SubjectTypeRole && s.roles != nil {
			role, roleErr := s.roles.Get(ctx, value.SubjectValue)
			if roleErr != nil {
				return roleErr
			}
			if role.Application != policy.Application {
				return fmt.Errorf("%w: role and policy must belong to the same application", ErrConflict)
			}
		}
	}
	return s.repository.ReplacePolicyBindings(ctx, policyID, values)
}

// ReplaceBindings is a concise alias for callers that already have a policy
// in hand.
func (s *PDPService) ReplaceBindings(ctx context.Context, policyID uuid.UUID, values []AuthorizationPolicyBinding) error {
	return s.ReplacePolicyBindings(ctx, policyID, values)
}

func (s *PDPService) PolicyBindings(ctx context.Context, policyID uuid.UUID) ([]*AuthorizationPolicyBinding, error) {
	if _, err := s.GetPolicy(ctx, policyID); err != nil {
		return nil, err
	}
	return s.repository.ListPolicyBindings(ctx, policyID)
}

func (s *PDPService) Bindings(ctx context.Context, policyID uuid.UUID) ([]*AuthorizationPolicyBinding, error) {
	return s.PolicyBindings(ctx, policyID)
}

// Decide implements Phase A RBAC policy evaluation. A well-formed but
// unauthorized request is represented as an Allow=false decision; malformed
// requests or storage failures return an error to the caller.
func (s *PDPService) Decide(ctx context.Context, request DecisionRequest) (Decision, error) {
	decision := Decision{MatchedPolicyIDs: []uuid.UUID{}}
	if err := s.ensureReady(); err != nil {
		return decision, err
	}
	serviceResource, err := setServiceResourceScope(request.ServiceResource, request.Application)
	if err != nil {
		return decision, err
	}
	subjectID, err := validateID(request.SubjectID, "subject_id")
	if err != nil {
		return decision, err
	}
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return decision, err
	}
	request.ServiceResource, request.Application, request.SubjectID = serviceResource, serviceResource, subjectID

	// The UI keeps a default HTTP method in the simulation form even for an
	// entity decision. A route lookup therefore starts only when a service or
	// path is supplied; once started, all three route fields are required.
	hasRoute := strings.TrimSpace(request.ServiceCode) != "" || strings.TrimSpace(request.PathTemplate) != ""
	if hasRoute {
		serviceCode, method, pathTemplate, normalizeErr := normalizeEndpointRoute(request.ServiceCode, request.Method, request.PathTemplate)
		if normalizeErr != nil {
			return decision, normalizeErr
		}
		endpoint, endpointErr := s.repository.GetAPIEndpointByRoute(ctx, serviceResource, serviceCode, method, pathTemplate)
		if endpointErr != nil {
			if errors.Is(endpointErr, ErrNotFound) {
				decision.ReasonCode = "ENDPOINT_NOT_FOUND"
				return decision, nil
			}
			return decision, endpointErr
		}
		if !endpoint.Enabled || endpoint.EnforcementMode == EnforcementModeDisabled {
			decision.ReasonCode = "ENDPOINT_DISABLED"
			return decision, nil
		}
		request.ServiceCode, request.Method, request.PathTemplate = serviceCode, method, pathTemplate
		resource, resourceErr := s.repository.GetResource(ctx, endpoint.ResourceID)
		if resourceErr != nil {
			if errors.Is(resourceErr, ErrNotFound) {
				decision.ReasonCode = "RESOURCE_NOT_FOUND"
				return decision, nil
			}
			return decision, resourceErr
		}
		action, actionErr := s.repository.GetAction(ctx, endpoint.ActionID)
		if actionErr != nil {
			if errors.Is(actionErr, ErrNotFound) {
				decision.ReasonCode = "ACTION_NOT_FOUND"
				return decision, nil
			}
			return decision, actionErr
		}
		if request.ResourceCode != "" && strings.TrimSpace(request.ResourceCode) != resource.Code {
			decision.ReasonCode = "ENDPOINT_RESOURCE_MISMATCH"
			return decision, nil
		}
		if request.Action != "" && strings.TrimSpace(request.Action) != action.Code {
			decision.ReasonCode = "ENDPOINT_ACTION_MISMATCH"
			return decision, nil
		}
		request.ResourceCode, request.Action = resource.Code, action.Code
		if request.ResourceType == "" {
			request.ResourceType = resource.Type
		}
	}

	request.ResourceCode = strings.TrimSpace(request.ResourceCode)
	request.Action = strings.TrimSpace(request.Action)
	if request.ResourceCode == "" || request.Action == "" {
		return decision, fmt.Errorf("%w: resource_code and action are required", ErrInvalidArgument)
	}
	resource, err := s.repository.GetResourceByCode(ctx, serviceResource, request.ResourceCode)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			decision.ReasonCode = "RESOURCE_NOT_FOUND"
			return decision, nil
		}
		return decision, err
	}
	if !resource.Enabled {
		decision.ReasonCode = "RESOURCE_DISABLED"
		return decision, nil
	}
	if request.ResourceType != "" && request.ResourceType != resource.Type {
		decision.ReasonCode = "RESOURCE_TYPE_MISMATCH"
		return decision, nil
	}
	action, err := s.repository.GetActionByCode(ctx, serviceResource, request.Action)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			decision.ReasonCode = "ACTION_NOT_FOUND"
			return decision, nil
		}
		return decision, err
	}
	if !action.Enabled {
		decision.ReasonCode = "ACTION_DISABLED"
		return decision, nil
	}

	roleIDs := []string{}
	if s.userRoles != nil {
		roleIDs, err = s.userRoles.RoleIDs(ctx, subjectID, serviceResource)
		if err != nil {
			return decision, err
		}
	}
	policies, err := s.repository.ListMatchingPolicies(ctx, serviceResource, subjectID, roleIDs)
	if err != nil {
		return decision, err
	}
	sort.SliceStable(policies, func(i, j int) bool {
		if policies[i].Priority != policies[j].Priority {
			return policies[i].Priority > policies[j].Priority
		}
		return policies[i].ID.String() < policies[j].ID.String()
	})
	matched := make([]*AuthorizationPolicy, 0, len(policies))
	seenPolicyIDs := make(map[uuid.UUID]struct{}, len(policies))
	for _, policy := range policies {
		if policy == nil || !policy.Enabled || !selectorMatches(policy.ResourceCodes, request.ResourceCode) || !selectorMatches(policy.ActionCodes, request.Action) {
			continue
		}
		if _, seen := seenPolicyIDs[policy.ID]; !seen {
			seenPolicyIDs[policy.ID] = struct{}{}
			matched = append(matched, policy)
			decision.MatchedPolicyIDs = append(decision.MatchedPolicyIDs, policy.ID)
		}
	}
	for _, policy := range matched {
		if policy.Effect == PolicyEffectDeny {
			decision.ReasonCode = "EXPLICIT_DENY"
			return decision, nil
		}
	}
	for _, policy := range matched {
		if policy.Effect == PolicyEffectAllow {
			decision.Allow = true
			decision.ReasonCode = "ALLOW"
			return decision, nil
		}
	}
	decision.ReasonCode = "NO_MATCHING_POLICY"
	return decision, nil
}

func (s *PDPService) Simulate(ctx context.Context, request DecisionRequest) (Decision, error) {
	return s.Decide(ctx, request)
}

func (s *PDPService) validateAPIEndpoint(ctx context.Context, value *AuthorizationAPIEndpoint) error {
	serviceCode, method, pathTemplate, err := normalizeEndpointRoute(value.ServiceCode, value.Method, value.PathTemplate)
	if err != nil {
		return err
	}
	if value.ResourceID == uuid.Nil || value.ActionID == uuid.Nil {
		return fmt.Errorf("%w: resource_id and action_id are required", ErrInvalidArgument)
	}
	resource, err := s.repository.GetResource(ctx, value.ResourceID)
	if err != nil {
		return err
	}
	action, err := s.repository.GetAction(ctx, value.ActionID)
	if err != nil {
		return err
	}
	endpointScope := serviceResourceValue(value.ServiceResource, value.Application)
	if serviceResourceValue(resource.ServiceResource, resource.Application) != endpointScope || serviceResourceValue(action.ServiceResource, action.Application) != endpointScope {
		return fmt.Errorf("%w: endpoint, resource, and action must belong to the same application", ErrConflict)
	}
	if resource.Type != ResourceTypeAPI && resource.Type != ResourceTypeEntity {
		return fmt.Errorf("%w: endpoint resource type must be api or entity", ErrInvalidArgument)
	}
	mode := value.EnforcementMode
	if mode == "" {
		mode = EnforcementModeEnforce
	}
	if mode != EnforcementModeEnforce && mode != EnforcementModeAudit && mode != EnforcementModeDisabled {
		return fmt.Errorf("%w: enforcement_mode must be enforce, audit, or disabled", ErrInvalidArgument)
	}
	value.ServiceResource, value.Application = endpointScope, endpointScope
	value.ServiceCode, value.Method, value.PathTemplate, value.EnforcementMode = serviceCode, method, pathTemplate, mode
	return nil
}

func validateAuthorizationResource(value *AuthorizationResource) error {
	if _, _, err := validateCodeAndName(value.Code, value.Name); err != nil {
		return err
	}
	value.Code, value.Name = strings.TrimSpace(value.Code), strings.TrimSpace(value.Name)
	if value.Type != ResourceTypeAPI && value.Type != ResourceTypeEntity {
		return fmt.Errorf("%w: type must be api or entity", ErrInvalidArgument)
	}
	value.Matcher = strings.TrimSpace(value.Matcher)
	if len(value.Matcher) > 500 {
		return fmt.Errorf("%w: matcher must be at most 500 characters", ErrInvalidArgument)
	}
	value.Description = strings.TrimSpace(value.Description)
	return nil
}

func validateAuthorizationAction(value *AuthorizationAction) error {
	if _, _, err := validateCodeAndName(value.Code, value.Name); err != nil {
		return err
	}
	value.Code, value.Name = strings.TrimSpace(value.Code), strings.TrimSpace(value.Name)
	value.Description = strings.TrimSpace(value.Description)
	return nil
}

func validateAuthorizationPolicy(value *AuthorizationPolicy) error {
	if _, _, err := validateCodeAndName(value.Code, value.Name); err != nil {
		return err
	}
	if value.Effect != PolicyEffectAllow && value.Effect != PolicyEffectDeny {
		return fmt.Errorf("%w: effect must be allow or deny", ErrInvalidArgument)
	}
	if value.Priority < -1000000 || value.Priority > 1000000 {
		return fmt.Errorf("%w: priority must be between -1000000 and 1000000", ErrInvalidArgument)
	}
	resourceCodes, err := normalizeSelectors(value.ResourceCodes, "resource_codes")
	if err != nil {
		return err
	}
	actionCodes, err := normalizeSelectors(value.ActionCodes, "action_codes")
	if err != nil {
		return err
	}
	value.Code, value.Name = strings.TrimSpace(value.Code), strings.TrimSpace(value.Name)
	value.Description = strings.TrimSpace(value.Description)
	value.ResourceCodes, value.ActionCodes = resourceCodes, actionCodes
	return nil
}

func normalizeSelectors(values []string, name string) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("%w: %s must contain at least one selector", ErrInvalidArgument, name)
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > 160 {
			return nil, fmt.Errorf("%w: %s values must be 1-160 characters", ErrInvalidArgument, name)
		}
		if _, ok := seen[value]; ok {
			return nil, fmt.Errorf("%w: duplicate %s selector %q", ErrInvalidArgument, name, value)
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func selectorMatches(selectors []string, value string) bool {
	for _, selector := range selectors {
		if selector == "*" || selector == value {
			return true
		}
	}
	return false
}

func normalizeEndpointRoute(serviceCode, method, pathTemplate string) (string, string, string, error) {
	serviceCode = strings.TrimSpace(serviceCode)
	method = strings.ToUpper(strings.TrimSpace(method))
	pathTemplate = strings.TrimSpace(pathTemplate)
	if serviceCode == "" || len(serviceCode) > 100 {
		return "", "", "", fmt.Errorf("%w: service_code must be 1-100 characters", ErrInvalidArgument)
	}
	if method == "" || len(method) > 16 {
		return "", "", "", fmt.Errorf("%w: method is required and must be at most 16 characters", ErrInvalidArgument)
	}
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE":
	default:
		return "", "", "", fmt.Errorf("%w: unsupported HTTP method %q", ErrInvalidArgument, method)
	}
	if pathTemplate == "" || len(pathTemplate) > 500 || !strings.HasPrefix(pathTemplate, "/") {
		return "", "", "", fmt.Errorf("%w: path_template must be an absolute route path", ErrInvalidArgument)
	}
	return serviceCode, method, pathTemplate, nil
}
