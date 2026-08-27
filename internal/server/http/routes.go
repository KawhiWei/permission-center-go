package httpserver

import "net/http"

func NewServer(handler *Handler, authMiddleware ...func(http.Handler) http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /auth/config", handler.AuthConfig)
	mux.HandleFunc("GET /auth/login", handler.AuthLogin)
	mux.HandleFunc("GET /auth/callback", handler.AuthCallback)
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
	mux.Handle("GET /v1/roles", protect(http.HandlerFunc(handler.ListRoles)))
	mux.Handle("POST /v1/resources", protect(http.HandlerFunc(handler.CreateResource)))
	mux.Handle("GET /v1/resources/tree", protect(http.HandlerFunc(handler.ResourceTree)))
	mux.Handle("PUT /v1/roles/{roleID}/resources", protect(http.HandlerFunc(handler.GrantRoleResources)))
	mux.Handle("GET /v1/roles/{roleID}/resources", protect(http.HandlerFunc(handler.RoleResources)))
	mux.Handle("PUT /v1/users/{userID}/roles", protect(http.HandlerFunc(handler.ReplaceUserRoles)))
	mux.Handle("GET /v1/users/{userID}/roles", protect(http.HandlerFunc(handler.UserRoles)))
	return mux
}
