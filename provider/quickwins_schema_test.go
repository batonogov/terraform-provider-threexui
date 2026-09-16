package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestProviderMetadata(t *testing.T) {
	p := &ThreeXUIProvider{version: "1.0.0"}
	var resp provider.MetadataResponse
	p.Metadata(context.Background(), provider.MetadataRequest{}, &resp)
	if resp.TypeName != "threexui" {
		t.Fatalf("expected threexui, got %s", resp.TypeName)
	}
	if resp.Version != "1.0.0" {
		t.Fatalf("expected 1.0.0, got %s", resp.Version)
	}
}

func TestProviderResources(t *testing.T) {
	p := &ThreeXUIProvider{}
	resources := p.Resources(context.Background())
	if len(resources) != 19 {
		t.Fatalf("expected 19 resources, got %d", len(resources))
	}
}

func TestProviderDataSources(t *testing.T) {
	p := &ThreeXUIProvider{}
	dataSources := p.DataSources(context.Background())
	if len(dataSources) != 8 {
		t.Fatalf("expected 8 data sources, got %d", len(dataSources))
	}
}

// --- xray_reverse_schema.go ---

func TestBuildXrayReverseJSON_Roundtrip_MultipleEntries(t *testing.T) {
	input := map[string]any{
		"bridge": []any{
			map[string]any{"tag": "bridge-1", "domain": "bridge.example.com"},
			map[string]any{"tag": "bridge-2", "domain": "bridge2.example.com"},
		},
		"portal": []any{
			map[string]any{"tag": "portal-1", "domain": "portal.example.com"},
		},
	}
	flattened := flattenXrayReverseToMap(buildXrayReverseJSON(input))

	bridges, ok := flattened["bridge"].([]any)
	if !ok || len(bridges) != 2 {
		t.Fatalf("expected 2 bridges, got %v", flattened["bridge"])
	}
	b1 := bridges[0].(map[string]any)
	if b1["tag"] != "bridge-1" || b1["domain"] != "bridge.example.com" {
		t.Fatalf("unexpected first bridge: %v", b1)
	}
	b2 := bridges[1].(map[string]any)
	if b2["tag"] != "bridge-2" || b2["domain"] != "bridge2.example.com" {
		t.Fatalf("unexpected second bridge: %v", b2)
	}

	portals, ok := flattened["portal"].([]any)
	if !ok || len(portals) != 1 {
		t.Fatalf("expected 1 portal, got %v", flattened["portal"])
	}
	p1 := portals[0].(map[string]any)
	if p1["tag"] != "portal-1" || p1["domain"] != "portal.example.com" {
		t.Fatalf("unexpected portal: %v", p1)
	}
}

func TestFlattenXrayReverseToMap_Empty(t *testing.T) {
	result := flattenXrayReverseToMap(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %v", result)
	}

	result = flattenXrayReverseToMap(map[string]any{})
	if len(result) != 0 {
		t.Fatalf("expected empty map for empty payload, got %v", result)
	}
}

func TestFlattenReverseEntries(t *testing.T) {
	list := []any{
		map[string]any{"tag": "a", "domain": "a.com"},
		map[string]any{"tag": "b", "domain": "b.com"},
	}
	result := flattenReverseEntries(list)
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	first := result[0].(map[string]any)
	if first["tag"] != "a" || first["domain"] != "a.com" {
		t.Fatalf("unexpected first entry: %v", first)
	}
	second := result[1].(map[string]any)
	if second["tag"] != "b" || second["domain"] != "b.com" {
		t.Fatalf("unexpected second entry: %v", second)
	}
}

func TestFlattenReverseEntries_SkipsNonMap(t *testing.T) {
	list := []any{
		"not-a-map",
		map[string]any{"tag": "valid", "domain": "valid.com"},
	}
	result := flattenReverseEntries(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry (non-map skipped), got %d", len(result))
	}
}

func TestFlattenXrayReverseToMap_FromJSON(t *testing.T) {
	input := `{"bridges":[{"tag":"b1","domain":"b.example.com"}],"portals":[{"tag":"p1","domain":"p.example.com"}]}`
	result := flattenXrayReverseToMap(input)
	bridges, ok := result["bridge"].([]any)
	if !ok || len(bridges) != 1 {
		t.Fatalf("expected 1 bridge from JSON string, got %v", result["bridge"])
	}
	portals, ok := result["portal"].([]any)
	if !ok || len(portals) != 1 {
		t.Fatalf("expected 1 portal from JSON string, got %v", result["portal"])
	}
}

// --- xray_balancers_schema.go ---

func TestBuildXrayBalancersJSON_Roundtrip_MultipleSelectors(t *testing.T) {
	input := map[string]any{
		"balancer": []any{
			map[string]any{
				"tag":      "bal1",
				"selector": []any{"proxy-*", "vpn-*"},
				"strategy": []any{map[string]any{"type": "leastPing"}},
			},
		},
	}
	built := buildXrayBalancersJSON(input)
	builtList := built.([]any)
	if len(builtList) != 1 {
		t.Fatalf("expected 1 built balancer, got %d", len(builtList))
	}
	builtEntry := builtList[0].(map[string]any)
	if builtEntry["tag"] != "bal1" {
		t.Fatalf("expected tag bal1, got %v", builtEntry["tag"])
	}
	sel, ok := builtEntry["selector"].([]string)
	if !ok || len(sel) != 2 {
		t.Fatalf("expected 2 selectors ([]string), got %v", builtEntry["selector"])
	}
	if sel[0] != "proxy-*" || sel[1] != "vpn-*" {
		t.Fatalf("unexpected selectors: %v", sel)
	}
	strategy, ok := builtEntry["strategy"].(map[string]any)
	if !ok || strategy["type"] != "leastPing" {
		t.Fatalf("expected leastPing strategy, got %v", builtEntry["strategy"])
	}
}

func TestFlattenXrayBalancersToMap_Empty(t *testing.T) {
	result := flattenXrayBalancersToMap(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty map for nil, got %v", result)
	}

	result = flattenXrayBalancersToMap([]any{})
	balancers, ok := result["balancer"].([]any)
	if !ok || len(balancers) != 0 {
		t.Fatalf("expected empty balancer list, got %v", result["balancer"])
	}
}

func TestFlattenBalancers(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "mybal",
			"selector": []any{"out-*"},
			"strategy": map[string]any{"type": "random"},
		},
	}
	result := flattenBalancers(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 balancer, got %d", len(result))
	}
	bal := result[0].(map[string]any)
	if bal["tag"] != "mybal" {
		t.Fatalf("expected tag mybal, got %v", bal["tag"])
	}
	sel, ok := bal["selector"].([]any)
	if !ok || len(sel) != 1 || sel[0] != "out-*" {
		t.Fatalf("unexpected selectors: %v", bal["selector"])
	}
	strategies := bal["strategy"].([]any)
	if len(strategies) != 1 {
		t.Fatalf("expected 1 strategy, got %d", len(strategies))
	}
	if strategies[0].(map[string]any)["type"] != "random" {
		t.Fatalf("expected random, got %v", strategies[0])
	}
}

func TestFlattenBalancers_SkipsNonMap(t *testing.T) {
	list := []any{
		"not-a-map",
		map[string]any{"tag": "valid", "selector": []any{"s-*"}},
	}
	result := flattenBalancers(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 balancer (non-map skipped), got %d", len(result))
	}
}

func TestFlattenXrayBalancersToMap_FromJSON(t *testing.T) {
	input := `[{"tag":"bal1","selector":["proxy-*"],"strategy":{"type":"leastPing"}}]`
	result := flattenXrayBalancersToMap(input)
	balancers, ok := result["balancer"].([]any)
	if !ok || len(balancers) != 1 {
		t.Fatalf("expected 1 balancer from JSON string, got %v", result["balancer"])
	}
	bal := balancers[0].(map[string]any)
	if bal["tag"] != "bal1" {
		t.Fatalf("expected tag bal1, got %v", bal["tag"])
	}
}
