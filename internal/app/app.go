package app

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luck/permission-center-go/internal/auth"
	"github.com/luck/permission-center-go/internal/biz"
	"github.com/luck/permission-center-go/internal/config"
	"github.com/luck/permission-center-go/internal/data/db"
	"github.com/luck/permission-center-go/internal/data/db/repo"
)

type Application struct {
	Pool        *pgxpool.Pool
	Permissions *biz.PermissionService
	Auth        *auth.Service
}

func New(ctx context.Context, cfg *config.Config) (*Application, error) {
	pool, err := db.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	roles := repo.NewRoleRepository(pool)
	resources := repo.NewResourceRepository(pool)
	userRoles := repo.NewUserRoleRepository(pool)
	authenticator, err := auth.New(ctx, cfg.OIDC)
	if err != nil {
		pool.Close()
		return nil, err
	}
	return &Application{Pool: pool, Permissions: biz.NewPermissionService(roles, resources, userRoles), Auth: authenticator}, nil
}

func (a *Application) Close() {
	if a != nil && a.Pool != nil {
		a.Pool.Close()
	}
}
