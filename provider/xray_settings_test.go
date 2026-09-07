package provider

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDeepMergeJSON_Flat(t *testing.T) {
	dst := map[string]any{"a": "1"}
	src := map[string]any{"b": "2"}
	result := deepMergeJSON(dst, src)
	if result["a"] != "1" || result["b"] != "2" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestDeepMergeJSON_Overlap(t *testing.T) {
	dst := map[string]any{"a": "old"}
	src := map[string]any{"a": "new"}
	result := deepMergeJSON(dst, src)
	if result["a"] != "new" {
		t.Fatalf("expected overwrite, got %v", result["a"])
	}
}

func TestDeepMergeJSON_Nested(t *testing.T) {
	dst := map[string]any{"a": map[string]any{"x": 1}}
	src := map[string]any{"a": map[string]any{"y": 2}}
	result := deepMergeJSON(dst, src)
	inner, ok := result["a"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested map")
	}
	if inner["x"] != 1 || inner["y"] != 2 {
		t.Fatalf("unexpected nested result: %v", inner)
	}
}

func TestDeepMergeJSON_NilDst(t *testing.T) {
	src := map[string]any{"a": "1"}
	result := deepMergeJSON(nil, src)
	if result["a"] != "1" {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestDeepMergeJSON_MapReplacedByScalar(t *testing.T) {
	dst := map[string]any{"a": map[string]any{"x": 1}}
	src := map[string]any{"a": "scalar"}
	result := deepMergeJSON(dst, src)
	if result["a"] != "scalar" {
		t.Fatalf("expected scalar, got %v", result["a"])
	}
}

func TestSetJSONPath_SingleLevel(t *testing.T) {
	root := map[string]any{}
	setJSONPath(root, []string{"key"}, "val")
	if root["key"] != "val" {
		t.Fatalf("unexpected: %v", root)
	}
}

func TestSetJSONPath_MultiLevel(t *testing.T) {
	root := map[string]any{}
	setJSONPath(root, []string{"a", "b", "c"}, 42)
	a, _ := root["a"].(map[string]any)
	b, _ := a["b"].(map[string]any)
	if b["c"] != 42 {
		t.Fatalf("unexpected: %v", root)
	}
}

func TestSetJSONPath_CreateIntermediates(t *testing.T) {
	root := map[string]any{"x": "keep"}
	setJSONPath(root, []string{"a", "b"}, "new")
	if root["x"] != "keep" {
		t.Fatalf("existing key lost")
	}
	a, _ := root["a"].(map[string]any)
	if a["b"] != "new" {
		t.Fatalf("unexpected: %v", root)
	}
}

func TestSetJSONPath_Overwrite(t *testing.T) {
	root := map[string]any{"a": "old"}
	setJSONPath(root, []string{"a"}, "new")
	if root["a"] != "new" {
		t.Fatalf("expected overwrite")
	}
}

func TestGetJSONPath_Existing(t *testing.T) {
	root := map[string]any{"a": map[string]any{"b": "val"}}
	got := getJSONPath(root, []string{"a", "b"})
	if got != "val" {
		t.Fatalf("expected val, got %v", got)
	}
}

func TestGetJSONPath_Missing(t *testing.T) {
	root := map[string]any{"a": "val"}
	got := getJSONPath(root, []string{"b"})
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestGetJSONPath_NonMapIntermediate(t *testing.T) {
	root := map[string]any{"a": "string"}
	got := getJSONPath(root, []string{"a", "b"})
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestCloneJSONMap_Nil(t *testing.T) {
	result := cloneJSONMap(nil)
	if result == nil || len(result) != 0 {
		t.Fatalf("expected empty map, got %v", result)
	}
}

func TestCloneJSONMap_Independence(t *testing.T) {
	original := map[string]any{"a": "1", "b": "2"}
	clone := cloneJSONMap(original)
	clone["a"] = "changed"
	if original["a"] != "1" {
		t.Fatalf("clone modified original")
	}
}

func TestDeepEqualJSON_EqualMaps(t *testing.T) {
	a := map[string]any{"x": float64(1)}
	b := map[string]any{"x": float64(1)}
	if !deepEqualJSON(a, b) {
		t.Fatalf("expected equal")
	}
}

func TestDeepEqualJSON_DifferentMaps(t *testing.T) {
	a := map[string]any{"x": float64(1)}
	b := map[string]any{"x": float64(2)}
	if deepEqualJSON(a, b) {
		t.Fatalf("expected not equal")
	}
}

func TestDeepEqualJSON_Arrays(t *testing.T) {
	a := []any{float64(1), float64(2)}
	b := []any{float64(1), float64(2)}
	if !deepEqualJSON(a, b) {
		t.Fatalf("expected equal")
	}
}

func TestDeepEqualJSON_DifferentTypes(t *testing.T) {
	if deepEqualJSON("string", float64(1)) {
		t.Fatalf("expected not equal for different types")
	}
}

func TestExtractXraySection_MergeRoot(t *testing.T) {
	current := map[string]any{
		"log":       map[string]any{"loglevel": "debug"},
		"policy":    map[string]any{},
		"api":       map[string]any{"tag": "api"},
		"stats":     map[string]any{},
		"dns":       map[string]any{"servers": []any{}},
		"outbounds": []any{},
	}
	result := extractXraySection(current, xraySectionBasics)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map")
	}
	if _, ok := m["log"]; !ok {
		t.Fatalf("missing log")
	}
	if _, ok := m["api"]; !ok {
		t.Fatalf("missing api")
	}
	if _, ok := m["stats"]; !ok {
		t.Fatalf("missing stats")
	}
	if _, ok := m["dns"]; ok {
		t.Fatalf("dns should not be in basics")
	}
	if _, ok := m["outbounds"]; ok {
		t.Fatalf("outbounds should not be in basics")
	}
}

func TestExtractXraySection_SetPath(t *testing.T) {
	current := map[string]any{"dns": map[string]any{"servers": []any{"8.8.8.8"}}}
	result := extractXraySection(current, xraySectionDNS)
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map")
	}
	if _, ok := m["servers"]; !ok {
		t.Fatalf("missing servers")
	}
}

func TestApplyXraySection_MergeRoot(t *testing.T) {
	current := map[string]any{"log": map[string]any{"loglevel": "info"}}
	desired := map[string]any{"log": map[string]any{"loglevel": "debug"}}
	result, err := applyXraySection(current, desired, xraySectionBasics)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	log, _ := result["log"].(map[string]any)
	if log["loglevel"] != "debug" {
		t.Fatalf("expected debug, got %v", log["loglevel"])
	}
}

func TestApplyXraySection_SetPath(t *testing.T) {
	current := map[string]any{}
	desired := map[string]any{"servers": []any{"8.8.8.8"}}
	result, err := applyXraySection(current, desired, xraySectionDNS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dns, ok := result["dns"].(map[string]any)
	if !ok {
		t.Fatalf("expected dns key")
	}
	if _, ok := dns["servers"]; !ok {
		t.Fatalf("missing servers in dns")
	}
}

func TestApplyXraySection_MergeRoot_NotObject(t *testing.T) {
	_, err := applyXraySection(map[string]any{}, "string", xraySectionBasics)
	if err == nil {
		t.Fatalf("expected error for non-object")
	}
}

func TestApplyXraySection_SetPath_EmptyPath(t *testing.T) {
	section := xraySection{id: "test", mode: xraySectionSetPath, path: []string{}}
	_, err := applyXraySection(map[string]any{}, map[string]any{}, section)
	if err == nil {
		t.Fatalf("expected error for empty path")
	}
}

// --- Build/Flatten unit tests ---

func TestFlattenXrayReverseToMap(t *testing.T) {
	data := map[string]any{
		"bridges": []any{
			map[string]any{"tag": "b1", "domain": "test.com"},
		},
		"portals": []any{
			map[string]any{"tag": "p1", "domain": "test.com"},
		},
	}
	result := flattenXrayReverseToMap(data)
	bridges, ok := result["bridge"].([]any)
	if !ok || len(bridges) != 1 {
		t.Fatalf("expected 1 bridge, got %v", result["bridge"])
	}
	b := bridges[0].(map[string]any)
	if b["tag"] != "b1" {
		t.Fatalf("expected tag b1, got %v", b["tag"])
	}
}

func TestFlattenXrayBalancersToMap(t *testing.T) {
	data := []any{
		map[string]any{
			"tag":      "bal1",
			"selector": []any{"proxy-*"},
			"strategy": map[string]any{"type": "leastPing"},
		},
	}
	result := flattenXrayBalancersToMap(data)
	balancers, ok := result["balancer"].([]any)
	if !ok || len(balancers) != 1 {
		t.Fatalf("expected 1 balancer, got %v", result["balancer"])
	}
}

func TestFlattenXrayDNSToMap(t *testing.T) {
	data := map[string]any{
		"servers": []any{
			"8.8.8.8",
			map[string]any{
				"address":     "localhost",
				"port":        float64(53),
				"domains":     []any{"geosite:cn"},
				"expectedIPs": []any{"geoip:cn"},
			},
		},
		"queryStrategy": "UseIP",
	}
	result := flattenXrayDNSToMap(data)
	servers, ok := result["server"].([]any)
	if !ok || len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %v", result["server"])
	}
	// First server is string-only
	s0 := servers[0].(map[string]any)
	if s0["address"] != "8.8.8.8" {
		t.Fatalf("expected 8.8.8.8, got %v", s0["address"])
	}
	// Second server is object
	s1 := servers[1].(map[string]any)
	if s1["address"] != "localhost" {
		t.Fatalf("expected localhost, got %v", s1["address"])
	}
	if result["query_strategy"] != "UseIP" {
		t.Fatalf("expected UseIP, got %v", result["query_strategy"])
	}
}

func TestExpandDNSServers_StringOnly(t *testing.T) {
	list := []any{
		map[string]any{"address": "8.8.8.8"},
	}
	result := expandDNSServers(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 server")
	}
	// Should be a plain string
	if s, ok := result[0].(string); !ok || s != "8.8.8.8" {
		t.Fatalf("expected string 8.8.8.8, got %v", result[0])
	}
}

func TestExpandDNSServers_WithPort(t *testing.T) {
	list := []any{
		map[string]any{"address": "localhost", "port": 53},
	}
	result := expandDNSServers(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 server")
	}
	m, ok := result[0].(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result[0])
	}
	if m["address"] != "localhost" || m["port"] != 53 {
		t.Fatalf("unexpected: %v", m)
	}
}

func TestFlattenXrayRoutingToMap(t *testing.T) {
	data := map[string]any{
		"domainStrategy": "AsIs",
		"rules": []any{
			map[string]any{
				"type":        "field",
				"ip":          []any{"geoip:private"},
				"outboundTag": "blocked",
			},
		},
	}
	result := flattenXrayRoutingToMap(data)
	if result["domain_strategy"] != "AsIs" {
		t.Fatalf("expected AsIs, got %v", result["domain_strategy"])
	}
	rules, ok := result["rule"].([]any)
	if !ok || len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %v", result["rule"])
	}
	r := rules[0].(map[string]any)
	if r["outbound_tag"] != "blocked" {
		t.Fatalf("expected blocked, got %v", r["outbound_tag"])
	}
}

func TestFlattenXrayRoutingToMap_SkipsInternalAPIRule(t *testing.T) {
	data := map[string]any{
		"rules": []any{
			map[string]any{
				"type":        "field",
				"inboundTag":  []any{"api"},
				"outboundTag": "api",
			},
			map[string]any{
				"type":        "field",
				"ip":          []any{"geoip:private"},
				"outboundTag": "blocked",
			},
		},
	}
	result := flattenXrayRoutingToMap(data)
	rules, ok := result["rule"].([]any)
	if !ok || len(rules) != 1 {
		t.Fatalf("expected only user-managed rule, got %v", result["rule"])
	}
	rule := rules[0].(map[string]any)
	if rule["outbound_tag"] != "blocked" {
		t.Fatalf("expected blocked rule to remain, got %v", rule)
	}
}

func TestFlattenXrayBasicsToMap(t *testing.T) {
	data := map[string]any{
		"log": map[string]any{
			"loglevel": "warning",
			"dnsLog":   false,
		},
		"api": map[string]any{
			"tag":      "api",
			"services": []any{"HandlerService", "StatsService"},
		},
		"stats": map[string]any{},
	}
	result := flattenXrayBasicsToMap(data)
	log, ok := result["log"].(map[string]any)
	if !ok {
		t.Fatalf("expected log map, got %v", result["log"])
	}
	if log["loglevel"] != "warning" {
		t.Fatalf("expected warning, got %v", log["loglevel"])
	}
	if _, ok := result["stats"]; !ok {
		t.Fatalf("expected stats block")
	}
}

func TestFlattenXrayOutboundsToMap(t *testing.T) {
	data := []any{
		map[string]any{
			"tag":      "direct",
			"protocol": "freedom",
			"settings": map[string]any{
				"domainStrategy": "AsIs",
			},
		},
		map[string]any{
			"tag":      "blocked",
			"protocol": "blackhole",
			"settings": map[string]any{
				"response": map[string]any{"type": "none"},
			},
		},
	}
	result := flattenXrayOutboundsToMap(data)
	outbounds, ok := result["outbound"].([]any)
	if !ok || len(outbounds) != 2 {
		t.Fatalf("expected 2 outbounds, got %v", result["outbound"])
	}

	o0 := outbounds[0].(map[string]any)
	if o0["tag"] != "direct" || o0["protocol"] != "freedom" {
		t.Fatalf("unexpected first outbound: %v", o0)
	}
	freedomList, ok := o0["freedom_settings"].([]any)
	if !ok || len(freedomList) != 1 {
		t.Fatalf("expected freedom_settings, got %v", o0["freedom_settings"])
	}
	freedom := freedomList[0].(map[string]any)
	if freedom["domain_strategy"] != "AsIs" {
		t.Fatalf("expected AsIs, got %v", freedom["domain_strategy"])
	}

	o1 := outbounds[1].(map[string]any)
	bhList, ok := o1["blackhole_settings"].([]any)
	if !ok || len(bhList) != 1 {
		t.Fatalf("expected blackhole_settings, got %v", o1["blackhole_settings"])
	}
	bh := bhList[0].(map[string]any)
	if bh["response_type"] != "none" {
		t.Fatalf("expected none, got %v", bh["response_type"])
	}
}

func TestFlattenBlackholeSettings_EmptyDefaultsToNone(t *testing.T) {
	t.Parallel()
	in := map[string]any{}
	out := flattenBlackholeSettings(in)
	if out["response_type"] != "none" {
		t.Fatalf("expected response_type='none' for empty settings, got %v", out["response_type"])
	}
}

func TestFlattenBlackholeSettings_ExplicitTypePreserved(t *testing.T) {
	t.Parallel()
	in := map[string]any{"response": map[string]any{"type": "http"}}
	out := flattenBlackholeSettings(in)
	if out["response_type"] != "http" {
		t.Fatalf("expected response_type='http', got %v", out["response_type"])
	}
}

func TestFlattenBlackholeSettings_ExplicitNonePreserved(t *testing.T) {
	t.Parallel()
	in := map[string]any{"response": map[string]any{"type": "none"}}
	out := flattenBlackholeSettings(in)
	if out["response_type"] != "none" {
		t.Fatalf("expected response_type='none', got %v", out["response_type"])
	}
}

func TestFlattenXrayOutboundsToMap_EmptyBlackholeSettings(t *testing.T) {
	data := []any{
		map[string]any{
			"tag":      "blocked",
			"protocol": "blackhole",
			"settings": map[string]any{},
		},
	}
	result := flattenXrayOutboundsToMap(data)
	outbounds := result["outbound"].([]any)
	if len(outbounds) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(outbounds))
	}
	bh := outbounds[0].(map[string]any)["blackhole_settings"].([]any)[0].(map[string]any)
	if bh["response_type"] != "none" {
		t.Fatalf("expected response_type='none' for empty blackhole settings, got %v", bh["response_type"])
	}
}

func TestExpandReverseEntries(t *testing.T) {
	list := []any{
		map[string]any{"tag": "b1", "domain": "test.com"},
	}
	result := expandReverseEntries(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 entry")
	}
	m := result[0].(map[string]any)
	if m["tag"] != "b1" || m["domain"] != "test.com" {
		t.Fatalf("unexpected: %v", m)
	}
}

func TestExpandRoutingRules(t *testing.T) {
	list := []any{
		map[string]any{
			"type":         "field",
			"ip":           []any{"geoip:private"},
			"outbound_tag": "blocked",
		},
	}
	result := expandRoutingRules(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 rule")
	}
	r := result[0].(map[string]any)
	if r["type"] != "field" {
		t.Fatalf("expected field type")
	}
	if r["outboundTag"] != "blocked" {
		t.Fatalf("expected outboundTag blocked, got %v", r["outboundTag"])
	}
	ips, ok := r["ip"].([]string)
	if !ok || len(ips) != 1 || ips[0] != "geoip:private" {
		t.Fatalf("unexpected ips: %v", r["ip"])
	}
}

func TestExpandRoutingRules_SkipsInternalAPIRule(t *testing.T) {
	list := []any{
		map[string]any{
			"type":         "field",
			"inbound_tag":  []any{"api"},
			"outbound_tag": "api",
		},
		map[string]any{
			"type":         "field",
			"ip":           []any{"geoip:private"},
			"outbound_tag": "blocked",
		},
	}
	result := expandRoutingRules(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 rule (api rule filtered), got %d", len(result))
	}
	if result[0].(map[string]any)["outboundTag"] != "blocked" {
		t.Fatalf("expected blocked rule to remain, got %v", result[0])
	}
}

func TestBuildXrayRoutingJSON_AlwaysIncludesAPIRule(t *testing.T) {
	input := map[string]any{
		"domain_strategy": "AsIs",
		"rule": []any{
			map[string]any{"type": "field", "ip": []any{"geoip:private"}, "outbound_tag": "blocked"},
		},
	}
	result := buildXrayRoutingJSON(input).(map[string]any)
	rules, ok := result["rules"].([]any)
	if !ok || len(rules) != 2 {
		t.Fatalf("expected 2 rules (api + user), got %v", result["rules"])
	}
	api := rules[0].(map[string]any)
	if api["outboundTag"] != "api" {
		t.Fatalf("expected first rule to be api rule, got %v", api)
	}
	if api["type"] != "field" {
		t.Fatalf("expected api rule type field, got %v", api["type"])
	}
	tags, ok := api["inboundTag"].([]string)
	if !ok || len(tags) != 1 || tags[0] != "api" {
		t.Fatalf("expected inboundTag [api], got %v", api["inboundTag"])
	}
	user := rules[1].(map[string]any)
	if user["outboundTag"] != "blocked" {
		t.Fatalf("expected second rule to be user rule, got %v", user)
	}
}

func TestBuildXrayRoutingJSON_APIRuleRoundtrip(t *testing.T) {
	input := map[string]any{
		"domain_strategy": "IPIfNonMatch",
		"rule": []any{
			map[string]any{"type": "field", "ip": []any{"geoip:private"}, "outbound_tag": "direct"},
		},
	}
	built := buildXrayRoutingJSON(input)
	flattened := flattenXrayRoutingToMap(built)

	if flattened["domain_strategy"] != "IPIfNonMatch" {
		t.Fatalf("expected IPIfNonMatch, got %v", flattened["domain_strategy"])
	}
	rules := flattened["rule"].([]any)
	if len(rules) != 1 {
		t.Fatalf("expected 1 user rule after flatten (api rule hidden), got %d", len(rules))
	}
	if rules[0].(map[string]any)["outbound_tag"] != "direct" {
		t.Fatalf("expected direct, got %v", rules[0].(map[string]any)["outbound_tag"])
	}

	// Second build from flattened state should produce identical result
	built2 := buildXrayRoutingJSON(flattened)
	flattened2 := flattenXrayRoutingToMap(built2)
	rules2 := flattened2["rule"].([]any)
	if len(rules2) != 1 {
		t.Fatalf("expected 1 user rule after second roundtrip, got %d", len(rules2))
	}
	if rules2[0].(map[string]any)["outbound_tag"] != "direct" {
		t.Fatalf("expected direct after second roundtrip, got %v", rules2[0])
	}
}

func TestBuildXrayRoutingJSON_OnlyAPIRuleNoDrift(t *testing.T) {
	input := map[string]any{
		"domain_strategy": "AsIs",
		"rule": []any{
			map[string]any{
				"type":         "field",
				"inbound_tag":  []any{"api"},
				"outbound_tag": "api",
			},
		},
	}
	built := buildXrayRoutingJSON(input).(map[string]any)
	rules := built["rules"].([]any)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule (api only), got %d", len(rules))
	}
	if rules[0].(map[string]any)["outboundTag"] != "api" {
		t.Fatalf("expected api rule, got %v", rules[0])
	}

	flattened := flattenXrayRoutingToMap(built)
	flatRules, _ := flattened["rule"].([]any)
	if len(flatRules) != 0 {
		t.Fatalf("expected 0 rules in state after flatten, got %d", len(flatRules))
	}

	// Second cycle: build from flattened (no user rules) → flatten → should still be empty
	built2 := buildXrayRoutingJSON(flattened)
	flattened2 := flattenXrayRoutingToMap(built2)
	flatRules2, _ := flattened2["rule"].([]any)
	if len(flatRules2) != 0 {
		t.Fatalf("expected 0 rules after second roundtrip, got %d", len(flatRules2))
	}
}

func TestFlattenWireguardOutSettings(t *testing.T) {
	in := map[string]any{
		"secretKey":      "test-key",
		"address":        []any{"10.0.0.2/32"},
		"mtu":            float64(1420),
		"workers":        float64(2),
		"domainStrategy": "ForceIPv6v4",
		"reserved":       []any{float64(1), float64(2), float64(3)},
		"noKernelTun":    false,
		"peers": []any{
			map[string]any{
				"publicKey":  "pub-key",
				"endpoint":   "engage.cloudflareclient.com:2408",
				"allowedIPs": []any{"0.0.0.0/0", "::/0"},
				"keepAlive":  float64(30),
			},
		},
	}
	result := flattenWireguardOutSettings(in)
	if result["secret_key"] != "test-key" {
		t.Fatalf("expected test-key, got %v", result["secret_key"])
	}
	if result["domain_strategy"] != "ForceIPv6v4" {
		t.Fatalf("expected ForceIPv6v4, got %v", result["domain_strategy"])
	}
	peers, ok := result["peer"].([]any)
	if !ok || len(peers) != 1 {
		t.Fatalf("expected 1 peer, got %v", result["peer"])
	}
	p := peers[0].(map[string]any)
	if p["public_key"] != "pub-key" {
		t.Fatalf("expected pub-key, got %v", p["public_key"])
	}
	reserved, ok := result["reserved"].([]int)
	if !ok || !reflect.DeepEqual(reserved, []int{1, 2, 3}) {
		t.Fatalf("expected [1,2,3], got %v", result["reserved"])
	}
}

func TestFlattenBasicsPolicyLevels(t *testing.T) {
	in := map[string]any{
		"0": map[string]any{
			"handshake":         float64(4),
			"connIdle":          float64(300),
			"statsUserUplink":   false,
			"statsUserDownlink": false,
		},
	}
	result := flattenBasicsPolicyLevels(in)
	if len(result) != 1 {
		t.Fatalf("expected 1 level, got %d", len(result))
	}
	level := result[0].(map[string]any)
	if level["id"] != 0 {
		t.Fatalf("expected id 0, got %v", level["id"])
	}
	if level["handshake"] != 4 {
		t.Fatalf("expected handshake 4, got %v", level["handshake"])
	}
}

// --- Expand unit tests ---

func TestExpandBasicsLog(t *testing.T) {
	item := map[string]any{
		"loglevel": "warning",
		"access":   "/var/log/access.log",
		"error":    "/var/log/error.log",
		"dns_log":  true,
	}
	result := expandBasicsLog(item)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result["loglevel"] != "warning" {
		t.Fatalf("expected warning, got %v", result["loglevel"])
	}
	if result["access"] != "/var/log/access.log" {
		t.Fatalf("expected access path, got %v", result["access"])
	}
	if result["error"] != "/var/log/error.log" {
		t.Fatalf("expected error path, got %v", result["error"])
	}
	if result["dnsLog"] != true {
		t.Fatalf("expected dnsLog true, got %v", result["dnsLog"])
	}
}

func TestExpandBasicsLog_Empty(t *testing.T) {
	result := expandBasicsLog(map[string]any{})
	if result != nil {
		t.Fatalf("expected nil for empty map, got %v", result)
	}
}

func TestExpandBasicsPolicy(t *testing.T) {
	item := map[string]any{
		"system": map[string]any{
			"stats_inbound_downlink":  true,
			"stats_inbound_uplink":    true,
			"stats_outbound_downlink": false,
			"stats_outbound_uplink":   false,
		},
		"level": []any{
			map[string]any{
				"id":                  0,
				"handshake":           4,
				"conn_idle":           300,
				"stats_user_uplink":   true,
				"stats_user_downlink": true,
			},
		},
	}
	result := expandBasicsPolicy(item)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	sys, ok := result["system"].(map[string]any)
	if !ok {
		t.Fatalf("expected system map, got %T", result["system"])
	}
	if sys["statsInboundDownlink"] != true {
		t.Fatalf("expected statsInboundDownlink true, got %v", sys["statsInboundDownlink"])
	}

	levels, ok := result["levels"].(map[string]any)
	if !ok {
		t.Fatalf("expected levels map, got %T", result["levels"])
	}
	level0, ok := levels["0"].(map[string]any)
	if !ok {
		t.Fatalf("expected level 0 map")
	}
	if level0["handshake"] != 4 {
		t.Fatalf("expected handshake 4, got %v", level0["handshake"])
	}
	if level0["connIdle"] != 300 {
		t.Fatalf("expected connIdle 300, got %v", level0["connIdle"])
	}
}

func TestExpandBasicsAPI(t *testing.T) {
	item := map[string]any{
		"tag":      "api",
		"services": []any{"HandlerService", "StatsService"},
	}
	result := expandBasicsAPI(item)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result["tag"] != "api" {
		t.Fatalf("expected tag api, got %v", result["tag"])
	}
	services, ok := result["services"].([]string)
	if !ok || len(services) != 2 {
		t.Fatalf("expected 2 services, got %v", result["services"])
	}
	if services[0] != "HandlerService" {
		t.Fatalf("expected HandlerService, got %v", services[0])
	}
}

func TestExpandBasicsAPI_Empty(t *testing.T) {
	result := expandBasicsAPI(map[string]any{})
	if result != nil {
		t.Fatalf("expected nil for empty map, got %v", result)
	}
}

func TestExpandBalancers(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "bal1",
			"selector": []any{"proxy-*"},
			"strategy": []any{
				map[string]any{"type": "leastPing"},
			},
		},
	}
	result := expandBalancers(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 balancer, got %d", len(result))
	}
	m := result[0].(map[string]any)
	if m["tag"] != "bal1" {
		t.Fatalf("expected bal1, got %v", m["tag"])
	}
	sel, ok := m["selector"].([]string)
	if !ok || len(sel) != 1 || sel[0] != "proxy-*" {
		t.Fatalf("unexpected selector: %v", m["selector"])
	}
	strategy, ok := m["strategy"].(map[string]any)
	if !ok || strategy["type"] != "leastPing" {
		t.Fatalf("unexpected strategy: %v", m["strategy"])
	}
}

func TestExpandOutbounds_Freedom(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "direct",
			"protocol": "freedom",
			"freedom_settings": []any{
				map[string]any{
					"domain_strategy": "AsIs",
					"fragment": []any{
						map[string]any{
							"packets":  "tlshello",
							"length":   "100-200",
							"interval": "10-20",
						},
					},
					"noises": []any{
						map[string]any{
							"type":   "rand",
							"packet": "10-20",
							"delay":  "10-16",
						},
					},
				},
			},
		},
	}
	result := expandOutbounds(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(result))
	}
	m := result[0].(map[string]any)
	if m["protocol"] != "freedom" {
		t.Fatalf("expected freedom, got %v", m["protocol"])
	}
	settings, ok := m["settings"].(map[string]any)
	if !ok {
		t.Fatalf("expected settings map, got %T", m["settings"])
	}
	if settings["domainStrategy"] != "AsIs" {
		t.Fatalf("expected AsIs, got %v", settings["domainStrategy"])
	}
	fragment, ok := settings["fragment"].(map[string]any)
	if !ok {
		t.Fatalf("expected fragment map, got %T", settings["fragment"])
	}
	if fragment["packets"] != "tlshello" {
		t.Fatalf("expected tlshello, got %v", fragment["packets"])
	}
	noises, ok := settings["noises"].([]any)
	if !ok || len(noises) != 1 {
		t.Fatalf("expected 1 noise, got %v", settings["noises"])
	}
}

func TestExpandOutbounds_Vmess(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "vmess-out",
			"protocol": "vmess",
			"vmess_settings": []any{
				map[string]any{
					"address":  "example.com",
					"port":     443,
					"id":       "test-uuid",
					"security": "auto",
				},
			},
		},
	}
	result := expandOutbounds(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 outbound")
	}
	settings := result[0].(map[string]any)["settings"].(map[string]any)
	vnext, ok := settings["vnext"].([]any)
	if !ok || len(vnext) != 1 {
		t.Fatalf("expected vnext with 1 server, got %v", settings["vnext"])
	}
	server := vnext[0].(map[string]any)
	if server["address"] != "example.com" {
		t.Fatalf("expected example.com, got %v", server["address"])
	}
	users := server["users"].([]any)
	if len(users) != 1 {
		t.Fatalf("expected 1 user")
	}
	user := users[0].(map[string]any)
	if user["id"] != "test-uuid" || user["security"] != "auto" {
		t.Fatalf("unexpected user: %v", user)
	}
}

func TestExpandOutbounds_Vless(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "vless-out",
			"protocol": "vless",
			"vless_settings": []any{
				map[string]any{
					"address":    "example.com",
					"port":       443,
					"id":         "test-uuid",
					"flow":       "xtls-rprx-vision",
					"encryption": "none",
				},
			},
		},
	}
	result := expandOutbounds(list)
	settings := result[0].(map[string]any)["settings"].(map[string]any)
	vnext := settings["vnext"].([]any)
	server := vnext[0].(map[string]any)
	user := server["users"].([]any)[0].(map[string]any)
	if user["flow"] != "xtls-rprx-vision" {
		t.Fatalf("expected xtls-rprx-vision, got %v", user["flow"])
	}
	if user["encryption"] != "none" {
		t.Fatalf("expected none, got %v", user["encryption"])
	}
}

func TestExpandOutbounds_Trojan(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "trojan-out",
			"protocol": "trojan",
			"trojan_settings": []any{
				map[string]any{
					"address":  "example.com",
					"port":     443,
					"password": "secret",
				},
			},
		},
	}
	result := expandOutbounds(list)
	settings := result[0].(map[string]any)["settings"].(map[string]any)
	servers := settings["servers"].([]any)
	if len(servers) != 1 {
		t.Fatalf("expected 1 server")
	}
	server := servers[0].(map[string]any)
	if server["password"] != "secret" {
		t.Fatalf("expected secret, got %v", server["password"])
	}
}

func TestExpandOutbounds_Socks(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "socks-out",
			"protocol": "socks",
			"socks_settings": []any{
				map[string]any{
					"address": "127.0.0.1",
					"port":    1080,
					"user":    "admin",
					"pass":    "password",
				},
			},
		},
	}
	result := expandOutbounds(list)
	settings := result[0].(map[string]any)["settings"].(map[string]any)
	servers := settings["servers"].([]any)
	server := servers[0].(map[string]any)
	if server["address"] != "127.0.0.1" {
		t.Fatalf("expected 127.0.0.1, got %v", server["address"])
	}
	users := server["users"].([]any)
	user := users[0].(map[string]any)
	if user["user"] != "admin" || user["pass"] != "password" {
		t.Fatalf("unexpected user: %v", user)
	}
}

func TestExpandOutbounds_HTTP(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "http-out",
			"protocol": "http",
			"http_settings": []any{
				map[string]any{
					"address": "proxy.example.com",
					"port":    8080,
					"user":    "user1",
					"pass":    "pass1",
				},
			},
		},
	}
	result := expandOutbounds(list)
	settings := result[0].(map[string]any)["settings"].(map[string]any)
	servers := settings["servers"].([]any)
	server := servers[0].(map[string]any)
	if server["address"] != "proxy.example.com" {
		t.Fatalf("expected proxy.example.com, got %v", server["address"])
	}
}

func TestExpandOutbounds_Hysteria(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "hysteria-out",
			"protocol": "hysteria",
			"hysteria_settings": []any{
				map[string]any{
					"address": "example.com",
					"port":    443,
					"version": 2,
				},
			},
		},
	}
	result := expandOutbounds(list)
	settings := result[0].(map[string]any)["settings"].(map[string]any)
	servers := settings["servers"].([]any)
	server := servers[0].(map[string]any)
	if server["version"] != 2 {
		t.Fatalf("expected version 2, got %v", server["version"])
	}
}

func TestFlattenOutbounds_DNS(t *testing.T) {
	data := []any{
		map[string]any{
			"tag":      "dns-out",
			"protocol": "dns",
			"settings": map[string]any{
				"network": "udp",
				"address": "1.1.1.1",
				"port":    float64(53),
			},
		},
	}
	result := flattenXrayOutboundsToMap(data)
	outbounds := result["outbound"].([]any)
	o := outbounds[0].(map[string]any)
	dnsSettings := o["dns_settings"].([]any)[0].(map[string]any)
	if dnsSettings["network"] != "udp" {
		t.Fatalf("expected udp, got %v", dnsSettings["network"])
	}
	if dnsSettings["address"] != "1.1.1.1" {
		t.Fatalf("expected 1.1.1.1, got %v", dnsSettings["address"])
	}
	if dnsSettings["port"] != 53 {
		t.Fatalf("expected 53, got %v", dnsSettings["port"])
	}
}

func TestFlattenOutbounds_Vmess(t *testing.T) {
	data := []any{
		map[string]any{
			"tag":      "vmess-out",
			"protocol": "vmess",
			"settings": map[string]any{
				"vnext": []any{
					map[string]any{
						"address": "example.com",
						"port":    float64(443),
						"users": []any{
							map[string]any{"id": "uuid-1", "security": "auto"},
						},
					},
				},
			},
		},
	}
	result := flattenXrayOutboundsToMap(data)
	outbounds := result["outbound"].([]any)
	o := outbounds[0].(map[string]any)
	vmess := o["vmess_settings"].([]any)[0].(map[string]any)
	if vmess["address"] != "example.com" {
		t.Fatalf("expected example.com, got %v", vmess["address"])
	}
	if vmess["id"] != "uuid-1" {
		t.Fatalf("expected uuid-1, got %v", vmess["id"])
	}
	if vmess["security"] != "auto" {
		t.Fatalf("expected auto, got %v", vmess["security"])
	}
}

func TestFlattenOutbounds_Vless(t *testing.T) {
	data := []any{
		map[string]any{
			"tag":      "vless-out",
			"protocol": "vless",
			"settings": map[string]any{
				"vnext": []any{
					map[string]any{
						"address": "example.com",
						"port":    float64(443),
						"users": []any{
							map[string]any{"id": "uuid-2", "flow": "xtls-rprx-vision", "encryption": "none"},
						},
					},
				},
			},
		},
	}
	result := flattenXrayOutboundsToMap(data)
	outbounds := result["outbound"].([]any)
	vless := outbounds[0].(map[string]any)["vless_settings"].([]any)[0].(map[string]any)
	if vless["flow"] != "xtls-rprx-vision" {
		t.Fatalf("expected xtls-rprx-vision, got %v", vless["flow"])
	}
}

func TestFlattenOutbounds_Trojan(t *testing.T) {
	data := []any{
		map[string]any{
			"tag":      "trojan-out",
			"protocol": "trojan",
			"settings": map[string]any{
				"servers": []any{
					map[string]any{"address": "example.com", "port": float64(443), "password": "secret"},
				},
			},
		},
	}
	result := flattenXrayOutboundsToMap(data)
	trojan := result["outbound"].([]any)[0].(map[string]any)["trojan_settings"].([]any)[0].(map[string]any)
	if trojan["password"] != "secret" {
		t.Fatalf("expected secret, got %v", trojan["password"])
	}
}

func TestFlattenOutbounds_Socks(t *testing.T) {
	data := []any{
		map[string]any{
			"tag":      "socks-out",
			"protocol": "socks",
			"settings": map[string]any{
				"servers": []any{
					map[string]any{
						"address": "127.0.0.1",
						"port":    float64(1080),
						"users": []any{
							map[string]any{"user": "admin", "pass": "pass"},
						},
					},
				},
			},
		},
	}
	result := flattenXrayOutboundsToMap(data)
	socks := result["outbound"].([]any)[0].(map[string]any)["socks_settings"].([]any)[0].(map[string]any)
	if socks["user"] != "admin" {
		t.Fatalf("expected admin, got %v", socks["user"])
	}
	if socks["pass"] != "pass" {
		t.Fatalf("expected pass, got %v", socks["pass"])
	}
}

func TestFlattenOutbounds_HTTP(t *testing.T) {
	data := []any{
		map[string]any{
			"tag":      "http-out",
			"protocol": "http",
			"settings": map[string]any{
				"servers": []any{
					map[string]any{
						"address": "proxy.example.com",
						"port":    float64(8080),
						"users": []any{
							map[string]any{"user": "user1", "pass": "pass1"},
						},
					},
				},
			},
		},
	}
	result := flattenXrayOutboundsToMap(data)
	http := result["outbound"].([]any)[0].(map[string]any)["http_settings"].([]any)[0].(map[string]any)
	if http["user"] != "user1" {
		t.Fatalf("expected user1, got %v", http["user"])
	}
}

func TestFlattenOutbounds_Hysteria(t *testing.T) {
	data := []any{
		map[string]any{
			"tag":      "hysteria-out",
			"protocol": "hysteria",
			"settings": map[string]any{
				"servers": []any{
					map[string]any{"address": "example.com", "port": float64(443), "version": float64(2)},
				},
			},
		},
	}
	result := flattenXrayOutboundsToMap(data)
	hysteria := result["outbound"].([]any)[0].(map[string]any)["hysteria_settings"].([]any)[0].(map[string]any)
	if hysteria["version"] != 2 {
		t.Fatalf("expected 2, got %v", hysteria["version"])
	}
}

func TestExpandOutbounds_Blackhole(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "blocked",
			"protocol": "blackhole",
			"blackhole_settings": []any{
				map[string]any{"response_type": "http"},
			},
		},
	}
	result := expandOutbounds(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(result))
	}
	settings := result[0].(map[string]any)["settings"].(map[string]any)
	resp := settings["response"].(map[string]any)
	if resp["type"] != "http" {
		t.Fatalf("expected http, got %v", resp["type"])
	}
}

func TestExpandOutbounds_DNS(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "dns-out",
			"protocol": "dns",
			"dns_settings": []any{
				map[string]any{"network": "udp", "address": "1.1.1.1", "port": 53},
			},
		},
	}
	result := expandOutbounds(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(result))
	}
	settings := result[0].(map[string]any)["settings"].(map[string]any)
	if settings["network"] != "udp" {
		t.Fatalf("expected udp, got %v", settings["network"])
	}
	if settings["address"] != "1.1.1.1" {
		t.Fatalf("expected 1.1.1.1, got %v", settings["address"])
	}
	if settings["port"] != 53 {
		t.Fatalf("expected 53, got %v", settings["port"])
	}
}

func TestExpandOutbounds_Shadowsocks(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "ss-out",
			"protocol": "shadowsocks",
			"shadowsocks_settings": []any{
				map[string]any{
					"address": "ss.example.com", "port": 8388,
					"password": "secret", "method": "aes-256-gcm", "uot": true,
				},
			},
		},
	}
	result := expandOutbounds(list)
	server := result[0].(map[string]any)["settings"].(map[string]any)["servers"].([]any)[0].(map[string]any)
	if server["address"] != "ss.example.com" {
		t.Fatalf("expected ss.example.com, got %v", server["address"])
	}
	if server["method"] != "aes-256-gcm" {
		t.Fatalf("expected aes-256-gcm, got %v", server["method"])
	}
	if server["uot"] != true {
		t.Fatalf("expected uot true, got %v", server["uot"])
	}
}

func TestExpandOutbounds_Wireguard(t *testing.T) {
	list := []any{
		map[string]any{
			"tag": "wg-out", "protocol": "wireguard",
			"wireguard_settings": []any{
				map[string]any{
					"secret_key": "wg-secret", "address": []any{"10.0.0.2/32"},
					"mtu": 1420, "domain_strategy": "ForceIPv4",
					"peer": []any{
						map[string]any{
							"public_key": "wg-pub", "endpoint": "wg.example.com:51820",
							"allowed_ips": []any{"0.0.0.0/0"}, "keep_alive": 25,
						},
					},
				},
			},
		},
	}
	result := expandOutbounds(list)
	settings := result[0].(map[string]any)["settings"].(map[string]any)
	if settings["secretKey"] != "wg-secret" {
		t.Fatalf("expected wg-secret, got %v", settings["secretKey"])
	}
	if settings["mtu"] != 1420 {
		t.Fatalf("expected 1420, got %v", settings["mtu"])
	}
	peers := settings["peers"].([]any)
	if len(peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(peers))
	}
	if peers[0].(map[string]any)["publicKey"] != "wg-pub" {
		t.Fatalf("expected wg-pub, got %v", peers[0].(map[string]any)["publicKey"])
	}
}

func TestFlattenOutbounds_Shadowsocks(t *testing.T) {
	data := []any{
		map[string]any{
			"tag": "ss-out", "protocol": "shadowsocks",
			"settings": map[string]any{
				"servers": []any{
					map[string]any{
						"address": "ss.example.com", "port": float64(8388),
						"password": "secret", "method": "aes-256-gcm", "uot": true,
					},
				},
			},
		},
	}
	result := flattenXrayOutboundsToMap(data)
	ss := result["outbound"].([]any)[0].(map[string]any)["shadowsocks_settings"].([]any)[0].(map[string]any)
	if ss["address"] != "ss.example.com" {
		t.Fatalf("expected ss.example.com, got %v", ss["address"])
	}
	if ss["method"] != "aes-256-gcm" {
		t.Fatalf("expected aes-256-gcm, got %v", ss["method"])
	}
	if ss["uot"] != true {
		t.Fatalf("expected uot true, got %v", ss["uot"])
	}
}

func TestExpandOutboundMux(t *testing.T) {
	list := []any{
		map[string]any{
			"enabled": true, "concurrency": 8,
			"xudp_concurrency": 16, "xudp_proxy_udp443": "reject",
		},
	}
	result := expandOutboundMux(list)
	if result == nil {
		t.Fatal("expected non-nil mux")
	}
	if result["enabled"] != true {
		t.Fatalf("expected enabled true, got %v", result["enabled"])
	}
	if result["concurrency"] != 8 {
		t.Fatalf("expected 8, got %v", result["concurrency"])
	}
	if result["xudpConcurrency"] != 16 {
		t.Fatalf("expected 16, got %v", result["xudpConcurrency"])
	}
	if result["xudpProxyUDP443"] != "reject" {
		t.Fatalf("expected reject, got %v", result["xudpProxyUDP443"])
	}
}

func TestFlattenOutboundMux(t *testing.T) {
	in := map[string]any{
		"enabled": true, "concurrency": float64(8),
		"xudpConcurrency": float64(16), "xudpProxyUDP443": "reject",
	}
	result := flattenOutboundMux(in)
	if result == nil {
		t.Fatal("expected non-nil mux")
	}
	if result["enabled"] != true {
		t.Fatalf("expected true, got %v", result["enabled"])
	}
	if result["concurrency"] != 8 {
		t.Fatalf("expected 8, got %v", result["concurrency"])
	}
	if result["xudp_concurrency"] != 16 {
		t.Fatalf("expected 16, got %v", result["xudp_concurrency"])
	}
	if result["xudp_proxy_udp443"] != "reject" {
		t.Fatalf("expected reject, got %v", result["xudp_proxy_udp443"])
	}
}

func TestExpandDNSServers_WithAllFields(t *testing.T) {
	list := []any{
		map[string]any{
			"address": "dns.example.com", "port": 53,
			"domains":       []any{"example.com", "example.org"},
			"expect_ips":    []any{"1.2.3.0/24"},
			"skip_fallback": true, "query_strategy": "UseIPv4",
		},
	}
	result := expandDNSServers(list)
	m := result[0].(map[string]any)
	if m["address"] != "dns.example.com" {
		t.Fatalf("expected dns.example.com, got %v", m["address"])
	}
	if m["port"] != 53 {
		t.Fatalf("expected 53, got %v", m["port"])
	}
	if m["skipFallback"] != true {
		t.Fatalf("expected skipFallback true, got %v", m["skipFallback"])
	}
	if m["queryStrategy"] != "UseIPv4" {
		t.Fatalf("expected UseIPv4, got %v", m["queryStrategy"])
	}
}

func TestBuildXrayDNSJSON_Roundtrip(t *testing.T) {
	input := map[string]any{
		"server": []any{
			map[string]any{"address": "8.8.8.8"},
			map[string]any{"address": "localhost", "port": 53, "domains": []any{"example.com"}},
		},
		"query_strategy": "UseIP",
	}
	flattened := flattenXrayDNSToMap(buildXrayDNSJSON(input))
	if flattened["query_strategy"] != "UseIP" {
		t.Fatalf("expected UseIP, got %v", flattened["query_strategy"])
	}
	servers := flattened["server"].([]any)
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
	if servers[0].(map[string]any)["address"] != "8.8.8.8" {
		t.Fatalf("expected 8.8.8.8, got %v", servers[0].(map[string]any)["address"])
	}
}

func TestBuildXrayDNSJSON_NewFields_Roundtrip(t *testing.T) {
	input := map[string]any{
		"server": []any{
			map[string]any{"address": "8.8.8.8"},
		},
		"query_strategy":        "UseIP",
		"enable_parallel_query": true,
		"use_system_hosts":      false,
	}
	flattened := flattenXrayDNSToMap(buildXrayDNSJSON(input))
	if flattened["enable_parallel_query"] != true {
		t.Fatalf("expected enable_parallel_query true, got %v", flattened["enable_parallel_query"])
	}
	if flattened["use_system_hosts"] != false {
		t.Fatalf("expected use_system_hosts false, got %v", flattened["use_system_hosts"])
	}
}

func TestFlattenXrayDNSToMap_NewFields(t *testing.T) {
	data := map[string]any{
		"servers":             []any{"8.8.8.8"},
		"queryStrategy":       "UseIP",
		"enableParallelQuery": true,
		"useSystemHosts":      true,
	}
	result := flattenXrayDNSToMap(data)
	if result["enable_parallel_query"] != true {
		t.Fatalf("expected enable_parallel_query true, got %v", result["enable_parallel_query"])
	}
	if result["use_system_hosts"] != true {
		t.Fatalf("expected use_system_hosts true, got %v", result["use_system_hosts"])
	}
}

func TestFlattenXrayDNSToMap_NewFields_Missing(t *testing.T) {
	data := map[string]any{
		"servers":       []any{"8.8.8.8"},
		"queryStrategy": "UseIP",
	}
	result := flattenXrayDNSToMap(data)
	if _, ok := result["enable_parallel_query"]; ok {
		t.Fatalf("enable_parallel_query should be absent when not in payload")
	}
	if _, ok := result["use_system_hosts"]; ok {
		t.Fatalf("use_system_hosts should be absent when not in payload")
	}
}

func TestExpandXrayDNS_NewFields(t *testing.T) {
	m := &XrayDNSModel{
		ID:                     types.StringValue("xray_dns"),
		Hosts:                  types.MapNull(types.StringType),
		QueryStrategy:          types.StringValue("UseIP"),
		Tag:                    types.StringNull(),
		DisableCache:           types.BoolNull(),
		DisableFallback:        types.BoolNull(),
		DisableFallbackIfMatch: types.BoolNull(),
		ClientIP:               types.StringNull(),
		EnableParallelQuery:    types.BoolValue(true),
		UseSystemHosts:         types.BoolValue(false),
	}
	result := expandXrayDNS(m)
	if result["enable_parallel_query"] != true {
		t.Fatalf("expected enable_parallel_query true, got %v", result["enable_parallel_query"])
	}
	if result["use_system_hosts"] != false {
		t.Fatalf("expected use_system_hosts false, got %v", result["use_system_hosts"])
	}
}

func TestExpandXrayDNS_NewFields_Null(t *testing.T) {
	m := &XrayDNSModel{
		ID:                     types.StringValue("xray_dns"),
		Hosts:                  types.MapNull(types.StringType),
		QueryStrategy:          types.StringNull(),
		Tag:                    types.StringNull(),
		DisableCache:           types.BoolNull(),
		DisableFallback:        types.BoolNull(),
		DisableFallbackIfMatch: types.BoolNull(),
		ClientIP:               types.StringNull(),
		EnableParallelQuery:    types.BoolNull(),
		UseSystemHosts:         types.BoolNull(),
	}
	result := expandXrayDNS(m)
	if _, ok := result["enable_parallel_query"]; ok {
		t.Fatalf("null enable_parallel_query should not be in expanded map")
	}
	if _, ok := result["use_system_hosts"]; ok {
		t.Fatalf("null use_system_hosts should not be in expanded map")
	}
}

func TestFlattenXrayDNS_NewFields(t *testing.T) {
	data := map[string]any{
		"enable_parallel_query": true,
		"use_system_hosts":      false,
	}
	result := flattenXrayDNS(data)
	if result.EnableParallelQuery.ValueBool() != true {
		t.Fatalf("expected EnableParallelQuery true, got %v", result.EnableParallelQuery)
	}
	if result.UseSystemHosts.ValueBool() != false {
		t.Fatalf("expected UseSystemHosts false, got %v", result.UseSystemHosts)
	}
}

func TestFlattenXrayDNS_NewFields_Missing(t *testing.T) {
	data := map[string]any{}
	result := flattenXrayDNS(data)
	if !result.EnableParallelQuery.IsNull() {
		t.Fatalf("expected null EnableParallelQuery when not in data")
	}
	if !result.UseSystemHosts.IsNull() {
		t.Fatalf("expected null UseSystemHosts when not in data")
	}
}

func TestBuildXrayRoutingJSON_Roundtrip(t *testing.T) {
	input := map[string]any{
		"domain_strategy": "IPIfNonMatch",
		"rule": []any{
			map[string]any{"type": "field", "ip": []any{"geoip:private"}, "outbound_tag": "direct"},
		},
	}
	flattened := flattenXrayRoutingToMap(buildXrayRoutingJSON(input))
	if flattened["domain_strategy"] != "IPIfNonMatch" {
		t.Fatalf("expected IPIfNonMatch, got %v", flattened["domain_strategy"])
	}
	rules := flattened["rule"].([]any)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].(map[string]any)["outbound_tag"] != "direct" {
		t.Fatalf("expected direct, got %v", rules[0].(map[string]any)["outbound_tag"])
	}
}

func TestBuildXrayBasicsJSON_Roundtrip(t *testing.T) {
	input := map[string]any{
		"log":   map[string]any{"loglevel": "debug", "dns_log": true},
		"api":   map[string]any{"tag": "api", "services": []any{"HandlerService", "StatsService"}},
		"stats": map[string]any{},
	}
	flattened := flattenXrayBasicsToMap(buildXrayBasicsJSON(input))
	log := flattened["log"].(map[string]any)
	if log["loglevel"] != "debug" {
		t.Fatalf("expected debug, got %v", log["loglevel"])
	}
	if log["dns_log"] != true {
		t.Fatalf("expected dns_log true, got %v", log["dns_log"])
	}
	if _, ok := flattened["stats"]; !ok {
		t.Fatalf("expected stats block")
	}
}

func TestBuildXrayReverseJSON_Roundtrip(t *testing.T) {
	input := map[string]any{
		"bridge": []any{map[string]any{"tag": "b1", "domain": "bridge.example.com"}},
		"portal": []any{map[string]any{"tag": "p1", "domain": "portal.example.com"}},
	}
	flattened := flattenXrayReverseToMap(buildXrayReverseJSON(input))
	b := flattened["bridge"].([]any)[0].(map[string]any)
	if b["tag"] != "b1" || b["domain"] != "bridge.example.com" {
		t.Fatalf("unexpected bridge: %v", b)
	}
	p := flattened["portal"].([]any)[0].(map[string]any)
	if p["tag"] != "p1" || p["domain"] != "portal.example.com" {
		t.Fatalf("unexpected portal: %v", p)
	}
}

func TestBuildXrayBalancersJSON_Roundtrip(t *testing.T) {
	input := map[string]any{
		"balancer": []any{
			map[string]any{
				"tag": "bal1", "selector": []any{"proxy-*"},
				"strategy": []any{map[string]any{"type": "random"}},
			},
		},
	}
	flattened := flattenXrayBalancersToMap(buildXrayBalancersJSON(input))
	bal := flattened["balancer"].([]any)[0].(map[string]any)
	if bal["tag"] != "bal1" {
		t.Fatalf("expected bal1, got %v", bal["tag"])
	}
	strategy := bal["strategy"].([]any)[0].(map[string]any)
	if strategy["type"] != "random" {
		t.Fatalf("expected random, got %v", strategy["type"])
	}
}

func TestBuildXrayOutboundsJSON_Roundtrip(t *testing.T) {
	input := map[string]any{
		"outbound": []any{
			map[string]any{
				"tag": "direct", "protocol": "freedom",
				"freedom_settings": []any{map[string]any{"domain_strategy": "AsIs"}},
			},
			map[string]any{
				"tag": "blocked", "protocol": "blackhole",
				"blackhole_settings": []any{map[string]any{"response_type": "none"}},
			},
		},
	}
	flattened := flattenXrayOutboundsToMap(buildXrayOutboundsJSON(input))
	outbounds := flattened["outbound"].([]any)
	if len(outbounds) != 2 {
		t.Fatalf("expected 2 outbounds, got %d", len(outbounds))
	}
	freedom := outbounds[0].(map[string]any)["freedom_settings"].([]any)[0].(map[string]any)
	if freedom["domain_strategy"] != "AsIs" {
		t.Fatalf("expected AsIs, got %v", freedom["domain_strategy"])
	}
	bh := outbounds[1].(map[string]any)["blackhole_settings"].([]any)[0].(map[string]any)
	if bh["response_type"] != "none" {
		t.Fatalf("expected none, got %v", bh["response_type"])
	}
}

// Stream settings declared on an outbound must make it all the way through
// buildXrayOutboundsJSON into the camelCase streamSettings xray stores — the
// lossy round-trip that used to silently strip the TLS settings from vless
// outbounds (the whole outbounds array is rewritten via setJSONPath).
func TestBuildXrayOutboundsJSON_StreamSettings(t *testing.T) {
	input := map[string]any{
		"outbound": []any{
			map[string]any{
				"tag":      "proxy",
				"protocol": "vless",
				"vless_settings": []any{map[string]any{
					"address": "34.88.128.98", "port": 443, "id": "uuid",
					"encryption": "none",
				}},
				"stream_settings": []any{
					map[string]any{
						"network":  "tcp",
						"security": "tls",
						"tls_settings": []any{
							map[string]any{
								"server_name":    "gcp.aistreams.cloud",
								"fingerprint":    "chrome",
								"allow_insecure": false,
								"alpn":           []any{"h2", "http/1.1"},
							},
						},
					},
				},
			},
		},
	}
	wire := buildXrayOutboundsJSON(input)
	entries, ok := wire.([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("expected 1 outbound, got %#v", wire)
	}
	entry, ok := entries[0].(map[string]any)
	if !ok {
		t.Fatalf("entry not a map: %#v", entries[0])
	}
	ss, ok := entry["streamSettings"].(map[string]any)
	if !ok {
		t.Fatalf("expected streamSettings object, got %#v", entry)
	}
	if ss["security"] != "tls" {
		t.Fatalf("expected security=tls, got %v", ss["security"])
	}
	tls, ok := ss["tlsSettings"].(map[string]any)
	if !ok {
		t.Fatalf("expected tlsSettings object, got %#v", ss)
	}
	if tls["serverName"] != "gcp.aistreams.cloud" {
		t.Fatalf("expected serverName, got %v", tls["serverName"])
	}

	// The round-trip back into the snake_case map must preserve the block.
	flat := flattenXrayOutboundsToMap(wire)
	flattened := flat["outbound"].([]any)[0].(map[string]any)
	ssOut, ok := flattened["stream_settings"].([]any)
	if !ok || len(ssOut) != 1 {
		t.Fatalf("expected stream_settings on the read path, got %#v", flattened)
	}
	first := ssOut[0].(map[string]any)
	tlsOut, ok := first["tls_settings"].([]any)
	if !ok || len(tlsOut) != 1 {
		t.Fatalf("expected tls_settings on the read path, got %#v", first)
	}
	ts := tlsOut[0].(map[string]any)
	if ts["server_name"] != "gcp.aistreams.cloud" {
		t.Fatalf("expected server_name after round-trip, got %v", ts["server_name"])
	}
}

// An outbound without stream settings must NOT gain a streamSettings key — a
// hypothetical drift signal for every protocol that predates this feature.
func TestBuildXrayOutboundsJSON_StreamSettingsAbsent(t *testing.T) {
	input := map[string]any{
		"outbound": []any{
			map[string]any{
				"tag":              "direct",
				"protocol":         "freedom",
				"freedom_settings": []any{map[string]any{"domain_strategy": "AsIs"}},
			},
		},
	}
	wire := buildXrayOutboundsJSON(input)
	entry := wire.([]any)[0].(map[string]any)
	if _, ok := entry["streamSettings"]; ok {
		t.Fatalf("unexpected streamSettings key: %#v", entry)
	}
}

// --- mergeOutboundStreamSettingsFromState ---

func mergeTestOutboundObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"tag": types.StringType,
		"stream_settings": types.ObjectType{AttrTypes: map[string]attr.Type{
			"network":      types.StringType,
			"security":     types.StringType,
			"tls_settings": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{}}},
		}},
	}}
}

func mergeTestOutboundObj(objectType types.ObjectType, tag string, ss attr.Value) types.Object {
	attrs := map[string]attr.Value{
		"tag":             types.StringValue(tag),
		"stream_settings": ss,
	}
	return types.ObjectValueMust(objectType.AttrTypes, attrs)
}

func mergeTestStreamSettings() types.Object {
	return types.ObjectValueMust(map[string]attr.Type{
		"network":      types.StringType,
		"security":     types.StringType,
		"tls_settings": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{}}},
	}, map[string]attr.Value{
		"network":  types.StringValue("tcp"),
		"security": types.StringValue("tls"),
		"tls_settings": types.ListValueMust(
			types.ObjectType{AttrTypes: map[string]attr.Type{}},
			[]attr.Value{types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{})},
		),
	})
}

func TestMergeOutboundStreamSettingsFromState_PreservesUndeclared(t *testing.T) {
	obType := mergeTestOutboundObjectType()
	ssType := obType.AttrTypes["stream_settings"].(types.ObjectType)
	nullSS := types.ObjectNull(ssType.AttrTypes)
	configured := types.ListValueMust(obType, []attr.Value{
		mergeTestOutboundObj(obType, "proxy", nullSS),
	})
	state := types.ListValueMust(obType, []attr.Value{
		mergeTestOutboundObj(obType, "proxy", mergeTestStreamSettings()),
	})

	merged := mergeOutboundStreamSettingsFromState(configured, state)
	elems := merged.Elements()
	if len(elems) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elems))
	}
	obj := elems[0].(types.Object)
	saved := obj.Attributes()["stream_settings"]
	if saved.IsNull() || saved.IsUnknown() {
		t.Fatalf("expected stream_settings to be inherited from state, got %v", saved)
	}
	if got := saved.(types.Object).Attributes()["security"].(types.String).ValueString(); got != "tls" {
		t.Fatalf("expected security=tls, got %v", got)
	}
	if merged.Equal(configured) {
		t.Fatal("merged must differ from configured so ModifyPlan persists it")
	}
}

func TestMergeOutboundStreamSettingsFromState_DeclaredBlockWins(t *testing.T) {
	obType := mergeTestOutboundObjectType()
	ssType := obType.AttrTypes["stream_settings"].(types.ObjectType)
	declared := types.ObjectValueMust(ssType.AttrTypes, map[string]attr.Value{
		"network":  types.StringValue("ws"),
		"security": types.StringValue("none"),
		"tls_settings": types.ListNull(
			types.ObjectType{AttrTypes: map[string]attr.Type{}},
		),
	})
	configured := types.ListValueMust(obType, []attr.Value{
		mergeTestOutboundObj(obType, "proxy", declared),
	})
	state := types.ListValueMust(obType, []attr.Value{
		mergeTestOutboundObj(obType, "proxy", mergeTestStreamSettings()),
	})

	merged := mergeOutboundStreamSettingsFromState(configured, state)
	if !merged.Equal(configured) {
		t.Fatal("a declared stream_settings block must not be overwritten by state")
	}
	obj := merged.Elements()[0].(types.Object)
	if got := obj.Attributes()["stream_settings"].(types.Object).Attributes()["network"].(types.String).ValueString(); got != "ws" {
		t.Fatalf("expected network=ws from config, got %v", got)
	}
}

func TestMergeOutboundStreamSettingsFromState_NoState(t *testing.T) {
	obType := mergeTestOutboundObjectType()
	ssType := obType.AttrTypes["stream_settings"].(types.ObjectType)
	configured := types.ListValueMust(obType, []attr.Value{
		mergeTestOutboundObj(obType, "proxy", types.ObjectNull(ssType.AttrTypes)),
	})
	if merged := mergeOutboundStreamSettingsFromState(configured, types.ListNull(obType)); !merged.Equal(configured) {
		t.Fatal("null state must not alter the configured list")
	}
	if merged := mergeOutboundStreamSettingsFromState(configured, types.ListValueMust(obType, nil)); !merged.Equal(configured) {
		t.Fatal("empty state must not alter the configured list")
	}
}

func TestMergeOutboundStreamSettingsFromState_TagMismatch(t *testing.T) {
	obType := mergeTestOutboundObjectType()
	ssType := obType.AttrTypes["stream_settings"].(types.ObjectType)
	configured := types.ListValueMust(obType, []attr.Value{
		mergeTestOutboundObj(obType, "other", types.ObjectNull(ssType.AttrTypes)),
	})
	state := types.ListValueMust(obType, []attr.Value{
		mergeTestOutboundObj(obType, "proxy", mergeTestStreamSettings()),
	})
	if merged := mergeOutboundStreamSettingsFromState(configured, state); !merged.Equal(configured) {
		t.Fatal("different tags must not inherit stream_settings")
	}
}

// --- Tests for review fixes ---

func TestFlattenBasicsPolicyLevels_Sorted(t *testing.T) {
	in := map[string]any{
		"2": map[string]any{"handshake": float64(8)},
		"0": map[string]any{"handshake": float64(4)},
		"1": map[string]any{"handshake": float64(6)},
	}
	// Run 20 times to catch non-determinism.
	for i := 0; i < 20; i++ {
		result := flattenBasicsPolicyLevels(in)
		if len(result) != 3 {
			t.Fatalf("expected 3 levels, got %d", len(result))
		}
		ids := make([]int, len(result))
		for j, item := range result {
			ids[j] = item.(map[string]any)["id"].(int)
		}
		if ids[0] != 0 || ids[1] != 1 || ids[2] != 2 {
			t.Fatalf("iteration %d: expected sorted [0,1,2], got %v", i, ids)
		}
	}
}

func TestExpandInt64List_WithNullAndUnknown(t *testing.T) {
	elems := []attr.Value{
		types.Int64Value(1),
		types.Int64Null(),
		types.Int64Unknown(),
		types.Int64Value(3),
	}
	l, diags := types.ListValue(types.Int64Type, elems)
	if diags.HasError() {
		t.Fatalf("failed to create list: %v", diags)
	}
	result := expandInt64List(l)
	expected := []any{1, 3}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestExpandInt64List_NullList(t *testing.T) {
	l := types.ListNull(types.Int64Type)
	result := expandInt64List(l)
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestExpandInt64List_UnknownList(t *testing.T) {
	l := types.ListUnknown(types.Int64Type)
	result := expandInt64List(l)
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestExpandXrayOutbounds_EmptyMuxBlock(t *testing.T) {
	m := &XrayOutboundsModel{
		Outbound: []XrayOutboundEntry{
			{
				Tag:      types.StringValue("test"),
				Protocol: types.StringValue("freedom"),
				Mux: []XrayOutboundMux{
					{
						Enabled:         types.BoolNull(),
						Concurrency:     types.Int64Null(),
						XudpConcurrency: types.Int64Null(),
						XudpProxyUDP443: types.StringNull(),
					},
				},
			},
		},
	}
	result := expandXrayOutbounds(m)
	outbounds := result["outbound"].([]any)
	if len(outbounds) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(outbounds))
	}
	entry := outbounds[0].(map[string]any)
	if _, ok := entry["mux"]; ok {
		t.Fatalf("mux with all-null fields should not be in result")
	}
}

func TestExpandXrayOutbounds_EmptySettingsBlock(t *testing.T) {
	m := &XrayOutboundsModel{
		Outbound: []XrayOutboundEntry{
			{
				Tag:      types.StringValue("test"),
				Protocol: types.StringValue("blackhole"),
				BlackholeSettings: []XrayBlackholeSettings{
					{ResponseType: types.StringNull()},
				},
			},
		},
	}
	result := expandXrayOutbounds(m)
	outbounds := result["outbound"].([]any)
	entry := outbounds[0].(map[string]any)
	if _, ok := entry["blackhole_settings"]; ok {
		t.Fatalf("blackhole_settings with all-null fields should not be in result")
	}
}

func TestExpandOutbounds_FreedomIPsBlocked(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "direct",
			"protocol": "freedom",
			"freedom_settings": []any{
				map[string]any{
					"domain_strategy": "AsIs",
					"ips_blocked":     []any{"geoip:cn", "10.0.0.0/8"},
				},
			},
		},
	}
	result := expandOutbounds(list)
	if len(result) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(result))
	}
	m := result[0].(map[string]any)
	settings, ok := m["settings"].(map[string]any)
	if !ok {
		t.Fatalf("expected settings map, got %T", m["settings"])
	}
	ipsBlocked, ok := settings["ipsBlocked"].([]any)
	if !ok || len(ipsBlocked) != 2 {
		t.Fatalf("expected 2 ipsBlocked entries, got %v", settings["ipsBlocked"])
	}
	if ipsBlocked[0] != "geoip:cn" || ipsBlocked[1] != "10.0.0.0/8" {
		t.Fatalf("unexpected ipsBlocked values: %v", ipsBlocked)
	}
}

func TestExpandOutbounds_FreedomFinalRules(t *testing.T) {
	list := []any{
		map[string]any{
			"tag":      "direct",
			"protocol": "freedom",
			"freedom_settings": []any{
				map[string]any{
					"domain_strategy": "AsIs",
					"final_rule": []any{
						map[string]any{
							"action":      "block",
							"network":     "tcp",
							"port":        "443",
							"ip":          []any{"geoip:private"},
							"block_delay": "0",
						},
					},
				},
			},
		},
	}
	result := expandOutbounds(list)
	settings := result[0].(map[string]any)["settings"].(map[string]any)
	finalRules, ok := settings["finalRules"].([]any)
	if !ok || len(finalRules) != 1 {
		t.Fatalf("expected 1 finalRules entry, got %v", settings["finalRules"])
	}
	rule := finalRules[0].(map[string]any)
	if rule["blockDelay"] != "0" {
		t.Fatalf("expected blockDelay 0, got %v", rule["blockDelay"])
	}
	ips, ok := rule["ip"].([]string)
	if !ok || len(ips) != 1 || ips[0] != "geoip:private" {
		t.Fatalf("unexpected finalRules ip: %v", rule["ip"])
	}
}

func TestFlattenOutbounds_FreedomIPsBlocked(t *testing.T) {
	data := []any{
		map[string]any{
			"tag":      "direct",
			"protocol": "freedom",
			"settings": map[string]any{
				"domainStrategy": "AsIs",
				"ipsBlocked":     []any{"geoip:cn", "10.0.0.0/8"},
			},
		},
	}
	result := flattenXrayOutboundsToMap(data)
	outbounds := result["outbound"].([]any)
	if len(outbounds) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(outbounds))
	}
	freedom := outbounds[0].(map[string]any)["freedom_settings"].([]any)[0].(map[string]any)
	ipsBlocked, ok := freedom["ips_blocked"].([]any)
	if !ok || len(ipsBlocked) != 2 {
		t.Fatalf("expected 2 ips_blocked entries, got %v", freedom["ips_blocked"])
	}
	if ipsBlocked[0] != "geoip:cn" || ipsBlocked[1] != "10.0.0.0/8" {
		t.Fatalf("unexpected ips_blocked values: %v", ipsBlocked)
	}
}

func TestFlattenOutbounds_FreedomFinalRules(t *testing.T) {
	data := []any{
		map[string]any{
			"tag":      "direct",
			"protocol": "freedom",
			"settings": map[string]any{
				"domainStrategy": "AsIs",
				"finalRules": []any{
					map[string]any{
						"action":     "block",
						"network":    "tcp",
						"port":       "443",
						"ip":         []any{"geoip:private"},
						"blockDelay": "0",
					},
				},
			},
		},
	}
	result := flattenXrayOutboundsToMap(data)
	freedom := result["outbound"].([]any)[0].(map[string]any)["freedom_settings"].([]any)[0].(map[string]any)
	finalRules, ok := freedom["final_rule"].([]any)
	if !ok || len(finalRules) != 1 {
		t.Fatalf("expected 1 final_rule entry, got %v", freedom["final_rule"])
	}
	rule := finalRules[0].(map[string]any)
	if rule["block_delay"] != "0" {
		t.Fatalf("expected block_delay 0, got %v", rule["block_delay"])
	}
}

func TestOutboundsVlessReverseTagRoundtrip(t *testing.T) {
	input := map[string]any{
		"outbound": []any{
			map[string]any{
				"tag": "vless-out", "protocol": "vless",
				"vless_settings": []any{map[string]any{
					"address":     "example.com",
					"port":        443,
					"id":          "test-uuid",
					"encryption":  "none",
					"reverse_tag": "reverse-a",
				}},
			},
		},
	}
	flattened := flattenXrayOutboundsToMap(buildXrayOutboundsJSON(input))
	vless := flattened["outbound"].([]any)[0].(map[string]any)["vless_settings"].([]any)[0].(map[string]any)
	if vless["reverse_tag"] != "reverse-a" {
		t.Fatalf("expected reverse-a, got %v", vless["reverse_tag"])
	}
}

func TestFreedomIPsBlocked_Roundtrip(t *testing.T) {
	input := map[string]any{
		"outbound": []any{
			map[string]any{
				"tag": "direct", "protocol": "freedom",
				"freedom_settings": []any{map[string]any{
					"domain_strategy": "AsIs",
					"ips_blocked":     []any{"geoip:cn", "192.168.0.0/16"},
				}},
			},
		},
	}
	flattened := flattenXrayOutboundsToMap(buildXrayOutboundsJSON(input))
	outbounds := flattened["outbound"].([]any)
	if len(outbounds) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(outbounds))
	}
	freedom := outbounds[0].(map[string]any)["freedom_settings"].([]any)[0].(map[string]any)
	ipsBlocked, ok := freedom["ips_blocked"].([]any)
	if !ok || len(ipsBlocked) != 2 {
		t.Fatalf("expected 2 ips_blocked entries after roundtrip, got %v", freedom["ips_blocked"])
	}
	if ipsBlocked[0] != "geoip:cn" || ipsBlocked[1] != "192.168.0.0/16" {
		t.Fatalf("unexpected ips_blocked values: %v", ipsBlocked)
	}
}

func TestFreedomIPsBlocked_TypedModelRoundtrip(t *testing.T) {
	ipsBlockedList := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("geoip:cn"),
		types.StringValue("10.0.0.0/8"),
	})

	model := &XrayOutboundsModel{
		Outbound: []XrayOutboundEntry{
			{
				Tag:      types.StringValue("direct"),
				Protocol: types.StringValue("freedom"),
				FreedomSettings: []XrayFreedomSettings{
					{
						DomainStrategy: types.StringValue("AsIs"),
						Redirect:       types.StringNull(),
						IPsBlocked:     ipsBlockedList,
					},
				},
			},
		},
	}

	// Typed model -> untyped map
	expanded := expandXrayOutbounds(model)
	outbounds := expanded["outbound"].([]any)
	entry := outbounds[0].(map[string]any)
	fsList := entry["freedom_settings"].([]any)
	fs := fsList[0].(map[string]any)
	ips, ok := fs["ips_blocked"].([]any)
	if !ok || len(ips) != 2 {
		t.Fatalf("expected 2 ips_blocked in expanded map, got %v", fs["ips_blocked"])
	}
	if ips[0] != "geoip:cn" || ips[1] != "10.0.0.0/8" {
		t.Fatalf("unexpected expanded ips_blocked: %v", ips)
	}

	// Untyped map -> typed model (flatten back)
	flatModel := flattenXrayOutbounds(expanded)
	if len(flatModel.Outbound) != 1 {
		t.Fatalf("expected 1 outbound in flattened model, got %d", len(flatModel.Outbound))
	}
	flatFS := flatModel.Outbound[0].FreedomSettings
	if len(flatFS) != 1 {
		t.Fatalf("expected 1 freedom_settings, got %d", len(flatFS))
	}
	if flatFS[0].IPsBlocked.IsNull() || flatFS[0].IPsBlocked.IsUnknown() {
		t.Fatalf("expected non-null ips_blocked in flattened model")
	}
	elems := flatFS[0].IPsBlocked.Elements()
	if len(elems) != 2 {
		t.Fatalf("expected 2 elements in ips_blocked, got %d", len(elems))
	}
	if elems[0].(types.String).ValueString() != "geoip:cn" {
		t.Fatalf("expected geoip:cn, got %v", elems[0])
	}
	if elems[1].(types.String).ValueString() != "10.0.0.0/8" {
		t.Fatalf("expected 10.0.0.0/8, got %v", elems[1])
	}
}

func TestFreedomIPsBlocked_NullHandling(t *testing.T) {
	model := &XrayOutboundsModel{
		Outbound: []XrayOutboundEntry{
			{
				Tag:      types.StringValue("direct"),
				Protocol: types.StringValue("freedom"),
				FreedomSettings: []XrayFreedomSettings{
					{
						DomainStrategy: types.StringValue("AsIs"),
						Redirect:       types.StringNull(),
						IPsBlocked:     types.ListNull(types.StringType),
					},
				},
			},
		},
	}

	expanded := expandXrayOutbounds(model)
	outbounds := expanded["outbound"].([]any)
	entry := outbounds[0].(map[string]any)
	fsList := entry["freedom_settings"].([]any)
	fs := fsList[0].(map[string]any)
	if _, ok := fs["ips_blocked"]; ok {
		t.Fatalf("null ips_blocked should not appear in expanded map")
	}

	// Flatten with missing ips_blocked -> should get null list
	untypedMap := map[string]any{
		"outbound": []any{
			map[string]any{
				"protocol": "freedom",
				"freedom_settings": []any{
					map[string]any{
						"domain_strategy": "AsIs",
					},
				},
			},
		},
	}
	flatModel := flattenXrayOutbounds(untypedMap)
	flatFS := flatModel.Outbound[0].FreedomSettings[0]
	if !flatFS.IPsBlocked.IsNull() {
		t.Fatalf("expected null ips_blocked when key is missing, got %v", flatFS.IPsBlocked)
	}
}

func TestAlignBasicsBlocksWithPlan_NilsMissingBlocks(t *testing.T) {
	state := &XrayBasicsModel{
		ID: types.StringValue("xray_basics"),
		Log: []XrayBasicsLog{{
			Loglevel: types.StringValue("warning"),
			DNSLog:   types.BoolValue(false),
		}},
		Policy: []XrayBasicsPolicy{{
			System: []XrayBasicsPolicySystem{{
				StatsInboundDownlink: types.BoolValue(false),
			}},
			Level: []XrayBasicsPolicyLevel{{
				ID:           types.Int64Value(0),
				Handshake:    types.Int64Value(4),
				ConnIdle:     types.Int64Value(300),
				UplinkOnly:   types.Int64Value(2),
				DownlinkOnly: types.Int64Value(5),
				BufferSize:   types.Int64Value(4),
			}},
		}},
		API:   []XrayBasicsAPI{{Tag: types.StringValue("api")}},
		Stats: []XrayBasicsStats{{}},
	}

	plan := &XrayBasicsModel{
		ID: types.StringValue("xray_basics"),
		Log: []XrayBasicsLog{{
			Loglevel: types.StringValue("warning"),
			DNSLog:   types.BoolValue(false),
		}},
	}

	alignBasicsBlocksWithPlan(state, plan)

	if state.Policy != nil {
		t.Fatal("expected policy to be nil after alignment")
	}
	if state.API != nil {
		t.Fatal("expected api to be nil after alignment")
	}
	if state.Stats != nil {
		t.Fatal("expected stats to be nil after alignment")
	}
	if len(state.Log) != 1 {
		t.Fatal("expected log to remain")
	}
}

func TestAlignBasicsBlocksWithPlan_PreservesPresentBlocks(t *testing.T) {
	state := &XrayBasicsModel{
		ID:  types.StringValue("xray_basics"),
		Log: []XrayBasicsLog{{Loglevel: types.StringValue("warning")}},
		Policy: []XrayBasicsPolicy{{
			System: []XrayBasicsPolicySystem{{StatsInboundDownlink: types.BoolValue(false)}},
			Level:  []XrayBasicsPolicyLevel{{ID: types.Int64Value(0), Handshake: types.Int64Value(4)}},
		}},
		API:   []XrayBasicsAPI{{Tag: types.StringValue("api")}},
		Stats: []XrayBasicsStats{{}},
	}

	plan := &XrayBasicsModel{
		ID:     types.StringValue("xray_basics"),
		Log:    []XrayBasicsLog{{Loglevel: types.StringValue("warning")}},
		Policy: []XrayBasicsPolicy{{System: []XrayBasicsPolicySystem{{}}, Level: []XrayBasicsPolicyLevel{{ID: types.Int64Value(0)}}}},
		API:    []XrayBasicsAPI{{Tag: types.StringValue("api")}},
		Stats:  []XrayBasicsStats{{}},
	}

	alignBasicsBlocksWithPlan(state, plan)

	if state.Policy == nil {
		t.Fatal("expected policy to be preserved")
	}
	if state.API == nil {
		t.Fatal("expected api to be preserved")
	}
	if state.Stats == nil {
		t.Fatal("expected stats to be preserved")
	}
}

func TestAlignBasicsBlocksWithPlan_NilsNestedPolicyBlocks(t *testing.T) {
	state := &XrayBasicsModel{
		ID: types.StringValue("xray_basics"),
		Policy: []XrayBasicsPolicy{{
			System: []XrayBasicsPolicySystem{{StatsInboundDownlink: types.BoolValue(false)}},
			Level:  []XrayBasicsPolicyLevel{{ID: types.Int64Value(0), Handshake: types.Int64Value(4)}},
		}},
	}

	plan := &XrayBasicsModel{
		ID:     types.StringValue("xray_basics"),
		Policy: []XrayBasicsPolicy{{}},
	}

	alignBasicsBlocksWithPlan(state, plan)

	if len(state.Policy) == 0 {
		t.Fatal("expected policy block to remain")
	}
	if state.Policy[0].System != nil {
		t.Fatal("expected policy.system to be nil")
	}
	if state.Policy[0].Level != nil {
		t.Fatal("expected policy.level to be nil")
	}
}

func TestValidateNoAPIRoutingRules(t *testing.T) {
	tests := []struct {
		name    string
		rules   []XrayRoutingRule
		wantErr bool
	}{
		{
			name:    "no rules",
			rules:   []XrayRoutingRule{},
			wantErr: false,
		},
		{
			name: "normal rule",
			rules: []XrayRoutingRule{
				{
					Type:        types.StringValue("field"),
					OutboundTag: types.StringValue("blocked"),
					InboundTag:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("http")}),
				},
			},
			wantErr: false,
		},
		{
			name: "API routing rule",
			rules: []XrayRoutingRule{
				{
					Type:        types.StringValue("field"),
					OutboundTag: types.StringValue("api"),
					InboundTag:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("api")}),
				},
			},
			wantErr: true,
		},
		{
			name: "API outbound with non-api inbound",
			rules: []XrayRoutingRule{
				{
					Type:        types.StringValue("field"),
					OutboundTag: types.StringValue("api"),
					InboundTag:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("something-else")}),
				},
			},
			wantErr: false,
		},
		{
			name: "API rule mixed with normal rules",
			rules: []XrayRoutingRule{
				{
					Type:        types.StringValue("field"),
					OutboundTag: types.StringValue("blocked"),
					InboundTag:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("http")}),
				},
				{
					Type:        types.StringValue("field"),
					OutboundTag: types.StringValue("api"),
					InboundTag:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("api")}),
				},
			},
			wantErr: true,
		},
		{
			name: "API outbound with null inbound_tag",
			rules: []XrayRoutingRule{
				{
					Type:        types.StringValue("field"),
					OutboundTag: types.StringValue("api"),
					InboundTag:  types.ListNull(types.StringType),
				},
			},
			wantErr: false,
		},
		{
			name: "API outbound with api mixed in inbound_tag",
			rules: []XrayRoutingRule{
				{
					Type:        types.StringValue("field"),
					OutboundTag: types.StringValue("api"),
					InboundTag:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("http"), types.StringValue("api")}),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := validateNoAPIRoutingRules(tt.rules)
			if (msg != "") != tt.wantErr {
				t.Errorf("validateNoAPIRoutingRules() = %q, wantErr %v", msg, tt.wantErr)
			}
		})
	}
}

func TestEnsureNoAPIRoutingRules(t *testing.T) {
	var diags diag.Diagnostics
	ensureNoAPIRoutingRules([]XrayRoutingRule{
		{
			OutboundTag: types.StringValue("api"),
			InboundTag:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("api")}),
		},
	}, &diags)
	if !diags.HasError() {
		t.Fatal("expected error diagnostic for API routing rule")
	}

	diags = nil
	ensureNoAPIRoutingRules([]XrayRoutingRule{
		{
			OutboundTag: types.StringValue("blocked"),
			InboundTag:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("http")}),
		},
	}, &diags)
	if diags.HasError() {
		t.Fatalf("expected no error, got %v", diags)
	}
}
