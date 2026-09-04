package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luck/permission-center-go/internal/biz"
	"github.com/luck/permission-center-go/internal/data/db/model"
)

const authorizationResourceColumns = `id, application, code, resource_type, name, description, matcher,
	enabled, created_by_id, created_by_name, created_at, updated_by_id, updated_by_name, updated_at, is_deleted`

const authorizationActionColumns = `id, application, code, name, description, enabled,
	created_by_id, created_by_name, created_at, updated_by_id, updated_by_name, updated_at, is_deleted`

const authorizationAPIEndpointColumns = `id, application, service_code, method, path_template,
	resource_id, action_id, enforcement_mode, enabled, created_by_id, created_by_name,
	created_at, updated_by_id, updated_by_name, updated_at, is_deleted`

const authorizationPolicyColumns = `id, application, code, name, description, effect, priority,
	resource_codes, action_codes, enabled, created_by_id, created_by_name, created_at,
	updated_by_id, updated_by_name, updated_at, is_deleted`

const authorizationPolicyBindingColumns = `policy_id, subject_type, subject_value, enabled,
	created_by_id, created_by_name, created_at, updated_by_id, updated_by_name, updated_at, is_deleted`

// PDPRepository persists authorization metadata in PostgreSQL.
type PDPRepository struct {
	pool *pgxpool.Pool
}

var _ biz.PDPRepository = (*PDPRepository)(nil)

func NewPDPRepository(pool *pgxpool.Pool) *PDPRepository {
	return &PDPRepository{pool: pool}
}

func (r *PDPRepository) CreateResource(ctx context.Context, value *biz.AuthorizationResource) (*biz.AuthorizationResource, error) {
	if value == nil {
		return nil, fmt.Errorf("%w: resource is required", biz.ErrInvalidArgument)
	}
	id := value.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	actor := biz.AuditActorFromContext(ctx)
	const query = `
		INSERT INTO authorization_resources (
			id, application, code, resource_type, name, description, matcher, enabled,
			created_by_id, created_by_name, updated_by_id, updated_by_name, is_deleted
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $9, $10, FALSE)
		RETURNING ` + authorizationResourceColumns
	stored, err := scanAuthorizationResource(r.pool.QueryRow(ctx, query,
		id, value.Application, value.Code, string(value.Type), value.Name,
		value.Description, value.Matcher, value.Enabled, actor.ID, actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationResource(stored), nil
}

func (r *PDPRepository) GetResource(ctx context.Context, id uuid.UUID) (*biz.AuthorizationResource, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: resource id is required", biz.ErrInvalidArgument)
	}
	value, err := scanAuthorizationResource(r.pool.QueryRow(ctx, `SELECT `+authorizationResourceColumns+` FROM authorization_resources WHERE id = $1 AND is_deleted = FALSE`, id))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationResource(value), nil
}

func (r *PDPRepository) GetResourceByCode(ctx context.Context, application, code string) (*biz.AuthorizationResource, error) {
	value, err := scanAuthorizationResource(r.pool.QueryRow(ctx, `SELECT `+authorizationResourceColumns+` FROM authorization_resources WHERE application = $1 AND code = $2 AND is_deleted = FALSE`, application, code))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationResource(value), nil
}

func (r *PDPRepository) ListResources(ctx context.Context, application string) ([]*biz.AuthorizationResource, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+authorizationResourceColumns+` FROM authorization_resources WHERE application = $1 AND is_deleted = FALSE ORDER BY name, code, id`, application)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	values := make([]*biz.AuthorizationResource, 0)
	for rows.Next() {
		value, scanErr := scanAuthorizationResource(rows)
		if scanErr != nil {
			return nil, mapDBError(scanErr)
		}
		values = append(values, toBizAuthorizationResource(value))
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return values, nil
}

func (r *PDPRepository) UpdateResource(ctx context.Context, value *biz.AuthorizationResource) (*biz.AuthorizationResource, error) {
	if value == nil || value.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: resource id is required", biz.ErrInvalidArgument)
	}
	actor := biz.AuditActorFromContext(ctx)
	const query = `
		UPDATE authorization_resources
		SET code = $2, resource_type = $3, name = $4, description = $5, matcher = $6,
			enabled = $7, updated_by_id = $8, updated_by_name = $9, updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
		RETURNING ` + authorizationResourceColumns
	stored, err := scanAuthorizationResource(r.pool.QueryRow(ctx, query,
		value.ID, value.Code, string(value.Type), value.Name, value.Description,
		value.Matcher, value.Enabled, actor.ID, actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationResource(stored), nil
}

func (r *PDPRepository) SoftDeleteResource(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: resource id is required", biz.ErrInvalidArgument)
	}
	actor := biz.AuditActorFromContext(ctx)
	result, err := r.pool.Exec(ctx, `UPDATE authorization_resources SET is_deleted = TRUE, enabled = FALSE, updated_by_id = $2, updated_by_name = $3, updated_at = NOW() WHERE id = $1 AND is_deleted = FALSE`, id, actor.ID, actor.Name)
	if err != nil {
		return mapDBError(err)
	}
	if result.RowsAffected() == 0 {
		return biz.ErrNotFound
	}
	return nil
}

func (r *PDPRepository) CreateAction(ctx context.Context, value *biz.AuthorizationAction) (*biz.AuthorizationAction, error) {
	if value == nil {
		return nil, fmt.Errorf("%w: action is required", biz.ErrInvalidArgument)
	}
	id := value.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	actor := biz.AuditActorFromContext(ctx)
	const query = `
		INSERT INTO authorization_actions (
			id, application, code, name, description, enabled,
			created_by_id, created_by_name, updated_by_id, updated_by_name, is_deleted
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $7, $8, FALSE)
		RETURNING ` + authorizationActionColumns
	stored, err := scanAuthorizationAction(r.pool.QueryRow(ctx, query,
		id, value.Application, value.Code, value.Name, value.Description,
		value.Enabled, actor.ID, actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationAction(stored), nil
}

func (r *PDPRepository) GetAction(ctx context.Context, id uuid.UUID) (*biz.AuthorizationAction, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: action id is required", biz.ErrInvalidArgument)
	}
	value, err := scanAuthorizationAction(r.pool.QueryRow(ctx, `SELECT `+authorizationActionColumns+` FROM authorization_actions WHERE id = $1 AND is_deleted = FALSE`, id))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationAction(value), nil
}

func (r *PDPRepository) GetActionByCode(ctx context.Context, application, code string) (*biz.AuthorizationAction, error) {
	value, err := scanAuthorizationAction(r.pool.QueryRow(ctx, `SELECT `+authorizationActionColumns+` FROM authorization_actions WHERE application = $1 AND code = $2 AND is_deleted = FALSE`, application, code))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationAction(value), nil
}

func (r *PDPRepository) ListActions(ctx context.Context, application string) ([]*biz.AuthorizationAction, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+authorizationActionColumns+` FROM authorization_actions WHERE application = $1 AND is_deleted = FALSE ORDER BY name, code, id`, application)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	values := make([]*biz.AuthorizationAction, 0)
	for rows.Next() {
		value, scanErr := scanAuthorizationAction(rows)
		if scanErr != nil {
			return nil, mapDBError(scanErr)
		}
		values = append(values, toBizAuthorizationAction(value))
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return values, nil
}

func (r *PDPRepository) UpdateAction(ctx context.Context, value *biz.AuthorizationAction) (*biz.AuthorizationAction, error) {
	if value == nil || value.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: action id is required", biz.ErrInvalidArgument)
	}
	actor := biz.AuditActorFromContext(ctx)
	const query = `
		UPDATE authorization_actions
		SET code = $2, name = $3, description = $4, enabled = $5,
			updated_by_id = $6, updated_by_name = $7, updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
		RETURNING ` + authorizationActionColumns
	stored, err := scanAuthorizationAction(r.pool.QueryRow(ctx, query,
		value.ID, value.Code, value.Name, value.Description, value.Enabled,
		actor.ID, actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationAction(stored), nil
}

func (r *PDPRepository) SoftDeleteAction(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: action id is required", biz.ErrInvalidArgument)
	}
	actor := biz.AuditActorFromContext(ctx)
	result, err := r.pool.Exec(ctx, `UPDATE authorization_actions SET is_deleted = TRUE, enabled = FALSE, updated_by_id = $2, updated_by_name = $3, updated_at = NOW() WHERE id = $1 AND is_deleted = FALSE`, id, actor.ID, actor.Name)
	if err != nil {
		return mapDBError(err)
	}
	if result.RowsAffected() == 0 {
		return biz.ErrNotFound
	}
	return nil
}

func (r *PDPRepository) CreateAPIEndpoint(ctx context.Context, value *biz.AuthorizationAPIEndpoint) (*biz.AuthorizationAPIEndpoint, error) {
	if value == nil {
		return nil, fmt.Errorf("%w: api endpoint is required", biz.ErrInvalidArgument)
	}
	id := value.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	actor := biz.AuditActorFromContext(ctx)
	const query = `
		INSERT INTO authorization_api_endpoints (
			id, application, service_code, method, path_template, resource_id, action_id,
			enforcement_mode, enabled, created_by_id, created_by_name, updated_by_id,
			updated_by_name, is_deleted
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $10, $11, FALSE)
		RETURNING ` + authorizationAPIEndpointColumns
	stored, err := scanAuthorizationAPIEndpoint(r.pool.QueryRow(ctx, query,
		id, value.Application, value.ServiceCode, value.Method, value.PathTemplate,
		value.ResourceID, value.ActionID, string(value.EnforcementMode), value.Enabled,
		actor.ID, actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationAPIEndpoint(stored), nil
}

func (r *PDPRepository) GetAPIEndpoint(ctx context.Context, id uuid.UUID) (*biz.AuthorizationAPIEndpoint, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: api endpoint id is required", biz.ErrInvalidArgument)
	}
	value, err := scanAuthorizationAPIEndpoint(r.pool.QueryRow(ctx, `SELECT `+authorizationAPIEndpointColumns+` FROM authorization_api_endpoints WHERE id = $1 AND is_deleted = FALSE`, id))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationAPIEndpoint(value), nil
}

func (r *PDPRepository) GetAPIEndpointByRoute(ctx context.Context, application, serviceCode, method, pathTemplate string) (*biz.AuthorizationAPIEndpoint, error) {
	value, err := scanAuthorizationAPIEndpoint(r.pool.QueryRow(ctx, `SELECT `+authorizationAPIEndpointColumns+` FROM authorization_api_endpoints WHERE application = $1 AND service_code = $2 AND method = $3 AND path_template = $4 AND is_deleted = FALSE`, application, serviceCode, method, pathTemplate))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationAPIEndpoint(value), nil
}

func (r *PDPRepository) ListAPIEndpoints(ctx context.Context, application string) ([]*biz.AuthorizationAPIEndpoint, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+authorizationAPIEndpointColumns+` FROM authorization_api_endpoints WHERE application = $1 AND is_deleted = FALSE ORDER BY service_code, method, path_template, id`, application)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	values := make([]*biz.AuthorizationAPIEndpoint, 0)
	for rows.Next() {
		value, scanErr := scanAuthorizationAPIEndpoint(rows)
		if scanErr != nil {
			return nil, mapDBError(scanErr)
		}
		values = append(values, toBizAuthorizationAPIEndpoint(value))
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return values, nil
}

func (r *PDPRepository) UpdateAPIEndpoint(ctx context.Context, value *biz.AuthorizationAPIEndpoint) (*biz.AuthorizationAPIEndpoint, error) {
	if value == nil || value.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: api endpoint id is required", biz.ErrInvalidArgument)
	}
	actor := biz.AuditActorFromContext(ctx)
	const query = `
		UPDATE authorization_api_endpoints
		SET service_code = $2, method = $3, path_template = $4, resource_id = $5,
			action_id = $6, enforcement_mode = $7, enabled = $8, updated_by_id = $9,
			updated_by_name = $10, updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
		RETURNING ` + authorizationAPIEndpointColumns
	stored, err := scanAuthorizationAPIEndpoint(r.pool.QueryRow(ctx, query,
		value.ID, value.ServiceCode, value.Method, value.PathTemplate, value.ResourceID,
		value.ActionID, string(value.EnforcementMode), value.Enabled, actor.ID, actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationAPIEndpoint(stored), nil
}

func (r *PDPRepository) SoftDeleteAPIEndpoint(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: api endpoint id is required", biz.ErrInvalidArgument)
	}
	actor := biz.AuditActorFromContext(ctx)
	result, err := r.pool.Exec(ctx, `UPDATE authorization_api_endpoints SET is_deleted = TRUE, enabled = FALSE, updated_by_id = $2, updated_by_name = $3, updated_at = NOW() WHERE id = $1 AND is_deleted = FALSE`, id, actor.ID, actor.Name)
	if err != nil {
		return mapDBError(err)
	}
	if result.RowsAffected() == 0 {
		return biz.ErrNotFound
	}
	return nil
}

func (r *PDPRepository) CreatePolicy(ctx context.Context, value *biz.AuthorizationPolicy) (*biz.AuthorizationPolicy, error) {
	if value == nil {
		return nil, fmt.Errorf("%w: policy is required", biz.ErrInvalidArgument)
	}
	id := value.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	resourceCodes, err := json.Marshal(value.ResourceCodes)
	if err != nil {
		return nil, fmt.Errorf("marshal resource selectors: %w", err)
	}
	actionCodes, err := json.Marshal(value.ActionCodes)
	if err != nil {
		return nil, fmt.Errorf("marshal action selectors: %w", err)
	}
	actor := biz.AuditActorFromContext(ctx)
	const query = `
		INSERT INTO authorization_policies (
			id, application, code, name, description, effect, priority, resource_codes,
			action_codes, enabled, created_by_id, created_by_name, updated_by_id,
			updated_by_name, is_deleted
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::JSONB, $9::JSONB, $10, $11, $12, $11, $12, FALSE)
		RETURNING ` + authorizationPolicyColumns
	stored, err := scanAuthorizationPolicy(r.pool.QueryRow(ctx, query,
		id, value.Application, value.Code, value.Name, value.Description,
		string(value.Effect), value.Priority, resourceCodes, actionCodes, value.Enabled,
		actor.ID, actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationPolicy(stored), nil
}

func (r *PDPRepository) GetPolicy(ctx context.Context, id uuid.UUID) (*biz.AuthorizationPolicy, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: policy id is required", biz.ErrInvalidArgument)
	}
	value, err := scanAuthorizationPolicy(r.pool.QueryRow(ctx, `SELECT `+authorizationPolicyColumns+` FROM authorization_policies WHERE id = $1 AND is_deleted = FALSE`, id))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationPolicy(value), nil
}

func (r *PDPRepository) ListPolicies(ctx context.Context, application string) ([]*biz.AuthorizationPolicy, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+authorizationPolicyColumns+` FROM authorization_policies WHERE application = $1 AND is_deleted = FALSE ORDER BY priority DESC, name, id`, application)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	values := make([]*biz.AuthorizationPolicy, 0)
	for rows.Next() {
		value, scanErr := scanAuthorizationPolicy(rows)
		if scanErr != nil {
			return nil, mapDBError(scanErr)
		}
		values = append(values, toBizAuthorizationPolicy(value))
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return values, nil
}

func (r *PDPRepository) UpdatePolicy(ctx context.Context, value *biz.AuthorizationPolicy) (*biz.AuthorizationPolicy, error) {
	if value == nil || value.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: policy id is required", biz.ErrInvalidArgument)
	}
	resourceCodes, err := json.Marshal(value.ResourceCodes)
	if err != nil {
		return nil, fmt.Errorf("marshal resource selectors: %w", err)
	}
	actionCodes, err := json.Marshal(value.ActionCodes)
	if err != nil {
		return nil, fmt.Errorf("marshal action selectors: %w", err)
	}
	actor := biz.AuditActorFromContext(ctx)
	const query = `
		UPDATE authorization_policies
		SET code = $2, name = $3, description = $4, effect = $5, priority = $6,
			resource_codes = $7::JSONB, action_codes = $8::JSONB, enabled = $9,
			updated_by_id = $10, updated_by_name = $11, updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
		RETURNING ` + authorizationPolicyColumns
	stored, err := scanAuthorizationPolicy(r.pool.QueryRow(ctx, query,
		value.ID, value.Code, value.Name, value.Description, string(value.Effect),
		value.Priority, resourceCodes, actionCodes, value.Enabled, actor.ID, actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizAuthorizationPolicy(stored), nil
}

func (r *PDPRepository) SoftDeletePolicy(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: policy id is required", biz.ErrInvalidArgument)
	}
	actor := biz.AuditActorFromContext(ctx)
	result, err := r.pool.Exec(ctx, `UPDATE authorization_policies SET is_deleted = TRUE, enabled = FALSE, updated_by_id = $2, updated_by_name = $3, updated_at = NOW() WHERE id = $1 AND is_deleted = FALSE`, id, actor.ID, actor.Name)
	if err != nil {
		return mapDBError(err)
	}
	if result.RowsAffected() == 0 {
		return biz.ErrNotFound
	}
	return nil
}

func (r *PDPRepository) ReplacePolicyBindings(ctx context.Context, policyID uuid.UUID, values []biz.AuthorizationPolicyBinding) error {
	if policyID == uuid.Nil {
		return fmt.Errorf("%w: policy id is required", biz.ErrInvalidArgument)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return mapDBError(err)
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT TRUE FROM authorization_policies WHERE id = $1 AND is_deleted = FALSE FOR UPDATE`, policyID).Scan(&exists); err != nil {
		return mapDBError(err)
	}
	actor := biz.AuditActorFromContext(ctx)
	if _, err := tx.Exec(ctx, `UPDATE authorization_policy_bindings SET is_deleted = TRUE, enabled = FALSE, updated_by_id = $2, updated_by_name = $3, updated_at = NOW() WHERE policy_id = $1 AND is_deleted = FALSE`, policyID, actor.ID, actor.Name); err != nil {
		return mapDBError(err)
	}
	for _, value := range values {
		if _, err := tx.Exec(ctx, `
			INSERT INTO authorization_policy_bindings (
				policy_id, subject_type, subject_value, enabled,
				created_by_id, created_by_name, updated_by_id, updated_by_name, is_deleted
			)
			VALUES ($1, $2, $3, $4, $5, $6, $5, $6, FALSE)
			ON CONFLICT (policy_id, subject_type, subject_value) DO UPDATE SET
				enabled = EXCLUDED.enabled, is_deleted = FALSE,
				updated_by_id = EXCLUDED.updated_by_id,
				updated_by_name = EXCLUDED.updated_by_name, updated_at = NOW()`,
			policyID, string(value.SubjectType), value.SubjectValue, value.Enabled, actor.ID, actor.Name); err != nil {
			return mapDBError(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return mapDBError(err)
	}
	return nil
}

func (r *PDPRepository) ListPolicyBindings(ctx context.Context, policyID uuid.UUID) ([]*biz.AuthorizationPolicyBinding, error) {
	if policyID == uuid.Nil {
		return nil, fmt.Errorf("%w: policy id is required", biz.ErrInvalidArgument)
	}
	rows, err := r.pool.Query(ctx, `SELECT `+authorizationPolicyBindingColumns+` FROM authorization_policy_bindings WHERE policy_id = $1 AND is_deleted = FALSE ORDER BY subject_type, subject_value`, policyID)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	values := make([]*biz.AuthorizationPolicyBinding, 0)
	for rows.Next() {
		value, scanErr := scanAuthorizationPolicyBinding(rows)
		if scanErr != nil {
			return nil, mapDBError(scanErr)
		}
		values = append(values, toBizAuthorizationPolicyBinding(value))
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return values, nil
}

func (r *PDPRepository) ListMatchingPolicies(ctx context.Context, application, subject string, roleIDs []string) ([]*biz.AuthorizationPolicy, error) {
	if roleIDs == nil {
		roleIDs = []string{}
	}
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT p.id, p.application, p.code, p.name, p.description, p.effect, p.priority,
			p.resource_codes, p.action_codes, p.enabled, p.created_by_id, p.created_by_name,
			p.created_at, p.updated_by_id, p.updated_by_name, p.updated_at, p.is_deleted
		FROM authorization_policies p
		JOIN authorization_policy_bindings b ON b.policy_id = p.id
		WHERE p.application = $1 AND p.enabled = TRUE AND p.is_deleted = FALSE
		  AND b.enabled = TRUE AND b.is_deleted = FALSE
		  AND ((b.subject_type = 'subject' AND b.subject_value = $2)
		       OR (b.subject_type = 'role' AND b.subject_value = ANY($3::TEXT[])))
		ORDER BY p.priority DESC, p.id`, application, subject, roleIDs)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	values := make([]*biz.AuthorizationPolicy, 0)
	for rows.Next() {
		value, scanErr := scanAuthorizationPolicy(rows)
		if scanErr != nil {
			return nil, mapDBError(scanErr)
		}
		values = append(values, toBizAuthorizationPolicy(value))
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return values, nil
}

func scanAuthorizationResource(row rowScanner) (*model.AuthorizationResource, error) {
	value := &model.AuthorizationResource{}
	err := row.Scan(&value.ID, &value.Application, &value.Code, &value.Type, &value.Name,
		&value.Description, &value.Matcher, &value.Enabled, &value.CreatedByID,
		&value.CreatedByName, &value.CreatedAt, &value.UpdatedByID, &value.UpdatedByName,
		&value.UpdatedAt, &value.IsDeleted)
	return value, err
}

func scanAuthorizationAction(row rowScanner) (*model.AuthorizationAction, error) {
	value := &model.AuthorizationAction{}
	err := row.Scan(&value.ID, &value.Application, &value.Code, &value.Name, &value.Description,
		&value.Enabled, &value.CreatedByID, &value.CreatedByName, &value.CreatedAt,
		&value.UpdatedByID, &value.UpdatedByName, &value.UpdatedAt, &value.IsDeleted)
	return value, err
}

func scanAuthorizationAPIEndpoint(row rowScanner) (*model.AuthorizationAPIEndpoint, error) {
	value := &model.AuthorizationAPIEndpoint{}
	err := row.Scan(&value.ID, &value.Application, &value.ServiceCode, &value.Method,
		&value.PathTemplate, &value.ResourceID, &value.ActionID, &value.EnforcementMode,
		&value.Enabled, &value.CreatedByID, &value.CreatedByName, &value.CreatedAt,
		&value.UpdatedByID, &value.UpdatedByName, &value.UpdatedAt, &value.IsDeleted)
	return value, err
}

func scanAuthorizationPolicy(row rowScanner) (*model.AuthorizationPolicy, error) {
	value := &model.AuthorizationPolicy{}
	var resourceCodes, actionCodes []byte
	err := row.Scan(&value.ID, &value.Application, &value.Code, &value.Name, &value.Description,
		&value.Effect, &value.Priority, &resourceCodes, &actionCodes, &value.Enabled,
		&value.CreatedByID, &value.CreatedByName, &value.CreatedAt, &value.UpdatedByID,
		&value.UpdatedByName, &value.UpdatedAt, &value.IsDeleted)
	if err != nil {
		return value, err
	}
	if err := json.Unmarshal(resourceCodes, &value.ResourceCodes); err != nil {
		return value, fmt.Errorf("decode resource selectors: %w", err)
	}
	if err := json.Unmarshal(actionCodes, &value.ActionCodes); err != nil {
		return value, fmt.Errorf("decode action selectors: %w", err)
	}
	return value, nil
}

func scanAuthorizationPolicyBinding(row rowScanner) (*model.AuthorizationPolicyBinding, error) {
	value := &model.AuthorizationPolicyBinding{}
	err := row.Scan(&value.PolicyID, &value.SubjectType, &value.SubjectValue, &value.Enabled,
		&value.CreatedByID, &value.CreatedByName, &value.CreatedAt, &value.UpdatedByID,
		&value.UpdatedByName, &value.UpdatedAt, &value.IsDeleted)
	return value, err
}

func toBizAuthorizationResource(value *model.AuthorizationResource) *biz.AuthorizationResource {
	if value == nil {
		return nil
	}
	return &biz.AuthorizationResource{
		BaseFields: baseFields(value.BaseFields), ID: value.ID, Application: value.Application,
		Code: value.Code, Type: biz.ResourceType(value.Type), Name: value.Name,
		Description: value.Description, Matcher: value.Matcher, Enabled: value.Enabled,
	}
}

func toBizAuthorizationAction(value *model.AuthorizationAction) *biz.AuthorizationAction {
	if value == nil {
		return nil
	}
	return &biz.AuthorizationAction{
		BaseFields: baseFields(value.BaseFields), ID: value.ID, Application: value.Application,
		Code: value.Code, Name: value.Name, Description: value.Description, Enabled: value.Enabled,
	}
}

func toBizAuthorizationAPIEndpoint(value *model.AuthorizationAPIEndpoint) *biz.AuthorizationAPIEndpoint {
	if value == nil {
		return nil
	}
	return &biz.AuthorizationAPIEndpoint{
		BaseFields: baseFields(value.BaseFields), ID: value.ID, Application: value.Application,
		ServiceCode: value.ServiceCode, Method: value.Method, PathTemplate: value.PathTemplate,
		ResourceID: value.ResourceID, ActionID: value.ActionID,
		EnforcementMode: biz.EnforcementMode(value.EnforcementMode), Enabled: value.Enabled,
	}
}

func toBizAuthorizationPolicy(value *model.AuthorizationPolicy) *biz.AuthorizationPolicy {
	if value == nil {
		return nil
	}
	return &biz.AuthorizationPolicy{
		BaseFields: baseFields(value.BaseFields), ID: value.ID, Application: value.Application,
		Code: value.Code, Name: value.Name, Description: value.Description,
		Effect: biz.PolicyEffect(value.Effect), Priority: value.Priority,
		ResourceCodes: append([]string(nil), value.ResourceCodes...),
		ActionCodes:   append([]string(nil), value.ActionCodes...), Enabled: value.Enabled,
	}
}

func toBizAuthorizationPolicyBinding(value *model.AuthorizationPolicyBinding) *biz.AuthorizationPolicyBinding {
	if value == nil {
		return nil
	}
	return &biz.AuthorizationPolicyBinding{
		BaseFields: baseFields(value.BaseFields), PolicyID: value.PolicyID,
		SubjectType: biz.SubjectType(value.SubjectType), SubjectValue: value.SubjectValue,
		Enabled: value.Enabled,
	}
}

func baseFields(value model.BaseFields) biz.BaseFields {
	return biz.BaseFields{
		CreatedByID: value.CreatedByID, CreatedByName: value.CreatedByName,
		CreatedAt: value.CreatedAt, UpdatedByID: value.UpdatedByID,
		UpdatedByName: value.UpdatedByName, UpdatedAt: value.UpdatedAt,
		IsDeleted: value.IsDeleted,
	}
}
