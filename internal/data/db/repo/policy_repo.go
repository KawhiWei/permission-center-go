package repo

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luck/permission-center-go/internal/biz"
	"github.com/luck/permission-center-go/internal/policy/model"
)

type AuthorizationPolicyRepository struct{ pool *pgxpool.Pool }

var _ biz.AuthorizationPolicyRepository = (*AuthorizationPolicyRepository)(nil)

func NewAuthorizationPolicyRepository(pool *pgxpool.Pool) *AuthorizationPolicyRepository {
	return &AuthorizationPolicyRepository{pool: pool}
}

const policyColumns = `id, service_resource, code, name, description, effect, status, scope_level, priority, condition, obligations, current_version, created_by_id, created_by_name, created_at, updated_by_id, updated_by_name, updated_at, is_deleted`

func (r *AuthorizationPolicyRepository) Create(ctx context.Context, value *biz.AuthorizationPolicy) (*biz.AuthorizationPolicy, error) {
	if value.ID == uuid.Nil {
		value.ID = uuid.New()
	}
	return r.save(ctx, value, true)
}
func (r *AuthorizationPolicyRepository) Update(ctx context.Context, value *biz.AuthorizationPolicy) (*biz.AuthorizationPolicy, error) {
	return r.save(ctx, value, false)
}
func (r *AuthorizationPolicyRepository) save(ctx context.Context, value *biz.AuthorizationPolicy, create bool) (*biz.AuthorizationPolicy, error) {
	condition, err := marshalJSON(value.Condition)
	if err != nil {
		return nil, err
	}
	obligations, err := marshalJSON(value.Obligations)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer tx.Rollback(ctx)
	actor := biz.AuditActorFromContext(ctx)
	var stored *biz.AuthorizationPolicy
	if create {
		row := tx.QueryRow(ctx, `INSERT INTO authorization_policies (`+policyColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NOW(),$13,$14,NOW(),FALSE) RETURNING `+policyColumns, value.ID, value.ServiceResource, value.Code, value.Name, value.Description, value.Effect, value.Status, value.ScopeLevel, value.Priority, condition, obligations, value.CurrentVersion, actor.ID, actor.Name)
		stored, err = scanPolicy(row)
	} else {
		row := tx.QueryRow(ctx, `UPDATE authorization_policies SET code=$2,name=$3,description=$4,effect=$5,status=$6,scope_level=$7,priority=$8,condition=$9,obligations=$10,current_version=$11,updated_by_id=$12,updated_by_name=$13,updated_at=NOW() WHERE id=$1 AND is_deleted=FALSE RETURNING `+policyColumns, value.ID, value.Code, value.Name, value.Description, value.Effect, value.Status, value.ScopeLevel, value.Priority, condition, obligations, value.CurrentVersion, actor.ID, actor.Name)
		stored, err = scanPolicy(row)
	}
	if err != nil {
		return nil, mapDBError(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM authorization_policy_role_bindings WHERE policy_id=$1`, value.ID); err != nil {
		return nil, mapDBError(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM authorization_policy_api_targets WHERE policy_id=$1`, value.ID); err != nil {
		return nil, mapDBError(err)
	}
	for _, roleID := range value.RoleIDs {
		if _, err = tx.Exec(ctx, `INSERT INTO authorization_policy_role_bindings (policy_id,role_id) VALUES ($1,$2)`, value.ID, roleID); err != nil {
			return nil, mapDBError(err)
		}
	}
	for _, endpointID := range value.EndpointIDs {
		if _, err = tx.Exec(ctx, `INSERT INTO authorization_policy_api_targets (policy_id,endpoint_id) VALUES ($1,$2)`, value.ID, endpointID); err != nil {
			return nil, mapDBError(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, mapDBError(err)
	}
	stored.RoleIDs = value.RoleIDs
	stored.EndpointIDs = value.EndpointIDs
	return stored, nil
}
func (r *AuthorizationPolicyRepository) Get(ctx context.Context, id uuid.UUID) (*biz.AuthorizationPolicy, error) {
	value, err := scanPolicy(r.pool.QueryRow(ctx, `SELECT `+policyColumns+` FROM authorization_policies WHERE id=$1 AND is_deleted=FALSE`, id))
	if err != nil {
		return nil, mapDBError(err)
	}
	return r.loadBindings(ctx, value)
}
func (r *AuthorizationPolicyRepository) ListByServiceResource(ctx context.Context, scope string) ([]*biz.AuthorizationPolicy, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+policyColumns+` FROM authorization_policies WHERE service_resource=$1 AND is_deleted=FALSE ORDER BY priority DESC,name,id`, scope)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	result := []*biz.AuthorizationPolicy{}
	for rows.Next() {
		value, e := scanPolicy(rows)
		if e != nil {
			return nil, mapDBError(e)
		}
		value, e = r.loadBindings(ctx, value)
		if e != nil {
			return nil, e
		}
		result = append(result, value)
	}
	return result, mapDBError(rows.Err())
}
func (r *AuthorizationPolicyRepository) loadBindings(ctx context.Context, value *biz.AuthorizationPolicy) (*biz.AuthorizationPolicy, error) {
	rows, err := r.pool.Query(ctx, `SELECT role_id FROM authorization_policy_role_bindings WHERE policy_id=$1 ORDER BY role_id`, value.ID)
	if err != nil {
		return nil, mapDBError(err)
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, mapDBError(err)
		}
		value.RoleIDs = append(value.RoleIDs, id)
	}
	rows.Close()
	targetRows, err := r.pool.Query(ctx, `SELECT endpoint_id FROM authorization_policy_api_targets WHERE policy_id=$1 ORDER BY endpoint_id`, value.ID)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer targetRows.Close()
	for targetRows.Next() {
		var id uuid.UUID
		if err = targetRows.Scan(&id); err != nil {
			return nil, mapDBError(err)
		}
		value.EndpointIDs = append(value.EndpointIDs, id)
	}
	return value, mapDBError(targetRows.Err())
}
func (r *AuthorizationPolicyRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	actor := biz.AuditActorFromContext(ctx)
	tag, err := r.pool.Exec(ctx, `UPDATE authorization_policies SET is_deleted=TRUE,status='archived',updated_by_id=$2,updated_by_name=$3,updated_at=NOW() WHERE id=$1 AND is_deleted=FALSE`, id, actor.ID, actor.Name)
	if err != nil {
		return mapDBError(err)
	}
	if tag.RowsAffected() == 0 {
		return biz.ErrNotFound
	}
	return nil
}
func (r *AuthorizationPolicyRepository) SaveVersion(ctx context.Context, value biz.PolicyVersion) error {
	snapshot, err := marshalJSON(value.Snapshot)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO authorization_policy_versions (policy_id,version,snapshot,checksum,published_by,published_at) VALUES ($1,$2,$3,$4,$5,$6)`, value.PolicyID, value.Version, snapshot, value.Checksum, value.PublishedBy, value.PublishedAt)
	return mapDBError(err)
}
func (r *AuthorizationPolicyRepository) GetVersion(ctx context.Context, id uuid.UUID, version int) (*biz.PolicyVersion, error) {
	var value biz.PolicyVersion
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT policy_id,version,snapshot,checksum,published_by,published_at FROM authorization_policy_versions WHERE policy_id=$1 AND version=$2`, id, version).Scan(&value.PolicyID, &value.Version, &raw, &value.Checksum, &value.PublishedBy, &value.PublishedAt)
	if err != nil {
		return nil, mapDBError(err)
	}
	if err = json.Unmarshal(raw, &value.Snapshot); err != nil {
		return nil, err
	}
	return &value, nil
}
func (r *AuthorizationPolicyRepository) ListPublishedByEndpoint(ctx context.Context, scope string, endpointID uuid.UUID) ([]model.Snapshot, error) {
	rows, err := r.pool.Query(ctx, `SELECT version.snapshot FROM authorization_policies p JOIN authorization_policy_api_targets t ON t.policy_id=p.id JOIN authorization_policy_versions version ON version.policy_id=p.id AND version.version=p.current_version WHERE p.service_resource=$1 AND p.status='published' AND p.is_deleted=FALSE AND t.endpoint_id=$2 ORDER BY p.priority DESC,p.id`, scope, endpointID)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	result := []model.Snapshot{}
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return nil, mapDBError(err)
		}
		var value model.Snapshot
		if err = json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, mapDBError(rows.Err())
}
func (r *AuthorizationPolicyRepository) SaveDecisionLog(ctx context.Context, value biz.DecisionLog) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO authorization_decision_logs (decision_id,request_id,service_resource,tenant_id,subject_id,endpoint_id,decision,reason_code,matched_policy_ids,policy_snapshot_version,latency_ms,occurred_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, value.DecisionID, value.RequestID, value.ServiceResource, value.TenantID, value.SubjectID, value.EndpointID, value.Decision, value.ReasonCode, value.MatchedPolicyIDs, value.SnapshotVersion, value.LatencyMS, value.OccurredAt)
	return mapDBError(err)
}
func (r *AuthorizationPolicyRepository) ListDecisionLogs(ctx context.Context, scope string, endpointID uuid.UUID) ([]biz.DecisionLog, error) {
	rows, err := r.pool.Query(ctx, `SELECT decision_id,request_id,service_resource,tenant_id,subject_id,endpoint_id,decision,reason_code,matched_policy_ids,policy_snapshot_version,latency_ms,occurred_at FROM authorization_decision_logs WHERE service_resource=$1 AND ($2::uuid IS NULL OR endpoint_id=$2) ORDER BY occurred_at DESC LIMIT 100`, scope, nilUUID(endpointID))
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	result := []biz.DecisionLog{}
	for rows.Next() {
		var value biz.DecisionLog
		if err = rows.Scan(&value.DecisionID, &value.RequestID, &value.ServiceResource, &value.TenantID, &value.SubjectID, &value.EndpointID, &value.Decision, &value.ReasonCode, &value.MatchedPolicyIDs, &value.SnapshotVersion, &value.LatencyMS, &value.OccurredAt); err != nil {
			return nil, mapDBError(err)
		}
		result = append(result, value)
	}
	return result, mapDBError(rows.Err())
}
func scanPolicy(row interface{ Scan(...any) error }) (*biz.AuthorizationPolicy, error) {
	var value biz.AuthorizationPolicy
	var condition, obligations []byte
	err := row.Scan(&value.ID, &value.ServiceResource, &value.Code, &value.Name, &value.Description, &value.Effect, &value.Status, &value.ScopeLevel, &value.Priority, &condition, &obligations, &value.CurrentVersion, &value.CreatedByID, &value.CreatedByName, &value.CreatedAt, &value.UpdatedByID, &value.UpdatedByName, &value.UpdatedAt, &value.IsDeleted)
	if err != nil {
		return nil, err
	}
	if len(condition) > 0 && string(condition) != "null" {
		value.Condition = &model.Condition{}
		if err = json.Unmarshal(condition, value.Condition); err != nil {
			return nil, err
		}
	}
	if len(obligations) > 0 {
		if err = json.Unmarshal(obligations, &value.Obligations); err != nil {
			return nil, err
		}
	}
	return &value, nil
}
func marshalJSON(value any) ([]byte, error) { return json.Marshal(value) }
func nilUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}
