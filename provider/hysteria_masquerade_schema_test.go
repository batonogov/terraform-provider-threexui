package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func masqueradeStreamSettings(m *InboundHysteriaMasqueradeModel) *InboundStreamSettingsModel {
	return &InboundStreamSettingsModel{
		Network:          types.StringValue("hysteria"),
		Security:         types.StringValue("tls"),
		HysteriaSettings: &InboundHysteriaStreamSettingsModel{Version: types.Int64Value(2), Masquerade: m},
	}
}

// nullMasquerade fills every field the test leaves out with a typed null, as
// the framework and flatten produce them.
func nullMasquerade(m InboundHysteriaMasqueradeModel) *InboundHysteriaMasqueradeModel {
	for _, v := range m.strings() {
		if reflect.DeepEqual(*v, types.String{}) {
			*v = types.StringNull()
		}
	}
	for _, v := range m.bools() {
		if reflect.DeepEqual(*v, types.Bool{}) {
			*v = types.BoolNull()
		}
	}
	if reflect.DeepEqual(m.Headers, types.Map{}) {
		m.Headers = types.MapNull(types.StringType)
	}
	if reflect.DeepEqual(m.StatusCode, types.Int64{}) {
		m.StatusCode = types.Int64Null()
	}
	return &m
}

func masqueradeFromStreamJSON(t *testing.T, streamJSON string) *InboundHysteriaMasqueradeModel {
	t.Helper()
	flat, err := flattenStreamSettingsToMap(streamJSON)
	if err != nil {
		t.Fatalf("flattenStreamSettingsToMap(%s): %v", streamJSON, err)
	}
	hy := flattenStreamSettingsToModel(flat).HysteriaSettings
	if hy == nil {
		t.Fatalf("hysteria_settings lost on read-back of %s", streamJSON)
	}
	return hy.Masquerade
}

func assertMasqueradeEqual(t *testing.T, got, want *InboundHysteriaMasqueradeModel) {
	t.Helper()
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("masquerade = %+v, want %+v", got, want)
		}
		return
	}
	gv, wv := reflect.ValueOf(*got), reflect.ValueOf(*want)
	for i := 0; i < gv.NumField(); i++ {
		g := gv.Field(i).Interface().(attr.Value)
		w := wv.Field(i).Interface().(attr.Value)
		if !g.Equal(w) {
			t.Errorf("%s: got %s, want %s", gv.Type().Field(i).Tag.Get("tfsdk"), g, w)
		}
	}
}

func TestMasqueradeWireRoundTrip(t *testing.T) {
	headers := types.MapValueMust(types.StringType, map[string]attr.Value{"Server": types.StringValue("nginx")})
	cases := []struct {
		name string
		in   *InboundHysteriaMasqueradeModel
		wire map[string]any
	}{
		{
			name: "file",
			in:   nullMasquerade(InboundHysteriaMasqueradeModel{Type: types.StringValue("file"), Dir: types.StringValue("/var/www/site")}),
			wire: map[string]any{"type": "file", "dir": "/var/www/site"},
		},
		{
			name: "proxy",
			in: nullMasquerade(InboundHysteriaMasqueradeModel{
				Type: types.StringValue("proxy"), URL: types.StringValue("https://example.com"),
				RewriteHost: types.BoolValue(true), XForwarded: types.BoolValue(true), Insecure: types.BoolValue(true),
			}),
			wire: map[string]any{"type": "proxy", "url": "https://example.com", "rewriteHost": true, "xForwarded": true, "insecure": true},
		},
		{
			name: "string",
			in: nullMasquerade(InboundHysteriaMasqueradeModel{
				Type: types.StringValue("string"), Content: types.StringValue("hello"), Headers: headers, StatusCode: types.Int64Value(503),
			}),
			wire: map[string]any{"type": "string", "content": "hello", "headers": map[string]any{"Server": "nginx"}, "statusCode": float64(503)},
		},
		{
			// xray-core answers 404 for an empty object, and the block must
			// come back as a block, not as null.
			name: "empty block",
			in:   nullMasquerade(InboundHysteriaMasqueradeModel{}),
			wire: map[string]any{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			streamJSON := buildStreamSettingsJSON(expandStreamSettingsFromModel(masqueradeStreamSettings(tc.in)))
			hy, _ := decodeStream(t, streamJSON)["hysteriaSettings"].(map[string]any)
			if !reflect.DeepEqual(hy["masquerade"], tc.wire) {
				t.Fatalf("masquerade on the wire = %#v, want %#v", hy["masquerade"], tc.wire)
			}
			assertMasqueradeEqual(t, masqueradeFromStreamJSON(t, streamJSON), tc.in)
		})
	}
}

// The 3x-ui form writes zeros for the keys a type does not use; xray-core
// reads them as unset, so they must read back as null rather than plan a diff.
func TestMasqueradeReadsPanelFormValues(t *testing.T) {
	stored := `{"network":"hysteria","hysteriaSettings":{"version":2,"masquerade":{` +
		`"type":"file","dir":"/var/www","url":"","rewriteHost":false,"xForwarded":false,"insecure":false,` +
		`"content":"","headers":{},"statusCode":0}}}`
	assertMasqueradeEqual(t, masqueradeFromStreamJSON(t, stored), nullMasquerade(InboundHysteriaMasqueradeModel{
		Type: types.StringValue("file"), Dir: types.StringValue("/var/www"),
	}))

	// Values xray-core cannot decode read as unset, so the next write replaces them.
	odd := `{"network":"hysteria","hysteriaSettings":{"masquerade":{"type":"string","headers":{"X-N":1},"statusCode":200.5}}}`
	assertMasqueradeEqual(t, masqueradeFromStreamJSON(t, odd), nullMasquerade(InboundHysteriaMasqueradeModel{
		Type: types.StringValue("string"),
	}))
}

func TestMasqueradeRemovedFromConfig(t *testing.T) {
	stream := decodeStream(t, buildStreamSettingsJSON(expandStreamSettingsFromModel(masqueradeStreamSettings(nil))))
	hy, _ := stream["hysteriaSettings"].(map[string]any)
	if _, ok := hy["masquerade"]; ok {
		t.Fatalf("a removed block must leave no masquerade key: %#v", hy)
	}
	if got := masqueradeFromStreamJSON(t, `{"network":"hysteria","hysteriaSettings":{"version":2}}`); got != nil {
		t.Fatalf("no masquerade on the panel must read as no block: %+v", got)
	}

	// A removed attribute must not stay on the wire either.
	unknown := nullMasquerade(InboundHysteriaMasqueradeModel{
		Type: types.StringValue("string"), Content: types.StringUnknown(), RewriteHost: types.BoolUnknown(),
		Headers: types.MapUnknown(types.StringType), StatusCode: types.Int64Unknown(),
	})
	hy, _ = decodeStream(t, buildStreamSettingsJSON(expandStreamSettingsFromModel(masqueradeStreamSettings(unknown))))["hysteriaSettings"].(map[string]any)
	if !reflect.DeepEqual(hy["masquerade"], map[string]any{"type": "string"}) {
		t.Fatalf("unknown values must not be written: %#v", hy["masquerade"])
	}
}

// hysteria_settings with nothing but a masquerade must still read back.
func TestMasqueradeKeepsHysteriaSettingsOnOtherNetworks(t *testing.T) {
	got := masqueradeFromStreamJSON(t, `{"network":"tcp","hysteriaSettings":{"masquerade":{"type":"string","content":"ok"}}}`)
	assertMasqueradeEqual(t, got, nullMasquerade(InboundHysteriaMasqueradeModel{
		Type: types.StringValue("string"), Content: types.StringValue("ok"),
	}))
}

// ---------------------------------------------------------------------------
// Plan-time validation
// ---------------------------------------------------------------------------

func TestMasqueradeConfigValidation(t *testing.T) {
	str := types.StringValue
	cases := []struct {
		name    string
		m       InboundHysteriaMasqueradeModel
		wantErr string
	}{
		{name: "empty block"},
		{name: "file", m: InboundHysteriaMasqueradeModel{Type: str("file"), Dir: str("/var/www")}},
		{name: "proxy over https", m: InboundHysteriaMasqueradeModel{Type: str("proxy"), URL: str("https://example.com/"), RewriteHost: types.BoolValue(true)}},
		{name: "proxy over http", m: InboundHysteriaMasqueradeModel{Type: str("proxy"), URL: str("http://127.0.0.1:8080")}},
		{name: "proxy to a unix socket path", m: InboundHysteriaMasqueradeModel{Type: str("proxy"), URL: str("/run/site.sock")}},
		{name: "proxy to a unix url", m: InboundHysteriaMasqueradeModel{Type: str("proxy"), URL: str("unix:///run/site.sock")}},
		{name: "string", m: InboundHysteriaMasqueradeModel{
			Type: str("string"), Content: str("ok"), StatusCode: types.Int64Value(200),
			Headers: types.MapValueMust(types.StringType, map[string]attr.Value{"Content-Type": str("text/plain")}),
		}},
		{name: "unknown values are checked at apply", m: InboundHysteriaMasqueradeModel{Type: str("proxy"), URL: types.StringUnknown()}},
		{name: "unknown type", m: InboundHysteriaMasqueradeModel{Type: types.StringUnknown(), Dir: str("/var/www")}},
		{name: "404 is the default, not a type", m: InboundHysteriaMasqueradeModel{Type: str("404")}, wantErr: "value must be one of"},
		{name: "file without dir", m: InboundHysteriaMasqueradeModel{Type: str("file")}, wantErr: "serves its own working directory"},
		{name: "proxy without url", m: InboundHysteriaMasqueradeModel{Type: str("proxy")}, wantErr: "requires url"},
		{name: "proxy with another scheme", m: InboundHysteriaMasqueradeModel{Type: str("proxy"), URL: str("ftp://example.com")}, wantErr: "accepts only http://"},
		{name: "proxy without host", m: InboundHysteriaMasqueradeModel{Type: str("proxy"), URL: str("https://")}, wantErr: "has no host"},
		{name: "unix url without path", m: InboundHysteriaMasqueradeModel{Type: str("proxy"), URL: str("unix://")}, wantErr: "has no unix socket path"},
		{name: "unparsable url", m: InboundHysteriaMasqueradeModel{Type: str("proxy"), URL: str("http://[::1")}, wantErr: "is not a URL"},
		{name: "empty dir", m: InboundHysteriaMasqueradeModel{Type: str("file"), Dir: str("")}, wantErr: "string length must be at least 1"},
		{name: "false is the default", m: InboundHysteriaMasqueradeModel{Insecure: types.BoolValue(false)}, wantErr: "omit the attribute"},
		{name: "empty headers", m: InboundHysteriaMasqueradeModel{Headers: types.MapValueMust(types.StringType, map[string]attr.Value{})}, wantErr: "map must contain at least 1"},
		{name: "status code zero", m: InboundHysteriaMasqueradeModel{StatusCode: types.Int64Value(0)}, wantErr: "between 100 and 999"},
		{name: "status code too large", m: InboundHysteriaMasqueradeModel{StatusCode: types.Int64Value(1000)}, wantErr: "between 100 and 999"},
	}
	for _, tc := range cases {
		for typeName, config := range map[string]func(*InboundHysteriaMasqueradeModel) tfsdk.Plan{
			"threexui_inbound": func(m *InboundHysteriaMasqueradeModel) tfsdk.Plan {
				return inboundResourceFixture(t, &InboundResource{}, &InboundResourceModel{
					Port: types.Int64Value(443), Protocol: types.StringValue("hysteria"),
					StreamSettings: masqueradeStreamSettings(m),
				})
			},
			"threexui_xray_outbounds": func(m *InboundHysteriaMasqueradeModel) tfsdk.Plan {
				return outboundsStreamConfig(t, masqueradeStreamSettings(m))
			},
		} {
			t.Run(typeName+"/"+tc.name, func(t *testing.T) {
				assertConfigErrors(t, validateResourceConfigErrors(t, typeName, config(nullMasquerade(tc.m))), tc.wantErr)
			})
		}
	}

	ctx := context.Background()
	for _, d := range []interface {
		Description(context.Context) string
		MarkdownDescription(context.Context) string
	}{masqueradeTrueOnlyValidator{}, hysteriaMasqueradeValidator{}} {
		if d.Description(ctx) == "" || d.MarkdownDescription(ctx) != d.Description(ctx) {
			t.Errorf("%T must describe itself", d)
		}
	}
}

func TestMasqueradeSchemaNotComputed(t *testing.T) {
	block := hysteriaMasqueradeBlockSchema()
	for name, a := range block.Attributes {
		if !a.IsOptional() || a.IsComputed() {
			t.Errorf("masquerade.%s must be Optional and not Computed: removing it must remove the key", name)
		}
	}
	model := reflect.TypeOf(InboundHysteriaMasqueradeModel{})
	if model.NumField() != len(block.Attributes) {
		t.Fatalf("model has %d fields, schema %d attributes", model.NumField(), len(block.Attributes))
	}
	if len(masqueradeStringWireKeys)+len(masqueradeBoolWireKeys)+2 != len(block.Attributes) {
		t.Fatal("every masquerade attribute must have a wire key")
	}
}

// ---------------------------------------------------------------------------
// Removing the block
// ---------------------------------------------------------------------------

func TestAlignBlocksWithPlan_Masquerade(t *testing.T) {
	stored := func() *InboundResourceModel {
		return &InboundResourceModel{StreamSettings: &InboundStreamSettingsModel{
			HysteriaSettings: &InboundHysteriaStreamSettingsModel{Masquerade: &InboundHysteriaMasqueradeModel{Type: types.StringValue("file")}},
		}}
	}
	withHysteria := func(m *InboundHysteriaMasqueradeModel) *InboundResourceModel {
		return &InboundResourceModel{StreamSettings: &InboundStreamSettingsModel{
			HysteriaSettings: &InboundHysteriaStreamSettingsModel{Masquerade: m},
		}}
	}

	state := stored()
	alignBlocksWithPlan(state, withHysteria(nil))
	if state.StreamSettings.HysteriaSettings.Masquerade != nil {
		t.Fatalf("masquerade absent from the plan must be dropped: %+v", state.StreamSettings.HysteriaSettings.Masquerade)
	}

	state = stored()
	alignBlocksWithPlan(state, withHysteria(&InboundHysteriaMasqueradeModel{}))
	if state.StreamSettings.HysteriaSettings.Masquerade == nil {
		t.Fatal("masquerade present in the plan must be kept")
	}

	state = stored()
	alignBlocksWithPlan(state, &InboundResourceModel{StreamSettings: &InboundStreamSettingsModel{}})
	if state.StreamSettings.HysteriaSettings != nil {
		t.Fatalf("hysteria_settings absent from the plan must be dropped with its masquerade: %+v", state.StreamSettings.HysteriaSettings)
	}

	state = &InboundResourceModel{StreamSettings: &InboundStreamSettingsModel{}}
	alignBlocksWithPlan(state, withHysteria(nil))
	if state.StreamSettings.HysteriaSettings != nil {
		t.Fatalf("absent hysteria_settings must stay absent: %+v", state.StreamSettings.HysteriaSettings)
	}
}

func TestXrayOutboundsResource_ModifyPlan_MasqueradeNotRefilledFromState(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	NewXrayOutboundsResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	outbound := func(m *InboundHysteriaMasqueradeModel) XrayOutboundEntry {
		return XrayOutboundEntry{
			Tag: types.StringValue("hy"), Protocol: types.StringValue("hysteria"),
			StreamSettings: masqueradeStreamSettings(m),
		}
	}
	prior := nullMasquerade(InboundHysteriaMasqueradeModel{
		Type: types.StringValue("proxy"), URL: types.StringValue("https://example.com"), RewriteHost: types.BoolValue(true),
	})
	stateRaw := xrayOutboundsRawFromModel(t, schemaResp, outbound(prior))
	for _, tc := range []struct {
		name       string
		configured *InboundHysteriaMasqueradeModel
	}{
		{"removed block", nil},
		{"removed attributes", nullMasquerade(InboundHysteriaMasqueradeModel{Type: types.StringValue("proxy"), URL: types.StringValue("https://example.org")})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configRaw := xrayOutboundsRawFromModel(t, schemaResp, outbound(tc.configured))
			plan := runXrayOutboundsModifyPlan(t, schemaResp, stateRaw, configRaw, stateRaw)
			var got XrayOutboundsModel
			if diags := plan.Get(ctx, &got); diags.HasError() {
				t.Fatalf("reading plan: %v", diags)
			}
			assertMasqueradeEqual(t, got.Outbound[0].StreamSettings.HysteriaSettings.Masquerade, tc.configured)
		})
	}
}
