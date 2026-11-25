package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestXrayObservatoryRoundTrip exercises all four conversion layers for both
// observatory and burst_observatory:
// model → untyped (expandXrayObservatory) → wire (buildXrayObservatoryJSON) →
// untyped (flattenXrayObservatoryToMap) → model (flattenXrayObservatory).
func TestXrayObservatoryRoundTrip(t *testing.T) {
	model := &XrayObservatoryModel{
		Observatory: []XrayObservatoryEntry{
			{
				Tag: types.StringValue("obs1"),
				SubjectSelector: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("out-1"),
					types.StringValue("out-2"),
				}),
				ProbeURL:          types.StringValue("https://www.google.com/generate_204"),
				ProbeInterval:     types.StringValue("1m"),
				EnableConcurrency: types.BoolValue(true),
			},
		},
		BurstObservatory: []XrayBurstObservatory{
			{
				Tag: types.StringValue("burst1"),
				SubjectSelector: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("out-3"),
				}),
				PingConfig: []XrayBurstPingConfig{
					{
						Destination:    types.StringValue("https://www.google.com/generate_204"),
						Interval:       types.StringValue("1m"),
						ConnectTimeout: types.StringValue("5s"),
						Timeout:        types.StringValue("5s"),
						Samples:        types.Int64Value(3),
						SamplingCount:  types.Int64Value(2),
						Lazy:           types.BoolValue(true),
					},
				},
			},
		},
	}

	// 1. model → untyped
	expanded := expandXrayObservatory(model)
	obsList, ok := expanded["observatory"].([]any)
	if !ok || len(obsList) != 1 {
		t.Fatalf("model→untyped: expected 1 observatory, got %v", expanded["observatory"])
	}
	obsEntry, _ := obsList[0].(map[string]any)
	if obsEntry["probeURL"] != "https://www.google.com/generate_204" {
		t.Fatalf("model→untyped: expected probeURL, got %v", obsEntry["probeURL"])
	}

	burstList, ok := expanded["burst_observatory"].([]any)
	if !ok || len(burstList) != 1 {
		t.Fatalf("model→untyped: expected 1 burst_observatory, got %v", expanded["burst_observatory"])
	}

	// 2. untyped → wire (keyed by tag)
	wire := buildXrayObservatoryJSON(expanded).(map[string]any)
	wireObs, ok := wire["observatory"].(map[string]any)
	if !ok {
		t.Fatalf("untyped→wire: expected observatory object, got %T", wire["observatory"])
	}
	wireObsEntry, ok := wireObs["obs1"].(map[string]any)
	if !ok {
		t.Fatalf("untyped→wire: expected key 'obs1' in observatory, got %v", wireObs)
	}
	if wireObsEntry["probeURL"] != "https://www.google.com/generate_204" {
		t.Fatalf("untyped→wire: expected probeURL, got %v", wireObsEntry["probeURL"])
	}
	if wireObsEntry["enableConcurrency"] != true {
		t.Fatalf("untyped→wire: expected enableConcurrency=true, got %v", wireObsEntry["enableConcurrency"])
	}

	wireBurst, ok := wire["burstObservatory"].(map[string]any)
	if !ok {
		t.Fatalf("untyped→wire: expected burstObservatory object, got %T", wire["burstObservatory"])
	}
	wireBurstEntry, ok := wireBurst["burst1"].(map[string]any)
	if !ok {
		t.Fatalf("untyped→wire: expected key 'burst1', got %v", wireBurst)
	}
	wirePC, ok := wireBurstEntry["pingConfig"].(map[string]any)
	if !ok {
		t.Fatalf("untyped→wire: expected pingConfig map, got %T", wireBurstEntry["pingConfig"])
	}
	if wirePC["destination"] != "https://www.google.com/generate_204" {
		t.Fatalf("untyped→wire: expected destination, got %v", wirePC["destination"])
	}

	// 3. wire → untyped (flat lists)
	flat := flattenXrayObservatoryToMap(wire)
	flatObs, ok := flat["observatory"].([]any)
	if !ok || len(flatObs) != 1 {
		t.Fatalf("wire→untyped: expected 1 observatory entry, got %v", flat["observatory"])
	}
	flatObsEntry, _ := flatObs[0].(map[string]any)
	if flatObsEntry["tag"] != "obs1" {
		t.Fatalf("wire→untyped: expected tag=obs1, got %v", flatObsEntry["tag"])
	}

	// 4. untyped → model
	model2 := flattenXrayObservatory(flat)
	if len(model2.Observatory) != 1 {
		t.Fatalf("untyped→model: expected 1 observatory, got %d", len(model2.Observatory))
	}
	obs2 := model2.Observatory[0]
	if obs2.Tag.ValueString() != "obs1" {
		t.Fatalf("untyped→model: expected tag=obs1, got %q", obs2.Tag)
	}
	if obs2.ProbeURL.ValueString() != "https://www.google.com/generate_204" {
		t.Fatalf("untyped→model: ProbeURL round-trip failed: %q", obs2.ProbeURL)
	}
	if obs2.ProbeInterval.ValueString() != "1m" {
		t.Fatalf("untyped→model: ProbeInterval round-trip failed: %q", obs2.ProbeInterval)
	}
	if !obs2.EnableConcurrency.ValueBool() {
		t.Fatalf("untyped→model: EnableConcurrency round-trip failed")
	}

	if len(model2.BurstObservatory) != 1 {
		t.Fatalf("untyped→model: expected 1 burst_observatory, got %d", len(model2.BurstObservatory))
	}
	burst2 := model2.BurstObservatory[0]
	if burst2.Tag.ValueString() != "burst1" {
		t.Fatalf("untyped→model: expected burst tag=burst1, got %q", burst2.Tag)
	}
	if len(burst2.PingConfig) != 1 {
		t.Fatalf("untyped→model: expected 1 ping_config, got %d", len(burst2.PingConfig))
	}
	pc2 := burst2.PingConfig[0]
	if pc2.Destination.ValueString() != "https://www.google.com/generate_204" {
		t.Fatalf("untyped→model: Destination round-trip failed: %q", pc2.Destination)
	}
	if pc2.Samples.ValueInt64() != 3 {
		t.Fatalf("untyped→model: Samples round-trip failed: %d", pc2.Samples.ValueInt64())
	}
	if !pc2.Lazy.ValueBool() {
		t.Fatalf("untyped→model: Lazy round-trip failed")
	}
}

// TestXrayObservatoryEmptyModel confirms an empty model produces an empty
// payload on all layers.
func TestXrayObservatoryEmptyModel(t *testing.T) {
	model := &XrayObservatoryModel{}
	expanded := expandXrayObservatory(model)
	if len(expanded) != 0 {
		t.Fatalf("empty model should produce empty map, got %v", expanded)
	}

	wire := buildXrayObservatoryJSON(expanded).(map[string]any)
	if len(wire) != 0 {
		t.Fatalf("empty expanded should produce empty wire, got %v", wire)
	}

	flat := flattenXrayObservatoryToMap(wire)
	model2 := flattenXrayObservatory(flat)
	if len(model2.Observatory) != 0 || len(model2.BurstObservatory) != 0 {
		t.Fatalf("empty round-trip should produce empty model, got %+v", model2)
	}
}

// TestXrayObservatoryMultipleEntries confirms multiple entries per type
// round-trip correctly with distinct tags.
func TestXrayObservatoryMultipleEntries(t *testing.T) {
	model := &XrayObservatoryModel{
		Observatory: []XrayObservatoryEntry{
			{Tag: types.StringValue("obs-a"), ProbeURL: types.StringValue("https://a.example.com")},
			{Tag: types.StringValue("obs-b"), ProbeURL: types.StringValue("https://b.example.com")},
		},
	}

	expanded := expandXrayObservatory(model)
	wire := buildXrayObservatoryJSON(expanded).(map[string]any)
	wireObs := wire["observatory"].(map[string]any)
	if len(wireObs) != 2 {
		t.Fatalf("expected 2 wire observatory entries, got %d", len(wireObs))
	}
	if _, ok := wireObs["obs-a"]; !ok {
		t.Fatal("expected key obs-a in wire observatory")
	}
	if _, ok := wireObs["obs-b"]; !ok {
		t.Fatal("expected key obs-b in wire observatory")
	}

	// Round-trip back
	flat := flattenXrayObservatoryToMap(wire)
	model2 := flattenXrayObservatory(flat)
	if len(model2.Observatory) != 2 {
		t.Fatalf("expected 2 model observatory entries after round-trip, got %d", len(model2.Observatory))
	}
}

// TestXrayObservatorySchema exercises the schema definition so the lines
// count toward Codecov patch coverage.
func TestXrayObservatorySchema(t *testing.T) {
	s := xrayObservatorySchema()
	if s.Attributes["id"] == nil {
		t.Fatal("expected id attribute")
	}
	if s.Blocks["observatory"] == nil {
		t.Fatal("expected observatory block")
	}
	if s.Blocks["burst_observatory"] == nil {
		t.Fatal("expected burst_observatory block")
	}
}

// TestFlattenXrayObservatoryEdgeCases covers defensive branches in
// flattenXrayObservatory: missing keys, non-list items, non-map items.
func TestFlattenXrayObservatoryEdgeCases(t *testing.T) {
	cases := []struct {
		name string
		data map[string]any
	}{
		{"empty map", map[string]any{}},
		{"observatory not a map", map[string]any{"observatory": "oops"}},
		{"non-map item in observatory", map[string]any{"observatory": []any{"str"}}},
		{"burst_observatory not a map", map[string]any{"burst_observatory": "oops"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := flattenXrayObservatory(tc.data)
			if m == nil {
				t.Fatal("expected non-nil model")
			}
		})
	}
}

// TestFlattenObservatoryEntryListEdgeCases covers the nil and non-map cases.
func TestFlattenObservatoryEntryListEdgeCases(t *testing.T) {
	if flattenObservatoryEntryList(nil) != nil {
		t.Fatal("nil list should return nil")
	}
	// Non-map items are skipped, yielding an empty (but non-nil) slice
	result := flattenObservatoryEntryList([]any{"not-a-map"})
	if len(result) != 0 {
		t.Fatalf("non-map items should be skipped, got %v", result)
	}
	// Entry with tag but no other fields
	result = flattenObservatoryEntryList([]any{map[string]any{"tag": "x"}})
	if len(result) != 1 || result[0].Tag.ValueString() != "x" {
		t.Fatalf("expected 1 entry with tag=x, got %+v", result)
	}
}

// TestFlattenBurstObservatoryListEdgeCases covers defensive branches.
func TestFlattenBurstObservatoryListEdgeCases(t *testing.T) {
	if flattenBurstObservatoryList(nil) != nil {
		t.Fatal("nil list should return nil")
	}
	// Non-map items are skipped, yielding an empty (but non-nil) slice
	result := flattenBurstObservatoryList([]any{"not-a-map"})
	if len(result) != 0 {
		t.Fatalf("non-map items should be skipped, got %v", result)
	}
	// Entry with tag + pingConfig
	result = flattenBurstObservatoryList([]any{
		map[string]any{
			"tag": "burst-x",
			"pingConfig": map[string]any{
				"destination": "https://example.com",
				"interval":    "30s",
			},
		},
	})
	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	if result[0].Tag.ValueString() != "burst-x" {
		t.Fatalf("expected tag=burst-x, got %q", result[0].Tag)
	}
	if len(result[0].PingConfig) != 1 {
		t.Fatalf("expected 1 ping_config, got %d", len(result[0].PingConfig))
	}
}

// TestFlattenBurstPingConfigEdgeCases covers the numeric coercion and
// empty input branches.
func TestFlattenBurstPingConfigEdgeCases(t *testing.T) {
	// With float64 values (as they arrive from JSON)
	res := flattenBurstPingConfig(map[string]any{
		"destination":    "https://test.com",
		"interval":       "1m",
		"connectTimeout": "3s",
		"timeout":        "5s",
		"samples":        float64(10),
		"samplingCount":  float64(5),
		"lazy":           true,
	})
	if len(res) != 1 {
		t.Fatalf("expected 1 ping_config, got %d", len(res))
	}
	st := res[0]
	if st.Destination.ValueString() != "https://test.com" {
		t.Fatalf("destination round-trip failed: %q", st.Destination)
	}
	if st.Samples.ValueInt64() != 10 {
		t.Fatalf("samples round-trip failed: %d", st.Samples.ValueInt64())
	}
	if st.SamplingCount.ValueInt64() != 5 {
		t.Fatalf("samplingCount round-trip failed: %d", st.SamplingCount.ValueInt64())
	}
	if !st.Lazy.ValueBool() {
		t.Fatal("lazy round-trip failed")
	}

	// Empty input
	res2 := flattenBurstPingConfig(map[string]any{})
	if len(res2) != 1 {
		t.Fatalf("expected 1 ping_config from empty input, got %d", len(res2))
	}
}

// TestExpandBurstPingConfigEdgeCases covers the nil/empty branches.
func TestExpandBurstPingConfigEdgeCases(t *testing.T) {
	if expandBurstPingConfig(nil) != nil {
		t.Fatal("nil configs should return nil")
	}
	if expandBurstPingConfig([]XrayBurstPingConfig{}) != nil {
		t.Fatal("empty configs should return nil")
	}
	// Config with only null fields → empty map → nil
	nullConfig := []XrayBurstPingConfig{{}}
	if expandBurstPingConfig(nullConfig) != nil {
		t.Fatal("config with all-null fields should return nil")
	}
}

// TestBuildXrayObservatoryJSONEdgeCases covers missing/non-map keys.
func TestBuildXrayObservatoryJSONEdgeCases(t *testing.T) {
	// Empty map
	res := buildXrayObservatoryJSON(map[string]any{})
	obj, ok := res.(map[string]any)
	if !ok || len(obj) != 0 {
		t.Fatalf("empty input must yield empty map, got %#v", res)
	}

	// Non-list observatory
	res = buildXrayObservatoryJSON(map[string]any{"observatory": "oops"})
	obj, _ = res.(map[string]any)
	if len(obj) != 0 {
		t.Fatalf("non-list observatory must yield empty map, got %#v", res)
	}
}

// TestFlattenXrayObservatoryToMapJSONString tests JSON string deserialization.
func TestFlattenXrayObservatoryToMapJSONString(t *testing.T) {
	jsonStr := `{"observatory":{"test-obs":{"probeURL":"https://test.com"}}}`
	result := flattenXrayObservatoryToMap(jsonStr)
	obs, ok := result["observatory"].([]any)
	if !ok || len(obs) != 1 {
		t.Fatalf("expected 1 observatory from JSON string, got %v", result["observatory"])
	}
}

// TestFlattenXrayObservatoryToMapInvalidJSON tests error handling.
func TestFlattenXrayObservatoryToMapInvalidJSON(t *testing.T) {
	result := flattenXrayObservatoryToMap("not json")
	if len(result) != 0 {
		t.Fatalf("invalid JSON should yield empty map, got %v", result)
	}
}

// TestFlattenXrayObservatoryToMapUnsupportedType tests non-map/string input.
func TestFlattenXrayObservatoryToMapUnsupportedType(t *testing.T) {
	result := flattenXrayObservatoryToMap(12345)
	if len(result) != 0 {
		t.Fatalf("unsupported type should yield empty map, got %v", result)
	}
}

// TestFlattenObservatoryObjectEdgeCases covers wire→untyped edge cases.
func TestFlattenObservatoryObjectEdgeCases(t *testing.T) {
	// Non-map value for a tag
	obj := map[string]any{"tag1": "not-a-map"}
	result := flattenObservatoryObject(obj)
	if len(result) != 0 {
		t.Fatalf("non-map value should be skipped, got %v", result)
	}

	// Empty object
	result = flattenObservatoryObject(map[string]any{})
	if len(result) != 0 {
		t.Fatalf("empty object should yield empty list, got %v", result)
	}
}

// TestFlattenBurstObservatoryObjectEdgeCases covers burst wire→untyped.
func TestFlattenBurstObservatoryObjectEdgeCases(t *testing.T) {
	obj := map[string]any{"tag1": "not-a-map"}
	result := flattenBurstObservatoryObject(obj)
	if len(result) != 0 {
		t.Fatalf("non-map value should be skipped, got %v", result)
	}
}

// TestBuildObservatoryObjectEdgeCases covers non-map items and missing tags.
func TestBuildObservatoryObjectEdgeCases(t *testing.T) {
	// Non-map item
	result := buildObservatoryObject([]any{"not-a-map"})
	if len(result) != 0 {
		t.Fatalf("non-map item should be skipped, got %v", result)
	}

	// Map without tag
	result = buildObservatoryObject([]any{map[string]any{"probeURL": "https://test.com"}})
	if len(result) != 0 {
		t.Fatalf("entry without tag should be skipped, got %v", result)
	}

	// Empty tag
	result = buildObservatoryObject([]any{map[string]any{"tag": ""}})
	if len(result) != 0 {
		t.Fatalf("empty tag should be skipped, got %v", result)
	}
}

// TestBuildBurstObservatoryObjectEdgeCases covers non-map items and missing tags.
func TestBuildBurstObservatoryObjectEdgeCases(t *testing.T) {
	// Non-map item
	result := buildBurstObservatoryObject([]any{"not-a-map"})
	if len(result) != 0 {
		t.Fatalf("non-map item should be skipped, got %v", result)
	}

	// Map without tag
	result = buildBurstObservatoryObject([]any{map[string]any{"pingConfig": map[string]any{}}})
	if len(result) != 0 {
		t.Fatalf("entry without tag should be skipped, got %v", result)
	}
}

// TestFlattenStringListToType covers the string list → types.List helper.
func TestFlattenStringListToType(t *testing.T) {
	result := flattenStringListToType([]any{"a", "b", "c"})
	if result.IsNull() {
		t.Fatal("expected non-null list")
	}
	elems := result.Elements()
	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}
}

// TestExpandObservatoryEntryListEmpty confirms empty input returns nil.
func TestExpandObservatoryEntryListEmpty(t *testing.T) {
	if expandObservatoryEntryList(nil) != nil {
		t.Fatal("nil list should return nil")
	}
	if expandObservatoryEntryList([]XrayObservatoryEntry{}) != nil {
		t.Fatal("empty list should return nil")
	}
}

// TestExpandBurstObservatoryListEmpty confirms empty input returns nil.
func TestExpandBurstObservatoryListEmpty(t *testing.T) {
	if expandBurstObservatoryList(nil) != nil {
		t.Fatal("nil list should return nil")
	}
	if expandBurstObservatoryList([]XrayBurstObservatory{}) != nil {
		t.Fatal("empty list should return nil")
	}
}

// TestFlattenObservatorySubjectSelectorAbsentGuard is a regression test for the
// zero-value types.List{} bug. When the panel returns an observatory (or burst
// observatory) entry WITHOUT a subjectSelector field, the flattened
// SubjectSelector must still be a valid types.List (an empty String list), not
// an uninitialized zero value — otherwise terraform-plugin-framework raises a
// "Value Conversion Error" (Expected framework type: ListType[StringType]) when
// the resource writes state via State.Set.
func TestFlattenObservatorySubjectSelectorAbsentGuard(t *testing.T) {
	empty := types.ListValueMust(types.StringType, nil)

	// Observatory entry present, subjectSelector omitted.
	obs := flattenObservatoryEntryList([]any{map[string]any{"tag": "x"}})
	if len(obs) != 1 {
		t.Fatalf("expected 1 observatory entry, got %d", len(obs))
	}
	if obs[0].SubjectSelector.IsNull() {
		t.Fatal("observatory SubjectSelector must be a non-null empty list, not null")
	}
	if !obs[0].SubjectSelector.Equal(empty) {
		t.Fatalf("observatory SubjectSelector must be a valid empty String list when absent, got %#v", obs[0].SubjectSelector)
	}

	// Burst observatory entry present, subjectSelector omitted.
	burst := flattenBurstObservatoryList([]any{map[string]any{"tag": "y"}})
	if len(burst) != 1 {
		t.Fatalf("expected 1 burst observatory entry, got %d", len(burst))
	}
	if burst[0].SubjectSelector.IsNull() {
		t.Fatal("burst SubjectSelector must be a non-null empty list, not null")
	}
	if !burst[0].SubjectSelector.Equal(empty) {
		t.Fatalf("burst SubjectSelector must be a valid empty String list when absent, got %#v", burst[0].SubjectSelector)
	}
}

// TestApplyObservatoryDesiredShape guards the double-wrap fix: the value passed
// to xrayApplyTyped for a set-path section must be the section content (the
// tag-keyed object), not re-wrapped with the key. buildXrayObservatoryJSON
// already returns the content keyed under "observatory"/"burstObservatory", so
// the value extracted from it (obs/burst) is what gets stored at the path. This
// test pins the contract by feeding applyXraySection directly.
func TestApplyObservatoryDesiredShape(t *testing.T) {
	content := map[string]any{"obs_default": map[string]any{"subjectSelector": []any{"proxy-*"}}}

	root, err := applyXraySection(map[string]any{}, content, xraySectionObservatory)
	if err != nil {
		t.Fatalf("applyXraySection failed: %v", err)
	}
	// The content must land directly at root["observatory"], NOT double-nested.
	got, ok := root["observatory"].(map[string]any)
	if !ok {
		t.Fatalf("expected root[observatory] to be a map, got %T", root["observatory"])
	}
	entry, ok := got["obs_default"].(map[string]any)
	if !ok {
		t.Fatalf("expected root[observatory][obs_default] map, got %v", got["obs_default"])
	}
	if _, ok := entry["subjectSelector"]; !ok {
		t.Fatalf("expected subjectSelector preserved at root[observatory][obs_default], got %v", entry)
	}
}

// TestXrayObservatoryFlatRoundTrip verifies that the default observatory uses
// Xray's singleton wire format and keeps its Terraform identity on refresh.
func TestXrayObservatoryFlatRoundTrip(t *testing.T) {
	model := &XrayObservatoryModel{
		Observatory: []XrayObservatoryEntry{{
			Tag:               types.StringValue("default"),
			SubjectSelector:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("relay-")}),
			ProbeURL:          types.StringValue("https://example.org/generate_204"),
			ProbeInterval:     types.StringValue("1m"),
			EnableConcurrency: types.BoolValue(true),
		}},
	}
	wire := buildXrayObservatoryJSON(expandXrayObservatory(model)).(map[string]any)
	obs := wire["observatory"].(map[string]any)
	selector, ok := obs["subjectSelector"].([]string)
	if !ok || len(selector) != 1 || selector[0] != "relay-" {
		t.Fatalf("expected flat subjectSelector, got %#v", obs)
	}
	raw, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	result := flattenXrayObservatory(flattenXrayObservatoryToMap(string(raw)))
	if len(result.Observatory) != 1 {
		t.Fatalf("expected one observatory after refresh, got %d", len(result.Observatory))
	}
	got := result.Observatory[0]
	want := model.Observatory[0]
	if !got.Tag.Equal(want.Tag) || !got.SubjectSelector.Equal(want.SubjectSelector) ||
		!got.ProbeURL.Equal(want.ProbeURL) || !got.ProbeInterval.Equal(want.ProbeInterval) ||
		!got.EnableConcurrency.Equal(want.EnableConcurrency) {
		t.Fatalf("observatory changed on refresh: got %#v", got)
	}
}

// TestBurstObservatoryFlatRoundTrip verifies that a single default burst entry
// uses Xray's singleton wire format and keeps its Terraform identity on refresh.
func TestBurstObservatoryFlatRoundTrip(t *testing.T) {
	model := &XrayObservatoryModel{
		BurstObservatory: []XrayBurstObservatory{{
			Tag:             types.StringValue("default"),
			SubjectSelector: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("proxy-")}),
			PingConfig: []XrayBurstPingConfig{{
				Destination:    types.StringValue("https://example.org/generate_204"),
				Interval:       types.StringValue("1m"),
				ConnectTimeout: types.StringValue("5s"),
				Timeout:        types.StringValue("10s"),
				Samples:        types.Int64Value(3),
				SamplingCount:  types.Int64Value(2),
				Lazy:           types.BoolValue(true),
			}},
		}},
	}
	wire := buildXrayObservatoryJSON(expandXrayObservatory(model)).(map[string]any)
	burst := wire["burstObservatory"].(map[string]any)
	selector, ok := burst["subjectSelector"].([]string)
	if !ok || len(selector) != 1 || selector[0] != "proxy-" {
		t.Fatalf("expected flat subjectSelector, got %#v", burst)
	}
	if _, ok := burst["pingConfig"].(map[string]any); !ok {
		t.Fatalf("expected flat pingConfig object, got %#v", burst)
	}
	raw, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	result := flattenXrayObservatory(flattenXrayObservatoryToMap(string(raw)))
	if len(result.BurstObservatory) != 1 {
		t.Fatalf("expected one burst observatory after refresh, got %d", len(result.BurstObservatory))
	}
	got := result.BurstObservatory[0]
	want := model.BurstObservatory[0]
	if !got.Tag.Equal(want.Tag) || !got.SubjectSelector.Equal(want.SubjectSelector) {
		t.Fatalf("burst observatory changed on refresh: got %#v", got)
	}
	if len(got.PingConfig) != 1 {
		t.Fatalf("expected one ping config after refresh, got %d", len(got.PingConfig))
	}
	gp, wp := got.PingConfig[0], want.PingConfig[0]
	if !gp.Destination.Equal(wp.Destination) || !gp.Interval.Equal(wp.Interval) ||
		!gp.ConnectTimeout.Equal(wp.ConnectTimeout) || !gp.Timeout.Equal(wp.Timeout) ||
		!gp.Samples.Equal(wp.Samples) || !gp.SamplingCount.Equal(wp.SamplingCount) ||
		!gp.Lazy.Equal(wp.Lazy) {
		t.Fatalf("burst ping config changed on refresh: got %#v", gp)
	}
}

// TestBurstObservatoryFlatGuards pins the tag-keyed fallbacks: only a single
// entry with tag "default" may use the flat singleton form, and a tag literally
// named "pingConfig" must not be read back as the flat form.
func TestBurstObservatoryFlatGuards(t *testing.T) {
	single := buildBurstObservatoryObject([]any{
		map[string]any{"tag": "burst1", "subjectSelector": []any{"proxy-"}},
	})
	if len(single) != 1 {
		t.Fatalf("expected one tag-keyed entry, got %#v", single)
	}
	if _, ok := single["burst1"]; !ok {
		t.Fatalf("expected tag-keyed burst1 entry, got %#v", single)
	}

	two := buildBurstObservatoryObject([]any{
		map[string]any{"tag": "default", "subjectSelector": []any{"proxy-"}},
		map[string]any{"tag": "relay", "subjectSelector": []any{"direct-"}},
	})
	if len(two) != 2 {
		t.Fatalf("expected two tag-keyed entries, got %#v", two)
	}
	if _, ok := two["default"]; !ok {
		t.Fatalf("expected tag-keyed default entry, got %#v", two)
	}

	read := flattenBurstObservatoryObject(map[string]any{
		"pingConfig": map[string]any{"subjectSelector": []any{"proxy-"}},
	})
	if len(read) != 1 {
		t.Fatalf("expected one entry, got %#v", read)
	}
	if entry, _ := read[0].(map[string]any); entry["tag"] != "pingConfig" {
		t.Fatalf("expected tag pingConfig, got %#v", entry)
	}
}

// TestBurstObservatoryFlatPingOnlyRoundTrip covers the flat singleton without a
// subject selector: {"pingConfig": {...}} must not be read back as an entry
// named "pingConfig".
func TestBurstObservatoryFlatPingOnlyRoundTrip(t *testing.T) {
	model := &XrayObservatoryModel{
		BurstObservatory: []XrayBurstObservatory{{
			Tag:             types.StringValue("default"),
			SubjectSelector: types.ListNull(types.StringType),
			PingConfig: []XrayBurstPingConfig{{
				Destination: types.StringValue("https://example.org/generate_204"),
				Interval:    types.StringValue("1m"),
				Timeout:     types.StringValue("5s"),
			}},
		}},
	}
	wire := buildXrayObservatoryJSON(expandXrayObservatory(model)).(map[string]any)
	burst := wire["burstObservatory"].(map[string]any)
	if _, ok := burst["subjectSelector"]; ok {
		t.Fatalf("expected no subjectSelector in flat wire form, got %#v", burst)
	}
	raw, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	result := flattenXrayObservatory(flattenXrayObservatoryToMap(string(raw)))
	if len(result.BurstObservatory) != 1 {
		t.Fatalf("expected one burst entry, got %d", len(result.BurstObservatory))
	}
	got := result.BurstObservatory[0]
	if !got.Tag.Equal(types.StringValue("default")) {
		t.Fatalf("expected tag default after refresh, got %v", got.Tag)
	}
	if got.SubjectSelector.IsNull() || got.SubjectSelector.IsUnknown() {
		t.Fatalf("expected a valid empty subject selector list, got %#v", got.SubjectSelector)
	}
	if len(got.PingConfig) != 1 || !got.PingConfig[0].Destination.Equal(model.BurstObservatory[0].PingConfig[0].Destination) {
		t.Fatalf("ping config changed on refresh: got %#v", got.PingConfig)
	}
}
