package pdp

import (
	"context"
	"net/http"
)

// Identity 必须来自已验证的 session/token，不能直接信任请求 JSON。
type Identity struct {
	SubjectID         string
	TenantID          string
	SubjectAttributes map[string]any
}

type IdentityResolver func(*http.Request) (Identity, error)
type EndpointResolver func(*http.Request) (string, error)

// RequireAPI 是 fail-closed 的 API 鉴权中间件；身份解析或 PDP 调用失败时不会进入业务处理器。
func RequireAPI(client *Client, serviceResource string, identity IdentityResolver, endpoint EndpointResolver, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resolvedIdentity, err := identity(r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		endpointID, err := endpoint(r)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		result, err := client.Decide(r.Context(), Input{RequestID: r.Header.Get("X-Request-ID"), AuthorizationType: "api", ServiceResource: serviceResource, TenantID: resolvedIdentity.TenantID, Subject: Subject{ID: resolvedIdentity.SubjectID, Attributes: resolvedIdentity.SubjectAttributes}, Action: Action{Kind: "http", Method: r.Method}, Resource: Resource{Kind: "api_endpoint", EndpointID: endpointID}})
		if err != nil || result.Decision != DecisionAllow {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), decisionContextKey{}, result)))
	})
}

type decisionContextKey struct{}

func DecisionFromContext(ctx context.Context) (Result, bool) {
	result, ok := ctx.Value(decisionContextKey{}).(Result)
	return result, ok
}
