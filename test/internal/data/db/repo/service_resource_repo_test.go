package repo

import (
	"errors"
	"strings"
	"testing"

	"github.com/luck/permission-center-go/internal/biz"
	"github.com/luck/permission-center-go/internal/data/db/model"
)

func TestNormalizeServiceResourceKey(t *testing.T) {
	value, err := normalizeServiceResourceKey(" orders ")
	if err != nil || value != "orders" {
		t.Fatalf("normalized key = %q, %v", value, err)
	}
	if _, err := normalizeServiceResourceKey(""); !errors.Is(err, biz.ErrInvalidArgument) {
		t.Fatalf("empty key error = %v", err)
	}
	if _, err := normalizeServiceResourceKey(strings.Repeat("x", 129)); !errors.Is(err, biz.ErrInvalidArgument) {
		t.Fatalf("long key error = %v", err)
	}
}

func TestServiceResourceRepositoryImplementsBusinessContract(t *testing.T) {
	var _ biz.ServiceResourceRepository = (*ServiceResourceRepository)(nil)
	var _ biz.ServiceResourceSourceRepository = (*ServiceResourceRepository)(nil)
}

func TestToBizServiceResourceMapsResourceKeyToKey(t *testing.T) {
	value := toBizServiceResource(&model.ServiceResource{
		ResourceKey: "orders",
		Source:      biz.ServiceResourceSourceLocal,
		DisplayName: "订单服务",
		Enabled:     true,
	})
	if value.Key != "orders" || value.Name != "订单服务" || value.DisplayName != "订单服务" {
		t.Fatalf("service resource = %#v", value)
	}
}

func TestNormalizeServiceResourceSource(t *testing.T) {
	value, err := normalizeServiceResourceSource(" NEXUSAUTH ")
	if err != nil || value != biz.ServiceResourceSourceNexusAuth {
		t.Fatalf("normalized source = %q, %v", value, err)
	}
	if _, err := normalizeServiceResourceSource("unsupported"); !errors.Is(err, biz.ErrInvalidArgument) {
		t.Fatalf("unsupported source error = %v", err)
	}
}
