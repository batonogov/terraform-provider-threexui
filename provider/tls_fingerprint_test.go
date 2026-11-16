package provider

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStoreInboundStreamWaitsForTLS(t *testing.T) {
	const oldStream = `{"tlsSettings":{"settings":{"fingerprint":"chrome"}}}`
	const newStream = `{"tlsSettings":{"settings":{"fingerprint":"safari"}}}`
	var reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inbound := Inbound{ID: 1, Port: 25522, StreamSettings: oldStream}
		switch r.URL.Path {
		case "/login":
			w.Write(okResponse(nil))
		case "/panel/api/inbounds/list":
			w.Write(okResponse([]Inbound{inbound}))
		case "/panel/api/inbounds/update/1":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if got := r.FormValue("streamSettings"); got != newStream {
				t.Errorf("posted streamSettings = %s, want %s", got, newStream)
			}
			w.Write(okResponse(inbound))
		case "/panel/api/inbounds/get/1":
			if reads.Add(1) > 1 {
				inbound.StreamSettings = newStream
			}
			w.Write(okResponse(inbound))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv(envEndpoint, server.URL)
	t.Setenv(envBasePath, "/")

	testAccStoreInboundStream(t, 25522, func(stream map[string]any) {
		tls := stream["tlsSettings"].(map[string]any)
		tls["settings"] = map[string]any{"fingerprint": "safari"}
	})
	if got := reads.Load(); got != 2 {
		t.Fatalf("GET count = %d, want 2: the first read still has the old TLS fingerprint", got)
	}
}

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
