package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// Per-protocol models
// ---------------------------------------------------------------------------

type InboundVlessSettingsModel struct {
	Decryption   types.String           `tfsdk:"decryption"`
	Encryption   types.String           `tfsdk:"encryption"`
	SelectedAuth types.String           `tfsdk:"selected_auth"`
	Fallback     []InboundFallbackModel `tfsdk:"fallback"`
}

type InboundTrojanSettingsModel struct {
	Fallback []InboundFallbackModel `tfsdk:"fallback"`
}

type InboundShadowsocksSettingsModel struct {
	Method   types.String `tfsdk:"method"`
	Password types.String `tfsdk:"password"`
	Network  types.String `tfsdk:"network"`
	IVCheck  types.Bool   `tfsdk:"iv_check"`
}

type InboundHTTPSettingsModel struct {
	Auth             types.String          `tfsdk:"auth"`
	AllowTransparent types.Bool            `tfsdk:"allow_transparent"`
	Account          []InboundAccountModel `tfsdk:"account"`
}

type InboundSocksSettingsModel struct {
	Auth    types.String          `tfsdk:"auth"`
	UDP     types.Bool            `tfsdk:"udp"`
	IP      types.String          `tfsdk:"ip"`
	Account []InboundAccountModel `tfsdk:"account"`
}

type InboundWireguardSettingsModel struct {
	MTU         types.List                    `tfsdk:"mtu"` // list of int64
	SecretKey   types.String                  `tfsdk:"secret_key"`
	NoKernelTun types.Bool                    `tfsdk:"no_kernel_tun"`
	Gateway     types.List                    `tfsdk:"gateway"` // list of string
	DNS         types.List                    `tfsdk:"dns"`     // list of string
	Peer        []InboundWireguardPeerModel   `tfsdk:"peer"`
	Clients     []InboundWireguardClientModel `tfsdk:"clients"`
}

type InboundDokodemoSettingsModel struct {
	Address        types.String `tfsdk:"address"`
	Port           types.Int64  `tfsdk:"port"`
	RewriteAddress types.String `tfsdk:"rewrite_address"`
	RewritePort    types.Int64  `tfsdk:"rewrite_port"`
	PortMap        types.Map    `tfsdk:"port_map"` // map of string
	Network        types.String `tfsdk:"network"`
	AllowedNetwork types.String `tfsdk:"allowed_network"`
	FollowRedirect types.Bool   `tfsdk:"follow_redirect"`
}

type InboundMixedSettingsModel struct {
	Auth    types.String          `tfsdk:"auth"`
	UDP     types.Bool            `tfsdk:"udp"`
	IP      types.String          `tfsdk:"ip"`
	Account []InboundAccountModel `tfsdk:"account"`
}

type InboundHysteriaSettingsModel struct {
	Version types.Int64 `tfsdk:"version"`
}

type InboundMtprotoSettingsModel struct {
	FakeTlsDomain         types.String                        `tfsdk:"fake_tls_domain"`
	ProxyProtocolListener types.Bool                          `tfsdk:"proxy_protocol_listener"`
	PreferIp              types.String                        `tfsdk:"prefer_ip"`
	Debug                 types.Bool                          `tfsdk:"debug"`
	DomainFronting        []InboundMtprotoDomainFrontingModel `tfsdk:"domain_fronting"`
	OutboundTag           types.String                        `tfsdk:"outbound_tag"`
	RouteThroughXray      types.Bool                          `tfsdk:"route_through_xray"`
	RouteXrayPort         types.Int64                         `tfsdk:"route_xray_port"`
	PublicIpv4            types.String                        `tfsdk:"public_ipv4"`
	PublicIpv6            types.String                        `tfsdk:"public_ipv6"`
}

type InboundMtprotoDomainFrontingModel struct {
	IP            types.String `tfsdk:"ip"`
	Port          types.Int64  `tfsdk:"port"`
	ProxyProtocol types.Bool   `tfsdk:"proxy_protocol"`
}

// Sub-models

type InboundAccountModel struct {
	User types.String `tfsdk:"user"`
	Pass types.String `tfsdk:"pass"`
}

type InboundFallbackModel struct {
	Name types.String `tfsdk:"name"`
	Alpn types.String `tfsdk:"alpn"`
	Path types.String `tfsdk:"path"`
	Dest types.String `tfsdk:"dest"`
	Xver types.Int64  `tfsdk:"xver"`
}

type InboundWireguardPeerModel struct {
	PrivateKey   types.String `tfsdk:"private_key"`
	PublicKey    types.String `tfsdk:"public_key"`
	PreSharedKey types.String `tfsdk:"pre_shared_key"`
	AllowedIPs   types.List   `tfsdk:"allowed_ips"` // list of string
	KeepAlive    types.Int64  `tfsdk:"keep_alive"`
}

// InboundWireguardClientModel is one WireGuard inbound client under the
// multi-client model introduced in 3x-ui v3.4.2 (wireguard_settings.clients[]).
// Each client is a peer the server accepts; the panel stores the keypair so it
// can render a full .conf/QR. Unlike InboundWireguardPeerModel (which uses the
// legacy snake_case settings keys), the multi-client array is serialised with
// camelCase JSON keys to match the upstream WireguardClient zod schema.
type InboundWireguardClientModel struct {
	PrivateKey   types.String `tfsdk:"private_key"`
	PublicKey    types.String `tfsdk:"public_key"`
	PreSharedKey types.String `tfsdk:"pre_shared_key"`
	AllowedIPs   types.List   `tfsdk:"allowed_ips"` // list of string
	KeepAlive    types.Int64  `tfsdk:"keep_alive"`
	Email        types.String `tfsdk:"email"`
	LimitIP      types.Int64  `tfsdk:"limit_ip"`
	TotalGB      types.Int64  `tfsdk:"total_gb"`
	ExpiryTime   types.Int64  `tfsdk:"expiry_time"`
	Enable       types.Bool   `tfsdk:"enable"`
	TgID         types.Int64  `tfsdk:"tg_id"`
	SubID        types.String `tfsdk:"sub_id"`
	Comment      types.String `tfsdk:"comment"`
	Reset        types.Int64  `tfsdk:"reset"`
}

// ---------------------------------------------------------------------------
// Schema blocks
// ---------------------------------------------------------------------------

func inboundSettingsBlockSchemas() map[string]schema.Block {
	return map[string]schema.Block{
		"vless_settings": schema.SingleNestedBlock{
			Description: "Settings for VLESS protocol.",
			Attributes: map[string]schema.Attribute{
				"decryption": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "Decryption method (usually 'none').",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"encryption": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "Encryption method.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"selected_auth": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "Selected authentication type.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
			},
			Blocks: map[string]schema.Block{
				"fallback": schema.ListNestedBlock{
					Description: "Fallback destinations.",
					NestedObject: schema.NestedBlockObject{
						Attributes: inboundFallbackAttributes(),
					},
				},
			},
		},
		"trojan_settings": schema.SingleNestedBlock{
			Description: "Settings for Trojan protocol.",
			Blocks: map[string]schema.Block{
				"fallback": schema.ListNestedBlock{
					Description: "Fallback destinations.",
					NestedObject: schema.NestedBlockObject{
						Attributes: inboundFallbackAttributes(),
					},
				},
			},
		},
		"shadowsocks_settings": schema.SingleNestedBlock{
			Description: "Settings for Shadowsocks protocol.",
			Attributes: map[string]schema.Attribute{
				"method": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "Encryption method (e.g. chacha20-ietf-poly1305, 2022-blake3-aes-256-gcm). On 3x-ui v2.9.3+ the legacy aes-128-gcm/aes-256-gcm ciphers were dropped from the xray user switch and silently route through Shadowsocks-2022; pick a chacha20 variant or a 2022-blake3-* method to stay compatible across the matrix.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"password": schema.StringAttribute{
					Optional: true, Computed: true, Sensitive: true,
					Description: "Password for encryption.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"network": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "Network type (e.g. 'tcp,udp').",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"iv_check": schema.BoolAttribute{
					Optional: true, Computed: true,
					Description: "Enable IV check.",
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
			},
		},
		"http_settings": schema.SingleNestedBlock{
			Description: "Settings for HTTP proxy protocol.",
			Attributes: map[string]schema.Attribute{
				"auth": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "Authentication type (e.g. 'password', 'noauth').",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"allow_transparent": schema.BoolAttribute{
					Optional: true, Computed: true,
					Description: "Allow transparent proxy.",
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
			},
			Blocks: map[string]schema.Block{
				"account": schema.ListNestedBlock{
					Description: "User accounts for authentication.",
					NestedObject: schema.NestedBlockObject{
						Attributes: inboundAccountAttributes(),
					},
				},
			},
		},
		"socks_settings": schema.SingleNestedBlock{
			Description: "Settings for SOCKS proxy protocol.",
			Attributes: map[string]schema.Attribute{
				"auth": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "Authentication type (e.g. 'password', 'noauth').",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"udp": schema.BoolAttribute{
					Optional: true, Computed: true,
					Description: "Enable UDP support.",
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
				"ip": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "IP address for UDP.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
			},
			Blocks: map[string]schema.Block{
				"account": schema.ListNestedBlock{
					Description: "User accounts for authentication.",
					NestedObject: schema.NestedBlockObject{
						Attributes: inboundAccountAttributes(),
					},
				},
			},
		},
		"mixed_settings": schema.SingleNestedBlock{
			Description: "Settings for mixed (HTTP+SOCKS) proxy protocol.",
			Attributes: map[string]schema.Attribute{
				"auth": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "Authentication type (e.g. 'password', 'noauth').",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"udp": schema.BoolAttribute{
					Optional: true, Computed: true,
					Description: "Enable UDP support.",
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
				"ip": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "IP address for UDP.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
			},
			Blocks: map[string]schema.Block{
				"account": schema.ListNestedBlock{
					Description: "User accounts for authentication.",
					NestedObject: schema.NestedBlockObject{
						Attributes: inboundAccountAttributes(),
					},
				},
			},
		},
		"wireguard_settings": schema.SingleNestedBlock{
			Description: "Settings for WireGuard protocol.",
			Attributes: map[string]schema.Attribute{
				"mtu": schema.ListAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.Int64Type,
					Description: "MTU values [IPv4, IPv6].",
					PlanModifiers: []planmodifier.List{
						listplanmodifier.UseStateForUnknown(),
					},
				},
				"secret_key": schema.StringAttribute{
					Optional: true, Computed: true, Sensitive: true,
					Description: "WireGuard secret key.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"no_kernel_tun": schema.BoolAttribute{
					Optional: true, Computed: true,
					Description: "Disable kernel TUN.",
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
				"gateway": schema.ListAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "Gateway addresses.",
					PlanModifiers: []planmodifier.List{
						listplanmodifier.UseStateForUnknown(),
					},
				},
				"dns": schema.ListAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "DNS server addresses.",
					PlanModifiers: []planmodifier.List{
						listplanmodifier.UseStateForUnknown(),
					},
				},
			},
			Blocks: map[string]schema.Block{
				"peer": schema.ListNestedBlock{
					Description: "WireGuard peers.",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"private_key": schema.StringAttribute{
								Optional: true, Computed: true, Sensitive: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"public_key": schema.StringAttribute{
								Optional: true, Computed: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"pre_shared_key": schema.StringAttribute{
								Optional: true, Computed: true, Sensitive: true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"allowed_ips": schema.ListAttribute{
								Optional:    true,
								Computed:    true,
								ElementType: types.StringType,
								PlanModifiers: []planmodifier.List{
									listplanmodifier.UseStateForUnknown(),
								},
							},
							"keep_alive": schema.Int64Attribute{
								Optional: true, Computed: true,
								PlanModifiers: []planmodifier.Int64{
									int64planmodifier.UseStateForUnknown(),
								},
							},
						},
					},
				},
				"clients": wireguardClientsBlock(),
			},
		},
		"amneziawg_settings": amneziawgSettingsBlock(),
		"tuic_settings":      tuicSettingsBlock(),
		"dokodemo_settings": schema.SingleNestedBlock{
			Description: "Settings for Dokodemo-door / tunnel protocol.",
			Attributes: map[string]schema.Attribute{
				"address": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "Destination address.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"port": schema.Int64Attribute{
					Optional: true, Computed: true,
					Description: "Destination port.",
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"rewrite_address": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "Tunnel rewrite address (3x-ui v3.0.2+). For tunnel inbounds, this is mirrored " +
						"with address for compatibility with older panels.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"rewrite_port": schema.Int64Attribute{
					Optional: true, Computed: true,
					Description: "Tunnel rewrite port (3x-ui v3.0.2+). For tunnel inbounds, this is mirrored " +
						"with port for compatibility with older panels.",
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"port_map": schema.MapAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "Port mapping (e.g. {\"80\": \"127.0.0.1:8080\"}).",
					PlanModifiers: []planmodifier.Map{
						mapplanmodifier.UseStateForUnknown(),
					},
				},
				"network": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "Network type (e.g. 'tcp', 'udp', 'tcp,udp').",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"allowed_network": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "Tunnel allowed network (3x-ui v3.0.2+). For tunnel inbounds, this is mirrored " +
						"with network for compatibility with older panels.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"follow_redirect": schema.BoolAttribute{
					Optional: true, Computed: true,
					Description: "Follow redirect.",
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
			},
		},
		"hysteria_settings": schema.SingleNestedBlock{
			Description: "Settings for Hysteria protocol.",
			Attributes: map[string]schema.Attribute{
				"version": schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Description: "Hysteria version (1 or 2, default 2).",
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
			},
		},
		"mtproto_settings": schema.SingleNestedBlock{
			Description: "Settings for MTProto protocol (3x-ui v3.3.0+). fake_tls_domain is the key field used by the panel to heal per-client FakeTLS secrets.",
			Attributes: map[string]schema.Attribute{
				"fake_tls_domain": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "FakeTLS domain (default 'www.cloudflare.com'). Used by the panel to rebuild per-client secret domain suffixes.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"proxy_protocol_listener": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Enable PROXY protocol listener.",
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
				"prefer_ip": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "IP preference: prefer-ipv6, prefer-ipv4, only-ipv6, or only-ipv4.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"debug": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Enable debug mode.",
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
				"outbound_tag": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Outbound tag for routing.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"route_through_xray": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Route traffic through Xray core.",
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
				"route_xray_port": schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Description: "Port for Xray routing.",
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"public_ipv4": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Public IPv4 address.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"public_ipv6": schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Public IPv6 address.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
			},
			Blocks: map[string]schema.Block{
				"domain_fronting": schema.ListNestedBlock{
					Description: "Domain fronting entries.",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"ip": schema.StringAttribute{
								Optional: true, Computed: true,
								Description: "Fronting IP address.",
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"port": schema.Int64Attribute{
								Optional: true, Computed: true,
								Description: "Fronting port.",
								PlanModifiers: []planmodifier.Int64{
									int64planmodifier.UseStateForUnknown(),
								},
							},
							"proxy_protocol": schema.BoolAttribute{
								Optional: true, Computed: true,
								Description: "Enable PROXY protocol for this fronting entry.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
					},
				},
			},
		},
	}
}

func inboundFallbackAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Optional: true, Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"alpn": schema.StringAttribute{
			Optional: true, Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"path": schema.StringAttribute{
			Optional: true, Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"dest": schema.StringAttribute{
			Optional: true, Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"xver": schema.Int64Attribute{
			Optional: true, Computed: true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
	}
}

func inboundAccountAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"user": schema.StringAttribute{
			Optional: true, Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"pass": schema.StringAttribute{
			Optional: true, Computed: true, Sensitive: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Typed model -> untyped map (for buildSettingsJSON)
// ---------------------------------------------------------------------------

func expandSettingsFromModel(protocol string, m *InboundResourceModel) map[string]any {
	switch protocol {
	case "vless":
		return expandVlessInboundSettings(m.VlessSettings)
	case "trojan":
		return expandTrojanInboundSettings(m.TrojanSettings)
	case "shadowsocks":
		return expandShadowsocksInboundSettings(m.ShadowsocksSettings)
	case "http":
		return expandHTTPInboundSettings(m.HTTPSettings)
	case "socks":
		return expandSocksInboundSettings(m.SocksSettings)
	case "mixed":
		return expandMixedInboundSettings(m.MixedSettings)
	case "wireguard":
		return expandWireguardInboundSettings(m.WireguardSettings)
	case "amneziawg":
		return expandAmneziawgInboundSettings(m.AmneziawgSettings)
	case "tuic":
		return expandTuicInboundSettings(m.TuicSettings)
	case "dokodemo-door", "tunnel", "tun":
		return expandDokodemoInboundSettings(protocol, m.DokodemoSettings)
	case "hysteria", "hysteria2":
		return expandHysteriaInboundSettings(m.HysteriaSettings)
	case "mtproto":
		return expandMtprotoInboundSettings(m.MtprotoSettings)
	default:
		return nil
	}
}

func expandVlessInboundSettings(m *InboundVlessSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.Decryption.IsNull() && !m.Decryption.IsUnknown() {
		out["decryption"] = m.Decryption.ValueString()
	}
	if !m.Encryption.IsNull() && !m.Encryption.IsUnknown() {
		out["encryption"] = m.Encryption.ValueString()
	}
	if !m.SelectedAuth.IsNull() && !m.SelectedAuth.IsUnknown() {
		out["selected_auth"] = m.SelectedAuth.ValueString()
	}
	if len(m.Fallback) > 0 {
		out["fallbacks"] = expandFallbacksFromModel(m.Fallback)
	}
	return out
}

func expandTrojanInboundSettings(m *InboundTrojanSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if len(m.Fallback) > 0 {
		out["fallbacks"] = expandFallbacksFromModel(m.Fallback)
	}
	return out
}

func expandShadowsocksInboundSettings(m *InboundShadowsocksSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.Method.IsNull() && !m.Method.IsUnknown() {
		out["method"] = m.Method.ValueString()
	}
	if !m.Password.IsNull() && !m.Password.IsUnknown() {
		out["password"] = m.Password.ValueString()
	}
	if !m.Network.IsNull() && !m.Network.IsUnknown() {
		out["network"] = m.Network.ValueString()
	}
	if !m.IVCheck.IsNull() && !m.IVCheck.IsUnknown() {
		out["iv_check"] = m.IVCheck.ValueBool()
	}
	return out
}

func expandHTTPInboundSettings(m *InboundHTTPSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.Auth.IsNull() && !m.Auth.IsUnknown() {
		out["auth"] = m.Auth.ValueString()
	}
	if !m.AllowTransparent.IsNull() && !m.AllowTransparent.IsUnknown() {
		out["allow_transparent"] = m.AllowTransparent.ValueBool()
	}
	if len(m.Account) > 0 {
		out["accounts"] = expandAccountsFromModel(m.Account)
	}
	return out
}

func expandSocksInboundSettings(m *InboundSocksSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.Auth.IsNull() && !m.Auth.IsUnknown() {
		out["auth"] = m.Auth.ValueString()
	}
	if !m.UDP.IsNull() && !m.UDP.IsUnknown() {
		out["udp"] = m.UDP.ValueBool()
	}
	if !m.IP.IsNull() && !m.IP.IsUnknown() {
		out["ip"] = m.IP.ValueString()
	}
	if len(m.Account) > 0 {
		out["accounts"] = expandAccountsFromModel(m.Account)
	}
	return out
}

func expandMixedInboundSettings(m *InboundMixedSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.Auth.IsNull() && !m.Auth.IsUnknown() {
		out["auth"] = m.Auth.ValueString()
	}
	if !m.UDP.IsNull() && !m.UDP.IsUnknown() {
		out["udp"] = m.UDP.ValueBool()
	}
	if !m.IP.IsNull() && !m.IP.IsUnknown() {
		out["ip"] = m.IP.ValueString()
	}
	if len(m.Account) > 0 {
		out["accounts"] = expandAccountsFromModel(m.Account)
	}
	return out
}

func expandWireguardInboundSettings(m *InboundWireguardSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.MTU.IsNull() && !m.MTU.IsUnknown() {
		out["mtu"] = typesListInt64ToAnySlice(m.MTU)
	}
	if !m.SecretKey.IsNull() && !m.SecretKey.IsUnknown() {
		out["secret_key"] = m.SecretKey.ValueString()
	}
	if !m.NoKernelTun.IsNull() && !m.NoKernelTun.IsUnknown() {
		out["no_kernel_tun"] = m.NoKernelTun.ValueBool()
	}
	if !m.Gateway.IsNull() && !m.Gateway.IsUnknown() {
		out["gateway"] = typesListToAnySlice(m.Gateway)
	}
	if !m.DNS.IsNull() && !m.DNS.IsUnknown() {
		out["dns"] = typesListToAnySlice(m.DNS)
	}
	if len(m.Peer) > 0 {
		out["peers"] = expandWireguardPeersFromModel(m.Peer)
	}
	if len(m.Clients) > 0 {
		out["clients"] = expandWireguardClientsFromModel(m.Clients)
	}
	return out
}

func expandDokodemoInboundSettings(protocol string, m *InboundDokodemoSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	if protocol == "tunnel" || protocol == "tun" {
		return expandTunnelInboundSettings(m)
	}
	out := map[string]any{}
	if !m.Address.IsNull() && !m.Address.IsUnknown() {
		out["address"] = m.Address.ValueString()
	}
	if !m.Port.IsNull() && !m.Port.IsUnknown() {
		out["port"] = int(m.Port.ValueInt64())
	}
	if !m.PortMap.IsNull() && !m.PortMap.IsUnknown() {
		pm := map[string]any{}
		for k, v := range m.PortMap.Elements() {
			if sv, ok := v.(types.String); ok {
				pm[k] = sv.ValueString()
			}
		}
		if len(pm) > 0 {
			out["port_map"] = pm
		}
	}
	if !m.Network.IsNull() && !m.Network.IsUnknown() {
		out["network"] = m.Network.ValueString()
	}
	if !m.FollowRedirect.IsNull() && !m.FollowRedirect.IsUnknown() {
		out["follow_redirect"] = m.FollowRedirect.ValueBool()
	}
	return out
}

func expandTunnelInboundSettings(m *InboundDokodemoSettingsModel) map[string]any {
	out := map[string]any{}

	if address, ok := firstKnownString(m.RewriteAddress, m.Address); ok {
		out["rewrite_address"] = address
		out["address"] = address
	}
	if port, ok := firstKnownInt64(m.RewritePort, m.Port); ok {
		out["rewrite_port"] = int(port)
		out["port"] = int(port)
	}
	if !m.PortMap.IsNull() && !m.PortMap.IsUnknown() {
		pm := map[string]any{}
		for k, v := range m.PortMap.Elements() {
			if sv, ok := v.(types.String); ok {
				pm[k] = sv.ValueString()
			}
		}
		if len(pm) > 0 {
			out["port_map"] = pm
		}
	}
	if network, ok := firstKnownString(m.AllowedNetwork, m.Network); ok {
		out["allowed_network"] = network
		out["network"] = network
	}
	if !m.FollowRedirect.IsNull() && !m.FollowRedirect.IsUnknown() {
		out["follow_redirect"] = m.FollowRedirect.ValueBool()
	}
	return out
}

func firstKnownString(values ...types.String) (string, bool) {
	for _, v := range values {
		if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
			return v.ValueString(), true
		}
	}
	return "", false
}

func firstKnownInt64(values ...types.Int64) (int64, bool) {
	for _, v := range values {
		if !v.IsNull() && !v.IsUnknown() {
			return v.ValueInt64(), true
		}
	}
	return 0, false
}

func expandHysteriaInboundSettings(m *InboundHysteriaSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.Version.IsNull() && !m.Version.IsUnknown() {
		out["version"] = int(m.Version.ValueInt64())
	}
	return out
}

func expandMtprotoInboundSettings(m *InboundMtprotoSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if !m.FakeTlsDomain.IsNull() && !m.FakeTlsDomain.IsUnknown() {
		out["fake_tls_domain"] = m.FakeTlsDomain.ValueString()
	}
	if !m.ProxyProtocolListener.IsNull() && !m.ProxyProtocolListener.IsUnknown() {
		out["proxy_protocol_listener"] = m.ProxyProtocolListener.ValueBool()
	}
	if !m.PreferIp.IsNull() && !m.PreferIp.IsUnknown() {
		out["prefer_ip"] = m.PreferIp.ValueString()
	}
	if !m.Debug.IsNull() && !m.Debug.IsUnknown() {
		out["debug"] = m.Debug.ValueBool()
	}
	if len(m.DomainFronting) > 0 {
		out["domain_fronting"] = expandMtprotoDomainFrontingFromModel(m.DomainFronting)
	}
	if !m.OutboundTag.IsNull() && !m.OutboundTag.IsUnknown() {
		out["outbound_tag"] = m.OutboundTag.ValueString()
	}
	if !m.RouteThroughXray.IsNull() && !m.RouteThroughXray.IsUnknown() {
		out["route_through_xray"] = m.RouteThroughXray.ValueBool()
	}
	if !m.RouteXrayPort.IsNull() && !m.RouteXrayPort.IsUnknown() {
		out["route_xray_port"] = int(m.RouteXrayPort.ValueInt64())
	}
	if !m.PublicIpv4.IsNull() && !m.PublicIpv4.IsUnknown() {
		out["public_ipv4"] = m.PublicIpv4.ValueString()
	}
	if !m.PublicIpv6.IsNull() && !m.PublicIpv6.IsUnknown() {
		out["public_ipv6"] = m.PublicIpv6.ValueString()
	}
	return out
}

func expandMtprotoDomainFrontingFromModel(list []InboundMtprotoDomainFrontingModel) []any {
	out := make([]any, 0, len(list))
	for _, df := range list {
		entry := map[string]any{}
		if !df.IP.IsNull() && !df.IP.IsUnknown() {
			entry["ip"] = df.IP.ValueString()
		}
		if !df.Port.IsNull() && !df.Port.IsUnknown() {
			entry["port"] = int(df.Port.ValueInt64())
		}
		if !df.ProxyProtocol.IsNull() && !df.ProxyProtocol.IsUnknown() {
			entry["proxy_protocol"] = df.ProxyProtocol.ValueBool()
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func expandFallbacksFromModel(list []InboundFallbackModel) []any {
	out := make([]any, 0, len(list))
	for _, fb := range list {
		entry := map[string]any{}
		if !fb.Name.IsNull() && !fb.Name.IsUnknown() {
			entry["name"] = fb.Name.ValueString()
		}
		if !fb.Alpn.IsNull() && !fb.Alpn.IsUnknown() {
			entry["alpn"] = fb.Alpn.ValueString()
		}
		if !fb.Path.IsNull() && !fb.Path.IsUnknown() {
			entry["path"] = fb.Path.ValueString()
		}
		if !fb.Dest.IsNull() && !fb.Dest.IsUnknown() {
			entry["dest"] = fb.Dest.ValueString()
		}
		if !fb.Xver.IsNull() && !fb.Xver.IsUnknown() {
			entry["xver"] = int(fb.Xver.ValueInt64())
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func expandAccountsFromModel(list []InboundAccountModel) []any {
	out := make([]any, 0, len(list))
	for _, acc := range list {
		entry := map[string]any{}
		if !acc.User.IsNull() && !acc.User.IsUnknown() {
			entry["user"] = acc.User.ValueString()
		}
		if !acc.Pass.IsNull() && !acc.Pass.IsUnknown() {
			entry["pass"] = acc.Pass.ValueString()
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func expandWireguardPeersFromModel(list []InboundWireguardPeerModel) []any {
	out := make([]any, 0, len(list))
	for _, p := range list {
		entry := map[string]any{}
		if !p.PrivateKey.IsNull() && !p.PrivateKey.IsUnknown() {
			entry["private_key"] = p.PrivateKey.ValueString()
		}
		if !p.PublicKey.IsNull() && !p.PublicKey.IsUnknown() {
			entry["public_key"] = p.PublicKey.ValueString()
		}
		if !p.PreSharedKey.IsNull() && !p.PreSharedKey.IsUnknown() {
			entry["pre_shared_key"] = p.PreSharedKey.ValueString()
		}
		if !p.AllowedIPs.IsNull() && !p.AllowedIPs.IsUnknown() {
			entry["allowed_ips"] = typesListToAnySlice(p.AllowedIPs)
		}
		if !p.KeepAlive.IsNull() && !p.KeepAlive.IsUnknown() {
			entry["keep_alive"] = int(p.KeepAlive.ValueInt64())
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Untyped map -> typed model (from flattenSettings output)
// ---------------------------------------------------------------------------

func flattenSettingsToModel(protocol string, data map[string]any, m *InboundResourceModel) {
	switch protocol {
	case "vless":
		m.VlessSettings = flattenVlessInboundSettings(data)
	case "trojan":
		m.TrojanSettings = flattenTrojanInboundSettings(data)
	case "shadowsocks":
		m.ShadowsocksSettings = flattenShadowsocksInboundSettings(data)
	case "http":
		m.HTTPSettings = flattenHTTPInboundSettings(data)
	case "socks":
		m.SocksSettings = flattenSocksInboundSettings(data)
	case "mixed":
		m.MixedSettings = flattenMixedInboundSettings(data)
	case "wireguard":
		m.WireguardSettings = flattenWireguardInboundSettings(data)
	case "amneziawg":
		m.AmneziawgSettings = flattenAmneziawgInboundSettings(data)
	case "tuic":
		m.TuicSettings = flattenTuicInboundSettings(data)
	case "dokodemo-door", "tunnel", "tun":
		m.DokodemoSettings = flattenDokodemoInboundSettings(protocol, data)
	case "hysteria", "hysteria2":
		m.HysteriaSettings = flattenHysteriaInboundSettings(data)
	case "mtproto":
		m.MtprotoSettings = flattenMtprotoInboundSettings(data)
	}
}

func flattenVlessInboundSettings(data map[string]any) *InboundVlessSettingsModel {
	if len(data) == 0 {
		return nil
	}
	m := &InboundVlessSettingsModel{}
	if v, ok := data["decryption"].(string); ok {
		m.Decryption = types.StringValue(v)
	} else {
		m.Decryption = types.StringNull()
	}
	if v, ok := data["encryption"].(string); ok {
		m.Encryption = types.StringValue(v)
	} else {
		m.Encryption = types.StringNull()
	}
	if v, ok := data["selected_auth"].(string); ok && v != "" {
		m.SelectedAuth = types.StringValue(v)
	} else {
		m.SelectedAuth = types.StringNull()
	}
	if v, ok := data["fallbacks"].([]any); ok && len(v) > 0 {
		m.Fallback = flattenFallbacksToModel(v)
	}
	return m
}

func flattenTrojanInboundSettings(data map[string]any) *InboundTrojanSettingsModel {
	if len(data) == 0 {
		return nil
	}
	m := &InboundTrojanSettingsModel{}
	if v, ok := data["fallbacks"].([]any); ok && len(v) > 0 {
		m.Fallback = flattenFallbacksToModel(v)
	}
	return m
}

func flattenShadowsocksInboundSettings(data map[string]any) *InboundShadowsocksSettingsModel {
	if len(data) == 0 {
		return nil
	}
	m := &InboundShadowsocksSettingsModel{}
	if v, ok := data["method"].(string); ok {
		m.Method = types.StringValue(v)
	} else {
		m.Method = types.StringNull()
	}
	if v, ok := data["password"].(string); ok {
		m.Password = types.StringValue(v)
	} else {
		m.Password = types.StringNull()
	}
	if v, ok := data["network"].(string); ok {
		m.Network = types.StringValue(v)
	} else {
		m.Network = types.StringNull()
	}
	if v, ok := data["iv_check"].(bool); ok {
		m.IVCheck = types.BoolValue(v)
	} else {
		m.IVCheck = types.BoolNull()
	}
	return m
}

func flattenHTTPInboundSettings(data map[string]any) *InboundHTTPSettingsModel {
	if len(data) == 0 {
		return nil
	}
	m := &InboundHTTPSettingsModel{}
	if v, ok := data["auth"].(string); ok {
		m.Auth = types.StringValue(v)
	} else {
		m.Auth = types.StringNull()
	}
	if v, ok := data["allow_transparent"].(bool); ok {
		m.AllowTransparent = types.BoolValue(v)
	} else {
		m.AllowTransparent = types.BoolNull()
	}
	if v, ok := data["accounts"].([]any); ok && len(v) > 0 {
		m.Account = flattenAccountsToModel(v)
	}
	return m
}

func flattenSocksInboundSettings(data map[string]any) *InboundSocksSettingsModel {
	if len(data) == 0 {
		return nil
	}
	m := &InboundSocksSettingsModel{}
	if v, ok := data["auth"].(string); ok {
		m.Auth = types.StringValue(v)
	} else {
		m.Auth = types.StringNull()
	}
	if v, ok := data["udp"].(bool); ok {
		m.UDP = types.BoolValue(v)
	} else {
		m.UDP = types.BoolNull()
	}
	if v, ok := data["ip"].(string); ok && v != "" {
		m.IP = types.StringValue(v)
	} else {
		m.IP = types.StringNull()
	}
	if v, ok := data["accounts"].([]any); ok && len(v) > 0 {
		m.Account = flattenAccountsToModel(v)
	}
	return m
}

func flattenMixedInboundSettings(data map[string]any) *InboundMixedSettingsModel {
	if len(data) == 0 {
		return nil
	}
	m := &InboundMixedSettingsModel{}
	if v, ok := data["auth"].(string); ok {
		m.Auth = types.StringValue(v)
	} else {
		m.Auth = types.StringNull()
	}
	if v, ok := data["udp"].(bool); ok {
		m.UDP = types.BoolValue(v)
	} else {
		m.UDP = types.BoolNull()
	}
	if v, ok := data["ip"].(string); ok && v != "" {
		m.IP = types.StringValue(v)
	} else {
		m.IP = types.StringNull()
	}
	if v, ok := data["accounts"].([]any); ok && len(v) > 0 {
		m.Account = flattenAccountsToModel(v)
	}
	return m
}

func flattenWireguardInboundSettings(data map[string]any) *InboundWireguardSettingsModel {
	if len(data) == 0 {
		return nil
	}
	m := &InboundWireguardSettingsModel{}
	if v, ok := data["mtu"]; ok {
		switch val := v.(type) {
		case []any:
			elems := make([]attr.Value, len(val))
			for i, item := range val {
				elems[i] = types.Int64Value(int64(intValue(item)))
			}
			m.MTU, _ = types.ListValue(types.Int64Type, elems)
		default:
			n := int64(intValue(v))
			m.MTU, _ = types.ListValue(types.Int64Type, []attr.Value{types.Int64Value(n), types.Int64Value(n)})
		}
	} else {
		m.MTU = types.ListNull(types.Int64Type)
	}
	if v, ok := data["secret_key"].(string); ok && v != "" {
		m.SecretKey = types.StringValue(v)
	} else {
		m.SecretKey = types.StringNull()
	}
	if v, ok := data["no_kernel_tun"].(bool); ok {
		m.NoKernelTun = types.BoolValue(v)
	} else {
		m.NoKernelTun = types.BoolNull()
	}
	if v, ok := data["gateway"]; ok {
		m.Gateway = anySliceToTypesList(v)
	} else {
		m.Gateway = types.ListNull(types.StringType)
	}
	if v, ok := data["dns"]; ok {
		m.DNS = anySliceToTypesList(v)
	} else {
		m.DNS = types.ListNull(types.StringType)
	}
	if v, ok := data["peers"].([]any); ok && len(v) > 0 {
		m.Peer = flattenInboundWireguardPeersToModel(v)
	}
	if v, ok := data["clients"].([]any); ok && len(v) > 0 {
		m.Clients = flattenInboundWireguardClientsToModel(v)
	}
	return m
}

func flattenDokodemoInboundSettings(protocol string, data map[string]any) *InboundDokodemoSettingsModel {
	if len(data) == 0 {
		return nil
	}
	m := &InboundDokodemoSettingsModel{}
	address, hasAddress := firstMapString(data, "address", "rewrite_address")
	if hasAddress {
		m.Address = types.StringValue(address)
	} else {
		m.Address = types.StringNull()
	}
	if v, ok := firstMapValue(data, "port", "rewrite_port"); ok {
		m.Port = types.Int64Value(int64(intValue(v)))
	} else {
		m.Port = types.Int64Null()
	}
	if protocol == "tunnel" || protocol == "tun" {
		rewriteAddress, hasRewriteAddress := firstMapString(data, "rewrite_address", "address")
		if hasRewriteAddress {
			m.RewriteAddress = types.StringValue(rewriteAddress)
		} else {
			m.RewriteAddress = types.StringNull()
		}
		if v, ok := firstMapValue(data, "rewrite_port", "port"); ok {
			m.RewritePort = types.Int64Value(int64(intValue(v)))
		} else {
			m.RewritePort = types.Int64Null()
		}
	} else {
		m.RewriteAddress = types.StringNull()
		m.RewritePort = types.Int64Null()
	}
	if v, ok := data["port_map"]; ok {
		switch pm := v.(type) {
		case map[string]any:
			m.PortMap = anyMapToTypesMap(pm)
		case map[string]string:
			elems := make(map[string]any, len(pm))
			for k, s := range pm {
				elems[k] = s
			}
			m.PortMap = anyMapToTypesMap(elems)
		default:
			m.PortMap = types.MapNull(types.StringType)
		}
	} else {
		m.PortMap = types.MapNull(types.StringType)
	}
	network, hasNetwork := firstMapString(data, "network", "allowed_network")
	if hasNetwork {
		m.Network = types.StringValue(network)
	} else {
		m.Network = types.StringNull()
	}
	if protocol == "tunnel" || protocol == "tun" {
		allowedNetwork, hasAllowedNetwork := firstMapString(data, "allowed_network", "network")
		if hasAllowedNetwork {
			m.AllowedNetwork = types.StringValue(allowedNetwork)
		} else {
			m.AllowedNetwork = types.StringNull()
		}
	} else {
		m.AllowedNetwork = types.StringNull()
	}
	if v, ok := data["follow_redirect"].(bool); ok {
		m.FollowRedirect = types.BoolValue(v)
	} else {
		m.FollowRedirect = types.BoolNull()
	}
	return m
}

func firstMapValue(data map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if v, ok := data[key]; ok {
			return v, true
		}
	}
	return nil, false
}

func firstMapString(data map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if v, ok := data[key].(string); ok && v != "" {
			return v, true
		}
	}
	return "", false
}

func flattenHysteriaInboundSettings(data map[string]any) *InboundHysteriaSettingsModel {
	if len(data) == 0 {
		return nil
	}
	m := &InboundHysteriaSettingsModel{}
	if v, ok := data["version"]; ok {
		m.Version = types.Int64Value(int64(intValue(v)))
	} else {
		m.Version = types.Int64Null()
	}
	return m
}

func flattenMtprotoInboundSettings(data map[string]any) *InboundMtprotoSettingsModel {
	if len(data) == 0 {
		return nil
	}
	m := &InboundMtprotoSettingsModel{}
	if v, ok := data["fake_tls_domain"].(string); ok && v != "" {
		m.FakeTlsDomain = types.StringValue(v)
	} else {
		m.FakeTlsDomain = types.StringNull()
	}
	if v, ok := data["proxy_protocol_listener"].(bool); ok {
		m.ProxyProtocolListener = types.BoolValue(v)
	} else {
		m.ProxyProtocolListener = types.BoolNull()
	}
	if v, ok := data["prefer_ip"].(string); ok && v != "" {
		m.PreferIp = types.StringValue(v)
	} else {
		m.PreferIp = types.StringNull()
	}
	if v, ok := data["debug"].(bool); ok {
		m.Debug = types.BoolValue(v)
	} else {
		m.Debug = types.BoolNull()
	}
	if v, ok := data["domain_fronting"].([]any); ok && len(v) > 0 {
		m.DomainFronting = flattenMtprotoDomainFrontingToModel(v)
	}
	if v, ok := data["outbound_tag"].(string); ok && v != "" {
		m.OutboundTag = types.StringValue(v)
	} else {
		m.OutboundTag = types.StringNull()
	}
	if v, ok := data["route_through_xray"].(bool); ok {
		m.RouteThroughXray = types.BoolValue(v)
	} else {
		m.RouteThroughXray = types.BoolNull()
	}
	if v, ok := data["route_xray_port"]; ok {
		m.RouteXrayPort = types.Int64Value(int64(intValue(v)))
	} else {
		m.RouteXrayPort = types.Int64Null()
	}
	if v, ok := data["public_ipv4"].(string); ok && v != "" {
		m.PublicIpv4 = types.StringValue(v)
	} else {
		m.PublicIpv4 = types.StringNull()
	}
	if v, ok := data["public_ipv6"].(string); ok && v != "" {
		m.PublicIpv6 = types.StringValue(v)
	} else {
		m.PublicIpv6 = types.StringNull()
	}
	return m
}

func flattenMtprotoDomainFrontingToModel(list []any) []InboundMtprotoDomainFrontingModel {
	out := make([]InboundMtprotoDomainFrontingModel, 0, len(list))
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		df := InboundMtprotoDomainFrontingModel{}
		if v, ok := raw["ip"].(string); ok && v != "" {
			df.IP = types.StringValue(v)
		} else {
			df.IP = types.StringNull()
		}
		if v, ok := raw["port"]; ok {
			df.Port = types.Int64Value(int64(intValue(v)))
		} else {
			df.Port = types.Int64Null()
		}
		if v, ok := raw["proxy_protocol"].(bool); ok {
			df.ProxyProtocol = types.BoolValue(v)
		} else {
			df.ProxyProtocol = types.BoolNull()
		}
		out = append(out, df)
	}
	return out
}

func flattenFallbacksToModel(list []any) []InboundFallbackModel {
	out := make([]InboundFallbackModel, 0, len(list))
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		fb := InboundFallbackModel{}
		if v, ok := raw["name"].(string); ok && v != "" {
			fb.Name = types.StringValue(v)
		} else {
			fb.Name = types.StringNull()
		}
		if v, ok := raw["alpn"].(string); ok && v != "" {
			fb.Alpn = types.StringValue(v)
		} else {
			fb.Alpn = types.StringNull()
		}
		if v, ok := raw["path"].(string); ok && v != "" {
			fb.Path = types.StringValue(v)
		} else {
			fb.Path = types.StringNull()
		}
		if v, ok := raw["dest"].(string); ok && v != "" {
			fb.Dest = types.StringValue(v)
		} else {
			fb.Dest = types.StringNull()
		}
		if v, ok := raw["xver"]; ok {
			fb.Xver = types.Int64Value(int64(intValue(v)))
		} else {
			fb.Xver = types.Int64Null()
		}
		out = append(out, fb)
	}
	return out
}

func flattenAccountsToModel(list []any) []InboundAccountModel {
	out := make([]InboundAccountModel, 0, len(list))
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		acc := InboundAccountModel{}
		if v, ok := raw["user"].(string); ok && v != "" {
			acc.User = types.StringValue(v)
		} else {
			acc.User = types.StringNull()
		}
		if v, ok := raw["pass"].(string); ok && v != "" {
			acc.Pass = types.StringValue(v)
		} else {
			acc.Pass = types.StringNull()
		}
		out = append(out, acc)
	}
	return out
}

func flattenInboundWireguardPeersToModel(list []any) []InboundWireguardPeerModel {
	out := make([]InboundWireguardPeerModel, 0, len(list))
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		p := InboundWireguardPeerModel{}
		if v, ok := raw["private_key"].(string); ok && v != "" {
			p.PrivateKey = types.StringValue(v)
		} else {
			p.PrivateKey = types.StringNull()
		}
		if v, ok := raw["public_key"].(string); ok && v != "" {
			p.PublicKey = types.StringValue(v)
		} else {
			p.PublicKey = types.StringNull()
		}
		if v, ok := raw["pre_shared_key"].(string); ok && v != "" {
			p.PreSharedKey = types.StringValue(v)
		} else {
			p.PreSharedKey = types.StringNull()
		}
		if v, ok := raw["allowed_ips"]; ok {
			p.AllowedIPs = anySliceToTypesList(v)
		} else {
			p.AllowedIPs = types.ListNull(types.StringType)
		}
		if v, ok := raw["keep_alive"]; ok {
			p.KeepAlive = types.Int64Value(int64(intValue(v)))
		} else {
			p.KeepAlive = types.Int64Null()
		}
		out = append(out, p)
	}
	return out
}

// wireguardClientsBlock returns the schema.ListNestedBlock for the WireGuard
// multi-client array (3x-ui v3.4.2+). Defined as a helper to keep the
// wireguard_settings Blocks map readable; the peer block stays inline.
func wireguardClientsBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "WireGuard multi-client peers (3x-ui v3.4.2+). Each entry is one client device the server accepts, with its own keypair and traffic limits. Absent/empty on older panels. Use EITHER this `clients` block OR the legacy `peer` block for an inbound, not both — the panel treats them as separate models and populating both yields undefined behavior.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"private_key": schema.StringAttribute{
					Optional: true, Computed: true, Sensitive: true,
					Description: "Client private key (panel renders configs from it; generated server-side when absent).",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"public_key": schema.StringAttribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"pre_shared_key": schema.StringAttribute{
					Optional: true, Computed: true, Sensitive: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"allowed_ips": schema.ListAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "Client tunnel addresses (allocated server-side when empty).",
					PlanModifiers: []planmodifier.List{
						listplanmodifier.UseStateForUnknown(),
					},
				},
				"keep_alive": schema.Int64Attribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"email": schema.StringAttribute{
					Optional: true, Computed: true,
					Description: "Client email identifier. The panel requires a non-empty unique email (it keys traffic counters on it; an empty value is rejected at runtime), so set this even though it is Optional in the schema.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"limit_ip": schema.Int64Attribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"total_gb": schema.Int64Attribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"expiry_time": schema.Int64Attribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"enable": schema.BoolAttribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
				"tg_id": schema.Int64Attribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"sub_id": schema.StringAttribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"comment": schema.StringAttribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"reset": schema.Int64Attribute{
					Optional: true, Computed: true,
					Description: "Traffic-counter reset period in days (0 = no periodic reset).",
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
			},
		},
	}
}

// expandWireguardClientsFromModel serialises typed WG client entries to the
// upstream camelCase JSON shape (wireguard_settings.clients[]). camelCase is
// mandatory here, unlike the legacy peer array which uses snake_case.
func expandWireguardClientsFromModel(list []InboundWireguardClientModel) []any {
	out := make([]any, 0, len(list))
	for _, c := range list {
		entry := map[string]any{}
		if !c.PrivateKey.IsNull() && !c.PrivateKey.IsUnknown() {
			entry["privateKey"] = c.PrivateKey.ValueString()
		}
		if !c.PublicKey.IsNull() && !c.PublicKey.IsUnknown() {
			entry["publicKey"] = c.PublicKey.ValueString()
		}
		if !c.PreSharedKey.IsNull() && !c.PreSharedKey.IsUnknown() {
			entry["preSharedKey"] = c.PreSharedKey.ValueString()
		}
		if !c.AllowedIPs.IsNull() && !c.AllowedIPs.IsUnknown() {
			entry["allowedIPs"] = typesListToAnySlice(c.AllowedIPs)
		}
		if !c.KeepAlive.IsNull() && !c.KeepAlive.IsUnknown() {
			entry["keepAlive"] = int(c.KeepAlive.ValueInt64())
		}
		if !c.Email.IsNull() && !c.Email.IsUnknown() {
			entry["email"] = c.Email.ValueString()
		}
		if !c.LimitIP.IsNull() && !c.LimitIP.IsUnknown() {
			entry["limitIp"] = int(c.LimitIP.ValueInt64())
		}
		if !c.TotalGB.IsNull() && !c.TotalGB.IsUnknown() {
			entry["totalGB"] = c.TotalGB.ValueInt64()
		}
		if !c.ExpiryTime.IsNull() && !c.ExpiryTime.IsUnknown() {
			entry["expiryTime"] = c.ExpiryTime.ValueInt64()
		}
		if !c.Enable.IsNull() && !c.Enable.IsUnknown() {
			entry["enable"] = c.Enable.ValueBool()
		}
		if !c.TgID.IsNull() && !c.TgID.IsUnknown() {
			entry["tgId"] = c.TgID.ValueInt64()
		}
		if !c.SubID.IsNull() && !c.SubID.IsUnknown() {
			entry["subId"] = c.SubID.ValueString()
		}
		if !c.Comment.IsNull() && !c.Comment.IsUnknown() {
			entry["comment"] = c.Comment.ValueString()
		}
		if !c.Reset.IsNull() && !c.Reset.IsUnknown() {
			entry["reset"] = int(c.Reset.ValueInt64())
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

// flattenInboundWireguardClientsToModel parses the upstream camelCase
// wireguard_settings.clients[] array back into typed entries. Returns nil when
// the array is absent (old panels ≤ v3.4.1 never carry this key).
func flattenInboundWireguardClientsToModel(list []any) []InboundWireguardClientModel {
	out := make([]InboundWireguardClientModel, 0, len(list))
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		c := InboundWireguardClientModel{}
		if v, ok := raw["privateKey"].(string); ok && v != "" {
			c.PrivateKey = types.StringValue(v)
		} else {
			c.PrivateKey = types.StringNull()
		}
		if v, ok := raw["publicKey"].(string); ok && v != "" {
			c.PublicKey = types.StringValue(v)
		} else {
			c.PublicKey = types.StringNull()
		}
		if v, ok := raw["preSharedKey"].(string); ok && v != "" {
			c.PreSharedKey = types.StringValue(v)
		} else {
			c.PreSharedKey = types.StringNull()
		}
		if v, ok := raw["allowedIPs"]; ok {
			c.AllowedIPs = anySliceToTypesList(v)
		} else {
			c.AllowedIPs = types.ListNull(types.StringType)
		}
		if v, ok := raw["keepAlive"]; ok {
			c.KeepAlive = types.Int64Value(int64(intValue(v)))
		} else {
			c.KeepAlive = types.Int64Null()
		}
		if v, ok := raw["email"].(string); ok {
			c.Email = types.StringValue(v)
		} else {
			c.Email = types.StringNull()
		}
		if v, ok := raw["limitIp"]; ok {
			c.LimitIP = types.Int64Value(int64(intValue(v)))
		} else {
			c.LimitIP = types.Int64Null()
		}
		if v, ok := raw["totalGB"]; ok {
			c.TotalGB = types.Int64Value(int64(intValue(v)))
		} else {
			c.TotalGB = types.Int64Null()
		}
		if v, ok := raw["expiryTime"]; ok {
			c.ExpiryTime = types.Int64Value(int64(intValue(v)))
		} else {
			c.ExpiryTime = types.Int64Null()
		}
		if v, ok := raw["enable"]; ok {
			c.Enable = types.BoolValue(boolValue(v))
		} else {
			c.Enable = types.BoolNull()
		}
		if v, ok := raw["tgId"]; ok {
			c.TgID = types.Int64Value(int64(intValue(v)))
		} else {
			c.TgID = types.Int64Null()
		}
		if v, ok := raw["subId"].(string); ok {
			c.SubID = types.StringValue(v)
		} else {
			c.SubID = types.StringNull()
		}
		if v, ok := raw["comment"].(string); ok {
			c.Comment = types.StringValue(v)
		} else {
			c.Comment = types.StringNull()
		}
		if v, ok := raw["reset"]; ok {
			c.Reset = types.Int64Value(int64(intValue(v)))
		} else {
			c.Reset = types.Int64Null()
		}
		out = append(out, c)
	}
	return out
}

// ---------------------------------------------------------------------------
// JSON string -> untyped map (wraps existing flattenSettings)
// ---------------------------------------------------------------------------

func flattenSettingsToMap(settingsJSON string, protocol string) (map[string]any, error) {
	flat, err := flattenSettings(settingsJSON, protocol)
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
