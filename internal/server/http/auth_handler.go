package httpserver

import (
	"errors"
	"net/http"
	"strings"

	"github.com/luck/permission-center-go/internal/auth"
)

func (h *Handler) AuthConfig(w http.ResponseWriter, _ *http.Request) {
	if h.auth == nil {
		writeJSON(w, http.StatusOK, auth.PublicConfig{})
		return
	}
	writeJSON(w, http.StatusOK, h.auth.PublicConfig())
}

func (h *Handler) AuthLogin(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil {
		writeAuthError(w, auth.ErrDisabled)
		return
	}
	authorizeURL, err := h.auth.Login(w, r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"authorizeUrl": authorizeURL})
}

func (h *Handler) AuthCallback(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil {
		writeAuthError(w, auth.ErrDisabled)
		return
	}
	if err := h.auth.Callback(w, r); err != nil {
		writeAuthError(w, err)
		return
	}
	// Follow the NexusAuth Workbench BFF pattern: the backend owns the OIDC
	// callback, then returns the browser to the frontend once its session exists.
	http.Redirect(w, r, dashboardCallbackURL(h.auth), http.StatusFound)
}

func dashboardCallbackURL(authenticator *auth.Service) string {
	if authenticator == nil {
		return "/auth/callback"
	}
	frontendBase := strings.TrimRight(authenticator.PublicConfig().PostLogoutRedirectURI, "/")
	if frontendBase == "" {
		return "/auth/callback"
	}
	return frontendBase + "/auth/callback"
}

func (h *Handler) AuthMe(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil {
		writeAuthError(w, auth.ErrUnauthenticated)
		return
	}
	if !h.auth.Enabled() {
		writeJSON(w, http.StatusOK, map[string]any{
			"isAuthenticated": true,
			"user":            map[string]string{"sub": "development", "name": "开发模式"},
		})
		return
	}
	user, ok := h.auth.Me(r)
	if !ok {
		writeAuthError(w, auth.ErrUnauthenticated)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"isAuthenticated": true, "user": user})
}

func (h *Handler) AuthLogout(w http.ResponseWriter, r *http.Request) {
	if h.auth == nil {
		writeJSON(w, http.StatusOK, map[string]string{"logoutUrl": ""})
		return
	}
	logoutURL, err := h.auth.Logout(w, r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"logoutUrl": logoutURL})
}

func writeAuthError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, auth.ErrUnauthenticated):
		status = http.StatusUnauthorized
	case errors.Is(err, auth.ErrDisabled):
		status = http.StatusServiceUnavailable
	case errors.Is(err, auth.ErrInvalidRequest), errors.Is(err, auth.ErrInvalidState), errors.Is(err, auth.ErrInvalidNonce):
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
