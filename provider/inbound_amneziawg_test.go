package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// awgFullServerModel is a server block with every attribute populated, so the
// round-trip tests below fail on a key that is dropped anywhere along
// model → untyped map → JSON → untyped map → model.
func awgFullServerModel() *InboundAmneziawgServerModel {
	return &InboundAmneziawgServerModel{
		PrivateKey:             types.StringValue("privateKeyForTestsOnly000000000000000000000="),
		PublicKey:              types.StringValue("publicKeyForTestsOnly0000000000000000000000="),
		SubnetIP:               types.StringValue("10.8.1.0"),
		SubnetCIDR:             types.Int64Value(24),
		MTU:                    types.Int64Value(1380),
		PrimaryDNS:             types.StringValue("8.8.8.8"),
		SecondaryDNS:           types.StringValue("8.8.4.4"),
		ExternalInterface:      types.StringValue("eth0"),
		IPv6Enabled:            types.BoolValue(true),
		IPv6Subnet:             types.StringValue("fd00:8:1::/64"),
		IPv6ExternalInterface:  types.StringValue("eth1"),
		RouteThroughXray:       types.BoolValue(true),
		KernelMode:             types.BoolValue(false),
		Jc:                     types.Int64Value(4),
		Jmin:                   types.Int64Value(56),
		Jmax:                   types.Int64Value(190),
		S1:                     types.Int64Value(88),
		S2:                     types.Int64Value(33),
		S3:                     types.Int64Value(41),
		S4:                     types.Int64Value(19),
		H1:                     types.StringValue("1553264530-1553265530"),
		H2:                     types.StringValue("2553264530-2553265530"),
		H3:                     types.StringValue("3553264530-3553265530"),
		H4:                     types.StringValue("4553264530-4553265530"),
		I1:                     types.StringValue("<r 64>"),
		I2:                     types.StringValue("<r 48>"),
		I3:                     types.StringValue("<r 40>"),
		I4:                     types.StringValue("<r 36>"),
		I5:                     types.StringValue("<r 32>"),
		HeaderProtectionKey:    types.StringValue("MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="),
		ContentPaddingAddition: types.StringValue("12-40"),
		RekeyAfterTime:         types.StringValue("110-140"),
		RekeyTimeout:           types.StringValue("4-7"),
		RejectAfterTime:        types.StringValue("180-220"),
		KeepaliveTimeout:       types.StringValue("9-13"),
		MaxHandshakeAttempts:   types.StringValue("18-24"),
		RandomTrailers:         types.BoolValue(true),
		DisableCookies:         types.BoolValue(false),
	}
}

func awgClientModel() InboundAmneziawgClientModel {
	return InboundAmneziawgClientModel{
		Email:           types.StringValue("peer-one@test.com"),
		PrivateKey:      types.StringValue("clientPrivateKey0000000000000000000000000000="),
		PublicKey:       types.StringValue("clientPublicKey00000000000000000000000000000="),
		PreSharedKey:    types.StringValue("presharedKey000000000000000000000000000000000="),
		AllowedIPs:      anySliceToTypesList([]any{"10.8.1.2/32", "fd00:8:1::2/128"}),
		KeepAlive:       types.Int64Value(25),
		ForwardedPorts:  types.StringValue("80,443,8000-8100"),
		Enable:          types.BoolValue(true),
		LimitIP:         types.Int64Value(2),
		TotalGB:         types.Int64Value(107374182400),
		ExpiryTime:      types.Int64Value(1767225600000),
		TgID:            types.Int64Value(12345),
		SubID:           types.StringValue("abcdef0123456789"),
		Comment:         types.StringValue("first peer"),
		Reset:           types.Int64Value(0),
		Group:           types.StringValue("mobile"),
		ResetDay:        types.Int64Value(1),
		ResetMax:        types.Int64Value(12),
		TrafficReset:    types.StringValue("monthly"),
		TrafficResetDay: types.Int64Value(5),
		CreatedAt:       types.Int64Value(1767225600000),
		UpdatedAt:       types.Int64Value(1767225600001),
	}
}

// The full path an inbound settings blob travels on apply and read back:
// typed model → expandSettingsFromModel → buildSettingsJSON → flattenSettings →
// flattenSettingsToModel. Every attribute must survive it unchanged.
func TestExpandFlattenAmneziawgSettingsModel_RoundTrip(t *testing.T) {
	original := &InboundResourceModel{
		AmneziawgSettings: &InboundAmneziawgSettingsModel{
			Server:  awgFullServerModel(),
			Clients: []InboundAmneziawgClientModel{awgClientModel()},
		},
	}

	expanded := expandSettingsFromModel("amneziawg", original)
	if expanded == nil {
		t.Fatal("expected non-nil expansion for amneziawg")
	}
	server, ok := expanded["server"].(map[string]any)
	if !ok {
		t.Fatalf("expected a nested server map, got %T", expanded["server"])
	}
	// The expander must emit the upstream camelCase spelling directly — settings.go
	// forwards this object verbatim rather than translating it.
	if server["subnetIp"] != "10.8.1.0" {
		t.Errorf("expected camelCase subnetIp, got %v (keys: %v)", server["subnetIp"], mapKeys(server))
	}
	if server["headerProtectionKey"] == nil {
		t.Error("headerProtectionKey missing from the expanded server block")
	}

	raw := buildSettingsJSON(expanded, "amneziawg")

	var wire map[string]any
	if err := json.Unmarshal([]byte(raw), &wire); err != nil {
		t.Fatalf("buildSettingsJSON produced invalid JSON: %v (%s)", err, raw)
	}
	if _, ok := wire["server"]; !ok {
		t.Fatalf("server block missing from the wire payload: %s", raw)
	}
	if _, ok := wire["clients"]; !ok {
		t.Fatalf("clients array missing from the wire payload: %s", raw)
	}

	flattened, err := flattenSettings(raw, "amneziawg")
	if err != nil {
		t.Fatalf("flattenSettings: %v", err)
	}
	if len(flattened) != 1 {
		t.Fatalf("expected one settings entry, got %d", len(flattened))
	}

	restoredModel := &InboundResourceModel{}
	flattenSettingsToModel("amneziawg", flattened[0].(map[string]any), restoredModel)

	got := restoredModel.AmneziawgSettings
	if got == nil {
		t.Fatal("amneziawg_settings was dropped on the way back")
	}
	if got.Server == nil {
		t.Fatal("server block was dropped on the way back")
	}
	if *got.Server != *original.AmneziawgSettings.Server {
		t.Errorf("server block changed across the round trip:\n before: %+v\n after:  %+v",
			*original.AmneziawgSettings.Server, *got.Server)
	}
	if len(got.Clients) != 1 {
		t.Fatalf("expected one client, got %d", len(got.Clients))
	}
	before, after := original.AmneziawgSettings.Clients[0], got.Clients[0]
	if !before.AllowedIPs.Equal(after.AllowedIPs) {
		t.Errorf("allowed_ips changed: %v -> %v", before.AllowedIPs, after.AllowedIPs)
	}
	if before.ForwardedPorts != after.ForwardedPorts {
		t.Errorf("forwarded_ports changed: %v -> %v", before.ForwardedPorts, after.ForwardedPorts)
	}
	// The renewal fields are the ones a practitioner sets in the panel UI; the
	// inbound rewrites clients[] wholesale, so any of them the provider fails to
	// carry is silently zeroed on the next apply.
	if before.Group != after.Group || before.ResetDay != after.ResetDay ||
		before.ResetMax != after.ResetMax || before.TrafficReset != after.TrafficReset ||
		before.TrafficResetDay != after.TrafficResetDay ||
		before.CreatedAt != after.CreatedAt || before.UpdatedAt != after.UpdatedAt {
		t.Errorf("client renewal fields changed across the round trip:\n before: %+v\n after:  %+v", before, after)
	}
	if before.Email != after.Email || before.PrivateKey != after.PrivateKey ||
		before.PublicKey != after.PublicKey || before.PreSharedKey != after.PreSharedKey ||
		before.KeepAlive != after.KeepAlive || before.Enable != after.Enable ||
		before.LimitIP != after.LimitIP || before.TotalGB != after.TotalGB ||
		before.ExpiryTime != after.ExpiryTime || before.TgID != after.TgID ||
		before.SubID != after.SubID || before.Comment != after.Comment ||
		before.Reset != after.Reset {
		t.Errorf("client changed across the round trip:\n before: %+v\n after:  %+v", before, after)
	}
}

// A blank private key makes the panel generate a NEW server keypair, silently
// invalidating every existing client config. Nulls must therefore be omitted
// from the payload rather than sent as "".
func TestExpandAmneziawgServer_OmitsUnsetFields(t *testing.T) {
	server := expandAmneziawgServerFromModel(&InboundAmneziawgServerModel{
		SubnetIP:   types.StringValue("10.9.0.0"),
		PrivateKey: types.StringNull(),
		MTU:        types.Int64Unknown(),
	})

	if _, ok := server["privateKey"]; ok {
		t.Error("a null private_key must not be sent — the panel would rotate the server keypair")
	}
	if _, ok := server["mtu"]; ok {
		t.Error("an unknown mtu must not be sent")
	}
	if server["subnetIp"] != "10.9.0.0" {
		t.Errorf("configured value lost: %v", server["subnetIp"])
	}
}

// Upstream declares ipv6Enabled/routeThroughXray/disableCookies with omitempty,
// so a false value is stripped from the stored blob. Reading a stripped key back
// as null would break a configuration that sets it to false explicitly
// ("inconsistent result after apply"), so absence must read as false.
func TestFlattenAmneziawgServer_StrippedBoolsReadAsFalse(t *testing.T) {
	server := flattenAmneziawgServerToModel(map[string]any{"subnetIp": "10.8.1.0"})
	if server == nil {
		t.Fatal("expected a server model")
	}
	for name, got := range map[string]types.Bool{
		"ipv6_enabled":       server.IPv6Enabled,
		"route_through_xray": server.RouteThroughXray,
		"random_trailers":    server.RandomTrailers,
		"disable_cookies":    server.DisableCookies,
	} {
		if got.IsNull() || got.ValueBool() {
			t.Errorf("%s should read as false when the panel strips the key, got %v", name, got)
		}
	}
	// String and int fields keep null for absent keys — the panel fills them in.
	if !server.MTU.IsNull() {
		t.Errorf("absent mtu should stay null, got %v", server.MTU)
	}
	if !server.HeaderProtectionKey.IsNull() {
		t.Errorf("absent header_protection_key should stay null, got %v", server.HeaderProtectionKey)
	}
}

// primaryDns/secondaryDns and h1-h4 are NOT omitempty upstream: an empty string
// is a meaningful cleared value there, and must survive as "" rather than
// collapsing to null.
func TestFlattenAmneziawgServer_KeepsClearedStrings(t *testing.T) {
	server := flattenAmneziawgServerToModel(map[string]any{
		"primaryDns":   "",
		"secondaryDns": "",
		"h3":           "",
	})
	if server == nil {
		t.Fatal("expected a server model")
	}
	for name, got := range map[string]types.String{
		"primary_dns":   server.PrimaryDNS,
		"secondary_dns": server.SecondaryDNS,
		"h3":            server.H3,
	} {
		if got.IsNull() {
			t.Errorf("%s must read back as an empty string, not null", name)
		}
	}
}

// An AmneziaWG inbound with no settings at all is valid: the panel generates the
// whole server block on save. The expander must not invent one.
func TestExpandAmneziawgSettings_EmptyModel(t *testing.T) {
	if got := expandAmneziawgInboundSettings(nil); got != nil {
		t.Errorf("nil model should expand to nil, got %v", got)
	}
	got := expandAmneziawgInboundSettings(&InboundAmneziawgSettingsModel{})
	if len(got) != 0 {
		t.Errorf("empty model should expand to an empty map, got %v", got)
	}
	if flattenAmneziawgInboundSettings(map[string]any{}) != nil {
		t.Error("empty settings should flatten to nil")
	}
}

// clients[] is protocol-gated in settings.go: forwarding it for a protocol whose
// clients belong to threexui_inbound_client would make the two resources fight.
// AmneziaWG must be on the same side of that gate as WireGuard (#342).
func TestAmneziawgClientsSurviveSettingsJSON(t *testing.T) {
	item := map[string]any{
		"server":  map[string]any{"subnetIp": "10.8.1.0"},
		"clients": []any{map[string]any{"email": "peer@test.com"}},
	}

	raw := buildSettingsJSON(item, "amneziawg")
	var wire map[string]any
	if err := json.Unmarshal([]byte(raw), &wire); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := wire["clients"]; !ok {
		t.Fatalf("clients[] stripped for amneziawg: %s", raw)
	}

	// The same payload under a client-resource-owned protocol must lose clients[].
	rawVless := buildSettingsJSON(item, "vless")
	var wireVless map[string]any
	if err := json.Unmarshal([]byte(rawVless), &wireVless); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := wireVless["clients"]; ok {
		t.Errorf("clients[] must stay stripped for vless: %s", rawVless)
	}
	if _, ok := wireVless["server"]; ok {
		t.Errorf("the amneziawg server block must not leak into other protocols: %s", rawVless)
	}
}

// The nested server block is a schema.SingleNestedBlock, and clients is a list
// block — the shapes docs and plan modifiers depend on.
func TestAmneziawgSettingsBlockShape(t *testing.T) {
	block := amneziawgSettingsBlock()

	server, ok := block.Blocks["server"].(schema.SingleNestedBlock)
	if !ok {
		t.Fatalf("server must be a SingleNestedBlock, got %T", block.Blocks["server"])
	}
	if _, ok := block.Blocks["clients"].(schema.ListNestedBlock); !ok {
		t.Fatalf("clients must be a ListNestedBlock, got %T", block.Blocks["clients"])
	}

	for _, name := range []string{"private_key", "header_protection_key"} {
		if !server.Attributes[name].IsSensitive() {
			t.Errorf("server.%s must be Sensitive", name)
		}
	}

	// Every documented server field must exist in the schema, with the tfsdk
	// model and the schema agreeing on the attribute set.
	want := []string{
		"private_key", "public_key", "subnet_ip", "subnet_cidr", "mtu",
		"primary_dns", "secondary_dns", "external_interface",
		"ipv6_enabled", "ipv6_subnet", "ipv6_external_interface", "route_through_xray", "kernel_mode",
		"jc", "jmin", "jmax", "s1", "s2", "s3", "s4",
		"h1", "h2", "h3", "h4", "i1", "i2", "i3", "i4", "i5",
		"header_protection_key", "content_padding_addition",
		"rekey_after_time", "rekey_timeout", "reject_after_time",
		"keepalive_timeout", "max_handshake_attempts",
		"random_trailers", "disable_cookies",
	}
	for _, name := range want {
		if _, ok := server.Attributes[name]; !ok {
			t.Errorf("server block is missing attribute %q", name)
		}
	}
	if len(server.Attributes) != len(want) {
		t.Errorf("server block has %d attributes, expected %d — update the model, the docs and this list together",
			len(server.Attributes), len(want))
	}
}

// The validator is the only thing standing between a practitioner and a silent
// server-key rotation, so it has to fire on exactly the right configurations.
func TestAmneziawgServerRequiredValidator(t *testing.T) {
	cases := []struct {
		name      string
		model     InboundResourceModel
		wantError bool
	}{
		{
			name: "amneziawg without a settings block is rejected",
			model: InboundResourceModel{
				Protocol: types.StringValue("amneziawg"),
			},
			wantError: true,
		},
		{
			name: "amneziawg with settings but no server block is rejected",
			model: InboundResourceModel{
				Protocol:          types.StringValue("amneziawg"),
				AmneziawgSettings: &InboundAmneziawgSettingsModel{},
			},
			wantError: true,
		},
		{
			name: "an empty server block is enough — the panel fills it in",
			model: InboundResourceModel{
				Protocol: types.StringValue("amneziawg"),
				AmneziawgSettings: &InboundAmneziawgSettingsModel{
					Server: &InboundAmneziawgServerModel{},
				},
			},
		},
		{
			name: "other protocols are untouched",
			model: InboundResourceModel{
				Protocol: types.StringValue("wireguard"),
			},
		},
		{
			name: "an unknown protocol cannot be checked here",
			model: InboundResourceModel{
				Protocol: types.StringUnknown(),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotError := amneziawgConfigRejected(t, tc.model)
			if gotError != tc.wantError {
				t.Errorf("validator error = %v, want %v", gotError, tc.wantError)
			}
		})
	}
}

// amneziawgConfigRejected runs the validator against a config built from model
// and reports whether it produced an error diagnostic.
func amneziawgConfigRejected(t *testing.T, model InboundResourceModel) bool {
	t.Helper()

	var schemaResp resource.SchemaResponse
	(&InboundResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("building schema: %v", schemaResp.Diagnostics)
	}

	ctx := context.Background()
	obj, diags := types.ObjectValueFrom(ctx, schemaResp.Schema.Type().(types.ObjectType).AttrTypes, model)
	if diags.HasError() {
		t.Fatalf("building config object: %v", diags)
	}
	raw, err := obj.ToTerraformValue(ctx)
	if err != nil {
		t.Fatalf("converting config object: %v", err)
	}

	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{Raw: raw, Schema: schemaResp.Schema},
	}
	resp := &resource.ValidateConfigResponse{}
	amneziawgServerRequiredValidator{}.ValidateResource(ctx, req, resp)
	return resp.Diagnostics.HasError()
}

// The validator has to be wired into the resource, or it never runs.
func TestInboundResourceRegistersAmneziawgValidator(t *testing.T) {
	validators := (&InboundResource{}).ConfigValidators(context.Background())
	for _, v := range validators {
		if _, ok := v.(amneziawgServerRequiredValidator); ok {
			if v.Description(context.Background()) == "" {
				t.Error("validator must describe itself")
			}
			return
		}
	}
	t.Fatalf("threexui_inbound does not register the amneziawg server validator (got %d validators)", len(validators))
}

// The two-phase create is the only thing keeping a partial server block from
// silently disabling obfuscation, so both halves are pinned here.
func TestSplitAmneziawgServer(t *testing.T) {
	cases := []struct {
		name        string
		settings    string
		wantRest    string
		wantServer  map[string]any
		wantNoSplit bool
	}{
		{
			name:       "server is lifted out, clients stay",
			settings:   `{"server":{"subnetIp":"10.9.1.0"},"clients":[{"email":"a@b.c"}]}`,
			wantRest:   `{"clients":[{"email":"a@b.c"}]}`,
			wantServer: map[string]any{"subnetIp": "10.9.1.0"},
		},
		{
			name:        "no server object: nothing to hold back",
			settings:    `{"clients":[]}`,
			wantNoSplit: true,
		},
		{
			name:        "empty server object is not worth a second write",
			settings:    `{"server":{}}`,
			wantNoSplit: true,
		},
		{
			name:        "empty settings",
			settings:    "",
			wantNoSplit: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rest, server, err := splitAmneziawgServer(tc.settings)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNoSplit {
				if server != nil {
					t.Errorf("expected no server to be split off, got %v", server)
				}
				if rest != tc.settings {
					t.Errorf("settings should be untouched, got %s", rest)
				}
				return
			}
			if rest != tc.wantRest {
				t.Errorf("rest = %s, want %s", rest, tc.wantRest)
			}
			if len(server) != len(tc.wantServer) {
				t.Fatalf("server = %v, want %v", server, tc.wantServer)
			}
			for k, v := range tc.wantServer {
				if server[k] != v {
					t.Errorf("server[%q] = %v, want %v", k, server[k], v)
				}
			}
		})
	}
}

func TestSplitAmneziawgServer_InvalidJSON(t *testing.T) {
	if _, _, err := splitAmneziawgServer(`{"server":`); err == nil {
		t.Error("expected an error on malformed settings")
	}
}

func TestApplyAmneziawgServerOverrides(t *testing.T) {
	// What the panel generates when it is handed no server block.
	generated := `{"server":{"privateKey":"gen","publicKey":"genpub","subnetIp":"10.8.1.0","subnetCidr":24,"jc":6,"jmin":60,"h1":"1553264530-1553265530"},"clients":[]}`

	merged, changed, err := applyAmneziawgServerOverrides(generated, map[string]any{
		"subnetIp":   "10.9.1.0",
		"primaryDns": "1.1.1.1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected the merge to report a change")
	}

	var parsed struct {
		Server map[string]any `json:"server"`
	}
	if err := json.Unmarshal([]byte(merged), &parsed); err != nil {
		t.Fatalf("merged settings are not valid JSON: %v", err)
	}

	// Configured values win.
	if parsed.Server["subnetIp"] != "10.9.1.0" {
		t.Errorf("configured subnetIp lost: %v", parsed.Server["subnetIp"])
	}
	if parsed.Server["primaryDns"] != "1.1.1.1" {
		t.Errorf("configured primaryDns lost: %v", parsed.Server["primaryDns"])
	}
	// Generated obfuscation survives — this is the whole point.
	if parsed.Server["jc"] != float64(6) || parsed.Server["jmin"] != float64(60) {
		t.Errorf("generated obfuscation was clobbered: jc=%v jmin=%v", parsed.Server["jc"], parsed.Server["jmin"])
	}
	if parsed.Server["h1"] != "1553264530-1553265530" {
		t.Errorf("generated h1 was clobbered: %v", parsed.Server["h1"])
	}
	if parsed.Server["privateKey"] != "gen" {
		t.Errorf("generated keypair was clobbered: %v", parsed.Server["privateKey"])
	}
}

// A no-op merge must not trigger a second write to the panel.
func TestApplyAmneziawgServerOverrides_NoChange(t *testing.T) {
	generated := `{"server":{"subnetIp":"10.8.1.0","subnetCidr":24}}`

	if _, changed, err := applyAmneziawgServerOverrides(generated, nil); err != nil || changed {
		t.Errorf("empty overrides: changed=%v err=%v", changed, err)
	}
	// Numbers survive a JSON round trip as float64, so an int64 override that
	// matches must not read as a difference.
	if _, changed, err := applyAmneziawgServerOverrides(generated, map[string]any{"subnetCidr": int64(24)}); err != nil || changed {
		t.Errorf("identical value should not count as a change: changed=%v err=%v", changed, err)
	}
	if _, changed, err := applyAmneziawgServerOverrides(generated, map[string]any{"subnetCidr": int64(25)}); err != nil || !changed {
		t.Errorf("a real difference must be applied: changed=%v err=%v", changed, err)
	}
}

// clients[] is preserved on update only for protocols whose peers belong to
// threexui_inbound_client. Getting this wrong makes removing the last WireGuard
// or AmneziaWG peer impossible.
func TestProtocolOwnsClients(t *testing.T) {
	for _, p := range []string{"wireguard", "amneziawg"} {
		if !protocolOwnsClients(p) {
			t.Errorf("%s peers are managed by threexui_inbound and must not be preserved on update", p)
		}
	}
	for _, p := range []string{"vless", "vmess", "trojan", "shadowsocks", "hysteria", "mtproto"} {
		if protocolOwnsClients(p) {
			t.Errorf("%s clients belong to threexui_inbound_client and must be preserved", p)
		}
	}
}

// The clients block carries the fix for the panel's keyless-peer rejection, and
// mutation testing showed nothing covered it.
func TestAmneziawgClientsBlockShape(t *testing.T) {
	clients := amneziawgClientsBlock()

	publicKey, ok := clients.NestedObject.Attributes["public_key"]
	if !ok {
		t.Fatal("clients block has no public_key attribute")
	}
	if !publicKey.IsRequired() {
		t.Error("clients.public_key must be Required: the panel rejects a keyless peer on inbound create and update, " +
			"and never derives the key on this path")
	}

	for _, name := range []string{"private_key", "pre_shared_key"} {
		if !clients.NestedObject.Attributes[name].IsSensitive() {
			t.Errorf("clients.%s must be Sensitive", name)
		}
	}

	want := []string{
		"email", "private_key", "public_key", "pre_shared_key", "allowed_ips",
		"keep_alive", "keep_alive_range", "client_host_rule_ids", "forwarded_ports", "enable", "limit_ip", "total_gb",
		"expiry_time", "tg_id", "sub_id", "comment", "reset", "group",
		"reset_day", "reset_max", "traffic_reset", "traffic_reset_day",
		"created_at", "updated_at",
	}
	for _, name := range want {
		if _, ok := clients.NestedObject.Attributes[name]; !ok {
			t.Errorf("clients block is missing attribute %q", name)
		}
	}
	if len(clients.NestedObject.Attributes) != len(want) {
		t.Errorf("clients block has %d attributes, expected %d — update the model, the docs and this list together",
			len(clients.NestedObject.Attributes), len(want))
	}
}

// The cross-field rules the panel would otherwise reject mid-apply.
func TestAmneziawgServerConstraints(t *testing.T) {
	cases := []struct {
		name      string
		server    InboundAmneziawgServerModel
		wantError bool
	}{
		{
			name:      "jmin above jmax",
			server:    InboundAmneziawgServerModel{Jmin: types.Int64Value(90), Jmax: types.Int64Value(50)},
			wantError: true,
		},
		{
			name:   "jmin below jmax is fine",
			server: InboundAmneziawgServerModel{Jmin: types.Int64Value(50), Jmax: types.Int64Value(90)},
		},
		{
			name:      "s1 + 56 equals s2",
			server:    InboundAmneziawgServerModel{S1: types.Int64Value(30), S2: types.Int64Value(86)},
			wantError: true,
		},
		{
			name:   "s1 + 56 differs from s2",
			server: InboundAmneziawgServerModel{S1: types.Int64Value(30), S2: types.Int64Value(87)},
		},
		{
			name: "header protection with a junk size below 12",
			server: InboundAmneziawgServerModel{
				HeaderProtectionKey: types.StringValue("MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="),
				S1:                  types.Int64Value(20), S2: types.Int64Value(20),
				S3: types.Int64Value(8), S4: types.Int64Value(20),
			},
			wantError: true,
		},
		{
			name: "header protection with junk sizes at the floor",
			server: InboundAmneziawgServerModel{
				HeaderProtectionKey: types.StringValue("MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="),
				S1:                  types.Int64Value(12), S2: types.Int64Value(20),
				S3: types.Int64Value(12), S4: types.Int64Value(13),
			},
		},
		{
			name:      "ipv6 enabled without a subnet",
			server:    InboundAmneziawgServerModel{IPv6Enabled: types.BoolValue(true), IPv6Subnet: types.StringNull()},
			wantError: true,
		},
		{
			name:   "ipv6 enabled with a subnet",
			server: InboundAmneziawgServerModel{IPv6Enabled: types.BoolValue(true), IPv6Subnet: types.StringValue("fd00:8:1::/64")},
		},
		{
			name:   "ipv6 disabled needs no subnet",
			server: InboundAmneziawgServerModel{IPv6Enabled: types.BoolValue(false)},
		},
		{
			name:   "unknown values are left to the panel",
			server: InboundAmneziawgServerModel{Jmin: types.Int64Unknown(), Jmax: types.Int64Value(50)},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &resource.ValidateConfigResponse{}
			server := tc.server
			validateAmneziawgServerConstraints(&server, resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantError {
				t.Errorf("error = %v, want %v (%v)", got, tc.wantError, resp.Diagnostics)
			}
		})
	}
}

func TestAmneziawgValidatorDescriptions(t *testing.T) {
	v := amneziawgServerRequiredValidator{}
	ctx := context.Background()
	if v.Description(ctx) == "" {
		t.Error("Description must not be empty")
	}
	if v.MarkdownDescription(ctx) != v.Description(ctx) {
		t.Error("MarkdownDescription should mirror Description")
	}
}

// Absent and malformed payloads must produce a nil model rather than a
// half-populated one, so a broken panel response cannot land in state.
func TestFlattenAmneziawgInboundSettings_Edges(t *testing.T) {
	if got := flattenAmneziawgInboundSettings(nil); got != nil {
		t.Errorf("nil data should flatten to nil, got %+v", got)
	}
	if got := flattenAmneziawgInboundSettings(map[string]any{"server": "not-an-object"}); got != nil {
		t.Errorf("a non-object server should flatten to nil, got %+v", got)
	}
	if got := flattenAmneziawgServerToModel(nil); got != nil {
		t.Errorf("nil server should flatten to nil, got %+v", got)
	}
	if got := flattenAmneziawgClientsToModel([]any{"not-an-object"}); got != nil {
		t.Errorf("a client list with no usable entries should flatten to nil, got %+v", got)
	}

	// A clients array alone (no server) is still a valid block.
	got := flattenAmneziawgInboundSettings(map[string]any{
		"clients": []any{map[string]any{"email": "a@b.c"}},
	})
	if got == nil || len(got.Clients) != 1 {
		t.Fatalf("expected one client, got %+v", got)
	}
	if got.Server != nil {
		t.Errorf("no server was present, so none should be modelled: %+v", got.Server)
	}
	// An absent allowed_ips must be a null list, not an empty one — the two plan
	// differently.
	if !got.Clients[0].AllowedIPs.IsNull() {
		t.Errorf("absent allowedIPs should read as null, got %v", got.Clients[0].AllowedIPs)
	}
}

// preserveInboundSettings must not resurrect peers the plan removed for
// protocols where the inbound owns clients[], and must keep preserving them for
// the protocols where threexui_inbound_client does.
func TestPreserveInboundSettings_ClientOwnership(t *testing.T) {
	existingWithClients := `{"clients":[{"email":"kept@test.com"}],"testseed":[900,500]}`

	t.Run("amneziawg peer removal survives", func(t *testing.T) {
		desired := &Inbound{Protocol: "amneziawg", Settings: `{"server":{"subnetIp":"10.8.1.0"}}`}
		existing := &Inbound{Protocol: "amneziawg", Settings: existingWithClients}

		if err := preserveInboundSettings(desired, existing); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(desired.Settings, "kept@test.com") {
			t.Errorf("the removed peer was put back, so the last peer could never be deleted: %s", desired.Settings)
		}
		if !strings.Contains(desired.Settings, "10.8.1.0") {
			t.Errorf("the server block was lost: %s", desired.Settings)
		}
	})

	t.Run("wireguard peer removal survives", func(t *testing.T) {
		desired := &Inbound{Protocol: "wireguard", Settings: `{"mtu":[1420]}`}
		existing := &Inbound{Protocol: "wireguard", Settings: existingWithClients}

		if err := preserveInboundSettings(desired, existing); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(desired.Settings, "kept@test.com") {
			t.Errorf("the removed peer was put back: %s", desired.Settings)
		}
	})

	t.Run("vless clients are still preserved", func(t *testing.T) {
		desired := &Inbound{Protocol: "vless", Settings: `{"decryption":"none"}`}
		existing := &Inbound{Protocol: "vless", Settings: existingWithClients}

		if err := preserveInboundSettings(desired, existing); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(desired.Settings, "kept@test.com") {
			t.Errorf("clients owned by threexui_inbound_client must be preserved: %s", desired.Settings)
		}
	})

	t.Run("testseed is preserved for inbound-owned protocols too", func(t *testing.T) {
		desired := &Inbound{Protocol: "amneziawg", Settings: `{"server":{}}`}
		existing := &Inbound{Protocol: "amneziawg", Settings: existingWithClients}

		if err := preserveInboundSettings(desired, existing); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(desired.Settings, "testseed") {
			t.Errorf("testseed should still be carried over: %s", desired.Settings)
		}
	})
}

// alignBlocksWithPlan nils blocks the configuration does not declare, so the
// framework does not report "was absent, but now present". The nested server
// block needs the same treatment as the top-level ones.
func TestAlignBlocksWithPlan_AmneziawgServer(t *testing.T) {
	t.Run("whole block absent from plan", func(t *testing.T) {
		state := &InboundResourceModel{AmneziawgSettings: &InboundAmneziawgSettingsModel{Server: &InboundAmneziawgServerModel{}}}
		alignBlocksWithPlan(state, &InboundResourceModel{})
		if state.AmneziawgSettings != nil {
			t.Error("amneziawg_settings should be nil when the plan has no block")
		}
	})

	t.Run("block present, nested server absent", func(t *testing.T) {
		state := &InboundResourceModel{AmneziawgSettings: &InboundAmneziawgSettingsModel{
			Server:  &InboundAmneziawgServerModel{},
			Clients: []InboundAmneziawgClientModel{{Email: types.StringValue("a@b.c")}},
		}}
		plan := &InboundResourceModel{AmneziawgSettings: &InboundAmneziawgSettingsModel{}}

		alignBlocksWithPlan(state, plan)

		if state.AmneziawgSettings == nil {
			t.Fatal("the block itself was declared and must survive")
		}
		if state.AmneziawgSettings.Server != nil {
			t.Error("the nested server block should be nil when the plan does not declare it")
		}
		if len(state.AmneziawgSettings.Clients) != 1 {
			t.Error("clients must not be touched by the server alignment")
		}
	})

	t.Run("both declared", func(t *testing.T) {
		state := &InboundResourceModel{AmneziawgSettings: &InboundAmneziawgSettingsModel{Server: &InboundAmneziawgServerModel{}}}
		plan := &InboundResourceModel{AmneziawgSettings: &InboundAmneziawgSettingsModel{Server: &InboundAmneziawgServerModel{}}}

		alignBlocksWithPlan(state, plan)

		if state.AmneziawgSettings == nil || state.AmneziawgSettings.Server == nil {
			t.Error("a declared server block must survive")
		}
	})
}

// A malformed blob from the panel must surface as an error rather than a
// half-applied merge.
func TestApplyAmneziawgServerOverrides_InvalidJSON(t *testing.T) {
	if _, _, err := applyAmneziawgServerOverrides(`{"server":`, map[string]any{"mtu": 1380}); err == nil {
		t.Error("expected an error on malformed generated settings")
	}
}

// A blob with no server object at all still has to accept the overrides — the
// panel is expected to have produced one, but the merge must not silently drop
// the configuration if it did not.
func TestApplyAmneziawgServerOverrides_NoServerInPayload(t *testing.T) {
	merged, changed, err := applyAmneziawgServerOverrides(`{"clients":[]}`, map[string]any{"subnetIp": "10.9.1.0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected the merge to report a change")
	}
	if !strings.Contains(merged, "10.9.1.0") {
		t.Errorf("configured value lost: %s", merged)
	}
}

// The second phase of the AmneziaWG create is what keeps a configured subnet
// from costing the inbound its obfuscation, so it is exercised here against a
// stub panel rather than only through the acceptance suite.
func TestApplyAmneziawgServerPhaseTwo(t *testing.T) {
	generated := `{"server":{"privateKey":"gen","publicKey":"genpub","subnetIp":"10.8.1.0","jc":6,"h1":"1-2"},"clients":[]}`

	newStubPanel := func(t *testing.T, updates *int32, failUpdate bool) *Client {
		t.Helper()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/login":
				_, _ = w.Write(okResponse(nil))
			case strings.Contains(r.URL.Path, "/update/"):
				atomic.AddInt32(updates, 1)
				if failUpdate {
					_, _ = w.Write([]byte(`{"success":false,"msg":"panel refused"}`))
					return
				}
				_, _ = w.Write(okResponse(map[string]any{"id": 7}))
			case strings.Contains(r.URL.Path, "/get/"):
				_, _ = w.Write(okResponse(map[string]any{
					"id":       7,
					"settings": `{"server":{"privateKey":"gen","publicKey":"genpub","subnetIp":"10.9.1.0","jc":6,"h1":"1-2"}}`,
				}))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		t.Cleanup(srv.Close)
		return newTestClient(t, srv.URL)
	}

	t.Run("configured fields are applied and the inbound re-read", func(t *testing.T) {
		var updates int32
		client := newStubPanel(t, &updates, false)

		got, err := applyAmneziawgServerPhaseTwo(context.Background(), client,
			&Inbound{ID: 7, Settings: generated}, map[string]any{"subnetIp": "10.9.1.0"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if atomic.LoadInt32(&updates) != 1 {
			t.Errorf("expected exactly one update call, got %d", updates)
		}
		if !strings.Contains(got.Settings, "10.9.1.0") {
			t.Errorf("configured value missing from the settled inbound: %s", got.Settings)
		}
		// The generated obfuscation must have survived the merge.
		if !strings.Contains(got.Settings, `"jc":6`) {
			t.Errorf("generated obfuscation lost: %s", got.Settings)
		}
	})

	t.Run("nothing to apply means no write", func(t *testing.T) {
		var updates int32
		client := newStubPanel(t, &updates, false)
		in := &Inbound{ID: 7, Settings: generated}

		got, err := applyAmneziawgServerPhaseTwo(context.Background(), client, in, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != in {
			t.Error("with no overrides the inbound should be returned untouched")
		}
		if atomic.LoadInt32(&updates) != 0 {
			t.Errorf("expected no update call, got %d", updates)
		}
	})

	t.Run("overrides matching the generated block do not write either", func(t *testing.T) {
		var updates int32
		client := newStubPanel(t, &updates, false)

		if _, err := applyAmneziawgServerPhaseTwo(context.Background(), client,
			&Inbound{ID: 7, Settings: generated}, map[string]any{"subnetIp": "10.8.1.0"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if atomic.LoadInt32(&updates) != 0 {
			t.Errorf("an identical value must not trigger a write, got %d", updates)
		}
	})

	t.Run("a refused update surfaces as an error", func(t *testing.T) {
		var updates int32
		client := newStubPanel(t, &updates, true)

		if _, err := applyAmneziawgServerPhaseTwo(context.Background(), client,
			&Inbound{ID: 7, Settings: generated}, map[string]any{"subnetIp": "10.9.1.0"}); err == nil {
			t.Error("expected the panel's refusal to surface")
		}
	})

	t.Run("malformed generated settings surface as an error", func(t *testing.T) {
		var updates int32
		client := newStubPanel(t, &updates, false)

		if _, err := applyAmneziawgServerPhaseTwo(context.Background(), client,
			&Inbound{ID: 7, Settings: `{"server":`}, map[string]any{"subnetIp": "10.9.1.0"}); err == nil {
			t.Error("expected a parse error")
		}
	})
}

// Peers are only released for the protocols whose clients the inbound owns.
// Reaching into any other protocol's clients[] would delete rows that
// threexui_inbound_client still tracks in its own state.
func TestInboundOwnedPeerEmails(t *testing.T) {
	awg := &InboundResourceModel{AmneziawgSettings: &InboundAmneziawgSettingsModel{
		Clients: []InboundAmneziawgClientModel{
			{Email: types.StringValue("a@test.com")},
			{Email: types.StringValue("b@test.com")},
			{Email: types.StringNull()},    // never created
			{Email: types.StringValue("")}, // nothing to release
			{Email: types.StringUnknown()}, // not known at destroy time
		},
	}}
	if got := inboundOwnedPeerEmails("amneziawg", awg); len(got) != 2 || got[0] != "a@test.com" || got[1] != "b@test.com" {
		t.Errorf("amneziawg peers = %v, want the two real emails", got)
	}

	wg := &InboundResourceModel{WireguardSettings: &InboundWireguardSettingsModel{
		Clients: []InboundWireguardClientModel{{Email: types.StringValue("wg@test.com")}},
	}}
	if got := inboundOwnedPeerEmails("wireguard", wg); len(got) != 1 || got[0] != "wg@test.com" {
		t.Errorf("wireguard peers = %v", got)
	}

	// A wireguard inbound using the legacy peer[] block has no emails to free.
	if got := inboundOwnedPeerEmails("wireguard", &InboundResourceModel{
		WireguardSettings: &InboundWireguardSettingsModel{Peer: []InboundWireguardPeerModel{{}}},
	}); len(got) != 0 {
		t.Errorf("legacy peer[] carries no emails, got %v", got)
	}

	for _, protocol := range []string{"vless", "vmess", "trojan", "mtproto"} {
		if got := inboundOwnedPeerEmails(protocol, awg); len(got) != 0 {
			t.Errorf("%s clients belong to threexui_inbound_client and must not be touched, got %v", protocol, got)
		}
	}
	if got := inboundOwnedPeerEmails("amneziawg", &InboundResourceModel{}); len(got) != 0 {
		t.Errorf("a model without the block yields nothing, got %v", got)
	}
}

func TestReleaseInboundOwnedPeers(t *testing.T) {
	newClient := func(t *testing.T, deleted *[]string, mu *sync.Mutex, fail bool) *Client {
		t.Helper()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/login":
				_, _ = w.Write(okResponse(nil))
			case strings.Contains(r.URL.Path, "/panel/api/clients/del/"):
				if fail {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				parts := strings.Split(r.URL.Path, "/")
				mu.Lock()
				*deleted = append(*deleted, parts[len(parts)-1])
				mu.Unlock()
				_, _ = w.Write(okResponse(nil))
			case strings.HasPrefix(r.URL.Path, "/panel/api/clients/list"):
				// Probed once to pick the v3.1.0+ client API.
				_, _ = w.Write(okResponse([]any{}))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		t.Cleanup(srv.Close)
		return newTestClient(t, srv.URL)
	}

	state := &InboundResourceModel{AmneziawgSettings: &InboundAmneziawgSettingsModel{
		Clients: []InboundAmneziawgClientModel{
			{Email: types.StringValue("one@test.com")},
			{Email: types.StringValue("two@test.com")},
		},
	}}

	t.Run("peers are deleted", func(t *testing.T) {
		var deleted []string
		var mu sync.Mutex
		client := newClient(t, &deleted, &mu, false)

		var d diag.Diagnostics
		releaseInboundOwnedPeers(context.Background(), client, 7, "amneziawg", state, &d)

		if d.HasError() {
			t.Fatalf("unexpected errors: %v", d.Errors())
		}
		mu.Lock()
		defer mu.Unlock()
		if len(deleted) != 2 {
			t.Fatalf("expected both peers deleted, got %v", deleted)
		}
	})

	t.Run("a protocol that does not own its clients is untouched", func(t *testing.T) {
		var deleted []string
		var mu sync.Mutex
		client := newClient(t, &deleted, &mu, false)

		var d diag.Diagnostics
		releaseInboundOwnedPeers(context.Background(), client, 7, "vless", state, &d)

		mu.Lock()
		defer mu.Unlock()
		if len(deleted) != 0 {
			t.Errorf("vless clients belong to threexui_inbound_client, got %v", deleted)
		}
	})

	t.Run("a failure warns but does not block the delete", func(t *testing.T) {
		var deleted []string
		var mu sync.Mutex
		client := newClient(t, &deleted, &mu, true)

		var d diag.Diagnostics
		releaseInboundOwnedPeers(context.Background(), client, 7, "amneziawg", state, &d)

		if d.HasError() {
			t.Error("a peer that cannot be released must not fail the destroy")
		}
		if d.WarningsCount() != 2 {
			t.Errorf("expected a warning naming each stranded peer, got %d", d.WarningsCount())
		}
	})

	t.Run("nothing to release is a no-op", func(t *testing.T) {
		var deleted []string
		var mu sync.Mutex
		client := newClient(t, &deleted, &mu, true)

		var d diag.Diagnostics
		releaseInboundOwnedPeers(context.Background(), client, 7, "amneziawg", &InboundResourceModel{}, &d)
		releaseInboundOwnedPeers(context.Background(), client, 7, "amneziawg", nil, &d)

		if d.HasError() || d.WarningsCount() != 0 {
			t.Errorf("expected silence, got %v", d)
		}
	})
}

// A repeat destroy — or a peer already removed in the panel — must be quiet.
// The panel reports the miss as `client "<email>" not found in any inbound or
// client record`, which the provider has to read as success; otherwise every
// such destroy tells the practitioner to go clean up rows that are already gone.
func TestReleaseInboundOwnedPeers_AlreadyGoneIsQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/login":
			_, _ = w.Write(okResponse(nil))
		case strings.Contains(r.URL.Path, "/panel/api/clients/del/"):
			parts := strings.Split(r.URL.Path, "/")
			email := parts[len(parts)-1]
			_, _ = w.Write(failResponse(`client "` + email + `" not found in any inbound or client record`))
		case strings.HasPrefix(r.URL.Path, "/panel/api/clients/list"):
			_, _ = w.Write(okResponse([]any{}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	state := &InboundResourceModel{AmneziawgSettings: &InboundAmneziawgSettingsModel{
		Clients: []InboundAmneziawgClientModel{{Email: types.StringValue("gone@test.com")}},
	}}

	var d diag.Diagnostics
	releaseInboundOwnedPeers(context.Background(), newTestClient(t, srv.URL), 7, "amneziawg", state, &d)

	if d.HasError() {
		t.Fatalf("unexpected errors: %v", d.Errors())
	}
	if d.WarningsCount() != 0 {
		t.Errorf("a peer that is already gone must not warn, got: %v", d.Warnings())
	}
}

// Dropping a peer from the configuration is an update, and it strands the email
// exactly as a destroy does. Only the peers that actually left may be released:
// releasing one that is still configured would delete a live peer mid-apply.
func TestReleasePeerEmailsRemovedByUpdate(t *testing.T) {
	awg := func(emails ...string) *InboundResourceModel {
		clients := make([]InboundAmneziawgClientModel, 0, len(emails))
		for _, e := range emails {
			clients = append(clients, InboundAmneziawgClientModel{Email: types.StringValue(e)})
		}
		return &InboundResourceModel{AmneziawgSettings: &InboundAmneziawgSettingsModel{Clients: clients}}
	}

	cases := []struct {
		name  string
		state *InboundResourceModel
		plan  *InboundResourceModel
		want  []string
	}{
		{
			name:  "the only peer is dropped",
			state: awg("gone@test.com"),
			plan:  awg(),
			want:  []string{"gone@test.com"},
		},
		{
			name:  "one of several is dropped",
			state: awg("keep@test.com", "gone@test.com"),
			plan:  awg("keep@test.com"),
			want:  []string{"gone@test.com"},
		},
		{
			name:  "peers only added",
			state: awg("keep@test.com"),
			plan:  awg("keep@test.com", "new@test.com"),
			want:  nil,
		},
		{
			name:  "unchanged",
			state: awg("keep@test.com"),
			plan:  awg("keep@test.com"),
			want:  nil,
		},
		{
			// A peer moved to a different position must not read as removed.
			name:  "reordered",
			state: awg("a@test.com", "b@test.com"),
			plan:  awg("b@test.com", "a@test.com"),
			want:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var deleted []string
			var mu sync.Mutex
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/login":
					_, _ = w.Write(okResponse(nil))
				case strings.Contains(r.URL.Path, "/panel/api/clients/del/"):
					parts := strings.Split(r.URL.Path, "/")
					mu.Lock()
					deleted = append(deleted, parts[len(parts)-1])
					mu.Unlock()
					_, _ = w.Write(okResponse(nil))
				case strings.HasPrefix(r.URL.Path, "/panel/api/clients/list"):
					_, _ = w.Write(okResponse([]any{}))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer srv.Close()

			var d diag.Diagnostics
			releasePeerEmailsRemovedByUpdate(context.Background(), newTestClient(t, srv.URL), 7, "amneziawg", tc.state, tc.plan, &d)

			if d.HasError() {
				t.Fatalf("unexpected errors: %v", d.Errors())
			}
			mu.Lock()
			defer mu.Unlock()
			if len(deleted) != len(tc.want) {
				t.Fatalf("released %v, want %v", deleted, tc.want)
			}
			for i, email := range tc.want {
				if deleted[i] != email {
					t.Errorf("released[%d] = %q, want %q", i, deleted[i], email)
				}
			}
		})
	}

	t.Run("nil models are a no-op", func(t *testing.T) {
		var d diag.Diagnostics
		releasePeerEmailsRemovedByUpdate(context.Background(), nil, 7, "amneziawg", nil, awg(), &d)
		releasePeerEmailsRemovedByUpdate(context.Background(), nil, 7, "amneziawg", awg(), nil, &d)
		if d.HasError() || d.WarningsCount() != 0 {
			t.Errorf("expected silence, got %v", d)
		}
	})
}
