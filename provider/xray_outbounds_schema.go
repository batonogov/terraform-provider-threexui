package provider

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// Typed models
// ---------------------------------------------------------------------------

type XrayOutboundsModel struct {
	ID       types.String        `tfsdk:"id"`
	Outbound []XrayOutboundEntry `tfsdk:"outbound"`
}

type XrayOutboundEntry struct {
	Tag                 types.String                 `tfsdk:"tag"`
	Protocol            types.String                 `tfsdk:"protocol"`
	SendThrough         types.String                 `tfsdk:"send_through"`
	TargetStrategy      types.String                 `tfsdk:"target_strategy"`
	Mux                 []XrayOutboundMux            `tfsdk:"mux"`
	StreamSettings      *InboundStreamSettingsModel  `tfsdk:"stream_settings"`
	FreedomSettings     []XrayFreedomSettings        `tfsdk:"freedom_settings"`
	BlackholeSettings   []XrayBlackholeSettings      `tfsdk:"blackhole_settings"`
	DNSSettings         []XrayOutboundDNSSettings    `tfsdk:"dns_settings"`
	VmessSettings       []XrayVmessOutSettings       `tfsdk:"vmess_settings"`
	VlessSettings       []XrayVlessOutSettings       `tfsdk:"vless_settings"`
	TrojanSettings      []XrayTrojanOutSettings      `tfsdk:"trojan_settings"`
	ShadowsocksSettings []XrayShadowsocksOutSettings `tfsdk:"shadowsocks_settings"`
	SocksSettings       []XraySocksOutSettings       `tfsdk:"socks_settings"`
	HTTPSettings        []XrayHTTPOutSettings        `tfsdk:"http_settings"`
	WireguardSettings   []XrayWireguardOutSettings   `tfsdk:"wireguard_settings"`
	HysteriaSettings    []XrayHysteriaOutSettings    `tfsdk:"hysteria_settings"`
}

type XrayOutboundMux struct {
	Enabled         types.Bool   `tfsdk:"enabled"`
	Concurrency     types.Int64  `tfsdk:"concurrency"`
	XudpConcurrency types.Int64  `tfsdk:"xudp_concurrency"`
	XudpProxyUDP443 types.String `tfsdk:"xudp_proxy_udp443"`
}

type XrayFreedomSettings struct {
	DomainStrategy types.String           `tfsdk:"domain_strategy"`
	Redirect       types.String           `tfsdk:"redirect"`
	Fragment       []XrayFreedomFragment  `tfsdk:"fragment"`
	Noises         []XrayFreedomNoise     `tfsdk:"noises"`
	FinalRules     []XrayFreedomFinalRule `tfsdk:"final_rule"`
	IPsBlocked     types.List             `tfsdk:"ips_blocked"` // list of string
}

type XrayFreedomFragment struct {
	Packets  types.String `tfsdk:"packets"`
	Length   types.String `tfsdk:"length"`
	Interval types.String `tfsdk:"interval"`
}

type XrayFreedomNoise struct {
	Type   types.String `tfsdk:"type"`
	Packet types.String `tfsdk:"packet"`
	Delay  types.String `tfsdk:"delay"`
}

type XrayFreedomFinalRule struct {
	Action     types.String `tfsdk:"action"`
	Network    types.String `tfsdk:"network"`
	Port       types.String `tfsdk:"port"`
	IP         types.List   `tfsdk:"ip"` // list of string
	BlockDelay types.String `tfsdk:"block_delay"`
}

type XrayBlackholeSettings struct {
	ResponseType types.String `tfsdk:"response_type"`
}

type XrayOutboundDNSSettings struct {
	Network    types.String `tfsdk:"network"`
	Address    types.String `tfsdk:"address"`
	Port       types.Int64  `tfsdk:"port"`
	NonIPQuery types.String `tfsdk:"non_ip_query"`
	BlockTypes types.List   `tfsdk:"block_types"` // list of int64
}

type XrayVmessOutSettings struct {
	Address  types.String `tfsdk:"address"`
	Port     types.Int64  `tfsdk:"port"`
	ID       types.String `tfsdk:"id"`
	Security types.String `tfsdk:"security"`
}

type XrayVlessOutSettings struct {
	Address    types.String `tfsdk:"address"`
	Port       types.Int64  `tfsdk:"port"`
	ID         types.String `tfsdk:"id"`
	Flow       types.String `tfsdk:"flow"`
	Encryption types.String `tfsdk:"encryption"`
	ReverseTag types.String `tfsdk:"reverse_tag"`
}

type XrayTrojanOutSettings struct {
	Address  types.String `tfsdk:"address"`
	Port     types.Int64  `tfsdk:"port"`
	Password types.String `tfsdk:"password"`
}

type XrayShadowsocksOutSettings struct {
	Address    types.String `tfsdk:"address"`
	Port       types.Int64  `tfsdk:"port"`
	Password   types.String `tfsdk:"password"`
	Method     types.String `tfsdk:"method"`
	UOT        types.Bool   `tfsdk:"uot"`
	UOTVersion types.Int64  `tfsdk:"uot_version"`
}

type XraySocksOutSettings struct {
	Address types.String `tfsdk:"address"`
	Port    types.Int64  `tfsdk:"port"`
	User    types.String `tfsdk:"user"`
	Pass    types.String `tfsdk:"pass"`
}

type XrayHTTPOutSettings struct {
	Address types.String `tfsdk:"address"`
	Port    types.Int64  `tfsdk:"port"`
	User    types.String `tfsdk:"user"`
	Pass    types.String `tfsdk:"pass"`
}

type XrayWireguardOutSettings struct {
	MTU            types.Int64         `tfsdk:"mtu"`
	SecretKey      types.String        `tfsdk:"secret_key"`
	Address        types.List          `tfsdk:"address"` // list of strings
	Workers        types.Int64         `tfsdk:"workers"`
	DomainStrategy types.String        `tfsdk:"domain_strategy"`
	Reserved       types.List          `tfsdk:"reserved"` // list of int64
	NoKernelTun    types.Bool          `tfsdk:"no_kernel_tun"`
	Peer           []XrayWireguardPeer `tfsdk:"peer"`
}

type XrayWireguardPeer struct {
	PublicKey    types.String `tfsdk:"public_key"`
	PreSharedKey types.String `tfsdk:"pre_shared_key"`
	AllowedIPs   types.List   `tfsdk:"allowed_ips"` // list of strings
	Endpoint     types.String `tfsdk:"endpoint"`
	KeepAlive    types.Int64  `tfsdk:"keep_alive"`
}

type XrayHysteriaOutSettings struct {
	Address types.String `tfsdk:"address"`
	Port    types.Int64  `tfsdk:"port"`
	Version types.Int64  `tfsdk:"version"`
}

// ---------------------------------------------------------------------------
// Int64 list helpers (local to this file)
// ---------------------------------------------------------------------------

// expandInt64List converts a types.List of Int64Type to []any ([]int) for the
// untyped map format.
func expandInt64List(l types.List) []any {
	if l.IsNull() || l.IsUnknown() {
		return nil
	}
	elems := l.Elements()
	out := make([]any, 0, len(elems))
	for _, e := range elems {
		if iv, ok := e.(types.Int64); ok && !iv.IsNull() && !iv.IsUnknown() {
			out = append(out, int(iv.ValueInt64()))
		}
	}
	return out
}

// flattenToInt64List converts a []any of numbers (from the untyped map) to a
// types.List of Int64Type. Returns types.ListNull if the slice is nil or empty.
func flattenToInt64List(v any) types.List {
	slice, ok := v.([]any)
	if !ok || len(slice) == 0 {
		return types.ListNull(types.Int64Type)
	}
	elems := make([]attr.Value, 0, len(slice))
	for _, item := range slice {
		elems = append(elems, types.Int64Value(int64(intValue(item))))
	}
	if len(elems) == 0 {
		return types.ListNull(types.Int64Type)
	}
	return types.ListValueMust(types.Int64Type, elems)
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

func xrayOutboundsSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"outbound": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"tag": schema.StringAttribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"protocol": schema.StringAttribute{
							Required:    true,
							Description: "Outbound protocol (freedom, blackhole, dns, vmess, vless, trojan, shadowsocks, socks, http, wireguard, hysteria).",
						},
						"send_through": schema.StringAttribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"target_strategy": schema.StringAttribute{
							Optional: true, Computed: true,
							Description: "Domain strategy applied to this outbound's destination (3x-ui v3.5.0+, " +
								"xray-core v26.7.11+). One of: AsIs, UseIP, UseIPv4, UseIPv6, UseIPv6v4, " +
								"UseIPv4v6, ForceIPv6v4. Empty/AsIs means xray resolves the destination as-is " +
								"(the key is omitted on the wire when empty, AsIs being xray-core's default). " +
								"Older xray cores silently ignore the unknown key; freedom_settings.domain_strategy " +
								"is a separate, pre-existing field.",
							Validators: []validator.String{
								stringvalidator.OneOf("AsIs", "UseIP", "UseIPv4", "UseIPv6", "UseIPv6v4", "UseIPv4v6", "ForceIPv6v4"),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
					Blocks: map[string]schema.Block{
						"mux":                  singletonListNestedBlock(muxBlock()),
						"stream_settings":      outboundStreamSettingsBlock(),
						"freedom_settings":     singletonListNestedBlock(freedomSettingsBlock()),
						"blackhole_settings":   singletonListNestedBlock(blackholeSettingsBlock()),
						"dns_settings":         singletonListNestedBlock(dnsSettingsBlock()),
						"vmess_settings":       singletonListNestedBlock(vmessSettingsBlock()),
						"vless_settings":       singletonListNestedBlock(vlessSettingsBlock()),
						"trojan_settings":      singletonListNestedBlock(trojanSettingsBlock()),
						"shadowsocks_settings": singletonListNestedBlock(shadowsocksSettingsBlock()),
						"socks_settings":       singletonListNestedBlock(socksSettingsBlock()),
						"http_settings":        singletonListNestedBlock(httpSettingsBlock()),
						"wireguard_settings":   singletonListNestedBlock(wireguardSettingsBlock()),
						"hysteria_settings":    singletonListNestedBlock(hysteriaSettingsBlock()),
					},
				},
			},
		},
	}
}

// outboundStreamSettingsBlock wraps the shared inbound stream-settings schema
// for outbound use. Unlike inbounds, an outbound's streamSettings was previously
// unmodeled and would be silently dropped on any outbound write (the whole
// outbounds array is replaced via setJSONPath). Marking the object Optional +
// Computed + UseStateForUnknown lets an omitted stream_settings survive an
// unrelated outbound update by inheriting prior state. SingleNestedBlock has no
// Optional/Computed/Required fields of its own; the flags are set on the
// objectplanmodifier via the BlockWithObjectPlanModifiers interface, which
// SingleNestedBlock implements.
func outboundStreamSettingsBlock() schema.SingleNestedBlock {
	block := inboundStreamSettingsBlockSchema()
	block.PlanModifiers = []planmodifier.Object{
		objectplanmodifier.UseStateForUnknown(),
	}
	return block
}

// muxBlock returns the mux schema block shared across all outbound types.
func muxBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"enabled": schema.BoolAttribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
				"concurrency": schema.Int64Attribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"xudp_concurrency": schema.Int64Attribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.Int64{
						int64planmodifier.UseStateForUnknown(),
					},
				},
				"xudp_proxy_udp443": schema.StringAttribute{
					Optional: true, Computed: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Typed model -> untyped map (for buildXrayOutboundsJSON)
// ---------------------------------------------------------------------------

// hasNonEmptyEntries returns true if at least one entry in the slice is a
// non-empty map. This prevents emitting keys whose expand produced only empty
// maps (e.g. when all typed fields are null/unknown).
func hasNonEmptyEntries(list []any) bool {
	for _, item := range list {
		if m, ok := item.(map[string]any); ok && len(m) > 0 {
			return true
		}
	}
	return false
}

// expandXrayOutbounds converts the typed model to the untyped map format that
// buildXrayOutboundsJSON expects.
func expandXrayOutbounds(m *XrayOutboundsModel) map[string]any {
	payload := map[string]any{}
	if m.Outbound == nil {
		return payload
	}

	outbounds := make([]any, 0, len(m.Outbound))
	for _, ob := range m.Outbound {
		entry := map[string]any{}

		if !ob.Tag.IsNull() && !ob.Tag.IsUnknown() {
			entry["tag"] = ob.Tag.ValueString()
		}
		if !ob.Protocol.IsNull() && !ob.Protocol.IsUnknown() {
			entry["protocol"] = ob.Protocol.ValueString()
		}
		if !ob.SendThrough.IsNull() && !ob.SendThrough.IsUnknown() {
			entry["send_through"] = ob.SendThrough.ValueString()
		}
		if !ob.TargetStrategy.IsNull() && !ob.TargetStrategy.IsUnknown() {
			entry["target_strategy"] = ob.TargetStrategy.ValueString()
		}

		// Mux
		if len(ob.Mux) > 0 {
			if result := expandOutboundMuxFromModel(ob.Mux); hasNonEmptyEntries(result) {
				entry["mux"] = result
			}
		}

		// Stream settings
		if ob.StreamSettings != nil {
			if result := expandStreamSettingsFromModel(ob.StreamSettings); len(result) > 0 {
				entry["stream_settings"] = []any{result}
			}
		}

		// Protocol-specific settings
		if len(ob.FreedomSettings) > 0 {
			if result := expandFreedomSettingsFromModel(ob.FreedomSettings); hasNonEmptyEntries(result) {
				entry["freedom_settings"] = result
			}
		}
		if len(ob.BlackholeSettings) > 0 {
			if result := expandBlackholeSettingsFromModel(ob.BlackholeSettings); hasNonEmptyEntries(result) {
				entry["blackhole_settings"] = result
			}
		}
		if len(ob.DNSSettings) > 0 {
			if result := expandDNSSettingsFromModel(ob.DNSSettings); hasNonEmptyEntries(result) {
				entry["dns_settings"] = result
			}
		}
		if len(ob.VmessSettings) > 0 {
			if result := expandVmessSettingsFromModel(ob.VmessSettings); hasNonEmptyEntries(result) {
				entry["vmess_settings"] = result
			}
		}
		if len(ob.VlessSettings) > 0 {
			if result := expandVlessSettingsFromModel(ob.VlessSettings); hasNonEmptyEntries(result) {
				entry["vless_settings"] = result
			}
		}
		if len(ob.TrojanSettings) > 0 {
			if result := expandTrojanSettingsFromModel(ob.TrojanSettings); hasNonEmptyEntries(result) {
				entry["trojan_settings"] = result
			}
		}
		if len(ob.ShadowsocksSettings) > 0 {
			if result := expandShadowsocksSettingsFromModel(ob.ShadowsocksSettings); hasNonEmptyEntries(result) {
				entry["shadowsocks_settings"] = result
			}
		}
		if len(ob.SocksSettings) > 0 {
			if result := expandSocksSettingsFromModel(ob.SocksSettings); hasNonEmptyEntries(result) {
				entry["socks_settings"] = result
			}
		}
		if len(ob.HTTPSettings) > 0 {
			if result := expandHTTPSettingsFromModel(ob.HTTPSettings); hasNonEmptyEntries(result) {
				entry["http_settings"] = result
			}
		}
		if len(ob.WireguardSettings) > 0 {
			if result := expandWireguardSettingsFromModel(ob.WireguardSettings); hasNonEmptyEntries(result) {
				entry["wireguard_settings"] = result
			}
		}
		if len(ob.HysteriaSettings) > 0 {
			if result := expandHysteriaSettingsFromModel(ob.HysteriaSettings); hasNonEmptyEntries(result) {
				entry["hysteria_settings"] = result
			}
		}

		if len(entry) > 0 {
			outbounds = append(outbounds, entry)
		}
	}

	payload["outbound"] = outbounds
	return payload
}

func expandOutboundMuxFromModel(muxList []XrayOutboundMux) []any {
	out := make([]any, 0, len(muxList))
	for _, mux := range muxList {
		entry := map[string]any{}
		if !mux.Enabled.IsNull() && !mux.Enabled.IsUnknown() {
			entry["enabled"] = mux.Enabled.ValueBool()
		}
		if !mux.Concurrency.IsNull() && !mux.Concurrency.IsUnknown() {
			entry["concurrency"] = int(mux.Concurrency.ValueInt64())
		}
		if !mux.XudpConcurrency.IsNull() && !mux.XudpConcurrency.IsUnknown() {
			entry["xudp_concurrency"] = int(mux.XudpConcurrency.ValueInt64())
		}
		if !mux.XudpProxyUDP443.IsNull() && !mux.XudpProxyUDP443.IsUnknown() {
			entry["xudp_proxy_udp443"] = mux.XudpProxyUDP443.ValueString()
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Untyped map -> typed model (from flattenXrayOutboundsToMap output)
// ---------------------------------------------------------------------------

// flattenXrayOutbounds converts the output of flattenXrayOutboundsToMap back
// to the typed model.
func flattenXrayOutbounds(data map[string]any) *XrayOutboundsModel {
	m := &XrayOutboundsModel{
		ID: types.StringValue("xray_outbounds"),
	}

	v, ok := data["outbound"]
	if !ok {
		return m
	}

	list, ok := v.([]any)
	if !ok {
		return m
	}

	outbounds := make([]XrayOutboundEntry, 0, len(list))
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}

		entry := XrayOutboundEntry{}

		// Tag
		if v, ok := raw["tag"].(string); ok && v != "" {
			entry.Tag = types.StringValue(v)
		} else {
			entry.Tag = types.StringNull()
		}

		// Protocol
		if v, ok := raw["protocol"].(string); ok && v != "" {
			entry.Protocol = types.StringValue(v)
		} else {
			entry.Protocol = types.StringNull()
		}

		// SendThrough
		if v, ok := raw["send_through"].(string); ok && v != "" {
			entry.SendThrough = types.StringValue(v)
		} else {
			entry.SendThrough = types.StringNull()
		}
		// TargetStrategy
		if v, ok := raw["target_strategy"].(string); ok && v != "" {
			entry.TargetStrategy = types.StringValue(v)
		} else {
			entry.TargetStrategy = types.StringNull()
		}

		// Mux
		if v, ok := raw["mux"].([]any); ok && len(v) > 0 {
			entry.Mux = flattenOutboundMuxToModel(v)
		}

		// Stream settings
		if v, ok := raw["stream_settings"].([]any); ok && len(v) > 0 {
			if rawSS, ok := v[0].(map[string]any); ok {
				entry.StreamSettings = flattenStreamSettingsToModel(rawSS)
			}
		}

		// Protocol-specific settings
		if v, ok := raw["freedom_settings"].([]any); ok && len(v) > 0 {
			entry.FreedomSettings = flattenFreedomSettingsToModel(v)
		}
		if v, ok := raw["blackhole_settings"].([]any); ok && len(v) > 0 {
			entry.BlackholeSettings = flattenBlackholeSettingsToModel(v)
		}
		if v, ok := raw["dns_settings"].([]any); ok && len(v) > 0 {
			entry.DNSSettings = flattenDNSSettingsToModel(v)
		}
		if v, ok := raw["vmess_settings"].([]any); ok && len(v) > 0 {
			entry.VmessSettings = flattenVmessSettingsToModel(v)
		}
		if v, ok := raw["vless_settings"].([]any); ok && len(v) > 0 {
			entry.VlessSettings = flattenVlessSettingsToModel(v)
		}
		if v, ok := raw["trojan_settings"].([]any); ok && len(v) > 0 {
			entry.TrojanSettings = flattenTrojanSettingsToModel(v)
		}
		if v, ok := raw["shadowsocks_settings"].([]any); ok && len(v) > 0 {
			entry.ShadowsocksSettings = flattenShadowsocksSettingsToModel(v)
		}
		if v, ok := raw["socks_settings"].([]any); ok && len(v) > 0 {
			entry.SocksSettings = flattenSocksSettingsToModel(v)
		}
		if v, ok := raw["http_settings"].([]any); ok && len(v) > 0 {
			entry.HTTPSettings = flattenHTTPSettingsToModel(v)
		}
		if v, ok := raw["wireguard_settings"].([]any); ok && len(v) > 0 {
			entry.WireguardSettings = flattenWireguardSettingsToModel(v)
		}
		if v, ok := raw["hysteria_settings"].([]any); ok && len(v) > 0 {
			entry.HysteriaSettings = flattenHysteriaSettingsToModel(v)
		}

		outbounds = append(outbounds, entry)
	}

	m.Outbound = outbounds
	return m
}

func flattenOutboundMuxToModel(list []any) []XrayOutboundMux {
	out := make([]XrayOutboundMux, 0, len(list))
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		mux := XrayOutboundMux{}

		if v, ok := raw["enabled"].(bool); ok {
			mux.Enabled = types.BoolValue(v)
		} else {
			mux.Enabled = types.BoolNull()
		}

		if v, ok := raw["concurrency"]; ok {
			mux.Concurrency = types.Int64Value(int64(intValue(v)))
		} else {
			mux.Concurrency = types.Int64Null()
		}

		if v, ok := raw["xudp_concurrency"]; ok {
			mux.XudpConcurrency = types.Int64Value(int64(intValue(v)))
		} else {
			mux.XudpConcurrency = types.Int64Null()
		}

		if v, ok := raw["xudp_proxy_udp443"].(string); ok && v != "" {
			mux.XudpProxyUDP443 = types.StringValue(v)
		} else {
			mux.XudpProxyUDP443 = types.StringNull()
		}

		out = append(out, mux)
	}
	return out
}

// ---------------------------------------------------------------------------
// Shared JSON flatten helpers (used by vmess, vless, trojan, shadowsocks,
// socks, http, hysteria outbounds)
// ---------------------------------------------------------------------------

func flattenVnextFirstUser(in map[string]any, fields ...string) map[string]any {
	out := map[string]any{}
	server := firstVnextServer(in)
	if server == nil {
		return out
	}
	if v, ok := server["address"].(string); ok {
		out["address"] = v
	}
	if v, ok := server["port"]; ok {
		out["port"] = intValue(v)
	}
	users, ok := server["users"].([]any)
	if ok && len(users) > 0 {
		user, ok := users[0].(map[string]any)
		if ok {
			for _, f := range fields {
				if v, ok := user[f]; ok {
					out[f] = v
				}
			}
		}
	}
	return out
}

func firstVnextServer(in map[string]any) map[string]any {
	vnext, ok := in["vnext"].([]any)
	if !ok || len(vnext) == 0 {
		return nil
	}
	server, ok := vnext[0].(map[string]any)
	if !ok {
		return nil
	}
	return server
}

func flattenServersFirst(in map[string]any, fields ...string) map[string]any {
	out := map[string]any{}
	servers, ok := in["servers"].([]any)
	if !ok || len(servers) == 0 {
		return out
	}
	server, ok := servers[0].(map[string]any)
	if !ok {
		return out
	}
	if v, ok := server["address"].(string); ok {
		out["address"] = v
	}
	if v, ok := server["port"]; ok {
		out["port"] = intValue(v)
	}
	for _, f := range fields {
		if v, ok := server[f]; ok {
			out[f] = v
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Existing build/flatten functions (untyped map <-> Xray JSON)
// ---------------------------------------------------------------------------

func buildXrayOutboundsJSON(d map[string]any) any {
	v, ok := d["outbound"]
	if !ok {
		return []any{}
	}
	list, ok := v.([]any)
	if !ok {
		return []any{}
	}
	return expandOutbounds(list)
}

func expandOutbounds(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		protocol := ""

		if v, ok := m["tag"].(string); ok && v != "" {
			entry["tag"] = v
		}
		if v, ok := m["protocol"].(string); ok && v != "" {
			protocol = v
			entry["protocol"] = v
		}
		if v, ok := m["send_through"].(string); ok && v != "" {
			entry["sendThrough"] = v
		}
		if v, ok := m["target_strategy"].(string); ok && v != "" {
			entry["targetStrategy"] = v
		}

		// Mux
		if v, ok := m["mux"]; ok {
			if mux := expandOutboundMux(v.([]any)); mux != nil {
				entry["mux"] = mux
			}
		}

		// Stream settings
		if v, ok := m["stream_settings"]; ok {
			if list, ok := v.([]any); ok && len(list) > 0 {
				if ss, ok := list[0].(map[string]any); ok {
					if obj := expandOutboundStreamSettings(ss); obj != nil {
						entry["streamSettings"] = obj
					}
				}
			}
		}

		// Protocol-specific settings
		if settings := expandOutboundSettings(m, protocol); settings != nil {
			entry["settings"] = settings
		}

		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func expandOutboundMux(list []any) map[string]any {
	if len(list) == 0 {
		return nil
	}
	item, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	if v, ok := item["enabled"].(bool); ok {
		out["enabled"] = v
	}
	if v, ok := item["concurrency"].(int); ok && v != 0 {
		out["concurrency"] = v
	}
	if v, ok := item["xudp_concurrency"].(int); ok && v != 0 {
		out["xudpConcurrency"] = v
	}
	if v, ok := item["xudp_proxy_udp443"].(string); ok && v != "" {
		out["xudpProxyUDP443"] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// expandOutboundStreamSettings converts the snake_case stream settings map into
// the camelCase object that xray stores on an outbound, reusing the exact
// encoding the inbound resources send as a JSON string. Returns nil when the
// map carries nothing to emit.
func expandOutboundStreamSettings(ss map[string]any) map[string]any {
	if len(ss) == 0 {
		return nil
	}
	j := buildStreamSettingsJSON(ss)
	if j == "{}" {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(j), &obj); err != nil {
		return nil
	}
	if len(obj) == 0 {
		return nil
	}
	return obj
}

func expandOutboundSettings(m map[string]any, protocol string) map[string]any {
	switch protocol {
	case "freedom":
		return expandFreedomSettings(m)
	case "blackhole":
		return expandBlackholeSettings(m)
	case "dns":
		return expandOutboundDNSSettings(m)
	case "vmess":
		return expandVmessOutSettings(m)
	case "vless":
		return expandVlessOutSettings(m)
	case "trojan":
		return expandTrojanOutSettings(m)
	case "shadowsocks":
		return expandShadowsocksOutSettings(m)
	case "socks":
		return expandSocksOutSettings(m)
	case "http":
		return expandHTTPOutSettings(m)
	case "wireguard":
		return expandWireguardOutSettings(m)
	case "hysteria", "hysteria2":
		return expandHysteriaOutSettings(m)
	default:
		return nil
	}
}

// --- Flatten ---

func flattenXrayOutboundsToMap(data any) map[string]any {
	out := map[string]any{}
	if data == nil {
		return out
	}

	var list []any
	switch v := data.(type) {
	case []any:
		list = v
	case string:
		if err := json.Unmarshal([]byte(v), &list); err != nil {
			return out
		}
	default:
		return out
	}

	out["outbound"] = flattenOutbounds(list)
	return out
}

func flattenOutbounds(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		protocol := ""

		if v, ok := m["tag"].(string); ok {
			entry["tag"] = v
		}
		if v, ok := m["protocol"].(string); ok {
			protocol = v
			entry["protocol"] = v
		}
		if v, ok := m["sendThrough"].(string); ok {
			entry["send_through"] = v
		}
		if v, ok := m["targetStrategy"].(string); ok {
			entry["target_strategy"] = v
		}

		// Mux
		if v, ok := m["mux"].(map[string]any); ok {
			if mux := flattenOutboundMux(v); mux != nil {
				entry["mux"] = []any{mux}
			}
		}

		// Stream settings
		if v, ok := m["streamSettings"].(map[string]any); ok && len(v) > 0 {
			if b, err := json.Marshal(v); err == nil {
				if list, err := flattenStreamSettings(string(b)); err == nil && len(list) > 0 {
					entry["stream_settings"] = list
				}
			}
		}

		// Protocol-specific settings
		settings, _ := m["settings"].(map[string]any)
		flattenOutboundProtocolSettings(entry, settings, protocol)

		out = append(out, entry)
	}
	return out
}

func flattenOutboundMux(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := map[string]any{}
	if v, ok := in["enabled"].(bool); ok {
		out["enabled"] = v
	}
	if v, ok := in["concurrency"]; ok {
		out["concurrency"] = intValue(v)
	}
	if v, ok := in["xudpConcurrency"]; ok {
		out["xudp_concurrency"] = intValue(v)
	}
	if v, ok := in["xudpProxyUDP443"].(string); ok {
		out["xudp_proxy_udp443"] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func flattenOutboundProtocolSettings(entry map[string]any, settings map[string]any, protocol string) {
	if settings == nil {
		return
	}
	switch protocol {
	case "freedom":
		entry["freedom_settings"] = []any{flattenFreedomSettings(settings)}
	case "blackhole":
		entry["blackhole_settings"] = []any{flattenBlackholeSettings(settings)}
	case "dns":
		entry["dns_settings"] = []any{flattenOutboundDNSSettings(settings)}
	case "vmess":
		entry["vmess_settings"] = []any{flattenVmessOutSettings(settings)}
	case "vless":
		entry["vless_settings"] = []any{flattenVlessOutSettings(settings)}
	case "trojan":
		entry["trojan_settings"] = []any{flattenTrojanOutSettings(settings)}
	case "shadowsocks":
		entry["shadowsocks_settings"] = []any{flattenShadowsocksOutSettings(settings)}
	case "socks":
		entry["socks_settings"] = []any{flattenSocksOutSettings(settings)}
	case "http":
		entry["http_settings"] = []any{flattenHTTPOutSettings(settings)}
	case "wireguard":
		entry["wireguard_settings"] = []any{flattenWireguardOutSettings(settings)}
	case "hysteria", "hysteria2":
		entry["hysteria_settings"] = []any{flattenHysteriaOutSettings(settings)}
	}
}
