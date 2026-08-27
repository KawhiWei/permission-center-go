package auth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/luck/permission-center-go/internal/config"
)

func testConfig(enabled bool) config.OIDCConfig {
	return config.OIDCConfig{
		Enabled:       enabled,
		SessionSecret: "test-session-secret-at-least-32-characters",
		CookieName:    "permission_test_session",
	}
}

func TestSessionCookieRoundTripAndTamperRejection(t *testing.T) {
	store, err := newSessionStore(testConfig(true))
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	if err := store.save(recorder, &sessionData{User: &sessionUser{Subject: "user-1", Name: "Alice", ExpiresAt: time.Now().Add(time.Hour).Unix()}}); err != nil {
		t.Fatal(err)
	}
	cookie := recorder.Result().Cookies()[0]
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(cookie)
	data, err := store.load(request)
	if err != nil || data.User == nil || data.User.Subject != "user-1" {
		t.Fatalf("loaded session = %#v, err = %v", data, err)
	}
	cookie.Value += "tampered"
	tampered := httptest.NewRequest(http.MethodGet, "/", nil)
	tampered.AddCookie(cookie)
	if _, err := store.load(tampered); err == nil {
		t.Fatal("expected a tampered cookie to be rejected")
	}
}

func TestMiddlewareRequiresSessionOnlyWhenEnabled(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	disabled, err := NewForTest(testConfig(false))
	if err != nil {
		t.Fatal(err)
	}
	disabledResponse := httptest.NewRecorder()
	disabled.Middleware(next).ServeHTTP(disabledResponse, httptest.NewRequest(http.MethodGet, "/v1/roles", nil))
	if disabledResponse.Code != http.StatusNoContent {
		t.Fatalf("disabled auth status = %d", disabledResponse.Code)
	}

	enabled, err := NewForTest(testConfig(true))
	if err != nil {
		t.Fatal(err)
	}
	enabledResponse := httptest.NewRecorder()
	enabled.Middleware(next).ServeHTTP(enabledResponse, httptest.NewRequest(http.MethodGet, "/v1/roles", nil))
	if enabledResponse.Code != http.StatusUnauthorized {
		t.Fatalf("enabled auth status = %d", enabledResponse.Code)
	}
}

func TestLogoutUsesNexusAuthIDTokenHint(t *testing.T) {
	service, err := NewForTest(testConfig(true))
	if err != nil {
		t.Fatal(err)
	}
	service.endSessionEndpoint = "http://localhost:5100/connect/endsession"
	service.cfg.ClientID = "permission-center-web"
	service.cfg.PostLogoutRedirectURI = "http://localhost:5274/"

	loginRecorder := httptest.NewRecorder()
	if err := service.sessions.save(loginRecorder, &sessionData{User: &sessionUser{
		Subject: "user-1", IDToken: "signed-id-token", ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.AddCookie(loginRecorder.Result().Cookies()[0])
	logoutRecorder := httptest.NewRecorder()
	logoutURL, err := service.Logout(logoutRecorder, request)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(logoutURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("id_token_hint") != "signed-id-token" {
		t.Fatalf("id_token_hint = %q", parsed.Query().Get("id_token_hint"))
	}
	if parsed.Query().Get("post_logout_redirect_uri") != "http://localhost:5274/" {
		t.Fatalf("post_logout_redirect_uri = %q", parsed.Query().Get("post_logout_redirect_uri"))
	}
	if len(logoutRecorder.Result().Cookies()) != 1 || logoutRecorder.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("logout cookie = %#v", logoutRecorder.Result().Cookies())
	}
}
