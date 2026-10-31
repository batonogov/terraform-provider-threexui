package provider

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// InboundSockoptModel mirrors xray-core's SocketConfig
// (infra/conf/transport_sockopt.go); shared by inbound and xray_outbounds.
type InboundSockoptModel struct {
	Mark                 types.Int64  `tfsdk:"mark"`
	TCPKeepAliveInterval types.Int64  `tfsdk:"tcp_keep_alive_interval"`
	TCPKeepAliveIdle     types.Int64  `tfsdk:"tcp_keep_alive_idle"`
	TCPUserTimeout       types.Int64  `tfsdk:"tcp_user_timeout"`
	TCPMaxSeg            types.Int64  `tfsdk:"tcp_max_seg"`
	TCPWindowClamp       types.Int64  `tfsdk:"tcp_window_clamp"`
	TCPCongestion        types.String `tfsdk:"tcp_congestion"`
	TCPNoDelay           types.Bool   `tfsdk:"tcp_no_delay"`
	TFOEnable            types.Bool   `tfsdk:"tfo_enable"`
	TCPMptcp             types.Bool   `tfsdk:"tcp_mptcp"`
	Tproxy               types.String `tfsdk:"tproxy"`
	DomainStrategy       types.String `tfsdk:"domain_strategy"`
	AcceptProxyProtocol  types.Bool   `tfsdk:"accept_proxy_protocol"`
	TrustedXForwardedFor types.List   `tfsdk:"trusted_x_forwarded_for"` // list of string
	V6Only               types.Bool   `tfsdk:"v6_only"`
	Interface            types.String `tfsdk:"interface"`
	DialerProxy          types.String `tfsdk:"dialer_proxy"`
	AddressPortStrategy  types.String `tfsdk:"address_port_strategy"`
	Penetrate            types.Bool   `tfsdk:"penetrate"`
	HappyEyeballs        types.Object `tfsdk:"happy_eyeballs"` // sockoptHappyEyeballsAttrTypes
	CustomSockopt        types.List   `tfsdk:"custom_sockopt"` // list of sockoptCustomAttrTypes
}

var sockoptHappyEyeballsAttrTypes = map[string]attr.Type{
	"try_delay_ms":       types.Int64Type,
	"prioritize_ipv6":    types.BoolType,
	"interleave":         types.Int64Type,
	"max_concurrent_try": types.Int64Type,
}

var sockoptCustomAttrTypes = map[string]attr.Type{
	"system":  types.StringType,
	"network": types.StringType,
	"level":   types.StringType,
	"opt":     types.StringType,
	"type":    types.StringType,
	"value":   types.StringType,
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

func sockoptBlockSchema() schema.SingleNestedBlock {
	int32Range := int64validator.Between(math.MinInt32, math.MaxInt32)
	nonNegativeInt32 := int64validator.Between(0, math.MaxInt32)

	return schema.SingleNestedBlock{
		Description: "Socket options (xray `sockopt`). Removing the block drops `sockopt` from the stream settings. " +
			"Omitting an attribute keeps its current value; clear it with `0`, `false`, `\"\"` or `[]`.",
		Validators: []validator.Object{sockoptKeepAliveValidator{}},
		Attributes: map[string]schema.Attribute{
			"mark": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description: "Linux SO_MARK for policy routing (needs CAP_NET_ADMIN). `0` leaves the mark unset.",
				Validators:  []validator.Int64{int32Range},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"tcp_keep_alive_interval": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description: "TCP keep-alive probe interval in seconds (TCP_KEEPINTVL). `0` uses the xray-core default: 45 s on outbounds; " +
					"on inbounds keep-alive stays off unless `tcp_keep_alive_idle` is positive. A negative value disables keep-alive. " +
					"Must not have the opposite sign of `tcp_keep_alive_idle`.",
				Validators: []validator.Int64{int32Range},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"tcp_keep_alive_idle": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description: "TCP keep-alive idle time in seconds (TCP_KEEPIDLE). `0` uses the xray-core default: 45 s on outbounds; " +
					"on inbounds keep-alive stays off unless `tcp_keep_alive_interval` is positive. A negative value disables keep-alive. " +
					"Must not have the opposite sign of `tcp_keep_alive_interval`.",
				Validators: []validator.Int64{int32Range},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"tcp_user_timeout": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description: "TCP_USER_TIMEOUT in milliseconds (Linux). `0` keeps the OS default.",
				Validators:  []validator.Int64{nonNegativeInt32},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"tcp_max_seg": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description: "TCP_MAXSEG (MSS) in bytes (Linux). `0` keeps the OS default.",
				Validators:  []validator.Int64{nonNegativeInt32},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"tcp_window_clamp": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description: "TCP_WINDOW_CLAMP in bytes (Linux). `0` keeps the OS default; small values throttle throughput.",
				Validators:  []validator.Int64{nonNegativeInt32},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"tcp_congestion": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "TCP congestion control algorithm (e.g. `bbr`, `cubic`, `reno`); it must be available in the host kernel. `\"\"` keeps the OS default.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tcp_no_delay": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:        "Deprecated: xray-core ignores `tcpNoDelay`.",
				DeprecationMessage: "Removed in xray-core v24.12.31 (every supported 3x-ui release ships a newer core); xray ignores unknown JSON keys, so existing configs still work but this attribute has no effect.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"tfo_enable": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description: "TCP Fast Open (`tcpFastOpen`). `false` explicitly disables it.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"tcp_mptcp": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description: "Enable Multipath TCP.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"tproxy": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Transparent proxy mode: `off`, `redirect` or `tproxy` (Linux, needs CAP_NET_ADMIN).",
				Validators:  tproxyValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_strategy": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Outbound only. How the dialer resolves a domain target (`AsIs`, `UseIP`, `UseIPv4`, `ForceIPv6v4`, …).",
				Validators:  domainStrategyValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"accept_proxy_protocol": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description: "Inbound only. Require a PROXY protocol v1/v2 header on stream (TCP or Unix socket) listeners: raw TCP, " +
					"WebSocket, HTTPUpgrade, gRPC and XHTTP over TCP. UDP-based transports (mKCP, XHTTP/3, Hysteria) ignore it.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"trusted_x_forwarded_for": schema.ListAttribute{
				Optional: true, Computed: true,
				ElementType: types.StringType,
				Description: "Inbound only (ws, httpupgrade, xhttp, grpc). Header names that mark a request as coming from a trusted " +
					"reverse proxy or CDN: when one of them is present, xray takes the client IP from the first `X-Forwarded-For` " +
					"entry. Since xray-core v26.6.22 (3x-ui v3.4.0) `X-Forwarded-For` is ignored unless this is set. `[]` clears it.",
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"v6_only": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description: "Inbound only. Make a `::` listener IPv6-only (IPV6_V6ONLY). Since 3x-ui v3.9.0 the panel's port-conflict " +
					"check then lets an inbound on a specific IPv4 address share the port (a `0.0.0.0` or empty listen still conflicts); " +
					"v3.3.x-v3.8.x panels reject such an inbound regardless.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"interface": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Bind outgoing sockets to this network interface (e.g. `eth0`, `wg0`). On inbounds xray-core applies it " +
					"only on macOS; Linux and Windows listeners ignore it. `\"\"` clears it.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dialer_proxy": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Outbound only. Tag of another outbound to dial through (proxy chaining; replaces the removed `proxySettings.tag`). `\"\"` clears it.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"address_port_strategy": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Outbound only. Take the target address and/or port from DNS SRV or TXT records: `none`, `SrvPortOnly`, " +
					"`SrvAddressOnly`, `SrvPortAndAddress`, `TxtPortOnly`, `TxtAddressOnly`, `TxtPortAndAddress`. " +
					"xray-core v26.9.8+ (3x-ui v3.8.0+) rejects it on `freedom` outbounds.",
				Validators: []validator.String{
					stringvalidator.OneOf("none", "SrvPortOnly", "SrvAddressOnly", "SrvPortAndAddress",
						"TxtPortOnly", "TxtAddressOnly", "TxtPortAndAddress"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"penetrate": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description: "Outbound only (XHTTP). Reuse these socket options for the `downloadSettings` connection.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"happy_eyeballs": schema.SingleNestedAttribute{
				Optional: true, Computed: true,
				Description: "Outbound only. RFC 8305 Happy Eyeballs for TCP targets that resolve to several IPs " +
					"(needs a resolving `domain_strategy`; inactive with `dialer_proxy`).",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"try_delay_ms": schema.Int64Attribute{
						Optional: true, Computed: true,
						Description: "Delay in milliseconds before racing the next address (3x-ui uses 250). `0` disables Happy Eyeballs.",
						Validators:  []validator.Int64{int64validator.AtLeast(0)},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"prioritize_ipv6": schema.BoolAttribute{
						Optional: true, Computed: true,
						Description: "Try IPv6 addresses first.",
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"interleave": schema.Int64Attribute{
						Optional: true, Computed: true,
						Description: "Addresses of one family to try before switching family (xray default 1).",
						Validators:  []validator.Int64{int64validator.Between(1, math.MaxUint32)},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"max_concurrent_try": schema.Int64Attribute{
						Optional: true, Computed: true,
						Description: "Maximum concurrent connection attempts (xray default 4). `0` disables Happy Eyeballs.",
						Validators:  []validator.Int64{int64validator.Between(0, math.MaxUint32)},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			// Element fields are not Computed: the list is index-matched, and
			// UseStateForUnknown would bleed system/level onto a neighbouring entry.
			"custom_sockopt": schema.ListNestedAttribute{
				Optional: true, Computed: true,
				Description: "Raw setsockopt(2) calls applied to every socket. `[]` clears the list.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Validators: []validator.Object{customSockoptIntValueValidator{}},
					Attributes: map[string]schema.Attribute{
						"system": schema.StringAttribute{
							Optional:    true,
							Description: "Apply only on this OS (Go GOOS: `linux`, `windows`, `darwin`, …). Unset applies everywhere.",
							Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
						},
						"network": schema.StringAttribute{
							Optional:    true,
							Description: "Apply only to sockets whose network starts with this prefix (`tcp`, `udp`, `tcp6`, …). Unset applies to all.",
							Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
						},
						"level": schema.StringAttribute{
							Optional:    true,
							Description: "Option level as a decimal 32-bit integer (`6` = IPPROTO_TCP, `1` = SOL_SOCKET on Linux). Unset means `6`.",
							Validators:  []validator.String{sockoptInt32StringValidator{}},
						},
						"opt": schema.StringAttribute{
							Required:    true,
							Description: "Option name as a decimal 32-bit integer (e.g. `13` = TCP_CONGESTION on Linux).",
							Validators:  []validator.String{sockoptInt32StringValidator{}},
						},
						"type": schema.StringAttribute{
							Required:    true,
							Description: "Value type: `int` or `str`.",
							Validators:  []validator.String{stringvalidator.OneOf("int", "str")},
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "Option value; a decimal 32-bit integer when `type = \"int\"`.",
						},
					},
				},
			},
		},
	}
}

// xray-core parses level, opt and an int value with strconv.Atoi, ignoring the
// error, and setsockopt(2) takes them as a C int: anything outside int32 is
// silently replaced or truncated instead of failing.
func validSockoptInt32(s string) bool {
	_, err := strconv.ParseInt(s, 10, 32)
	return err == nil
}

type sockoptInt32StringValidator struct{}

func (sockoptInt32StringValidator) Description(context.Context) string {
	return fmt.Sprintf("must be a decimal integer between %d and %d", math.MinInt32, math.MaxInt32)
}

func (v sockoptInt32StringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v sockoptInt32StringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || validSockoptInt32(req.ConfigValue.ValueString()) {
		return
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid custom_sockopt value",
		fmt.Sprintf("%q %s.", req.ConfigValue.ValueString(), v.Description(ctx)))
}

type customSockoptIntValueValidator struct{}

func (customSockoptIntValueValidator) Description(context.Context) string {
	return fmt.Sprintf(`value must be a decimal integer between %d and %d when type is "int"`, math.MinInt32, math.MaxInt32)
}

func (v customSockoptIntValueValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v customSockoptIntValueValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	attrs := req.ConfigValue.Attributes()
	typ, _ := attrs["type"].(types.String)
	value, _ := attrs["value"].(types.String)
	if typ.IsNull() || typ.IsUnknown() || typ.ValueString() != "int" || value.IsNull() || value.IsUnknown() {
		return
	}
	if !validSockoptInt32(value.ValueString()) {
		resp.Diagnostics.AddAttributeError(req.Path.AtName("value"), "Invalid custom_sockopt value",
			fmt.Sprintf("%q: %s.", value.ValueString(), v.Description(ctx)))
	}
}

// sockoptKeepAliveValidator checks the configuration only; both resources'
// ModifyPlan repeat the check for a value the plan keeps from state.
type sockoptKeepAliveValidator struct{}

func (sockoptKeepAliveValidator) Description(context.Context) string {
	return "tcp_keep_alive_idle and tcp_keep_alive_interval must not have opposite signs"
}

func (v sockoptKeepAliveValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (sockoptKeepAliveValidator) ValidateObject(_ context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	validateSockoptKeepAliveObject(req.ConfigValue, req.Path, &resp.Diagnostics)
}

func validateSockoptKeepAliveObject(sockopt types.Object, p path.Path, diags *diag.Diagnostics) {
	if sockopt.IsNull() || sockopt.IsUnknown() {
		return
	}
	idle, _ := sockopt.Attributes()["tcp_keep_alive_idle"].(types.Int64)
	interval, _ := sockopt.Attributes()["tcp_keep_alive_interval"].(types.Int64)
	validateSockoptKeepAlive(idle, interval, p, diags)
}

func validateSockoptKeepAlive(idle, interval types.Int64, p path.Path, diags *diag.Diagnostics) {
	if idle.IsNull() || idle.IsUnknown() || interval.IsNull() || interval.IsUnknown() {
		return
	}
	a, b := idle.ValueInt64(), interval.ValueInt64()
	// Out-of-range values are the int32 attribute validators' to report.
	if a < math.MinInt32 || a > math.MaxInt32 || b < math.MinInt32 || b > math.MaxInt32 {
		return
	}
	// xray-core fails every listen and dial when the int32 product is negative,
	// overflow included (transport/internet/system_listener.go, system_dialer.go).
	if int32(a)*int32(b) >= 0 {
		return
	}
	diags.AddAttributeError(p.AtName("tcp_keep_alive_idle"), "Invalid TCP keep-alive settings",
		fmt.Sprintf("xray-core refuses to listen or dial when tcp_keep_alive_idle (%d) multiplied by "+
			"tcp_keep_alive_interval (%d) is negative as a 32-bit integer. Give both the same sign "+
			"(a negative value disables keep-alive) or set one of them to 0; an attribute omitted "+
			"from the configuration keeps its current value.", a, b))
}

// ---------------------------------------------------------------------------
// Typed model -> untyped map
// ---------------------------------------------------------------------------

func expandSockoptFromModel(m *InboundSockoptModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	for key, v := range map[string]types.Int64{
		"mark":                    m.Mark,
		"tcp_keep_alive_interval": m.TCPKeepAliveInterval,
		"tcp_keep_alive_idle":     m.TCPKeepAliveIdle,
		"tcp_user_timeout":        m.TCPUserTimeout,
		"tcp_max_seg":             m.TCPMaxSeg,
		"tcp_window_clamp":        m.TCPWindowClamp,
	} {
		if !v.IsNull() && !v.IsUnknown() {
			out[key] = v.ValueInt64()
		}
	}
	for key, v := range map[string]types.Bool{
		"tcp_no_delay":          m.TCPNoDelay,
		"tfo_enable":            m.TFOEnable,
		"tcp_mptcp":             m.TCPMptcp,
		"accept_proxy_protocol": m.AcceptProxyProtocol,
		"v6_only":               m.V6Only,
		"penetrate":             m.Penetrate,
	} {
		if !v.IsNull() && !v.IsUnknown() {
			out[key] = v.ValueBool()
		}
	}
	for key, v := range map[string]types.String{
		"tcp_congestion":        m.TCPCongestion,
		"tproxy":                m.Tproxy,
		"domain_strategy":       m.DomainStrategy,
		"interface":             m.Interface,
		"dialer_proxy":          m.DialerProxy,
		"address_port_strategy": m.AddressPortStrategy,
	} {
		if !v.IsNull() && !v.IsUnknown() {
			out[key] = v.ValueString()
		}
	}
	if !m.TrustedXForwardedFor.IsNull() && !m.TrustedXForwardedFor.IsUnknown() {
		out["trusted_x_forwarded_for"] = typesListToAnySlice(m.TrustedXForwardedFor)
	}
	if !m.HappyEyeballs.IsNull() && !m.HappyEyeballs.IsUnknown() {
		out["happy_eyeballs"] = expandHappyEyeballsFromObject(m.HappyEyeballs)
	}
	if !m.CustomSockopt.IsNull() && !m.CustomSockopt.IsUnknown() {
		out["custom_sockopt"] = expandCustomSockoptFromList(m.CustomSockopt)
	}
	return out
}

func expandHappyEyeballsFromObject(obj types.Object) map[string]any {
	out := map[string]any{}
	for key, v := range obj.Attributes() {
		switch tv := v.(type) {
		case types.Int64:
			if !tv.IsNull() && !tv.IsUnknown() {
				out[key] = tv.ValueInt64()
			}
		case types.Bool:
			if !tv.IsNull() && !tv.IsUnknown() {
				out[key] = tv.ValueBool()
			}
		}
	}
	return out
}

func expandCustomSockoptFromList(list types.List) []any {
	out := make([]any, 0, len(list.Elements()))
	for _, e := range list.Elements() {
		obj, ok := e.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		entry := map[string]any{}
		for key, v := range obj.Attributes() {
			if sv, ok := v.(types.String); ok && !sv.IsNull() && !sv.IsUnknown() {
				entry[key] = sv.ValueString()
			}
		}
		out = append(out, entry)
	}
	return out
}

// ---------------------------------------------------------------------------
// Untyped map -> typed model
// ---------------------------------------------------------------------------

func flattenSockoptToModel(data map[string]any) *InboundSockoptModel {
	m := &InboundSockoptModel{
		Mark:                 sockoptInt64(data, "mark"),
		TCPKeepAliveInterval: sockoptInt64(data, "tcp_keep_alive_interval"),
		TCPKeepAliveIdle:     sockoptInt64(data, "tcp_keep_alive_idle"),
		TCPUserTimeout:       sockoptInt64(data, "tcp_user_timeout"),
		TCPMaxSeg:            sockoptInt64(data, "tcp_max_seg"),
		TCPWindowClamp:       sockoptInt64(data, "tcp_window_clamp"),
		TCPCongestion:        sockoptString(data, "tcp_congestion"),
		TCPNoDelay:           sockoptBool(data, "tcp_no_delay"),
		TFOEnable:            sockoptBool(data, "tfo_enable"),
		TCPMptcp:             sockoptBool(data, "tcp_mptcp"),
		AcceptProxyProtocol:  sockoptBool(data, "accept_proxy_protocol"),
		V6Only:               sockoptBool(data, "v6_only"),
		Interface:            sockoptString(data, "interface"),
		DialerProxy:          sockoptString(data, "dialer_proxy"),
		Penetrate:            sockoptBool(data, "penetrate"),
		TrustedXForwardedFor: types.ListNull(types.StringType),
		HappyEyeballs:        types.ObjectNull(sockoptHappyEyeballsAttrTypes),
		CustomSockopt:        types.ListNull(types.ObjectType{AttrTypes: sockoptCustomAttrTypes}),
	}
	// An empty enum is xray's default, and earlier provider versions read it
	// back as null; keep that behaviour.
	for _, f := range []struct {
		key string
		dst *types.String
	}{
		{"tproxy", &m.Tproxy},
		{"domain_strategy", &m.DomainStrategy},
		{"address_port_strategy", &m.AddressPortStrategy},
	} {
		if v, ok := data[f.key].(string); ok && v != "" {
			*f.dst = types.StringValue(v)
		} else {
			*f.dst = types.StringNull()
		}
	}
	if v, ok := data["trusted_x_forwarded_for"].([]any); ok {
		m.TrustedXForwardedFor = stringListKeepEmpty(v)
	}
	if v, ok := data["happy_eyeballs"].(map[string]any); ok {
		m.HappyEyeballs = flattenHappyEyeballsToObject(v)
	}
	if v, ok := data["custom_sockopt"].([]any); ok {
		m.CustomSockopt = flattenCustomSockoptToList(v)
	}
	return m
}

func sockoptInt64(data map[string]any, key string) types.Int64 {
	if v, ok := data[key]; ok {
		return types.Int64Value(int64Value(v))
	}
	return types.Int64Null()
}

func sockoptBool(data map[string]any, key string) types.Bool {
	if v, ok := data[key].(bool); ok {
		return types.BoolValue(v)
	}
	return types.BoolNull()
}

func sockoptString(data map[string]any, key string) types.String {
	if v, ok := data[key].(string); ok {
		return types.StringValue(v)
	}
	return types.StringNull()
}

// stringListKeepEmpty, unlike anySliceToTypesList, keeps `[]`: an explicit
// clear read back as null fails the apply with an inconsistent result.
func stringListKeepEmpty(slice []any) types.List {
	elems := make([]attr.Value, 0, len(slice))
	for _, item := range slice {
		if s, ok := item.(string); ok {
			elems = append(elems, types.StringValue(s))
		}
	}
	return types.ListValueMust(types.StringType, elems)
}

func flattenHappyEyeballsToObject(data map[string]any) types.Object {
	attrs := map[string]attr.Value{
		"try_delay_ms":       sockoptInt64(data, "try_delay_ms"),
		"prioritize_ipv6":    sockoptBool(data, "prioritize_ipv6"),
		"interleave":         sockoptInt64(data, "interleave"),
		"max_concurrent_try": sockoptInt64(data, "max_concurrent_try"),
	}
	return types.ObjectValueMust(sockoptHappyEyeballsAttrTypes, attrs)
}

func flattenCustomSockoptToList(items []any) types.List {
	elemType := types.ObjectType{AttrTypes: sockoptCustomAttrTypes}
	elems := make([]attr.Value, 0, len(items))
	for _, item := range items {
		entry, _ := item.(map[string]any)
		attrs := make(map[string]attr.Value, len(sockoptCustomAttrTypes))
		for key := range sockoptCustomAttrTypes {
			attrs[key] = sockoptString(entry, key)
		}
		elems = append(elems, types.ObjectValueMust(sockoptCustomAttrTypes, attrs))
	}
	return types.ListValueMust(elemType, elems)
}
