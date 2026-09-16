package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The REALITY client-version gate fields (minClientVer/maxClientVer/maxTimediff)
// must survive the typed-model <-> untyped-map conversion. The map <-> JSON layer
// is covered by stream_settings_test.go.

func TestExpandRealitySettingsFromModel_ClientVerFields(t *testing.T) {
	m := &InboundRealitySettingsModel{
		Target:       types.StringValue("example.com:443"),
		MinClientVer: types.StringValue("26.3.27"),
		MaxClientVer: types.StringValue("26.9.9"),
		MaxTimediff:  types.Int64Value(60000),
	}
	out := expandRealitySettingsFromModel(m)
	if out["min_client_ver"] != "26.3.27" {
		t.Fatalf("min_client_ver: %v", out["min_client_ver"])
	}
	if out["max_client_ver"] != "26.9.9" {
		t.Fatalf("max_client_ver: %v", out["max_client_ver"])
	}
	if out["max_timediff"] != int64(60000) {
		t.Fatalf("max_timediff: %T %v", out["max_timediff"], out["max_timediff"])
	}
}

func TestFlattenRealitySettingsToModel_ClientVerFields(t *testing.T) {
	data := map[string]any{
		"min_client_ver": "26.3.27",
		"max_client_ver": "26.9.9",
		"max_timediff":   60000,
	}
	m := flattenRealitySettingsToModel(data)
	if m.MinClientVer.ValueString() != "26.3.27" {
		t.Fatalf("MinClientVer: %v", m.MinClientVer)
	}
	if m.MaxClientVer.ValueString() != "26.9.9" {
		t.Fatalf("MaxClientVer: %v", m.MaxClientVer)
	}
	if m.MaxTimediff.ValueInt64() != 60000 {
		t.Fatalf("MaxTimediff: %v", m.MaxTimediff)
	}
}

// Absent gate fields must flatten to null so the Optional+Computed attributes do
// not force a spurious diff against a panel that omits them.
func TestFlattenRealitySettingsToModel_ClientVerAbsent(t *testing.T) {
	m := flattenRealitySettingsToModel(map[string]any{})
	if !m.MinClientVer.IsNull() || !m.MaxClientVer.IsNull() || !m.MaxTimediff.IsNull() {
		t.Fatalf("expected null client-ver fields when absent, got %v/%v/%v",
			m.MinClientVer, m.MaxClientVer, m.MaxTimediff)
	}
}

// "0.0.0" is the documented way to remove the REALITY lower bound: Xray only
// skips the gate for a literal zero version, and replaces an empty minClientVer
// with its own default. The value therefore has to survive both conversion
// layers unchanged — typed model -> snake_case map -> panel JSON and back.
func TestRealitySettings_ZeroVersionRoundTrip(t *testing.T) {
	in := &InboundRealitySettingsModel{
		Target:       types.StringValue("example.com:443"),
		MinClientVer: types.StringValue("0.0.0"),
		MaxClientVer: types.StringValue("255.255.255"),
		MaxTimediff:  types.Int64Value(0),
	}

	wire := expandRealitySettings([]any{expandRealitySettingsFromModel(in)})
	if wire["minClientVer"] != "0.0.0" {
		t.Fatalf("minClientVer lost on the wire: %v", wire["minClientVer"])
	}

	if got, ok := wire["maxTimediff"].(int64); !ok || got != 0 {
		t.Fatalf("maxTimediff should stay a concrete int64 zero, got %T %v",
			wire["maxTimediff"], wire["maxTimediff"])
	}

	out := flattenRealitySettingsToModel(flattenRealitySettings(wire))
	if got := out.MinClientVer.ValueString(); got != "0.0.0" {
		t.Fatalf("MinClientVer: want 0.0.0, got %q", got)
	}
	if got := out.MaxClientVer.ValueString(); got != "255.255.255" {
		t.Fatalf("MaxClientVer: want 255.255.255, got %q", got)
	}
	// max_timediff = 0 disables the time check; it must stay a concrete zero
	// rather than collapsing to null, which would break the Optional+Computed
	// contract on the next plan.
	if out.MaxTimediff.IsNull() || out.MaxTimediff.ValueInt64() != 0 {
		t.Fatalf("MaxTimediff: want 0, got %v", out.MaxTimediff)
	}
}

// xray-core spells the field maxTimeDiff, the panel spells it maxTimediff, and
// 3x-ui keeps streamSettings as opaque JSON text — so an inbound authored
// outside the panel can carry either. Reading only one spelling would import
// the other as null and then drop it on the next update, quietly turning the
// time-difference gate off.
func TestFlattenRealitySettings_MaxTimediffAlias(t *testing.T) {
	canonical := flattenRealitySettings(map[string]any{"maxTimeDiff": float64(45000)})
	if got := canonical["max_timediff"]; got != int64(45000) {
		t.Fatalf("canonical maxTimeDiff not read: %v", got)
	}

	panelSpelling := flattenRealitySettings(map[string]any{"maxTimediff": float64(45000)})
	if got := panelSpelling["max_timediff"]; got != int64(45000) {
		t.Fatalf("panel maxTimediff not read: %v", got)
	}

	// Both present: precedence is fixed so a plan never depends on map order.
	both := flattenRealitySettings(map[string]any{
		"maxTimeDiff": float64(1000),
		"maxTimediff": float64(2000),
	})
	if got := both["max_timediff"]; got != int64(1000) {
		t.Fatalf("canonical spelling should win, got %v", got)
	}

	if _, ok := flattenRealitySettings(map[string]any{})["max_timediff"]; ok {
		t.Fatalf("absent field must not materialise a zero")
	}
}

// An inbound imported with the canonical spelling has to reach the typed model
// with its value intact — this is the path that decides whether the next
// unrelated update preserves or drops the gate.
func TestFlattenRealitySettingsToModel_MaxTimediffAliasImport(t *testing.T) {
	m := flattenRealitySettingsToModel(
		flattenRealitySettings(map[string]any{
			"target":      "example.com:443",
			"maxTimeDiff": float64(45000),
		}),
	)
	if m.MaxTimediff.IsNull() || m.MaxTimediff.ValueInt64() != 45000 {
		t.Fatalf("MaxTimediff after import: want 45000, got %v", m.MaxTimediff)
	}

	// ...and survives being written back out under the panel spelling.
	wire := expandRealitySettings([]any{expandRealitySettingsFromModel(m)})
	if got := wire["maxTimediff"]; got != int64(45000) {
		t.Fatalf("maxTimediff dropped on re-serialise: %v", got)
	}
}

// `int` is 32 bits on the 386 and arm release targets, so the conversion path
// must not narrow: a value the schema accepts would otherwise wrap to a
// negative maxTimediff on those builds.
func TestRealityMaxTimediff_NoPlatformNarrowing(t *testing.T) {
	const beyond32Bit = int64(4_294_967_296) // 2^32

	wire := expandRealitySettings([]any{expandRealitySettingsFromModel(
		&InboundRealitySettingsModel{MaxTimediff: types.Int64Value(beyond32Bit)},
	)})
	if got := wire["maxTimediff"]; got != beyond32Bit {
		t.Fatalf("maxTimediff narrowed on write: %v", got)
	}

	m := flattenRealitySettingsToModel(flattenRealitySettings(wire))
	if got := m.MaxTimediff.ValueInt64(); got != beyond32Bit {
		t.Fatalf("maxTimediff narrowed on read: want %d, got %d", beyond32Bit, got)
	}
}

// --- TLS ---

func TestExpandTLSSettingsFromModel_Full(t *testing.T) {
	alpn, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("h2"),
		types.StringValue("http/1.1"),
	})
	m := &InboundTLSSettingsModel{
		ServerName:    types.StringValue("gcp.example.com"),
		Fingerprint:   types.StringValue("chrome"),
		AllowInsecure: types.BoolValue(false),
		Alpn:          alpn,
		MinVersion:    types.StringValue("1.2"),
		MaxVersion:    types.StringValue("1.3"),
		Cipher:        types.StringValue("AES128-GCM-SHA256"),
	}
	out := expandTLSSettingsFromModel(m)
	if out["server_name"] != "gcp.example.com" {
		t.Fatalf("server_name: %v", out["server_name"])
	}
	if out["fingerprint"] != "chrome" {
		t.Fatalf("fingerprint: %v", out["fingerprint"])
	}
	if out["allow_insecure"] != false {
		t.Fatalf("allow_insecure: %v", out["allow_insecure"])
	}
	got, ok := out["alpn"].([]any)
	if !ok || len(got) != 2 || got[0] != "h2" || got[1] != "http/1.1" {
		t.Fatalf("alpn: %#v", out["alpn"])
	}
}

func TestExpandTLSSettingsFromModel_Empty(t *testing.T) {
	if out := expandTLSSettingsFromModel(nil); out != nil {
		t.Fatalf("expected nil for nil model, got %v", out)
	}
	if out := expandTLSSettingsFromModel(&InboundTLSSettingsModel{}); len(out) != 0 {
		t.Fatalf("expected empty map for null model, got %v", out)
	}
}

func TestFlattenTLSSettingsToModel_Full(t *testing.T) {
	m := flattenTLSSettingsToModel(map[string]any{
		"server_name":    "gcp.example.com",
		"fingerprint":    "chrome",
		"allow_insecure": false,
		"alpn":           []any{"h2", "http/1.1"},
		"min_version":    "1.2",
		"max_version":    "1.3",
		"cipher":         "AES128-GCM-SHA256",
	})
	if m.ServerName.ValueString() != "gcp.example.com" {
		t.Fatalf("ServerName: %v", m.ServerName)
	}
	if m.AllowInsecure.IsNull() || m.AllowInsecure.ValueBool() {
		t.Fatalf("AllowInsecure: %v", m.AllowInsecure)
	}
	if m.Alpn.IsNull() || len(m.Alpn.Elements()) != 2 {
		t.Fatalf("Alpn: %v", m.Alpn)
	}
	if m.MinVersion.ValueString() != "1.2" || m.MaxVersion.ValueString() != "1.3" {
		t.Fatalf("versions: %v / %v", m.MinVersion, m.MaxVersion)
	}
}

func TestFlattenTLSSettingsToModel_Absent(t *testing.T) {
	m := flattenTLSSettingsToModel(map[string]any{})
	if !m.ServerName.IsNull() || !m.AllowInsecure.IsNull() || !m.Alpn.IsNull() {
		t.Fatalf("expected all-null on absent, got %#v", m)
	}
}

// Inbound stream_settings model round-trip: a tls_settings block declared on an
// inbound must survive model -> snake_case map -> snake_case map -> model.
func TestInboundStreamSettings_TLSModelRoundTrip(t *testing.T) {
	alpn, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("h2"),
		types.StringValue("http/1.1"),
	})
	in := &InboundStreamSettingsModel{
		Network:  types.StringValue("tcp"),
		Security: types.StringValue("tls"),
		TLSSettings: &InboundTLSSettingsModel{
			ServerName:    types.StringValue("gcp.example.com"),
			Fingerprint:   types.StringValue("chrome"),
			AllowInsecure: types.BoolValue(false),
			Alpn:          alpn,
		},
	}
	wire := expandStreamSettingsFromModel(in)
	back := flattenStreamSettingsToModel(wire)
	if back.TLSSettings == nil {
		t.Fatal("expected TLSSettings to survive round-trip")
	}
	if back.TLSSettings.ServerName.ValueString() != "gcp.example.com" {
		t.Fatalf("ServerName after round-trip: %v", back.TLSSettings.ServerName)
	}
	if back.TLSSettings.Fingerprint.ValueString() != "chrome" {
		t.Fatalf("Fingerprint after round-trip: %v", back.TLSSettings.Fingerprint)
	}
}

// --- tls_settings.certificates (3x-ui v3.8.0+ TLS validation) ---

func TestExpandTLSSettingsFromModel_Certificates(t *testing.T) {
	m := &InboundTLSSettingsModel{
		ServerName: types.StringValue("tls.example.com"),
		Certificates: []InboundTLSCertModel{
			{
				// File-backed server certificate.
				CertificateFile: types.StringValue("/etc/certs/server.pem"),
				KeyFile:         types.StringValue("/etc/certs/server.key"),
				OcspStapling:    types.Int64Value(3),
				OneTimeLoading:  types.BoolValue(true),
				BuildChain:      types.BoolValue(false),
				Usage:           types.StringValue("encipherment"),
			},
			{
				// Inline verification CA — public cert only, no key.
				Certificate: mustList(t, []string{"-----BEGIN CERTIFICATE-----", "MIIB...", "-----END CERTIFICATE-----"}),
				Usage:       types.StringValue("verify"),
			},
		},
	}
	out := expandTLSSettingsFromModel(m)
	certs, ok := out["certificates"].([]any)
	if !ok || len(certs) != 2 {
		t.Fatalf("certificates: %#v", out["certificates"])
	}
	first := certs[0].(map[string]any)
	if first["certificate_file"] != "/etc/certs/server.pem" || first["key_file"] != "/etc/certs/server.key" {
		t.Fatalf("file-backed entry: %#v", first)
	}
	if first["ocsp_stapling"] != 3 || first["one_time_loading"] != true {
		t.Fatalf("flags: %#v", first)
	}
	if first["usage"] != "encipherment" {
		t.Fatalf("usage: %#v", first)
	}
	if v, has := first["build_chain"]; !has || v != false {
		t.Fatalf("explicitly-false build_chain must still be written: %#v", first)
	}
	second := certs[1].(map[string]any)
	if _, has := second["key"]; has {
		t.Fatalf("null key list must be omitted: %#v", second)
	}
	if second["usage"] != "verify" {
		t.Fatalf("verify usage: %#v", second)
	}
}

// Absent (nil) and explicitly-empty certificates must be distinguishable:
// nil keeps prior entries via UseStateForUnknown, [] writes an empty array
// the panel can observe (and refuse on v3.8.0+ while security = "tls").
func TestExpandTLSSettingsFromModel_CertificatesEmptyVsAbsent(t *testing.T) {
	absent := expandTLSSettingsFromModel(&InboundTLSSettingsModel{})
	if _, has := absent["certificates"]; has {
		t.Fatalf("absent certificates must not be written: %#v", absent)
	}
	out := expandTLSSettingsFromModel(&InboundTLSSettingsModel{Certificates: []InboundTLSCertModel{}})
	certs, ok := out["certificates"].([]any)
	if !ok || len(certs) != 0 {
		t.Fatalf("explicit empty must round-trip as an empty array: %#v", out["certificates"])
	}
}

func TestFlattenTLSSettingsToModel_Certificates(t *testing.T) {
	m := flattenTLSSettingsToModel(map[string]any{
		"server_name": "tls.example.com",
		"certificates": []any{
			map[string]any{
				"certificate_file": "/etc/certs/server.pem",
				"key_file":         "/etc/certs/server.key",
				"ocsp_stapling":    float64(3),
				"one_time_loading": true,
				"usage":            "encipherment",
			},
			map[string]any{
				"certificate": []any{"-----BEGIN CERTIFICATE-----", "MIIB..."},
				"key":         []any{"-----BEGIN PRIVATE KEY-----", "MC4..."},
			},
		},
	})
	if len(m.Certificates) != 2 {
		t.Fatalf("certificates: %#v", m.Certificates)
	}
	if m.Certificates[0].CertificateFile.ValueString() != "/etc/certs/server.pem" {
		t.Fatalf("file entry: %#v", m.Certificates[0])
	}
	if m.Certificates[0].OcspStapling.ValueInt64() != 3 || !m.Certificates[0].OneTimeLoading.ValueBool() {
		t.Fatalf("flags: %#v", m.Certificates[0])
	}
	if len(m.Certificates[1].Certificate.Elements()) != 2 {
		t.Fatalf("inline certificate: %#v", m.Certificates[1].Certificate)
	}
	if len(m.Certificates[1].Key.Elements()) != 2 {
		t.Fatalf("inline key: %#v", m.Certificates[1].Key)
	}
	if m.Certificates[1].Usage.ValueString() != "" {
		t.Fatalf("absent usage must flatten null, got %#v", m.Certificates[1].Usage)
	}
}

func TestFlattenTLSSettingsToModel_CertificatesAbsent(t *testing.T) {
	m := flattenTLSSettingsToModel(map[string]any{"server_name": "tls.example.com"})
	if m.Certificates != nil {
		t.Fatalf("absent certificates must flatten to nil (null), got %#v", m.Certificates)
	}
}

// TestTLSCertificatesWireRoundTrip walks the full four-layer pipeline:
// typed model → untyped snake → camelCase tlsSettings JSON (what goes over
// the wire) → back. This is the path that previously DROPPED certificates
// entirely (the field was unmodeled), breaking TLS inbounds on 3x-ui v3.8.0+.
func TestTLSCertificatesWireRoundTrip(t *testing.T) {
	model := &InboundStreamSettingsModel{
		Network:  types.StringValue("tcp"),
		Security: types.StringValue("tls"),
		TLSSettings: &InboundTLSSettingsModel{
			ServerName: types.StringValue("tls.example.com"),
			Certificates: []InboundTLSCertModel{{
				CertificateFile: types.StringValue("/etc/certs/server.pem"),
				KeyFile:         types.StringValue("/etc/certs/server.key"),
				Usage:           types.StringValue("encipherment"),
			}},
		},
	}

	wire := buildStreamSettingsJSON(expandStreamSettingsFromModel(model))
	var payload map[string]any
	if err := json.Unmarshal([]byte(wire), &payload); err != nil {
		t.Fatalf("wire JSON invalid: %v\n%s", err, wire)
	}
	tls, ok := payload["tlsSettings"].(map[string]any)
	if !ok {
		t.Fatalf("tlsSettings missing from wire JSON: %s", wire)
	}
	certs, ok := tls["certificates"].([]any)
	if !ok || len(certs) != 1 {
		t.Fatalf("certificates missing from wire JSON: %s", wire)
	}
	entry := certs[0].(map[string]any)
	if entry["certificateFile"] != "/etc/certs/server.pem" || entry["keyFile"] != "/etc/certs/server.key" {
		t.Fatalf("wire entry: %#v", entry)
	}

	// And back: JSON → untyped snake → typed model.
	flat, err := flattenStreamSettings(wire)
	if err != nil {
		t.Fatalf("flattenStreamSettings: %v", err)
	}
	tlsFlat, ok := flat[0].(map[string]any)["tls_settings"].([]any)[0].(map[string]any)
	if !ok {
		t.Fatalf("tls_settings missing from flattened: %#v", flat)
	}
	back := flattenTLSSettingsToModel(tlsFlat)
	if len(back.Certificates) != 1 {
		t.Fatalf("round-trip lost certificates: %#v", back.Certificates)
	}
	if back.Certificates[0].CertificateFile.ValueString() != "/etc/certs/server.pem" {
		t.Fatalf("round-trip certificate_file: %#v", back.Certificates[0])
	}
	if back.ServerName.ValueString() != "tls.example.com" {
		t.Fatalf("round-trip server_name: %#v", back.ServerName)
	}
}

// mustList builds a types.List of strings, failing the test on error.
func mustList(t *testing.T, items []string) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(t.Context(), types.StringType, items)
	if diags.HasError() {
		t.Fatalf("ListValueFrom: %v", diags)
	}
	return list
}
