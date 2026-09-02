package repo

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luck/permission-center-go/internal/biz"
)

const applicationColumns = `application, name, description, enabled, created_by_id, created_by_name, created_at, updated_by_id, updated_by_name, updated_at, is_deleted`

type ApplicationRepository struct{ pool *pgxpool.Pool }

var _ biz.ApplicationRepository = (*ApplicationRepository)(nil)

func NewApplicationRepository(pool *pgxpool.Pool) *ApplicationRepository {
	return &ApplicationRepository{pool: pool}
}
func (r *ApplicationRepository) Create(ctx context.Context, a *biz.Application) (*biz.Application, error) {
	actor := biz.AuditActorFromContext(ctx)
	return scanBizApplication(r.pool.QueryRow(ctx, `INSERT INTO applications (application,name,description,enabled,created_by_id,created_by_name,updated_by_id,updated_by_name,is_deleted) VALUES ($1,$2,$3,$4,$5,$6,$5,$6,FALSE) RETURNING `+applicationColumns, a.Application, a.Name, a.Description, a.Enabled, actor.ID, actor.Name))
}
func (r *ApplicationRepository) Get(ctx context.Context, id string) (*biz.Application, error) {
	return scanBizApplication(r.pool.QueryRow(ctx, `SELECT `+applicationColumns+` FROM applications WHERE application=$1 AND is_deleted=FALSE`, id))
}
func (r *ApplicationRepository) List(ctx context.Context) ([]*biz.Application, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+applicationColumns+` FROM applications WHERE is_deleted=FALSE ORDER BY name, application`)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	result := []*biz.Application{}
	for rows.Next() {
		v, e := scanBizApplication(rows)
		if e != nil {
			return nil, e
		}
		result = append(result, v)
	}
	return result, rows.Err()
}
func (r *ApplicationRepository) Update(ctx context.Context, a *biz.Application) (*biz.Application, error) {
	actor := biz.AuditActorFromContext(ctx)
	return scanBizApplication(r.pool.QueryRow(ctx, `UPDATE applications SET name=$2,description=$3,enabled=$4,updated_by_id=$5,updated_by_name=$6,updated_at=NOW() WHERE application=$1 AND is_deleted=FALSE RETURNING `+applicationColumns, a.Application, a.Name, a.Description, a.Enabled, actor.ID, actor.Name))
}
func (r *ApplicationRepository) SoftDelete(ctx context.Context, id string) error {
	var inUse bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM roles WHERE application=$1 AND is_deleted=FALSE) OR EXISTS(SELECT 1 FROM menus WHERE application=$1 AND is_deleted=FALSE)`, id).Scan(&inUse); err != nil {
		return mapDBError(err)
	}
	if inUse {
		return fmt.Errorf("%w: application has active roles or menus", biz.ErrConflict)
	}
	actor := biz.AuditActorFromContext(ctx)
	tag, err := r.pool.Exec(ctx, `UPDATE applications SET is_deleted=TRUE,enabled=FALSE,updated_by_id=$2,updated_by_name=$3,updated_at=NOW() WHERE application=$1 AND is_deleted=FALSE`, id, actor.ID, actor.Name)
	if err != nil {
		return mapDBError(err)
	}
	if tag.RowsAffected() == 0 {
		return biz.ErrNotFound
	}
	return nil
}
func scanBizApplication(row rowScanner) (*biz.Application, error) {
	a := &biz.Application{}
	err := row.Scan(&a.Application, &a.Name, &a.Description, &a.Enabled, &a.CreatedByID, &a.CreatedByName, &a.CreatedAt, &a.UpdatedByID, &a.UpdatedByName, &a.UpdatedAt, &a.IsDeleted)
	if err != nil {
		return nil, mapDBError(err)
	}
	return a, nil
}
