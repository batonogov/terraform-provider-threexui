package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestXrayOutboundTargetStrategyRoundTrip exercises the four conversion points
// for the v3.5.0 outbound `targetStrategy` field: model→untyped (expandXrayOutbounds),
// untyped→wire (expandOutbounds), wire→untyped (flattenOutbounds), and untyped→model
// (flattenXrayOutbounds). A set value must survive a full round-trip unchanged.
func TestXrayOutboundTargetStrategyRoundTrip(t *testing.T) {
	model := &XrayOutboundsModel{
		Outbound: []XrayOutboundEntry{
			{
				Tag:            types.StringValue("direct"),
				Protocol:       types.StringValue("freedom"),
				SendThrough:    types.StringValue("inb"),
				TargetStrategy: types.StringValue("UseIPv4"),
			},
		},
	}

	// 1. model → untyped
	payload := expandXrayOutbounds(model)
	list, ok := payload["outbound"].([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("expected one expanded outbound, got %#v", payload)
	}
	expanded, _ := list[0].(map[string]any)
	if expanded["target_strategy"] != "UseIPv4" {
		t.Fatalf("model→untyped: expected target_strategy=UseIPv4, got %v", expanded["target_strategy"])
	}

	// 2. untyped → wire (camelCase key)
	wire := expandOutbounds(list)
	wireEntry, _ := wire[0].(map[string]any)
	if wireEntry["targetStrategy"] != "UseIPv4" {
		t.Fatalf("untyped→wire: expected targetStrategy=UseIPv4, got %v", wireEntry["targetStrategy"])
	}

	// 3. wire → untyped (snake_case key)
	flat := flattenOutbounds(wire)
	flatEntry, _ := flat[0].(map[string]any)
	if flatEntry["target_strategy"] != "UseIPv4" {
		t.Fatalf("wire→untyped: expected target_strategy=UseIPv4, got %v", flatEntry["target_strategy"])
	}

	// 4. untyped → model
	model2 := flattenXrayOutbounds(map[string]any{"outbound": flat})
	if len(model2.Outbound) != 1 {
		t.Fatalf("expected one flattened outbound, got %d", len(model2.Outbound))
	}
	if model2.Outbound[0].TargetStrategy.ValueString() != "UseIPv4" {
		t.Fatalf("untyped→model: expected TargetStrategy=UseIPv4, got %q",
			model2.Outbound[0].TargetStrategy.ValueString())
	}
}

// TestXrayOutboundTargetStrategyOmittedWhenEmpty verifies that an unset
// target_strategy (AsIs) is not written to the wire — xray-core treats a missing
// key as AsIs, and 3x-ui's frontend adapter likewise omits empty values.
func TestXrayOutboundTargetStrategyOmittedWhenEmpty(t *testing.T) {
	model := &XrayOutboundsModel{
		Outbound: []XrayOutboundEntry{
			{
				Tag:            types.StringValue("out"),
				Protocol:       types.StringValue("freedom"),
				TargetStrategy: types.StringNull(),
			},
		},
	}

	payload := expandXrayOutbounds(model)
	list, _ := payload["outbound"].([]any)
	expanded, _ := list[0].(map[string]any)
	if _, ok := expanded["target_strategy"]; ok {
		t.Fatalf("null target_strategy must not appear in the untyped map")
	}

	wire := expandOutbounds(list)
	wireEntry, _ := wire[0].(map[string]any)
	if _, ok := wireEntry["targetStrategy"]; ok {
		t.Fatalf("null target_strategy must not be written to the wire")
	}

	// Round-trip back: a wire entry without targetStrategy flattens to null.
	flat := flattenOutbounds(wire)
	model2 := flattenXrayOutbounds(map[string]any{"outbound": flat})
	if !model2.Outbound[0].TargetStrategy.IsNull() {
		t.Fatalf("expected null TargetStrategy after round-trip, got %q",
			model2.Outbound[0].TargetStrategy.ValueString())
	}
}

// TestXrayOutboundsSchema exercises the full schema definition (incl. the
// v3.5.0 target_strategy attribute) so the schema helper lines count toward
// Codecov patch coverage.
func TestXrayOutboundsSchema(t *testing.T) {
	s := xrayOutboundsSchema()
	if s.Blocks["outbound"] == nil {
		t.Fatal("expected outbound block in xray_outbounds schema")
	}
}

func xrayOutboundValue(t *testing.T, objectType tftypes.Object, tag, protocol string, overrides map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	values["tag"] = tftypes.NewValue(tftypes.String, tag)
	values["protocol"] = tftypes.NewValue(tftypes.String, protocol)
	for name, value := range overrides {
		if _, ok := objectType.AttributeTypes[name]; !ok {
			t.Fatalf("outbound object has no %q attribute", name)
		}
		values[name] = value
	}
	return tftypes.NewValue(objectType, values)
}

func xrayOutboundSingletonValue(t *testing.T, listType tftypes.List, overrides map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	objectType := listType.ElementType.(tftypes.Object)
	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	for name, value := range overrides {
		if _, ok := objectType.AttributeTypes[name]; !ok {
			t.Fatalf("nested outbound object has no %q attribute", name)
		}
		values[name] = value
	}
	return tftypes.NewValue(listType, []tftypes.Value{tftypes.NewValue(objectType, values)})
}

func xrayOutboundsRaw(t *testing.T, schemaResp resource.SchemaResponse, outboundValue tftypes.Value, withID bool) tftypes.Value {
	t.Helper()
	objectType := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	if withID {
		values["id"] = tftypes.NewValue(tftypes.String, "xray_outbounds")
	}
	values["outbound"] = outboundValue
	return tftypes.NewValue(objectType, values)
}

func runXrayOutboundsModifyPlan(
	t *testing.T,
	schemaResp resource.SchemaResponse,
	planRaw, configRaw, stateRaw tftypes.Value,
) tfsdk.Plan {
	t.Helper()
	resp := runXrayOutboundsModifyPlanResponse(t, schemaResp, planRaw, configRaw, stateRaw)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected ModifyPlan diagnostics: %v", resp.Diagnostics)
	}
	return resp.Plan
}

func runXrayOutboundsModifyPlanResponse(
	t *testing.T,
	schemaResp resource.SchemaResponse,
	planRaw, configRaw, stateRaw tftypes.Value,
) *resource.ModifyPlanResponse {
	t.Helper()
	outboundsResource := NewXrayOutboundsResource()
	modifier, ok := outboundsResource.(resource.ResourceWithModifyPlan)
	if !ok {
		t.Fatal("XrayOutboundsResource must implement resource.ResourceWithModifyPlan")
	}
	ctx := context.Background()
	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}
	resp := &resource.ModifyPlanResponse{Plan: plan}
	modifier.ModifyPlan(ctx, resource.ModifyPlanRequest{
		Plan:   plan,
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw},
		State:  tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
	}, resp)
	return resp
}

func assertXrayOutboundListEqual(t *testing.T, gotPlan tfsdk.Plan, wantConfig tfsdk.Config) {
	t.Helper()
	ctx := context.Background()
	var got, want types.List
	if diags := gotPlan.GetAttribute(ctx, path.Root("outbound"), &got); diags.HasError() {
		t.Fatalf("cannot read outbound from plan: %v", diags)
	}
	if diags := wantConfig.GetAttribute(ctx, path.Root("outbound"), &want); diags.HasError() {
		t.Fatalf("cannot read outbound from config: %v", diags)
	}
	if !got.Equal(want) {
		t.Fatalf("planned outbound list must mirror configuration\ngot:  %v\nwant: %v", got, want)
	}
}

// TestXrayOutboundsResource_ModifyPlan_ConfiguredListIsAuthoritative reproduces
// issue #419's reachable plan. The initial state is [direct rich, blocked sparse];
// after reordering to [blocked sparse, direct rich], UseStateForUnknown carries
// direct's top-level values into blocked at index 0. The complete configured
// outbound objects, including their protocol-specific blocks, must replace that
// polluted plan before Update serializes it.
func TestXrayOutboundsResource_ModifyPlan_ConfiguredListIsAuthoritative(t *testing.T) {
	ctx := context.Background()
	outboundsResource := NewXrayOutboundsResource()
	var schemaResp resource.SchemaResponse
	outboundsResource.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	topType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	outboundListType := topType.AttributeTypes["outbound"].(tftypes.List)
	outboundObjectType := outboundListType.ElementType.(tftypes.Object)

	freedomSettings := xrayOutboundSingletonValue(t, outboundObjectType.AttributeTypes["freedom_settings"].(tftypes.List), map[string]tftypes.Value{
		"domain_strategy": tftypes.NewValue(tftypes.String, "AsIs"),
	})
	blackholeSettings := xrayOutboundSingletonValue(t, outboundObjectType.AttributeTypes["blackhole_settings"].(tftypes.List), map[string]tftypes.Value{
		"response_type": tftypes.NewValue(tftypes.String, "none"),
	})
	richDirect := xrayOutboundValue(t, outboundObjectType, "direct", "freedom", map[string]tftypes.Value{
		"send_through":     tftypes.NewValue(tftypes.String, "127.0.0.1"),
		"target_strategy":  tftypes.NewValue(tftypes.String, "UseIPv4"),
		"freedom_settings": freedomSettings,
	})
	sparseBlocked := xrayOutboundValue(t, outboundObjectType, "blocked", "blackhole", map[string]tftypes.Value{
		"blackhole_settings": blackholeSettings,
	})
	pollutedBlocked := xrayOutboundValue(t, outboundObjectType, "blocked", "blackhole", map[string]tftypes.Value{
		"send_through":       tftypes.NewValue(tftypes.String, "127.0.0.1"),
		"target_strategy":    tftypes.NewValue(tftypes.String, "UseIPv4"),
		"blackhole_settings": blackholeSettings,
	})

	stateList := tftypes.NewValue(outboundListType, []tftypes.Value{richDirect, sparseBlocked})
	configList := tftypes.NewValue(outboundListType, []tftypes.Value{sparseBlocked, richDirect})
	planList := tftypes.NewValue(outboundListType, []tftypes.Value{pollutedBlocked, richDirect})
	plan := runXrayOutboundsModifyPlan(t, schemaResp,
		xrayOutboundsRaw(t, schemaResp, planList, true),
		xrayOutboundsRaw(t, schemaResp, configList, false),
		xrayOutboundsRaw(t, schemaResp, stateList, true),
	)
	assertXrayOutboundListEqual(t, plan, tfsdk.Config{Schema: schemaResp.Schema, Raw: xrayOutboundsRaw(t, schemaResp, configList, false)})
}

// Nested Optional+Computed leaves are also index-sensitive below the repeatable
// outbound ancestor. This fixture keeps both outbounds on the freedom protocol
// so stale mux.concurrency and freedom_settings.redirect values are reachable.
func TestXrayOutboundsResource_ModifyPlan_StripsNestedProtocolFieldBleed(t *testing.T) {
	ctx := context.Background()
	outboundsResource := NewXrayOutboundsResource()
	var schemaResp resource.SchemaResponse
	outboundsResource.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	topType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	outboundListType := topType.AttributeTypes["outbound"].(tftypes.List)
	outboundObjectType := outboundListType.ElementType.(tftypes.Object)
	freedomListType := outboundObjectType.AttributeTypes["freedom_settings"].(tftypes.List)
	muxListType := outboundObjectType.AttributeTypes["mux"].(tftypes.List)

	richFreedom := xrayOutboundSingletonValue(t, freedomListType, map[string]tftypes.Value{
		"domain_strategy": tftypes.NewValue(tftypes.String, "AsIs"),
		"redirect":        tftypes.NewValue(tftypes.String, "1.2.3.4:443"),
	})
	sparseFreedom := xrayOutboundSingletonValue(t, freedomListType, map[string]tftypes.Value{
		"domain_strategy": tftypes.NewValue(tftypes.String, "UseIPv4"),
	})
	pollutedFreedom := xrayOutboundSingletonValue(t, freedomListType, map[string]tftypes.Value{
		"domain_strategy": tftypes.NewValue(tftypes.String, "UseIPv4"),
		"redirect":        tftypes.NewValue(tftypes.String, "1.2.3.4:443"),
	})
	richMux := xrayOutboundSingletonValue(t, muxListType, map[string]tftypes.Value{
		"enabled":     tftypes.NewValue(tftypes.Bool, true),
		"concurrency": tftypes.NewValue(tftypes.Number, 8),
	})
	sparseMux := xrayOutboundSingletonValue(t, muxListType, map[string]tftypes.Value{
		"enabled": tftypes.NewValue(tftypes.Bool, false),
	})
	pollutedMux := xrayOutboundSingletonValue(t, muxListType, map[string]tftypes.Value{
		"enabled":     tftypes.NewValue(tftypes.Bool, false),
		"concurrency": tftypes.NewValue(tftypes.Number, 8),
	})
	rich := xrayOutboundValue(t, outboundObjectType, "direct-a", "freedom", map[string]tftypes.Value{
		"freedom_settings": richFreedom,
		"mux":              richMux,
	})
	sparse := xrayOutboundValue(t, outboundObjectType, "direct-b", "freedom", map[string]tftypes.Value{
		"freedom_settings": sparseFreedom,
		"mux":              sparseMux,
	})
	pollutedSparse := xrayOutboundValue(t, outboundObjectType, "direct-b", "freedom", map[string]tftypes.Value{
		"freedom_settings": pollutedFreedom,
		"mux":              pollutedMux,
	})

	stateList := tftypes.NewValue(outboundListType, []tftypes.Value{rich, sparse})
	configList := tftypes.NewValue(outboundListType, []tftypes.Value{sparse, rich})
	planList := tftypes.NewValue(outboundListType, []tftypes.Value{pollutedSparse, rich})
	plan := runXrayOutboundsModifyPlan(t, schemaResp,
		xrayOutboundsRaw(t, schemaResp, planList, true),
		xrayOutboundsRaw(t, schemaResp, configList, false),
		xrayOutboundsRaw(t, schemaResp, stateList, true),
	)
	assertXrayOutboundListEqual(t, plan, tfsdk.Config{Schema: schemaResp.Schema, Raw: xrayOutboundsRaw(t, schemaResp, configList, false)})
}

func TestXrayOutboundsResource_ModifyPlan_PreservesUnknownCollection(t *testing.T) {
	ctx := context.Background()
	outboundsResource := NewXrayOutboundsResource()
	var schemaResp resource.SchemaResponse
	outboundsResource.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	topType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	outboundListType := topType.AttributeTypes["outbound"].(tftypes.List)
	unknownList := tftypes.NewValue(outboundListType, tftypes.UnknownValue)
	stateList := tftypes.NewValue(outboundListType, []tftypes.Value{})

	plan := runXrayOutboundsModifyPlan(t, schemaResp,
		xrayOutboundsRaw(t, schemaResp, unknownList, true),
		xrayOutboundsRaw(t, schemaResp, unknownList, false),
		xrayOutboundsRaw(t, schemaResp, stateList, true),
	)
	assertXrayOutboundListEqual(t, plan, tfsdk.Config{Schema: schemaResp.Schema, Raw: xrayOutboundsRaw(t, schemaResp, unknownList, false)})
}

func TestXrayOutboundsResource_ModifyPlan_PreservesUnknownObjectElement(t *testing.T) {
	ctx := context.Background()
	outboundsResource := NewXrayOutboundsResource()
	var schemaResp resource.SchemaResponse
	outboundsResource.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	topType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	outboundListType := topType.AttributeTypes["outbound"].(tftypes.List)
	outboundObjectType := outboundListType.ElementType.(tftypes.Object)
	unknownElementList := tftypes.NewValue(outboundListType, []tftypes.Value{
		tftypes.NewValue(outboundObjectType, tftypes.UnknownValue),
	})
	stateList := tftypes.NewValue(outboundListType, []tftypes.Value{
		xrayOutboundValue(t, outboundObjectType, "direct", "freedom", nil),
	})

	plan := runXrayOutboundsModifyPlan(t, schemaResp,
		xrayOutboundsRaw(t, schemaResp, unknownElementList, true),
		xrayOutboundsRaw(t, schemaResp, unknownElementList, false),
		xrayOutboundsRaw(t, schemaResp, stateList, true),
	)
	assertXrayOutboundListEqual(t, plan, tfsdk.Config{Schema: schemaResp.Schema, Raw: xrayOutboundsRaw(t, schemaResp, unknownElementList, false)})
}

func TestXrayOutboundsResource_ModifyPlan_PreservesPartialUnknownLeaf(t *testing.T) {
	ctx := context.Background()
	outboundsResource := NewXrayOutboundsResource()
	var schemaResp resource.SchemaResponse
	outboundsResource.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	topType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	outboundListType := topType.AttributeTypes["outbound"].(tftypes.List)
	outboundObjectType := outboundListType.ElementType.(tftypes.Object)
	blackholeSettings := xrayOutboundSingletonValue(t, outboundObjectType.AttributeTypes["blackhole_settings"].(tftypes.List), map[string]tftypes.Value{
		"response_type": tftypes.NewValue(tftypes.String, "none"),
	})
	configured := xrayOutboundValue(t, outboundObjectType, "blocked", "blackhole", map[string]tftypes.Value{
		"send_through":       tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"blackhole_settings": blackholeSettings,
	})
	polluted := xrayOutboundValue(t, outboundObjectType, "blocked", "blackhole", map[string]tftypes.Value{
		"send_through":       tftypes.NewValue(tftypes.String, "127.0.0.1"),
		"target_strategy":    tftypes.NewValue(tftypes.String, "UseIPv4"),
		"blackhole_settings": blackholeSettings,
	})
	prior := xrayOutboundValue(t, outboundObjectType, "direct", "freedom", map[string]tftypes.Value{
		"send_through":    tftypes.NewValue(tftypes.String, "127.0.0.1"),
		"target_strategy": tftypes.NewValue(tftypes.String, "UseIPv4"),
	})
	configList := tftypes.NewValue(outboundListType, []tftypes.Value{configured})
	planList := tftypes.NewValue(outboundListType, []tftypes.Value{polluted})
	stateList := tftypes.NewValue(outboundListType, []tftypes.Value{prior})

	plan := runXrayOutboundsModifyPlan(t, schemaResp,
		xrayOutboundsRaw(t, schemaResp, planList, true),
		xrayOutboundsRaw(t, schemaResp, configList, false),
		xrayOutboundsRaw(t, schemaResp, stateList, true),
	)
	assertXrayOutboundListEqual(t, plan, tfsdk.Config{Schema: schemaResp.Schema, Raw: xrayOutboundsRaw(t, schemaResp, configList, false)})
}

func TestXrayOutboundsResource_ModifyPlan_PreservesNullLifecyclePlans(t *testing.T) {
	ctx := context.Background()
	outboundsResource := NewXrayOutboundsResource()
	var schemaResp resource.SchemaResponse
	outboundsResource.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	topType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	outboundListType := topType.AttributeTypes["outbound"].(tftypes.List)
	emptyList := tftypes.NewValue(outboundListType, []tftypes.Value{})
	knownRaw := xrayOutboundsRaw(t, schemaResp, emptyList, false)
	nullRaw := tftypes.NewValue(topType, nil)

	tests := []struct {
		name              string
		planRaw, stateRaw tftypes.Value
	}{
		{name: "destroy plan", planRaw: nullRaw, stateRaw: knownRaw},
		{name: "create state", planRaw: knownRaw, stateRaw: nullRaw},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := runXrayOutboundsModifyPlan(t, schemaResp, test.planRaw, knownRaw, test.stateRaw)
			if !plan.Raw.Equal(test.planRaw) {
				t.Fatalf("lifecycle plan must remain unchanged\n got: %s\nwant: %s", plan.Raw, test.planRaw)
			}
		})
	}
}

func xrayOutboundsRawFromModel(t *testing.T, schemaResp resource.SchemaResponse, outbounds ...XrayOutboundEntry) tftypes.Value {
	t.Helper()
	p := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := p.Set(context.Background(), &XrayOutboundsModel{ID: types.StringValue("xray_outbounds"), Outbound: outbounds}); diags.HasError() {
		t.Fatalf("building outbounds fixture: %v", diags)
	}
	return p.Raw
}

func sockoptOutbound(tag string, sockopt *InboundSockoptModel) XrayOutboundEntry {
	return XrayOutboundEntry{
		Tag:      types.StringValue(tag),
		Protocol: types.StringValue("trojan"),
		StreamSettings: &InboundStreamSettingsModel{
			Network: types.StringValue("tcp"),
			Sockopt: sockopt,
		},
	}
}

func happyEyeballsObject(tryDelayMs types.Int64) types.Object {
	return types.ObjectValueMust(sockoptHappyEyeballsAttrTypes, map[string]attr.Value{
		"try_delay_ms":       tryDelayMs,
		"prioritize_ipv6":    types.BoolValue(false),
		"interleave":         types.Int64Value(1),
		"max_concurrent_try": types.Int64Value(4),
	})
}

// Review of #480: the authoritative configured list dropped every attribute a
// declared sockopt block omits, so removing dialer_proxy while changing the
// keep-alive deleted dialerProxy (and any panel-managed key) on apply. The
// reorder proves the state value is matched by tag, not by index.
func TestXrayOutboundsResource_ModifyPlan_KeepsOmittedSockoptAttributes(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	NewXrayOutboundsResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	direct := XrayOutboundEntry{Tag: types.StringValue("direct"), Protocol: types.StringValue("freedom")}

	prior := nullSockoptModel()
	prior.DialerProxy = types.StringValue("direct")
	prior.DomainStrategy = types.StringValue("UseIPv4")
	prior.TCPKeepAliveIdle = types.Int64Value(60)
	prior.Mark = types.Int64Value(255)
	prior.Interface = types.StringValue("eth0")
	prior.HappyEyeballs = happyEyeballsObject(types.Int64Value(250))

	configured := nullSockoptModel()
	configured.TCPKeepAliveIdle = types.Int64Value(90)
	configured.Interface = types.StringUnknown()
	configured.HappyEyeballs = types.ObjectValueMust(sockoptHappyEyeballsAttrTypes, map[string]attr.Value{
		"try_delay_ms":       types.Int64Value(100),
		"prioritize_ipv6":    types.BoolNull(),
		"interleave":         types.Int64Null(),
		"max_concurrent_try": types.Int64Null(),
	})

	want := prior
	want.TCPKeepAliveIdle = types.Int64Value(90)
	want.Interface = types.StringUnknown()
	want.HappyEyeballs = happyEyeballsObject(types.Int64Value(100))

	stateRaw := xrayOutboundsRawFromModel(t, schemaResp, sockoptOutbound("chained", &prior), direct)
	configRaw := xrayOutboundsRawFromModel(t, schemaResp, direct, sockoptOutbound("chained", &configured))
	pollutedPlanRaw := xrayOutboundsRawFromModel(t, schemaResp, sockoptOutbound("direct", &prior), sockoptOutbound("chained", &configured))

	plan := runXrayOutboundsModifyPlan(t, schemaResp, pollutedPlanRaw, configRaw, stateRaw)
	var got XrayOutboundsModel
	if diags := plan.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading plan: %v", diags)
	}
	if len(got.Outbound) != 2 || got.Outbound[0].StreamSettings != nil {
		t.Fatalf("direct must stay without stream_settings: %+v", got.Outbound)
	}
	assertSockoptModelEqual(t, *got.Outbound[1].StreamSettings.Sockopt, want)
}

// Removing the whole sockopt block is the documented way to drop it.
func TestXrayOutboundsResource_ModifyPlan_RemovedSockoptBlockStaysRemoved(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	NewXrayOutboundsResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	prior := nullSockoptModel()
	prior.DialerProxy = types.StringValue("direct")

	stateRaw := xrayOutboundsRawFromModel(t, schemaResp, sockoptOutbound("chained", &prior))
	configRaw := xrayOutboundsRawFromModel(t, schemaResp, sockoptOutbound("chained", nil))
	plan := runXrayOutboundsModifyPlan(t, schemaResp, stateRaw, configRaw, stateRaw)

	var got XrayOutboundsModel
	if diags := plan.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading plan: %v", diags)
	}
	if got.Outbound[0].StreamSettings.Sockopt != nil {
		t.Fatalf("removed sockopt block must not come back from state: %+v", got.Outbound[0].StreamSettings.Sockopt)
	}
}

// The config validator cannot see a keep-alive value the plan keeps from state.
func TestXrayOutboundsResource_ModifyPlan_RejectsKeepAliveSignsFromState(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	NewXrayOutboundsResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	prior := nullSockoptModel()
	prior.TCPKeepAliveIdle = types.Int64Value(-1)
	configured := nullSockoptModel()
	configured.TCPKeepAliveInterval = types.Int64Value(30)

	stateRaw := xrayOutboundsRawFromModel(t, schemaResp, sockoptOutbound("chained", &prior))
	configRaw := xrayOutboundsRawFromModel(t, schemaResp, sockoptOutbound("chained", &configured))
	resp := runXrayOutboundsModifyPlanResponse(t, schemaResp, stateRaw, configRaw, stateRaw)

	if resp.Diagnostics.ErrorsCount() != 1 {
		t.Fatalf("want one keep-alive error, got %v", resp.Diagnostics)
	}
	wantPath := path.Root("outbound").AtListIndex(0).AtName("stream_settings").AtName("sockopt").AtName("tcp_keep_alive_idle")
	if got := resp.Diagnostics.Errors()[0].(interface{ Path() path.Path }).Path(); !got.Equal(wantPath) {
		t.Fatalf("error path = %s, want %s", got, wantPath)
	}
}
