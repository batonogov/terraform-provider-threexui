package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Imported scripts must survive a full read/write cycle, including an explicit
// empty string: omission preserves state, so "" is the way to disable a script.
func TestXrayScriptsRoundTrip(t *testing.T) {
	for _, script := range []string{"/etc/xray/route.lua", ""} {
		t.Run(script, func(t *testing.T) {
			routing := flattenXrayRouting(flattenXrayRoutingToMap(map[string]any{"script": script, "domainStrategy": "AsIs"}))
			routing.DomainStrategy = types.StringValue("IPIfNonMatch")
			routingWire := buildXrayRoutingJSON(expandXrayRouting(routing)).(map[string]any)
			if got, ok := routingWire["script"].(string); !ok || got != script {
				t.Fatalf("routing script lost after unrelated edit: %#v", routingWire)
			}
			dns := flattenXrayDNS(flattenXrayDNSToMap(map[string]any{"script": script, "servers": []any{"8.8.8.8"}}))
			dns.QueryStrategy = types.StringValue("UseIPv4")
			dnsWire := buildXrayDNSJSON(expandXrayDNS(dns)).(map[string]any)
			if got, ok := dnsWire["script"].(string); !ok || got != script {
				t.Fatalf("DNS script lost after unrelated edit: %#v", dnsWire)
			}
		})
	}
	for _, value := range []types.String{types.StringNull(), types.StringUnknown()} {
		routing := buildXrayRoutingJSON(expandXrayRouting(&XrayRoutingModel{Script: value})).(map[string]any)
		dns := buildXrayDNSJSON(expandXrayDNS(&XrayDNSModel{Script: value})).(map[string]any)
		if _, ok := routing["script"]; ok {
			t.Fatalf("unset routing script emitted: %#v", routing)
		}
		if _, ok := dns["script"]; ok {
			t.Fatalf("unset DNS script emitted: %#v", dns)
		}
	}
	if !flattenXrayRouting(flattenXrayRoutingToMap(map[string]any{})).Script.IsNull() || !flattenXrayDNS(flattenXrayDNSToMap(map[string]any{})).Script.IsNull() {
		t.Fatal("old panel without scripts must produce null script attributes")
	}
}

func TestDNSServerIDRoundTrip(t *testing.T) {
	// An ID is the only extra field: serializing this as an address-only string
	// would remove the ID that a Lua script uses to select the resolver.
	for _, id := range []types.String{types.StringValue("resolver-a"), types.StringValue(""), types.StringNull(), types.StringUnknown()} {
		t.Run(id.String(), func(t *testing.T) {
			model := &XrayDNSModel{Server: []XrayDNSServer{{ID: id, Address: types.StringValue("8.8.8.8")}}}
			wire := buildXrayDNSJSON(expandXrayDNS(model)).(map[string]any)
			servers := wire["servers"].([]any)
			if id.IsNull() || id.IsUnknown() {
				if servers[0] != "8.8.8.8" {
					t.Fatalf("address-only server must stay a string: %#v", servers)
				}
			} else if server, ok := servers[0].(map[string]any); !ok || server["id"] != id.ValueString() {
				t.Fatalf("server ID must force object encoding: %#v", servers)
			}
			out := flattenXrayDNS(flattenXrayDNSToMap(wire))
			want := id
			if want.IsUnknown() {
				want = types.StringNull()
			}
			if len(out.Server) != 1 || !out.Server[0].ID.Equal(want) {
				t.Fatalf("server ID lost: %#v", out.Server)
			}
		})
	}
}

func TestTLSUseSystemCARoundTrip(t *testing.T) {
	for _, value := range []types.Bool{types.BoolValue(true), types.BoolValue(false), types.BoolNull(), types.BoolUnknown()} {
		t.Run(value.String(), func(t *testing.T) {
			model := &XrayOutboundsModel{Outbound: []XrayOutboundEntry{{
				Tag: types.StringValue("upstream"), Protocol: types.StringValue("trojan"),
				StreamSettings: &InboundStreamSettingsModel{Network: types.StringValue("tcp"), Security: types.StringValue("tls"), TLSSettings: &InboundTLSSettingsModel{UseSystemCA: value}},
			}}}
			untyped := expandXrayOutbounds(model)
			wire := expandOutbounds(untyped["outbound"].([]any))
			ss := wire[0].(map[string]any)["streamSettings"].(map[string]any)
			tls, _ := ss["tlsSettings"].(map[string]any)
			if value.IsNull() || value.IsUnknown() {
				if _, ok := tls["useSystemCA"]; ok {
					t.Fatalf("unset CA switch emitted: %#v", tls)
				}
			} else if got, ok := tls["useSystemCA"].(bool); !ok || got != value.ValueBool() {
				t.Fatalf("CA switch lost: %#v", tls)
			}
			// Exercise the actual JSON boundary, where bool and string types matter.
			raw, err := json.Marshal(wire)
			if err != nil {
				t.Fatal(err)
			}
			var decoded []any
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatal(err)
			}
			out := flattenXrayOutbounds(map[string]any{"outbound": flattenOutbounds(decoded)})
			want := value
			if want.IsUnknown() {
				want = types.BoolNull()
			}
			got := types.BoolNull()
			if out.Outbound[0].StreamSettings.TLSSettings != nil {
				got = out.Outbound[0].StreamSettings.TLSSettings.UseSystemCA
			}
			if !got.Equal(want) {
				t.Fatalf("use_system_ca: want %s, got %s", want, got)
			}
		})
	}
}
