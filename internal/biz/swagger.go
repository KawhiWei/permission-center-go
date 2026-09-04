package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const maxSwaggerDocumentSize = 8 << 20

var swaggerUIURLPattern = regexp.MustCompile(`(?i)(?:["']?url["']?)\s*[:=]\s*["']([^"']+)["']`)
var swaggerUIConfigURLPattern = regexp.MustCompile(`(?i)(?:["']?configUrl["']?)\s*[:=]\s*["']([^"']+)["']`)
var swaggerInitializerPattern = regexp.MustCompile(`(?i)<script[^>]+src=["']([^"']*swagger-initializer[^"']*)["']`)
var swaggerUIScriptPattern = regexp.MustCompile(`(?i)<script[^>]+src=["']([^"']+\.js(?:\?[^"']*)?)["']`)

type SwaggerImportRequest struct {
	ServiceResource string
	// Application is accepted for callers using the pre-service-resource API.
	Application string
	SwaggerURL  string
}

type SwaggerImportResult struct {
	Total   int `json:"total"`
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

type swaggerDocument struct {
	Swagger  string `json:"swagger" yaml:"swagger"`
	OpenAPI  string `json:"openapi" yaml:"openapi"`
	BasePath string `json:"basePath" yaml:"basePath"`
	Servers  []struct {
		URL string `json:"url" yaml:"url"`
	} `json:"servers" yaml:"servers"`
	Info struct {
		Title string `json:"title" yaml:"title"`
	} `json:"info" yaml:"info"`
	Paths map[string]swaggerPathItem `json:"paths" yaml:"paths"`
}

type swaggerOperation struct {
	Tags        []string `json:"tags" yaml:"tags"`
	OperationID string   `json:"operationId" yaml:"operationId"`
}

// Swagger and OpenAPI allow metadata such as parameters and summary beside
// methods in a path item. Declaring methods explicitly makes those legal keys
// ignorable rather than attempting to decode them as operations.
type swaggerPathItem struct {
	Get     *swaggerOperation `json:"get" yaml:"get"`
	Post    *swaggerOperation `json:"post" yaml:"post"`
	Put     *swaggerOperation `json:"put" yaml:"put"`
	Patch   *swaggerOperation `json:"patch" yaml:"patch"`
	Delete  *swaggerOperation `json:"delete" yaml:"delete"`
	Head    *swaggerOperation `json:"head" yaml:"head"`
	Options *swaggerOperation `json:"options" yaml:"options"`
	Trace   *swaggerOperation `json:"trace" yaml:"trace"`
}

func (p swaggerPathItem) operations() map[string]*swaggerOperation {
	return map[string]*swaggerOperation{"GET": p.Get, "POST": p.Post, "PUT": p.Put, "PATCH": p.Patch, "DELETE": p.Delete, "HEAD": p.Head, "OPTIONS": p.Options, "TRACE": p.Trace}
}

// ImportSwaggerAPIEndpoints imports only HTTP operations. The tag used by
// Java, .NET, and Go generators is retained as the endpoint service code.
func (s *PDPService) ImportSwaggerAPIEndpoints(ctx context.Context, request SwaggerImportRequest) (*SwaggerImportResult, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	serviceResource, err := setServiceResourceScope(request.ServiceResource, request.Application)
	if err != nil {
		return nil, err
	}
	if err := s.ensureServiceResource(ctx, serviceResource); err != nil {
		return nil, err
	}
	document, err := fetchSwaggerDocument(ctx, request.SwaggerURL)
	if err != nil {
		return nil, err
	}
	operations, err := parseSwaggerOperations(document)
	if err != nil {
		return nil, err
	}
	existing, err := s.repository.ListAPIEndpoints(ctx, serviceResource)
	if err != nil {
		return nil, err
	}
	routes := make(map[string]struct{}, len(existing))
	for _, endpoint := range existing {
		routes[endpoint.Method+"\x00"+endpoint.PathTemplate] = struct{}{}
	}
	result := &SwaggerImportResult{Total: len(operations)}
	resources := map[string]*AuthorizationResource{}
	actions := map[string]*AuthorizationAction{}
	for _, operation := range operations {
		key := operation.Method + "\x00" + operation.PathTemplate
		if _, exists := routes[key]; exists {
			result.Skipped++
			continue
		}
		resource, ok := resources[operation.ServiceCode]
		if !ok {
			resource, err = s.swaggerResource(ctx, serviceResource, operation.ServiceCode)
			if err != nil {
				return nil, err
			}
			resources[operation.ServiceCode] = resource
		}
		action, ok := actions[operation.Method]
		if !ok {
			action, err = s.swaggerAction(ctx, serviceResource, operation.Method)
			if err != nil {
				return nil, err
			}
			actions[operation.Method] = action
		}
		_, err = s.CreateAPIEndpoint(ctx, &AuthorizationAPIEndpoint{
			ServiceResource: serviceResource, Application: serviceResource, ServiceCode: operation.ServiceCode, Method: operation.Method,
			PathTemplate: operation.PathTemplate, ResourceID: resource.ID, ActionID: action.ID,
			EnforcementMode: EnforcementModeDisabled, Enabled: false,
		})
		if err != nil {
			return nil, err
		}
		routes[key] = struct{}{}
		result.Created++
	}
	return result, nil
}

type swaggerEndpointOperation struct{ ServiceCode, Method, PathTemplate string }

func parseSwaggerOperations(body []byte) ([]swaggerEndpointOperation, error) {
	var document swaggerDocument
	if err := json.Unmarshal(body, &document); err != nil {
		if yamlErr := yaml.Unmarshal(body, &document); yamlErr != nil {
			return nil, fmt.Errorf("invalid Swagger or OpenAPI document: %w", err)
		}
	}
	if document.Swagger != "2.0" && !strings.HasPrefix(document.OpenAPI, "3.") {
		return nil, fmt.Errorf("%w: only Swagger 2.0 and OpenAPI 3.x documents are supported", ErrInvalidArgument)
	}
	basePath := strings.TrimSpace(document.BasePath)
	if basePath == "" && len(document.Servers) > 0 {
		if serverURL, err := url.Parse(document.Servers[0].URL); err == nil {
			basePath = serverURL.Path
		}
	}
	if basePath == "/" {
		basePath = ""
	}
	operations := make([]swaggerEndpointOperation, 0)
	for route, item := range document.Paths {
		for method, operation := range item.operations() {
			if operation == nil || !isHTTPMethod(method) {
				continue
			}
			serviceCode := swaggerServiceCode(*operation, document.Info.Title)
			pathTemplate := joinSwaggerPath(basePath, route)
			if _, _, _, err := normalizeEndpointRoute(serviceCode, method, pathTemplate); err != nil {
				return nil, err
			}
			operations = append(operations, swaggerEndpointOperation{ServiceCode: serviceCode, Method: method, PathTemplate: pathTemplate})
		}
	}
	if len(operations) == 0 {
		return nil, fmt.Errorf("%w: Swagger document contains no HTTP operations", ErrInvalidArgument)
	}
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].PathTemplate == operations[j].PathTemplate {
			return operations[i].Method < operations[j].Method
		}
		return operations[i].PathTemplate < operations[j].PathTemplate
	})
	return operations, nil
}

func fetchSwaggerDocument(ctx context.Context, rawURL string) ([]byte, error) {
	documentURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || documentURL.Scheme == "" || documentURL.Host == "" || (documentURL.Scheme != "http" && documentURL.Scheme != "https") {
		return nil, fmt.Errorf("%w: swagger_url must be an absolute http or https URL", ErrInvalidArgument)
	}
	document, err := fetchSwaggerDocumentAt(ctx, documentURL, 0)
	if err == nil || (documentURL.Hostname() != "localhost" && documentURL.Hostname() != "127.0.0.1") {
		return document, err
	}
	// In Docker, a Swagger URL entered as localhost points at the API container
	// rather than the developer machine. Compose provides this alias for the
	// host while native deployments continue using the original URL.
	hostURL := *documentURL
	hostURL.Host = "host.docker.internal"
	if documentURL.Port() != "" {
		hostURL.Host += ":" + documentURL.Port()
	}
	return fetchSwaggerDocumentAt(ctx, &hostURL, 0)
}

func fetchSwaggerDocumentAt(ctx context.Context, documentURL *url.URL, depth int) ([]byte, error) {
	if depth > 3 {
		return nil, fmt.Errorf("%w: Swagger UI document reference is nested too deeply", ErrInvalidArgument)
	}
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	fetch := func(value *url.URL) ([]byte, string, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, value.String(), nil)
		if err != nil {
			return nil, "", err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("fetch Swagger document: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, "", fmt.Errorf("fetch Swagger document: received HTTP %d", resp.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxSwaggerDocumentSize+1))
		if err != nil {
			return nil, "", err
		}
		if len(body) > maxSwaggerDocumentSize {
			return nil, "", fmt.Errorf("%w: Swagger document exceeds 8 MiB", ErrInvalidArgument)
		}
		return body, resp.Header.Get("Content-Type"), nil
	}
	body, contentType, err := fetch(documentURL)
	if err != nil {
		return nil, err
	}
	if bytes.Contains(bytes.ToLower(body), []byte("<html")) || strings.Contains(strings.ToLower(contentType), "text/html") {
		match := swaggerUIURLPattern.FindSubmatch(body)
		if len(match) < 2 {
			match = swaggerUIConfigURLPattern.FindSubmatch(body)
		}
		if len(match) < 2 {
			match, err = swaggerUIReference(body, documentURL, fetch)
			if err != nil {
				return nil, err
			}
		}
		if len(match) < 2 {
			return nil, fmt.Errorf("%w: Swagger UI page does not declare a document URL", ErrInvalidArgument)
		}
		specURL, err := swaggerReferenceURL(documentURL, match[1])
		if err != nil {
			return nil, fmt.Errorf("%w: invalid Swagger UI document URL", ErrInvalidArgument)
		}
		return fetchSwaggerDocumentAt(ctx, specURL, depth+1)
	}
	if isSwaggerDocument(body) {
		return body, nil
	}
	if match := swaggerUIURLPattern.FindSubmatch(body); len(match) >= 2 {
		specURL, parseErr := swaggerReferenceURL(documentURL, match[1])
		if parseErr != nil {
			return nil, fmt.Errorf("%w: invalid Swagger configuration document URL", ErrInvalidArgument)
		}
		return fetchSwaggerDocumentAt(ctx, specURL, depth+1)
	}
	if match := swaggerUIConfigURLPattern.FindSubmatch(body); len(match) >= 2 {
		specURL, parseErr := swaggerReferenceURL(documentURL, match[1])
		if parseErr != nil {
			return nil, fmt.Errorf("%w: invalid Swagger configuration URL", ErrInvalidArgument)
		}
		return fetchSwaggerDocumentAt(ctx, specURL, depth+1)
	}
	return body, nil
}

func swaggerUIReference(body []byte, documentURL *url.URL, fetch func(*url.URL) ([]byte, string, error)) ([][]byte, error) {
	scripts := swaggerInitializerPattern.FindAllSubmatch(body, -1)
	if len(scripts) == 0 {
		scripts = swaggerUIScriptPattern.FindAllSubmatch(body, -1)
	}
	for index := len(scripts) - 1; index >= 0; index-- {
		scriptURL, err := swaggerReferenceURL(documentURL, scripts[index][1])
		if err != nil {
			return nil, fmt.Errorf("%w: invalid Swagger UI script URL", ErrInvalidArgument)
		}
		script, _, err := fetch(scriptURL)
		if err != nil {
			return nil, err
		}
		if match := swaggerUIURLPattern.FindSubmatch(script); len(match) >= 2 {
			return match, nil
		}
		if match := swaggerUIConfigURLPattern.FindSubmatch(script); len(match) >= 2 {
			return match, nil
		}
	}
	return nil, nil
}

// Swagger UI serializes its inline JSON with escaped forward slashes, for
// example `\/swagger\/openapi.json`. URL parsing expects the decoded form.
func swaggerReferenceURL(documentURL *url.URL, reference []byte) (*url.URL, error) {
	value := strings.ReplaceAll(strings.TrimSpace(string(reference)), `\/`, `/`)
	return documentURL.Parse(value)
}

func isSwaggerDocument(body []byte) bool {
	var document swaggerDocument
	if json.Unmarshal(body, &document) != nil {
		if yaml.Unmarshal(body, &document) != nil {
			return false
		}
	}
	return document.Swagger == "2.0" || strings.HasPrefix(document.OpenAPI, "3.")
}

func (s *PDPService) swaggerResource(ctx context.Context, application, serviceCode string) (*AuthorizationResource, error) {
	code := "swagger-" + serviceCode
	if value, err := s.repository.GetResourceByCode(ctx, application, code); err == nil {
		return value, nil
	} else if !errorsIsNotFound(err) {
		return nil, err
	}
	return s.CreateResource(ctx, &AuthorizationResource{ServiceResource: application, Application: application, Code: code, Type: ResourceTypeAPI, Name: serviceCode, Description: "Imported from Swagger", Enabled: true})
}

func (s *PDPService) swaggerAction(ctx context.Context, application, method string) (*AuthorizationAction, error) {
	code := strings.ToLower(method)
	if value, err := s.repository.GetActionByCode(ctx, application, code); err == nil {
		return value, nil
	} else if !errorsIsNotFound(err) {
		return nil, err
	}
	return s.CreateAction(ctx, &AuthorizationAction{ServiceResource: application, Application: application, Code: code, Name: method, Description: "Imported from Swagger", Enabled: true})
}

func errorsIsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }
func isHTTPMethod(value string) bool {
	switch value {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE":
		return true
	}
	return false
}
func swaggerServiceCode(operation swaggerOperation, fallback string) string {
	if len(operation.Tags) > 0 && strings.TrimSpace(operation.Tags[0]) != "" {
		return sanitizeSwaggerCode(operation.Tags[0])
	}
	if operation.OperationID != "" {
		return operationIDServiceCode(operation.OperationID)
	}
	return sanitizeSwaggerCode(fallback)
}

func operationIDServiceCode(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '.' || r == '/' || r == ':' || r == '-'
	})
	if len(parts) == 0 {
		return "default"
	}
	return sanitizeSwaggerCode(parts[0])
}
func sanitizeSwaggerCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = regexp.MustCompile(`[^a-z0-9._-]+`).ReplaceAllString(value, "-")
	value = strings.Trim(value, "-._")
	if value == "" {
		return "default"
	}
	if len(value) > 100 {
		return value[:100]
	}
	return value
}
func joinSwaggerPath(basePath, route string) string {
	return path.Clean("/" + strings.Trim(basePath, "/") + "/" + strings.TrimSpace(route))
}
