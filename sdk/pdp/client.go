// Package pdp 提供业务服务（PEP）调用权限中心 PDP 的客户端契约。
package pdp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Decision string

const (
	DecisionAllow Decision = "allow"
	DecisionDeny  Decision = "deny"
)

type Input struct {
	RequestID         string         `json:"request_id,omitempty"`
	AuthorizationType string         `json:"authorization_type"`
	ServiceResource   string         `json:"service_resource"`
	TenantID          string         `json:"tenant_id"`
	Subject           Subject        `json:"subject"`
	Action            Action         `json:"action"`
	Resource          Resource       `json:"resource"`
	Context           map[string]any `json:"context,omitempty"`
}

type Subject struct {
	ID         string         `json:"id"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type Action struct {
	Kind   string `json:"kind"`
	Method string `json:"method,omitempty"`
}

type Resource struct {
	Kind       string         `json:"kind"`
	EndpointID string         `json:"endpoint_id"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type Result struct {
	Decision         Decision `json:"decision"`
	ReasonCode       string   `json:"reason_code"`
	MatchedPolicyIDs []string `json:"matched_policy_ids"`
	SnapshotVersion  int      `json:"snapshot_version"`
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	credential string
}

func NewClient(baseURL, credential string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("pdp base URL is invalid")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 2 * time.Second}
	}
	return &Client{baseURL: parsed, httpClient: httpClient, credential: strings.TrimSpace(credential)}, nil
}

func (c *Client) Decide(ctx context.Context, input Input) (Result, error) {
	// 鉴权类型必须由业务路由明确指定，禁止用缺省值掩盖接入错误。
	if input.AuthorizationType != "api" && input.AuthorizationType != "data" || strings.TrimSpace(input.ServiceResource) == "" || strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.Subject.ID) == "" || input.Action.Kind != "http" || input.Resource.Kind != "api_endpoint" || strings.TrimSpace(input.Resource.EndpointID) == "" {
		return Result{}, fmt.Errorf("invalid PDP decision input")
	}
	body, err := json.Marshal(input)
	if err != nil {
		return Result{}, err
	}
	target := c.baseURL.ResolveReference(&url.URL{Path: "/v1/pdp/decisions"})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	if c.credential != "" {
		request.Header.Set("Authorization", "Bearer "+c.credential)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return Result{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("PDP returned HTTP %d", response.StatusCode)
	}
	var result Result
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return Result{}, err
	}
	if result.Decision != DecisionAllow && result.Decision != DecisionDeny {
		return Result{}, fmt.Errorf("PDP returned invalid decision")
	}
	return result, nil
}
