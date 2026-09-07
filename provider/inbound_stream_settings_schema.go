package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------------

type InboundStreamSettingsModel struct {
	Network             types.String                        `tfsdk:"network"`
	Security            types.String                        `tfsdk:"security"`
	ExternalProxy       []InboundExternalProxyModel         `tfsdk:"external_proxy"`
	RealitySettings     *InboundRealitySettingsModel        `tfsdk:"reality_settings"`
	TLSSettings         *InboundTLSSettingsModel            `tfsdk:"tls_settings"`
	TCPSettings         *InboundTCPSettingsModel            `tfsdk:"tcp_settings"`
	WSSettings          *InboundWSSettingsModel             `tfsdk:"ws_settings"`
	GRPCSettings        *InboundGRPCSettingsModel           `tfsdk:"grpc_settings"`
	HTTPUpgradeSettings *InboundHTTPUpgradeSettingsModel    `tfsdk:"httpupgrade_settings"`
	XHTTPSettings       *InboundXHTTPSettingsModel          `tfsdk:"xhttp_settings"`
	KCPSettings         *InboundKCPSettingsModel            `tfsdk:"kcp_settings"`
	HysteriaSettings    *InboundHysteriaStreamSettingsModel `tfsdk:"hysteria_settings"`
	Sockopt             *InboundSockoptModel                `tfsdk:"sockopt"`
}

type InboundExternalProxyModel struct {
	Dest     types.String `tfsdk:"dest"`
	Port     types.Int64  `tfsdk:"port"`
	Remark   types.String `tfsdk:"remark"`
	ForceTLS types.String `tfsdk:"force_tls"`
}

type InboundRealitySettingsModel struct {
	Show         types.Bool   `tfsdk:"show"`
	Xver         types.Int64  `tfsdk:"xver"`
	Target       types.String `tfsdk:"target"`
	ServerNames  types.List   `tfsdk:"server_names"` // list of string
	PrivateKey   types.String `tfsdk:"private_key"`
	MinClientVer types.String `tfsdk:"min_client_ver"`
	MaxClientVer types.String `tfsdk:"max_client_ver"`
	MaxTimediff  types.Int64  `tfsdk:"max_timediff"`
	ShortIDs     types.List   `tfsdk:"short_ids"` // list of string
	Mldsa65Seed  types.String `tfsdk:"mldsa65_seed"`
	Settings     types.Object `tfsdk:"settings"` // realityInnerSettingsAttrTypes
}

type InboundTLSSettingsModel struct {
	ServerName    types.String `tfsdk:"server_name"`
	Fingerprint   types.String `tfsdk:"fingerprint"`
	AllowInsecure types.Bool   `tfsdk:"allow_insecure"`
	Alpn          types.List   `tfsdk:"alpn"` // list of string
	MinVersion    types.String `tfsdk:"min_version"`
	MaxVersion    types.String `tfsdk:"max_version"`
	Cipher        types.String `tfsdk:"cipher"`
}

// realityInnerSettingsAttrTypes defines the attribute types for the
// reality_settings.settings nested attribute (SingleNestedAttribute).
var realityInnerSettingsAttrTypes = map[string]attr.Type{
	"public_key":     types.StringType,
	"fingerprint":    types.StringType,
	"server_name":    types.StringType,
	"spider_x":       types.StringType,
	"mldsa65_verify": types.StringType,
}

type InboundTCPSettingsModel struct {
	AcceptProxyProtocol types.Bool   `tfsdk:"accept_proxy_protocol"`
	HeaderType          types.String `tfsdk:"header_type"`
}

type InboundWSSettingsModel struct {
	Path    types.String `tfsdk:"path"`
	Headers types.Map    `tfsdk:"headers"` // map of string
}

type InboundGRPCSettingsModel struct {
	ServiceName         types.String `tfsdk:"service_name"`
	MultiMode           types.Bool   `tfsdk:"multi_mode"`
	IdleTimeout         types.Int64  `tfsdk:"idle_timeout"`
	HealthCheckTimeout  types.Int64  `tfsdk:"health_check_timeout"`
	PermitWithoutStream types.Bool   `tfsdk:"permit_without_stream"`
	InitialWindowsSize  types.Int64  `tfsdk:"initial_windows_size"`
}

type InboundHTTPUpgradeSettingsModel struct {
	Path types.String `tfsdk:"path"`
	Host types.String `tfsdk:"host"`
}

type InboundXHTTPSettingsModel struct {
	Path              types.String `tfsdk:"path"`
	Mode              types.String `tfsdk:"mode"`
	NoSSEHeader       types.Bool   `tfsdk:"no_sse_header"`
	KeepAliveInterval types.Int64  `tfsdk:"keep_alive_interval"`
	XPaddingBytes     types.String `tfsdk:"x_padding_bytes"`
	XPaddingObfsMode  types.Bool   `tfsdk:"x_padding_obfs_mode"`
	XPaddingKey       types.String `tfsdk:"x_padding_key"`
	XPaddingHeader    types.String `tfsdk:"x_padding_header"`
	XPaddingPlacement types.String `tfsdk:"x_padding_placement"`
	XPaddingMethod    types.String `tfsdk:"x_padding_method"`
}

type InboundKCPSettingsModel struct {
	MTU              types.Int64  `tfsdk:"mtu"`
	TTI              types.Int64  `tfsdk:"tti"`
	UplinkCapacity   types.Int64  `tfsdk:"uplink_capacity"`
	DownlinkCapacity types.Int64  `tfsdk:"downlink_capacity"`
	CwndMultiplier   types.Int64  `tfsdk:"cwnd_multiplier"`
	MaxSendingWindow types.Int64  `tfsdk:"max_sending_window"`
	HeaderType       types.String `tfsdk:"header_type"`
}

type InboundHysteriaStreamSettingsModel struct {
	Protocol       types.String `tfsdk:"protocol"`
	Version        types.Int64  `tfsdk:"version"`
	Auth           types.String `tfsdk:"auth"`
	UDPIdleTimeout types.Int64  `tfsdk:"udp_idle_timeout"`
}

type InboundSockoptModel struct {
	Mark                 types.Int64  `tfsdk:"mark"`
	TCPKeepAliveInterval types.Int64  `tfsdk:"tcp_keep_alive_interval"`
	TCPNoDelay           types.Bool   `tfsdk:"tcp_no_delay"`
	TFOEnable            types.Bool   `tfsdk:"tfo_enable"`
	Tproxy               types.String `tfsdk:"tproxy"`
	DomainStrategy       types.String `tfsdk:"domain_strategy"`
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

func inboundStreamSettingsBlockSchema() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Stream (transport) settings for the inbound.",
		Attributes: map[string]schema.Attribute{
			"network": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Transport network (tcp, ws, grpc, httpupgrade, xhttp, kcp, hysteria).",
				Validators:  networkValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"security": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Security type (none, reality, tls).",
				Validators:  securityValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"external_proxy": schema.ListNestedBlock{
				Description: "External proxy entries.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"dest": schema.StringAttribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"port": schema.Int64Attribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"remark": schema.StringAttribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"force_tls": schema.StringAttribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"reality_settings": schema.SingleNestedBlock{
				Description: "Reality settings.",
				Attributes: map[string]schema.Attribute{
					"show": schema.BoolAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"xver": schema.Int64Attribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"target": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "Target server (e.g. 'google.com:443').",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"server_names": schema.ListAttribute{
						Optional: true, Computed: true,
						ElementType: types.StringType,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"private_key": schema.StringAttribute{
						Optional: true, Computed: true, Sensitive: true,
						Description: "Reality private key (auto-generated if empty).",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"min_client_ver": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "Minimum client Xray version the REALITY server accepts, as 'major.minor.patch' (e.g. '26.3.27'). Never configuring it does NOT disable the gate: Xray 26.7.x substitutes 26.3.27 for an unset value and rejects clients reporting an older or absent version (e.g. sing-box). Set '0.0.0' — the canonical spelling; '0' and '0.0' are zero-filled to the same version — to remove the lower bound. Like every Optional+Computed attribute here, removing it from the configuration keeps the last applied value rather than clearing it; change the bound by setting the value you want.",
						Validators:  realityClientVerValidators(),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"max_client_ver": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "Maximum client Xray version the REALITY server accepts, as 'major.minor.patch'. Never configuring it means no upper bound; once set, use '255.255.255' to widen it back to every version, since removing the attribute keeps the last applied value.",
						Validators:  realityClientVerValidators(),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"max_timediff": schema.Int64Attribute{
						Optional: true, Computed: true,
						Description: "Maximum allowed time difference with the client, in milliseconds. 0 disables the check, and is also how the check is turned back off once set — removing the attribute keeps the last applied value.",
						Validators:  realityMaxTimediffValidators(),
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"short_ids": schema.ListAttribute{
						Optional: true, Computed: true,
						Sensitive:   true,
						ElementType: types.StringType,
						Description: "Short IDs (auto-generated if empty).",
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"mldsa65_seed": schema.StringAttribute{
						Optional: true, Computed: true,
						Sensitive: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"settings": schema.SingleNestedAttribute{
						Optional: true, Computed: true,
						Description: "Reality inner settings (client-side).",
						PlanModifiers: []planmodifier.Object{
							objectplanmodifier.UseStateForUnknown(),
						},
						Attributes: map[string]schema.Attribute{
							"public_key": schema.StringAttribute{
								Optional: true, Computed: true,
								Description: "Reality public key (auto-generated if empty).",
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"fingerprint": schema.StringAttribute{
								Optional: true, Computed: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"server_name": schema.StringAttribute{
								Optional: true, Computed: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"spider_x": schema.StringAttribute{
								Optional: true, Computed: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"mldsa65_verify": schema.StringAttribute{
								Optional: true, Computed: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
				},
			},
			"tls_settings": schema.SingleNestedBlock{
				Description: "TLS transport settings (requires security = 'tls').",
				Attributes: map[string]schema.Attribute{
					"server_name": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "SNI/server name for TLS (e.g. the domain the client should verify).",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"fingerprint": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "TLS ClientHello fingerprint (e.g. 'chrome', 'firefox', 'random').",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"allow_insecure": schema.BoolAttribute{
						Optional: true, Computed: true,
						Description: "Accept the server certificate only if verification succeeds; set true to permit an insecure connection.",
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"alpn": schema.ListAttribute{
						Optional: true, Computed: true,
						ElementType: types.StringType,
						Description: "ALPN protocols to offer (e.g. ['h2', 'http/1.1']).",
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"min_version": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "Minimum TLS version (e.g. '1.2').",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"max_version": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "Maximum TLS version (e.g. '1.3').",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"cipher": schema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"tcp_settings": schema.SingleNestedBlock{
				Description: "TCP transport settings.",
				Attributes: map[string]schema.Attribute{
					"accept_proxy_protocol": schema.BoolAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"header_type": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "Header type (e.g. 'none', 'http').",
						Validators:  tcpHeaderTypeValidators(),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"ws_settings": schema.SingleNestedBlock{
				Description: "WebSocket transport settings.",
				Attributes: map[string]schema.Attribute{
					"path": schema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"headers": schema.MapAttribute{
						Optional:    true,
						Computed:    true,
						ElementType: types.StringType,
						Description: "Custom headers as key-value pairs.",
						PlanModifiers: []planmodifier.Map{
							mapplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"grpc_settings": schema.SingleNestedBlock{
				Description: "gRPC transport settings.",
				Attributes: map[string]schema.Attribute{
					"service_name": schema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"multi_mode": schema.BoolAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"idle_timeout": schema.Int64Attribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"health_check_timeout": schema.Int64Attribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"permit_without_stream": schema.BoolAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"initial_windows_size": schema.Int64Attribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"httpupgrade_settings": schema.SingleNestedBlock{
				Description: "HTTP Upgrade transport settings.",
				Attributes: map[string]schema.Attribute{
					"path": schema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"host": schema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"xhttp_settings": schema.SingleNestedBlock{
				Description: "XHTTP transport settings.",
				Attributes: map[string]schema.Attribute{
					"path": schema.StringAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"mode": schema.StringAttribute{
						Optional: true, Computed: true,
						Validators: xhttpModeValidators(),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"no_sse_header": schema.BoolAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"keep_alive_interval": schema.Int64Attribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"x_padding_bytes": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "xPadding bytes range (e.g. '100-1000').",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"x_padding_obfs_mode": schema.BoolAttribute{
						Optional: true, Computed: true,
						Description: "Enable xPadding obfuscation mode.",
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"x_padding_key": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "xPadding encryption key.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"x_padding_header": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "xPadding header name.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"x_padding_placement": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "xPadding placement (e.g. 'header', 'body').",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"x_padding_method": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "xPadding method.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"kcp_settings": schema.SingleNestedBlock{
				Description: "mKCP transport settings.",
				Attributes: map[string]schema.Attribute{
					"mtu": schema.Int64Attribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"tti": schema.Int64Attribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"uplink_capacity": schema.Int64Attribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"downlink_capacity": schema.Int64Attribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"cwnd_multiplier": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "CWND multiplier.",
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"max_sending_window": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Maximum sending window size.",
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"header_type": schema.StringAttribute{
						Optional: true, Computed: true,
						Description: "Header type (e.g. 'none', 'srtp', 'utp', 'wechat-video', 'dtls', 'wireguard').",
						Validators:  kcpHeaderTypeValidators(),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"hysteria_settings": schema.SingleNestedBlock{
				Description: "Hysteria transport settings.",
				Attributes: map[string]schema.Attribute{
					"protocol": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Hysteria transport protocol.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"version": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "Hysteria version (default 2).",
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"auth": schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Hysteria auth string.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"udp_idle_timeout": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Description: "UDP idle timeout in seconds (default 60).",
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"sockopt": schema.SingleNestedBlock{
				Description: "Socket options.",
				Attributes: map[string]schema.Attribute{
					"mark": schema.Int64Attribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"tcp_keep_alive_interval": schema.Int64Attribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.UseStateForUnknown(),
						},
					},
					"tcp_no_delay": schema.BoolAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"tfo_enable": schema.BoolAttribute{
						Optional: true, Computed: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"tproxy": schema.StringAttribute{
						Optional: true, Computed: true,
						Validators: tproxyValidators(),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"domain_strategy": schema.StringAttribute{
						Optional: true, Computed: true,
						Validators: domainStrategyValidators(),
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Typed model -> untyped map (for buildStreamSettingsJSON)
// ---------------------------------------------------------------------------

func expandStreamSettingsFromModel(m *InboundStreamSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}

	if !m.Network.IsNull() && !m.Network.IsUnknown() {
		out["network"] = m.Network.ValueString()
	}
	if !m.Security.IsNull() && !m.Security.IsUnknown() {
		out["security"] = m.Security.ValueString()
	}
	if len(m.ExternalProxy) > 0 {
		out["external_proxy"] = expandExternalProxyFromModel(m.ExternalProxy)
	}
	if m.RealitySettings != nil {
		if rs := expandRealitySettingsFromModel(m.RealitySettings); len(rs) > 0 {
			out["reality_settings"] = []any{rs}
		}
	}
	if m.TLSSettings != nil {
		if ts := expandTLSSettingsFromModel(m.TLSSettings); len(ts) > 0 {
			out["tls_settings"] = []any{ts}
		}
	}
	if m.TCPSettings != nil {
		out["tcp_settings"] = []any{expandTCPSettingsFromModel(m.TCPSettings)}
	}
	if m.WSSettings != nil {
		out["ws_settings"] = []any{expandWSSettingsFromModel(m.WSSettings)}
	}
	if m.GRPCSettings != nil {
		out["grpc_settings"] = []any{expandGRPCSettingsFromModel(m.GRPCSettings)}
	}
	if m.HTTPUpgradeSettings != nil {
		out["httpupgrade_settings"] = []any{expandHTTPUpgradeSettingsFromModel(m.HTTPUpgradeSettings)}
	}
	if m.XHTTPSettings != nil {
		out["xhttp_settings"] = []any{expandXHTTPSettingsFromModel(m.XHTTPSettings)}
	}
	if m.KCPSettings != nil {
		out["kcp_settings"] = []any{expandKCPSettingsFromModel(m.KCPSettings)}
	}
	if m.HysteriaSettings != nil {
		out["hysteria_settings"] = []any{expandHysteriaStreamSettingsFromModel(m.HysteriaSettings)}
	}
	if m.Sockopt != nil {
		out["sockopt"] = []any{expandSockoptFromModel(m.Sockopt)}
	}

	return out
}

func expandExternalProxyFromModel(list []InboundExternalProxyModel) []any {
	out := make([]any, 0, len(list))
	for _, ep := range list {
		entry := map[string]any{}
		if !ep.Dest.IsNull() && !ep.Dest.IsUnknown() {
			entry["dest"] = ep.Dest.ValueString()
		}
		if !ep.Port.IsNull() && !ep.Port.IsUnknown() {
			entry["port"] = int(ep.Port.ValueInt64())
		}
		if !ep.Remark.IsNull() && !ep.Remark.IsUnknown() {
			entry["remark"] = ep.Remark.ValueString()
		}
		if !ep.ForceTLS.IsNull() && !ep.ForceTLS.IsUnknown() {
			entry["force_tls"] = ep.ForceTLS.ValueString()
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func expandRealitySettingsFromModel(m *InboundRealitySettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.Show.IsNull() && !m.Show.IsUnknown() {
		out["show"] = m.Show.ValueBool()
	}
	if !m.Xver.IsNull() && !m.Xver.IsUnknown() {
		out["xver"] = int(m.Xver.ValueInt64())
	}
	if !m.Target.IsNull() && !m.Target.IsUnknown() {
		out["target"] = m.Target.ValueString()
	}
	if !m.ServerNames.IsNull() && !m.ServerNames.IsUnknown() {
		out["server_names"] = typesListToAnySlice(m.ServerNames)
	}
	if !m.PrivateKey.IsNull() && !m.PrivateKey.IsUnknown() {
		out["private_key"] = m.PrivateKey.ValueString()
	}
	if !m.MinClientVer.IsNull() && !m.MinClientVer.IsUnknown() {
		out["min_client_ver"] = m.MinClientVer.ValueString()
	}
	if !m.MaxClientVer.IsNull() && !m.MaxClientVer.IsUnknown() {
		out["max_client_ver"] = m.MaxClientVer.ValueString()
	}
	if !m.MaxTimediff.IsNull() && !m.MaxTimediff.IsUnknown() {
		out["max_timediff"] = m.MaxTimediff.ValueInt64()
	}
	if !m.ShortIDs.IsNull() && !m.ShortIDs.IsUnknown() {
		out["short_ids"] = typesListToAnySlice(m.ShortIDs)
	}
	if !m.Mldsa65Seed.IsNull() && !m.Mldsa65Seed.IsUnknown() {
		out["mldsa65_seed"] = m.Mldsa65Seed.ValueString()
	}
	if !m.Settings.IsNull() && !m.Settings.IsUnknown() {
		if s := expandRealityInnerSettingsFromObject(m.Settings); len(s) > 0 {
			out["settings"] = s
		}
	}
	return out
}

func expandTLSSettingsFromModel(m *InboundTLSSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.ServerName.IsNull() && !m.ServerName.IsUnknown() {
		out["server_name"] = m.ServerName.ValueString()
	}
	if !m.Fingerprint.IsNull() && !m.Fingerprint.IsUnknown() {
		out["fingerprint"] = m.Fingerprint.ValueString()
	}
	if !m.AllowInsecure.IsNull() && !m.AllowInsecure.IsUnknown() {
		out["allow_insecure"] = m.AllowInsecure.ValueBool()
	}
	if !m.Alpn.IsNull() && !m.Alpn.IsUnknown() {
		out["alpn"] = typesListToAnySlice(m.Alpn)
	}
	if !m.MinVersion.IsNull() && !m.MinVersion.IsUnknown() {
		out["min_version"] = m.MinVersion.ValueString()
	}
	if !m.MaxVersion.IsNull() && !m.MaxVersion.IsUnknown() {
		out["max_version"] = m.MaxVersion.ValueString()
	}
	if !m.Cipher.IsNull() && !m.Cipher.IsUnknown() {
		out["cipher"] = m.Cipher.ValueString()
	}
	return out
}

func expandRealityInnerSettingsFromObject(obj types.Object) map[string]any {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}
	out := map[string]any{}
	attrs := obj.Attributes()
	for _, key := range []string{"public_key", "fingerprint", "server_name", "spider_x", "mldsa65_verify"} {
		if v, ok := attrs[key].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
			out[key] = v.ValueString()
		}
	}
	return out
}

func expandTCPSettingsFromModel(m *InboundTCPSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.AcceptProxyProtocol.IsNull() && !m.AcceptProxyProtocol.IsUnknown() {
		out["accept_proxy_protocol"] = m.AcceptProxyProtocol.ValueBool()
	}
	if !m.HeaderType.IsNull() && !m.HeaderType.IsUnknown() {
		out["header_type"] = m.HeaderType.ValueString()
	}
	return out
}

func expandWSSettingsFromModel(m *InboundWSSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.Path.IsNull() && !m.Path.IsUnknown() {
		out["path"] = m.Path.ValueString()
	}
	if !m.Headers.IsNull() && !m.Headers.IsUnknown() {
		headers := map[string]any{}
		for k, v := range m.Headers.Elements() {
			if sv, ok := v.(types.String); ok && !sv.IsNull() && !sv.IsUnknown() {
				headers[k] = sv.ValueString()
			}
		}
		if len(headers) > 0 {
			out["headers"] = headers
		}
	}
	return out
}

func expandGRPCSettingsFromModel(m *InboundGRPCSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.ServiceName.IsNull() && !m.ServiceName.IsUnknown() {
		out["service_name"] = m.ServiceName.ValueString()
	}
	if !m.MultiMode.IsNull() && !m.MultiMode.IsUnknown() {
		out["multi_mode"] = m.MultiMode.ValueBool()
	}
	if !m.IdleTimeout.IsNull() && !m.IdleTimeout.IsUnknown() {
		out["idle_timeout"] = int(m.IdleTimeout.ValueInt64())
	}
	if !m.HealthCheckTimeout.IsNull() && !m.HealthCheckTimeout.IsUnknown() {
		out["health_check_timeout"] = int(m.HealthCheckTimeout.ValueInt64())
	}
	if !m.PermitWithoutStream.IsNull() && !m.PermitWithoutStream.IsUnknown() {
		out["permit_without_stream"] = m.PermitWithoutStream.ValueBool()
	}
	if !m.InitialWindowsSize.IsNull() && !m.InitialWindowsSize.IsUnknown() {
		out["initial_windows_size"] = int(m.InitialWindowsSize.ValueInt64())
	}
	return out
}

func expandHTTPUpgradeSettingsFromModel(m *InboundHTTPUpgradeSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.Path.IsNull() && !m.Path.IsUnknown() {
		out["path"] = m.Path.ValueString()
	}
	if !m.Host.IsNull() && !m.Host.IsUnknown() {
		out["host"] = m.Host.ValueString()
	}
	return out
}

func expandXHTTPSettingsFromModel(m *InboundXHTTPSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.Path.IsNull() && !m.Path.IsUnknown() {
		out["path"] = m.Path.ValueString()
	}
	if !m.Mode.IsNull() && !m.Mode.IsUnknown() {
		out["mode"] = m.Mode.ValueString()
	}
	if !m.NoSSEHeader.IsNull() && !m.NoSSEHeader.IsUnknown() {
		out["no_sse_header"] = m.NoSSEHeader.ValueBool()
	}
	if !m.KeepAliveInterval.IsNull() && !m.KeepAliveInterval.IsUnknown() {
		out["keep_alive_interval"] = int(m.KeepAliveInterval.ValueInt64())
	}
	if !m.XPaddingBytes.IsNull() && !m.XPaddingBytes.IsUnknown() {
		out["x_padding_bytes"] = m.XPaddingBytes.ValueString()
	}
	if !m.XPaddingObfsMode.IsNull() && !m.XPaddingObfsMode.IsUnknown() {
		out["x_padding_obfs_mode"] = m.XPaddingObfsMode.ValueBool()
	}
	if !m.XPaddingKey.IsNull() && !m.XPaddingKey.IsUnknown() {
		out["x_padding_key"] = m.XPaddingKey.ValueString()
	}
	if !m.XPaddingHeader.IsNull() && !m.XPaddingHeader.IsUnknown() {
		out["x_padding_header"] = m.XPaddingHeader.ValueString()
	}
	if !m.XPaddingPlacement.IsNull() && !m.XPaddingPlacement.IsUnknown() {
		out["x_padding_placement"] = m.XPaddingPlacement.ValueString()
	}
	if !m.XPaddingMethod.IsNull() && !m.XPaddingMethod.IsUnknown() {
		out["x_padding_method"] = m.XPaddingMethod.ValueString()
	}
	return out
}

func expandKCPSettingsFromModel(m *InboundKCPSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.MTU.IsNull() && !m.MTU.IsUnknown() {
		out["mtu"] = int(m.MTU.ValueInt64())
	}
	if !m.TTI.IsNull() && !m.TTI.IsUnknown() {
		out["tti"] = int(m.TTI.ValueInt64())
	}
	if !m.UplinkCapacity.IsNull() && !m.UplinkCapacity.IsUnknown() {
		out["uplink_capacity"] = int(m.UplinkCapacity.ValueInt64())
	}
	if !m.DownlinkCapacity.IsNull() && !m.DownlinkCapacity.IsUnknown() {
		out["downlink_capacity"] = int(m.DownlinkCapacity.ValueInt64())
	}
	if !m.CwndMultiplier.IsNull() && !m.CwndMultiplier.IsUnknown() {
		out["cwnd_multiplier"] = int(m.CwndMultiplier.ValueInt64())
	}
	if !m.MaxSendingWindow.IsNull() && !m.MaxSendingWindow.IsUnknown() {
		out["max_sending_window"] = int(m.MaxSendingWindow.ValueInt64())
	}
	if !m.HeaderType.IsNull() && !m.HeaderType.IsUnknown() {
		out["header_type"] = m.HeaderType.ValueString()
	}
	return out
}

func expandHysteriaStreamSettingsFromModel(m *InboundHysteriaStreamSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.Protocol.IsNull() && !m.Protocol.IsUnknown() {
		out["protocol"] = m.Protocol.ValueString()
	}
	if !m.Version.IsNull() && !m.Version.IsUnknown() {
		out["version"] = int(m.Version.ValueInt64())
	}
	if !m.Auth.IsNull() && !m.Auth.IsUnknown() {
		out["auth"] = m.Auth.ValueString()
	}
	if !m.UDPIdleTimeout.IsNull() && !m.UDPIdleTimeout.IsUnknown() {
		out["udp_idle_timeout"] = int(m.UDPIdleTimeout.ValueInt64())
	}
	return out
}

func expandSockoptFromModel(m *InboundSockoptModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.Mark.IsNull() && !m.Mark.IsUnknown() {
		out["mark"] = int(m.Mark.ValueInt64())
	}
	if !m.TCPKeepAliveInterval.IsNull() && !m.TCPKeepAliveInterval.IsUnknown() {
		out["tcp_keep_alive_interval"] = int(m.TCPKeepAliveInterval.ValueInt64())
	}
	if !m.TCPNoDelay.IsNull() && !m.TCPNoDelay.IsUnknown() {
		out["tcp_no_delay"] = m.TCPNoDelay.ValueBool()
	}
	if !m.TFOEnable.IsNull() && !m.TFOEnable.IsUnknown() {
		out["tfo_enable"] = m.TFOEnable.ValueBool()
	}
	if !m.Tproxy.IsNull() && !m.Tproxy.IsUnknown() {
		out["tproxy"] = m.Tproxy.ValueString()
	}
	if !m.DomainStrategy.IsNull() && !m.DomainStrategy.IsUnknown() {
		out["domain_strategy"] = m.DomainStrategy.ValueString()
	}
	return out
}

// ---------------------------------------------------------------------------
// Untyped map -> typed model (from flattenStreamSettings output)
// ---------------------------------------------------------------------------

func flattenStreamSettingsToModel(data map[string]any) *InboundStreamSettingsModel {
	m := &InboundStreamSettingsModel{}

	if v, ok := data["network"].(string); ok {
		m.Network = types.StringValue(v)
	} else {
		m.Network = types.StringNull()
	}

	if v, ok := data["security"].(string); ok {
		m.Security = types.StringValue(v)
	} else {
		m.Security = types.StringNull()
	}

	if v, ok := data["external_proxy"].([]any); ok && len(v) > 0 {
		m.ExternalProxy = flattenExternalProxyToModel(v)
	}

	if v, ok := data["reality_settings"].([]any); ok && len(v) > 0 {
		if raw, ok := v[0].(map[string]any); ok {
			m.RealitySettings = flattenRealitySettingsToModel(raw)
		}
	}

	if v, ok := data["tls_settings"].([]any); ok && len(v) > 0 {
		if raw, ok := v[0].(map[string]any); ok {
			m.TLSSettings = flattenTLSSettingsToModel(raw)
		}
	}

	if v, ok := data["tcp_settings"].([]any); ok && len(v) > 0 {
		if raw, ok := v[0].(map[string]any); ok {
			m.TCPSettings = flattenTCPSettingsToModel(raw)
		}
	}

	if v, ok := data["ws_settings"].([]any); ok && len(v) > 0 {
		if raw, ok := v[0].(map[string]any); ok {
			m.WSSettings = flattenWSSettingsToModel(raw)
		}
	}

	if v, ok := data["grpc_settings"].([]any); ok && len(v) > 0 {
		if raw, ok := v[0].(map[string]any); ok {
			m.GRPCSettings = flattenGRPCSettingsToModel(raw)
		}
	}

	if v, ok := data["httpupgrade_settings"].([]any); ok && len(v) > 0 {
		if raw, ok := v[0].(map[string]any); ok {
			m.HTTPUpgradeSettings = flattenHTTPUpgradeSettingsToModel(raw)
		}
	}

	if v, ok := data["xhttp_settings"].([]any); ok && len(v) > 0 {
		if raw, ok := v[0].(map[string]any); ok {
			m.XHTTPSettings = flattenXHTTPSettingsToModel(raw)
		}
	}

	if v, ok := data["kcp_settings"].([]any); ok && len(v) > 0 {
		if raw, ok := v[0].(map[string]any); ok {
			m.KCPSettings = flattenKCPSettingsToModel(raw)
		}
	}

	if v, ok := data["hysteria_settings"].([]any); ok && len(v) > 0 {
		if raw, ok := v[0].(map[string]any); ok {
			m.HysteriaSettings = flattenHysteriaStreamSettingsToModel(raw)
		}
	}

	if v, ok := data["sockopt"].([]any); ok && len(v) > 0 {
		if raw, ok := v[0].(map[string]any); ok {
			m.Sockopt = flattenSockoptToModel(raw)
		}
	}

	return m
}

func flattenExternalProxyToModel(list []any) []InboundExternalProxyModel {
	out := make([]InboundExternalProxyModel, 0, len(list))
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ep := InboundExternalProxyModel{}
		if v, ok := raw["dest"].(string); ok && v != "" {
			ep.Dest = types.StringValue(v)
		} else {
			ep.Dest = types.StringNull()
		}
		if v, ok := raw["port"]; ok {
			ep.Port = types.Int64Value(int64(intValue(v)))
		} else {
			ep.Port = types.Int64Null()
		}
		if v, ok := raw["remark"].(string); ok && v != "" {
			ep.Remark = types.StringValue(v)
		} else {
			ep.Remark = types.StringNull()
		}
		if v, ok := raw["force_tls"].(string); ok && v != "" {
			ep.ForceTLS = types.StringValue(v)
		} else {
			ep.ForceTLS = types.StringNull()
		}
		out = append(out, ep)
	}
	return out
}

func flattenRealitySettingsToModel(data map[string]any) *InboundRealitySettingsModel {
	m := &InboundRealitySettingsModel{}
	if v, ok := data["show"].(bool); ok {
		m.Show = types.BoolValue(v)
	} else {
		m.Show = types.BoolNull()
	}
	if v, ok := data["xver"]; ok {
		m.Xver = types.Int64Value(int64(intValue(v)))
	} else {
		m.Xver = types.Int64Null()
	}
	if v, ok := data["target"].(string); ok && v != "" {
		m.Target = types.StringValue(v)
	} else {
		m.Target = types.StringNull()
	}
	if v, ok := data["server_names"]; ok {
		m.ServerNames = anySliceToTypesList(v)
	} else {
		m.ServerNames = types.ListNull(types.StringType)
	}
	if v, ok := data["private_key"].(string); ok && v != "" {
		m.PrivateKey = types.StringValue(v)
	} else {
		m.PrivateKey = types.StringNull()
	}
	if v, ok := data["min_client_ver"].(string); ok && v != "" {
		m.MinClientVer = types.StringValue(v)
	} else {
		m.MinClientVer = types.StringNull()
	}
	if v, ok := data["max_client_ver"].(string); ok && v != "" {
		m.MaxClientVer = types.StringValue(v)
	} else {
		m.MaxClientVer = types.StringNull()
	}
	if v, ok := data["max_timediff"]; ok {
		m.MaxTimediff = types.Int64Value(int64Value(v))
	} else {
		m.MaxTimediff = types.Int64Null()
	}
	if v, ok := data["short_ids"]; ok {
		m.ShortIDs = anySliceToTypesList(v)
	} else {
		m.ShortIDs = types.ListNull(types.StringType)
	}
	if v, ok := data["mldsa65_seed"].(string); ok && v != "" {
		m.Mldsa65Seed = types.StringValue(v)
	} else {
		m.Mldsa65Seed = types.StringNull()
	}
	if v, ok := data["settings"].(map[string]any); ok && len(v) > 0 {
		m.Settings = flattenRealityInnerSettingsToObject(v)
	} else {
		m.Settings = types.ObjectNull(realityInnerSettingsAttrTypes)
	}
	return m
}

func flattenTLSSettingsToModel(data map[string]any) *InboundTLSSettingsModel {
	m := &InboundTLSSettingsModel{}
	if v, ok := data["server_name"].(string); ok && v != "" {
		m.ServerName = types.StringValue(v)
	} else {
		m.ServerName = types.StringNull()
	}
	if v, ok := data["fingerprint"].(string); ok && v != "" {
		m.Fingerprint = types.StringValue(v)
	} else {
		m.Fingerprint = types.StringNull()
	}
	if v, ok := data["allow_insecure"].(bool); ok {
		m.AllowInsecure = types.BoolValue(v)
	} else {
		m.AllowInsecure = types.BoolNull()
	}
	if v, ok := data["alpn"]; ok {
		m.Alpn = anySliceToTypesList(v)
	} else {
		m.Alpn = types.ListNull(types.StringType)
	}
	if v, ok := data["min_version"].(string); ok && v != "" {
		m.MinVersion = types.StringValue(v)
	} else {
		m.MinVersion = types.StringNull()
	}
	if v, ok := data["max_version"].(string); ok && v != "" {
		m.MaxVersion = types.StringValue(v)
	} else {
		m.MaxVersion = types.StringNull()
	}
	if v, ok := data["cipher"].(string); ok && v != "" {
		m.Cipher = types.StringValue(v)
	} else {
		m.Cipher = types.StringNull()
	}
	return m
}

func flattenRealityInnerSettingsToObject(data map[string]any) types.Object {
	if len(data) == 0 {
		return types.ObjectNull(realityInnerSettingsAttrTypes)
	}
	attrs := map[string]attr.Value{
		"public_key":     types.StringNull(),
		"fingerprint":    types.StringNull(),
		"server_name":    types.StringNull(),
		"spider_x":       types.StringNull(),
		"mldsa65_verify": types.StringNull(),
	}
	for _, key := range []string{"public_key", "fingerprint", "server_name", "spider_x", "mldsa65_verify"} {
		if v, ok := data[key].(string); ok && v != "" {
			attrs[key] = types.StringValue(v)
		}
	}
	obj, _ := types.ObjectValue(realityInnerSettingsAttrTypes, attrs)
	return obj
}

func flattenTCPSettingsToModel(data map[string]any) *InboundTCPSettingsModel {
	m := &InboundTCPSettingsModel{}
	// header arrives as []any{map["type":...]} from flattenTCPSettings.
	if v, ok := data["accept_proxy_protocol"].(bool); ok {
		m.AcceptProxyProtocol = types.BoolValue(v)
	} else {
		m.AcceptProxyProtocol = types.BoolNull()
	}
	if v, ok := data["header"].([]any); ok && len(v) > 0 {
		if h, ok := v[0].(map[string]any); ok {
			if t, ok := h["type"].(string); ok {
				m.HeaderType = types.StringValue(t)
			} else {
				m.HeaderType = types.StringNull()
			}
		}
	} else if v, ok := data["header_type"].(string); ok {
		m.HeaderType = types.StringValue(v)
	} else {
		m.HeaderType = types.StringNull()
	}
	return m
}

func flattenWSSettingsToModel(data map[string]any) *InboundWSSettingsModel {
	m := &InboundWSSettingsModel{}
	if v, ok := data["path"].(string); ok && v != "" {
		m.Path = types.StringValue(v)
	} else {
		m.Path = types.StringNull()
	}
	if v, ok := data["headers"].(map[string]any); ok && len(v) > 0 {
		m.Headers = anyMapToTypesMap(v)
	} else {
		m.Headers = types.MapNull(types.StringType)
	}
	return m
}

func flattenGRPCSettingsToModel(data map[string]any) *InboundGRPCSettingsModel {
	m := &InboundGRPCSettingsModel{}
	if v, ok := data["service_name"].(string); ok && v != "" {
		m.ServiceName = types.StringValue(v)
	} else {
		m.ServiceName = types.StringNull()
	}
	if v, ok := data["multi_mode"].(bool); ok {
		m.MultiMode = types.BoolValue(v)
	} else {
		m.MultiMode = types.BoolNull()
	}
	if v, ok := data["idle_timeout"]; ok {
		m.IdleTimeout = types.Int64Value(int64(intValue(v)))
	} else {
		m.IdleTimeout = types.Int64Null()
	}
	if v, ok := data["health_check_timeout"]; ok {
		m.HealthCheckTimeout = types.Int64Value(int64(intValue(v)))
	} else {
		m.HealthCheckTimeout = types.Int64Null()
	}
	if v, ok := data["permit_without_stream"].(bool); ok {
		m.PermitWithoutStream = types.BoolValue(v)
	} else {
		m.PermitWithoutStream = types.BoolNull()
	}
	if v, ok := data["initial_windows_size"]; ok {
		m.InitialWindowsSize = types.Int64Value(int64(intValue(v)))
	} else {
		m.InitialWindowsSize = types.Int64Null()
	}
	return m
}

func flattenHTTPUpgradeSettingsToModel(data map[string]any) *InboundHTTPUpgradeSettingsModel {
	m := &InboundHTTPUpgradeSettingsModel{}
	if v, ok := data["path"].(string); ok && v != "" {
		m.Path = types.StringValue(v)
	} else {
		m.Path = types.StringNull()
	}
	if v, ok := data["host"].(string); ok && v != "" {
		m.Host = types.StringValue(v)
	} else {
		m.Host = types.StringNull()
	}
	return m
}

func flattenXHTTPSettingsToModel(data map[string]any) *InboundXHTTPSettingsModel {
	m := &InboundXHTTPSettingsModel{}
	if v, ok := data["path"].(string); ok && v != "" {
		m.Path = types.StringValue(v)
	} else {
		m.Path = types.StringNull()
	}
	if v, ok := data["mode"].(string); ok && v != "" {
		m.Mode = types.StringValue(v)
	} else {
		m.Mode = types.StringNull()
	}
	if v, ok := data["no_sse_header"].(bool); ok {
		m.NoSSEHeader = types.BoolValue(v)
	} else {
		m.NoSSEHeader = types.BoolNull()
	}
	if v, ok := data["keep_alive_interval"]; ok {
		m.KeepAliveInterval = types.Int64Value(int64(intValue(v)))
	} else {
		m.KeepAliveInterval = types.Int64Null()
	}
	if v, ok := data["x_padding_bytes"].(string); ok && v != "" {
		m.XPaddingBytes = types.StringValue(v)
	} else {
		m.XPaddingBytes = types.StringNull()
	}
	if v, ok := data["x_padding_obfs_mode"].(bool); ok {
		m.XPaddingObfsMode = types.BoolValue(v)
	} else {
		m.XPaddingObfsMode = types.BoolNull()
	}
	if v, ok := data["x_padding_key"].(string); ok && v != "" {
		m.XPaddingKey = types.StringValue(v)
	} else {
		m.XPaddingKey = types.StringNull()
	}
	if v, ok := data["x_padding_header"].(string); ok && v != "" {
		m.XPaddingHeader = types.StringValue(v)
	} else {
		m.XPaddingHeader = types.StringNull()
	}
	if v, ok := data["x_padding_placement"].(string); ok && v != "" {
		m.XPaddingPlacement = types.StringValue(v)
	} else {
		m.XPaddingPlacement = types.StringNull()
	}
	if v, ok := data["x_padding_method"].(string); ok && v != "" {
		m.XPaddingMethod = types.StringValue(v)
	} else {
		m.XPaddingMethod = types.StringNull()
	}
	return m
}

func flattenKCPSettingsToModel(data map[string]any) *InboundKCPSettingsModel {
	m := &InboundKCPSettingsModel{}
	if v, ok := data["mtu"]; ok {
		m.MTU = types.Int64Value(int64(intValue(v)))
	} else {
		m.MTU = types.Int64Null()
	}
	if v, ok := data["tti"]; ok {
		m.TTI = types.Int64Value(int64(intValue(v)))
	} else {
		m.TTI = types.Int64Null()
	}
	if v, ok := data["uplink_capacity"]; ok {
		m.UplinkCapacity = types.Int64Value(int64(intValue(v)))
	} else {
		m.UplinkCapacity = types.Int64Null()
	}
	if v, ok := data["downlink_capacity"]; ok {
		m.DownlinkCapacity = types.Int64Value(int64(intValue(v)))
	} else {
		m.DownlinkCapacity = types.Int64Null()
	}
	if v, ok := data["cwnd_multiplier"]; ok {
		m.CwndMultiplier = types.Int64Value(int64(intValue(v)))
	} else {
		m.CwndMultiplier = types.Int64Null()
	}
	if v, ok := data["max_sending_window"]; ok {
		m.MaxSendingWindow = types.Int64Value(int64(intValue(v)))
	} else {
		m.MaxSendingWindow = types.Int64Null()
	}
	if v, ok := data["header_type"].(string); ok && v != "" {
		m.HeaderType = types.StringValue(v)
	} else {
		m.HeaderType = types.StringNull()
	}
	return m
}

func flattenHysteriaStreamSettingsToModel(data map[string]any) *InboundHysteriaStreamSettingsModel {
	m := &InboundHysteriaStreamSettingsModel{}
	if v, ok := data["protocol"].(string); ok {
		m.Protocol = types.StringValue(v)
	} else {
		m.Protocol = types.StringNull()
	}
	if v, ok := data["version"]; ok {
		m.Version = types.Int64Value(int64(intValue(v)))
	} else {
		m.Version = types.Int64Null()
	}
	if v, ok := data["auth"].(string); ok {
		m.Auth = types.StringValue(v)
	} else {
		m.Auth = types.StringNull()
	}
	if v, ok := data["udp_idle_timeout"]; ok {
		m.UDPIdleTimeout = types.Int64Value(int64(intValue(v)))
	} else {
		m.UDPIdleTimeout = types.Int64Null()
	}
	return m
}

func flattenSockoptToModel(data map[string]any) *InboundSockoptModel {
	m := &InboundSockoptModel{}
	if v, ok := data["mark"]; ok {
		m.Mark = types.Int64Value(int64(intValue(v)))
	} else {
		m.Mark = types.Int64Null()
	}
	if v, ok := data["tcp_keep_alive_interval"]; ok {
		m.TCPKeepAliveInterval = types.Int64Value(int64(intValue(v)))
	} else {
		m.TCPKeepAliveInterval = types.Int64Null()
	}
	if v, ok := data["tcp_no_delay"].(bool); ok {
		m.TCPNoDelay = types.BoolValue(v)
	} else {
		m.TCPNoDelay = types.BoolNull()
	}
	if v, ok := data["tfo_enable"].(bool); ok {
		m.TFOEnable = types.BoolValue(v)
	} else {
		m.TFOEnable = types.BoolNull()
	}
	if v, ok := data["tproxy"].(string); ok && v != "" {
		m.Tproxy = types.StringValue(v)
	} else {
		m.Tproxy = types.StringNull()
	}
	if v, ok := data["domain_strategy"].(string); ok && v != "" {
		m.DomainStrategy = types.StringValue(v)
	} else {
		m.DomainStrategy = types.StringNull()
	}
	return m
}

// ---------------------------------------------------------------------------
// JSON string -> untyped map (wraps existing flattenStreamSettings)
// ---------------------------------------------------------------------------

func flattenStreamSettingsToMap(streamJSON string) (map[string]any, error) {
	flat, err := flattenStreamSettings(streamJSON)
	if err != nil {
		return nil, err
	}
	if len(flat) == 0 {
		return nil, nil
	}
	m, ok := flat[0].(map[string]any)
	if !ok {
		return nil, nil
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Map helper: map[string]any -> types.Map of StringType
// ---------------------------------------------------------------------------

func anyMapToTypesMap(m map[string]any) types.Map {
	if len(m) == 0 {
		return types.MapNull(types.StringType)
	}
	elems := map[string]types.String{}
	for k, v := range m {
		if s, ok := v.(string); ok {
			elems[k] = types.StringValue(s)
		}
	}
	result, _ := types.MapValueFrom(nil, types.StringType, elems) //nolint:staticcheck // nil context ok for MapValueFrom
	return result
}
