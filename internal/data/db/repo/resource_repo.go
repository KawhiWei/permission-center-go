package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luck/permission-center-go/internal/biz"
)

const resourceColumns = `id, application, parent_id, resource_type, code, name, description,
	route, component, action, http_method, icon, sort_order, metadata,
	enabled, created_by_id, created_by_name, created_at, updated_by_id, updated_by_name,
	updated_at, is_deleted`

// ResourceRepository stores both menu and button nodes. The database owns the
// same-application and parent-type constraints; this repository only maps the
// persistence record to the business entity.
type ResourceRepository struct {
	pool *pgxpool.Pool
}

func NewResourceRepository(pool *pgxpool.Pool) *ResourceRepository {
	return &ResourceRepository{pool: pool}
}

func (r *ResourceRepository) Create(ctx context.Context, resource *biz.Resource) (*biz.Resource, error) {
	if resource == nil || resource.Application == "" {
		return nil, fmt.Errorf("%w: resource and application are required", biz.ErrInvalidArgument)
	}
	resourceID := resource.ID
	if resourceID == uuid.Nil {
		resourceID = uuid.New()
	}
	const query = `
		INSERT INTO resources (
			id, application, parent_id, resource_type, code, name, description,
			route, component, action, http_method, icon, sort_order, metadata, enabled,
			created_by_id, created_by_name, updated_by_id, updated_by_name, is_deleted
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, '{}'::JSONB,
			$14, $15, $16, $15, $16, FALSE)
		RETURNING ` + resourceColumns
	actor := biz.AuditActorFromContext(ctx)
	value, err := scanResource(r.pool.QueryRow(ctx, query,
		resourceID,
		resource.Application,
		resource.ParentID,
		string(resource.Type),
		resource.Code,
		resource.Name,
		resource.Description,
		resource.Path,
		resource.Component,
		resource.APIPath,
		resource.HTTPMethod,
		resource.Icon,
		resource.Sort,
		resource.Enabled,
		actor.ID,
		actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizResource(value), nil
}

func (r *ResourceRepository) Get(ctx context.Context, id uuid.UUID) (*biz.Resource, error) {
	const query = `
		SELECT ` + resourceColumns + `
		FROM resources
		WHERE id = $1 AND is_deleted = FALSE AND enabled = TRUE`
	value, err := scanResource(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizResource(value), nil
}

func (r *ResourceRepository) ListByApplication(ctx context.Context, application string) ([]*biz.Resource, error) {
	const query = `
		SELECT ` + resourceColumns + `
		FROM resources
		WHERE application = $1 AND is_deleted = FALSE AND enabled = TRUE
		ORDER BY sort_order, name, id`
	rows, err := r.pool.Query(ctx, query, application)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	resources := make([]*biz.Resource, 0)
	for rows.Next() {
		value, err := scanResource(rows)
		if err != nil {
			return nil, mapDBError(err)
		}
		resources = append(resources, toBizResource(value))
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return resources, nil
}

// Update changes mutable resource metadata without changing its application
// or parent. Moving a node is deliberately a separate operation so a caller
// can validate a tree edit explicitly.
func (r *ResourceRepository) Update(ctx context.Context, resource *biz.Resource) (*biz.Resource, error) {
	if resource == nil || resource.ID == uuid.Nil {
		return nil, fmt.Errorf("%w: resource id is required", biz.ErrInvalidArgument)
	}
	const query = `
		UPDATE resources
		SET code = $2, name = $3, description = $4, route = $5, component = $6,
			action = $7, http_method = $8, icon = $9, sort_order = $10, enabled = $11,
			updated_by_id = $12, updated_by_name = $13, updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
		RETURNING ` + resourceColumns
	actor := biz.AuditActorFromContext(ctx)
	value, err := scanResource(r.pool.QueryRow(ctx, query,
		resource.ID, resource.Code, resource.Name, resource.Description, resource.Path,
		resource.Component, resource.APIPath, resource.HTTPMethod, resource.Icon, resource.Sort,
		resource.Enabled, actor.ID, actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizResource(value), nil
}

func (r *ResourceRepository) SetEnabled(ctx context.Context, id uuid.UUID, enabled bool) (*biz.Resource, error) {
	const query = `
		UPDATE resources
		SET enabled = $2, updated_by_id = $3, updated_by_name = $4, updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
		RETURNING ` + resourceColumns
	actor := biz.AuditActorFromContext(ctx)
	value, err := scanResource(r.pool.QueryRow(ctx, query, id, enabled, actor.ID, actor.Name))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizResource(value), nil
}

func (r *ResourceRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const query = `
		UPDATE resources
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
