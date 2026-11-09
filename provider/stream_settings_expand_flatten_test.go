package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// Table-driven round-trip tests for inbound stream settings expand/flatten.
// Each case builds a typed model, calls expand*FromModel, then flatten*ToModel,
// and asserts the fields survive the round-trip.
// ---------------------------------------------------------------------------

func TestExpandFlatten_TCPSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		model *InboundTCPSettingsModel
	}{
		{name: "nil_model", model: nil},
		{
			name: "full",
			model: &InboundTCPSettingsModel{
				AcceptProxyProtocol: types.BoolValue(true),
				HeaderType:          types.StringValue("http"),
			},
		},
		{
			name: "nulls",
			model: &InboundTCPSettingsModel{
				AcceptProxyProtocol: types.BoolNull(),
				HeaderType:          types.StringNull(),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandTCPSettingsFromModel(tc.model)
			if tc.model == nil {
				if expanded != nil {
					t.Fatalf("expected nil, got %v", expanded)
				}
				return
			}
			flat := flattenTCPSettingsToModel(expanded)
			if flat.AcceptProxyProtocol.ValueBool() != tc.model.AcceptProxyProtocol.ValueBool() {
				t.Fatalf("accept_proxy_protocol mismatch: got %v want %v", flat.AcceptProxyProtocol.ValueBool(), tc.model.AcceptProxyProtocol.ValueBool())
			}
		})
	}
}

func TestExpandFlatten_WSSettings_RoundTrip(t *testing.T) {
	headersVal, _ := types.MapValue(types.StringType, map[string]attr.Value{
		"Host": types.StringValue("example.com"),
	})
	cases := []struct {
		name  string
		model *InboundWSSettingsModel
	}{
		{name: "nil_model", model: nil},
		{
			name: "full",
			model: &InboundWSSettingsModel{
				Path:    types.StringValue("/ray"),
				Headers: headersVal,
			},
		},
		{
			name: "path_only",
			model: &InboundWSSettingsModel{
				Path: types.StringValue("/ws"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandWSSettingsFromModel(tc.model)
			if tc.model == nil {
				if expanded != nil {
					t.Fatalf("expected nil, got %v", expanded)
				}
				return
			}
			flat := flattenWSSettingsToModel(expanded)
			if flat.Path.ValueString() != tc.model.Path.ValueString() {
				t.Fatalf("path mismatch: got %q want %q", flat.Path.ValueString(), tc.model.Path.ValueString())
			}
		})
	}
}

func TestExpandFlatten_GRPCSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		model *InboundGRPCSettingsModel
	}{
		{name: "nil_model", model: nil},
		{
			name: "full",
			model: &InboundGRPCSettingsModel{
				ServiceName:         types.StringValue("GunService"),
				MultiMode:           types.BoolValue(true),
				IdleTimeout:         types.Int64Value(60),
				HealthCheckTimeout:  types.Int64Value(20),
				PermitWithoutStream: types.BoolValue(false),
				InitialWindowsSize:  types.Int64Value(0),
			},
		},
		{
			name: "minimal",
			model: &InboundGRPCSettingsModel{
				ServiceName: types.StringValue("grpc"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandGRPCSettingsFromModel(tc.model)
			if tc.model == nil {
				if expanded != nil {
					t.Fatalf("expected nil, got %v", expanded)
				}
				return
			}
			flat := flattenGRPCSettingsToModel(expanded)
			if flat.ServiceName.ValueString() != tc.model.ServiceName.ValueString() {
				t.Fatalf("service_name mismatch: got %q want %q", flat.ServiceName.ValueString(), tc.model.ServiceName.ValueString())
			}
			if flat.MultiMode.ValueBool() != tc.model.MultiMode.ValueBool() {
				t.Fatalf("multi_mode mismatch: got %v want %v", flat.MultiMode.ValueBool(), tc.model.MultiMode.ValueBool())
			}
		})
	}
}

func TestExpandFlatten_HTTPUpgradeSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		model *InboundHTTPUpgradeSettingsModel
	}{
		{name: "nil_model", model: nil},
		{
			name: "full",
			model: &InboundHTTPUpgradeSettingsModel{
				Path: types.StringValue("/upgrade"),
				Host: types.StringValue("cdn.example.com"),
			},
		},
		{
			name: "path_only",
			model: &InboundHTTPUpgradeSettingsModel{
				Path: types.StringValue("/upg"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandHTTPUpgradeSettingsFromModel(tc.model)
			if tc.model == nil {
				if expanded != nil {
					t.Fatalf("expected nil, got %v", expanded)
				}
				return
			}
			flat := flattenHTTPUpgradeSettingsToModel(expanded)
			if flat.Path.ValueString() != tc.model.Path.ValueString() {
				t.Fatalf("path mismatch: got %q want %q", flat.Path.ValueString(), tc.model.Path.ValueString())
			}
			if flat.Host.ValueString() != tc.model.Host.ValueString() {
				t.Fatalf("host mismatch: got %q want %q", flat.Host.ValueString(), tc.model.Host.ValueString())
			}
		})
	}
}

func TestExpandFlatten_XHTTPSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		model *InboundXHTTPSettingsModel
	}{
		{name: "nil_model", model: nil},
		{
			name: "full",
			model: &InboundXHTTPSettingsModel{
				Path:              types.StringValue("/xhttp"),
				Mode:              types.StringValue("auto"),
				NoSSEHeader:       types.BoolValue(true),
				KeepAliveInterval: types.Int64Value(30),
				XPaddingBytes:     types.StringValue("100-1000"),
				XPaddingObfsMode:  types.BoolValue(false),
				XPaddingKey:       types.StringValue("key1"),
				XPaddingHeader:    types.StringValue("X-Padding"),
				XPaddingPlacement: types.StringValue("header"),
				XPaddingMethod:    types.StringValue(" aes-256-gcm"),
			},
		},
		{
			name: "minimal",
			model: &InboundXHTTPSettingsModel{
				Mode: types.StringValue("packet-up"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandXHTTPSettingsFromModel(tc.model)
			if tc.model == nil {
				if expanded != nil {
					t.Fatalf("expected nil, got %v", expanded)
				}
				return
			}
			flat := flattenXHTTPSettingsToModel(expanded)
			if flat.Mode.ValueString() != tc.model.Mode.ValueString() {
				t.Fatalf("mode mismatch: got %q want %q", flat.Mode.ValueString(), tc.model.Mode.ValueString())
			}
			if flat.NoSSEHeader.ValueBool() != tc.model.NoSSEHeader.ValueBool() {
				t.Fatalf("no_sse_header mismatch: got %v want %v", flat.NoSSEHeader.ValueBool(), tc.model.NoSSEHeader.ValueBool())
			}
			if flat.KeepAliveInterval.ValueInt64() != tc.model.KeepAliveInterval.ValueInt64() {
				t.Fatalf("keep_alive_interval mismatch: got %d want %d", flat.KeepAliveInterval.ValueInt64(), tc.model.KeepAliveInterval.ValueInt64())
			}
		})
	}
}

func TestExpandFlatten_KCPSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		model *InboundKCPSettingsModel
	}{
		{name: "nil_model", model: nil},
		{
			name: "full",
			model: &InboundKCPSettingsModel{
				MTU:              types.Int64Value(1350),
				TTI:              types.Int64Value(50),
				UplinkCapacity:   types.Int64Value(5),
				DownlinkCapacity: types.Int64Value(20),
				CwndMultiplier:   types.Int64Value(2),
				MaxSendingWindow: types.Int64Value(1024),
				HeaderType:       types.StringValue("srtp"),
			},
		},
		{
			name:  "all_null",
			model: &InboundKCPSettingsModel{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandKCPSettingsFromModel(tc.model)
			if tc.model == nil {
				if expanded != nil {
					t.Fatalf("expected nil, got %v", expanded)
				}
				return
			}
			flat := flattenKCPSettingsToModel(expanded)
			if flat.MTU.ValueInt64() != tc.model.MTU.ValueInt64() {
				t.Fatalf("mtu mismatch: got %d want %d", flat.MTU.ValueInt64(), tc.model.MTU.ValueInt64())
			}
			if flat.HeaderType.ValueString() != tc.model.HeaderType.ValueString() {
				t.Fatalf("header_type mismatch: got %q want %q", flat.HeaderType.ValueString(), tc.model.HeaderType.ValueString())
			}
		})
	}
}

func TestExpandFlatten_HysteriaStreamSettings_RoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		model *InboundHysteriaStreamSettingsModel
	}{
		{name: "nil_model", model: nil},
		{
			name: "full",
			model: &InboundHysteriaStreamSettingsModel{
				Protocol:       types.StringValue("udp"),
				Version:        types.Int64Value(2),
				Auth:           types.StringValue("password123"),
				UDPIdleTimeout: types.Int64Value(30),
			},
		},
		{
			name:  "all_null",
			model: &InboundHysteriaStreamSettingsModel{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandHysteriaStreamSettingsFromModel(tc.model)
			if tc.model == nil {
				if expanded != nil {
					t.Fatalf("expected nil, got %v", expanded)
				}
				return
			}
			flat := flattenHysteriaStreamSettingsToModel(expanded)
			if flat.Protocol.ValueString() != tc.model.Protocol.ValueString() {
				t.Fatalf("protocol mismatch: got %q want %q", flat.Protocol.ValueString(), tc.model.Protocol.ValueString())
			}
			if flat.Version.ValueInt64() != tc.model.Version.ValueInt64() {
				t.Fatalf("version mismatch: got %d want %d", flat.Version.ValueInt64(), tc.model.Version.ValueInt64())
			}
		})
	}
}

func TestExpandFlatten_Sockopt_RoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		model *InboundSockoptModel
	}{
		{name: "nil_model", model: nil},
		{
			name: "full",
			model: &InboundSockoptModel{
				Mark:                 types.Int64Value(255),
				TCPKeepAliveInterval: types.Int64Value(15),
				TCPNoDelay:           types.BoolValue(true),
				TFOEnable:            types.BoolValue(true),
				Tproxy:               types.StringValue("redirect"),
				DomainStrategy:       types.StringValue("UseIP"),
			},
		},
		{
			name:  "all_null",
			model: &InboundSockoptModel{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandSockoptFromModel(tc.model)
			if tc.model == nil {
				if expanded != nil {
					t.Fatalf("expected nil, got %v", expanded)
				}
				return
			}
			flat := flattenSockoptToModel(expanded)
			if flat.Tproxy.ValueString() != tc.model.Tproxy.ValueString() {
				t.Fatalf("tproxy mismatch: got %q want %q", flat.Tproxy.ValueString(), tc.model.Tproxy.ValueString())
			}
			if flat.DomainStrategy.ValueString() != tc.model.DomainStrategy.ValueString() {
				t.Fatalf("domain_strategy mismatch: got %q want %q", flat.DomainStrategy.ValueString(), tc.model.DomainStrategy.ValueString())
			}
		})
	}
}

func TestExpandFlatten_RealitySettings_RoundTrip(t *testing.T) {
	serverNames, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("example.com"),
		types.StringValue("www.example.com"),
	})
	shortIDs, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("abc123"),
		types.StringValue(""),
	})
	cases := []struct {
		name  string
		model *InboundRealitySettingsModel
	}{
		{name: "nil_model", model: nil},
		{
			name: "full",
			model: &InboundRealitySettingsModel{
				Show:        types.BoolValue(true),
				Xver:        types.Int64Value(1),
				Target:      types.StringValue("google.com:443"),
				ServerNames: serverNames,
				PrivateKey:  types.StringValue("priv"),
				ShortIDs:    shortIDs,
			},
		},
		{
			name:  "all_null",
			model: &InboundRealitySettingsModel{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandRealitySettingsFromModel(tc.model)
			if tc.model == nil {
				if expanded != nil {
					t.Fatalf("expected nil, got %v", expanded)
				}
				return
			}
			flat := flattenRealitySettingsToModel(expanded)
			if flat.Target.ValueString() != tc.model.Target.ValueString() {
				t.Fatalf("target mismatch: got %q want %q", flat.Target.ValueString(), tc.model.Target.ValueString())
			}
			if flat.Xver.ValueInt64() != tc.model.Xver.ValueInt64() {
				t.Fatalf("xver mismatch: got %d want %d", flat.Xver.ValueInt64(), tc.model.Xver.ValueInt64())
			}
		})
	}
}

func TestExpandFlatten_ExternalProxy_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		list []InboundExternalProxyModel
	}{
		{name: "empty", list: nil},
		{
			name: "single",
			list: []InboundExternalProxyModel{
				{Dest: types.StringValue("1.2.3.4"), Port: types.Int64Value(443), Remark: types.StringValue("proxy1"), ForceTLS: types.StringValue("tls")},
			},
		},
		{
			name: "multiple_partial",
			list: []InboundExternalProxyModel{
				{Dest: types.StringValue("a.com"), Port: types.Int64Value(8080)},
				{Dest: types.StringValue("b.com"), Port: types.Int64Value(8443), Remark: types.StringValue("b")},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expanded := expandExternalProxyFromModel(tc.list)
			flat := flattenExternalProxyToModel(expanded)
			if len(flat) != len(tc.list) {
				t.Fatalf("count mismatch: got %d want %d", len(flat), len(tc.list))
			}
			for i, ep := range tc.list {
				if flat[i].Dest.ValueString() != ep.Dest.ValueString() {
					t.Fatalf("dest[%d] mismatch: got %q want %q", i, flat[i].Dest.ValueString(), ep.Dest.ValueString())
				}
			}
		})
	}
}

func TestExpandFlatten_ExternalProxy_EmptyRemarkWireRoundTrip(t *testing.T) {
	model := &InboundStreamSettingsModel{
		ExternalProxy: []InboundExternalProxyModel{{
			Dest: types.StringValue("proxy.example.com"), Port: types.Int64Value(443), Remark: types.StringValue(""),
		}},
	}
	expanded := expandStreamSettingsFromModel(model)
	wire := expandExternalProxy(expanded["external_proxy"].([]any))
	if len(wire) != 1 {
		t.Fatalf("external proxy count = %d, want 1", len(wire))
	}
	entry := wire[0].(map[string]any)
	if remark, ok := entry["remark"]; !ok || remark != "" {
		t.Fatalf("wire remark = %#v (present %t), want an explicit empty string", remark, ok)
	}
	flat := flattenExternalProxyToModel(wire)
	if len(flat) != 1 || flat[0].Remark.IsNull() || flat[0].Remark.IsUnknown() || flat[0].Remark.ValueString() != "" {
		t.Fatalf("empty remark did not round-trip: %#v", flat)
	}
}

func TestExpandFlatten_ExternalProxy_OmittedRemarkWireRoundTrip(t *testing.T) {
	model := &InboundStreamSettingsModel{
		ExternalProxy: []InboundExternalProxyModel{{
			Dest: types.StringValue("proxy.example.com"), Port: types.Int64Value(443), Remark: types.StringNull(),
		}},
	}
	expanded := expandStreamSettingsFromModel(model)
	wire := expandExternalProxy(expanded["external_proxy"].([]any))
	entry := wire[0].(map[string]any)
	if remark, ok := entry["remark"]; ok {
		t.Fatalf("omitted remark reached the wire as %#v", remark)
	}
	flat := flattenExternalProxyToModel(wire)
	if len(flat) != 1 || !flat[0].Remark.IsNull() {
		t.Fatalf("omitted remark did not stay null: %#v", flat)
	}
}

func TestExpandFlatten_StreamSettingsModel_Nil(t *testing.T) {
	if got := expandStreamSettingsFromModel(nil); got != nil {
		t.Fatalf("expected nil for nil model, got %v", got)
	}
}

func TestExpandFlatten_StreamSettingsModel_EmptyData(t *testing.T) {
	flat := flattenStreamSettingsToModel(map[string]any{})
	if flat == nil {
		t.Fatal("expected non-nil model for empty data")
	}
	if !flat.Network.IsNull() {
		t.Fatalf("expected null network for empty data, got %v", flat.Network)
	}
}

func TestExpandFlatten_StreamSettingsModel_RoundTrip(t *testing.T) {
	model := &InboundStreamSettingsModel{
		Network:  types.StringValue("ws"),
		Security: types.StringValue("none"),
		TCPSettings: &InboundTCPSettingsModel{
			AcceptProxyProtocol: types.BoolValue(true),
			HeaderType:          types.StringValue("none"),
		},
		WSSettings: &InboundWSSettingsModel{
			Path: types.StringValue("/v2ray"),
		},
		Sockopt: &InboundSockoptModel{
			TCPNoDelay: types.BoolValue(true),
			Tproxy:     types.StringValue("tproxy"),
		},
	}
	expanded := expandStreamSettingsFromModel(model)
	if expanded == nil {
		t.Fatal("expected non-nil expanded map")
	}
	if expanded["network"] != "ws" {
		t.Fatalf("unexpected network: %v", expanded["network"])
	}
	flat := flattenStreamSettingsToModel(expanded)
	if flat == nil {
		t.Fatal("expected non-nil flattened model")
	}
	if flat.Network.ValueString() != "ws" {
		t.Fatalf("network mismatch: got %q want %q", flat.Network.ValueString(), "ws")
	}
	if flat.TCPSettings == nil {
		t.Fatal("expected TCPSettings to be set")
	}
	if flat.WSSettings == nil {
		t.Fatal("expected WSSettings to be set")
	}
	if flat.Sockopt == nil {
		t.Fatal("expected Sockopt to be set")
	}
}

func TestExpandFlatten_RealityInnerSettings_RoundTrip(t *testing.T) {
	obj, _ := types.ObjectValue(realityInnerSettingsAttrTypes, map[string]attr.Value{
		"public_key":     types.StringValue("pub123"),
		"fingerprint":    types.StringValue("chrome"),
		"server_name":    types.StringValue("sni.example.com"),
		"spider_x":       types.StringValue(""),
		"mldsa65_verify": types.StringValue(""),
	})
	expanded := expandRealityInnerSettingsFromObject(obj)
	if expanded["public_key"] != "pub123" {
		t.Fatalf("unexpected public_key: %v", expanded["public_key"])
	}
	if expanded["fingerprint"] != "chrome" {
		t.Fatalf("unexpected fingerprint: %v", expanded["fingerprint"])
	}
	flatObj := flattenRealityInnerSettingsToObject(expanded)
	if flatObj.Attributes()["public_key"].(types.String).ValueString() != "pub123" {
		t.Fatalf("unexpected public_key in flat obj")
	}
}

func TestExpandRealityInnerSettingsFromObject_Null(t *testing.T) {
	if got := expandRealityInnerSettingsFromObject(types.ObjectNull(realityInnerSettingsAttrTypes)); got != nil {
		t.Fatalf("expected nil for null object, got %v", got)
	}
}

func TestFlattenRealityInnerSettingsToObject_Empty(t *testing.T) {
	obj := flattenRealityInnerSettingsToObject(map[string]any{})
	if !obj.IsNull() {
		t.Fatalf("expected null object for empty data, got %v", obj)
	}
}
