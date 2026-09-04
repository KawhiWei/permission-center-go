// Package pdp provides a small HTTP client for business services acting as
// PDP callers/PEPs. It deliberately contains no OIDC or framework code: the
// caller owns authentication and supplies a trusted subject ID.
package pdp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var ErrDenied = errors.New("pdp authorization denied")

type Config struct {
	// Endpoint is either the permission-center base URL (for example
	// http://permission-center:8080) or the full /v1/pdp/decisions URL.
	Endpoint   string
	HTTPClient *http.Client
	// Headers can contain a service-to-service credential such as a gateway
	// signature. They are copied per request and never logged by this package.
	Headers http.Header
}

type Client struct {
	decisionURL string
	httpClient  *http.Client
	headers     http.Header
}

// NewClient is a convenient base-URL constructor. For validation errors or
// custom headers use NewClientWithConfig.
func NewClient(endpoint string) *Client {
	client, err := NewClientWithConfig(Config{Endpoint: endpoint})
	if err != nil {
		return &Client{decisionURL: strings.TrimRight(endpoint, "/") + "/v1/pdp/decisions", httpClient: http.DefaultClient}
	}
	return client
}

// New is kept as a terse constructor for callers that prefer the package name
// to carry the PDP context.
func New(endpoint string) *Client {
	return NewClient(endpoint)
}

func NewWithConfig(config Config) (*Client, error) {
	return NewClientWithConfig(config)
}

func NewClientWithConfig(config Config) (*Client, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(config.Endpoint), "/")
	if endpoint == "" {
		return nil, errors.New("pdp endpoint is required")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("pdp endpoint must be an absolute http or https URL")
	}
	if !strings.HasSuffix(parsed.Path, "/v1/pdp/decisions") {
		endpoint += "/v1/pdp/decisions"
	}
	client := config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{decisionURL: endpoint, httpClient: client, headers: cloneHeaders(config.Headers)}, nil
}

type DecisionRequest struct {
	Application  string `json:"application"`
	SubjectID    string `json:"subject_id"`
	ResourceCode string `json:"resource_code"`
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`
	Action       string `json:"action"`
	ServiceCode  string `json:"service_code,omitempty"`
	Method       string `json:"method,omitempty"`
	PathTemplate string `json:"path_template,omitempty"`
}

type Decision struct {
	Allow            bool     `json:"allow"`
	ReasonCode       string   `json:"reasonCode"`
	MatchedPolicyIDs []string `json:"matchedPolicyIds"`
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e == nil {
		return "pdp api error"
	}
	if e.Message == "" {
		return fmt.Sprintf("pdp api error: %s", e.Code)
	}
	return fmt.Sprintf("pdp api error (%s): %s", e.Code, e.Message)
}

// Decide sends one authorization request and returns a decision even when the
// decision is deny. Transport, malformed response, and non-success API
// envelope errors are returned as Go errors.
func (c *Client) Decide(ctx context.Context, request DecisionRequest) (Decision, error) {
	var decision Decision
	if c == nil || strings.TrimSpace(c.decisionURL) == "" {
		return decision, errors.New("pdp client is not configured")
	}
	body, err := json.Marshal(request)
	if err != nil {
		return decision, fmt.Errorf("encode pdp decision request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.decisionURL, bytes.NewReader(body))
	if err != nil {
		return decision, fmt.Errorf("create pdp decision request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	for key, values := range c.headers {
		for _, value := range values {
			httpRequest.Header.Add(key, value)
		}
	}
	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return decision, fmt.Errorf("call pdp decision endpoint: %w", err)
	}
	defer response.Body.Close()
	var envelope responseEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return decision, fmt.Errorf("decode pdp decision response: %w", err)
	}
	if !envelope.Success || response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		code, message := envelope.ErrorCode, envelope.ErrorMessage
		if code == "" {
			code = http.StatusText(response.StatusCode)
		}
		return decision, &APIError{StatusCode: response.StatusCode, Code: code, Message: message}
	}
	if len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return decision, errors.New("pdp decision response has no result")
	}
	if err := json.Unmarshal(envelope.Result, &decision); err != nil {
		return decision, fmt.Errorf("decode pdp decision result: %w", err)
	}
	if decision.MatchedPolicyIDs == nil {
		decision.MatchedPolicyIDs = []string{}
	}
	return decision, nil
}

func (c *Client) Check(ctx context.Context, request DecisionRequest) (Decision, error) {
	return c.Decide(ctx, request)
}

// AuthorizeHTTP is a compact PEP helper for a route-level check. Callers that
// need entity attributes should use Decide and make a second entity decision
// after reading the instance from their own database.
func (c *Client) AuthorizeHTTP(ctx context.Context, subjectID, application, resourceCode, action string) (Decision, error) {
	decision, err := c.Decide(ctx, DecisionRequest{
		Application: application, SubjectID: subjectID, ResourceCode: resourceCode, Action: action,
	})
	if err != nil {
		return decision, err
	}
	if !decision.Allow {
		return decision, fmt.Errorf("%w: %s", ErrDenied, decision.ReasonCode)
	}
	return decision, nil
}

type responseEnvelope struct {
	Success      bool            `json:"success"`
	ErrorCode    string          `json:"errorCode"`
	ErrorMessage string          `json:"errorMessage"`
	Result       json.RawMessage `json:"result"`
}

func cloneHeaders(headers http.Header) http.Header {
	if headers == nil {
		return make(http.Header)
	}
	return headers.Clone()
}
