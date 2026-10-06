package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTuicExpandFlattenRoundTrip(t *testing.T) {
	alpn, diags := types.ListValueFrom(t.Context(), types.StringType, []string{"h3"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	m := &InboundTuicSettingsModel{
		Server: &InboundTuicServerModel{
			Certificate:           types.StringValue("-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"),
			PrivateKey:            types.StringValue("-----BEGIN PRIVATE KEY-----\nMC4\n-----END PRIVATE KEY-----"),
			CongestionControl:     types.StringValue("bbr"),
			ALPN:                  alpn,
			UDPRelayMode:          types.StringValue("native"),
			ZeroRTTHandshake:      types.BoolValue(true),
			LogLevel:              types.StringValue("info"),
			MaxIdleTime:           types.Int64Value(15),
			AuthenticationTimeout: types.Int64Value(3),
			MaxUDPRelayPacketSize: types.Int64Value(1500),
			SNI:                   types.StringValue("tuic.example.com"),
		},
		Clients: []InboundTuicClientModel{{
			Email:    types.StringValue("tuic-1@test.com"),
			ID:       types.StringValue("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
			Password: types.StringValue("tuic-pass-1"),
			Enable:   types.BoolValue(true),
			LimitIP:  types.Int64Value(2),
			SubID:    types.StringValue("sub123"),
			Comment:  types.StringValue("round trip"),
		}},
	}

	// Expand → wire JSON (what goes to the panel).
	wire := buildSettingsJSON(expandTuicInboundSettings(m), "tuic")
	var payload map[string]any
	if err := json.Unmarshal([]byte(wire), &payload); err != nil {
		t.Fatalf("wire JSON invalid: %v\n%s", err, wire)
	}
	server, ok := payload["server"].(map[string]any)
	if !ok {
		t.Fatalf("server block missing from wire JSON: %s", wire)
	}
	// TUIC wire keys are snake_case (tuic.TuicServerSettings json tags).
	for _, key := range []string{"certificate", "private_key", "congestion_control", "udp_relay_mode", "zero_rtt_handshake", "log_level", "max_idle_time", "authentication_timeout", "max_udp_relay_packet_size", "sni"} {
		if _, has := server[key]; !has {
			t.Errorf("server.%s missing from wire JSON: %s", key, wire)
		}
	}
	clients, ok := payload["clients"].([]any)
	if !ok || len(clients) != 1 {
		t.Fatalf("clients missing from wire JSON: %s", wire)
	}
	c0 := clients[0].(map[string]any)
	if c0["id"] != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" || c0["password"] != "tuic-pass-1" {
		t.Fatalf("client wire entry wrong: %#v", c0)
	}
	if c0["email"] != "tuic-1@test.com" || c0["subId"] != "sub123" {
		t.Fatalf("client bookkeeping keys wrong: %#v", c0)
	}

	// Wire JSON → flatten → model.
	flatList, err := flattenSettings(wire, "tuic")
	if err != nil {
		t.Fatalf("flattenSettings: %v", err)
	}
	flat, ok := flatList[0].(map[string]any)
	if !ok {
		t.Fatalf("flattenSettings returned no settings map: %#v", flatList)
	}
	back := flattenTuicInboundSettings(flat)
	if back == nil || back.Server == nil || len(back.Clients) != 1 {
		t.Fatalf("round-trip lost shape: %#v", back)
	}
	if back.Server.Certificate.ValueString() != m.Server.Certificate.ValueString() {
		t.Errorf("certificate round-trip: %#v", back.Server.Certificate)
	}
	if back.Server.MaxIdleTime.ValueInt64() != 15 || back.Server.MaxUDPRelayPacketSize.ValueInt64() != 1500 {
		t.Errorf("server ints round-trip: %#v / %#v", back.Server.MaxIdleTime, back.Server.MaxUDPRelayPacketSize)
	}
	if back.Clients[0].ID.ValueString() != m.Clients[0].ID.ValueString() {
		t.Errorf("client id round-trip: %#v", back.Clients[0].ID)
	}
	if back.Clients[0].SubID.ValueString() != "sub123" {
		t.Errorf("client subId round-trip: %#v", back.Clients[0].SubID)
	}
}

// The panel accepts either `uuid` or `id` on a stored client
// (InstanceFromInbound prefers uuid); the provider writes `id` and reads both.
func TestTuicClientUUIDSpellingFlattens(t *testing.T) {
	flatList, err := flattenSettings(`{"server":{"congestion_control":"bbr"},"clients":[{"uuid":"11111111-2222-3333-4444-555555555555","password":"p","email":"u@x"}]}`, "tuic")
	if err != nil {
		t.Fatal(err)
	}
	flat, ok := flatList[0].(map[string]any)
	if !ok {
		t.Fatalf("flattenSettings returned no settings map: %#v", flatList)
	}
	m := flattenTuicInboundSettings(flat)
	if len(m.Clients) != 1 || m.Clients[0].ID.ValueString() != "11111111-2222-3333-4444-555555555555" {
		t.Fatalf("uuid-spelled client must flatten: %#v", m.Clients)
	}
}

func TestTuicProtocolOwnsClients(t *testing.T) {
	if !protocolOwnsClients("tuic") {
		t.Fatal("tuic peers belong to threexui_inbound, not threexui_inbound_client")
	}
}

// TestTuicServerValidators mirrors the panel's normalizeTuicSettings
// (3x-ui v3.9.0 web/service/tuic_settings.go) at plan time: unknown enums are
// a hard save error upstream, and max_udp_relay_packet_size 65246-65507 is
// silently clamped to 65245 (non-round-trippable), so both must fail
// validation before apply.
func TestTuicServerValidators(t *testing.T) {
	block := tuicSettingsBlock()
	server, ok := block.Blocks["server"].(schema.SingleNestedBlock)
	if !ok {
		t.Fatalf("server block has unexpected type %T", block.Blocks["server"])
	}

	t.Run("congestion_control", func(t *testing.T) {
		attr, ok := server.Attributes["congestion_control"].(schema.StringAttribute)
		if !ok {
			t.Fatalf("congestion_control has unexpected type %T", server.Attributes["congestion_control"])
		}
		for _, valid := range []string{"bbr", "cubic", "new_reno"} {
			for _, v := range attr.Validators {
				if testStringValidator(t, v, valid) {
					t.Errorf("congestion_control should accept %q", valid)
				}
			}
		}
		// "reno" is an upstream alias the panel rewrites to "new_reno" on
		// save, so it cannot round-trip and must be rejected here.
		for _, invalid := range []string{"reno", "BBR", "vegas", ""} {
			for _, v := range attr.Validators {
				if !testStringValidator(t, v, invalid) {
					t.Errorf("congestion_control should reject %q", invalid)
				}
			}
		}
	})

	t.Run("log_level", func(t *testing.T) {
		attr, ok := server.Attributes["log_level"].(schema.StringAttribute)
		if !ok {
			t.Fatalf("log_level has unexpected type %T", server.Attributes["log_level"])
		}
		for _, valid := range []string{"debug", "info", "warn", "error"} {
			for _, v := range attr.Validators {
				if testStringValidator(t, v, valid) {
					t.Errorf("log_level should accept %q", valid)
				}
			}
		}
		// The panel case-folds and maps "warning" to "warn" on save; only the
		// canonical set round-trips.
		for _, invalid := range []string{"warning", "INFO", "trace", ""} {
			for _, v := range attr.Validators {
				if !testStringValidator(t, v, invalid) {
					t.Errorf("log_level should reject %q", invalid)
				}
			}
		}
	})

	t.Run("max_udp_relay_packet_size", func(t *testing.T) {
		attr, ok := server.Attributes["max_udp_relay_packet_size"].(schema.Int64Attribute)
		if !ok {
			t.Fatalf("max_udp_relay_packet_size has unexpected type %T", server.Attributes["max_udp_relay_packet_size"])
		}
		for _, valid := range []int64{1, 1500, 65245} {
			for _, v := range attr.Validators {
				if testInt64Validator(t, v, valid) {
					t.Errorf("max_udp_relay_packet_size should accept %d", valid)
				}
			}
		}
		// 65246-65507 is the panel's silent-clamp range; 65508+ is a hard
		// save error; 0 and negatives never meant anything.
		for _, invalid := range []int64{0, -1, 65246, 65507, 65508, 100000} {
			for _, v := range attr.Validators {
				if !testInt64Validator(t, v, invalid) {
					t.Errorf("max_udp_relay_packet_size should reject %d", invalid)
				}
			}
		}
	})
}
