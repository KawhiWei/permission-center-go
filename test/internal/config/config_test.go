package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadOIDCBackchannelAuthorityFromEnvironment(t *testing.T) {
	clearConfigEnv(t)
	configPath := filepath.Join(t.TempDir(), "app.yaml")
	contents := []byte(`
http:
  addr: ":8080"
database:
  url: "postgres://postgres:postgres@localhost:55433/permission_center?sslmode=disable"
service_resource:
  source: nexusauth
  nexusauth_base_url: "http://nexus-auth:5100"
  api_key: "directory-token"
oidc:
  enabled: true
  authority: "http://localhost:5100"
  backchannel_authority: "http://yaml-nexus-auth:5100"
  client_id: "permission-center-api"
  redirect_uri: "http://localhost:8080/signin-oidc"
  post_logout_redirect_uri: "http://localhost:5274/"
  scopes: ["openid"]
  session_secret: "test-session-secret-at-least-32-characters"
`)
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PERMISSION_CENTER_OIDC_BACKCHANNEL_AUTHORITY", "http://env-nexus-auth:5100")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OIDC.Authority != "http://localhost:5100" {
		t.Fatalf("public authority = %q", cfg.OIDC.Authority)
	}
	if cfg.OIDC.BackchannelAuthority != "http://env-nexus-auth:5100" {
		t.Fatalf("backchannel authority = %q", cfg.OIDC.BackchannelAuthority)
	}
}

func TestOIDCValidateBackchannelAuthority(t *testing.T) {
	cfg := validOIDCConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("missing backchannel authority should be valid: %v", err)
	}

	cfg.BackchannelAuthority = "nexus-auth:5100"
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "oidc.backchannel_authority") {
		t.Fatalf("invalid backchannel authority error = %v", err)
	}
}

func TestLoadServiceResourceFromEnvironment(t *testing.T) {
	clearConfigEnv(t)
	configPath := filepath.Join(t.TempDir(), "app.yaml")
	contents := []byte(`
http:
  addr: ":8080"
database:
  url: "postgres://postgres:postgres@localhost:55433/permission_center?sslmode=disable"
service_resource:
  source: nexusauth
  nexusauth_base_url: "http://yaml-nexus-auth:5100"
  api_key: "yaml-token"
  timeout: "1s"
oidc:
  enabled: false
`)
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PERMISSION_CENTER_SERVICE_RESOURCE_SOURCE", "NEXUSAUTH")
	t.Setenv("PERMISSION_CENTER_SERVICE_RESOURCE_NEXUSAUTH_BASE_URL", "http://nexus-auth:5100")
	t.Setenv("PERMISSION_CENTER_SERVICE_RESOURCE_API_KEY", "directory-token")
	t.Setenv("PERMISSION_CENTER_SERVICE_RESOURCE_TIMEOUT", "1500ms")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServiceResource.Source != "nexusauth" || cfg.ServiceResource.BaseURL() != "http://nexus-auth:5100" || cfg.ServiceResource.Credential() != "directory-token" {
		t.Fatalf("service-resource config = %#v", cfg.ServiceResource)
	}
	duration, err := cfg.ServiceResource.TimeoutDuration()
	if err != nil || duration != 1500*time.Millisecond {
		t.Fatalf("timeout = %v, %v", duration, err)
	}
}

func TestLoadRejectsNexusAuthServiceResourceWithoutCredential(t *testing.T) {
	clearConfigEnv(t)
	configPath := filepath.Join(t.TempDir(), "app.yaml")
	contents := []byte(`
http: {addr: ":8080"}
database: {url: "postgres://postgres:postgres@localhost:55433/permission_center?sslmode=disable"}
service_resource:
  source: nexusauth
  nexusauth_base_url: "http://nexus-auth:5100"
oidc: {enabled: false}
`)
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(configPath); err == nil || !strings.Contains(err.Error(), "credential") {
		t.Fatalf("missing credential error = %v", err)
	}
}

func TestLoadDefaultsServiceResourceToLocal(t *testing.T) {
	clearConfigEnv(t)
	configPath := filepath.Join(t.TempDir(), "app.yaml")
	contents := []byte(`
http: {addr: ":8080"}
database: {url: "postgres://postgres:postgres@localhost:55433/permission_center?sslmode=disable"}
oidc: {enabled: false}
`)
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServiceResource.Source != "local" {
		t.Fatalf("default service-resource source = %q", cfg.ServiceResource.Source)
	}
}

func TestLoadRejectsUnsupportedServiceResourceSource(t *testing.T) {
	clearConfigEnv(t)
	configPath := filepath.Join(t.TempDir(), "app.yaml")
	contents := []byte(`
http: {addr: ":8080"}
database: {url: "postgres://postgres:postgres@localhost:55433/permission_center?sslmode=disable"}
service_resource: {source: unsupported}
oidc: {enabled: false}
`)
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(configPath); err == nil || !strings.Contains(err.Error(), "service_resource.source must be local or nexusauth") {
		t.Fatalf("unsupported source error = %v", err)
	}
}

func TestLoadRejectsInvalidServiceResourceTimeoutSeconds(t *testing.T) {
	clearConfigEnv(t)
	configPath := filepath.Join(t.TempDir(), "app.yaml")
	contents := []byte(`
http: {addr: ":8080"}
database: {url: "postgres://postgres:postgres@localhost:55433/permission_center?sslmode=disable"}
service_resource:
  nexusauth_base_url: "http://nexus-auth:5100"
  api_key: "directory-token"
oidc: {enabled: false}
`)
	if err := os.WriteFile(configPath, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PERMISSION_CENTER_SERVICE_RESOURCE_TIMEOUT_SECONDS", "not-a-number")
	if _, err := Load(configPath); err == nil || !strings.Contains(err.Error(), "timeout_seconds") {
		t.Fatalf("invalid timeout_seconds error = %v", err)
	}
}

func validOIDCConfig() OIDCConfig {
	return OIDCConfig{
		Enabled:               true,
		Authority:             "http://localhost:5100",
		ClientID:              "permission-center-api",
		RedirectURI:           "http://localhost:8080/signin-oidc",
		PostLogoutRedirectURI: "http://localhost:5274/",
		Scopes:                []string{"openid"},
		SessionSecret:         "test-session-secret-at-least-32-characters",
		CookieName:            "permission_center_session",
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"PERMISSION_CENTER_DATABASE_URL",
		"PERMISSION_CENTER_HTTP_ADDR",
		"PERMISSION_CENTER_OIDC_ENABLED",
		"PERMISSION_CENTER_OIDC_AUTHORITY",
		"PERMISSION_CENTER_OIDC_BACKCHANNEL_AUTHORITY",
		"PERMISSION_CENTER_OIDC_CLIENT_ID",
		"PERMISSION_CENTER_OIDC_CLIENT_SECRET",
		"PERMISSION_CENTER_OIDC_REDIRECT_URI",
		"PERMISSION_CENTER_OIDC_POST_LOGOUT_REDIRECT_URI",
		"PERMISSION_CENTER_OIDC_SCOPES",
		"PERMISSION_CENTER_OIDC_AUDIENCE",
		"PERMISSION_CENTER_OIDC_SESSION_SECRET",
		"PERMISSION_CENTER_OIDC_COOKIE_SECURE",
		"PERMISSION_CENTER_OIDC_COOKIE_NAME",
		"PERMISSION_CENTER_SERVICE_RESOURCE_SOURCE",
		"PERMISSION_CENTER_SERVICE_RESOURCE_NEXUSAUTH_BASE_URL",
		"PERMISSION_CENTER_SERVICE_RESOURCE_NEXUSAUTH_OPENAPI_BASE_URL",
		"PERMISSION_CENTER_SERVICE_RESOURCE_API_KEY",
		"PERMISSION_CENTER_SERVICE_RESOURCE_OPEN_CREDENTIAL",
		"PERMISSION_CENTER_SERVICE_RESOURCE_TIMEOUT",
		"PERMISSION_CENTER_SERVICE_RESOURCE_TIMEOUT_SECONDS",
	} {
		t.Setenv(key, "")
	}
}
