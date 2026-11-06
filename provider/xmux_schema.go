package provider

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// InboundXmuxModel mirrors xray-core's XmuxConfig (infra/conf/transport_method.go),
// the connection reuse of the side that dials XHTTP; shared by inbound and
// xray_outbounds.
type InboundXmuxModel struct {
	MaxConcurrency   types.String `tfsdk:"max_concurrency"`
	MaxConnections   types.String `tfsdk:"max_connections"`
	CMaxReuseTimes   types.String `tfsdk:"c_max_reuse_times"`
	HMaxRequestTimes types.String `tfsdk:"h_max_request_times"`
	HMaxReusableSecs types.String `tfsdk:"h_max_reusable_secs"`
	HKeepAlivePeriod types.Int64  `tfsdk:"h_keep_alive_period"`
}

func (m *InboundXmuxModel) ranges() map[string]*types.String {
	return map[string]*types.String{
		"max_concurrency":     &m.MaxConcurrency,
		"max_connections":     &m.MaxConnections,
		"c_max_reuse_times":   &m.CMaxReuseTimes,
		"h_max_request_times": &m.HMaxRequestTimes,
		"h_max_reusable_secs": &m.HMaxReusableSecs,
	}
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

// Attributes are Optional without Computed, unlike the rest of stream_settings:
// removing one must remove its key. xray-core treats 0 as unset, so reading a
// zero back as null loses nothing.
func xmuxBlockSchema() schema.SingleNestedBlock {
	rangeAttribute := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			Optional:    true,
			Description: description + " A positive integer or a `from-to` range to pick a random value from.",
			Validators:  []validator.String{xmuxRangeValidator{}},
		}
	}
	return schema.SingleNestedBlock{
		Description: "XMUX: how the side that dials XHTTP reuses its connections (xray `xhttpSettings.xmux`). " +
			"On an inbound these are client settings the panel hands out through subscriptions; on an outbound they apply to the panel's own Xray. " +
			"An empty block is written as `{}`, which selects the xray-core defaults; setting any attribute disables all of them. " +
			"Removing the block, or an attribute, removes it from the stream settings.",
		Validators: []validator.Object{xmuxConnectionLimitsValidator{}},
		Attributes: map[string]schema.Attribute{
			"max_concurrency": rangeAttribute("Requests one connection carries at a time; " +
				"when every connection is at the limit another one is opened. Mutually exclusive with `max_connections`."),
			"max_connections":   rangeAttribute("Connections to open and spread requests over. Mutually exclusive with `max_concurrency`."),
			"c_max_reuse_times": rangeAttribute("Retire a connection after it has served this many proxied connections."),
			"h_max_request_times": rangeAttribute("Retire a connection after this many HTTP requests " +
				"(xray-core default `600-900`, applied only to an empty block)."),
			"h_max_reusable_secs": rangeAttribute("Stop reusing a connection this many seconds after it was opened " +
				"(xray-core default `1800-3000`, applied only to an empty block)."),
			"h_keep_alive_period": schema.Int64Attribute{
				Optional: true,
				Description: "Seconds of silence after which an HTTP/2 connection is checked with a PING (HTTP/3: the QUIC keep-alive period). " +
					"Unset uses the xray-core default (45 for HTTP/2, 10 for HTTP/3); a negative value disables the check.",
				Validators: []validator.Int64{xmuxKeepAlivePeriodValidator{}},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Validators
// ---------------------------------------------------------------------------

// The canonical Int32Range spelling only: xray-core also accepts signs and
// swapped bounds, which the range parser of the 3x-ui form misreads.
var xmuxRangePattern = regexp.MustCompile(`^(0|[1-9]\d*)(?:-(0|[1-9]\d*))?$`)

// parseXmuxRange parses the canonical spelling; ok is false for anything else,
// including a bound beyond int32, which xray-core silently wraps.
func parseXmuxRange(s string) (from, to int64, ok bool) {
	m := xmuxRangePattern.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, false
	}
	from, err := strconv.ParseInt(m[1], 10, 32)
	if err != nil {
		return 0, 0, false
	}
	to = from
	if m[2] != "" {
		if to, err = strconv.ParseInt(m[2], 10, 32); err != nil {
			return 0, 0, false
		}
	}
	return from, to, true
}

type xmuxRangeValidator struct{}

func (xmuxRangeValidator) Description(context.Context) string {
	return fmt.Sprintf("must be a positive integer or a from-to range with from <= to, at most %d", math.MaxInt32)
}

func (v xmuxRangeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (xmuxRangeValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	from, to, ok := parseXmuxRange(s)
	var detail string
	switch {
	case !xmuxRangePattern.MatchString(s):
		detail = fmt.Sprintf(`%q is not a number or a range: use digits without spaces, signs or leading zeros, such as "2" or "16-32".`, s)
	case !ok:
		detail = fmt.Sprintf("%q exceeds %d: xray-core reads each bound as a 32-bit integer and would silently wrap it.", s, math.MaxInt32)
	case from > to:
		detail = fmt.Sprintf(`%q: the lower bound is greater than the upper one; write "%d-%d".`, s, to, from)
	case to == 0:
		detail = fmt.Sprintf("%q: xray-core treats a zero range as unset, so omit the attribute instead.", s)
	default:
		return
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid XMUX range", detail)
}

// time.Duration(seconds) * time.Second in the xray-core dialer overflows beyond this.
const xmuxMaxKeepAlivePeriod = math.MaxInt64 / int64(time.Second)

type xmuxKeepAlivePeriodValidator struct{}

func (xmuxKeepAlivePeriodValidator) Description(context.Context) string {
	return fmt.Sprintf("must be a non-zero number of seconds between %d and %d", -xmuxMaxKeepAlivePeriod, xmuxMaxKeepAlivePeriod)
}

func (v xmuxKeepAlivePeriodValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (xmuxKeepAlivePeriodValidator) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	switch n := req.ConfigValue.ValueInt64(); {
	case n == 0:
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid XMUX keep-alive period",
			"0 selects the xray-core default and reads back as unset, so omit the attribute instead.")
	case n < -xmuxMaxKeepAlivePeriod || n > xmuxMaxKeepAlivePeriod:
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid XMUX keep-alive period",
			fmt.Sprintf("%d seconds overflows the duration xray-core converts it to; use a value between %d and %d.",
				n, -xmuxMaxKeepAlivePeriod, xmuxMaxKeepAlivePeriod))
	}
}

// xmuxConnectionLimitsValidator mirrors the SplitHTTPConfig.Build check in
// xray-core, which compares the upper bounds of both ranges.
type xmuxConnectionLimitsValidator struct{}

func (xmuxConnectionLimitsValidator) Description(context.Context) string {
	return "max_connections and max_concurrency must not both be set"
}

func (v xmuxConnectionLimitsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (xmuxConnectionLimitsValidator) ValidateObject(_ context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	attrs := req.ConfigValue.Attributes()
	connections, _ := attrs["max_connections"].(types.String)
	concurrency, _ := attrs["max_concurrency"].(types.String)
	if xmuxRangeUpperBound(connections) == 0 || xmuxRangeUpperBound(concurrency) == 0 {
		return
	}
	resp.Diagnostics.AddAttributeError(req.Path.AtName("max_concurrency"), "Conflicting XMUX limits",
		"xray-core refuses an XMUX configuration that sets both max_connections and max_concurrency "+
			"(\"maxConnections cannot be specified together with maxConcurrency\"). On an inbound it would break the "+
			"configuration of every subscription client, on an outbound the panel's own Xray, and the 3x-ui form drops "+
			"max_concurrency the next time the inbound is saved there. Set only one of them.")
}

// xmuxRangeUpperBound is 0 for null, unknown and malformed values, which the
// attribute validator reports on its own.
func xmuxRangeUpperBound(v types.String) int64 {
	if v.IsNull() || v.IsUnknown() {
		return 0
	}
	_, to, ok := parseXmuxRange(v.ValueString())
	if !ok {
		return 0
	}
	return to
}

// ---------------------------------------------------------------------------
// Typed model <-> untyped map
// ---------------------------------------------------------------------------

func expandXmuxFromModel(m *InboundXmuxModel) map[string]any {
	out := map[string]any{}
	for key, v := range m.ranges() {
		if !v.IsNull() && !v.IsUnknown() {
			out[key] = v.ValueString()
		}
	}
	if !m.HKeepAlivePeriod.IsNull() && !m.HKeepAlivePeriod.IsUnknown() {
		out["h_keep_alive_period"] = m.HKeepAlivePeriod.ValueInt64()
	}
	return out
}

func flattenXmuxToModel(data map[string]any) *InboundXmuxModel {
	m := &InboundXmuxModel{HKeepAlivePeriod: types.Int64Null()}
	for key, dst := range m.ranges() {
		if v, ok := data[key].(string); ok {
			*dst = types.StringValue(v)
		} else {
			*dst = types.StringNull()
		}
	}
	if v, ok := data["h_keep_alive_period"]; ok {
		m.HKeepAlivePeriod = types.Int64Value(int64Value(v))
	}
	return m
}
