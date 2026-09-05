package httpserver

import (
	"net/http"
	"strings"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// SwaggerUI serves the bundled Swagger UI assets locally. The UI retrieves the
// OpenAPI document from this service, so it works without a browser-side CDN.
// SwaggerUI 返回内嵌静态资源的 Swagger UI 处理器。
func SwaggerUI() http.Handler {
	return httpSwagger.Handler(
		httpSwagger.URL("/swagger/openapi.json"),
		httpSwagger.DocExpansion("list"),
		httpSwagger.PersistAuthorization(true),
	)
}

// OpenAPIDocument 返回权限中心当前公开 HTTP API 的 OpenAPI 文档。
func (h *Handler) OpenAPIDocument(w http.ResponseWriter, _ *http.Request) {
	writeRawJSON(w, http.StatusOK, openAPIDocument())
}

type openAPIOperation struct {
	method  string
	path    string
	tag     string
	summary string
	body    bool
}

// openAPIDocument 生成当前 HTTP API 的 OpenAPI 文档。
func openAPIDocument() map[string]any {
	paths := map[string]any{}
	for _, operation := range openAPIOperations {
		pathItem, ok := paths[operation.path].(map[string]any)
		if !ok {
			pathItem = map[string]any{}
			paths[operation.path] = pathItem
		}
		definition := map[string]any{
			"tags":        []string{operation.tag},
			"summary":     operation.summary,
			"operationId": strings.ToLower(operation.method) + "-" + strings.Trim(strings.NewReplacer("/", "-", "{", "", "}", "", ":", "-").Replace(strings.TrimPrefix(operation.path, "/")), "-"),
			"responses": map[string]any{
				"200": map[string]any{"description": "Success", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/APIResponse"}}}},
				"400": map[string]any{"description": "Invalid request"},
				"401": map[string]any{"description": "Authentication required"},
			},
			"security": []map[string][]string{{"cookieAuth": {}}},
		}
		if operation.method == http.MethodPost {
			definition["responses"].(map[string]any)["201"] = definition["responses"].(map[string]any)["200"]
		}
		if operation.body {
			definition["requestBody"] = map[string]any{
				"required": true,
				"content":  map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "additionalProperties": true}}},
			}
		}
		pathItem[strings.ToLower(operation.method)] = definition
	}
	return map[string]any{
		"openapi": "3.0.3",
		"info":    map[string]any{"title": "Permission Center API", "version": "1.0.0", "description": "Permission Center management API."},
		"servers": []map[string]string{{"url": "/", "description": "Current server"}},
		"paths":   paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{"cookieAuth": map[string]any{"type": "apiKey", "in": "cookie", "name": "permission_center_session"}},
			"schemas":         map[string]any{"APIResponse": map[string]any{"type": "object", "properties": map[string]any{"success": map[string]any{"type": "boolean"}, "errorCode": map[string]any{"nullable": true}, "errorMessage": map[string]any{"nullable": true}, "result": map[string]any{}}}},
		},
	}
}

var openAPIOperations = []openAPIOperation{
	{"GET", "/healthz", "System", "Health check", false},
	{"GET", "/auth/config", "Authentication", "Get authentication configuration", false},
	{"GET", "/auth/login", "Authentication", "Start sign-in", false},
	{"GET", "/auth/me", "Authentication", "Get current user", false},
	{"POST", "/auth/logout", "Authentication", "Sign out", false},
	{"POST", "/v1/service-resources", "Service resources", "Create service resource", true}, {"GET", "/v1/service-resources", "Service resources", "List service resources", false}, {"GET", "/v1/service-resources/{key}", "Service resources", "Get service resource", false}, {"PUT", "/v1/service-resources/{key}", "Service resources", "Update service resource", true}, {"DELETE", "/v1/service-resources/{key}", "Service resources", "Delete service resource", false},
	{"POST", "/v1/roles", "Roles", "Create role", true}, {"GET", "/v1/roles", "Roles", "List roles", false}, {"PUT", "/v1/roles/{roleID}", "Roles", "Update role", true}, {"DELETE", "/v1/roles/{roleID}", "Roles", "Delete role", false}, {"POST", "/v1/menus", "Menus", "Create menu", true}, {"GET", "/v1/menus/tree", "Menus", "Get menu tree", false}, {"PUT", "/v1/menus/{id}", "Menus", "Update menu", true}, {"DELETE", "/v1/menus/{id}", "Menus", "Delete menu", false}, {"PUT", "/v1/roles/{roleID}/menus", "Roles", "Replace role menus", true}, {"GET", "/v1/roles/{roleID}/menus", "Roles", "List role menus", false}, {"PUT", "/v1/users/{userID}/roles", "User roles", "Replace user roles", true}, {"GET", "/v1/users/{userID}/roles", "User roles", "List user roles", false},
	{"POST", "/v1/authorization/api-endpoints", "API endpoints", "Create API endpoint", true}, {"GET", "/v1/authorization/api-endpoints", "API endpoints", "List API endpoints", false}, {"POST", "/v1/authorization/api-endpoints/import-swagger", "API endpoints", "Import Swagger endpoints", true}, {"GET", "/v1/authorization/api-endpoints/{id}", "API endpoints", "Get API endpoint", false}, {"PUT", "/v1/authorization/api-endpoints/{id}", "API endpoints", "Update API endpoint", true}, {"DELETE", "/v1/authorization/api-endpoints/{id}", "API endpoints", "Delete API endpoint", false},
}
