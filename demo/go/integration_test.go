package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestPermissionCenterIntegration(t *testing.T) {
	if os.Getenv("DEMO_INTEGRATION") != "true" {
		t.Skip("set DEMO_INTEGRATION=true to call a running permission center")
	}

	pdp := &pdpHTTPClient{
		url:             requiredIntegrationEnv(t, "DEMO_PDP_DECISION_URL"),
		credential:      requiredIntegrationEnv(t, "DEMO_PDP_SERVICE_CREDENTIAL"),
		serviceResource: requiredIntegrationEnv(t, "DEMO_SERVICE_RESOURCE"),
		endpointID:      requiredIntegrationEnv(t, "DEMO_ENDPOINT_ID"),
		httpClient:      &http.Client{Timeout: 2 * time.Second},
	}
	verifier := fakeVerifier{identity: identity{
		subjectID: requiredIntegrationEnv(t, "DEMO_SUBJECT_ID"),
		tenantID:  requiredIntegrationEnv(t, "DEMO_TENANT_ID"),
	}}
	handler := authorize(verifier, pdp, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/orders/1001", nil)
	request.Header.Set("Authorization", "Bearer integration-test-token")
	request.Header.Set("X-Request-ID", "go-demo-live-smoke")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func requiredIntegrationEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required", name)
	}
	return value
}
