package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luck/permission-center-go/internal/biz"
)

const roleColumns = `id, application, code, name, description, enabled,
	created_by_id, created_by_name, created_at, updated_by_id, updated_by_name,
	updated_at, is_deleted`

// RoleRepository stores application-scoped roles and their resource grants.
type RoleRepository struct {
	pool *pgxpool.Pool
}

var _ biz.RoleRepository = (*RoleRepository)(nil)

func NewRoleRepository(pool *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{pool: pool}
}

func (r *RoleRepository) Create(ctx context.Context, role *biz.Role) (*biz.Role, error) {
	if role == nil || strings.TrimSpace(role.Application) == "" {
		return nil, fmt.Errorf("%w: role and application are required", biz.ErrInvalidArgument)
	}
	storedID := strings.TrimSpace(role.ID)
	if storedID == "" {
		generatedID, err := newRoleID()
		if err != nil {
			return nil, err
		}
		storedID = generatedID
	}
	const query = `
		INSERT INTO roles (
			id, application, code, name, description, enabled,
			created_by_id, created_by_name, updated_by_id, updated_by_name, is_deleted
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $7, $8, FALSE)
		RETURNING ` + roleColumns
	actor := biz.AuditActorFromContext(ctx)
	value, err := scanRole(r.pool.QueryRow(ctx, query,
		storedID,
		role.Application,
		role.Code,
		role.Name,
		role.Description,
		role.Enabled,
		actor.ID,
		actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizRole(value), nil
}

func newRoleID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate role id: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (r *RoleRepository) Get(ctx context.Context, id string) (*biz.Role, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: role id is required", biz.ErrInvalidArgument)
	}
	const query = `SELECT ` + roleColumns + `
		FROM roles
		WHERE id = $1 AND is_deleted = FALSE AND enabled = TRUE`
	value, err := scanRole(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizRole(value), nil
}

func (r *RoleRepository) ListByApplication(ctx context.Context, application string) ([]*biz.Role, error) {
	const query = `SELECT ` + roleColumns + `
		FROM roles
		WHERE application = $1 AND is_deleted = FALSE AND enabled = TRUE
		ORDER BY name, id`
	rows, err := r.pool.Query(ctx, query, application)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	roles := make([]*biz.Role, 0)
	for rows.Next() {
		value, err := scanRole(rows)
		if err != nil {
			return nil, mapDBError(err)
		}
		roles = append(roles, toBizRole(value))
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return roles, nil
}

func (r *RoleRepository) Update(ctx context.Context, role *biz.Role) (*biz.Role, error) {
	if role == nil || strings.TrimSpace(role.ID) == "" {
		return nil, fmt.Errorf("%w: role id is required", biz.ErrInvalidArgument)
	}
	const query = `
		UPDATE roles
		SET code = $2, name = $3, description = $4, enabled = $5,
			updated_by_id = $6, updated_by_name = $7, updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
		RETURNING ` + roleColumns
	actor := biz.AuditActorFromContext(ctx)
	value, err := scanRole(r.pool.QueryRow(ctx, query,
		role.ID,
		role.Code,
		role.Name,
		role.Description,
		role.Enabled,
		actor.ID,
		actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizRole(value), nil
}

func (r *RoleRepository) SetEnabled(ctx context.Context, id string, enabled bool) (*biz.Role, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: role id is required", biz.ErrInvalidArgument)
	}
	const query = `
		UPDATE roles
		SET enabled = $2, updated_by_id = $3, updated_by_name = $4, updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
		RETURNING ` + roleColumns
	actor := biz.AuditActorFromContext(ctx)
	value, err := scanRole(r.pool.QueryRow(ctx, query, id, enabled, actor.ID, actor.Name))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizRole(value), nil
}

func (r *RoleRepository) SoftDelete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: role id is required", biz.ErrInvalidArgument)
	}
	const query = `
		UPDATE roles
		SET is_deleted = TRUE, enabled = FALSE,
			updated_by_id = $2, updated_by_name = $3, updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE`
	actor := biz.AuditActorFromContext(ctx)
	result, err := r.pool.Exec(ctx, query, id, actor.ID, actor.Name)
	if err != nil {
		return mapDBError(err)
	}
	if result.RowsAffected() == 0 {
		return biz.ErrNotFound
	}
	return nil
}

// ReplaceResources atomically replaces all grants for a role. The database
// trigger independently checks that every resource belongs to the same app.
func (r *RoleRepository) ReplaceResources(ctx context.Context, roleID string, resourceIDs []uuid.UUID) error {
	if strings.TrimSpace(roleID) == "" {
		return fmt.Errorf("%w: role id is required", biz.ErrInvalidArgument)
	}
	roleID = strings.TrimSpace(roleID)
	for _, resourceID := range resourceIDs {
		if resourceID == uuid.Nil {
			return fmt.Errorf("%w: resource id is required", biz.ErrInvalidArgument)
		}
	}
	actor := biz.AuditActorFromContext(ctx)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return mapDBError(err)
	}
	defer tx.Rollback(ctx)

	var roleExists bool
	if err := tx.QueryRow(ctx, `
		SELECT TRUE FROM roles
		WHERE id = $1 AND is_deleted = FALSE AND enabled = TRUE
		FOR UPDATE`, roleID).Scan(&roleExists); err != nil {
		return mapDBError(err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE role_resources
		SET is_deleted = TRUE, updated_by_id = $2, updated_by_name = $3, updated_at = NOW()
		WHERE role_id = $1 AND is_deleted = FALSE`, roleID, actor.ID, actor.Name); err != nil {
		return mapDBError(err)
	}
	for _, resourceID := range resourceIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO role_resources (
				role_id, resource_id, created_by_id, created_by_name,
				updated_by_id, updated_by_name, is_deleted
			)
			VALUES ($1, $2, $3, $4, $3, $4, FALSE)
			ON CONFLICT (role_id, resource_id) DO UPDATE SET
				is_deleted = FALSE,
				updated_by_id = EXCLUDED.updated_by_id,
				updated_by_name = EXCLUDED.updated_by_name,
				updated_at = NOW()`, roleID, resourceID, actor.ID, actor.Name); err != nil {
			return mapDBError(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return mapDBError(err)
	}
	return nil
}

func (r *RoleRepository) ResourceIDs(ctx context.Context, roleID string) ([]uuid.UUID, error) {
	if strings.TrimSpace(roleID) == "" {
		return nil, fmt.Errorf("%w: role id is required", biz.ErrInvalidArgument)
	}
	const query = `
		SELECT rr.resource_id
		FROM role_resources rr
		JOIN roles ro ON ro.id = rr.role_id
		JOIN resources re ON re.id = rr.resource_id
		WHERE rr.role_id = $1
		  AND rr.is_deleted = FALSE
		  AND ro.is_deleted = FALSE AND ro.enabled = TRUE
		  AND re.is_deleted = FALSE AND re.enabled = TRUE
		ORDER BY re.sort_order, re.name, re.id`
	rows, err := r.pool.Query(ctx, query, roleID)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	resourceIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var resourceID uuid.UUID
		if err := rows.Scan(&resourceID); err != nil {
			return nil, mapDBError(err)
		}
		resourceIDs = append(resourceIDs, resourceID)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return resourceIDs, nil
}
