package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AmneziaWG (3x-ui v3.7.0+, #441).
//
// Unlike every other inbound protocol, the settings blob is NOT a flat object:
// it is `{"server": {...}, "clients": [...]}`
// (3x-ui-3.7.0/internal/amneziawg/types.go:242-245). The server block is passed
// through settings.go as one opaque nested map rather than being folded into the
// flat snake_case→camelCase key table there, because several of its keys (`mtu`,
// `privateKey`, `publicKey`, `routeThroughXray`) already have a different meaning
// at the top level for WireGuard and MTProto.
//
// Like WireGuard — and unlike vmess/vless/trojan/shadowsocks/hysteria — the
// clients array is managed by `threexui_inbound` itself, not by the separate
// `threexui_inbound_client` resource.

type InboundAmneziawgSettingsModel struct {
	Server  *InboundAmneziawgServerModel  `tfsdk:"server"`
	Clients []InboundAmneziawgClientModel `tfsdk:"clients"`
}

// InboundAmneziawgServerModel mirrors amneziawg.ServerSettings
// (3x-ui-3.7.0/internal/amneziawg/types.go:130-215). The listen port is NOT part
// of it — that is the inbound's own `port`.
type InboundAmneziawgServerModel struct {
	PrivateKey types.String `tfsdk:"private_key"`
	PublicKey  types.String `tfsdk:"public_key"`

	SubnetIP   types.String `tfsdk:"subnet_ip"`
	SubnetCIDR types.Int64  `tfsdk:"subnet_cidr"`
	MTU        types.Int64  `tfsdk:"mtu"`

	PrimaryDNS   types.String `tfsdk:"primary_dns"`
	SecondaryDNS types.String `tfsdk:"secondary_dns"`

	ExternalInterface     types.String `tfsdk:"external_interface"`
	IPv6Enabled           types.Bool   `tfsdk:"ipv6_enabled"`
	IPv6Subnet            types.String `tfsdk:"ipv6_subnet"`
	IPv6ExternalInterface types.String `tfsdk:"ipv6_external_interface"`

	RouteThroughXray types.Bool `tfsdk:"route_through_xray"`

	// Junk-packet and header obfuscation. The panel generates a randomised set
	// when the inbound is created without settings.
	Jc   types.Int64 `tfsdk:"jc"`
	Jmin types.Int64 `tfsdk:"jmin"`
	Jmax types.Int64 `tfsdk:"jmax"`
	S1   types.Int64 `tfsdk:"s1"`
	S2   types.Int64 `tfsdk:"s2"`
	S3   types.Int64 `tfsdk:"s3"`
	S4   types.Int64 `tfsdk:"s4"`

	// h1-h4 and i1-i5 hold numeric-looking values but are strings upstream:
	// each is either a plain integer or a "lo-hi" range.
	H1 types.String `tfsdk:"h1"`
	H2 types.String `tfsdk:"h2"`
	H3 types.String `tfsdk:"h3"`
	H4 types.String `tfsdk:"h4"`
	I1 types.String `tfsdk:"i1"`
	I2 types.String `tfsdk:"i2"`
	I3 types.String `tfsdk:"i3"`
	I4 types.String `tfsdk:"i4"`
	I5 types.String `tfsdk:"i5"`

	HeaderProtectionKey    types.String `tfsdk:"header_protection_key"`
	ContentPaddingAddition types.String `tfsdk:"content_padding_addition"`

	RekeyAfterTime       types.String `tfsdk:"rekey_after_time"`
	RekeyTimeout         types.String `tfsdk:"rekey_timeout"`
	RejectAfterTime      types.String `tfsdk:"reject_after_time"`
	KeepaliveTimeout     types.String `tfsdk:"keepalive_timeout"`
	MaxHandshakeAttempts types.String `tfsdk:"max_handshake_attempts"`

	RandomTrailers types.Bool `tfsdk:"random_trailers"`
	DisableCookies types.Bool `tfsdk:"disable_cookies"`
}

// InboundAmneziawgClientModel is one AmneziaWG peer. The generic client keys are
// the same as everywhere else (model.Client); `forwarded_ports` is AmneziaWG-only.
type InboundAmneziawgClientModel struct {
	Email          types.String `tfsdk:"email"`
	PrivateKey     types.String `tfsdk:"private_key"`
	PublicKey      types.String `tfsdk:"public_key"`
	PreSharedKey   types.String `tfsdk:"pre_shared_key"`
	AllowedIPs     types.List   `tfsdk:"allowed_ips"` // list of string
	KeepAlive      types.Int64  `tfsdk:"keep_alive"`
	ForwardedPorts types.String `tfsdk:"forwarded_ports"`
	Enable         types.Bool   `tfsdk:"enable"`
	LimitIP        types.Int64  `tfsdk:"limit_ip"`
	TotalGB        types.Int64  `tfsdk:"total_gb"`
	ExpiryTime     types.Int64  `tfsdk:"expiry_time"`
	TgID           types.Int64  `tfsdk:"tg_id"`
	SubID          types.String `tfsdk:"sub_id"`
	Comment        types.String `tfsdk:"comment"`
	Reset          types.Int64  `tfsdk:"reset"`
	Group          types.String `tfsdk:"group"`
	// Renewal and bookkeeping fields the panel keeps on every client
	// (model.Client:904-911, :843-844). They are modelled even though they are
	// not AmneziaWG-specific: the inbound rewrites clients[] wholesale on every
	// apply, so a field the provider does not read is silently zeroed the next
	// time an unrelated attribute changes.
	ResetDay        types.Int64  `tfsdk:"reset_day"`
	ResetMax        types.Int64  `tfsdk:"reset_max"`
	TrafficReset    types.String `tfsdk:"traffic_reset"`
	TrafficResetDay types.Int64  `tfsdk:"traffic_reset_day"`
	CreatedAt       types.Int64  `tfsdk:"created_at"`
	UpdatedAt       types.Int64  `tfsdk:"updated_at"`
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

// awgString builds the Optional+Computed string attribute shape every AmneziaWG
// field uses: the panel fills in whatever the configuration omits, so state must
// survive a plan without the value being re-planned as unknown.
func awgString(description string, validators ...validator.String) schema.StringAttribute {
	return schema.StringAttribute{
		Optional: true, Computed: true,
		Description: description,
		Validators:  validators,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

// awgSecret is awgString for a Sensitive field. It carries nonEmpty() for the
// same reason the plain fields do — every AmneziaWG secret is `omitempty`
// upstream, so "" is stripped on save and comes back null. On a Sensitive
// attribute that surfaces as the especially opaque "inconsistent values for
// sensitive attribute" error, so the value is rejected at plan time instead.
func awgSecret(description string, extra ...validator.String) schema.StringAttribute {
	a := awgString(description, append(nonEmpty(), extra...)...)
	a.Sensitive = true
	return a
}

func awgInt(description string, validators ...validator.Int64) schema.Int64Attribute {
	return schema.Int64Attribute{
		Optional: true, Computed: true,
		Description: description,
		Validators:  validators,
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
}

func awgBool(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional: true, Computed: true,
		Description: description,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	}
}

// nonEmpty rejects "" on the fields upstream declares `omitempty`: the panel
// strips them on save, so an empty string could never round-trip and would fail
// the apply with an inconsistent-result error. Omit the attribute instead.
func nonEmpty() []validator.String {
	return []validator.String{stringvalidator.LengthAtLeast(1)}
}

// uintRange matches the canonical form of the "N" / "lo-hi" fields. The panel
// runs CanonicalizeUintRange over them on save (inbound_amneziawg.go:202-208),
// which strips every space — so "110 - 140" is stored as "110-140" and the apply
// fails with an inconsistent-result error. Rejecting the un-canonical spelling
// at plan time turns that into a clear message.
func uintRange() []validator.String {
	return []validator.String{
		stringvalidator.RegexMatches(
			regexp.MustCompile(`^\d+(-\d+)?$`),
			"must be a number or a `lo-hi` range with no spaces (the panel stores it canonicalised, so a spaced value cannot round-trip)",
		),
	}
}

func amneziawgSettingsBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Settings for the AmneziaWG protocol (3x-ui v3.7.0+). AmneziaWG is WireGuard with DPI-resistant obfuscation, run in-process on an embedded userspace device. Peers are managed here through `clients`, not through separate `threexui_inbound_client` resources.",
		Blocks: map[string]schema.Block{
			"server": schema.SingleNestedBlock{
				Description: "AmneziaWG server parameters. Every field is optional: creating an inbound without them makes the panel generate a keypair, a subnet and a randomised obfuscation set, which the provider then reads back into state.",
				Attributes: map[string]schema.Attribute{
					"private_key": awgSecret(
						"Server private key (base64). Leave both server keys unset to let the panel generate the pair. Setting this REQUIRES setting `public_key` to its matching half: the panel derives a public key for clients but never for the server, so a lone `private_key` leaves the previous, unrelated `public_key` in state and in the rendered peer configs — the tunnel stops working while the apply still reports success.",
						stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("public_key")),
					),
					"public_key": awgString("Server public key (base64). The panel does NOT derive this from `private_key` for the server (it only does so per client), so set both together or let the panel generate both."),

					"subnet_ip":   awgString("Tunnel subnet address, e.g. `10.8.1.0`. Defaults to `10.8.1.0` server-side."),
					"subnet_cidr": awgInt("Tunnel subnet prefix length. Defaults to 24 server-side.", int64validator.Between(1, 32)),
					"mtu":         awgInt("Tunnel MTU. Omit to use the embedded device's default of 1420 — the panel strips a zero, so `0` is not a valid way to say \"default\".", int64validator.AtLeast(1)),

					"primary_dns":   awgString("Primary DNS handed to clients. Defaults to `8.8.8.8`. An empty string is meaningful here — it clears the DNS entry instead of restoring the default."),
					"secondary_dns": awgString("Secondary DNS handed to clients. Defaults to `8.8.4.4`. An empty string clears it."),

					"external_interface":      awgString("Host NIC used for egress NAT. Omit to auto-detect.", nonEmpty()...),
					"ipv6_enabled":            awgBool("Enable the IPv6 tunnel. Requires `ipv6_subnet`."),
					"ipv6_subnet":             awgString("IPv6 tunnel prefix, e.g. `fd00:8:1::/64`. Required when `ipv6_enabled` is true.", nonEmpty()...),
					"ipv6_external_interface": awgString("Host NIC used for IPv6 egress. Omit to reuse the IPv4 interface.", nonEmpty()...),

					"route_through_xray": awgBool("Vestigial upstream — the embedded relay is always on and nothing reads this flag. Round-tripped so the settings blob is not silently altered on save."),

					"jc":   awgInt("Junk packet count."),
					"jmin": awgInt("Minimum junk packet size. Must not exceed `jmax`."),
					"jmax": awgInt("Maximum junk packet size."),
					"s1":   awgInt("Init packet junk size. `s1 + 56` must not equal `s2`."),
					"s2":   awgInt("Response packet junk size."),
					"s3":   awgInt("Cookie reply packet junk size.", int64validator.Between(0, 64)),
					"s4":   awgInt("Transport packet junk size.", int64validator.Between(0, 32)),

					"h1": awgString("Init packet magic header: an integer or a `lo-hi` range (0-4294967295). Blank falls back to the classic WireGuard header `1` when a client config is rendered."),
					"h2": awgString("Response packet magic header. Blank falls back to `2`."),
					"h3": awgString("Underload packet magic header. Blank falls back to `3`."),
					"h4": awgString("Transport packet magic header. Blank falls back to `4`."),

					"i1": awgString("Custom signature packet 1, e.g. `<r 64>`.", nonEmpty()...),
					"i2": awgString("Custom signature packet 2.", nonEmpty()...),
					"i3": awgString("Custom signature packet 3.", nonEmpty()...),
					"i4": awgString("Custom signature packet 4.", nonEmpty()...),
					"i5": awgString("Custom signature packet 5.", nonEmpty()...),

					"header_protection_key":    awgSecret("Header-protection key: base64 of exactly 32 bytes. When set, all of `s1`-`s4` must be at least 12."),
					"content_padding_addition": awgString("Extra content padding, as an integer or a `lo-hi` range.", append(nonEmpty(), uintRange()...)...),

					"rekey_after_time":       awgString("Rekey interval in seconds, as an integer or a `lo-hi` range. The maximum must be lower than the minimum of `reject_after_time`.", append(nonEmpty(), uintRange()...)...),
					"rekey_timeout":          awgString("Rekey timeout in seconds, integer or `lo-hi` range.", append(nonEmpty(), uintRange()...)...),
					"reject_after_time":      awgString("Session reject time in seconds, integer or `lo-hi` range.", append(nonEmpty(), uintRange()...)...),
					"keepalive_timeout":      awgString("Keepalive timeout in seconds, integer or `lo-hi` range.", append(nonEmpty(), uintRange()...)...),
					"max_handshake_attempts": awgString("Maximum handshake attempts, integer or `lo-hi` range.", append(nonEmpty(), uintRange()...)...),

					"random_trailers": awgBool("Append random trailers to packets."),
					"disable_cookies": awgBool("Disable the cookie-reply mechanism."),
				},
			},
			"clients": amneziawgClientsBlock(),
		},
	}
}

func amneziawgClientsBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "AmneziaWG peers. Each entry is one client device the server accepts. Managed by `threexui_inbound` itself — do NOT attach `threexui_inbound_client` resources to an AmneziaWG inbound.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"email":       awgString("Client email identifier. The panel keys traffic counters on it and requires a non-empty unique value, so set this even though the schema marks it Optional."),
				"private_key": awgSecret("Client private key. Optional: the panel stores it only to render a ready-made peer config, so it can be left out when the key is kept elsewhere. It is NOT generated on this path."),
				// Required, unlike its WireGuard counterpart, because the panel
				// rejects a keyless peer outright on inbound create AND update
				// ("wireguard client requires a key" — internal/web/service/inbound.go:1042-1044,
				// client_inbound_apply.go:441). Key generation and the
				// privateKey→publicKey derivation live only on the /panel/api/clients
				// CRUD endpoints, which do not own these peers, so an omitted value
				// could never be filled in — it would fail every apply. Required turns
				// that into a plan-time error instead.
				"public_key": schema.StringAttribute{
					Required:    true,
					Description: "Client public key (base64). Required: the panel rejects an AmneziaWG peer without one, and does not derive it from `private_key` on the inbound path.",
				},
				"pre_shared_key": awgSecret("Optional pre-shared key for this peer."),
				"allowed_ips": schema.ListAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "Peer tunnel addresses, e.g. `[\"10.9.1.2/32\"]`. Set at least one: address allocation happens on the `/panel/api/clients` endpoints, which do not own these peers, so a peer declared here without addresses is saved unroutable. Normalised server-side — a bare address becomes `/32` and duplicates are dropped.",
					Validators: []validator.List{
						// The key is `omitempty` upstream, so an empty list is stripped
						// on save and reads back null, failing the apply.
						listvalidator.SizeAtLeast(1),
					},
					PlanModifiers: []planmodifier.List{
						listplanmodifier.UseStateForUnknown(),
					},
				},
				"keep_alive":      awgInt("Persistent keepalive in seconds. Omit to disable it — the panel strips a zero, so `0` cannot round-trip.", int64validator.AtLeast(1)),
				"forwarded_ports": awgString("Ports DNAT-forwarded to this peer, e.g. `80,443,8000-8100`. Expands to at most 100 ports. The panel rejects a spec that collides with the panel's own port, any enabled inbound's port, or this inbound's SOCKS relay port (65100 + inbound id); malformed tokens are silently dropped.", nonEmpty()...),
				"enable":          awgBool("Whether the client is enabled."),
				"limit_ip":        awgInt("Concurrent IP limit (0 = unlimited)."),
				"total_gb":        awgInt("Traffic limit in bytes (0 = unlimited)."),
				"expiry_time":     awgInt("Expiry timestamp in milliseconds since epoch (0 = never)."),
				"tg_id":           awgInt("Telegram user id associated with the client."),
				"sub_id":          awgString("Subscription id."),
				"comment":         awgString("Free-form comment."),
				"reset":           awgInt("Traffic reset interval in days (0 = never)."),
				"group":           awgString("Logical grouping label.", nonEmpty()...),
				"reset_day":       awgInt("Calendar day of month (1-31) the peer's traffic renews on. 0 keeps the rolling `reset` interval.", int64validator.Between(0, 31)),
				"reset_max":       awgInt("Maximum number of automatic renewals; 0 means unlimited.", int64validator.AtLeast(0)),
				"traffic_reset": awgString("Per-peer traffic reset cycle: `never`, `hourly`, `daily`, `weekly` or `monthly`. Independent of the inbound's own cycle.",
					stringvalidator.OneOf("never", "hourly", "daily", "weekly", "monthly")),
				"traffic_reset_day": awgInt("Day of month for a monthly per-peer reset. The panel clamps anything below 1 up to 1, so 0 cannot round-trip.", int64validator.Between(1, 31)),
				"created_at":        awgInt("Creation timestamp in milliseconds since epoch, set by the panel."),
				"updated_at":        awgInt("Last-update timestamp in milliseconds since epoch, set by the panel."),
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Typed model -> untyped map
// ---------------------------------------------------------------------------
//
// The AmneziaWG expanders emit camelCase keys directly, matching the wire
// format. settings.go forwards `server` and `clients` verbatim for this
// protocol instead of running them through its flat key table.

func awgPutString(out map[string]any, key string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() {
		out[key] = v.ValueString()
	}
}

func awgPutInt(out map[string]any, key string, v types.Int64) {
	if !v.IsNull() && !v.IsUnknown() {
		out[key] = v.ValueInt64()
	}
}

func awgPutBool(out map[string]any, key string, v types.Bool) {
	if !v.IsNull() && !v.IsUnknown() {
		out[key] = v.ValueBool()
	}
}

func expandAmneziawgInboundSettings(m *InboundAmneziawgSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if server := expandAmneziawgServerFromModel(m.Server); len(server) > 0 {
		out["server"] = server
	}
	if len(m.Clients) > 0 {
		out["clients"] = expandAmneziawgClientsFromModel(m.Clients)
	}
	return out
}

func expandAmneziawgServerFromModel(m *InboundAmneziawgServerModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}

	awgPutString(out, "privateKey", m.PrivateKey)
	awgPutString(out, "publicKey", m.PublicKey)
	awgPutString(out, "subnetIp", m.SubnetIP)
	awgPutInt(out, "subnetCidr", m.SubnetCIDR)
	awgPutInt(out, "mtu", m.MTU)
	awgPutString(out, "primaryDns", m.PrimaryDNS)
	awgPutString(out, "secondaryDns", m.SecondaryDNS)
	awgPutString(out, "externalInterface", m.ExternalInterface)
	awgPutBool(out, "ipv6Enabled", m.IPv6Enabled)
	awgPutString(out, "ipv6Subnet", m.IPv6Subnet)
	awgPutString(out, "ipv6ExternalInterface", m.IPv6ExternalInterface)
	awgPutBool(out, "routeThroughXray", m.RouteThroughXray)

	awgPutInt(out, "jc", m.Jc)
	awgPutInt(out, "jmin", m.Jmin)
	awgPutInt(out, "jmax", m.Jmax)
	awgPutInt(out, "s1", m.S1)
	awgPutInt(out, "s2", m.S2)
	awgPutInt(out, "s3", m.S3)
	awgPutInt(out, "s4", m.S4)

	awgPutString(out, "h1", m.H1)
	awgPutString(out, "h2", m.H2)
	awgPutString(out, "h3", m.H3)
	awgPutString(out, "h4", m.H4)
	awgPutString(out, "i1", m.I1)
	awgPutString(out, "i2", m.I2)
	awgPutString(out, "i3", m.I3)
	awgPutString(out, "i4", m.I4)
	awgPutString(out, "i5", m.I5)

	awgPutString(out, "headerProtectionKey", m.HeaderProtectionKey)
	awgPutString(out, "contentPaddingAddition", m.ContentPaddingAddition)
	awgPutString(out, "rekeyAfterTime", m.RekeyAfterTime)
	awgPutString(out, "rekeyTimeout", m.RekeyTimeout)
	awgPutString(out, "rejectAfterTime", m.RejectAfterTime)
	awgPutString(out, "keepaliveTimeout", m.KeepaliveTimeout)
	awgPutString(out, "maxHandshakeAttempts", m.MaxHandshakeAttempts)

	awgPutBool(out, "randomTrailers", m.RandomTrailers)
	awgPutBool(out, "disableCookies", m.DisableCookies)

	return out
}

func expandAmneziawgClientsFromModel(list []InboundAmneziawgClientModel) []any {
	out := make([]any, 0, len(list))
	for _, c := range list {
		entry := map[string]any{}
		awgPutString(entry, "email", c.Email)
		awgPutString(entry, "privateKey", c.PrivateKey)
		awgPutString(entry, "publicKey", c.PublicKey)
		awgPutString(entry, "preSharedKey", c.PreSharedKey)
		if !c.AllowedIPs.IsNull() && !c.AllowedIPs.IsUnknown() {
			entry["allowedIPs"] = typesListToAnySlice(c.AllowedIPs)
		}
		awgPutInt(entry, "keepAlive", c.KeepAlive)
		awgPutString(entry, "forwardedPorts", c.ForwardedPorts)
		awgPutBool(entry, "enable", c.Enable)
		awgPutInt(entry, "limitIp", c.LimitIP)
		awgPutInt(entry, "totalGB", c.TotalGB)
		awgPutInt(entry, "expiryTime", c.ExpiryTime)
		awgPutInt(entry, "tgId", c.TgID)
		awgPutString(entry, "subId", c.SubID)
		awgPutString(entry, "comment", c.Comment)
		awgPutInt(entry, "reset", c.Reset)
		awgPutString(entry, "group", c.Group)
		awgPutInt(entry, "resetDay", c.ResetDay)
		awgPutInt(entry, "resetMax", c.ResetMax)
		awgPutString(entry, "trafficReset", c.TrafficReset)
		awgPutInt(entry, "trafficResetDay", c.TrafficResetDay)
		awgPutInt(entry, "created_at", c.CreatedAt)
		awgPutInt(entry, "updated_at", c.UpdatedAt)
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Untyped map -> typed model
// ---------------------------------------------------------------------------

func awgReadString(raw map[string]any, key string) types.String {
	if v, ok := raw[key].(string); ok {
		return types.StringValue(v)
	}
	return types.StringNull()
}

func awgReadInt(raw map[string]any, key string) types.Int64 {
	if v, ok := raw[key]; ok {
		return types.Int64Value(int64Value(v))
	}
	return types.Int64Null()
}

// awgReadBool reads a bool the panel declares `omitempty`: a stripped key means
// false, not "unset". Returning null there would make a configured `false` fail
// the apply with an inconsistent-result error.
func awgReadBool(raw map[string]any, key string) types.Bool {
	if v, ok := raw[key]; ok {
		return types.BoolValue(boolValue(v))
	}
	return types.BoolValue(false)
}

func flattenAmneziawgInboundSettings(data map[string]any) *InboundAmneziawgSettingsModel {
	if len(data) == 0 {
		return nil
	}
	out := &InboundAmneziawgSettingsModel{}
	if server, ok := data["server"].(map[string]any); ok {
		out.Server = flattenAmneziawgServerToModel(server)
	}
	if clients, ok := data["clients"].([]any); ok {
		out.Clients = flattenAmneziawgClientsToModel(clients)
	}
	if out.Server == nil && len(out.Clients) == 0 {
		return nil
	}
	return out
}

func flattenAmneziawgServerToModel(raw map[string]any) *InboundAmneziawgServerModel {
	if len(raw) == 0 {
		return nil
	}
	return &InboundAmneziawgServerModel{
		PrivateKey: awgReadString(raw, "privateKey"),
		PublicKey:  awgReadString(raw, "publicKey"),

		SubnetIP:   awgReadString(raw, "subnetIp"),
		SubnetCIDR: awgReadInt(raw, "subnetCidr"),
		MTU:        awgReadInt(raw, "mtu"),

		PrimaryDNS:   awgReadString(raw, "primaryDns"),
		SecondaryDNS: awgReadString(raw, "secondaryDns"),

		ExternalInterface:     awgReadString(raw, "externalInterface"),
		IPv6Enabled:           awgReadBool(raw, "ipv6Enabled"),
		IPv6Subnet:            awgReadString(raw, "ipv6Subnet"),
		IPv6ExternalInterface: awgReadString(raw, "ipv6ExternalInterface"),

		RouteThroughXray: awgReadBool(raw, "routeThroughXray"),

		Jc:   awgReadInt(raw, "jc"),
		Jmin: awgReadInt(raw, "jmin"),
		Jmax: awgReadInt(raw, "jmax"),
		S1:   awgReadInt(raw, "s1"),
		S2:   awgReadInt(raw, "s2"),
		S3:   awgReadInt(raw, "s3"),
		S4:   awgReadInt(raw, "s4"),

		H1: awgReadString(raw, "h1"),
		H2: awgReadString(raw, "h2"),
		H3: awgReadString(raw, "h3"),
		H4: awgReadString(raw, "h4"),
		I1: awgReadString(raw, "i1"),
		I2: awgReadString(raw, "i2"),
		I3: awgReadString(raw, "i3"),
		I4: awgReadString(raw, "i4"),
		I5: awgReadString(raw, "i5"),

		HeaderProtectionKey:    awgReadString(raw, "headerProtectionKey"),
		ContentPaddingAddition: awgReadString(raw, "contentPaddingAddition"),

		RekeyAfterTime:       awgReadString(raw, "rekeyAfterTime"),
		RekeyTimeout:         awgReadString(raw, "rekeyTimeout"),
		RejectAfterTime:      awgReadString(raw, "rejectAfterTime"),
		KeepaliveTimeout:     awgReadString(raw, "keepaliveTimeout"),
		MaxHandshakeAttempts: awgReadString(raw, "maxHandshakeAttempts"),

		RandomTrailers: awgReadBool(raw, "randomTrailers"),
		DisableCookies: awgReadBool(raw, "disableCookies"),
	}
}

func flattenAmneziawgClientsToModel(list []any) []InboundAmneziawgClientModel {
	out := make([]InboundAmneziawgClientModel, 0, len(list))
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		c := InboundAmneziawgClientModel{
			Email:           awgReadString(raw, "email"),
			PrivateKey:      awgReadString(raw, "privateKey"),
			PublicKey:       awgReadString(raw, "publicKey"),
			PreSharedKey:    awgReadString(raw, "preSharedKey"),
			KeepAlive:       awgReadInt(raw, "keepAlive"),
			ForwardedPorts:  awgReadString(raw, "forwardedPorts"),
			Enable:          awgReadBool(raw, "enable"),
			LimitIP:         awgReadInt(raw, "limitIp"),
			TotalGB:         awgReadInt(raw, "totalGB"),
			ExpiryTime:      awgReadInt(raw, "expiryTime"),
			TgID:            awgReadInt(raw, "tgId"),
			SubID:           awgReadString(raw, "subId"),
			Comment:         awgReadString(raw, "comment"),
			Reset:           awgReadInt(raw, "reset"),
			Group:           awgReadString(raw, "group"),
			ResetDay:        awgReadInt(raw, "resetDay"),
			ResetMax:        awgReadInt(raw, "resetMax"),
			TrafficReset:    awgReadString(raw, "trafficReset"),
			TrafficResetDay: awgReadInt(raw, "trafficResetDay"),
			CreatedAt:       awgReadInt(raw, "created_at"),
			UpdatedAt:       awgReadInt(raw, "updated_at"),
		}
		if v, ok := raw["allowedIPs"]; ok {
			c.AllowedIPs = anySliceToTypesList(v)
		} else {
			c.AllowedIPs = types.ListNull(types.StringType)
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ---------------------------------------------------------------------------
// Config validation
// ---------------------------------------------------------------------------

// amneziawgServerRequiredValidator rejects an AmneziaWG inbound that does not
// declare `amneziawg_settings.server`.
//
// The panel regenerates the entire server block — including a fresh keypair —
// whenever the settings blob it receives is empty or carries no `server` object
// (normalizeAmneziaWGSettings, 3x-ui-3.7.0/internal/web/service/inbound_amneziawg.go:171-200,
// which runs on UpdateInbound as well as AddInbound). Without the block there is
// nothing in state for the provider to send back, so an unrelated edit — a
// changed remark — would rotate the server keypair and silently invalidate every
// peer config already handed out. Verified against a live v3.7.0 panel: an update
// posting `settings = {}` returns a different `publicKey` than the create did.
//
// Declaring the block, even empty, is enough: the panel fills the attributes in
// on create, Read records them, and UseStateForUnknown replays them on every
// later apply, so the blob is never empty again.
type amneziawgServerRequiredValidator struct{}

func (v amneziawgServerRequiredValidator) Description(_ context.Context) string {
	return "amneziawg inbounds must declare an amneziawg_settings.server block"
}

func (v amneziawgServerRequiredValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v amneziawgServerRequiredValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config InboundResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// An unknown protocol (interpolated from another resource) cannot be checked
	// here; the panel still enforces its own rules at apply time.
	if config.Protocol.IsNull() || config.Protocol.IsUnknown() {
		return
	}
	if config.Protocol.ValueString() != "amneziawg" {
		return
	}
	if config.AmneziawgSettings != nil && config.AmneziawgSettings.Server != nil {
		validateAmneziawgServerConstraints(config.AmneziawgSettings.Server, resp)
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("amneziawg_settings"),
		"Missing amneziawg_settings.server block",
		"An amneziawg inbound must declare an `amneziawg_settings` block containing a `server` block, even if every "+
			"attribute inside it is left to the panel:\n\n"+
			"  amneziawg_settings {\n    server {}\n  }\n\n"+
			"3x-ui regenerates the whole server block — including a new keypair — whenever it receives an inbound "+
			"whose settings carry no `server` object, on update as well as on create. Without the block in the "+
			"configuration there is nothing for the provider to send back, so an unrelated change such as a new "+
			"remark would rotate the server keys and invalidate every peer configuration already distributed.",
	)
}

// ---------------------------------------------------------------------------
// Two-phase create
// ---------------------------------------------------------------------------

// splitAmneziawgServer removes the `server` object from a settings blob and
// returns it separately, leaving the rest (notably `clients`) intact.
//
// The panel only generates its randomised obfuscation set — jc/jmin/jmax,
// s1-s4, h1-h4, i1, headerProtectionKey, contentPaddingAddition and the timing
// ranges — when the settings it receives carry NO `server` object at all
// (normalizeAmneziaWGSettings, 3x-ui-3.7.0/internal/web/service/inbound_amneziawg.go:171-200).
// A partial block is taken literally: every field the configuration omits is
// stored as its zero value, which means jc=0, blank h1-h4 and no header
// protection. That inbound is plain WireGuard wearing AmneziaWG's name, and
// nothing reports it — the save succeeds and the state is consistent. Measured
// on a live v3.7.0 panel: `{"server":{"subnetIp":"10.9.9.0","subnetCidr":24}}`
// comes back with jc=0, h1="" and an empty headerProtectionKey, while `{}`
// comes back fully populated.
//
// So Create posts the inbound without the block, lets the panel generate a
// complete server, and applies the configured fields on top (see
// applyAmneziawgServerOverrides). Update needs none of this: by then state
// holds the full generated set and UseStateForUnknown replays it.
func splitAmneziawgServer(settingsJSON string) (rest string, server map[string]any, err error) {
	trimmed := strings.TrimSpace(settingsJSON)
	if trimmed == "" || trimmed == "null" {
		return settingsJSON, nil, nil
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return settingsJSON, nil, fmt.Errorf("amneziawg: parsing settings: %w", err)
	}

	raw, ok := parsed["server"].(map[string]any)
	if !ok || len(raw) == 0 {
		return settingsJSON, nil, nil
	}
	delete(parsed, "server")

	encoded, err := json.Marshal(parsed)
	if err != nil {
		return settingsJSON, nil, fmt.Errorf("amneziawg: re-encoding settings: %w", err)
	}
	return string(encoded), raw, nil
}

// applyAmneziawgServerOverrides merges the configured server fields over the
// block the panel generated, returning the settings blob to save. Returns
// ok=false when there is nothing to change, so the caller can skip the write.
func applyAmneziawgServerOverrides(generatedSettings string, overrides map[string]any) (merged string, ok bool, err error) {
	if len(overrides) == 0 {
		return generatedSettings, false, nil
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(generatedSettings)), &parsed); err != nil {
		return generatedSettings, false, fmt.Errorf("amneziawg: parsing generated settings: %w", err)
	}

	server, _ := parsed["server"].(map[string]any)
	if server == nil {
		server = map[string]any{}
	}

	changed := false
	for key, value := range overrides {
		if existing, present := server[key]; present && sameJSONValue(existing, value) {
			continue
		}
		server[key] = value
		changed = true
	}
	if !changed {
		return generatedSettings, false, nil
	}
	parsed["server"] = server

	encoded, err := json.Marshal(parsed)
	if err != nil {
		return generatedSettings, false, fmt.Errorf("amneziawg: re-encoding settings: %w", err)
	}
	return string(encoded), true, nil
}

// sameJSONValue compares two values that may have crossed a JSON round trip,
// where every number arrives as float64.
func sameJSONValue(a, b any) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// validateAmneziawgServerConstraints checks the cross-field rules the panel
// enforces at save time (3x-ui-3.7.0/internal/amneziawg/params.go:136-239).
// Without these the configuration applies, fails mid-flight against the panel,
// and leaves the practitioner with an opaque "amneziawg: ..." message; here they
// surface at plan time against the offending attribute.
//
// Only the rules that are decidable from a single resource's configuration are
// checked. The forwarded-port limits are not: they depend on the ports of every
// other inbound on the panel, which the provider cannot see while planning.
func validateAmneziawgServerConstraints(server *InboundAmneziawgServerModel, resp *resource.ValidateConfigResponse) {
	serverPath := path.Root("amneziawg_settings").AtName("server")

	known := func(v types.Int64) (int64, bool) {
		if v.IsNull() || v.IsUnknown() {
			return 0, false
		}
		return v.ValueInt64(), true
	}

	jmin, jminOK := known(server.Jmin)
	jmax, jmaxOK := known(server.Jmax)
	if jminOK && jmaxOK && jmin > jmax {
		resp.Diagnostics.AddAttributeError(
			serverPath.AtName("jmin"),
			"jmin is greater than jmax",
			fmt.Sprintf("jmin (%d) must not exceed jmax (%d); the panel rejects the inbound otherwise.", jmin, jmax),
		)
	}

	s1, s1OK := known(server.S1)
	s2, s2OK := known(server.S2)
	if s1OK && s2OK && s1+56 == s2 {
		resp.Diagnostics.AddAttributeError(
			serverPath.AtName("s2"),
			"s1 + 56 must not equal s2",
			fmt.Sprintf("With s1 = %d, s2 = %d makes the junk-padded init packet indistinguishable in size from the response packet, "+
				"which defeats the obfuscation. The panel rejects this combination; pick a different s2.", s1, s2),
		)
	}

	// Header protection needs at least 12 bytes of junk in every packet class,
	// or the protected header is recoverable.
	if !server.HeaderProtectionKey.IsNull() && !server.HeaderProtectionKey.IsUnknown() && server.HeaderProtectionKey.ValueString() != "" {
		for name, attr := range map[string]types.Int64{
			"s1": server.S1, "s2": server.S2, "s3": server.S3, "s4": server.S4,
		} {
			if v, ok := known(attr); ok && v < 12 {
				resp.Diagnostics.AddAttributeError(
					serverPath.AtName(name),
					"header protection requires s1-s4 of at least 12",
					fmt.Sprintf("header_protection_key is set, so every junk size must be at least 12; %s is %d.", name, v),
				)
			}
		}
	}

	if !server.IPv6Enabled.IsNull() && !server.IPv6Enabled.IsUnknown() && server.IPv6Enabled.ValueBool() {
		if server.IPv6Subnet.IsNull() && !server.IPv6Subnet.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				serverPath.AtName("ipv6_subnet"),
				"ipv6_subnet is required when ipv6_enabled is true",
				"Set ipv6_subnet to the tunnel prefix (for example `fd00:8:1::/64`); the panel refuses to save an IPv6-enabled AmneziaWG inbound without one.",
			)
		}
	}
}

// applyAmneziawgServerPhaseTwo completes the two-phase create: it lays the
// configured server fields over the block the panel generated and, when that
// changes anything, saves and re-reads the inbound. Returns the inbound to
// record in state, which is the argument itself when there is nothing to apply.
func applyAmneziawgServerPhaseTwo(ctx context.Context, client *Client, created *Inbound, overrides map[string]any) (*Inbound, error) {
	if len(overrides) == 0 || created == nil {
		return created, nil
	}

	merged, changed, err := applyAmneziawgServerOverrides(created.Settings, overrides)
	if err != nil {
		return nil, err
	}
	if !changed {
		return created, nil
	}

	created.Settings = merged
	updated, err := client.UpdateInbound(ctx, created)
	if err != nil {
		return nil, err
	}
	if updated != nil && updated.ID != 0 {
		created = updated
	}

	// Re-read for the same reason Create does after AddInbound: the write
	// endpoint can return incomplete data while SQLite catches up.
	settled := created
	if err := client.WithReadAfterWriteRetry(ctx, fmt.Sprintf("read amneziawg inbound %d", created.ID), func() (bool, error) {
		got, getErr := client.GetInbound(ctx, created.ID)
		if getErr != nil {
			return false, getErr
		}
		settled = got
		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("reading the inbound back after applying server settings: %w", err)
	}
	return settled, nil
}

// ---------------------------------------------------------------------------
// Peer cleanup on delete
// ---------------------------------------------------------------------------

// releaseInboundOwnedPeers deletes an inbound's peers through the per-client
// endpoint before the inbound itself is removed (#452).
//
// 3x-ui's DelInbound removes the inbound and, via ClientService.DetachInbound
// (3x-ui-3.7.0/internal/web/service/client_link.go:291-296), the
// client_inbounds link rows — but not the `clients` rows, which carry the
// unique index on `email`. Peers owned by threexui_inbound rather than by
// threexui_inbound_client (WireGuard clients[] since v3.4.2, AmneziaWG since
// v3.7.0) are therefore left behind holding their address, and a later
// `terraform apply` that recreates the same inbound fails with
// "Duplicate email: <address>" naming a client the practitioner can no longer
// see. Deleting each peer frees the row; verified against a live v3.7.0 panel,
// where the same create/delete/create sequence fails without this and succeeds
// with it.
//
// Called AFTER the inbound is deleted, not before. ClientService.DeleteByEmail
// looks the row up by email directly (client_lookup.go:20) and its
// client_inbounds loop no-ops on an empty link list (client_crud.go:678-707),
// so the endpoint works just as well once the inbound is gone — confirmed
// live. Running it first would buy nothing and open a window: if the inbound
// delete then failed, the peers would already be gone while the inbound stayed
// up serving nobody and still in state. This order degrades to the old
// behaviour instead — the inbound is gone, the rows leak, and the warning says
// so.
//
// Only protocols whose peers the inbound owns yield addresses (see
// inboundOwnedPeerEmails). For everything else clients[] belongs to
// threexui_inbound_client, which deletes its own rows through the same
// endpoint.
//
// The addresses come from state rather than from a fresh GET. For WireGuard and
// AmneziaWG the two agree after any refresh — flattenSettings reads the panel's
// clients[] back into the inbound's own block for exactly these protocols — so
// a GET would mostly cost a round-trip. A peer added in the panel since the
// last refresh is invisible either way, and leaks, which is the same outcome as
// before this function existed.
//
// Failures are reported as warnings rather than errors: the inbound delete is
// what the practitioner asked for, and refusing to proceed because a peer row
// could not be freed would strand the resource in state. The warning names the
// email so the leftover can be cleaned up by hand.
func releaseInboundOwnedPeers(ctx context.Context, client *Client, inboundID int, protocol string, state *InboundResourceModel, diags *diag.Diagnostics) {
	if state == nil {
		return
	}

	// inboundOwnedPeerEmails is the single gate: it yields addresses only for
	// the protocols whose peers the inbound owns.
	releasePeerEmails(ctx, client, inboundID, inboundOwnedPeerEmails(protocol, state), diags)
}

// releasePeerEmailsRemovedByUpdate frees the addresses of peers that were in
// state but are no longer in the plan (#452).
//
// Dropping a peer from an inbound is an update, not a delete, and it strands
// the email exactly as removing the whole inbound does: the panel rewrites
// settings.clients[] without the peer but keeps its `clients` row, unique index
// and all. Measured on v3.7.0 — remove the only peer through UpdateInbound, then
// create any inbound reusing that email, and the create fails with "Duplicate
// email". Nothing in the panel's own UI surfaces the leftover.
func releasePeerEmailsRemovedByUpdate(ctx context.Context, client *Client, inboundID int, protocol string, state, plan *InboundResourceModel, diags *diag.Diagnostics) {
	if state == nil || plan == nil {
		return
	}

	kept := make(map[string]bool)
	for _, email := range inboundOwnedPeerEmails(protocol, plan) {
		kept[email] = true
	}

	var removed []string
	for _, email := range inboundOwnedPeerEmails(protocol, state) {
		if !kept[email] {
			removed = append(removed, email)
		}
	}
	releasePeerEmails(ctx, client, inboundID, removed, diags)
}

// releasePeerEmails deletes each address through the per-client endpoint,
// warning rather than failing on any that cannot be removed.
func releasePeerEmails(ctx context.Context, client *Client, inboundID int, emails []string, diags *diag.Diagnostics) {
	if len(emails) == 0 {
		return
	}

	// Serialise against threexui_inbound_client's read-modify-write cycles, the
	// same lock the create and update paths take when settings carry clients[].
	inboundClientMu.Lock()
	defer inboundClientMu.Unlock()

	for _, email := range emails {
		// The v3.1.0+ endpoint keys on email; the legacy fallback wants a client
		// id, which these peers do not have, so email stands in for both.
		if err := client.DeleteInboundClient(ctx, inboundID, email, email); err != nil {
			diags.AddWarning(
				"Could not release peer email before deleting the inbound",
				fmt.Sprintf("The peer %q could not be deleted (%s). The inbound will still be removed, but 3x-ui keeps the "+
					"client row, so recreating this inbound with the same peer email will fail with a duplicate-email error. "+
					"Delete the leftover client in the panel, or use a different email.", email, err.Error()),
			)
		}
	}
}

// inboundOwnedPeerEmails collects the peer emails recorded in state for the
// protocols whose clients the inbound owns.
func inboundOwnedPeerEmails(protocol string, state *InboundResourceModel) []string {
	var emails []string
	add := func(v types.String) {
		if v.IsNull() || v.IsUnknown() {
			return
		}
		if email := v.ValueString(); email != "" {
			emails = append(emails, email)
		}
	}

	switch protocol {
	case "amneziawg":
		if state.AmneziawgSettings == nil {
			return nil
		}
		for _, c := range state.AmneziawgSettings.Clients {
			add(c.Email)
		}
	case "wireguard":
		if state.WireguardSettings == nil {
			return nil
		}
		for _, c := range state.WireguardSettings.Clients {
			add(c.Email)
		}
	case "tuic":
		if state.TuicSettings == nil {
			return nil
		}
		for _, c := range state.TuicSettings.Clients {
			add(c.Email)
		}
	}
	return emails
}
