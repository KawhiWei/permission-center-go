package httpserver

import "net/http"

func NewServer(handler *Handler, authMiddleware ...func(http.Handler) http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
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
	mux.Handle("POST /v1/applications", protect(http.HandlerFunc(handler.CreateApplication)))
	mux.Handle("GET /v1/applications", protect(http.HandlerFunc(handler.ListApplications)))
	mux.Handle("GET /v1/applications/{application}", protect(http.HandlerFunc(handler.GetApplication)))
	mux.Handle("PUT /v1/applications/{application}", protect(http.HandlerFunc(handler.UpdateApplication)))
	mux.Handle("DELETE /v1/applications/{application}", protect(http.HandlerFunc(handler.DeleteApplication)))
	mux.Handle("GET /v1/roles", protect(http.HandlerFunc(handler.ListRoles)))
	mux.Handle("POST /v1/menus", protect(http.HandlerFunc(handler.CreateMenu)))
	mux.Handle("GET /v1/menus/tree", protect(http.HandlerFunc(handler.MenuTree)))
	mux.Handle("PUT /v1/roles/{roleID}/menus", protect(http.HandlerFunc(handler.GrantRoleMenus)))
	mux.Handle("GET /v1/roles/{roleID}/menus", protect(http.HandlerFunc(handler.RoleMenus)))
	mux.Handle("PUT /v1/users/{userID}/roles", protect(http.HandlerFunc(handler.ReplaceUserRoles)))
	mux.Handle("GET /v1/users/{userID}/roles", protect(http.HandlerFunc(handler.UserRoles)))

	// PDP management and decision endpoints are independent from the existing
	// menu/button tree. The POST policy item route also catches the colon form
	// `/policies/{id}:simulate`; the handler strips the suffix before parsing
	// the UUID. A slash alias is provided for clients that cannot emit colons in
	// route templates.
	mux.Handle("POST /v1/authorization/resources", protect(http.HandlerFunc(handler.CreateAuthorizationResource)))
	mux.Handle("GET /v1/authorization/resources", protect(http.HandlerFunc(handler.ListAuthorizationResources)))
	mux.Handle("GET /v1/authorization/resources/{id}", protect(http.HandlerFunc(handler.GetAuthorizationResource)))
	mux.Handle("PUT /v1/authorization/resources/{id}", protect(http.HandlerFunc(handler.UpdateAuthorizationResource)))
	mux.Handle("DELETE /v1/authorization/resources/{id}", protect(http.HandlerFunc(handler.DeleteAuthorizationResource)))

	mux.Handle("POST /v1/authorization/actions", protect(http.HandlerFunc(handler.CreateAuthorizationAction)))
	mux.Handle("GET /v1/authorization/actions", protect(http.HandlerFunc(handler.ListAuthorizationActions)))
	mux.Handle("GET /v1/authorization/actions/{id}", protect(http.HandlerFunc(handler.GetAuthorizationAction)))
	mux.Handle("PUT /v1/authorization/actions/{id}", protect(http.HandlerFunc(handler.UpdateAuthorizationAction)))
	mux.Handle("DELETE /v1/authorization/actions/{id}", protect(http.HandlerFunc(handler.DeleteAuthorizationAction)))

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
	mux.Handle("GET /v1/authorization/policies/{id}/bindings", protect(http.HandlerFunc(handler.GetAuthorizationPolicyBindings)))
	mux.Handle("PUT /v1/authorization/policies/{id}/bindings", protect(http.HandlerFunc(handler.ReplaceAuthorizationPolicyBindings)))
	mux.Handle("POST /v1/authorization/policies/{id}/simulate", protect(http.HandlerFunc(handler.SimulateAuthorizationPolicy)))
	// Go's ServeMux requires a wildcard to occupy an entire path segment. This
	// route therefore handles `/v1/authorization/policies/{id}:simulate` by
	// passing the suffix through to SimulateAuthorizationPolicy.
	mux.Handle("POST /v1/authorization/policies/{id}", protect(http.HandlerFunc(handler.SimulateAuthorizationPolicy)))
	mux.Handle("POST /v1/pdp/decisions", protect(http.HandlerFunc(handler.DecidePDP)))
	return mux
}
