package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTP               HTTPConfig               `yaml:"http"`
	Database           DatabaseConfig           `yaml:"database"`
	OIDC               OIDCConfig               `yaml:"oidc"`
	ApplicationCatalog ApplicationCatalogConfig `yaml:"application_catalog"`
}
type ApplicationCatalogConfig struct {
	Source string `yaml:"source"`
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
