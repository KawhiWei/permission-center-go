package pdp

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestClientDecide(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/v1/pdp/decisions" || request.Header.Get("Authorization") != "Bearer service-token" {
			t.Errorf("unexpected PDP request: path=%s authorization=%q", request.URL.Path, request.Header.Get("Authorization"))
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"decision":"allow","reason_code":"allowed_by_policy","matched_policy_ids":[],"snapshot_version":1,"obligations":{}}`)), Header: make(http.Header)}, nil
	})
	client, err := NewClient("https://permission-center.example", "service-token", &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Decide(context.Background(), Input{ServiceResource: "orders", TenantID: "tenant-a", Subject: Subject{ID: "subject-a"}, Action: Action{Kind: "http", Method: "GET"}, Resource: Resource{Kind: "api_endpoint", EndpointID: "endpoint-a"}})
	if err != nil || result.Decision != DecisionAllow {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
