package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPanelSettingsV311RoundTrip(t *testing.T) {
	t.Parallel()

	general := flattenPanelGeneral(map[string]any{"nodeDownThreshold": float64(4)})
	if general.NodeDownThreshold.ValueInt64() != 4 {
		t.Fatalf("nodeDownThreshold flattened as %d", general.NodeDownThreshold.ValueInt64())
	}
	if got := expandPanelGeneral(general)["nodeDownThreshold"]; got != 4 {
		t.Fatalf("nodeDownThreshold expanded as %#v", got)
	}
	if !flattenPanelGeneral(map[string]any{}).NodeDownThreshold.IsNull() {
		t.Fatal("an old panel without nodeDownThreshold must produce null")
	}

	subscription := flattenPanelSubscription(map[string]any{
		"subHostSortMode":   "hosts-first",
		"subClashRemoteDNS": true,
	})
	if subscription.SubHostSortMode.ValueString() != "hosts-first" || !subscription.SubClashRemoteDNS.ValueBool() {
		t.Fatalf("subscription v3.11.0 fields flattened incorrectly: %#v", subscription)
	}
	wire := expandPanelSubscription(subscription)
	if wire["subHostSortMode"] != "hosts-first" || wire["subClashRemoteDNS"] != true {
		t.Fatalf("subscription v3.11.0 fields expanded incorrectly: %#v", wire)
	}
	oldPanel := flattenPanelSubscription(map[string]any{})
	if !oldPanel.SubHostSortMode.IsNull() || !oldPanel.SubClashRemoteDNS.IsNull() {
		t.Fatal("old panels without the v3.11.0 subscription fields must produce null")
	}
}

func TestPanelSettingsNeedRestartV311(t *testing.T) {
	t.Parallel()

	if !panelSettingsNeedRestart(map[string]any{"subClashRemoteDNS": false}, map[string]any{"subClashRemoteDNS": true}) {
		t.Fatal("subClashRemoteDNS is frozen by initRouter and must restart the panel")
	}
	for key, change := range map[string][2]any{
		"subHostSortMode":   {"inbound-first", "hosts-first"},
		"nodeDownThreshold": {2, 4},
	} {
		if panelSettingsNeedRestart(map[string]any{key: change[0]}, map[string]any{key: change[1]}) {
			t.Errorf("%s is read at runtime and must not restart the panel", key)
		}
	}
}

func TestPanelSettingsV311SchemaShape(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"sub_host_sort_mode", "sub_clash_remote_dns"} {
		attr, ok := panelSubscriptionSchema().Attributes[name]
		if !ok || !attr.IsOptional() || !attr.IsComputed() {
			t.Errorf("%s must be Optional+Computed", name)
		}
	}
	attr, ok := panelGeneralSchema().Attributes["node_down_threshold"]
	if !ok || !attr.IsOptional() || !attr.IsComputed() {
		t.Fatal("node_down_threshold must be Optional+Computed")
	}
}

func TestPanelSettingsV311ExpandOmitsUnset(t *testing.T) {
	t.Parallel()

	generalWire := expandPanelGeneral(&PanelGeneralModel{NodeDownThreshold: types.Int64Null()})
	if _, ok := generalWire["nodeDownThreshold"]; ok {
		t.Fatal("unset nodeDownThreshold must not be sent")
	}
	subscriptionWire := expandPanelSubscription(&PanelSubscriptionModel{
		SubHostSortMode:   types.StringUnknown(),
		SubClashRemoteDNS: types.BoolUnknown(),
	})
	if _, ok := subscriptionWire["subHostSortMode"]; ok {
		t.Fatal("unknown subHostSortMode must not be sent")
	}
	if _, ok := subscriptionWire["subClashRemoteDNS"]; ok {
		t.Fatal("unknown subClashRemoteDNS must not be sent")
	}
}

func v311Int64List(values ...int64) types.List {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.Int64Value(value))
	}
	return types.ListValueMust(types.Int64Type, elements)
}

func v311StringList(values ...string) types.List {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.ListValueMust(types.StringType, elements)
}

func TestHostGroupV311RoundTrip(t *testing.T) {
	t.Parallel()

	model := &HostGroupResourceModel{
		Snis:            v311StringList("a.example.com", "b.example.com"),
		SniSharedRandom: types.BoolValue(true),
		RandomCount:     types.Int64Value(42),
	}
	hg := hostGroupFromModel(context.Background(), model)
	if len(hg.Snis) != 2 || hg.Snis[1] != "b.example.com" || !hg.SniSharedRandom || hg.RandomCount != 42 {
		t.Fatalf("host-group v3.11.0 fields expanded incorrectly: %#v", hg)
	}
	flat := &HostGroupResourceModel{}
	flattenHostGroupToModel(hg, flat)
	if !flat.Snis.Equal(model.Snis) || !flat.SniSharedRandom.Equal(model.SniSharedRandom) || !flat.RandomCount.Equal(model.RandomCount) {
		t.Fatalf("host-group v3.11.0 fields did not round-trip: %#v", flat)
	}
	flattenHostGroupToModel(&HostGroup{}, flat)
	if !flat.RandomCount.IsNull() {
		t.Fatal("an old panel without randomCount must produce null")
	}

	hg.Sni = "one.example.com"
	normalizeHostGroupSniPrimary(context.Background(), &HostGroupResourceModel{
		Snis: v311StringList("three.example.com"),
	}, &HostGroupResourceModel{Sni: types.StringNull()}, hg)
	if hg.Sni != "three.example.com" {
		t.Fatalf("state-carried primary SNI was not aligned with the new pool: %q", hg.Sni)
	}
	normalizeHostGroupSniPrimary(context.Background(), &HostGroupResourceModel{
		Snis: v311StringList("ignored.example.com"),
	}, &HostGroupResourceModel{Sni: types.StringValue("explicit.example.com")}, hg)
	if hg.Sni != "three.example.com" {
		t.Fatal("an explicitly configured primary SNI must not be overwritten")
	}
}

func TestInboundClientHostRuleIDsRoundTrip(t *testing.T) {
	t.Parallel()

	model := &InboundClientResourceModel{ClientHostRuleIDs: v311Int64List(7, 11)}
	wire := expandInboundClientFromModel(model)
	ids, ok := wire["clientHostRuleIds"].([]any)
	if !ok || len(ids) != 2 || ids[0] != 7 || ids[1] != 11 {
		t.Fatalf("clientHostRuleIds expanded incorrectly: %#v", wire["clientHostRuleIds"])
	}
	flat := inboundClientToModel(1, "client", map[string]any{"clientHostRuleIds": []any{float64(7), float64(11)}})
	if !flat.ClientHostRuleIDs.Equal(model.ClientHostRuleIDs) {
		t.Fatalf("clientHostRuleIds flattened as %s", flat.ClientHostRuleIDs)
	}

	model.ClientHostRuleIDs = v311Int64List()
	wire = expandInboundClientFromModel(model)
	if ids, ok := wire["clientHostRuleIds"].([]any); !ok || len(ids) != 0 {
		t.Fatalf("an explicit empty list must clear bindings: %#v", wire["clientHostRuleIds"])
	}
	if got := flattenClientHostRuleIDs([]any{}); got.IsNull() || len(got.Elements()) != 0 {
		t.Fatalf("an empty response list must stay empty, got %s", got)
	}
	if got := flattenClientHostRuleIDs(nil); got.IsNull() || len(got.Elements()) != 0 {
		t.Fatalf("an absent response key must normalize to empty, got %s", got)
	}
	model.ClientHostRuleIDs = types.ListUnknown(types.Int64Type)
	if _, ok := expandInboundClientFromModel(model)["clientHostRuleIds"]; ok {
		t.Fatal("unknown clientHostRuleIds must not be sent")
	}
}

func TestClientHostRuleIDsValidator(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		ids       types.List
		wantError bool
	}{
		{ids: v311Int64List(1, 2)},
		{ids: v311Int64List()},
		{ids: v311Int64List(0), wantError: true},
		{ids: v311Int64List(-1), wantError: true},
		{ids: v311Int64List(1, 1), wantError: true},
	} {
		resp := &validator.ListResponse{}
		clientHostRuleIDsValidator{}.ValidateList(context.Background(), validator.ListRequest{ConfigValue: tc.ids}, resp)
		if resp.Diagnostics.HasError() != tc.wantError {
			t.Errorf("%s: error = %v, want %v", tc.ids, resp.Diagnostics.HasError(), tc.wantError)
		}
	}
}

func TestOwnedClientsHostRuleIDsRoundTrip(t *testing.T) {
	t.Parallel()

	ids := v311Int64List(3, 5)
	wgWire := expandWireguardClientsFromModel([]InboundWireguardClientModel{{ClientHostRuleIDs: ids}})
	if len(wgWire) != 1 || len(wgWire[0].(map[string]any)["clientHostRuleIds"].([]any)) != 2 {
		t.Fatalf("WireGuard host bindings expanded incorrectly: %#v", wgWire)
	}
	wgFlat := flattenInboundWireguardClientsToModel(wgWire)
	if len(wgFlat) != 1 || !wgFlat[0].ClientHostRuleIDs.Equal(ids) {
		t.Fatalf("WireGuard host bindings did not round-trip: %#v", wgFlat)
	}

	tuicWire := expandTuicClientsFromModel([]InboundTuicClientModel{{ClientHostRuleIDs: ids}})
	if len(tuicWire) != 1 || len(tuicWire[0].(map[string]any)["clientHostRuleIds"].([]any)) != 2 {
		t.Fatalf("TUIC host bindings expanded incorrectly: %#v", tuicWire)
	}
	tuicFlat := flattenTuicClientsToModel(tuicWire)
	if len(tuicFlat) != 1 || !tuicFlat[0].ClientHostRuleIDs.Equal(ids) {
		t.Fatalf("TUIC host bindings did not round-trip: %#v", tuicFlat)
	}
	if got := flattenTuicClientsToModel([]any{map[string]any{}}); len(got) != 1 || got[0].ClientHostRuleIDs.IsNull() || len(got[0].ClientHostRuleIDs.Elements()) != 0 {
		t.Fatalf("an absent TUIC binding list must normalize to empty: %#v", got)
	}

	awgWire := expandAmneziawgClientsFromModel([]InboundAmneziawgClientModel{{ClientHostRuleIDs: ids}})
	if len(awgWire) != 1 || len(awgWire[0].(map[string]any)["clientHostRuleIds"].([]any)) != 2 {
		t.Fatalf("AmneziaWG host bindings expanded incorrectly: %#v", awgWire)
	}
	awgFlat := flattenAmneziawgClientsToModel(awgWire)
	if len(awgFlat) != 1 || !awgFlat[0].ClientHostRuleIDs.Equal(ids) {
		t.Fatalf("AmneziaWG host bindings did not round-trip: %#v", awgFlat)
	}
}

func TestAmneziawgKernelModeRoundTrip(t *testing.T) {
	t.Parallel()

	model := &InboundAmneziawgServerModel{KernelMode: types.BoolValue(true)}
	wire := expandAmneziawgServerFromModel(model)
	if wire["kernelMode"] != true {
		t.Fatalf("kernelMode expanded as %#v", wire["kernelMode"])
	}
	if got := flattenAmneziawgServerToModel(wire); !got.KernelMode.ValueBool() {
		t.Fatal("kernelMode did not round-trip")
	}
}

func TestAmneziawgKeepAliveRangeRoundTrip(t *testing.T) {
	t.Parallel()

	model := InboundAmneziawgClientModel{KeepAliveRange: types.StringValue("15-25")}
	wire := expandAmneziawgClientsFromModel([]InboundAmneziawgClientModel{model})
	if got := wire[0].(map[string]any)["keepAlive"]; got != "15-25" {
		t.Fatalf("keepAlive range expanded as %#v", got)
	}
	flat := flattenAmneziawgClientsToModel(wire)
	if len(flat) != 1 || !flat[0].KeepAliveRange.Equal(model.KeepAliveRange) || !flat[0].KeepAlive.IsNull() {
		t.Fatalf("keepAlive range did not round-trip: %#v", flat)
	}

	wire = expandAmneziawgClientsFromModel([]InboundAmneziawgClientModel{{KeepAlive: types.Int64Value(25), KeepAliveRange: types.StringValue("15-25")}})
	if got := wire[0].(map[string]any)["keepAlive"]; got != "15-25" {
		t.Fatalf("a configured range must win over a state-carried scalar: %#v", got)
	}
	flat = flattenAmneziawgClientsToModel(wire)
	if len(flat) != 1 || flat[0].KeepAliveRange.ValueString() != "15-25" || !flat[0].KeepAlive.IsNull() {
		t.Fatalf("state-carried scalar did not yield to the range: %#v", flat)
	}

	wire = expandAmneziawgClientsFromModel([]InboundAmneziawgClientModel{{KeepAlive: types.Int64Value(25)}})
	flat = flattenAmneziawgClientsToModel(wire)
	if len(flat) != 1 || flat[0].KeepAlive.ValueInt64() != 25 || !flat[0].KeepAliveRange.IsNull() {
		t.Fatalf("numeric keepAlive did not round-trip: %#v", flat)
	}
	flat = flattenAmneziawgClientsToModel([]any{map[string]any{}})
	if len(flat) != 1 || flat[0].KeepAlive.ValueInt64() != 0 {
		t.Fatalf("an absent scalar must normalize to disabled keepalive: %#v", flat)
	}
}

func TestAmneziawgKeepAliveRangeValidator(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"0-0", "15-25", "0-65535", "65535-65535"} {
		resp := &validator.StringResponse{}
		awgKeepAliveRangeValidator{}.ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue(value)}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("%q must be accepted: %v", value, resp.Diagnostics)
		}
	}
	for _, value := range []string{"25-15", "-1-5", "0-65536"} {
		resp := &validator.StringResponse{}
		awgKeepAliveRangeValidator{}.ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue(value)}, resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("%q must be rejected", value)
		}
	}
}

func TestNodeUnavailableMetricsV311(t *testing.T) {
	t.Parallel()

	node := &Node{UnavailableMetrics: []string{"cpuPct", "swapPct"}}
	model := &NodeResourceModel{}
	flattenNodeToModel(node, model)
	if !model.UnavailableMetrics.Equal(v311StringList("cpuPct", "swapPct")) {
		t.Fatalf("unavailableMetrics flattened as %s", model.UnavailableMetrics)
	}
	payload, err := json.Marshal(node)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(payload, &wire); err != nil {
		t.Fatal(err)
	}
	if _, ok := wire["unavailableMetrics"]; !ok {
		t.Fatalf("nodes data-source payload lost unavailableMetrics: %s", payload)
	}
	if attr, ok := nodeResourceSchema().Attributes["unavailable_metrics"]; !ok || !attr.IsComputed() {
		t.Fatal("unavailable_metrics must be a computed resource attribute")
	}
}
