package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// NexusAuth currently limits API resource names to 128 characters. Keeping
	// the same bound here lets the local and remote catalogs share validation.
	maxServiceResourceLength      = 128
	serviceResourceDefaultTimeout = 5 * time.Second
)

// ServiceResource is the permission namespace published by an identity
// platform. Permission Center only consumes its metadata; it does not manage
// NexusAuth resources.
type ServiceResource struct {
	ID          string    `json:"id,omitempty"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Audience    string    `json:"audience"`
	Description string    `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// ServiceResourceCatalog is deliberately read-only. It is used to validate
// permission scope and to populate the post-login scope selector; mutation is
// owned by the configured catalog provider.
type ServiceResourceCatalog interface {
	List(context.Context) ([]*ServiceResource, error)
	Get(context.Context, string) (*ServiceResource, error)
}

// LocalServiceResourceCatalog adapts the existing local applications table to
// the service-resource contract. The application identifier is the stable
// service-resource name for local deployments.
type LocalServiceResourceCatalog struct {
	applications ApplicationRepository
}

func NewLocalServiceResourceCatalog(applications ApplicationRepository) *LocalServiceResourceCatalog {
	return &LocalServiceResourceCatalog{applications: applications}
}

func (c *LocalServiceResourceCatalog) List(ctx context.Context) ([]*ServiceResource, error) {
	if c == nil || c.applications == nil {
		return nil, fmt.Errorf("service resource catalog is not configured")
	}
	values, err := c.applications.List(ctx)
	if err != nil {
		return nil, err
	}
	resources := make([]*ServiceResource, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		resources = append(resources, localServiceResource(value))
	}
	return resources, nil
}

func (c *LocalServiceResourceCatalog) Get(ctx context.Context, name string) (*ServiceResource, error) {
	if c == nil || c.applications == nil {
		return nil, fmt.Errorf("service resource catalog is not configured")
	}
	value, err := c.applications.Get(ctx, strings.TrimSpace(name))
	if err != nil {
		return nil, err
	}
	return localServiceResource(value), nil
}

func localServiceResource(value *Application) *ServiceResource {
	if value == nil {
		return nil
	}
	return &ServiceResource{
		ID:          value.Application,
		Name:        value.Application,
		DisplayName: value.Name,
		Audience:    value.Application,
		Description: value.Description,
		IsActive:    value.Enabled && !value.IsDeleted,
		CreatedAt:   value.CreatedAt,
	}
}

// NexusAuthServiceResourceCatalog reads the NexusAuth OpenAPI directory with
// a dedicated bearer credential. The credential is never returned or logged.
type NexusAuthServiceResourceCatalog struct {
	baseURL    string
	credential string
	client     *http.Client
}

func NewNexusAuthServiceResourceCatalog(baseURL, credential string, timeout time.Duration) (*NexusAuthServiceResourceCatalog, error) {
	return NewNexusAuthServiceResourceCatalogWithClient(baseURL, credential, timeout, nil)
}

func NewNexusAuthServiceResourceCatalogWithClient(baseURL, credential string, timeout time.Duration, client *http.Client) (*NexusAuthServiceResourceCatalog, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("nexusauth service-resource base URL must be an absolute http or https URL")
	}
	credential = strings.TrimSpace(credential)
	if credential == "" {
		return nil, fmt.Errorf("nexusauth service-resource credential is required")
	}
	if timeout <= 0 {
		timeout = serviceResourceDefaultTimeout
	}
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &NexusAuthServiceResourceCatalog{baseURL: baseURL, credential: credential, client: client}, nil
}

func (c *NexusAuthServiceResourceCatalog) List(ctx context.Context) ([]*ServiceResource, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("service resource catalog is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/openapi/v1/service-resources", nil)
	if err != nil {
		return nil, fmt.Errorf("create NexusAuth service-resource request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.credential)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request NexusAuth service-resource directory: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("read NexusAuth service-resource directory: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("NexusAuth service-resource directory returned HTTP %d", resp.StatusCode)
	}
	return decodeNexusAuthServiceResources(body)
}

func (c *NexusAuthServiceResourceCatalog) Get(ctx context.Context, name string) (*ServiceResource, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: service_resource is required", ErrInvalidArgument)
	}
	resources, err := c.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, resource := range resources {
		if resource != nil && resource.Name == name {
			return resource, nil
		}
	}
	return nil, ErrNotFound
}

type nexusAuthServiceResource struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Audience    string    `json:"audience"`
	Description string    `json:"description"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
}

func decodeNexusAuthServiceResources(body []byte) ([]*ServiceResource, error) {
	var values []nexusAuthServiceResource
	if err := json.Unmarshal(body, &values); err != nil {
		var envelope struct {
			Items []nexusAuthServiceResource `json:"items"`
		}
		if envelopeErr := json.Unmarshal(body, &envelope); envelopeErr != nil {
			return nil, fmt.Errorf("decode NexusAuth service-resource directory: %w", err)
		}
		values = envelope.Items
	}
	resources := make([]*ServiceResource, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(value.Name)
		if name == "" {
			return nil, fmt.Errorf("decode NexusAuth service-resource directory: resource name is empty")
		}
		resources = append(resources, &ServiceResource{
			ID: value.ID, Name: name, DisplayName: value.DisplayName, Audience: value.Audience,
			Description: value.Description, IsActive: value.IsActive, CreatedAt: value.CreatedAt,
		})
	}
	return resources, nil
}

func serviceResourceValue(serviceResource, application string) string {
	if value := strings.TrimSpace(serviceResource); value != "" {
		return value
	}
	return strings.TrimSpace(application)
}

func setServiceResourceScope(serviceResource, application string) (string, error) {
	serviceResource, application = strings.TrimSpace(serviceResource), strings.TrimSpace(application)
	if serviceResource != "" && application != "" && serviceResource != application {
		return "", fmt.Errorf("%w: application and service_resource must identify the same scope", ErrConflict)
	}
	value := serviceResourceValue(serviceResource, application)
	return validateServiceResource(value)
}

func validateServiceResource(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > maxServiceResourceLength {
		return "", fmt.Errorf("%w: service_resource must be 1-%d characters", ErrInvalidArgument, maxServiceResourceLength)
	}
	return value, nil
}
