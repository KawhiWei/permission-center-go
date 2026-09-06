package pdp

import (
	"context"
	"net/http"
)

// Identity is produced from a verified session or token, never request JSON.
type Identity struct {
	SubjectID         string
	TenantID          string
	SubjectAttributes map[string]any
}

type IdentityResolver func(*http.Request) (Identity, error)
type EndpointResolver func(*http.Request) (string, error)

// RequireAPI is a fail-closed PEP middleware. Resolver failures and PDP
// transport failures intentionally produce 403 rather than reaching the API.
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
		result, err := client.Decide(r.Context(), Input{RequestID: r.Header.Get("X-Request-ID"), ServiceResource: serviceResource, TenantID: resolvedIdentity.TenantID, Subject: Subject{ID: resolvedIdentity.SubjectID, Attributes: resolvedIdentity.SubjectAttributes}, Action: Action{Kind: "http", Method: r.Method}, Resource: Resource{Kind: "api_endpoint", EndpointID: endpointID}})
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
