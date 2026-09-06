package httpserver

import "net/http"

// NewServer 注册健康检查、认证和权限管理路由。
func NewServer(handler *Handler, authMiddleware ...func(http.Handler) http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /swagger/", SwaggerUI())
	mux.HandleFunc("GET /swagger/openapi.json", handler.OpenAPIDocument)
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /auth/config", handler.AuthConfig)
	mux.HandleFunc("GET /auth/login", handler.AuthLogin)
	mux.HandleFunc("GET /signin-oidc", handler.AuthCallback)
	mux.HandleFunc("GET /auth/me", handler.AuthMe)
	mux.HandleFunc("POST /auth/logout", handler.AuthLogout)

	protect := func(next http.Handler) http.Handler {
		if len(authMiddleware) > 0 && authMiddleware[0] != nil {
			return authMiddleware[0](next)
		}
		return next
	}
	mux.Handle("/v1/", protect(http.NotFoundHandler()))
	mux.Handle("POST /v1/roles", protect(http.HandlerFunc(handler.CreateRole)))
	mux.Handle("PUT /v1/roles/{roleID}", protect(http.HandlerFunc(handler.UpdateRole)))
	mux.Handle("DELETE /v1/roles/{roleID}", protect(http.HandlerFunc(handler.DeleteRole)))
	mux.Handle("POST /v1/service-resources", protect(http.HandlerFunc(handler.CreateServiceResource)))
	mux.Handle("GET /v1/service-resources", protect(http.HandlerFunc(handler.ListServiceResources)))
	mux.Handle("GET /v1/service-resources/{key}", protect(http.HandlerFunc(handler.GetServiceResource)))
	mux.Handle("PUT /v1/service-resources/{key}", protect(http.HandlerFunc(handler.UpdateServiceResource)))
	mux.Handle("DELETE /v1/service-resources/{key}", protect(http.HandlerFunc(handler.DeleteServiceResource)))
	mux.Handle("GET /v1/roles", protect(http.HandlerFunc(handler.ListRoles)))
	mux.Handle("POST /v1/menus", protect(http.HandlerFunc(handler.CreateMenu)))
	mux.Handle("PUT /v1/menus/{id}", protect(http.HandlerFunc(handler.UpdateMenu)))
	mux.Handle("DELETE /v1/menus/{id}", protect(http.HandlerFunc(handler.DeleteMenu)))
	mux.Handle("GET /v1/menus/tree", protect(http.HandlerFunc(handler.MenuTree)))
	mux.Handle("POST /v1/authorization/api-endpoints/import-swagger", protect(http.HandlerFunc(handler.ImportSwaggerAPIEndpoints)))
	mux.Handle("POST /v1/authorization/api-endpoints", protect(http.HandlerFunc(handler.CreateAuthorizationAPIEndpoint)))
	mux.Handle("GET /v1/authorization/api-endpoints", protect(http.HandlerFunc(handler.ListAuthorizationAPIEndpoints)))
	mux.Handle("GET /v1/authorization/api-endpoints/{id}", protect(http.HandlerFunc(handler.GetAuthorizationAPIEndpoint)))
	mux.Handle("PUT /v1/authorization/api-endpoints/{id}", protect(http.HandlerFunc(handler.UpdateAuthorizationAPIEndpoint)))
	mux.Handle("DELETE /v1/authorization/api-endpoints/{id}", protect(http.HandlerFunc(handler.DeleteAuthorizationAPIEndpoint)))
	mux.Handle("POST /v1/authorization/policies", protect(http.HandlerFunc(handler.CreateAuthorizationPolicy)))
	mux.Handle("GET /v1/authorization/policies", protect(http.HandlerFunc(handler.ListAuthorizationPolicies)))
	mux.Handle("GET /v1/authorization/policies/{id}", protect(http.HandlerFunc(handler.GetAuthorizationPolicy)))
	mux.Handle("PUT /v1/authorization/policies/{id}", protect(http.HandlerFunc(handler.UpdateAuthorizationPolicy)))
	mux.Handle("DELETE /v1/authorization/policies/{id}", protect(http.HandlerFunc(handler.DeleteAuthorizationPolicy)))
	mux.Handle("POST /v1/authorization/policies/{id}/publish", protect(http.HandlerFunc(handler.PublishAuthorizationPolicy)))
	mux.Handle("POST /v1/authorization/policies/{id}/rollback", protect(http.HandlerFunc(handler.RollbackAuthorizationPolicy)))
	mux.Handle("POST /v1/authorization/policies/simulate", protect(http.HandlerFunc(handler.SimulateAuthorizationPolicy)))
	mux.Handle("POST /v1/pdp/decisions", http.HandlerFunc(handler.DecideAuthorizationPolicy))
	mux.Handle("GET /v1/pdp/decision-logs", protect(http.HandlerFunc(handler.ListPDPDecisionLogs)))
	mux.Handle("PUT /v1/roles/{roleID}/menus", protect(http.HandlerFunc(handler.GrantRoleMenus)))
	mux.Handle("GET /v1/roles/{roleID}/menus", protect(http.HandlerFunc(handler.RoleMenus)))
	mux.Handle("PUT /v1/users/{userID}/roles", protect(http.HandlerFunc(handler.ReplaceUserRoles)))
	mux.Handle("GET /v1/users/{userID}/roles", protect(http.HandlerFunc(handler.UserRoles)))

	return mux
}
