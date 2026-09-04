package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type memoryPDPRepository struct {
	resources map[uuid.UUID]*AuthorizationResource
	actions   map[uuid.UUID]*AuthorizationAction
	policies  map[uuid.UUID]*AuthorizationPolicy
	bindings  map[uuid.UUID][]AuthorizationPolicyBinding
	endpoints map[uuid.UUID]*AuthorizationAPIEndpoint
}

func newMemoryPDPRepository() *memoryPDPRepository {
	return &memoryPDPRepository{
		resources: map[uuid.UUID]*AuthorizationResource{}, actions: map[uuid.UUID]*AuthorizationAction{},
		policies: map[uuid.UUID]*AuthorizationPolicy{}, bindings: map[uuid.UUID][]AuthorizationPolicyBinding{},
		endpoints: map[uuid.UUID]*AuthorizationAPIEndpoint{},
	}
}

func (r *memoryPDPRepository) CreateResource(_ context.Context, value *AuthorizationResource) (*AuthorizationResource, error) {
	if value.ID == uuid.Nil {
		value.ID = uuid.New()
	}
	r.resources[value.ID] = value
	return value, nil
}
func (r *memoryPDPRepository) GetResource(_ context.Context, id uuid.UUID) (*AuthorizationResource, error) {
	value, ok := r.resources[id]
	if !ok {
		return nil, ErrNotFound
	}
	return value, nil
}
func (r *memoryPDPRepository) GetResourceByCode(_ context.Context, application, code string) (*AuthorizationResource, error) {
	for _, value := range r.resources {
		if value.Application == application && value.Code == code {
			return value, nil
		}
	}
	return nil, ErrNotFound
}
func (r *memoryPDPRepository) ListResources(_ context.Context, application string) ([]*AuthorizationResource, error) {
	values := []*AuthorizationResource{}
	for _, value := range r.resources {
		if value.Application == application {
			values = append(values, value)
		}
	}
	return values, nil
}
func (r *memoryPDPRepository) UpdateResource(_ context.Context, value *AuthorizationResource) (*AuthorizationResource, error) {
	r.resources[value.ID] = value
	return value, nil
}
func (r *memoryPDPRepository) SoftDeleteResource(_ context.Context, id uuid.UUID) error {
	if _, ok := r.resources[id]; !ok {
		return ErrNotFound
	}
	delete(r.resources, id)
	return nil
}

func (r *memoryPDPRepository) CreateAction(_ context.Context, value *AuthorizationAction) (*AuthorizationAction, error) {
	if value.ID == uuid.Nil {
		value.ID = uuid.New()
	}
	r.actions[value.ID] = value
	return value, nil
}
func (r *memoryPDPRepository) GetAction(_ context.Context, id uuid.UUID) (*AuthorizationAction, error) {
	value, ok := r.actions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return value, nil
}
func (r *memoryPDPRepository) GetActionByCode(_ context.Context, application, code string) (*AuthorizationAction, error) {
	for _, value := range r.actions {
		if value.Application == application && value.Code == code {
			return value, nil
		}
	}
	return nil, ErrNotFound
}
func (r *memoryPDPRepository) ListActions(_ context.Context, application string) ([]*AuthorizationAction, error) {
	values := []*AuthorizationAction{}
	for _, value := range r.actions {
		if value.Application == application {
			values = append(values, value)
		}
	}
	return values, nil
}
func (r *memoryPDPRepository) UpdateAction(_ context.Context, value *AuthorizationAction) (*AuthorizationAction, error) {
	r.actions[value.ID] = value
	return value, nil
}
func (r *memoryPDPRepository) SoftDeleteAction(_ context.Context, id uuid.UUID) error {
	if _, ok := r.actions[id]; !ok {
		return ErrNotFound
	}
	delete(r.actions, id)
	return nil
}

func (r *memoryPDPRepository) CreateAPIEndpoint(_ context.Context, value *AuthorizationAPIEndpoint) (*AuthorizationAPIEndpoint, error) {
	if value.ID == uuid.Nil {
		value.ID = uuid.New()
	}
	r.endpoints[value.ID] = value
	return value, nil
}
func (r *memoryPDPRepository) GetAPIEndpoint(_ context.Context, id uuid.UUID) (*AuthorizationAPIEndpoint, error) {
	value, ok := r.endpoints[id]
	if !ok {
		return nil, ErrNotFound
	}
	return value, nil
}
func (r *memoryPDPRepository) GetAPIEndpointByRoute(_ context.Context, application, serviceCode, method, pathTemplate string) (*AuthorizationAPIEndpoint, error) {
	for _, value := range r.endpoints {
		if value.Application == application && value.ServiceCode == serviceCode && value.Method == method && value.PathTemplate == pathTemplate {
			return value, nil
		}
	}
	return nil, ErrNotFound
}
func (r *memoryPDPRepository) ListAPIEndpoints(_ context.Context, application string) ([]*AuthorizationAPIEndpoint, error) {
	values := []*AuthorizationAPIEndpoint{}
	for _, value := range r.endpoints {
		if value.Application == application {
			values = append(values, value)
		}
	}
	return values, nil
}
func (r *memoryPDPRepository) UpdateAPIEndpoint(_ context.Context, value *AuthorizationAPIEndpoint) (*AuthorizationAPIEndpoint, error) {
	r.endpoints[value.ID] = value
	return value, nil
}
func (r *memoryPDPRepository) SoftDeleteAPIEndpoint(_ context.Context, id uuid.UUID) error {
	if _, ok := r.endpoints[id]; !ok {
		return ErrNotFound
	}
	delete(r.endpoints, id)
	return nil
}

func (r *memoryPDPRepository) CreatePolicy(_ context.Context, value *AuthorizationPolicy) (*AuthorizationPolicy, error) {
	if value.ID == uuid.Nil {
		value.ID = uuid.New()
	}
	r.policies[value.ID] = value
	return value, nil
}
func (r *memoryPDPRepository) GetPolicy(_ context.Context, id uuid.UUID) (*AuthorizationPolicy, error) {
	value, ok := r.policies[id]
	if !ok {
		return nil, ErrNotFound
	}
	return value, nil
}
func (r *memoryPDPRepository) ListPolicies(_ context.Context, application string) ([]*AuthorizationPolicy, error) {
	values := []*AuthorizationPolicy{}
	for _, value := range r.policies {
		if value.Application == application {
			values = append(values, value)
		}
	}
	return values, nil
}
func (r *memoryPDPRepository) UpdatePolicy(_ context.Context, value *AuthorizationPolicy) (*AuthorizationPolicy, error) {
	r.policies[value.ID] = value
	return value, nil
}
func (r *memoryPDPRepository) SoftDeletePolicy(_ context.Context, id uuid.UUID) error {
	if _, ok := r.policies[id]; !ok {
		return ErrNotFound
	}
	delete(r.policies, id)
	return nil
}
func (r *memoryPDPRepository) ReplacePolicyBindings(_ context.Context, id uuid.UUID, values []AuthorizationPolicyBinding) error {
	r.bindings[id] = values
	return nil
}
func (r *memoryPDPRepository) ListPolicyBindings(_ context.Context, id uuid.UUID) ([]*AuthorizationPolicyBinding, error) {
	values := []*AuthorizationPolicyBinding{}
	for _, value := range r.bindings[id] {
		copied := value
		values = append(values, &copied)
	}
	return values, nil
}
func (r *memoryPDPRepository) ListMatchingPolicies(_ context.Context, application, subject string, roleIDs []string) ([]*AuthorizationPolicy, error) {
	roles := map[string]struct{}{}
	for _, roleID := range roleIDs {
		roles[roleID] = struct{}{}
	}
	values := []*AuthorizationPolicy{}
	for policyID, policy := range r.policies {
		if policy.Application != application || !policy.Enabled {
			continue
		}
		for _, binding := range r.bindings[policyID] {
			if !binding.Enabled {
				continue
			}
			if binding.SubjectType == SubjectTypeSubject && binding.SubjectValue == subject {
				values = append(values, policy)
				break
			}
			if binding.SubjectType == SubjectTypeRole {
				if _, ok := roles[binding.SubjectValue]; ok {
					values = append(values, policy)
					break
				}
			}
		}
	}
	return values, nil
}

type memoryPDPUserRoles struct{ roles []string }

func (r *memoryPDPUserRoles) ReplaceRoles(context.Context, string, string, []string) error {
	return nil
}
func (r *memoryPDPUserRoles) RoleIDs(context.Context, string, string) ([]string, error) {
	return append([]string(nil), r.roles...), nil
}

func TestPDPDecisionDenyOverridesAllowAndDefaultsToDeny(t *testing.T) {
	repository := newMemoryPDPRepository()
	resourceID, actionID := uuid.New(), uuid.New()
	repository.resources[resourceID] = &AuthorizationResource{ID: resourceID, Application: "forum", Code: "post", Type: ResourceTypeEntity, Enabled: true}
	repository.actions[actionID] = &AuthorizationAction{ID: actionID, Application: "forum", Code: "update", Enabled: true}
	allowID, denyID := uuid.New(), uuid.New()
	repository.policies[allowID] = &AuthorizationPolicy{ID: allowID, Application: "forum", Code: "post-update", Effect: PolicyEffectAllow, Priority: 10, ResourceCodes: []string{"post"}, ActionCodes: []string{"update"}, Enabled: true}
	repository.policies[denyID] = &AuthorizationPolicy{ID: denyID, Application: "forum", Code: "post-deny", Effect: PolicyEffectDeny, Priority: 1, ResourceCodes: []string{"*"}, ActionCodes: []string{"*"}, Enabled: true}
	repository.bindings[allowID] = []AuthorizationPolicyBinding{{SubjectType: SubjectTypeSubject, SubjectValue: "user-1", Enabled: true}}
	repository.bindings[denyID] = []AuthorizationPolicyBinding{{SubjectType: SubjectTypeSubject, SubjectValue: "user-1", Enabled: true}}
	service := NewPDPService(repository, &memoryPDPUserRoles{})
	decision, err := service.Decide(context.Background(), DecisionRequest{Application: "forum", SubjectID: "user-1", ResourceCode: "post", ResourceType: ResourceTypeEntity, Action: "update"})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allow || decision.ReasonCode != "EXPLICIT_DENY" || len(decision.MatchedPolicyIDs) != 2 {
		t.Fatalf("decision = %#v", decision)
	}
	delete(repository.bindings, denyID)
	decision, err = service.Decide(context.Background(), DecisionRequest{Application: "forum", SubjectID: "user-2", ResourceCode: "post", ResourceType: ResourceTypeEntity, Action: "update"})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allow || decision.ReasonCode != "NO_MATCHING_POLICY" {
		t.Fatalf("default decision = %#v", decision)
	}
}

func TestPDPDecisionResolvesRegisteredEndpoint(t *testing.T) {
	repository := newMemoryPDPRepository()
	resourceID, actionID, endpointID, policyID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repository.resources[resourceID] = &AuthorizationResource{ID: resourceID, Application: "forum", Code: "post-api", Type: ResourceTypeAPI, Enabled: true}
	repository.actions[actionID] = &AuthorizationAction{ID: actionID, Application: "forum", Code: "update", Enabled: true}
	repository.endpoints[endpointID] = &AuthorizationAPIEndpoint{ID: endpointID, Application: "forum", ServiceCode: "forum-api", Method: "PUT", PathTemplate: "/v1/posts/{postId}", ResourceID: resourceID, ActionID: actionID, EnforcementMode: EnforcementModeEnforce, Enabled: true}
	repository.policies[policyID] = &AuthorizationPolicy{ID: policyID, Application: "forum", Code: "post-api-update", Effect: PolicyEffectAllow, ResourceCodes: []string{"post-api"}, ActionCodes: []string{"update"}, Enabled: true}
	repository.bindings[policyID] = []AuthorizationPolicyBinding{{SubjectType: SubjectTypeRole, SubjectValue: "editor", Enabled: true}}
	service := NewPDPService(repository, &memoryPDPUserRoles{roles: []string{"editor"}})
	decision, err := service.Decide(context.Background(), DecisionRequest{Application: "forum", SubjectID: "user-1", ServiceCode: "forum-api", Method: "put", PathTemplate: "/v1/posts/{postId}"})
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allow || decision.ReasonCode != "ALLOW" || len(decision.MatchedPolicyIDs) != 1 {
		t.Fatalf("endpoint decision = %#v", decision)
	}
	decision, err = service.Decide(context.Background(), DecisionRequest{Application: "forum", SubjectID: "user-1", ServiceCode: "forum-api", Method: "GET", PathTemplate: "/v1/missing"})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allow || decision.ReasonCode != "ENDPOINT_NOT_FOUND" {
		t.Fatalf("missing endpoint decision = %#v", decision)
	}
}

func TestPDPServiceValidatesEndpointScopeAndBindingValues(t *testing.T) {
	repository := newMemoryPDPRepository()
	resourceID, actionID, policyID := uuid.New(), uuid.New(), uuid.New()
	repository.resources[resourceID] = &AuthorizationResource{ID: resourceID, Application: "forum", Code: "post", Type: ResourceTypeEntity, Enabled: true}
	repository.actions[actionID] = &AuthorizationAction{ID: actionID, Application: "other", Code: "update", Enabled: true}
	repository.policies[policyID] = &AuthorizationPolicy{ID: policyID, Application: "forum", Code: "p", Effect: PolicyEffectAllow, ResourceCodes: []string{"post"}, ActionCodes: []string{"update"}, Enabled: true}
	service := NewPDPService(repository, nil)
	if _, err := service.CreateAPIEndpoint(context.Background(), &AuthorizationAPIEndpoint{Application: "forum", ServiceCode: "forum-api", Method: "GET", PathTemplate: "/v1/posts", ResourceID: resourceID, ActionID: actionID}); err == nil {
		t.Fatal("expected cross-application endpoint rejection")
	}
	if err := service.ReplacePolicyBindings(context.Background(), policyID, []AuthorizationPolicyBinding{{SubjectType: SubjectTypeRole, SubjectValue: ""}}); err == nil {
		t.Fatal("expected invalid binding rejection")
	}
}
