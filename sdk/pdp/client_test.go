package pdp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestClientDecideParsesStandardEnvelope(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/pdp/decisions" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Service") != "forum-api" {
			t.Fatalf("service header = %q", r.Header.Get("X-Service"))
		}
		var request DecisionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.SubjectID != "user-1" || request.ResourceCode != "post" {
			t.Fatalf("request body = %#v", request)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"success":true,"errorCode":null,"errorMessage":null,"result":{"allow":true,"reasonCode":"ALLOW","matchedPolicyIds":["00000000-0000-0000-0000-000000000001"]}}`))}, nil
	})
	client, err := NewClientWithConfig(Config{Endpoint: "http://pdp.test", HTTPClient: &http.Client{Transport: transport}, Headers: http.Header{"X-Service": []string{"forum-api"}}})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := client.Decide(context.Background(), DecisionRequest{Application: "forum", SubjectID: "user-1", ResourceCode: "post", Action: "read"})
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Allow || decision.ReasonCode != "ALLOW" || len(decision.MatchedPolicyIDs) != 1 {
		t.Fatalf("decision = %#v", decision)
	}
}

func TestClientAuthorizeHTTPReturnsDeniedError(t *testing.T) {
	client := newClientWithTransportForTest("http://pdp.test", `{"success":true,"errorCode":null,"errorMessage":null,"result":{"allow":false,"reasonCode":"NO_MATCHING_POLICY","matchedPolicyIds":[]}}`)
	decision, err := client.AuthorizeHTTP(context.Background(), "user-1", "forum", "post", "read")
	if decision.Allow || err == nil {
		t.Fatalf("decision = %#v error = %v", decision, err)
	}
	if err == nil || !containsError(err, ErrDenied) {
		t.Fatalf("error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newClientWithTransportForTest(endpoint, body string) *Client {
	return &Client{decisionURL: strings.TrimRight(endpoint, "/") + "/v1/pdp/decisions", httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})}}
}

func containsError(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unwrapper.Unwrap()
	}
	return false
}
