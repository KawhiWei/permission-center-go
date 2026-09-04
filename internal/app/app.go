package app

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/luck/permission-center-go/internal/auth"
	"github.com/luck/permission-center-go/internal/biz"
	"github.com/luck/permission-center-go/internal/config"
	"github.com/luck/permission-center-go/internal/data/db"
	"github.com/luck/permission-center-go/internal/data/db/repo"
)

type Application struct {
	Pool             *pgxpool.Pool
	Permissions      *biz.PermissionService
	PDP              *biz.PDPService
	Applications     *biz.ApplicationService
	ServiceResources biz.ServiceResourceCatalog
	Auth             *auth.Service
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
	serviceResources, err := newServiceResourceCatalog(applications, cfg.ServiceResourceCatalog)
	if err != nil {
		pool.Close()
		return nil, err
	}
	authenticator, err := auth.New(ctx, cfg.OIDC)
	if err != nil {
		pool.Close()
		return nil, err
	}
	permissions := biz.NewPermissionService(roles, menus, userRoles).WithApplicationRepository(applications).WithServiceResourceCatalog(serviceResources)
	pdpRepository := repo.NewPDPRepository(pool)
	pdp := biz.NewPDPService(pdpRepository, userRoles, roles).WithApplicationRepository(applications).WithServiceResourceCatalog(serviceResources)
	// The legacy application handlers remain wire-compatible during migration,
	// but NexusAuth-backed deployments must not mutate a second local catalog.
	return &Application{Pool: pool, Permissions: permissions, PDP: pdp, Applications: biz.NewApplicationService(applications, cfg.ServiceResourceCatalog.Source), ServiceResources: serviceResources, Auth: authenticator}, nil
}

func newServiceResourceCatalog(applications biz.ApplicationRepository, cfg config.ServiceResourceCatalogConfig) (biz.ServiceResourceCatalog, error) {
	switch cfg.Source {
	case "local":
		return biz.NewLocalServiceResourceCatalog(applications), nil
	case "nexusauth":
		timeout, err := cfg.TimeoutDuration()
		if err != nil {
			return nil, err
		}
		return biz.NewNexusAuthServiceResourceCatalog(cfg.BaseURL(), cfg.Credential(), timeout)
	default:
		return nil, fmt.Errorf("unsupported service resource catalog source %q", cfg.Source)
	}
}

func (a *Application) Close() {
	if a != nil && a.Pool != nil {
		a.Pool.Close()
	}
}
