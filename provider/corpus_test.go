package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// normalizeViaJSON normalizes a value by marshalling to JSON and back.
// This resolves type mismatches like []int vs []any{float64} and
// map[string]string vs map[string]any that arise from Go type differences
// in the flatten/build layers.
func normalizeViaJSON(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("normalizeViaJSON marshal: %v", err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("normalizeViaJSON unmarshal: %v", err)
	}
	return out
}

// testdataDir returns the absolute path to provider/testdata/.
func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata")
}

// loadFixture reads a JSON fixture file from testdata/ and returns its raw bytes.
func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(testdataDir(t), name))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return data
}

// ---------------------------------------------------------------------------
// Settings round-trip: JSON string -> flattenSettings -> buildSettingsJSON -> JSON
//
// The round-trip strips fields that the provider does not manage (clients,
// testseed, etc.) so we compare the flattened intermediate maps rather than
// raw JSON strings.
// ---------------------------------------------------------------------------

func TestCorpusSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		fixture  string
		desc     string
		protocol string // passed to flattenSettings/buildSettingsJSON; "wireguard" retains clients[]
	}{
		{"settings_vless.json", "VLESS with fallbacks and testseed", ""},
		{"settings_trojan.json", "Trojan with empty fallbacks", ""},
		{"settings_shadowsocks.json", "Shadowsocks with method/password/network", ""},
		{"settings_http.json", "HTTP with multiple accounts", ""},
		{"settings_socks.json", "SOCKS with auth and UDP", ""},
		{"settings_mixed.json", "Mixed with auth, accounts, UDP and IP", ""},
		{"settings_wireguard.json", "WireGuard with peers, gateway, dns, mtu, multi-client", "wireguard"},
		{"settings_amneziawg.json", "AmneziaWG with nested server block and a client", "amneziawg"},
		{"settings_dokodemo.json", "Dokodemo-door with portMap and followRedirect", ""},
		{"settings_hysteria.json", "Hysteria v2 with version field", ""},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			raw := string(loadFixture(t, tc.fixture))

			// Step 1: flatten (API JSON -> untyped map)
			flattened, err := flattenSettings(raw, tc.protocol)
			if err != nil {
				t.Fatalf("flattenSettings failed: %v", err)
			}
			if len(flattened) == 0 {
				t.Fatal("flattenSettings returned empty result")
			}
			firstFlat := flattened[0].(map[string]any)

			// Step 2: build (untyped map -> API JSON)
			rebuilt := buildSettingsJSON(firstFlat, tc.protocol)

			// Step 3: flatten again and compare
			reflattened, err := flattenSettings(rebuilt, tc.protocol)
			if err != nil {
				t.Fatalf("second flattenSettings failed: %v", err)
			}
			if len(reflattened) == 0 {
				t.Fatal("second flattenSettings returned empty result")
			}
			secondFlat := reflattened[0].(map[string]any)

			// Normalize via JSON to resolve Go type differences (e.g.
			// []int vs []any{float64}, map[string]string vs map[string]any).
			norm1 := normalizeViaJSON(t, firstFlat)
			norm2 := normalizeViaJSON(t, secondFlat)
			if !reflect.DeepEqual(norm1, norm2) {
				t.Fatalf("round-trip mismatch:\n  first:  %v\n  second: %v", norm1, norm2)
			}
		})
	}
}

// TestCorpusSettings_ClientsStripped verifies that flattenSettings drops the
// "clients" key for non-WireGuard protocols (clients are managed via
// threexui_inbound_client, not inbound settings).
func TestCorpusSettings_ClientsStripped(t *testing.T) {
	cases := []struct {
		fixture string
		desc    string
	}{
		{"settings_vless.json", "VLESS clients stripped"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			raw := string(loadFixture(t, tc.fixture))

			flattened, err := flattenSettings(raw, "")
			if err != nil {
				t.Fatalf("flattenSettings failed: %v", err)
			}
			if len(flattened) == 0 {
				t.Fatal("flattenSettings returned empty result")
			}
			firstFlat := flattened[0].(map[string]any)
			if _, ok := firstFlat["clients"]; ok {
				t.Fatal("clients should be stripped by flattenSettings for non-WireGuard protocols")
			}
		})
	}
}

// TestCorpusSettings_WireguardClientsRetained verifies the WireGuard
// multi-client surface (3x-ui v3.4.2+): unlike vmess/vless/trojan/SS/hysteria,
// WireGuard peers live in `settings.clients[]` and are managed via
// threexui_inbound itself — so flattenSettings/buildSettingsJSON MUST forward
// the clients array when protocol is "wireguard". Regression test for #342.
func TestCorpusSettings_WireguardClientsRetained(t *testing.T) {
	raw := string(loadFixture(t, "settings_wireguard.json"))

	// flatten (API JSON -> untyped map): clients[] must survive.
	flattened, err := flattenSettings(raw, "wireguard")
	if err != nil {
		t.Fatalf("flattenSettings failed: %v", err)
	}
	firstFlat := flattened[0].(map[string]any)
	clients, ok := firstFlat["clients"].([]any)
	if !ok || len(clients) != 1 {
		t.Fatalf("WireGuard clients[] must be retained on flatten (protocol=wireguard), got %#v", firstFlat["clients"])
	}

	// build (untyped map -> API JSON): clients[] must be written to the wire.
	rebuilt := buildSettingsJSON(firstFlat, "wireguard")
	reflattened, err := flattenSettings(rebuilt, "wireguard")
	if err != nil {
		t.Fatalf("second flattenSettings failed: %v", err)
	}
	secondFlat := reflattened[0].(map[string]any)
	if _, ok := secondFlat["clients"].([]any); !ok {
		t.Fatal("WireGuard clients[] must survive the build->flatten round-trip (protocol=wireguard)")
	}
}

// ---------------------------------------------------------------------------
// Stream settings round-trip
// ---------------------------------------------------------------------------

func TestCorpusStreamSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		fixture string
		desc    string
	}{
		{"stream_settings_reality_ws.json", "Reality + WebSocket with headers"},
		{"stream_settings_tcp_none.json", "TCP with none security"},
		{"stream_settings_grpc.json", "gRPC with reality"},
		{"stream_settings_kcp.json", "KCP with cwndMultiplier/maxSendingWindow (2.9.0)"},
		{"stream_settings_httpupgrade.json", "HTTP Upgrade transport"},
		{"stream_settings_xhttp.json", "XHTTP transport with keepAliveInterval"},
		{"stream_settings_xhttp_xpadding.json", "XHTTP transport with xPadding fields"},
		{"stream_settings_sockopt.json", "WebSocket with full sockopt"},
		{"stream_settings_hysteria.json", "Hysteria transport with auth and version"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			raw := string(loadFixture(t, tc.fixture))

			flattened, err := flattenStreamSettings(raw)
			if err != nil {
				t.Fatalf("flattenStreamSettings failed: %v", err)
			}
			if len(flattened) == 0 {
				t.Fatal("flattenStreamSettings returned empty result")
			}
			firstFlat := flattened[0].(map[string]any)

			rebuilt := buildStreamSettingsJSON(firstFlat)

			reflattened, err := flattenStreamSettings(rebuilt)
			if err != nil {
				t.Fatalf("second flattenStreamSettings failed: %v", err)
			}
			if len(reflattened) == 0 {
				t.Fatal("second flattenStreamSettings returned empty result")
			}
			secondFlat := reflattened[0].(map[string]any)

			if !reflect.DeepEqual(firstFlat, secondFlat) {
				t.Fatalf("round-trip mismatch:\n  first:  %v\n  second: %v", firstFlat, secondFlat)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Sniffing round-trip
// ---------------------------------------------------------------------------

func TestCorpusSniffing_RoundTrip(t *testing.T) {
	cases := []struct {
		fixture string
		desc    string
	}{
		{"sniffing_full.json", "Full sniffing with exclusions (2.9.0)"},
		{"sniffing_minimal.json", "Minimal sniffing without exclusions"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			raw := string(loadFixture(t, tc.fixture))

			flattened, err := flattenSniffing(raw)
			if err != nil {
				t.Fatalf("flattenSniffing failed: %v", err)
			}
			if len(flattened) == 0 {
				t.Fatal("flattenSniffing returned empty result")
			}
			firstFlat := flattened[0].(map[string]any)

			rebuilt := buildSniffingJSON(firstFlat)

			reflattened, err := flattenSniffing(rebuilt)
			if err != nil {
				t.Fatalf("second flattenSniffing failed: %v", err)
			}
			if len(reflattened) == 0 {
				t.Fatal("second flattenSniffing returned empty result")
			}
			secondFlat := reflattened[0].(map[string]any)

			if !reflect.DeepEqual(firstFlat, secondFlat) {
				t.Fatalf("round-trip mismatch:\n  first:  %v\n  second: %v", firstFlat, secondFlat)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Xray template: section-level round-trips via build/flatten*ToMap
// ---------------------------------------------------------------------------

func TestCorpusXrayTemplate_Basics(t *testing.T) {
	raw := loadFixture(t, "xray_template.json")
	var full map[string]any
	if err := json.Unmarshal(raw, &full); err != nil {
		t.Fatalf("unmarshal xray_template: %v", err)
	}

	// Extract basics-level fields
	basics := map[string]any{}
	for _, key := range []string{"log", "api", "policy", "stats"} {
		if v, ok := full[key]; ok {
			basics[key] = v
		}
	}

	flattened := flattenXrayBasicsToMap(basics)
	rebuilt := buildXrayBasicsJSON(flattened)
	// Normalize via JSON round-trip (expandStringList returns []string,
	// but after JSON marshal/unmarshal it becomes []any again)
	builtJSON, err := json.Marshal(rebuilt)
	if err != nil {
		t.Fatalf("marshal rebuilt basics: %v", err)
	}
	var builtMap map[string]any
	if err := json.Unmarshal(builtJSON, &builtMap); err != nil {
		t.Fatalf("unmarshal rebuilt basics: %v", err)
	}
	reflattened := flattenXrayBasicsToMap(builtMap)

	if !reflect.DeepEqual(flattened, reflattened) {
		t.Fatalf("xray basics round-trip mismatch:\n  first:  %v\n  second: %v", flattened, reflattened)
	}
}

func TestCorpusXrayTemplate_DNS(t *testing.T) {
	raw := loadFixture(t, "xray_template.json")
	var full map[string]any
	if err := json.Unmarshal(raw, &full); err != nil {
		t.Fatalf("unmarshal xray_template: %v", err)
	}

	dnsSection, ok := full["dns"].(map[string]any)
	if !ok {
		t.Fatal("missing dns section in xray_template")
	}

	flattened := flattenXrayDNSToMap(dnsSection)
	rebuilt := buildXrayDNSJSON(flattened)
	// buildXrayDNSJSON returns the dns value; re-parse via JSON for type normalization
	builtJSON, err := json.Marshal(rebuilt)
	if err != nil {
		t.Fatalf("marshal rebuilt dns: %v", err)
	}
	var builtMap map[string]any
	if err := json.Unmarshal(builtJSON, &builtMap); err != nil {
		t.Fatalf("unmarshal rebuilt dns: %v", err)
	}
	reflattened := flattenXrayDNSToMap(builtMap)

	if !reflect.DeepEqual(flattened, reflattened) {
		t.Fatalf("xray dns round-trip mismatch:\n  first:  %v\n  second: %v", flattened, reflattened)
	}
}

func TestCorpusXrayTemplate_Routing(t *testing.T) {
	raw := loadFixture(t, "xray_template.json")
	var full map[string]any
	if err := json.Unmarshal(raw, &full); err != nil {
		t.Fatalf("unmarshal xray_template: %v", err)
	}

	routingSection, ok := full["routing"].(map[string]any)
	if !ok {
		t.Fatal("missing routing section in xray_template")
	}

	flattened := flattenXrayRoutingToMap(routingSection)
	rebuilt := buildXrayRoutingJSON(flattened)
	builtJSON, err := json.Marshal(rebuilt)
	if err != nil {
		t.Fatalf("marshal rebuilt routing: %v", err)
	}
	var builtMap map[string]any
	if err := json.Unmarshal(builtJSON, &builtMap); err != nil {
		t.Fatalf("unmarshal rebuilt routing: %v", err)
	}
	reflattened := flattenXrayRoutingToMap(builtMap)

	if !reflect.DeepEqual(flattened, reflattened) {
		t.Fatalf("xray routing round-trip mismatch:\n  first:  %v\n  second: %v", flattened, reflattened)
	}
}

func TestCorpusXrayTemplate_Outbounds(t *testing.T) {
	raw := loadFixture(t, "xray_template.json")
	var full map[string]any
	if err := json.Unmarshal(raw, &full); err != nil {
		t.Fatalf("unmarshal xray_template: %v", err)
	}

	outbounds, ok := full["outbounds"].([]any)
	if !ok {
		t.Fatal("missing outbounds section in xray_template")
	}

	flattened := flattenXrayOutboundsToMap(outbounds)
	rebuilt := buildXrayOutboundsJSON(flattened)
	builtJSON, err := json.Marshal(rebuilt)
	if err != nil {
		t.Fatalf("marshal rebuilt outbounds: %v", err)
	}
	// The built result is an array
	var builtList []any
	if err := json.Unmarshal(builtJSON, &builtList); err != nil {
		t.Fatalf("unmarshal rebuilt outbounds: %v", err)
	}
	reflattened := flattenXrayOutboundsToMap(builtList)

	if !reflect.DeepEqual(flattened, reflattened) {
		t.Fatalf("xray outbounds round-trip mismatch:\n  first:  %v\n  second: %v", flattened, reflattened)
	}
}

func TestCorpusXrayTemplate_Reverse(t *testing.T) {
	raw := loadFixture(t, "xray_template.json")
	var full map[string]any
	if err := json.Unmarshal(raw, &full); err != nil {
		t.Fatalf("unmarshal xray_template: %v", err)
	}

	reverseSection, ok := full["reverse"].(map[string]any)
	if !ok {
		t.Fatal("missing reverse section in xray_template")
	}

	flattened := flattenXrayReverseToMap(reverseSection)
	rebuilt := buildXrayReverseJSON(flattened)
	builtJSON, err := json.Marshal(rebuilt)
	if err != nil {
		t.Fatalf("marshal rebuilt reverse: %v", err)
	}
	var builtMap map[string]any
	if err := json.Unmarshal(builtJSON, &builtMap); err != nil {
		t.Fatalf("unmarshal rebuilt reverse: %v", err)
	}
	reflattened := flattenXrayReverseToMap(builtMap)

	if !reflect.DeepEqual(flattened, reflattened) {
		t.Fatalf("xray reverse round-trip mismatch:\n  first:  %v\n  second: %v", flattened, reflattened)
	}
}

func TestCorpusXrayTemplate_Balancers(t *testing.T) {
	raw := loadFixture(t, "xray_template.json")
	var full map[string]any
	if err := json.Unmarshal(raw, &full); err != nil {
		t.Fatalf("unmarshal xray_template: %v", err)
	}

	balancersSection, ok := full["balancers"].([]any)
	if !ok {
		t.Fatal("missing balancers section in xray_template")
	}

	flattened := flattenXrayBalancersToMap(balancersSection)
	rebuilt := buildXrayBalancersJSON(flattened)
	builtJSON, err := json.Marshal(rebuilt)
	if err != nil {
		t.Fatalf("marshal rebuilt balancers: %v", err)
	}
	var builtList []any
	if err := json.Unmarshal(builtJSON, &builtList); err != nil {
		t.Fatalf("unmarshal rebuilt balancers: %v", err)
	}
	reflattened := flattenXrayBalancersToMap(builtList)

	if !reflect.DeepEqual(flattened, reflattened) {
		t.Fatalf("xray balancers round-trip mismatch:\n  first:  %v\n  second: %v", flattened, reflattened)
	}
}

// ---------------------------------------------------------------------------
// Malformed input handling
// ---------------------------------------------------------------------------

func TestCorpusMalformed_TruncatedJSON(t *testing.T) {
	raw := string(loadFixture(t, "malformed_truncated.txt"))

	_, err := flattenSettings(raw, "")
	if err == nil {
		t.Fatal("expected error for truncated JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse settings JSON") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = flattenStreamSettings(raw)
	if err == nil {
		t.Fatal("expected error for truncated JSON in stream settings")
	}
	if !strings.Contains(err.Error(), "failed to parse stream_settings JSON") {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = flattenSniffing(raw)
	if err == nil {
		t.Fatal("expected error for truncated JSON in sniffing")
	}
	if !strings.Contains(err.Error(), "failed to parse sniffing JSON") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCorpusMalformed_WrongTypes(t *testing.T) {
	raw := string(loadFixture(t, "malformed_wrong_types.json"))

	// flattenSettings should not error but should skip mistyped fields
	flattened, err := flattenSettings(raw, "")
	if err != nil {
		t.Fatalf("flattenSettings should handle wrong types gracefully: %v", err)
	}
	if len(flattened) == 0 {
		// All fields were skipped due to wrong types -> empty is fine
		return
	}
	m := flattened[0].(map[string]any)
	// "decryption" is 12345 (float64 after JSON unmarshal), not string -> should be skipped
	if _, ok := m["decryption"]; ok {
		t.Fatal("numeric decryption should be skipped")
	}
	// "fallbacks" is string -> should be skipped
	if _, ok := m["fallbacks"]; ok {
		t.Fatal("string fallbacks should be skipped")
	}
}

func TestCorpusMalformed_NullFields(t *testing.T) {
	raw := string(loadFixture(t, "malformed_null_fields.json"))

	flattened, err := flattenSettings(raw, "")
	if err != nil {
		t.Fatalf("flattenSettings should handle null fields: %v", err)
	}
	// All null fields should be ignored -> empty result
	if len(flattened) != 0 {
		if m, ok := flattened[0].(map[string]any); ok && len(m) > 0 {
			t.Fatalf("null fields should not produce output, got: %v", m)
		}
	}
}

func TestCorpusMalformed_EmptyObject(t *testing.T) {
	raw := string(loadFixture(t, "malformed_empty_object.json"))

	flattened, err := flattenSettings(raw, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flattened) != 0 {
		t.Fatalf("empty object should produce empty result, got: %v", flattened)
	}

	flatStream, err := flattenStreamSettings(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flatStream) != 0 {
		t.Fatalf("empty object should produce empty stream result, got: %v", flatStream)
	}

	flatSniff, err := flattenSniffing(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flatSniff) != 0 {
		t.Fatalf("empty object should produce empty sniffing result, got: %v", flatSniff)
	}
}

func TestCorpusMalformed_ExtraFields(t *testing.T) {
	raw := string(loadFixture(t, "malformed_extra_fields.json"))

	// Extra fields should be silently ignored, known fields preserved
	flattened, err := flattenSettings(raw, "")
	if err != nil {
		t.Fatalf("flattenSettings should ignore extra fields: %v", err)
	}
	if len(flattened) == 0 {
		t.Fatal("expected at least decryption and encryption")
	}
	m := flattened[0].(map[string]any)
	if m["decryption"] != "none" {
		t.Fatalf("expected decryption=none, got %v", m["decryption"])
	}
	if m["encryption"] != "none" {
		t.Fatalf("expected encryption=none, got %v", m["encryption"])
	}
	// Unknown fields should not appear in output
	if _, ok := m["unknownField1"]; ok {
		t.Fatal("unknownField1 should not be in flattened output")
	}
	if _, ok := m["nested_unknown"]; ok {
		t.Fatal("nested_unknown should not be in flattened output")
	}

	// Round-trip should still work for the known fields
	rebuilt := buildSettingsJSON(m, "")
	reflattened, err := flattenSettings(rebuilt, "")
	if err != nil {
		t.Fatalf("round-trip failed: %v", err)
	}
	secondFlat := reflattened[0].(map[string]any)
	if !reflect.DeepEqual(m, secondFlat) {
		t.Fatalf("round-trip mismatch after stripping extra fields:\n  first:  %v\n  second: %v", m, secondFlat)
	}
}

// ---------------------------------------------------------------------------
// Panel settings round-trip
//
// Panel settings use a different pipeline from inbound settings: typed
// attributes are directly expanded/flattened via expand*/flatten* functions
// that work with map[string]any (the API JSON body), not via the
// buildSettingsJSON/flattenSettings pair.  We test the expand/flatten cycle
// here to confirm no data is lost.
// ---------------------------------------------------------------------------

func TestCorpusPanelSettings_SecurityRoundTrip(t *testing.T) {
	input := map[string]any{
		"twoFactorEnable": true,
		"twoFactorToken":  "JBSWY3DPEHPK3PXP",
	}
	model := flattenPanelSecurity(input)
	rebuilt := expandPanelSecurity(model)

	if rebuilt["twoFactorEnable"] != true {
		t.Fatalf("expected twoFactorEnable=true, got %v", rebuilt["twoFactorEnable"])
	}
	if rebuilt["twoFactorToken"] != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("expected twoFactorToken, got %v", rebuilt["twoFactorToken"])
	}
}

func TestCorpusPanelSettings_TelegramRoundTrip(t *testing.T) {
	input := map[string]any{
		"tgBotEnable":      true,
		"tgBotToken":       "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
		"tgBotProxy":       "socks5://127.0.0.1:1080",
		"tgBotChatId":      "12345678",
		"tgRunTime":        "@daily",
		"tgBotBackup":      true,
		"tgBotLoginNotify": true,
		"tgCpu":            float64(80),
	}
	model := flattenPanelTelegram(input)
	rebuilt := expandPanelTelegram(model)

	if rebuilt["tgBotEnable"] != true {
		t.Fatalf("expected tgBotEnable=true, got %v", rebuilt["tgBotEnable"])
	}
	if rebuilt["tgBotToken"] != "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11" {
		t.Fatalf("expected tgBotToken, got %v", rebuilt["tgBotToken"])
	}
}

// ---------------------------------------------------------------------------
// InboundToModel: full inbound payloads through diagnostics path
// ---------------------------------------------------------------------------

func TestCorpusInboundToModel_VLESSFull(t *testing.T) {
	settings := string(loadFixture(t, "settings_vless.json"))
	stream := string(loadFixture(t, "stream_settings_reality_ws.json"))
	sniffing := string(loadFixture(t, "sniffing_full.json"))

	inbound := &Inbound{
		ID:             1,
		Up:             1024,
		Down:           2048,
		Total:          0,
		Remark:         "vless-reality-ws",
		Enable:         true,
		ExpiryTime:     0,
		Listen:         "",
		Port:           443,
		Protocol:       "vless",
		Settings:       settings,
		StreamSettings: stream,
		Sniffing:       sniffing,
		Tag:            "inbound-443",
	}

	model, diags := inboundToModel(inbound, true)
	if diags.HasError() {
		t.Fatalf("inboundToModel diagnostics: %v", diags)
	}
	if model == nil {
		t.Fatal("expected non-nil model")
		return
	}
	if model.Protocol.ValueString() != "vless" {
		t.Fatalf("expected protocol vless, got %s", model.Protocol.ValueString())
	}
	if model.Port.ValueInt64() != 443 {
		t.Fatalf("expected port 443, got %d", model.Port.ValueInt64())
	}
	if model.Remark.ValueString() != "vless-reality-ws" {
		t.Fatalf("expected remark vless-reality-ws, got %s", model.Remark.ValueString())
	}
	// IMPORTANT-7: verify typed blocks are populated, not nil
	if model.VlessSettings == nil {
		t.Fatal("expected VlessSettings to be populated")
	}
	if model.StreamSettings == nil {
		t.Fatal("expected StreamSettings to be populated")
	}
	if model.Sniffing == nil {
		t.Fatal("expected Sniffing to be populated")
	}
}

func TestCorpusInboundToModel_DokodemoFull(t *testing.T) {
	settings := string(loadFixture(t, "settings_dokodemo.json"))
	stream := string(loadFixture(t, "stream_settings_tcp_none.json"))
	sniffing := string(loadFixture(t, "sniffing_minimal.json"))

	inbound := &Inbound{
		ID:             2,
		Remark:         "dokodemo-transparent",
		Enable:         true,
		Port:           12345,
		Protocol:       "dokodemo-door",
		Settings:       settings,
		StreamSettings: stream,
		Sniffing:       sniffing,
	}

	model, diags := inboundToModel(inbound, true)
	if diags.HasError() {
		t.Fatalf("inboundToModel diagnostics: %v", diags)
	}
	if model == nil {
		t.Fatal("expected non-nil model")
		return
	}
	if model.Protocol.ValueString() != "dokodemo-door" {
		t.Fatalf("expected protocol dokodemo-door, got %s", model.Protocol.ValueString())
	}
	if model.DokodemoSettings == nil {
		t.Fatal("expected DokodemoSettings to be populated")
	}
	if model.StreamSettings == nil {
		t.Fatal("expected StreamSettings to be populated")
	}
	if model.Sniffing == nil {
		t.Fatal("expected Sniffing to be populated")
	}
}

func TestCorpusInboundToModel_ShadowsocksFull(t *testing.T) {
	settings := string(loadFixture(t, "settings_shadowsocks.json"))
	stream := string(loadFixture(t, "stream_settings_tcp_none.json"))

	inbound := &Inbound{
		ID:             3,
		Remark:         "ss-inbound",
		Enable:         true,
		Port:           8388,
		Protocol:       "shadowsocks",
		Settings:       settings,
		StreamSettings: stream,
	}

	model, diags := inboundToModel(inbound, true)
	if diags.HasError() {
		t.Fatalf("inboundToModel diagnostics: %v", diags)
	}
	if model == nil {
		t.Fatal("expected non-nil model")
		return
	}
	if model.ShadowsocksSettings == nil {
		t.Fatal("expected ShadowsocksSettings to be populated")
	}
	if model.StreamSettings == nil {
		t.Fatal("expected StreamSettings to be populated")
	}
}
