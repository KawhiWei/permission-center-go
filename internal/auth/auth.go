// Package auth implements the browser-facing OIDC authorization-code flow.
// It intentionally has no dependency on the permission database: identity is
// established by the configured provider and authorization remains a separate
// business concern.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/luck/permission-center-go/internal/config"
	"golang.org/x/oauth2"
)

var (
	ErrDisabled        = errors.New("oidc authentication is disabled")
	ErrUnauthenticated = errors.New("authentication required")
	ErrInvalidRequest  = errors.New("invalid authentication request")
	ErrInvalidState    = errors.New("invalid authentication state")
	ErrInvalidNonce    = errors.New("invalid authentication nonce")
)

// User is the small identity projection stored in the encrypted local
// session. Access and refresh tokens are deliberately never placed in it.
type User struct {
	Subject       string `json:"sub"`
	Email         string `json:"email,omitempty"`
	Name          string `json:"name,omitempty"`
	Picture       string `json:"picture,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
}

// PublicConfig is safe to return to a browser. It excludes client secrets and
// session encryption material.
type PublicConfig struct {
	Enabled               bool     `json:"enabled"`
	Authority             string   `json:"authority,omitempty"`
	ClientID              string   `json:"clientId,omitempty"`
	RedirectURI           string   `json:"redirectUri,omitempty"`
	PostLogoutRedirectURI string   `json:"postLogoutRedirectUri,omitempty"`
	Scopes                []string `json:"scopes,omitempty"`
	Audience              string   `json:"audience,omitempty"`
}

type Service struct {
	cfg                config.OIDCConfig
	enabled            bool
	oauthConfig        *oauth2.Config
	verifier           *oidc.IDTokenVerifier
	endSessionEndpoint string
	sessions           *sessionStore
	now                func() time.Time
}

// New discovers the provider's authorization, token, issuer and JWKS
// metadata. Discovery is done at startup so a bad authority fails fast rather
// than producing an invalid login URL on the first request.
func New(ctx context.Context, cfg config.OIDCConfig) (*Service, error) {
	service := &Service{cfg: cfg, enabled: cfg.Enabled, now: time.Now}
	if !cfg.Enabled {
		return service, nil
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	sessions, err := newSessionStore(cfg)
	if err != nil {
		return nil, err
	}
	provider, err := oidc.NewProvider(ctx, cfg.Authority)
	if err != nil {
		return nil, fmt.Errorf("discover oidc provider: %w", err)
	}
	service.sessions = sessions
	service.oauthConfig = &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.RedirectURI,
		Scopes:       append([]string(nil), cfg.Scopes...),
	}
	service.verifier = provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	var metadata struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	if err := provider.Claims(&metadata); err == nil {
		service.endSessionEndpoint = metadata.EndSessionEndpoint
	}
	return service, nil
}

// NewForTest creates a service without network discovery. It is useful for
// testing cookie and middleware behavior; production code should use New.
func NewForTest(cfg config.OIDCConfig) (*Service, error) {
	sessions, err := newSessionStore(cfg)
	if err != nil {
		return nil, err
	}
	return &Service{
		cfg:      cfg,
		enabled:  cfg.Enabled,
		sessions: sessions,
		now:      time.Now,
	}, nil
}

func (s *Service) Enabled() bool { return s != nil && s.enabled }

func (s *Service) PublicConfig() PublicConfig {
	if s == nil {
		return PublicConfig{}
	}
	return PublicConfig{
		Enabled:               s.cfg.Enabled,
		Authority:             s.cfg.Authority,
		ClientID:              s.cfg.ClientID,
		RedirectURI:           s.cfg.RedirectURI,
		PostLogoutRedirectURI: s.cfg.PostLogoutRedirectURI,
		Scopes:                append([]string(nil), s.cfg.Scopes...),
		Audience:              s.cfg.Audience,
	}
}

// Login creates a one-time state/nonce/PKCE transaction and persists it in
// the encrypted session cookie before returning the provider URL.
func (s *Service) Login(w http.ResponseWriter, r *http.Request) (string, error) {
	if s == nil || !s.enabled || s.oauthConfig == nil {
		return "", ErrDisabled
	}
	state, err := randomToken(32)
	if err != nil {
		return "", err
	}
	nonce, err := randomToken(32)
	if err != nil {
		return "", err
	}
	verifier, err := randomToken(32)
	if err != nil {
		return "", err
	}
	data, err := s.sessions.load(r)
	if err != nil {
		data = &sessionData{}
	}
	data.State, data.Nonce, data.PKCEVerifier = state, nonce, verifier
	data.FlowCreatedAt = s.now().Unix()
	data.User = nil
	if err := s.sessions.save(w, data); err != nil {
		return "", err
	}
	return s.oauthConfig.AuthCodeURL(state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.S256ChallengeOption(verifier),
	), nil
}

// Callback exchanges the authorization code, verifies the ID token using the
// discovery-backed verifier, and checks the nonce stored before redirect.
func (s *Service) Callback(w http.ResponseWriter, r *http.Request) error {
	if s == nil || !s.enabled || s.oauthConfig == nil || s.verifier == nil {
		return ErrDisabled
	}
	if providerError := r.URL.Query().Get("error"); providerError != "" {
		return fmt.Errorf("%w: provider returned %s", ErrInvalidRequest, providerError)
	}
	code, state := r.URL.Query().Get("code"), r.URL.Query().Get("state")
	if code == "" || state == "" {
		return fmt.Errorf("%w: code and state are required", ErrInvalidRequest)
	}
	data, err := s.sessions.load(r)
	if err != nil || data.State == "" || data.Nonce == "" || data.PKCEVerifier == "" {
		return ErrInvalidState
	}
	if subtle.ConstantTimeCompare([]byte(state), []byte(data.State)) != 1 {
		return ErrInvalidState
	}
	if data.FlowCreatedAt == 0 || s.now().Sub(time.Unix(data.FlowCreatedAt, 0)) > flowMaxAge {
		return ErrInvalidState
	}
	// Consume the flow before making the upstream request, preventing a code
	// callback from being replayed if the browser retries the same URL.
	verifier, nonce := data.PKCEVerifier, data.Nonce
	data.State, data.Nonce, data.PKCEVerifier, data.FlowCreatedAt = "", "", "", 0
	if err := s.sessions.save(w, data); err != nil {
		return err
	}
	token, err := s.oauthConfig.Exchange(r.Context(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		return fmt.Errorf("exchange oidc authorization code: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return fmt.Errorf("%w: provider did not return an id_token", ErrInvalidRequest)
	}
	idToken, err := s.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		return fmt.Errorf("verify oidc id_token: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(nonce)) != 1 {
		return ErrInvalidNonce
	}
	var claims struct {
		Email         string `json:"email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return fmt.Errorf("decode oidc claims: %w", err)
	}
	data.User = &sessionUser{
		Subject:       idToken.Subject,
		Email:         claims.Email,
		Name:          claims.Name,
		Picture:       claims.Picture,
		EmailVerified: claims.EmailVerified,
		ExpiresAt:     idToken.Expiry.Unix(),
	}
	if data.User.Subject == "" {
		return fmt.Errorf("%w: id_token subject is empty", ErrInvalidRequest)
	}
	return s.sessions.save(w, data)
}

func (s *Service) Me(r *http.Request) (User, bool) {
	if s == nil || s.sessions == nil {
		return User{}, false
	}
	data, err := s.sessions.load(r)
	if err != nil || data.User == nil {
		return User{}, false
	}
	if data.User.ExpiresAt == 0 || !s.now().Before(time.Unix(data.User.ExpiresAt, 0)) {
		return User{}, false
	}
	return User{
		Subject:       data.User.Subject,
		Email:         data.User.Email,
		Name:          data.User.Name,
		Picture:       data.User.Picture,
		EmailVerified: data.User.EmailVerified,
	}, true
}

func (s *Service) Logout(w http.ResponseWriter, r *http.Request) (string, error) {
	if s == nil {
		return "", ErrDisabled
	}
	if s.sessions != nil {
		s.sessions.clear(w)
	}
	if !s.enabled || s.endSessionEndpoint == "" {
		return "", nil
	}
	logoutURL, err := url.Parse(s.endSessionEndpoint)
	if err != nil || logoutURL.Scheme == "" || logoutURL.Host == "" {
		return "", fmt.Errorf("invalid oidc end_session_endpoint")
	}
	query := logoutURL.Query()
	if s.cfg.PostLogoutRedirectURI != "" {
		query.Set("post_logout_redirect_uri", s.cfg.PostLogoutRedirectURI)
	}
	query.Set("client_id", s.cfg.ClientID)
	logoutURL.RawQuery = query.Encode()
	return logoutURL.String(), nil
}

// Middleware protects API handlers while leaving the identity endpoints free
// to establish and clear a session.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s == nil || !s.enabled {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := s.Me(r); !ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"authentication required"}` + "\n"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate authentication value: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
