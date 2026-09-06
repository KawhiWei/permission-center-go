package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	policycompile "github.com/luck/permission-center-go/internal/policy/compile"
	policyeval "github.com/luck/permission-center-go/internal/policy/eval"
	"github.com/luck/permission-center-go/internal/policy/model"
)

// AuthorizationPolicy 是控制面可编辑的策略记录；发布后会生成不可变快照供 PDP 使用。
type AuthorizationPolicy struct {
	BaseFields
	ID                uuid.UUID               `json:"id"`
	ServiceResource   string                  `json:"service_resource"`
	Code              string                  `json:"code"`
	Name              string                  `json:"name"`
	Description       string                  `json:"description"`
	Effect            model.Effect            `json:"effect"`
	Status            model.Status            `json:"status"`
	AuthorizationType model.AuthorizationType `json:"authorization_type"`
	Priority          int                     `json:"priority"`
	Condition         *model.Condition        `json:"condition,omitempty"`
	CurrentVersion    int                     `json:"current_version"`
	RoleIDs           []string                `json:"role_ids"`
	EndpointIDs       []uuid.UUID             `json:"endpoint_ids"`
}

type PolicyVersion struct {
	PolicyID    uuid.UUID      `json:"policy_id"`
	Version     int            `json:"version"`
	Snapshot    model.Snapshot `json:"snapshot"`
	Checksum    string         `json:"checksum"`
	PublishedBy string         `json:"published_by"`
	PublishedAt time.Time      `json:"published_at"`
}
type DecisionLog struct {
	DecisionID       uuid.UUID        `json:"decision_id"`
	RequestID        string           `json:"request_id"`
	ServiceResource  string           `json:"service_resource"`
	TenantID         string           `json:"tenant_id"`
	SubjectID        string           `json:"subject_id"`
	EndpointID       uuid.UUID        `json:"endpoint_id"`
	Decision         model.Decision   `json:"decision"`
	ReasonCode       model.ReasonCode `json:"reason_code"`
	MatchedPolicyIDs []uuid.UUID      `json:"matched_policy_ids"`
	SnapshotVersion  int              `json:"snapshot_version"`
	LatencyMS        int64            `json:"latency_ms"`
	OccurredAt       time.Time        `json:"occurred_at"`
}

type AuthorizationPolicyRepository interface {
	Create(context.Context, *AuthorizationPolicy) (*AuthorizationPolicy, error)
	Get(context.Context, uuid.UUID) (*AuthorizationPolicy, error)
	ListByServiceResource(context.Context, string) ([]*AuthorizationPolicy, error)
	Update(context.Context, *AuthorizationPolicy) (*AuthorizationPolicy, error)
	SoftDelete(context.Context, uuid.UUID) error
	SaveVersion(context.Context, PolicyVersion) error
	GetVersion(context.Context, uuid.UUID, int) (*PolicyVersion, error)
	ListPublishedByEndpoint(context.Context, string, uuid.UUID) ([]model.Snapshot, error)
	SaveDecisionLog(context.Context, DecisionLog) error
	ListDecisionLogs(context.Context, string, uuid.UUID) ([]DecisionLog, error)
}
type PolicyEndpointRepository interface {
	Get(context.Context, uuid.UUID) (*APIEndpoint, error)
}
type PolicyRoleRepository interface {
	Get(context.Context, string) (*Role, error)
}

// AuthorizationPolicyService 负责策略生命周期、可信角色装载和 PDP 决策编排。
type AuthorizationPolicyService struct {
	repository       AuthorizationPolicyRepository
	endpoints        PolicyEndpointRepository
	roles            PolicyRoleRepository
	userRoles        UserRoleRepository
	serviceResources ServiceResourceCatalog
}

func NewAuthorizationPolicyService(repository AuthorizationPolicyRepository, endpoints PolicyEndpointRepository, roles PolicyRoleRepository, userRoles UserRoleRepository) *AuthorizationPolicyService {
	return &AuthorizationPolicyService{repository: repository, endpoints: endpoints, roles: roles, userRoles: userRoles}
}
func (s *AuthorizationPolicyService) WithServiceResourceCatalog(catalog ServiceResourceCatalog) *AuthorizationPolicyService {
	s.serviceResources = catalog
	return s
}

func (s *AuthorizationPolicyService) Create(ctx context.Context, value *AuthorizationPolicy) (*AuthorizationPolicy, error) {
	if value == nil {
		return nil, fmt.Errorf("%w: policy is required", ErrInvalidArgument)
	}
	if err := s.normalizeAndValidate(ctx, value); err != nil {
		return nil, err
	}
	value.Status = model.StatusDraft
	value.CurrentVersion = 0
	value.BaseFields = NewBaseFields(AuditActorFromContext(ctx))
	return s.repository.Create(ctx, value)
}
func (s *AuthorizationPolicyService) List(ctx context.Context, serviceResource string) ([]*AuthorizationPolicy, error) {
	serviceResource, err := validateServiceResource(serviceResource)
	if err != nil {
		return nil, err
	}
	return s.repository.ListByServiceResource(ctx, serviceResource)
}
func (s *AuthorizationPolicyService) Get(ctx context.Context, id uuid.UUID) (*AuthorizationPolicy, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: policy id is required", ErrInvalidArgument)
	}
	return s.repository.Get(ctx, id)
}
func (s *AuthorizationPolicyService) Update(ctx context.Context, value *AuthorizationPolicy) (*AuthorizationPolicy, error) {
	if value == nil || value.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: policy id is required", ErrInvalidArgument)
	}
	existing, err := s.repository.Get(ctx, value.ID)
	if err != nil {
		return nil, err
	}
	if existing.Status == model.StatusPublished {
		return nil, fmt.Errorf("%w: published policy must be changed through a new draft", ErrConflict)
	}
	value.ServiceResource = existing.ServiceResource
	value.Status = existing.Status
	if err := s.normalizeAndValidate(ctx, value); err != nil {
		return nil, err
	}
	value.CurrentVersion = existing.CurrentVersion
	value.BaseFields = existing.BaseFields
	actor := AuditActorFromContext(ctx)
	value.UpdatedByID, value.UpdatedByName = actor.ID, actor.Name
	return s.repository.Update(ctx, value)
}
func (s *AuthorizationPolicyService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repository.SoftDelete(ctx, id)
}
func (s *AuthorizationPolicyService) Publish(ctx context.Context, id uuid.UUID) (*AuthorizationPolicy, error) {
	value, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if value.Status != model.StatusDraft && value.Status != model.StatusDisabled {
		return nil, fmt.Errorf("%w: only draft or disabled policy can be published", ErrConflict)
	}
	// 发布时固化完整策略，运行时只读取这个版本化快照，不读取正在编辑的草稿。
	snapshot, err := policycompile.Snapshot(model.Snapshot{PolicyID: value.ID, Version: value.CurrentVersion + 1, AuthorizationType: value.AuthorizationType, ServiceResource: value.ServiceResource, Effect: value.Effect, Priority: value.Priority, RoleIDs: value.RoleIDs, EndpointIDs: value.EndpointIDs, Condition: value.Condition})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidArgument, err)
	}
	actor := AuditActorFromContext(ctx)
	version := PolicyVersion{PolicyID: value.ID, Version: snapshot.Version, Snapshot: snapshot, Checksum: policyChecksum(snapshot), PublishedBy: actor.ID, PublishedAt: time.Now().UTC()}
	if err := s.repository.SaveVersion(ctx, version); err != nil {
		return nil, err
	}
	value.Status = model.StatusPublished
	value.CurrentVersion = snapshot.Version
	value.UpdatedByID, value.UpdatedByName = actor.ID, actor.Name
	return s.repository.Update(ctx, value)
}
func (s *AuthorizationPolicyService) Rollback(ctx context.Context, id uuid.UUID, version int) (*AuthorizationPolicy, error) {
	if version < 1 {
		return nil, fmt.Errorf("%w: version must be positive", ErrInvalidArgument)
	}
	value, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	saved, err := s.repository.GetVersion(ctx, id, version)
	if err != nil {
		return nil, err
	}
	value.Effect = saved.Snapshot.Effect
	value.AuthorizationType = saved.Snapshot.AuthorizationType
	value.Priority = saved.Snapshot.Priority
	value.Condition = saved.Snapshot.Condition
	value.RoleIDs = saved.Snapshot.RoleIDs
	value.EndpointIDs = saved.Snapshot.EndpointIDs
	value.Status = model.StatusPublished
	value.CurrentVersion = version
	actor := AuditActorFromContext(ctx)
	value.UpdatedByID, value.UpdatedByName = actor.ID, actor.Name
	return s.repository.Update(ctx, value)
}

// Decide 根据已验证的 subject 在服务端查询角色；客户端无法通过 role_ids 提权。
func (s *AuthorizationPolicyService) Decide(ctx context.Context, input model.Input) (model.Result, error) {
	started := time.Now()
	input.ServiceResource = strings.TrimSpace(input.ServiceResource)
	input.Subject.ID = strings.TrimSpace(input.Subject.ID)
	if !input.AuthorizationType.Valid() {
		return model.Result{Decision: model.DecisionDeny, ReasonCode: model.ReasonInvalidInput, MatchedPolicyIDs: []uuid.UUID{}}, fmt.Errorf("%w: authorization_type must be api or data", ErrInvalidArgument)
	}
	if err := s.ensureScope(ctx, input.ServiceResource); err != nil {
		return model.Result{}, err
	}
	endpoint, err := s.endpoints.Get(ctx, input.Resource.EndpointID)
	if err != nil {
		return model.Result{}, err
	}
	if endpoint.ServiceResource != input.ServiceResource {
		return model.Result{Decision: model.DecisionDeny, ReasonCode: model.ReasonScopeMismatch, MatchedPolicyIDs: []uuid.UUID{}}, nil
	}
	if !endpoint.Enabled {
		return model.Result{Decision: model.DecisionDeny, ReasonCode: model.ReasonEndpointDisabled, MatchedPolicyIDs: []uuid.UUID{}}, nil
	}
	// 角色是鉴权中心掌握的可信事实，不能使用请求 JSON 中的角色信息。
	roleIDs, err := s.userRoles.RoleIDs(ctx, input.Subject.ID, input.ServiceResource)
	if err != nil {
		return model.Result{}, err
	}
	input.Subject.RoleIDs = roleIDs
	snapshots, err := s.repository.ListPublishedByEndpoint(ctx, input.ServiceResource, input.Resource.EndpointID)
	if err != nil {
		return model.Result{}, err
	}
	result, evaluateErr := policyeval.Evaluate(input, snapshots)
	log := DecisionLog{DecisionID: uuid.New(), RequestID: input.RequestID, ServiceResource: input.ServiceResource, TenantID: input.TenantID, SubjectID: input.Subject.ID, EndpointID: input.Resource.EndpointID, Decision: result.Decision, ReasonCode: result.ReasonCode, MatchedPolicyIDs: result.MatchedPolicyIDs, SnapshotVersion: result.SnapshotVersion, LatencyMS: time.Since(started).Milliseconds(), OccurredAt: time.Now().UTC()}
	_ = s.repository.SaveDecisionLog(ctx, log)
	if evaluateErr != nil {
		return result, fmt.Errorf("%w: %v", ErrInvalidArgument, evaluateErr)
	}
	return result, nil
}
func (s *AuthorizationPolicyService) DecisionLogs(ctx context.Context, serviceResource string, endpointID uuid.UUID) ([]DecisionLog, error) {
	return s.repository.ListDecisionLogs(ctx, serviceResource, endpointID)
}

func (s *AuthorizationPolicyService) normalizeAndValidate(ctx context.Context, value *AuthorizationPolicy) error {
	serviceResource, err := validateServiceResource(value.ServiceResource)
	if err != nil {
		return err
	}
	if err = s.ensureScope(ctx, serviceResource); err != nil {
		return err
	}
	value.ServiceResource = serviceResource
	value.Code, value.Name, err = validateCodeAndName(value.Code, value.Name)
	if err != nil {
		return err
	}
	value.Description = strings.TrimSpace(value.Description)
	if !value.Effect.Valid() || !value.AuthorizationType.Valid() {
		return fmt.Errorf("%w: effect and authorization_type are required", ErrInvalidArgument)
	}
	if len(value.RoleIDs) == 0 || len(value.EndpointIDs) == 0 {
		return fmt.Errorf("%w: at least one role and API endpoint are required", ErrInvalidArgument)
	}
	for _, roleID := range value.RoleIDs {
		role, err := s.roles.Get(ctx, roleID)
		if err != nil {
			return err
		}
		if role.ServiceResource != serviceResource || !role.Enabled {
			return fmt.Errorf("%w: role belongs to another service_resource or is disabled", ErrConflict)
		}
	}
	for _, endpointID := range value.EndpointIDs {
		endpoint, err := s.endpoints.Get(ctx, endpointID)
		if err != nil {
			return err
		}
		if endpoint.ServiceResource != serviceResource {
			return fmt.Errorf("%w: endpoint belongs to another service_resource", ErrConflict)
		}
	}
	snapshot, err := policycompile.Snapshot(model.Snapshot{ServiceResource: serviceResource, Effect: value.Effect, AuthorizationType: value.AuthorizationType, RoleIDs: value.RoleIDs, EndpointIDs: value.EndpointIDs, Condition: value.Condition})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidArgument, err)
	}
	value.RoleIDs, value.EndpointIDs = snapshot.RoleIDs, snapshot.EndpointIDs
	return nil
}
func (s *AuthorizationPolicyService) ensureScope(ctx context.Context, serviceResource string) error {
	if s.serviceResources == nil {
		return nil
	}
	resource, err := s.serviceResources.Get(ctx, serviceResource)
	if err != nil {
		return err
	}
	if resource == nil {
		return ErrNotFound
	}
	if !resource.IsActive {
		return fmt.Errorf("%w: service resource is disabled", ErrConflict)
	}
	return nil
}
func policyChecksum(value model.Snapshot) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}
