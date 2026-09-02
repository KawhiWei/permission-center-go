package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luck/permission-center-go/internal/biz"
)

const menuColumns = `id, application, parent_id, menu_type, code, name, description,
	route, component, action, http_method, icon, sort_order, metadata,
	enabled, created_by_id, created_by_name, created_at, updated_by_id, updated_by_name,
	updated_at, is_deleted`

// MenuRepository stores both menu and button nodes. The database owns the
// same-application and parent-type constraints; this repository only maps the
// persistence record to the business entity.
type MenuRepository struct {
	pool *pgxpool.Pool
}

func NewMenuRepository(pool *pgxpool.Pool) *MenuRepository {
	return &MenuRepository{pool: pool}
}

func (r *MenuRepository) Create(ctx context.Context, menu *biz.Menu) (*biz.Menu, error) {
	if menu == nil || menu.Application == "" {
		return nil, fmt.Errorf("%w: menu and application are required", biz.ErrInvalidArgument)
	}
	menuID := menu.ID
	if menuID == uuid.Nil {
		menuID = uuid.New()
	}
	const query = `
		INSERT INTO menus (
			id, application, parent_id, menu_type, code, name, description,
			route, component, action, http_method, icon, sort_order, metadata, enabled,
			created_by_id, created_by_name, updated_by_id, updated_by_name, is_deleted
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, '{}'::JSONB,
			$14, $15, $16, $15, $16, FALSE)
		RETURNING ` + menuColumns
	actor := biz.AuditActorFromContext(ctx)
	value, err := scanMenu(r.pool.QueryRow(ctx, query,
		menuID,
		menu.Application,
		menu.ParentID,
		string(menu.Type),
		menu.Code,
		menu.Name,
		menu.Description,
		menu.Path,
		menu.Component,
		menu.APIPath,
		menu.HTTPMethod,
		menu.Icon,
		menu.Sort,
		menu.Enabled,
		actor.ID,
		actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizMenu(value), nil
}

func (r *MenuRepository) Get(ctx context.Context, id uuid.UUID) (*biz.Menu, error) {
	const query = `
		SELECT ` + menuColumns + `
		FROM menus
		WHERE id = $1 AND is_deleted = FALSE AND enabled = TRUE`
	value, err := scanMenu(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizMenu(value), nil
}

func (r *MenuRepository) ListByApplication(ctx context.Context, application string) ([]*biz.Menu, error) {
	const query = `
		SELECT ` + menuColumns + `
		FROM menus
		WHERE application = $1 AND is_deleted = FALSE AND enabled = TRUE
		ORDER BY sort_order, name, id`
	rows, err := r.pool.Query(ctx, query, application)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	menus := make([]*biz.Menu, 0)
	for rows.Next() {
		value, err := scanMenu(rows)
		if err != nil {
			return nil, mapDBError(err)
		}
		menus = append(menus, toBizMenu(value))
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return menus, nil
}

// Update changes mutable menu metadata without changing its application
// or parent. Moving a node is deliberately a separate operation so a caller
// can validate a tree edit explicitly.
func (r *MenuRepository) Update(ctx context.Context, menu *biz.Menu) (*biz.Menu, error) {
	if menu == nil || menu.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: menu id is required", biz.ErrInvalidArgument)
	}
	const query = `
		UPDATE menus
		SET code = $2, name = $3, description = $4, route = $5, component = $6,
			action = $7, http_method = $8, icon = $9, sort_order = $10, enabled = $11,
			updated_by_id = $12, updated_by_name = $13, updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
		RETURNING ` + menuColumns
	actor := biz.AuditActorFromContext(ctx)
	value, err := scanMenu(r.pool.QueryRow(ctx, query,
		menu.ID, menu.Code, menu.Name, menu.Description, menu.Path,
		menu.Component, menu.APIPath, menu.HTTPMethod, menu.Icon, menu.Sort,
		menu.Enabled, actor.ID, actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizMenu(value), nil
}

func (r *MenuRepository) SetEnabled(ctx context.Context, id uuid.UUID, enabled bool) (*biz.Menu, error) {
	const query = `
		UPDATE menus
		SET enabled = $2, updated_by_id = $3, updated_by_name = $4, updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
		RETURNING ` + menuColumns
	actor := biz.AuditActorFromContext(ctx)
	value, err := scanMenu(r.pool.QueryRow(ctx, query, id, enabled, actor.ID, actor.Name))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizMenu(value), nil
}

func (r *MenuRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const query = `
		UPDATE menus
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
