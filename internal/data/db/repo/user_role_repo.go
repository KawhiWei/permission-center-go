package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luck/permission-center-go/internal/biz"
	"github.com/luck/permission-center-go/internal/data/db/model"
)

// UserRoleRepository manages subject-to-role assignments. The application is
// intentionally resolved through roles.user application, because user_roles
// has one canonical row per subject/role pair and must not duplicate scope
// data that can drift.
type UserRoleRepository struct {
	pool *pgxpool.Pool
}

var _ biz.UserRoleRepository = (*UserRoleRepository)(nil)

func NewUserRoleRepository(pool *pgxpool.Pool) *UserRoleRepository {
	return &UserRoleRepository{pool: pool}
}

// ListRoleIDs returns active, non-deleted role IDs assigned to subject in one
// service resource. Assignments to disabled/deleted roles are hidden from callers.
func (r *UserRoleRepository) ListRoleIDs(ctx context.Context, subject, application string) ([]string, error) {
	subject, application, err := normalizeUserRoleScope(subject, application)
	if err != nil {
		return nil, err
	}
	const query = `
		SELECT ur.role_id
		FROM user_roles ur
		JOIN roles ro ON ro.id = ur.role_id
		WHERE ur.subject = $1
		  AND ro.service_resource = $2
		  AND ur.is_deleted = FALSE
		  AND ro.is_deleted = FALSE
		  AND ro.enabled = TRUE
		ORDER BY ro.name, ro.id`
	rows, err := r.pool.Query(ctx, query, subject, application)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	roleIDs := make([]string, 0)
	for rows.Next() {
		value := &model.UserRole{}
		if err := rows.Scan(&value.RoleID); err != nil {
			return nil, mapDBError(err)
		}
		roleIDs = append(roleIDs, value.RoleID)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return roleIDs, nil
}

// RoleIDs is a concise alias for ListRoleIDs for callers that already scope
// the repository operation by subject and application.
func (r *UserRoleRepository) RoleIDs(ctx context.Context, subject, application string) ([]string, error) {
	return r.ListRoleIDs(ctx, subject, application)
}

// Replace atomically replaces assignments for subject within one application.
// Existing assignments in other applications are preserved. Every requested
// role is locked and checked before any delete occurs, so a bad role ID cannot
// leave a partially replaced assignment set.
func (r *UserRoleRepository) Replace(ctx context.Context, subject, application string, roleIDs []string) error {
	subject, application, err := normalizeUserRoleScope(subject, application)
	if err != nil {
		return err
	}
	roleIDs, err = normalizeRoleIDs(roleIDs)
	if err != nil {
		return err
	}
	actor := biz.AuditActorFromContext(ctx)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return mapDBError(err)
	}
	defer tx.Rollback(ctx)

	// FOR SHARE prevents a concurrent role update/delete from invalidating
	// validation between this check and insertion.
	for _, roleID := range roleIDs {
		var storedID string
		err := tx.QueryRow(ctx, `
			SELECT id
			FROM roles
				WHERE id = $1
				  AND service_resource = $2
			  AND is_deleted = FALSE
			  AND enabled = TRUE
			FOR SHARE`, roleID, application).Scan(&storedID)
		if err != nil {
			if err == pgx.ErrNoRows {
				return fmt.Errorf("%w: role %q does not exist, is disabled, deleted, or belongs to another application", biz.ErrConflict, roleID)
			}
			return mapDBError(err)
		}
	}

	// Do not delete by subject alone: a subject may have independent role
	// assignments in another application.
	if _, err := tx.Exec(ctx, `
		UPDATE user_roles ur
		SET is_deleted = TRUE,
			updated_by_id = $3,
			updated_by_name = $4,
			updated_at = NOW()
		FROM roles ro
		WHERE ur.role_id = ro.id
		  AND ur.subject = $1
		  AND ro.service_resource = $2
		  AND ur.is_deleted = FALSE`, subject, application, actor.ID, actor.Name); err != nil {
		return mapDBError(err)
	}
	for _, roleID := range roleIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_roles (
				subject, role_id, created_by_id, created_by_name,
				updated_by_id, updated_by_name, is_deleted
			)
			VALUES ($1, $2, $3, $4, $3, $4, FALSE)
			ON CONFLICT (subject, role_id) DO UPDATE SET
				is_deleted = FALSE,
				updated_by_id = EXCLUDED.updated_by_id,
				updated_by_name = EXCLUDED.updated_by_name,
				updated_at = NOW()`, subject, roleID, actor.ID, actor.Name); err != nil {
			return mapDBError(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return mapDBError(err)
	}
	return nil
}

// ReplaceRoles is the descriptive alias used by service layers.
func (r *UserRoleRepository) ReplaceRoles(ctx context.Context, subject, application string, roleIDs []string) error {
	return r.Replace(ctx, subject, application, roleIDs)
}

func normalizeUserRoleScope(subject, application string) (string, string, error) {
	subject, application = strings.TrimSpace(subject), strings.TrimSpace(application)
	if subject == "" || len(subject) > 80 {
		return "", "", fmt.Errorf("%w: subject must be 1-80 characters", biz.ErrInvalidArgument)
	}
	if application == "" || len([]rune(application)) > 128 {
		return "", "", fmt.Errorf("%w: service_resource must be 1-128 characters", biz.ErrInvalidArgument)
	}
	return subject, application, nil
}

func normalizeRoleIDs(roleIDs []string) ([]string, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	normalized := make([]string, 0, len(roleIDs))
	seen := make(map[string]struct{}, len(roleIDs))
	for _, roleID := range roleIDs {
		roleID = strings.TrimSpace(roleID)
		if roleID == "" || len(roleID) > 80 {
			return nil, fmt.Errorf("%w: role_id must be 1-80 characters", biz.ErrInvalidArgument)
		}
		if _, ok := seen[roleID]; ok {
			return nil, fmt.Errorf("%w: duplicate role_id %q", biz.ErrInvalidArgument, roleID)
		}
		seen[roleID] = struct{}{}
		normalized = append(normalized, roleID)
	}
	return normalized, nil
}
