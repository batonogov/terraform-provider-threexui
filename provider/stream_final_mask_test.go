package provider

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// What jsonencode() produces for a salamander mask: sorted keys, no spaces.
const salamanderFinalMask = `{"udp":[{"settings":{"password":"obfs-secret"},"type":"salamander"}]}`

func finalMaskStreamSettings(finalMask jsonObjectValue) *InboundStreamSettingsModel {
	return &InboundStreamSettingsModel{
		Network:          types.StringValue("hysteria"),
		Security:         types.StringValue("tls"),
		HysteriaSettings: &InboundHysteriaStreamSettingsModel{Version: types.Int64Value(2)},
		FinalMask:        finalMask,
	}
}

func decodeStream(t *testing.T, streamJSON string) map[string]any {
	t.Helper()
	var stream map[string]any
	if err := json.Unmarshal([]byte(streamJSON), &stream); err != nil {
		t.Fatalf("stream settings %s: %v", streamJSON, err)
	}
	return stream
}

func finalMaskFromStreamJSON(t *testing.T, streamJSON string) jsonObjectValue {
	t.Helper()
	flat, err := flattenStreamSettingsToMap(streamJSON)
	if err != nil {
		t.Fatalf("flattenStreamSettingsToMap(%s): %v", streamJSON, err)
	}
	return flattenStreamSettingsToModel(flat).FinalMask
}

func TestFinalMaskWireRoundTrip(t *testing.T) {
	streamJSON := buildStreamSettingsJSON(expandStreamSettingsFromModel(finalMaskStreamSettings(newJSONObjectValue(salamanderFinalMask))))
	stream := decodeStream(t, streamJSON)
	want := map[string]any{"udp": []any{map[string]any{
		"type": "salamander", "settings": map[string]any{"password": "obfs-secret"},
	}}}
	if !reflect.DeepEqual(stream["finalmask"], want) {
		t.Fatalf("finalmask on the wire = %#v, want %#v", stream["finalmask"], want)
	}
	if _, ok := stream["final_mask"]; ok {
		t.Fatal("the snake_case key must not reach the wire")
	}
	if got := finalMaskFromStreamJSON(t, streamJSON); got.ValueString() != salamanderFinalMask {
		t.Fatalf("read back %s, want %s", got, salamanderFinalMask)
	}
}

// The panel stores what the form or another tool wrote; reading it must give
// the jsonencode() spelling, so an imported inbound does not plan an update.
func TestFinalMaskReadsPanelLayout(t *testing.T) {
	stored := `{
  "network": "hysteria",
  "finalmask": {
    "udp": [ { "type": "salamander", "settings": { "password": "obfs-secret" } } ],
    "quicParams": { "congestion": "bbr", "maxIdleTimeout": 30 }
  }
}`
	want := `{"quicParams":{"congestion":"bbr","maxIdleTimeout":30},"udp":[{"settings":{"password":"obfs-secret"},"type":"salamander"}]}`
	if got := finalMaskFromStreamJSON(t, stored); got.ValueString() != want {
		t.Fatalf("read back %s, want %s", got, want)
	}

	// The rest of the payload goes through float64; finalmask must not.
	big := `{"network":"hysteria","finalmask":{"udp":[{"type":"noise","settings":{"seed":9007199254740993,"ratio":1.50}}]}}`
	got := finalMaskFromStreamJSON(t, big)
	if !strings.Contains(got.ValueString(), "9007199254740993") || !strings.Contains(got.ValueString(), "1.50") {
		t.Fatalf("numbers must be kept verbatim: %s", got)
	}

	for name, stream := range map[string]string{
		"absent":    `{"network":"hysteria"}`,
		"json null": `{"network":"hysteria","finalmask":null}`,
	} {
		if got := finalMaskFromStreamJSON(t, stream); !got.IsNull() {
			t.Errorf("%s: final_mask = %s, want null", name, got)
		}
	}
	if got := finalMaskFromStreamJSON(t, `{"finalmask":{}}`); got.ValueString() != "{}" {
		t.Errorf("an empty finalmask must stay {}: %s", got)
	}
}

// The bug this attribute fixes: every update rebuilds streamSettings from the
// model, so a finalmask the model does not carry is erased.
func TestFinalMaskSurvivesRebuild(t *testing.T) {
	stored := `{"network":"hysteria","security":"tls","hysteriaSettings":{"version":2},` +
		`"finalmask":{"udp":[{"type":"salamander","settings":{"password":"obfs-secret"}}]}}`
	flat, err := flattenStreamSettingsToMap(stored)
	if err != nil {
		t.Fatal(err)
	}
	rewritten := decodeStream(t, buildStreamSettingsJSON(expandStreamSettingsFromModel(flattenStreamSettingsToModel(flat))))
	if !reflect.DeepEqual(rewritten["finalmask"], decodeStream(t, stored)["finalmask"]) {
		t.Fatalf("finalmask lost on rewrite: %#v", rewritten)
	}
}

func TestFinalMaskRemovedFromConfig(t *testing.T) {
	stream := decodeStream(t, buildStreamSettingsJSON(expandStreamSettingsFromModel(finalMaskStreamSettings(jsonObjectNull()))))
	if _, ok := stream["finalmask"]; ok {
		t.Fatalf("a null final_mask must leave no finalmask key: %#v", stream)
	}
	stream = decodeStream(t, buildStreamSettingsJSON(expandStreamSettingsFromModel(finalMaskStreamSettings(
		jsonObjectValue{StringValue: types.StringUnknown()}))))
	if _, ok := stream["finalmask"]; ok {
		t.Fatalf("an unknown final_mask must not be written: %#v", stream)
	}
}

// buildStreamSettingsJSON falls back to "{}" when json.Marshal fails, which
// would wipe every stream setting; an invalid document is dropped instead.
func TestFinalMaskInvalidJSONNotWritten(t *testing.T) {
	stream := decodeStream(t, buildStreamSettingsJSON(map[string]any{"network": "hysteria", "final_mask": "{not json"}))
	if stream["network"] != "hysteria" {
		t.Fatalf("the rest of the stream settings must survive: %#v", stream)
	}
	if _, ok := stream["finalmask"]; ok {
		t.Fatalf("invalid finalmask must not be written: %#v", stream)
	}
}

func TestCanonicalJSON(t *testing.T) {
	for in, want := range map[string]string{
		` { "b" : 1 , "a" : [ true , null ] } `: `{"a":[true,null],"b":1}`,
		// jsonencode escapes these the same way.
		`{"html":"<&>"}`:       "{\"html\":\"\\u003c\\u0026\\u003e\"}",
		`12345678901234567890`: `12345678901234567890`,
	} {
		got, err := canonicalJSON([]byte(in))
		if err != nil || got != want {
			t.Errorf("canonicalJSON(%s) = %s, %v; want %s", in, got, err, want)
		}
	}
	for _, in := range []string{`{"a":`, `{} {}`, ``} {
		if got, err := canonicalJSON([]byte(in)); err == nil {
			t.Errorf("canonicalJSON(%q) = %s, want an error", in, got)
		}
	}
}

// ---------------------------------------------------------------------------
// Custom type
// ---------------------------------------------------------------------------

func TestJSONObjectSemanticEquals(t *testing.T) {
	ctx := context.Background()
	state := newJSONObjectValue(salamanderFinalMask)
	cases := []struct {
		name string
		new  basetypes.StringValuable
		want bool
	}{
		{"same text", newJSONObjectValue(salamanderFinalMask), true},
		{"other layout", newJSONObjectValue("{\n  \"udp\": [{\"type\": \"salamander\", \"settings\": {\"password\": \"obfs-secret\"}}]\n}"), true},
		{"plain string value", types.StringValue(`{"udp":[{"type":"salamander","settings":{"password":"obfs-secret"}}]}`), true},
		{"other password", newJSONObjectValue(`{"udp":[{"type":"salamander","settings":{"password":"other"}}]}`), false},
		{"mask order matters", newJSONObjectValue(`{"udp":[{"type":"noise"},{"type":"salamander"}]}`), false},
		{"invalid new value", newJSONObjectValue(`{`), false},
		{"null new value", jsonObjectNull(), false},
		{"unknown new value", jsonObjectValue{StringValue: types.StringUnknown()}, false},
	}
	for _, tc := range cases {
		got, diags := state.StringSemanticEquals(ctx, tc.new)
		if diags.HasError() || got != tc.want {
			t.Errorf("%s: StringSemanticEquals = %v (%v), want %v", tc.name, got, diags, tc.want)
		}
	}
	for name, prior := range map[string]jsonObjectValue{
		"null prior":    jsonObjectNull(),
		"unknown prior": {StringValue: types.StringUnknown()},
		"invalid prior": newJSONObjectValue(`{`),
	} {
		if got, _ := prior.StringSemanticEquals(ctx, state); got {
			t.Errorf("%s must not equal a value", name)
		}
	}
}

func TestJSONObjectType(t *testing.T) {
	ctx := context.Background()
	typ := jsonObjectType{}
	if !typ.Equal(jsonObjectType{}) || typ.Equal(types.StringType) {
		t.Error("jsonObjectType must equal only itself")
	}
	if typ.String() != "jsonObjectType" {
		t.Errorf("String() = %q", typ.String())
	}
	if _, ok := typ.ValueType(ctx).(jsonObjectValue); !ok {
		t.Errorf("ValueType = %T", typ.ValueType(ctx))
	}
	v, diags := typ.ValueFromString(ctx, types.StringValue("{}"))
	if diags.HasError() || v.(jsonObjectValue).ValueString() != "{}" {
		t.Errorf("ValueFromString = %v, %v", v, diags)
	}

	for name, tc := range map[string]struct {
		in   tftypes.Value
		want jsonObjectValue
	}{
		"value":   {tftypes.NewValue(tftypes.String, "{}"), newJSONObjectValue("{}")},
		"null":    {tftypes.NewValue(tftypes.String, nil), jsonObjectNull()},
		"unknown": {tftypes.NewValue(tftypes.String, tftypes.UnknownValue), jsonObjectValue{StringValue: types.StringUnknown()}},
	} {
		got, err := typ.ValueFromTerraform(ctx, tc.in)
		if err != nil || !got.Equal(tc.want) {
			t.Errorf("%s: ValueFromTerraform = %v, %v; want %v", name, got, err, tc.want)
		}
	}
	if _, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(tftypes.Number, 1)); err == nil {
		t.Error("ValueFromTerraform must reject a non-string")
	}

	value := newJSONObjectValue("{}")
	if !value.Type(ctx).Equal(typ) {
		t.Errorf("Type() = %v", value.Type(ctx))
	}
	if value.Equal(types.StringValue("{}")) {
		t.Error("a plain string must not equal the custom value")
	}
}

// With CustomType declared, State.Set rejects a model field of another type;
// both resources share the stream settings model.
func TestFinalMaskStateRoundTrip(t *testing.T) {
	ctx := context.Background()
	plan := inboundResourceFixture(t, &InboundResource{}, &InboundResourceModel{
		Port: types.Int64Value(443), Protocol: types.StringValue("hysteria"),
		StreamSettings: finalMaskStreamSettings(newJSONObjectValue(salamanderFinalMask)),
	})
	var got InboundResourceModel
	if diags := plan.Get(ctx, &got); diags.HasError() {
		t.Fatalf("Get: %v", diags)
	}
	if got.StreamSettings.FinalMask.ValueString() != salamanderFinalMask {
		t.Fatalf("final_mask = %s", got.StreamSettings.FinalMask)
	}

	var schemaResp resource.SchemaResponse
	NewXrayOutboundsResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	raw := xrayOutboundsRawFromModel(t, schemaResp, XrayOutboundEntry{
		Tag: types.StringValue("hy"), Protocol: types.StringValue("hysteria"),
		StreamSettings: finalMaskStreamSettings(newJSONObjectValue(salamanderFinalMask)),
	})
	var outbounds XrayOutboundsModel
	if diags := (tfsdk.State{Schema: schemaResp.Schema, Raw: raw}).Get(ctx, &outbounds); diags.HasError() {
		t.Fatalf("outbounds Get: %v", diags)
	}
	if outbounds.Outbound[0].StreamSettings.FinalMask.ValueString() != salamanderFinalMask {
		t.Fatalf("outbound final_mask = %s", outbounds.Outbound[0].StreamSettings.FinalMask)
	}
}

func TestFinalMaskSchema(t *testing.T) {
	attr := inboundStreamSettingsBlockSchema().Attributes["final_mask"]
	if !attr.IsSensitive() || !attr.IsOptional() || attr.IsComputed() {
		t.Fatal("final_mask must be Optional and Sensitive, not Computed: removing it must remove the key")
	}
}

// ---------------------------------------------------------------------------
// Plan-time validation
// ---------------------------------------------------------------------------

func TestFinalMaskConfigValidation(t *testing.T) {
	cases := []struct {
		name      string
		finalMask jsonObjectValue
		wantErr   string
	}{
		{name: "salamander", finalMask: newJSONObjectValue(salamanderFinalMask)},
		{name: "four-byte password", finalMask: newJSONObjectValue(`{"udp":[{"type":"salamander","settings":{"password":"abcd"}}]}`)},
		{name: "tcp and quicParams", finalMask: newJSONObjectValue(`{"tcp":[{"type":"fragment","settings":{"packets":"tlshello","length":"100-200"}}],"quicParams":{"congestion":"bbr"}}`)},
		{name: "short password of another mask", finalMask: newJSONObjectValue(`{"udp":[{"type":"noise","settings":{"password":"x"}}]}`)},
		{name: "empty object", finalMask: newJSONObjectValue(`{}`)},
		{name: "omitted", finalMask: jsonObjectNull()},
		{name: "unknown is checked at apply", finalMask: jsonObjectValue{StringValue: types.StringUnknown()}},
		{name: "invalid JSON", finalMask: newJSONObjectValue(`{"udp":`), wantErr: "must be a JSON object"},
		{name: "array", finalMask: newJSONObjectValue(`[]`), wantErr: "must be a JSON object"},
		{name: "json null", finalMask: newJSONObjectValue(`null`), wantErr: "must be a JSON object"},
		{name: "short salamander password", finalMask: newJSONObjectValue(`{"udp":[{"type":"salamander","settings":{"password":"abc"}}]}`), wantErr: "udp[0]: the salamander password must be at least 4 bytes"},
		{name: "type matched case-insensitively", finalMask: newJSONObjectValue(`{"udp":[{"type":"noise"},{"type":"Salamander","settings":{"password":""}}]}`), wantErr: "udp[1]"},
		{name: "missing password", finalMask: newJSONObjectValue(`{"udp":[{"type":"salamander"}]}`), wantErr: "at least 4 bytes"},
	}
	for _, tc := range cases {
		for typeName, config := range map[string]func() tfsdk.Plan{
			"threexui_inbound": func() tfsdk.Plan {
				return inboundResourceFixture(t, &InboundResource{}, &InboundResourceModel{
					Port: types.Int64Value(443), Protocol: types.StringValue("hysteria"),
					StreamSettings: finalMaskStreamSettings(tc.finalMask),
				})
			},
			"threexui_xray_outbounds": func() tfsdk.Plan {
				return outboundsStreamConfig(t, finalMaskStreamSettings(tc.finalMask))
			},
		} {
			t.Run(typeName+"/"+tc.name, func(t *testing.T) {
				assertConfigErrors(t, validateResourceConfigErrors(t, typeName, config()), tc.wantErr)
			})
		}
	}
	ctx := context.Background()
	if d := (finalMaskValidator{}); d.Description(ctx) == "" || d.MarkdownDescription(ctx) != d.Description(ctx) {
		t.Error("finalMaskValidator must describe itself")
	}
}

func outboundsStreamConfig(t *testing.T, stream *InboundStreamSettingsModel) tfsdk.Plan {
	t.Helper()
	var schemaResp resource.SchemaResponse
	NewXrayOutboundsResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	p := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := p.Set(context.Background(), &XrayOutboundsModel{
		ID: types.StringNull(),
		Outbound: []XrayOutboundEntry{{
			Tag: types.StringValue("proxy"), Protocol: types.StringValue("hysteria"),
			StreamSettings: stream,
		}},
	}); diags.HasError() {
		t.Fatalf("building outbounds config: %v", diags)
	}
	return p
}

// assertConfigErrors expects no error for an empty wantErr, otherwise one
// containing it.
func assertConfigErrors(t *testing.T, errs []*tfprotov6.Diagnostic, wantErr string) {
	t.Helper()
	var messages []string
	for _, d := range errs {
		msg := d.Summary + ": " + d.Detail
		messages = append(messages, msg)
		if wantErr != "" && strings.Contains(msg, wantErr) {
			return
		}
	}
	if wantErr != "" {
		t.Fatalf("want an error containing %q, got %v", wantErr, messages)
	}
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", messages)
	}
}

// ---------------------------------------------------------------------------
// Removing the attribute in xray_outbounds
// ---------------------------------------------------------------------------

// The outbounds ModifyPlan refills omitted sockopt attributes from state;
// final_mask is not Computed, so a removed one must stay removed.
func TestXrayOutboundsResource_ModifyPlan_FinalMaskNotRefilledFromState(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	NewXrayOutboundsResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	outbound := func(fm jsonObjectValue) XrayOutboundEntry {
		return XrayOutboundEntry{
			Tag: types.StringValue("hy"), Protocol: types.StringValue("hysteria"),
			StreamSettings: finalMaskStreamSettings(fm),
		}
	}
	stateRaw := xrayOutboundsRawFromModel(t, schemaResp, outbound(newJSONObjectValue(salamanderFinalMask)))
	configRaw := xrayOutboundsRawFromModel(t, schemaResp, outbound(jsonObjectNull()))
	plan := runXrayOutboundsModifyPlan(t, schemaResp, stateRaw, configRaw, stateRaw)
	var got XrayOutboundsModel
	if diags := plan.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading plan: %v", diags)
	}
	if fm := got.Outbound[0].StreamSettings.FinalMask; !fm.IsNull() {
		t.Fatalf("final_mask removed from the configuration came back from state: %s", fm)
	}
}
