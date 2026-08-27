package httpserver

import (
	"errors"
	"net/http"

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
	// The dashboard owns this route after the backend has established its
	// session. Keep the target stable so the frontend can finish its bootstrap.
	http.Redirect(w, r, "/auth/callback", http.StatusFound)
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
