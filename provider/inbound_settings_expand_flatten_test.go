package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// Table-driven round-trip tests for inbound per-protocol settings expand/flatten.
// Each case sets fields on the typed model, calls expand*InboundSettings to
// get an untyped map, then calls flatten*InboundSettings to get the model back,
// and asserts the relevant fields survive the round-trip.
//
// When all model fields are null, expand produces an empty map and flatten
// returns nil (this is the design). Those cases are marked expectNilFlat.
// ---------------------------------------------------------------------------

func TestExpandFlatten_VlessSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		name          string
		model         *InboundVlessSettingsModel
		expectNilFlat bool
	}{
		{
			name: "full",
			model: &InboundVlessSettingsModel{
				Decryption:   types.StringValue("none"),
				Encryption:   types.StringValue("none"),
				SelectedAuth: types.StringValue("none"),
				Fallback: []InboundFallbackModel{
					{Name: types.StringValue("fb1"), Alpn: types.StringValue("h2"), Path: types.StringValue("/ws"), Dest: types.StringValue("127.0.0.1:8080"), Xver: types.Int64Value(1)},
				},
			},
		},
		{
			name: "nil_fields",
			model: &InboundVlessSettingsModel{
				Decryption: types.StringNull(),
				Encryption: types.StringNull(),
			},
			expectNilFlat: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandVlessInboundSettings(tc.model)
			if expanded == nil {
				t.Fatal("expected non-nil expanded map")
			}
			flat := flattenVlessInboundSettings(expanded)
			if tc.expectNilFlat {
				if flat != nil {
					t.Fatalf("expected nil flat for empty expand, got %v", flat)
				}
				return
			}
			if flat == nil {
				t.Fatal("expected non-nil flattened model")
			}
			if flat.Decryption.ValueString() != tc.model.Decryption.ValueString() {
				t.Fatalf("decryption mismatch: got %q want %q", flat.Decryption.ValueString(), tc.model.Decryption.ValueString())
			}
			if len(tc.model.Fallback) > 0 {
				reExpanded := expandVlessInboundSettings(flat)
				fbs, ok := reExpanded["fallbacks"].([]any)
				if !ok {
					t.Fatal("expected fallbacks slice in re-expanded")
				}
				if len(fbs) != len(tc.model.Fallback) {
					t.Fatalf("fallback count mismatch: got %d want %d", len(fbs), len(tc.model.Fallback))
				}
			}
		})
	}
}

func TestExpandFlatten_VlessSettings_NilModel(t *testing.T) {
	if got := expandVlessInboundSettings(nil); got != nil {
		t.Fatalf("expected nil for nil model, got %v", got)
	}
}

func TestExpandFlatten_VlessSettings_EmptyData(t *testing.T) {
	if got := flattenVlessInboundSettings(map[string]any{}); got != nil {
		t.Fatalf("expected nil for empty data, got %v", got)
	}
}

func TestExpandFlatten_TrojanSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		name          string
		model         *InboundTrojanSettingsModel
		expectNilFlat bool
	}{
		{
			name: "with_fallbacks",
			model: &InboundTrojanSettingsModel{
				Fallback: []InboundFallbackModel{
					{Dest: types.StringValue("127.0.0.1:443"), Xver: types.Int64Value(0)},
					{Name: types.StringValue("fb2"), Dest: types.StringValue("127.0.0.1:80")},
				},
			},
		},
		{
			name:          "empty",
			model:         &InboundTrojanSettingsModel{},
			expectNilFlat: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandTrojanInboundSettings(tc.model)
			if expanded == nil {
				t.Fatal("expected non-nil expanded map")
			}
			flat := flattenTrojanInboundSettings(expanded)
			if tc.expectNilFlat {
				if flat != nil {
					t.Fatalf("expected nil flat for empty expand, got %v", flat)
				}
				return
			}
			if flat == nil {
				t.Fatal("expected non-nil flattened model")
			}
			reExpanded := expandTrojanInboundSettings(flat)
			if len(tc.model.Fallback) > 0 {
				fbs := reExpanded["fallbacks"].([]any)
				if len(fbs) != len(tc.model.Fallback) {
					t.Fatalf("fallback count mismatch: got %d want %d", len(fbs), len(tc.model.Fallback))
				}
			}
		})
	}
}

func TestExpandFlatten_TrojanSettings_NilModel(t *testing.T) {
	if got := expandTrojanInboundSettings(nil); got != nil {
		t.Fatalf("expected nil for nil model, got %v", got)
	}
}

func TestExpandFlatten_ShadowsocksSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		name          string
		model         *InboundShadowsocksSettingsModel
		expectNilFlat bool
	}{
		{
			name: "full",
			model: &InboundShadowsocksSettingsModel{
				Method:   types.StringValue("chacha20-ietf-poly1305"),
				Password: types.StringValue("s3cret"),
				Network:  types.StringValue("tcp,udp"),
				IVCheck:  types.BoolValue(true),
			},
		},
		{
			name: "minimal",
			model: &InboundShadowsocksSettingsModel{
				Method: types.StringValue("aes-256-gcm"),
			},
		},
		{
			name:          "all_null",
			model:         &InboundShadowsocksSettingsModel{},
			expectNilFlat: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandShadowsocksInboundSettings(tc.model)
			if expanded == nil {
				t.Fatal("expected non-nil expanded map")
			}
			flat := flattenShadowsocksInboundSettings(expanded)
			if tc.expectNilFlat {
				if flat != nil {
					t.Fatalf("expected nil flat for empty expand, got %v", flat)
				}
				return
			}
			if flat == nil {
				t.Fatal("expected non-nil flattened model")
			}
			if flat.Method.ValueString() != tc.model.Method.ValueString() {
				t.Fatalf("method mismatch: got %q want %q", flat.Method.ValueString(), tc.model.Method.ValueString())
			}
			if flat.Password.ValueString() != tc.model.Password.ValueString() {
				t.Fatalf("password mismatch: got %q want %q", flat.Password.ValueString(), tc.model.Password.ValueString())
			}
			if flat.Network.ValueString() != tc.model.Network.ValueString() {
				t.Fatalf("network mismatch: got %q want %q", flat.Network.ValueString(), tc.model.Network.ValueString())
			}
			if flat.IVCheck.ValueBool() != tc.model.IVCheck.ValueBool() {
				t.Fatalf("iv_check mismatch: got %v want %v", flat.IVCheck.ValueBool(), tc.model.IVCheck.ValueBool())
			}
		})
	}
}

func TestExpandFlatten_ShadowsocksSettings_NilModel(t *testing.T) {
	if got := expandShadowsocksInboundSettings(nil); got != nil {
		t.Fatalf("expected nil for nil model, got %v", got)
	}
}

func TestExpandFlatten_ShadowsocksSettings_EmptyData(t *testing.T) {
	if got := flattenShadowsocksInboundSettings(map[string]any{}); got != nil {
		t.Fatalf("expected nil for empty data, got %v", got)
	}
}

func TestExpandFlatten_HTTPSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		name          string
		model         *InboundHTTPSettingsModel
		expectNilFlat bool
	}{
		{
			name: "full",
			model: &InboundHTTPSettingsModel{
				Auth:             types.StringValue("password"),
				AllowTransparent: types.BoolValue(true),
				Account: []InboundAccountModel{
					{User: types.StringValue("u1"), Pass: types.StringValue("p1")},
					{User: types.StringValue("u2"), Pass: types.StringValue("p2")},
				},
			},
		},
		{
			name: "no_accounts",
			model: &InboundHTTPSettingsModel{
				Auth:             types.StringValue("noauth"),
				AllowTransparent: types.BoolValue(false),
			},
		},
		{
			name:          "all_null",
			model:         &InboundHTTPSettingsModel{},
			expectNilFlat: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandHTTPInboundSettings(tc.model)
			if expanded == nil {
				t.Fatal("expected non-nil expanded map")
			}
			flat := flattenHTTPInboundSettings(expanded)
			if tc.expectNilFlat {
				if flat != nil {
					t.Fatalf("expected nil flat for empty expand, got %v", flat)
				}
				return
			}
			if flat == nil {
				t.Fatal("expected non-nil flattened model")
			}
			if flat.Auth.ValueString() != tc.model.Auth.ValueString() {
				t.Fatalf("auth mismatch: got %q want %q", flat.Auth.ValueString(), tc.model.Auth.ValueString())
			}
			if flat.AllowTransparent.ValueBool() != tc.model.AllowTransparent.ValueBool() {
				t.Fatalf("allow_transparent mismatch: got %v want %v", flat.AllowTransparent.ValueBool(), tc.model.AllowTransparent.ValueBool())
			}
			if len(tc.model.Account) > 0 {
				if len(flat.Account) != len(tc.model.Account) {
					t.Fatalf("account count mismatch: got %d want %d", len(flat.Account), len(tc.model.Account))
				}
				for i, acc := range tc.model.Account {
					if flat.Account[i].User.ValueString() != acc.User.ValueString() {
						t.Fatalf("account[%d] user mismatch: got %q want %q", i, flat.Account[i].User.ValueString(), acc.User.ValueString())
					}
				}
			}
		})
	}
}

func TestExpandFlatten_HTTPSettings_NilModel(t *testing.T) {
	if got := expandHTTPInboundSettings(nil); got != nil {
		t.Fatalf("expected nil for nil model, got %v", got)
	}
}

func TestExpandFlatten_SocksSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		name          string
		model         *InboundSocksSettingsModel
		expectNilFlat bool
	}{
		{
			name: "full",
			model: &InboundSocksSettingsModel{
				Auth: types.StringValue("password"),
				UDP:  types.BoolValue(true),
				IP:   types.StringValue("127.0.0.1"),
				Account: []InboundAccountModel{
					{User: types.StringValue("admin"), Pass: types.StringValue("secret")},
				},
			},
		},
		{
			name: "minimal",
			model: &InboundSocksSettingsModel{
				UDP: types.BoolValue(false),
			},
		},
		{
			name:          "all_null",
			model:         &InboundSocksSettingsModel{},
			expectNilFlat: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandSocksInboundSettings(tc.model)
			if expanded == nil {
				t.Fatal("expected non-nil expanded map")
			}
			flat := flattenSocksInboundSettings(expanded)
			if tc.expectNilFlat {
				if flat != nil {
					t.Fatalf("expected nil flat for empty expand, got %v", flat)
				}
				return
			}
			if flat == nil {
				t.Fatal("expected non-nil flattened model")
			}
			if flat.Auth.ValueString() != tc.model.Auth.ValueString() {
				t.Fatalf("auth mismatch: got %q want %q", flat.Auth.ValueString(), tc.model.Auth.ValueString())
			}
			if flat.UDP.ValueBool() != tc.model.UDP.ValueBool() {
				t.Fatalf("udp mismatch: got %v want %v", flat.UDP.ValueBool(), tc.model.UDP.ValueBool())
			}
			if flat.IP.ValueString() != tc.model.IP.ValueString() {
				t.Fatalf("ip mismatch: got %q want %q", flat.IP.ValueString(), tc.model.IP.ValueString())
			}
		})
	}
}

func TestExpandFlatten_SocksSettings_NilModel(t *testing.T) {
	if got := expandSocksInboundSettings(nil); got != nil {
		t.Fatalf("expected nil for nil model, got %v", got)
	}
}

func TestExpandFlatten_HysteriaInboundSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		name          string
		model         *InboundHysteriaSettingsModel
		expectNilFlat bool
	}{
		{
			name: "version2",
			model: &InboundHysteriaSettingsModel{
				Version: types.Int64Value(2),
			},
		},
		{
			name: "version1",
			model: &InboundHysteriaSettingsModel{
				Version: types.Int64Value(1),
			},
		},
		{
			name:          "null_version",
			model:         &InboundHysteriaSettingsModel{},
			expectNilFlat: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandHysteriaInboundSettings(tc.model)
			if expanded == nil {
				t.Fatal("expected non-nil expanded map")
			}
			flat := flattenHysteriaInboundSettings(expanded)
			if tc.expectNilFlat {
				if flat != nil {
					t.Fatalf("expected nil flat for empty expand, got %v", flat)
				}
				return
			}
			if flat == nil {
				t.Fatal("expected non-nil flattened model")
			}
			if flat.Version.ValueInt64() != tc.model.Version.ValueInt64() {
				t.Fatalf("version mismatch: got %d want %d", flat.Version.ValueInt64(), tc.model.Version.ValueInt64())
			}
		})
	}
}

func TestExpandFlatten_HysteriaInboundSettings_NilModel(t *testing.T) {
	if got := expandHysteriaInboundSettings(nil); got != nil {
		t.Fatalf("expected nil for nil model, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// Fallback sub-model expand/flatten
// ---------------------------------------------------------------------------

func TestExpandFlatten_FallbacksFromModel_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		list []InboundFallbackModel
	}{
		{name: "empty", list: nil},
		{
			name: "single",
			list: []InboundFallbackModel{
				{Name: types.StringValue("fb"), Alpn: types.StringValue("h2"), Path: types.StringValue("/ws"), Dest: types.StringValue("127.0.0.1:8080"), Xver: types.Int64Value(1)},
			},
		},
		{
			name: "multiple_partial",
			list: []InboundFallbackModel{
				{Dest: types.StringValue("a:80"), Xver: types.Int64Value(0)},
				{Name: types.StringValue("x"), Dest: types.StringValue("b:443"), Alpn: types.StringValue("h3")},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandFallbacksFromModel(tc.list)
			flat := flattenFallbacksToModel(expanded)
			if len(flat) != len(tc.list) {
				t.Fatalf("count mismatch: got %d want %d", len(flat), len(tc.list))
			}
			for i, fb := range tc.list {
				if fb.Dest.ValueString() != "" {
					if flat[i].Dest.ValueString() != fb.Dest.ValueString() {
						t.Fatalf("dest[%d] mismatch: got %q want %q", i, flat[i].Dest.ValueString(), fb.Dest.ValueString())
					}
				}
			}
		})
	}
}

func TestExpandFlatten_AccountsFromModel_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		list []InboundAccountModel
	}{
		{name: "empty", list: nil},
		{
			name: "single",
			list: []InboundAccountModel{
				{User: types.StringValue("u"), Pass: types.StringValue("p")},
			},
		},
		{
			name: "multiple",
			list: []InboundAccountModel{
				{User: types.StringValue("a"), Pass: types.StringValue("b")},
				{User: types.StringValue("c"), Pass: types.StringValue("d")},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandAccountsFromModel(tc.list)
			flat := flattenAccountsToModel(expanded)
			if len(flat) != len(tc.list) {
				t.Fatalf("count mismatch: got %d want %d", len(flat), len(tc.list))
			}
			for i, acc := range tc.list {
				if flat[i].User.ValueString() != acc.User.ValueString() {
					t.Fatalf("user[%d] mismatch: got %q want %q", i, flat[i].User.ValueString(), acc.User.ValueString())
				}
				if flat[i].Pass.ValueString() != acc.Pass.ValueString() {
					t.Fatalf("pass[%d] mismatch: got %q want %q", i, flat[i].Pass.ValueString(), acc.Pass.ValueString())
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Dokodemo / tunnel settings expand/flatten
// ---------------------------------------------------------------------------

func TestExpandFlatten_DokodemoInboundSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		name     string
		protocol string
		model    *InboundDokodemoSettingsModel
	}{
		{
			name:     "dokodemo_full",
			protocol: "dokodemo-door",
			model: &InboundDokodemoSettingsModel{
				Address:        types.StringValue("127.0.0.1"),
				Port:           types.Int64Value(8080),
				Network:        types.StringValue("tcp,udp"),
				FollowRedirect: types.BoolValue(true),
			},
		},
		{
			name:     "dokodemo_minimal",
			protocol: "dokodemo-door",
			model: &InboundDokodemoSettingsModel{
				Address: types.StringValue("10.0.0.1"),
				Port:    types.Int64Value(53),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandDokodemoInboundSettings(tc.protocol, tc.model)
			if expanded == nil {
				t.Fatal("expected non-nil expanded map")
			}
			flat := flattenDokodemoInboundSettings(tc.protocol, expanded)
			if flat == nil {
				t.Fatal("expected non-nil flattened model")
			}
			if flat.Address.ValueString() != tc.model.Address.ValueString() {
				t.Fatalf("address mismatch: got %q want %q", flat.Address.ValueString(), tc.model.Address.ValueString())
			}
			if flat.Port.ValueInt64() != tc.model.Port.ValueInt64() {
				t.Fatalf("port mismatch: got %d want %d", flat.Port.ValueInt64(), tc.model.Port.ValueInt64())
			}
		})
	}
}

func TestExpandDokodemoInboundSettings_NilModel(t *testing.T) {
	if got := expandDokodemoInboundSettings("dokodemo-door", nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestExpandFlatten_DokodemoInboundSettings_WithPortMap(t *testing.T) {
	portMap, _ := types.MapValue(types.StringType, map[string]attr.Value{
		"80":  types.StringValue("127.0.0.1:8080"),
		"443": types.StringValue("127.0.0.1:8443"),
	})
	model := &InboundDokodemoSettingsModel{
		Address:        types.StringValue("127.0.0.1"),
		Port:           types.Int64Value(80),
		Network:        types.StringValue("tcp"),
		PortMap:        portMap,
		FollowRedirect: types.BoolValue(false),
	}
	expanded := expandDokodemoInboundSettings("dokodemo-door", model)
	if expanded == nil {
		t.Fatal("expected non-nil expanded map")
	}
	pm, ok := expanded["port_map"].(map[string]any)
	if !ok {
		t.Fatalf("expected port_map in expanded, got %T", expanded["port_map"])
	}
	if pm["80"] != "127.0.0.1:8080" {
		t.Fatalf("unexpected port_map[80]: %v", pm["80"])
	}
	flat := flattenDokodemoInboundSettings("dokodemo-door", expanded)
	if flat == nil {
		t.Fatal("expected non-nil flattened model")
	}
}

// ---------------------------------------------------------------------------
// tun-only settings (xray-core 26.9.30 / 3x-ui v3.9.0+)
// ---------------------------------------------------------------------------

func TestExpandFlatten_TunAutoSystemSettings(t *testing.T) {
	wfp := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("dns"), types.StringValue("misconfigtun"),
	})
	model := &InboundDokodemoSettingsModel{
		Address:                types.StringValue("10.0.0.1"),
		Port:                   types.Int64Value(53),
		AutoSystemDnsToGateway: types.BoolValue(true),
		AutoSystemWfpBlockLeak: wfp,
	}

	expanded := expandDokodemoInboundSettings("tun", model)
	if expanded["autoSystemDnsToGateway"] != true {
		t.Errorf("autoSystemDnsToGateway: got %v", expanded["autoSystemDnsToGateway"])
	}
	leak, ok := expanded["autoSystemWfpBlockLeak"].([]any)
	if !ok || len(leak) != 2 || leak[0] != "dns" || leak[1] != "misconfigtun" {
		t.Errorf("autoSystemWfpBlockLeak: got %#v", expanded["autoSystemWfpBlockLeak"])
	}

	flat := flattenDokodemoInboundSettings("tun", expanded)
	if flat == nil {
		t.Fatal("expected non-nil flattened model")
	}
	if !flat.AutoSystemDnsToGateway.ValueBool() {
		t.Errorf("flatten autoSystemDnsToGateway: %v", flat.AutoSystemDnsToGateway)
	}
	var got []string
	flat.AutoSystemWfpBlockLeak.ElementsAs(t.Context(), &got, false)
	if len(got) != 2 || got[0] != "dns" || got[1] != "misconfigtun" {
		t.Errorf("flatten autoSystemWfpBlockLeak: %v", got)
	}
}

// The tun-only keys must never leak onto dokodemo-door/tunnel settings, and
// must flatten to null there.
func TestExpandFlatten_TunAutoSystemSettings_ProtocolGated(t *testing.T) {
	model := &InboundDokodemoSettingsModel{
		Address:                types.StringValue("10.0.0.1"),
		Port:                   types.Int64Value(53),
		AutoSystemDnsToGateway: types.BoolValue(true),
		AutoSystemWfpBlockLeak: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("dns"),
		}),
	}

	for _, protocol := range []string{"dokodemo-door", "tunnel"} {
		expanded := expandDokodemoInboundSettings(protocol, model)
		if _, ok := expanded["autoSystemDnsToGateway"]; ok {
			t.Errorf("%s must not carry autoSystemDnsToGateway", protocol)
		}
		if _, ok := expanded["autoSystemWfpBlockLeak"]; ok {
			t.Errorf("%s must not carry autoSystemWfpBlockLeak", protocol)
		}
		flat := flattenDokodemoInboundSettings(protocol, expanded)
		if !flat.AutoSystemDnsToGateway.IsNull() {
			t.Errorf("%s flatten autoSystemDnsToGateway must be null, got %v", protocol, flat.AutoSystemDnsToGateway)
		}
		if !flat.AutoSystemWfpBlockLeak.IsNull() {
			t.Errorf("%s flatten autoSystemWfpBlockLeak must be null, got %v", protocol, flat.AutoSystemWfpBlockLeak)
		}
	}
}

// A tun settings blob without the optional keys flattens them to null, and an
// unset model does not emit the keys.
func TestExpandFlatten_TunAutoSystemSettings_Absent(t *testing.T) {
	model := &InboundDokodemoSettingsModel{
		Address: types.StringValue("10.0.0.1"),
		Port:    types.Int64Value(53),
	}
	expanded := expandDokodemoInboundSettings("tun", model)
	if _, ok := expanded["autoSystemDnsToGateway"]; ok {
		t.Error("unset autoSystemDnsToGateway must be omitted")
	}
	if _, ok := expanded["autoSystemWfpBlockLeak"]; ok {
		t.Error("unset autoSystemWfpBlockLeak must be omitted")
	}
	flat := flattenDokodemoInboundSettings("tun", expanded)
	if !flat.AutoSystemDnsToGateway.IsNull() {
		t.Errorf("absent autoSystemDnsToGateway must flatten to null, got %v", flat.AutoSystemDnsToGateway)
	}
	if !flat.AutoSystemWfpBlockLeak.IsNull() {
		t.Errorf("absent autoSystemWfpBlockLeak must flatten to null, got %v", flat.AutoSystemWfpBlockLeak)
	}
}
