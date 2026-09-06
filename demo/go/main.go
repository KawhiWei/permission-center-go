package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
)

type config struct {
	addr              string
	permissionURL     string
	pdpCredential     string
	serviceResource   string
	endpointID        string
	oidcIssuer        string
	oidcAudience      string
	tenantClaim       string
	attributeClaims   []string
	pdpRequestTimeout time.Duration
}

type identity struct {
	subjectID string
	tenantID  string
	attrs     map[string]any
}

type identityVerifier interface {
	Verify(context.Context, string) (identity, error)
}

type oidcIdentityVerifier struct {
	verifier        *oidc.IDTokenVerifier
	tenantClaim     string
	attributeClaims []string
}

type pdpHTTPClient struct {
	url             string
	credential      string
	serviceResource string
	endpointID      string
	httpClient      *http.Client
}

// decisionRequest 是直接调用权限中心 HTTP API 的请求体，不依赖任何 SDK。
type decisionRequest struct {
	RequestID         string           `json:"request_id,omitempty"`
	AuthorizationType string           `json:"authorization_type"`
	ServiceResource   string           `json:"service_resource"`
	TenantID          string           `json:"tenant_id"`
	Subject           decisionSubject  `json:"subject"`
	Action            decisionAction   `json:"action"`
	Resource          decisionResource `json:"resource"`
}

type decisionSubject struct {
	ID         string         `json:"id"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type decisionAction struct {
	Kind   string `json:"kind"`
	Method string `json:"method"`
}

type decisionResource struct {
	Kind       string `json:"kind"`
	EndpointID string `json:"endpoint_id"`
}

type decisionResponse struct {
	Decision   string `json:"decision"`
	ReasonCode string `json:"reason_code"`
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	provider, err := oidc.NewProvider(ctx, cfg.oidcIssuer)
	if err != nil {
		log.Fatalf("discover OIDC provider: %v", err)
	}
	verifier := &oidcIdentityVerifier{
		verifier:        provider.Verifier(&oidc.Config{ClientID: cfg.oidcAudience}),
		tenantClaim:     cfg.tenantClaim,
		attributeClaims: cfg.attributeClaims,
	}
	pdp := &pdpHTTPClient{
		url:             strings.TrimRight(cfg.permissionURL, "/") + "/v1/pdp/decisions",
		credential:      cfg.pdpCredential,
		serviceResource: cfg.serviceResource,
		endpointID:      cfg.endpointID,
		httpClient:      &http.Client{Timeout: cfg.pdpRequestTimeout},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /openapi.json", serveOpenAPI)
	mux.Handle("GET /api/orders/{id}", authorize(verifier, pdp, http.HandlerFunc(getOrder)))

	server := &http.Server{Addr: cfg.addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("demo business API listening on %s", cfg.addr)
	log.Fatal(server.ListenAndServe())
}

func loadConfig() (config, error) {
	timeout, err := time.ParseDuration(env("DEMO_PDP_TIMEOUT", "2s"))
	if err != nil || timeout <= 0 {
		return config{}, errors.New("DEMO_PDP_TIMEOUT must be a positive duration")
	}
	cfg := config{
		addr:              env("DEMO_ADDR", ":8090"),
		permissionURL:     strings.TrimSpace(os.Getenv("DEMO_PERMISSION_CENTER_URL")),
		pdpCredential:     strings.TrimSpace(os.Getenv("DEMO_PDP_SERVICE_CREDENTIAL")),
		serviceResource:   strings.TrimSpace(os.Getenv("DEMO_SERVICE_RESOURCE")),
		endpointID:        strings.TrimSpace(os.Getenv("DEMO_ENDPOINT_ID")),
		oidcIssuer:        strings.TrimSpace(os.Getenv("DEMO_OIDC_ISSUER")),
		oidcAudience:      strings.TrimSpace(os.Getenv("DEMO_OIDC_AUDIENCE")),
		tenantClaim:       env("DEMO_TENANT_CLAIM", "tenant_id"),
		attributeClaims:   splitCSV(env("DEMO_SUBJECT_ATTRIBUTE_CLAIMS", "department,account_status")),
		pdpRequestTimeout: timeout,
	}
	if cfg.permissionURL == "" || cfg.pdpCredential == "" || cfg.serviceResource == "" || cfg.oidcIssuer == "" || cfg.oidcAudience == "" {
		return config{}, errors.New("DEMO_PERMISSION_CENTER_URL, DEMO_PDP_SERVICE_CREDENTIAL, DEMO_SERVICE_RESOURCE, DEMO_OIDC_ISSUER and DEMO_OIDC_AUDIENCE are required")
	}
	parsedURL, err := url.Parse(cfg.permissionURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return config{}, errors.New("DEMO_PERMISSION_CENTER_URL must be an absolute HTTP(S) URL")
	}
	if _, err = uuid.Parse(cfg.endpointID); err != nil {
		return config{}, errors.New("DEMO_ENDPOINT_ID must be the endpoint UUID returned by permission center")
	}
	return cfg, nil
}

func (v *oidcIdentityVerifier) Verify(ctx context.Context, rawToken string) (identity, error) {
	// OIDC verifier 会校验签名、issuer、audience 和有效期，成功后 claims 才能作为可信身份事实。
	token, err := v.verifier.Verify(ctx, rawToken)
	if err != nil {
		return identity{}, fmt.Errorf("verify token: %w", err)
	}
	var claims map[string]any
	if err = token.Claims(&claims); err != nil {
		return identity{}, fmt.Errorf("decode claims: %w", err)
	}
	tenantID, ok := claims[v.tenantClaim].(string)
	if token.Subject == "" || !ok || strings.TrimSpace(tenantID) == "" {
		return identity{}, fmt.Errorf("verified token must contain sub and string claim %q", v.tenantClaim)
	}
	attrs := make(map[string]any, len(v.attributeClaims))
	// attributes 是可选的：只有策略引用的可信 claim 存在时才放入 PDP 请求。
	for _, name := range v.attributeClaims {
		if value, exists := claims[name]; exists {
			attrs[name] = value
		}
	}
	return identity{subjectID: token.Subject, tenantID: strings.TrimSpace(tenantID), attrs: attrs}, nil
}

func (c *pdpHTTPClient) Decide(ctx context.Context, id identity, method, requestID string) (decisionResponse, error) {
	// 当前受保护路由做 API 级鉴权，因此显式发送 api；数据级路由应固定发送 data。
	payload := decisionRequest{
		RequestID: requestID, AuthorizationType: "api", ServiceResource: c.serviceResource, TenantID: id.tenantID,
		Subject:  decisionSubject{ID: id.subjectID, Attributes: id.attrs},
		Action:   decisionAction{Kind: "http", Method: method},
		Resource: decisionResource{Kind: "api_endpoint", EndpointID: c.endpointID},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return decisionResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return decisionResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.credential)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return decisionResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return decisionResponse{}, fmt.Errorf("permission center returned HTTP %d", resp.StatusCode)
	}
	var result decisionResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return decisionResponse{}, err
	}
	if result.Decision != "allow" && result.Decision != "deny" {
		return decisionResponse{}, errors.New("permission center returned an invalid decision")
	}
	return result, nil
}

func authorize(verifier identityVerifier, pdp *pdpHTTPClient, next http.Handler) http.Handler {
	// 中间件先验证用户身份，再询问 PDP；只有明确 allow 才执行真实业务接口。
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawToken, err := bearerToken(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		id, err := verifier.Verify(r.Context(), rawToken)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = newRequestID()
		}
		result, err := pdp.Decide(r.Context(), id, r.Method, requestID)
		if err != nil {
			log.Printf("PDP request %s failed: %v", requestID, err)
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		if result.Decision != "allow" {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

func getOrder(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"id": r.PathValue("id"), "number": "ORD-2026-0001", "status": "paid",
	})
}

func serveOpenAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(openAPIDocument))
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("Bearer token is required")
	}
	return parts[1], nil
}

func newRequestID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("request-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(value[:])
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	result := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

const openAPIDocument = `{
  "openapi": "3.0.3",
  "info": {"title": "Permission Center HTTP Demo", "version": "1.0.0"},
  "paths": {
    "/api/orders/{id}": {
      "get": {
        "tags": ["Orders"],
        "summary": "Get an order",
        "parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "string"}}],
        "responses": {"200": {"description": "Order"}, "401": {"description": "Unauthenticated"}, "403": {"description": "Forbidden"}}
      }
    }
  }
}`
