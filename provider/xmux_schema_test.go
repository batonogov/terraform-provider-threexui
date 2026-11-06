package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// xmuxStreamJSON is the write path of both resources: model -> streamSettings JSON.
func xmuxStreamJSON(xmux *InboundXmuxModel) string {
	return buildStreamSettingsJSON(expandStreamSettingsFromModel(&InboundStreamSettingsModel{
		Network:       types.StringValue("xhttp"),
		XHTTPSettings: &InboundXHTTPSettingsModel{Path: types.StringValue("/x"), Xmux: xmux},
	}))
}

// xmuxFromStreamJSON is the read path of both resources: streamSettings JSON -> model.
func xmuxFromStreamJSON(t *testing.T, streamJSON string) *InboundXmuxModel {
	t.Helper()
	flat, err := flattenStreamSettingsToMap(streamJSON)
	if err != nil {
		t.Fatalf("flattenStreamSettingsToMap(%s): %v", streamJSON, err)
	}
	xhttp := flattenStreamSettingsToModel(flat).XHTTPSettings
	if xhttp == nil {
		t.Fatalf("xhttp_settings lost on read-back of %s", streamJSON)
	}
	return xhttp.Xmux
}

func assertXmuxModelEqual(t *testing.T, got, want *InboundXmuxModel) {
	t.Helper()
	if got == nil || want == nil {
		if got != want {
			t.Fatalf("xmux = %+v, want %+v", got, want)
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

// Ranges must reach the wire as strings (the panel form rejects numbers for
// three of them), the keep-alive as a number, and an empty block as {}: a
// missing key would bring the global subJsonMux back for the inbound.
func TestXmuxWireRoundTrip(t *testing.T) {
	cases := []struct {
		name     string
		in       *InboundXmuxModel
		wantWire string // empty: no xmux key at all
	}{
		{
			name: "connection pool",
			in: &InboundXmuxModel{
				MaxConnections:   types.StringValue("2"),
				HMaxRequestTimes: types.StringValue("6000-9000"),
				HMaxReusableSecs: types.StringValue("1800-3000"),
			},
			wantWire: `{"maxConnections": "2", "hMaxRequestTimes": "6000-9000", "hMaxReusableSecs": "1800-3000"}`,
		},
		{
			name: "concurrency with every limit",
			in: &InboundXmuxModel{
				MaxConcurrency:   types.StringValue("16-32"),
				CMaxReuseTimes:   types.StringValue("64-128"),
				HMaxRequestTimes: types.StringValue("600-900"),
				HMaxReusableSecs: types.StringValue("1800-3000"),
				HKeepAlivePeriod: types.Int64Value(30),
			},
			wantWire: `{"maxConcurrency": "16-32", "cMaxReuseTimes": "64-128", "hMaxRequestTimes": "600-900",
				"hMaxReusableSecs": "1800-3000", "hKeepAlivePeriod": 30}`,
		},
		{
			name:     "negative keep-alive",
			in:       &InboundXmuxModel{HKeepAlivePeriod: types.Int64Value(-1)},
			wantWire: `{"hKeepAlivePeriod": -1}`,
		},
		{name: "empty block", in: &InboundXmuxModel{}, wantWire: `{}`},
		{name: "no block", in: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			streamJSON := xmuxStreamJSON(tc.in)
			var stream struct {
				XHTTPSettings map[string]any `json:"xhttpSettings"`
			}
			if err := json.Unmarshal([]byte(streamJSON), &stream); err != nil {
				t.Fatalf("stream JSON %s: %v", streamJSON, err)
			}
			got, present := stream.XHTTPSettings["xmux"]
			if tc.wantWire == "" {
				if present {
					t.Fatalf("xmux must not be written without a block: %s", streamJSON)
				}
			} else {
				var want any
				if err := json.Unmarshal([]byte(tc.wantWire), &want); err != nil {
					t.Fatalf("bad expected JSON: %v", err)
				}
				if !reflect.DeepEqual(got, want) {
					gotJSON, _ := json.Marshal(got)
					t.Fatalf("wire xmux = %s, want %s", gotJSON, tc.wantWire)
				}
			}
			assertXmuxModelEqual(t, xmuxFromStreamJSON(t, streamJSON), tc.in)
		})
	}
}

// What the panel stores after its form saved the inbound: missing fields are
// backfilled with 0 / "", and maxConnections may be a JSON number. None of it
// may surface as a diff against a configuration that leaves those fields out.
func TestXmuxReadsPanelFormValues(t *testing.T) {
	cases := []struct {
		name   string
		stored string
		want   *InboundXmuxModel
	}{
		{
			name: "form save backfills zeros",
			stored: `{"maxConcurrency": "", "maxConnections": 2, "cMaxReuseTimes": 0,
				"hMaxRequestTimes": "6000-9000", "hMaxReusableSecs": "1800-3000", "hKeepAlivePeriod": 0}`,
			want: &InboundXmuxModel{
				MaxConnections:   types.StringValue("2"),
				HMaxRequestTimes: types.StringValue("6000-9000"),
				HMaxReusableSecs: types.StringValue("1800-3000"),
			},
		},
		{
			name:   "zero spellings",
			stored: `{"maxConnections": "0", "cMaxReuseTimes": "0-0", "hMaxRequestTimes": 0, "hMaxReusableSecs": ""}`,
			want:   &InboundXmuxModel{},
		},
		{
			name:   "numbers keep every digit",
			stored: `{"hMaxRequestTimes": 1000000, "hKeepAlivePeriod": 45}`,
			want:   &InboundXmuxModel{HMaxRequestTimes: types.StringValue("1000000"), HKeepAlivePeriod: types.Int64Value(45)},
		},
		{
			// Not canonical, so the plan shows it instead of hiding it.
			name:   "spaced range",
			stored: `{"maxConnections": "2 - 4"}`,
			want:   &InboundXmuxModel{MaxConnections: types.StringValue("2 - 4")},
		},
		{
			// xray-core cannot decode either into its int64 field.
			name:   "keep-alive xray-core rejects",
			stored: `{"maxConnections": "2", "hKeepAlivePeriod": "30", "cMaxReuseTimes": true}`,
			want:   &InboundXmuxModel{MaxConnections: types.StringValue("2")},
		},
		{
			name:   "fractional keep-alive",
			stored: `{"hKeepAlivePeriod": 1.5}`,
			want:   &InboundXmuxModel{},
		},
		{name: "empty object", stored: `{}`, want: &InboundXmuxModel{}},
		{name: "null", stored: `null`, want: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stream := fmt.Sprintf(`{"network": "xhttp", "xhttpSettings": {"path": "/x", "xmux": %s}}`, tc.stored)
			assertXmuxModelEqual(t, xmuxFromStreamJSON(t, stream), tc.want)
		})
	}
}

// ---------------------------------------------------------------------------
// Plan-time validation, through the framework as Terraform runs it
// ---------------------------------------------------------------------------

func validateResourceConfigErrors(t *testing.T, typeName string, config tfsdk.Plan) []*tfprotov6.Diagnostic {
	t.Helper()
	server, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatalf("provider server: %v", err)
	}
	value, err := tfprotov6.NewDynamicValue(config.Schema.Type().TerraformType(context.Background()), config.Raw)
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	resp, err := server.ValidateResourceConfig(context.Background(), &tfprotov6.ValidateResourceConfigRequest{
		TypeName: typeName,
		Config:   &value,
	})
	if err != nil {
		t.Fatalf("ValidateResourceConfig: %v", err)
	}
	var errs []*tfprotov6.Diagnostic
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			errs = append(errs, d)
		}
	}
	return errs
}

func xmuxStreamSettings(xmux *InboundXmuxModel) *InboundStreamSettingsModel {
	return &InboundStreamSettingsModel{
		Network:       types.StringValue("xhttp"),
		XHTTPSettings: &InboundXHTTPSettingsModel{Path: types.StringValue("/x"), Xmux: xmux},
	}
}

func inboundXmuxConfig(t *testing.T, xmux *InboundXmuxModel) tfsdk.Plan {
	t.Helper()
	return inboundResourceFixture(t, &InboundResource{}, &InboundResourceModel{
		Port: types.Int64Value(443), Protocol: types.StringValue("vless"),
		StreamSettings: xmuxStreamSettings(xmux),
	})
}

func outboundsXmuxConfig(t *testing.T, xmux *InboundXmuxModel) tfsdk.Plan {
	t.Helper()
	var schemaResp resource.SchemaResponse
	NewXrayOutboundsResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	p := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := p.Set(context.Background(), &XrayOutboundsModel{
		ID: types.StringNull(),
		Outbound: []XrayOutboundEntry{{
			Tag: types.StringValue("proxy"), Protocol: types.StringValue("vless"),
			StreamSettings: xmuxStreamSettings(xmux),
		}},
	}); diags.HasError() {
		t.Fatalf("building outbounds config: %v", diags)
	}
	return p
}

// Every rejected value either fails the xray-core config build, is silently
// rewritten by xray-core, or cannot round-trip; the schema is shared, so both
// resources must reject the same values.
func TestXmuxConfigValidation(t *testing.T) {
	str := types.StringValue
	cases := []struct {
		name      string
		xmux      InboundXmuxModel
		wantErr   string
		wantAttr  string // attribute the error points at
		wantNoErr string // must not appear even when wantErr does
	}{
		{name: "connection pool", xmux: InboundXmuxModel{
			MaxConnections: str("2"), HMaxRequestTimes: str("6000-9000"), HMaxReusableSecs: str("1800-3000"),
		}},
		{name: "concurrency with every limit", xmux: InboundXmuxModel{
			MaxConcurrency: str("16-32"), CMaxReuseTimes: str("64-128"), HMaxRequestTimes: str("600-900"),
			HMaxReusableSecs: str("1800-3000"), HKeepAlivePeriod: types.Int64Value(30),
		}},
		{name: "empty block"},
		{name: "range from zero", xmux: InboundXmuxModel{MaxConnections: str("0-4")}},
		{name: "largest int32 bound", xmux: InboundXmuxModel{HMaxRequestTimes: str("1-2147483647")}},
		{name: "keep-alive disabled", xmux: InboundXmuxModel{HKeepAlivePeriod: types.Int64Value(-1)}},
		{name: "longest keep-alive", xmux: InboundXmuxModel{HKeepAlivePeriod: types.Int64Value(xmuxMaxKeepAlivePeriod)}},
		{name: "unknown values are checked at apply", xmux: InboundXmuxModel{
			MaxConnections: types.StringUnknown(), MaxConcurrency: str("16-32"), HKeepAlivePeriod: types.Int64Unknown(),
		}},
		{name: "spaces", xmux: InboundXmuxModel{MaxConnections: str("2 - 4")}, wantErr: "is not a number or a range", wantAttr: "max_connections"},
		{name: "leading zero", xmux: InboundXmuxModel{MaxConcurrency: str("08-16")}, wantErr: "is not a number or a range", wantAttr: "max_concurrency"},
		{name: "negative", xmux: InboundXmuxModel{CMaxReuseTimes: str("-1")}, wantErr: "is not a number or a range", wantAttr: "c_max_reuse_times"},
		{name: "empty string", xmux: InboundXmuxModel{HMaxRequestTimes: str("")}, wantErr: "is not a number or a range", wantAttr: "h_max_request_times"},
		{name: "open range", xmux: InboundXmuxModel{HMaxReusableSecs: str("1800-")}, wantErr: "is not a number or a range", wantAttr: "h_max_reusable_secs"},
		{name: "three bounds", xmux: InboundXmuxModel{MaxConnections: str("1-2-3")}, wantErr: "is not a number or a range"},
		{name: "beyond int32", xmux: InboundXmuxModel{MaxConnections: str("2147483648")}, wantErr: "exceeds 2147483647"},
		{name: "upper bound beyond int32", xmux: InboundXmuxModel{HMaxRequestTimes: str("1-2147483648")}, wantErr: "exceeds 2147483647"},
		{name: "bound beyond int64", xmux: InboundXmuxModel{HMaxRequestTimes: str("99999999999999999999")}, wantErr: "exceeds 2147483647"},
		{name: "descending", xmux: InboundXmuxModel{HMaxRequestTimes: str("9000-6000")}, wantErr: `write "6000-9000"`},
		{name: "zero", xmux: InboundXmuxModel{MaxConnections: str("0")}, wantErr: "omit the attribute", wantAttr: "max_connections"},
		{name: "zero range", xmux: InboundXmuxModel{CMaxReuseTimes: str("0-0")}, wantErr: "omit the attribute"},
		{name: "zero keep-alive", xmux: InboundXmuxModel{HKeepAlivePeriod: types.Int64Value(0)}, wantErr: "omit the attribute", wantAttr: "h_keep_alive_period"},
		{name: "keep-alive overflow", xmux: InboundXmuxModel{HKeepAlivePeriod: types.Int64Value(xmuxMaxKeepAlivePeriod + 1)}, wantErr: "overflows"},
		{name: "negative keep-alive overflow", xmux: InboundXmuxModel{HKeepAlivePeriod: types.Int64Value(-xmuxMaxKeepAlivePeriod - 1)}, wantErr: "overflows"},
		{
			name:     "both connection limits",
			xmux:     InboundXmuxModel{MaxConnections: str("2"), MaxConcurrency: str("16-32")},
			wantErr:  "maxConnections cannot be specified together with maxConcurrency",
			wantAttr: "max_concurrency",
		},
		{
			// The malformed value is reported once, not as a conflict too.
			name:      "conflict needs a valid range",
			xmux:      InboundXmuxModel{MaxConnections: str("two"), MaxConcurrency: str("16-32")},
			wantErr:   "is not a number or a range",
			wantNoErr: "Conflicting XMUX limits",
		},
	}
	for _, tc := range cases {
		for _, target := range []struct {
			typeName string
			config   func(*testing.T, *InboundXmuxModel) tfsdk.Plan
		}{
			{"threexui_inbound", inboundXmuxConfig},
			{"threexui_xray_outbounds", outboundsXmuxConfig},
		} {
			t.Run(target.typeName+"/"+tc.name, func(t *testing.T) {
				xmux := tc.xmux
				errs := validateResourceConfigErrors(t, target.typeName, target.config(t, &xmux))
				var messages []string
				matched := false
				for _, d := range errs {
					msg := d.Summary + ": " + d.Detail
					messages = append(messages, msg)
					if tc.wantNoErr != "" && strings.Contains(msg, tc.wantNoErr) {
						t.Errorf("unexpected error %q", msg)
					}
					if tc.wantErr == "" || !strings.Contains(msg, tc.wantErr) {
						continue
					}
					matched = true
					if tc.wantAttr != "" {
						steps := d.Attribute.Steps()
						if last := steps[len(steps)-1]; last != tftypes.AttributeName(tc.wantAttr) {
							t.Errorf("error points at %s, want %s", d.Attribute, tc.wantAttr)
						}
					}
				}
				switch {
				case tc.wantErr == "" && len(errs) > 0:
					t.Fatalf("unexpected errors: %v", messages)
				case tc.wantErr != "" && !matched:
					t.Fatalf("want an error containing %q, got %v", tc.wantErr, messages)
				}
			})
		}
	}

	for typeName, config := range map[string]tfsdk.Plan{
		"threexui_inbound":        inboundXmuxConfig(t, nil),
		"threexui_xray_outbounds": outboundsXmuxConfig(t, nil),
	} {
		if errs := validateResourceConfigErrors(t, typeName, config); len(errs) > 0 {
			t.Errorf("%s without an xmux block: unexpected errors %v", typeName, errs)
		}
	}
}

func TestXmuxValidatorDescriptions(t *testing.T) {
	ctx := context.Background()
	for _, d := range []interface {
		Description(context.Context) string
		MarkdownDescription(context.Context) string
	}{xmuxRangeValidator{}, xmuxKeepAlivePeriodValidator{}, xmuxConnectionLimitsValidator{}} {
		if d.Description(ctx) == "" || d.MarkdownDescription(ctx) != d.Description(ctx) {
			t.Errorf("%T must describe itself", d)
		}
	}
}

// ---------------------------------------------------------------------------
// Removing the block
// ---------------------------------------------------------------------------

// A block absent from the plan must be absent from state, or Terraform fails
// the apply with "was absent, but now present".
func TestAlignBlocksWithPlan_Xmux(t *testing.T) {
	stored := func() *InboundResourceModel {
		return &InboundResourceModel{StreamSettings: &InboundStreamSettingsModel{
			XHTTPSettings: &InboundXHTTPSettingsModel{Xmux: &InboundXmuxModel{MaxConnections: types.StringValue("2")}},
		}}
	}
	withXHTTP := func(xmux *InboundXmuxModel) *InboundResourceModel {
		return &InboundResourceModel{StreamSettings: &InboundStreamSettingsModel{
			XHTTPSettings: &InboundXHTTPSettingsModel{Xmux: xmux},
		}}
	}

	state := stored()
	alignBlocksWithPlan(state, withXHTTP(nil))
	if state.StreamSettings.XHTTPSettings.Xmux != nil {
		t.Fatalf("xmux absent from the plan must be dropped: %+v", state.StreamSettings.XHTTPSettings.Xmux)
	}

	state = stored()
	alignBlocksWithPlan(state, withXHTTP(&InboundXmuxModel{}))
	if state.StreamSettings.XHTTPSettings.Xmux == nil {
		t.Fatal("xmux present in the plan must be kept")
	}

	state = stored()
	alignBlocksWithPlan(state, &InboundResourceModel{StreamSettings: &InboundStreamSettingsModel{}})
	if state.StreamSettings.XHTTPSettings != nil {
		t.Fatalf("xhttp_settings absent from the plan must be dropped with its xmux: %+v", state.StreamSettings.XHTTPSettings)
	}

	// The panel can omit xhttpSettings while the plan declares the block.
	state = &InboundResourceModel{StreamSettings: &InboundStreamSettingsModel{}}
	alignBlocksWithPlan(state, withXHTTP(nil))
	if state.StreamSettings.XHTTPSettings != nil {
		t.Fatalf("absent xhttp_settings must stay absent: %+v", state.StreamSettings.XHTTPSettings)
	}
}

func xmuxOutbound(tag string, xmux *InboundXmuxModel) XrayOutboundEntry {
	return XrayOutboundEntry{
		Tag:            types.StringValue(tag),
		Protocol:       types.StringValue("vless"),
		StreamSettings: xmuxStreamSettings(xmux),
	}
}

// The outbounds ModifyPlan refills omitted sockopt attributes from state; xmux
// attributes are not Computed, so neither the block nor an attribute removed
// from the configuration may come back that way.
func TestXrayOutboundsResource_ModifyPlan_XmuxNotRefilledFromState(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	NewXrayOutboundsResource().Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	prior := &InboundXmuxModel{
		MaxConnections:   types.StringValue("2"),
		HMaxRequestTimes: types.StringValue("6000-9000"),
		HKeepAlivePeriod: types.Int64Value(30),
	}
	stateRaw := xrayOutboundsRawFromModel(t, schemaResp, xmuxOutbound("proxy", prior))

	for _, tc := range []struct {
		name       string
		configured *InboundXmuxModel
	}{
		{"removed block", nil},
		{"removed attributes", &InboundXmuxModel{MaxConnections: types.StringValue("3")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configRaw := xrayOutboundsRawFromModel(t, schemaResp, xmuxOutbound("proxy", tc.configured))
			plan := runXrayOutboundsModifyPlan(t, schemaResp, stateRaw, configRaw, stateRaw)
			var got XrayOutboundsModel
			if diags := plan.Get(ctx, &got); diags.HasError() {
				t.Fatalf("reading plan: %v", diags)
			}
			assertXmuxModelEqual(t, got.Outbound[0].StreamSettings.XHTTPSettings.Xmux, tc.configured)
		})
	}
}
