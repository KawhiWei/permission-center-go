package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luck/permission-center-go/internal/biz"
	"github.com/luck/permission-center-go/internal/data/db/model"
)

const serviceResourceColumns = `resource_key, source, external_id, display_name, audience, description,
	enabled, created_by_id, created_by_name, created_at, updated_by_id, updated_by_name,
	updated_at, is_deleted`

// ServiceResourceRepository 将服务资源目录保存到 PostgreSQL。
type ServiceResourceRepository struct {
	pool *pgxpool.Pool
}

var _ biz.ServiceResourceRepository = (*ServiceResourceRepository)(nil)
var _ biz.ServiceResourceSourceRepository = (*ServiceResourceRepository)(nil)

// NewServiceResourceRepository 创建服务资源目录仓储。
func NewServiceResourceRepository(pool *pgxpool.Pool) *ServiceResourceRepository {
	return &ServiceResourceRepository{pool: pool}
}

// Create 创建一条本地服务资源记录；本地记录不接受外部 ID 和远程来源。
func (r *ServiceResourceRepository) Create(ctx context.Context, resource *biz.ServiceResource) (*biz.ServiceResource, error) {
	if resource == nil {
		return nil, fmt.Errorf("%w: service resource is required", biz.ErrInvalidArgument)
	}
	resourceKey, err := normalizeServiceResourceKey(resource.Key)
	if err != nil {
		return nil, err
	}
	if source := strings.TrimSpace(resource.Source); source != "" && !strings.EqualFold(source, biz.ServiceResourceSourceLocal) {
		return nil, fmt.Errorf("%w: local service resource cannot use source %q", biz.ErrConflict, source)
	}
	actor := biz.AuditActorFromContext(ctx)
	const query = `
		INSERT INTO service_resources (
			resource_key, source, external_id, display_name, audience, description, enabled,
			created_by_id, created_by_name, updated_by_id, updated_by_name, is_deleted
		)
		VALUES ($1, 'local', NULL, $2, $3, $4, $5, $6, $7, $6, $7, FALSE)
		RETURNING ` + serviceResourceColumns
	value, err := scanServiceResource(r.pool.QueryRow(ctx, query,
		resourceKey,
		strings.TrimSpace(resource.DisplayName),
		strings.TrimSpace(resource.Audience),
		strings.TrimSpace(resource.Description),
		resource.IsActive,
		actor.ID,
		actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizServiceResource(value), nil
}

// Get 返回指定业务键对应的当前未删除服务资源。
func (r *ServiceResourceRepository) Get(ctx context.Context, resourceKey string) (*biz.ServiceResource, error) {
	resourceKey, err := normalizeServiceResourceKey(resourceKey)
	if err != nil {
		return nil, err
	}
	const query = `
		SELECT ` + serviceResourceColumns + `
		FROM service_resources
		WHERE resource_key = $1 AND is_deleted = FALSE`
	value, err := scanServiceResource(r.pool.QueryRow(ctx, query, resourceKey))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizServiceResource(value), nil
}

// GetBySource 按业务键和来源返回当前未删除的服务资源。
func (r *ServiceResourceRepository) GetBySource(ctx context.Context, resourceKey, source string) (*biz.ServiceResource, error) {
	resourceKey, err := normalizeServiceResourceKey(resourceKey)
	if err != nil {
		return nil, err
	}
	source, err = normalizeServiceResourceSource(source)
	if err != nil {
		return nil, err
	}
	const query = `
		SELECT ` + serviceResourceColumns + `
		FROM service_resources
		WHERE resource_key = $1 AND source = $2 AND is_deleted = FALSE`
	value, err := scanServiceResource(r.pool.QueryRow(ctx, query, resourceKey, source))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizServiceResource(value), nil
}

// List 返回目录中全部当前未删除服务资源。
func (r *ServiceResourceRepository) List(ctx context.Context) ([]*biz.ServiceResource, error) {
	const query = `
		SELECT ` + serviceResourceColumns + `
		FROM service_resources
		WHERE is_deleted = FALSE
		ORDER BY display_name, resource_key`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	resources := make([]*biz.ServiceResource, 0)
	for rows.Next() {
		value, err := scanServiceResource(rows)
		if err != nil {
			return nil, mapDBError(err)
		}
		resources = append(resources, toBizServiceResource(value))
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return resources, nil
}

// ListBySource 返回指定来源的全部当前未删除服务资源。
func (r *ServiceResourceRepository) ListBySource(ctx context.Context, source string) ([]*biz.ServiceResource, error) {
	source, err := normalizeServiceResourceSource(source)
	if err != nil {
		return nil, err
	}
	const query = `
		SELECT ` + serviceResourceColumns + `
		FROM service_resources
		WHERE source = $1 AND is_deleted = FALSE
		ORDER BY display_name, resource_key`
	rows, err := r.pool.Query(ctx, query, source)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	resources := make([]*biz.ServiceResource, 0)
	for rows.Next() {
		value, err := scanServiceResource(rows)
		if err != nil {
			return nil, mapDBError(err)
		}
		resources = append(resources, toBizServiceResource(value))
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return resources, nil
}

// Update 更新本地服务资源的目录字段，不改变业务键、来源和外部 ID。
func (r *ServiceResourceRepository) Update(ctx context.Context, resource *biz.ServiceResource) (*biz.ServiceResource, error) {
	if resource == nil {
		return nil, fmt.Errorf("%w: service resource is required", biz.ErrInvalidArgument)
	}
	resourceKey, err := normalizeServiceResourceKey(resource.Key)
	if err != nil {
		return nil, err
	}
	if source := strings.TrimSpace(resource.Source); source != "" && !strings.EqualFold(source, biz.ServiceResourceSourceLocal) {
		return nil, fmt.Errorf("%w: NexusAuth service resources cannot be updated", biz.ErrConflict)
	}
	if err := r.ensureLocal(ctx, resourceKey); err != nil {
		return nil, err
	}
	actor := biz.AuditActorFromContext(ctx)
	const query = `
		UPDATE service_resources
		SET display_name = $2, audience = $3, description = $4, enabled = $5,
			updated_by_id = $6, updated_by_name = $7, updated_at = NOW()
		WHERE resource_key = $1 AND source = 'local' AND is_deleted = FALSE
		RETURNING ` + serviceResourceColumns
	value, err := scanServiceResource(r.pool.QueryRow(ctx, query,
		resourceKey,
		strings.TrimSpace(resource.DisplayName),
		strings.TrimSpace(resource.Audience),
		strings.TrimSpace(resource.Description),
		resource.IsActive,
		actor.ID,
		actor.Name,
	))
	if err != nil {
		return nil, mapDBError(err)
	}
	return toBizServiceResource(value), nil
}

// SoftDelete 软删除本地服务资源，并拒绝删除 NexusAuth 来源记录。
func (r *ServiceResourceRepository) SoftDelete(ctx context.Context, resourceKey string) error {
	resourceKey, err := normalizeServiceResourceKey(resourceKey)
	if err != nil {
		return err
	}
	if err := r.ensureLocal(ctx, resourceKey); err != nil {
		return err
	}
	actor := biz.AuditActorFromContext(ctx)
	result, err := r.pool.Exec(ctx, `
		UPDATE service_resources
		SET is_deleted = TRUE, enabled = FALSE,
			updated_by_id = $2, updated_by_name = $3, updated_at = NOW()
		WHERE resource_key = $1 AND source = 'local' AND is_deleted = FALSE
			AND NOT EXISTS (
				SELECT 1 FROM roles
				WHERE service_resource = $1 AND is_deleted = FALSE
			)
			AND NOT EXISTS (
				SELECT 1 FROM menus
				WHERE service_resource = $1 AND is_deleted = FALSE
			)
			AND NOT EXISTS (
				SELECT 1 FROM authorization_api_endpoints
				WHERE service_resource = $1 AND is_deleted = FALSE
			)`,
		resourceKey, actor.ID, actor.Name)
	if err != nil {
		return mapDBError(err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("%w: service resource %q is still referenced", biz.ErrConflict, resourceKey)
	}
	return nil
}

// Delete 是 SoftDelete 的目录仓储别名，供应用层按 CRUD 语义注入。
func (r *ServiceResourceRepository) Delete(ctx context.Context, resourceKey string) error {
	return r.SoftDelete(ctx, resourceKey)
}

// UpsertNexusAuth 同步远程记录；同业务键的本地记录始终保持不变。
func (r *ServiceResourceRepository) UpsertNexusAuth(ctx context.Context, resource *biz.ServiceResource) (*biz.ServiceResource, error) {
	if resource == nil {
		return nil, fmt.Errorf("%w: service resource is required", biz.ErrInvalidArgument)
	}
	resourceKey, err := normalizeServiceResourceKey(resource.Key)
	if err != nil {
		return nil, err
	}
	createdAt := any(nil)
	if !resource.CreatedAt.IsZero() {
		createdAt = resource.CreatedAt
	}
	externalID := any(nil)
	if value := strings.TrimSpace(resource.ID); value != "" {
		externalID = value
	}
	actor := biz.AuditActorFromContext(ctx)
	const query = `
		INSERT INTO service_resources (
			resource_key, source, external_id, display_name, audience, description, enabled,
			created_at, created_by_id, created_by_name, updated_by_id, updated_by_name, is_deleted
		)
		VALUES ($1, 'nexusauth', $2, $3, $4, $5, $6, COALESCE($7, NOW()), $8, $9, $8, $9, FALSE)
		ON CONFLICT (resource_key) DO UPDATE
		SET external_id = EXCLUDED.external_id,
			display_name = EXCLUDED.display_name,
			audience = EXCLUDED.audience,
			description = EXCLUDED.description,
			enabled = EXCLUDED.enabled,
			updated_by_id = EXCLUDED.updated_by_id,
			updated_by_name = EXCLUDED.updated_by_name,
			updated_at = NOW(),
			is_deleted = FALSE
		WHERE service_resources.source = 'nexusauth'
		RETURNING ` + serviceResourceColumns
	value, err := scanServiceResource(r.pool.QueryRow(ctx, query,
		resourceKey,
		externalID,
		strings.TrimSpace(resource.DisplayName),
		strings.TrimSpace(resource.Audience),
		strings.TrimSpace(resource.Description),
		resource.IsActive,
		createdAt,
		actor.ID,
		actor.Name,
	))
	if err == nil {
		return toBizServiceResource(value), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, mapDBError(err)
	}
	// PostgreSQL returns no row when the conflict target is a local record.
	// Return that record unchanged so callers can continue syncing other keys.
	value, lookupErr := r.getAny(ctx, resourceKey)
	if lookupErr != nil {
		return nil, mapDBError(lookupErr)
	}
	if value.Source == biz.ServiceResourceSourceLocal {
		return toBizServiceResource(value), nil
	}
	return nil, fmt.Errorf("%w: service resource %q cannot be synchronized", biz.ErrConflict, resourceKey)
}

func (r *ServiceResourceRepository) ensureLocal(ctx context.Context, resourceKey string) error {
	value, err := r.getAny(ctx, resourceKey)
	if err != nil {
		return mapDBError(err)
	}
	if value.Source != biz.ServiceResourceSourceLocal {
		return fmt.Errorf("%w: NexusAuth service resources cannot be modified", biz.ErrConflict)
	}
	if value.IsDeleted {
		return biz.ErrNotFound
	}
	return nil
}

func (r *ServiceResourceRepository) getAny(ctx context.Context, resourceKey string) (*model.ServiceResource, error) {
	const query = `
		SELECT ` + serviceResourceColumns + `
		FROM service_resources
		WHERE resource_key = $1`
	value, err := scanServiceResource(r.pool.QueryRow(ctx, query, resourceKey))
	if err != nil {
		return nil, err
	}
	return value, nil
}

func scanServiceResource(row rowScanner) (*model.ServiceResource, error) {
	value := &model.ServiceResource{}
	err := row.Scan(
		&value.ResourceKey,
		&value.Source,
		&value.ExternalID,
		&value.DisplayName,
		&value.Audience,
		&value.Description,
		&value.Enabled,
		&value.CreatedByID,
		&value.CreatedByName,
		&value.CreatedAt,
		&value.UpdatedByID,
		&value.UpdatedByName,
		&value.UpdatedAt,
		&value.IsDeleted,
	)
	return value, err
}

func toBizServiceResource(value *model.ServiceResource) *biz.ServiceResource {
	if value == nil {
		return nil
	}
	resource := &biz.ServiceResource{
		Key:         value.ResourceKey,
		Name:        value.DisplayName,
		DisplayName: value.DisplayName,
		Audience:    value.Audience,
		Description: value.Description,
		IsActive:    value.Enabled,
		Source:      value.Source,
		CreatedAt:   value.CreatedAt,
	}
	if value.ExternalID != nil {
		resource.ID = *value.ExternalID
	}
	return resource
}

func normalizeServiceResourceKey(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > 128 {
		return "", fmt.Errorf("%w: service_resource must be 1-128 characters", biz.ErrInvalidArgument)
	}
	return value, nil
}

func normalizeServiceResourceSource(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value != biz.ServiceResourceSourceLocal && value != biz.ServiceResourceSourceNexusAuth {
		return "", fmt.Errorf("%w: service resource source must be local or nexusauth", biz.ErrInvalidArgument)
	}
	return value, nil
}
