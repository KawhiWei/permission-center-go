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
	Pool         *pgxpool.Pool
	Permissions  *biz.PermissionService
	Applications *biz.ApplicationService
	Auth         *auth.Service
}

func New(ctx context.Context, cfg *config.Config) (*Application, error) {
	pool, err := db.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	roles := repo.NewRoleRepository(pool)
	menus := repo.NewMenuRepository(pool)
	userRoles := repo.NewUserRoleRepository(pool)
	applications := repo.NewApplicationRepository(pool)
	authenticator, err := auth.New(ctx, cfg.OIDC)
	if err != nil {
		pool.Close()
		return nil, err
	}
	permissions := biz.NewPermissionService(roles, menus, userRoles).WithApplicationRepository(applications)
	return &Application{Pool: pool, Permissions: permissions, Applications: biz.NewApplicationService(applications, cfg.ApplicationCatalog.Source), Auth: authenticator}, nil
}

func (a *Application) Close() {
	if a != nil && a.Pool != nil {
		a.Pool.Close()
	}
}
