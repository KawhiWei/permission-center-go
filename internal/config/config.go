package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTP                   HTTPConfig                   `yaml:"http"`
	Database               DatabaseConfig               `yaml:"database"`
	OIDC                   OIDCConfig                   `yaml:"oidc"`
	ApplicationCatalog     ApplicationCatalogConfig     `yaml:"application_catalog"`
	ServiceResourceCatalog ServiceResourceCatalogConfig `yaml:"service_resource_catalog"`
}
type ApplicationCatalogConfig struct {
	Source string `yaml:"source"`
}

// ServiceResourceCatalogConfig controls where permission scope names are
// resolved. The local provider adapts the existing applications table; the
// NexusAuth provider only reads its OpenAPI service-resource directory.
type ServiceResourceCatalogConfig struct {
	Source                  string `yaml:"source"`
	NexusAuthBaseURL        string `yaml:"nexusauth_base_url"`
	NexusAuthOpenAPIBaseURL string `yaml:"nexusauth_openapi_base_url"`
	APIKey                  string `yaml:"api_key"`
	OpenCredential          string `yaml:"open_credential"`
	Timeout                 string `yaml:"timeout"`
	TimeoutSeconds          int    `yaml:"timeout_seconds"`
}

func (c ServiceResourceCatalogConfig) BaseURL() string {
	if strings.TrimSpace(c.NexusAuthOpenAPIBaseURL) != "" {
		return strings.TrimSpace(c.NexusAuthOpenAPIBaseURL)
	}
	return strings.TrimSpace(c.NexusAuthBaseURL)
}

func (c ServiceResourceCatalogConfig) Credential() string {
	if strings.TrimSpace(c.OpenCredential) != "" {
		return strings.TrimSpace(c.OpenCredential)
	}
	return strings.TrimSpace(c.APIKey)
}

func (c ServiceResourceCatalogConfig) TimeoutDuration() (time.Duration, error) {
	if c.TimeoutSeconds > 0 {
		return time.Duration(c.TimeoutSeconds) * time.Second, nil
	}
	if strings.TrimSpace(c.Timeout) == "" {
		return 5 * time.Second, nil
	}
	duration, err := time.ParseDuration(strings.TrimSpace(c.Timeout))
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("service_resource_catalog.timeout must be a positive duration")
	}
	return duration, nil
}

type HTTPConfig struct {
	Addr string `yaml:"addr"`
}
type DatabaseConfig struct {
	URL      string `yaml:"url"`
	MaxConns int32  `yaml:"max_conns"`
	MinConns int32  `yaml:"min_conns"`
}

// OIDCConfig controls the browser login flow. The service does not create an
// identity provider; it discovers the provider endpoints from Authority.
type OIDCConfig struct {
	Enabled               bool     `yaml:"enabled"`
	Authority             string   `yaml:"authority"`
	BackchannelAuthority  string   `yaml:"backchannel_authority"`
	ClientID              string   `yaml:"client_id"`
	ClientSecret          string   `yaml:"client_secret"`
	RedirectURI           string   `yaml:"redirect_uri"`
	PostLogoutRedirectURI string   `yaml:"post_logout_redirect_uri"`
	Scopes                []string `yaml:"scopes"`
	Audience              string   `yaml:"audience"`
	SessionSecret         string   `yaml:"session_secret"`
	CookieSecure          bool     `yaml:"cookie_secure"`
	CookieName            string   `yaml:"cookie_name"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if value := os.Getenv("PERMISSION_CENTER_DATABASE_URL"); value != "" {
		cfg.Database.URL = value
	}
	if value := os.Getenv("PERMISSION_CENTER_HTTP_ADDR"); value != "" {
		cfg.HTTP.Addr = value
	}
	applyOIDCEnv(&cfg.OIDC)
	if value := os.Getenv("PERMISSION_CENTER_APPLICATION_CATALOG_SOURCE"); value != "" {
		cfg.ApplicationCatalog.Source = value
	}
	if cfg.ApplicationCatalog.Source == "" {
		cfg.ApplicationCatalog.Source = "local"
	}
	if cfg.ApplicationCatalog.Source != "local" && cfg.ApplicationCatalog.Source != "nexusauth" {
		return nil, fmt.Errorf("application_catalog.source must be local or nexusauth")
	}
	if err := applyServiceResourceCatalogEnv(&cfg.ServiceResourceCatalog); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.ServiceResourceCatalog.Source) == "" {
		cfg.ServiceResourceCatalog.Source = cfg.ApplicationCatalog.Source
	}
	if cfg.ServiceResourceCatalog.Source == "" {
		cfg.ServiceResourceCatalog.Source = "local"
	}
	cfg.ServiceResourceCatalog.Source = strings.ToLower(strings.TrimSpace(cfg.ServiceResourceCatalog.Source))
	if cfg.ServiceResourceCatalog.Source != "local" && cfg.ServiceResourceCatalog.Source != "nexusauth" {
		return nil, fmt.Errorf("service_resource_catalog.source must be local or nexusauth")
	}
	if _, err := cfg.ServiceResourceCatalog.TimeoutDuration(); err != nil {
		return nil, err
	}
	if cfg.ServiceResourceCatalog.Source == "nexusauth" {
		if err := validateHTTPURL(cfg.ServiceResourceCatalog.BaseURL(), "service_resource_catalog.nexusauth_base_url"); err != nil {
			return nil, err
		}
		if cfg.ServiceResourceCatalog.Credential() == "" {
			return nil, fmt.Errorf("service_resource_catalog.api_key or service_resource_catalog.open_credential is required when source is nexusauth")
		}
	}
	if len(cfg.OIDC.Scopes) == 0 {
		cfg.OIDC.Scopes = []string{"openid", "profile", "email"}
	}
	if cfg.OIDC.CookieName == "" {
		cfg.OIDC.CookieName = "permission_center_session"
	}
	if cfg.HTTP.Addr == "" || cfg.Database.URL == "" {
		return nil, fmt.Errorf("http.addr and database.url are required")
	}
	if err := cfg.OIDC.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyServiceResourceCatalogEnv(cfg *ServiceResourceCatalogConfig) error {
	if value := os.Getenv("PERMISSION_CENTER_SERVICE_RESOURCE_CATALOG_SOURCE"); value != "" {
		cfg.Source = value
	}
	if value := os.Getenv("PERMISSION_CENTER_SERVICE_RESOURCE_CATALOG_NEXUSAUTH_BASE_URL"); value != "" {
		cfg.NexusAuthBaseURL = value
	}
	if value := os.Getenv("PERMISSION_CENTER_SERVICE_RESOURCE_CATALOG_NEXUSAUTH_OPENAPI_BASE_URL"); value != "" {
		cfg.NexusAuthOpenAPIBaseURL = value
	}
	if value := os.Getenv("PERMISSION_CENTER_SERVICE_RESOURCE_CATALOG_API_KEY"); value != "" {
		cfg.APIKey = value
	}
	if value := os.Getenv("PERMISSION_CENTER_SERVICE_RESOURCE_CATALOG_OPEN_CREDENTIAL"); value != "" {
		cfg.OpenCredential = value
	}
	if value := os.Getenv("PERMISSION_CENTER_SERVICE_RESOURCE_CATALOG_TIMEOUT"); value != "" {
		cfg.Timeout = value
	}
	if value := os.Getenv("PERMISSION_CENTER_SERVICE_RESOURCE_CATALOG_TIMEOUT_SECONDS"); value != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || parsed <= 0 {
			return fmt.Errorf("service_resource_catalog.timeout_seconds must be a positive integer")
		}
		cfg.TimeoutSeconds = parsed
	}
	return nil
}

func applyOIDCEnv(cfg *OIDCConfig) {
	if value := os.Getenv("PERMISSION_CENTER_OIDC_ENABLED"); value != "" {
		cfg.Enabled = strings.EqualFold(value, "true") || value == "1"
	}
	if value := os.Getenv("PERMISSION_CENTER_OIDC_AUTHORITY"); value != "" {
		cfg.Authority = value
	}
	if value := os.Getenv("PERMISSION_CENTER_OIDC_BACKCHANNEL_AUTHORITY"); value != "" {
		cfg.BackchannelAuthority = value
	}
	if value := os.Getenv("PERMISSION_CENTER_OIDC_CLIENT_ID"); value != "" {
		cfg.ClientID = value
	}
	if value := os.Getenv("PERMISSION_CENTER_OIDC_CLIENT_SECRET"); value != "" {
		cfg.ClientSecret = value
	}
	if value := os.Getenv("PERMISSION_CENTER_OIDC_REDIRECT_URI"); value != "" {
		cfg.RedirectURI = value
	}
	if value := os.Getenv("PERMISSION_CENTER_OIDC_POST_LOGOUT_REDIRECT_URI"); value != "" {
		cfg.PostLogoutRedirectURI = value
	}
	if value := os.Getenv("PERMISSION_CENTER_OIDC_SCOPES"); value != "" {
		cfg.Scopes = strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' })
	}
	if value := os.Getenv("PERMISSION_CENTER_OIDC_AUDIENCE"); value != "" {
		cfg.Audience = value
	}
	if value := os.Getenv("PERMISSION_CENTER_OIDC_SESSION_SECRET"); value != "" {
		cfg.SessionSecret = value
	}
	if value := os.Getenv("PERMISSION_CENTER_OIDC_COOKIE_SECURE"); value != "" {
		cfg.CookieSecure = strings.EqualFold(value, "true") || value == "1"
	}
	if value := os.Getenv("PERMISSION_CENTER_OIDC_COOKIE_NAME"); value != "" {
		cfg.CookieName = value
	}
}

func (c OIDCConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.Authority) == "" || strings.TrimSpace(c.ClientID) == "" {
		return fmt.Errorf("oidc.authority and oidc.client_id are required when oidc is enabled")
	}
	if err := validateHTTPURL(c.Authority, "oidc.authority"); err != nil {
		return err
	}
	if strings.TrimSpace(c.BackchannelAuthority) != "" {
		if err := validateHTTPURL(c.BackchannelAuthority, "oidc.backchannel_authority"); err != nil {
			return err
		}
	}
	if err := validateHTTPURL(c.RedirectURI, "oidc.redirect_uri"); err != nil {
		return err
	}
	if err := validateHTTPURL(c.PostLogoutRedirectURI, "oidc.post_logout_redirect_uri"); err != nil {
		return err
	}
	if len(c.SessionSecret) < 32 || strings.HasPrefix(c.SessionSecret, "replace-with-") {
		return fmt.Errorf("oidc.session_secret must be at least 32 characters when oidc is enabled")
	}
	if strings.TrimSpace(c.CookieName) == "" || strings.ContainsAny(c.CookieName, " \t\r\n;") {
		return fmt.Errorf("oidc.cookie_name must be a valid cookie name")
	}
	if len(c.Scopes) == 0 {
		return fmt.Errorf("oidc.scopes must include at least one scope")
	}
	seen := make(map[string]struct{}, len(c.Scopes))
	for _, scope := range c.Scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			return fmt.Errorf("oidc.scopes must not contain empty values")
		}
		seen[scope] = struct{}{}
	}
	if _, ok := seen["openid"]; !ok {
		return fmt.Errorf("oidc.scopes must include openid")
	}
	return nil
}

func validateHTTPURL(value, field string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s must be an absolute http or https URL", field)
	}
	return nil
}
