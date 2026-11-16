package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestInboundTLSFingerprintWireFormat(t *testing.T) {
	for _, protocol := range []string{"vless", "trojan", "hysteria"} {
		t.Run(protocol, func(t *testing.T) {
			model := &InboundResourceModel{
				Protocol: types.StringValue(protocol),
				StreamSettings: &InboundStreamSettingsModel{
					Security: types.StringValue("tls"),
					TLSSettings: &InboundTLSSettingsModel{
						ServerName:  types.StringValue("example.com"),
						Fingerprint: types.StringValue("firefox"),
					},
				},
			}
			inbound := expandInboundFromModel(model)
			tls := decodeStream(t, inbound.StreamSettings)["tlsSettings"].(map[string]any)
			settings, _ := tls["settings"].(map[string]any)
			if settings["fingerprint"] != "firefox" {
				t.Fatalf("panel client settings = %#v, want fingerprint firefox", settings)
			}
			if _, ok := tls["fingerprint"]; ok {
				t.Fatal("inbound fingerprint must not be at the TLS root")
			}
			if tls["serverName"] != "example.com" {
				t.Fatal("server-side TLS fields must stay at the TLS root")
			}
			back, diags := inboundToModel(inbound, true)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if back.StreamSettings.TLSSettings.Fingerprint.ValueString() != "firefox" {
				t.Fatal("fingerprint lost on read-back")
			}
		})
	}
}

func TestFlattenTLSFingerprintFormats(t *testing.T) {
	for _, tc := range []struct {
		name string
		tls  string
		want string
	}{
		{"panel", `{"settings":{"fingerprint":"firefox"}}`, "firefox"},
		{"legacy provider or outbound", `{"fingerprint":"chrome"}`, "chrome"},
		{"panel wins", `{"fingerprint":"chrome","settings":{"fingerprint":"firefox"}}`, "firefox"},
		{"panel None wins", `{"fingerprint":"chrome","settings":{"fingerprint":""}}`, ""},
		{"unrelated client settings", `{"fingerprint":"chrome","settings":{"echConfigList":"abc"}}`, "chrome"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flat, err := flattenStreamSettingsToMap(`{"security":"tls","tlsSettings":` + tc.tls + `}`)
			if err != nil {
				t.Fatal(err)
			}
			model := flattenStreamSettingsToModel(flat)
			if got := model.TLSSettings.Fingerprint.ValueString(); got != tc.want {
				t.Fatalf("fingerprint = %q, want %q", got, tc.want)
			}
			if tc.want == "" {
				return
			}
			// An unrelated update migrates old provider JSON to the panel layout.
			wire := decodeStream(t, expandInboundFromModel(&InboundResourceModel{
				Protocol:       types.StringValue("vless"),
				StreamSettings: model,
			}).StreamSettings)
			tls := wire["tlsSettings"].(map[string]any)
			settings, _ := tls["settings"].(map[string]any)
			if settings["fingerprint"] != tc.want {
				t.Fatalf("rewritten client settings = %#v, want %q", settings, tc.want)
			}
		})
	}
}
