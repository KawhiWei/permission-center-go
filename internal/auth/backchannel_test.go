package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/luck/permission-center-go/internal/config"
)

func TestNewUsesBackchannelWithoutChangingBrowserEndpoints(t *testing.T) {
	const publicAuthority = "http://localhost:5100"
	const backchannelAuthority = "http://nexusauth.internal:5100"

	var discoveryRequests int
	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != backchannelAuthority+"/.well-known/openid-configuration" {
			t.Fatalf("unexpected upstream URL %q", r.URL.String())
		}
		discoveryRequests++
		body, err := json.Marshal(map[string]any{
			"issuer":                                publicAuthority,
			"authorization_endpoint":                publicAuthority + "/connect/authorize",
			"token_endpoint":                        publicAuthority + "/connect/token",
			"jwks_uri":                              publicAuthority + "/.well-known/openid-configuration/jwks",
			"end_session_endpoint":                  publicAuthority + "/connect/endsession",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(string(body))),
			Request:    r,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	cfg := config.OIDCConfig{
		Enabled:               true,
		Authority:             publicAuthority,
		BackchannelAuthority:  backchannelAuthority,
		ClientID:              "permission-center-api",
		ClientSecret:          "test-client-secret",
		RedirectURI:           "http://localhost:8080/signin-oidc",
		PostLogoutRedirectURI: "http://localhost:5274/",
		Scopes:                []string{"openid", "profile"},
		SessionSecret:         "test-session-secret-at-least-32-characters",
		CookieName:            "permission_center_session",
	}
	service, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if discoveryRequests != 1 {
		t.Fatalf("discovery requests = %d, want 1", discoveryRequests)
	}
	if service.oauthConfig.Endpoint.AuthURL != publicAuthority+"/connect/authorize" {
		t.Fatalf("authorization endpoint = %q", service.oauthConfig.Endpoint.AuthURL)
	}
	if service.endSessionEndpoint != publicAuthority+"/connect/endsession" {
		t.Fatalf("end-session endpoint = %q", service.endSessionEndpoint)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	authorizeURL, err := service.Login(recorder, request)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme+"://"+parsed.Host != publicAuthority {
		t.Fatalf("browser authority = %q", parsed.Scheme+"://"+parsed.Host)
	}
	if !strings.HasSuffix(parsed.Path, "/connect/authorize") {
		t.Fatalf("authorization path = %q", parsed.Path)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestAuthorityRoutingTransportOnlyRewritesProviderURLs(t *testing.T) {
	public, _ := url.Parse("http://localhost:5100/issuer")
	backchannel, _ := url.Parse("http://nexusauth:8080/internal")

	rewritten, ok := rewriteAuthorityURL(
		"http://localhost:5100/issuer/connect/token?mode=code",
		public,
		backchannel,
	)
	if !ok || rewritten != "http://nexusauth:8080/internal/connect/token?mode=code" {
		t.Fatalf("rewritten endpoint = %q, ok = %v", rewritten, ok)
	}
	if _, ok := rewriteAuthorityURL("http://localhost:5100/other/token", public, backchannel); ok {
		t.Fatal("URL outside the issuer path must not be rewritten")
	}
}
