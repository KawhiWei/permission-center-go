package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeVerifier struct {
	identity identity
	err      error
}

func (v fakeVerifier) Verify(context.Context, string) (identity, error) {
	return v.identity, v.err
}

func TestAuthorizeCallsPDPAndAllowsRequest(t *testing.T) {
	pdpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer service-secret" {
			t.Fatalf("Authorization = %q", got)
		}
		var input decisionRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		if input.AuthorizationType != "api" || input.Subject.ID != "user-1" || input.TenantID != "tenant-a" || input.Resource.EndpointID != "11111111-1111-1111-1111-111111111111" {
			t.Fatalf("unexpected PDP input: %#v", input)
		}
		writeJSON(w, http.StatusOK, decisionResponse{Decision: "allow", ReasonCode: "allowed_by_policy"})
	}))
	defer pdpServer.Close()

	pdp := testPDPClient(pdpServer.URL)
	handler := authorize(fakeVerifier{identity: identity{subjectID: "user-1", tenantID: "tenant-a"}}, pdp, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/orders/1", nil)
	req.Header.Set("Authorization", "Bearer user-token")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestAuthorizeFailsClosedOnDeny(t *testing.T) {
	pdpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, decisionResponse{Decision: "deny", ReasonCode: "default_deny"})
	}))
	defer pdpServer.Close()

	called := false
	handler := authorize(fakeVerifier{identity: identity{subjectID: "user-1", tenantID: "tenant-a"}}, testPDPClient(pdpServer.URL), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/orders/1", nil)
	req.Header.Set("Authorization", "Bearer user-token")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden || called {
		t.Fatalf("status = %d, business handler called = %v", recorder.Code, called)
	}
}

func TestAuthorizeRejectsMissingBearerToken(t *testing.T) {
	handler := authorize(fakeVerifier{}, testPDPClient("http://127.0.0.1"), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/orders/1", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func testPDPClient(baseURL string) *pdpHTTPClient {
	return &pdpHTTPClient{
		url: baseURL, credential: "service-secret", serviceResource: "orders-api",
		endpointID: "11111111-1111-1111-1111-111111111111", httpClient: http.DefaultClient,
	}
}
