package provider

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func nullSockoptModel() InboundSockoptModel {
	return InboundSockoptModel{
		Mark:                 types.Int64Null(),
		TCPKeepAliveInterval: types.Int64Null(),
		TCPKeepAliveIdle:     types.Int64Null(),
		TCPUserTimeout:       types.Int64Null(),
		TCPMaxSeg:            types.Int64Null(),
		TCPWindowClamp:       types.Int64Null(),
		TCPCongestion:        types.StringNull(),
		TCPNoDelay:           types.BoolNull(),
		TFOEnable:            types.BoolNull(),
		TCPMptcp:             types.BoolNull(),
		Tproxy:               types.StringNull(),
		DomainStrategy:       types.StringNull(),
		AcceptProxyProtocol:  types.BoolNull(),
		TrustedXForwardedFor: types.ListNull(types.StringType),
		V6Only:               types.BoolNull(),
		Interface:            types.StringNull(),
		DialerProxy:          types.StringNull(),
		AddressPortStrategy:  types.StringNull(),
		Penetrate:            types.BoolNull(),
		HappyEyeballs:        types.ObjectNull(sockoptHappyEyeballsAttrTypes),
		CustomSockopt:        types.ListNull(types.ObjectType{AttrTypes: sockoptCustomAttrTypes}),
	}
}

func customSockoptList(entries ...map[string]string) types.List {
	elems := make([]attr.Value, 0, len(entries))
	for _, e := range entries {
		attrs := map[string]attr.Value{}
		for key := range sockoptCustomAttrTypes {
			if v, ok := e[key]; ok {
				attrs[key] = types.StringValue(v)
			} else {
				attrs[key] = types.StringNull()
			}
		}
		elems = append(elems, types.ObjectValueMust(sockoptCustomAttrTypes, attrs))
	}
	return types.ListValueMust(types.ObjectType{AttrTypes: sockoptCustomAttrTypes}, elems)
}

// sockoptWireRoundTrip follows the inbound and xray_outbounds resources:
// model -> streamSettings JSON -> model.
func sockoptWireRoundTrip(t *testing.T, in InboundSockoptModel) (map[string]any, *InboundSockoptModel) {
	t.Helper()
	streamJSON := buildStreamSettingsJSON(expandStreamSettingsFromModel(&InboundStreamSettingsModel{
		Network: types.StringValue("ws"),
		Sockopt: &in,
	}))
	var payload map[string]any
	if err := json.Unmarshal([]byte(streamJSON), &payload); err != nil {
		t.Fatalf("stream JSON %q: %v", streamJSON, err)
	}
	wire, _ := payload["sockopt"].(map[string]any)

	flat, err := flattenStreamSettingsToMap(streamJSON)
	if err != nil {
		t.Fatalf("flattenStreamSettingsToMap: %v", err)
	}
	out := flattenStreamSettingsToModel(flat).Sockopt
	if out == nil {
		t.Fatalf("sockopt lost on read-back of %s", streamJSON)
	}
	return wire, out
}

func assertSockoptModelEqual(t *testing.T, got, want InboundSockoptModel) {
	t.Helper()
	gv, wv := reflect.ValueOf(got), reflect.ValueOf(want)
	for i := 0; i < gv.NumField(); i++ {
		name := gv.Type().Field(i).Tag.Get("tfsdk")
		g := gv.Field(i).Interface().(attr.Value)
		w := wv.Field(i).Interface().(attr.Value)
		if !g.Equal(w) {
			t.Errorf("%s: got %s, want %s", name, g, w)
		}
	}
}

func assertWireEqual(t *testing.T, got map[string]any, wantJSON string) {
	t.Helper()
	var want map[string]any
	if err := json.Unmarshal([]byte(wantJSON), &want); err != nil {
		t.Fatalf("bad expected JSON: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		gotJSON, _ := json.Marshal(got)
		t.Errorf("sockopt wire mismatch\n got: %s\nwant: %s", gotJSON, wantJSON)
	}
}

// Every attribute must reach the JSON under the key xray-core (for the
// case-insensitive aliases, the 3x-ui panel) reads, and come back unchanged.
func TestSockoptWireRoundTrip_AllFields(t *testing.T) {
	in := nullSockoptModel()
	in.Mark = types.Int64Value(255)
	in.TCPKeepAliveInterval = types.Int64Value(30)
	in.TCPKeepAliveIdle = types.Int64Value(300)
	in.TCPUserTimeout = types.Int64Value(10000)
	in.TCPMaxSeg = types.Int64Value(1440)
	in.TCPWindowClamp = types.Int64Value(600)
	in.TCPCongestion = types.StringValue("bbr")
	in.TCPNoDelay = types.BoolValue(true)
	in.TFOEnable = types.BoolValue(true)
	in.TCPMptcp = types.BoolValue(true)
	in.Tproxy = types.StringValue("tproxy")
	in.DomainStrategy = types.StringValue("UseIPv4")
	in.AcceptProxyProtocol = types.BoolValue(true)
	in.TrustedXForwardedFor = types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("X-Forwarded-For"), types.StringValue("CF-Connecting-IP"),
	})
	in.V6Only = types.BoolValue(true)
	in.Interface = types.StringValue("eth0")
	in.DialerProxy = types.StringValue("chain-out")
	in.AddressPortStrategy = types.StringValue("SrvPortOnly")
	in.Penetrate = types.BoolValue(true)
	in.HappyEyeballs = types.ObjectValueMust(sockoptHappyEyeballsAttrTypes, map[string]attr.Value{
		"try_delay_ms":       types.Int64Value(250),
		"prioritize_ipv6":    types.BoolValue(true),
		"interleave":         types.Int64Value(2),
		"max_concurrent_try": types.Int64Value(4),
	})
	in.CustomSockopt = customSockoptList(
		map[string]string{"system": "linux", "network": "tcp", "level": "6", "opt": "13", "type": "str", "value": "bbr"},
		map[string]string{"opt": "4", "type": "int", "value": "60"},
	)

	wire, out := sockoptWireRoundTrip(t, in)
	assertWireEqual(t, wire, `{
		"mark": 255, "tcpKeepAliveInterval": 30, "tcpKeepAliveIdle": 300,
		"tcpUserTimeout": 10000, "tcpMaxSeg": 1440, "tcpWindowClamp": 600,
		"tcpcongestion": "bbr", "tcpNoDelay": true, "tcpFastOpen": true, "tcpMptcp": true,
		"tproxy": "tproxy", "domainStrategy": "UseIPv4", "acceptProxyProtocol": true,
		"trustedXForwardedFor": ["X-Forwarded-For", "CF-Connecting-IP"],
		"v6only": true, "interface": "eth0", "dialerProxy": "chain-out",
		"addressPortStrategy": "SrvPortOnly", "penetrate": true,
		"happyEyeballs": {"tryDelayMs": 250, "prioritizeIPv6": true, "interleave": 2, "maxConcurrentTry": 4},
		"customSockopt": [
			{"system": "linux", "network": "tcp", "level": "6", "opt": "13", "type": "str", "value": "bbr"},
			{"opt": "4", "type": "int", "value": "60"}
		]
	}`)
	assertSockoptModelEqual(t, *out, in)
}

// An Optional+Computed attribute cannot be cleared by omission, so 0 / false /
// "" / [] must survive the round-trip; reading one back as null fails the apply
// with "Provider produced inconsistent result".
func TestSockoptWireRoundTrip_ClearValues(t *testing.T) {
	in := nullSockoptModel()
	in.Mark = types.Int64Value(0)
	in.TCPKeepAliveInterval = types.Int64Value(0)
	in.TCPKeepAliveIdle = types.Int64Value(-1)
	in.TCPMaxSeg = types.Int64Value(0)
	in.TCPCongestion = types.StringValue("")
	in.TFOEnable = types.BoolValue(false)
	in.AcceptProxyProtocol = types.BoolValue(false)
	in.V6Only = types.BoolValue(false)
	in.Interface = types.StringValue("")
	in.DialerProxy = types.StringValue("")
	in.AddressPortStrategy = types.StringValue("none")
	in.TrustedXForwardedFor = types.ListValueMust(types.StringType, []attr.Value{})
	in.CustomSockopt = customSockoptList()
	in.HappyEyeballs = types.ObjectValueMust(sockoptHappyEyeballsAttrTypes, map[string]attr.Value{
		"try_delay_ms":       types.Int64Value(0),
		"prioritize_ipv6":    types.BoolNull(),
		"interleave":         types.Int64Null(),
		"max_concurrent_try": types.Int64Null(),
	})

	wire, out := sockoptWireRoundTrip(t, in)
	assertWireEqual(t, wire, `{
		"mark": 0, "tcpKeepAliveInterval": 0, "tcpKeepAliveIdle": -1, "tcpMaxSeg": 0,
		"tcpcongestion": "", "tcpFastOpen": false, "acceptProxyProtocol": false, "v6only": false,
		"interface": "", "dialerProxy": "", "addressPortStrategy": "none",
		"trustedXForwardedFor": [], "customSockopt": [], "happyEyeballs": {"tryDelayMs": 0}
	}`)
	assertSockoptModelEqual(t, *out, in)
}

// A declared but empty happy_eyeballs must not read back as null.
func TestSockoptWireRoundTrip_EmptyHappyEyeballs(t *testing.T) {
	in := nullSockoptModel()
	in.HappyEyeballs = types.ObjectValueMust(sockoptHappyEyeballsAttrTypes, map[string]attr.Value{
		"try_delay_ms":       types.Int64Null(),
		"prioritize_ipv6":    types.BoolNull(),
		"interleave":         types.Int64Null(),
		"max_concurrent_try": types.Int64Null(),
	})
	wire, out := sockoptWireRoundTrip(t, in)
	assertWireEqual(t, wire, `{"happyEyeballs": {}}`)
	assertSockoptModelEqual(t, *out, in)
}

// xray-core decodes sockopt case-insensitively, so a stored config may use the
// panel form spelling (V6Only, tcpcongestion) or xray's (v6only, tcpCongestion).
func TestFlattenSockopt_AcceptsBothSpellings(t *testing.T) {
	cases := []struct {
		name       string
		stored     string
		v6Only     bool
		congestion string
	}{
		{"panel form", `{"V6Only": true, "tcpcongestion": "cubic"}`, true, "cubic"},
		{"xray canonical", `{"v6only": true, "tcpCongestion": "reno"}`, true, "reno"},
		{"both, xray spelling wins", `{"v6only": false, "V6Only": true, "tcpCongestion": "bbr", "tcpcongestion": "cubic"}`, false, "bbr"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flat, err := flattenStreamSettingsToMap(`{"network": "tcp", "sockopt": ` + tc.stored + `}`)
			if err != nil {
				t.Fatal(err)
			}
			got := flattenStreamSettingsToModel(flat).Sockopt
			if got.V6Only.ValueBool() != tc.v6Only || got.V6Only.IsNull() {
				t.Errorf("v6_only = %s, want %t", got.V6Only, tc.v6Only)
			}
			if got.TCPCongestion.ValueString() != tc.congestion {
				t.Errorf("tcp_congestion = %s, want %q", got.TCPCongestion, tc.congestion)
			}

			wire, _ := sockoptWireRoundTrip(t, *got)
			assertWireEqual(t, wire, `{"v6only": `+map[bool]string{true: "true", false: "false"}[tc.v6Only]+`, "tcpcongestion": "`+tc.congestion+`"}`)
		})
	}
}

// The 3x-ui form schema allows numbers in customSockopt; xray-core only strings.
func TestFlattenSockopt_CustomSockoptNumericFields(t *testing.T) {
	flat, err := flattenStreamSettingsToMap(`{"network": "tcp", "sockopt": {"customSockopt": [{"level": 6, "opt": 4, "type": "int", "value": 60}]}}`)
	if err != nil {
		t.Fatal(err)
	}
	got := flattenStreamSettingsToModel(flat).Sockopt
	want := customSockoptList(map[string]string{"level": "6", "opt": "4", "type": "int", "value": "60"})
	if !got.CustomSockopt.Equal(want) {
		t.Fatalf("custom_sockopt = %s, want %s", got.CustomSockopt, want)
	}
	wire, _ := sockoptWireRoundTrip(t, *got)
	assertWireEqual(t, wire, `{"customSockopt": [{"level": "6", "opt": "4", "type": "int", "value": "60"}]}`)
}

func TestCustomSockoptIntValueValidator(t *testing.T) {
	cases := []struct {
		name    string
		typ     attr.Value
		value   attr.Value
		wantErr bool
	}{
		{"int with decimal value", types.StringValue("int"), types.StringValue("-15"), false},
		{"int with non-numeric value", types.StringValue("int"), types.StringValue("bbr"), true},
		{"int with hex value", types.StringValue("int"), types.StringValue("0x10"), true},
		{"str with text value", types.StringValue("str"), types.StringValue("bbr"), false},
		{"unknown type", types.StringUnknown(), types.StringValue("bbr"), false},
		{"unknown value", types.StringValue("int"), types.StringUnknown(), false},
	}
	v := customSockoptIntValueValidator{}
	if v.Description(t.Context()) == "" || v.MarkdownDescription(t.Context()) == "" {
		t.Fatal("validator description must not be empty")
	}
	var nullResp validator.ObjectResponse
	v.ValidateObject(t.Context(), validator.ObjectRequest{ConfigValue: types.ObjectNull(sockoptCustomAttrTypes)}, &nullResp)
	if nullResp.Diagnostics.HasError() {
		t.Fatalf("null object must not be validated: %v", nullResp.Diagnostics)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			attrs := map[string]attr.Value{}
			for key := range sockoptCustomAttrTypes {
				attrs[key] = types.StringNull()
			}
			attrs["opt"] = types.StringValue("4")
			attrs["type"] = tc.typ
			attrs["value"] = tc.value
			req := validator.ObjectRequest{
				Path:        path.Root("custom_sockopt").AtListIndex(0),
				ConfigValue: types.ObjectValueMust(sockoptCustomAttrTypes, attrs),
			}
			var resp validator.ObjectResponse
			customSockoptIntValueValidator{}.ValidateObject(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Fatalf("HasError = %t, want %t: %v", resp.Diagnostics.HasError(), tc.wantErr, resp.Diagnostics)
			}
			if tc.wantErr {
				wantPath := path.Root("custom_sockopt").AtListIndex(0).AtName("value")
				if got := resp.Diagnostics.Errors()[0].(interface{ Path() path.Path }).Path(); !got.Equal(wantPath) {
					t.Fatalf("error path = %s, want %s", got, wantPath)
				}
			}
		})
	}
}

func TestSockoptDecimalIntValidator(t *testing.T) {
	cases := map[string]bool{"6": false, "-1": false, "SOL_SOCKET": true, "0x6": true, "": true}
	for value, wantErr := range cases {
		var resp validator.StringResponse
		sockoptDecimalIntValidator().ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("level"),
			ConfigValue: types.StringValue(value),
		}, &resp)
		if resp.Diagnostics.HasError() != wantErr {
			t.Errorf("%q: HasError = %t, want %t", value, resp.Diagnostics.HasError(), wantErr)
		}
	}
}

// xray rejects an empty customSockopt entry with "No opt!" and fails the
// listener, so unknown or malformed elements must not reach the JSON.
func TestSockoptCustomSockopt_SkipsUnusableEntries(t *testing.T) {
	elemType := types.ObjectType{AttrTypes: sockoptCustomAttrTypes}
	known := customSockoptList(map[string]string{"opt": "4", "type": "int", "value": "60"}).Elements()[0]
	in := nullSockoptModel()
	in.CustomSockopt = types.ListValueMust(elemType, []attr.Value{types.ObjectUnknown(sockoptCustomAttrTypes), known})
	wire, _ := sockoptWireRoundTrip(t, in)
	assertWireEqual(t, wire, `{"customSockopt": [{"opt": "4", "type": "int", "value": "60"}]}`)

	flat, err := flattenStreamSettingsToMap(`{"network": "tcp", "sockopt": {"customSockopt": ["junk", {"opt": "4", "type": "int", "value": "60"}]}}`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := flattenStreamSettingsToModel(flat).Sockopt.CustomSockopt, customSockoptList(map[string]string{"opt": "4", "type": "int", "value": "60"}); !got.Equal(want) {
		t.Fatalf("custom_sockopt = %s, want %s", got, want)
	}
}
