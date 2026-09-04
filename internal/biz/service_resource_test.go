package biz

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type serviceResourceRoundTripper func(*http.Request) (*http.Response, error)

func (f serviceResourceRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestNexusAuthServiceResourceCatalogList(t *testing.T) {
	client := &http.Client{Transport: serviceResourceRoundTripper(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://nexus.example/openapi/v1/service-resources" {
			t.Fatalf("request URL = %q", request.URL.String())
		}
		if request.Header.Get("Authorization") != "Bearer directory-token" {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`[{"id":"resource-id","name":"orders-api","displayName":"订单服务","audience":"orders","description":"订单资源","isActive":true,"createdAt":"2026-09-04T10:00:00Z"}]`)),
		}, nil
	})}
	catalog, err := NewNexusAuthServiceResourceCatalogWithClient("https://nexus.example", "directory-token", time.Second, client)
	if err != nil {
		t.Fatal(err)
	}
	resources, err := catalog.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 || resources[0].Name != "orders-api" || !resources[0].IsActive {
		t.Fatalf("resources = %#v", resources)
	}
}
