package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// TUIC v5 (3x-ui v3.8.0+)
//
// TUIC is terminated by a bundled `tuic-server` sidecar (not xray-core) which
// relays into Xray, so per-client stats, routing and quotas work like any
// other protocol (upstream #6337). The settings blob is
// {"server": {...}, "clients": [...]} — nested exactly like AmneziaWG, but
// with snake_case wire keys (tuic.TuicServerSettings json tags,
// 3x-ui-3.8.5/internal/tuic/types.go:14-26). Unlike AmneziaWG the panel does
// NOT generate the server block: defaults (bbr, native, info, 15, 3, 1500,
// h3+spdy/3.1) are applied at read time by InstanceFromInbound, and the
// certificate/private_key inline PEM pair is operator-supplied.
//
// Like WireGuard/AmneziaWG, the clients array is managed by threexui_inbound
// itself, not by threexui_inbound_client — and the panel validates each peer
// on inbound add AND update: non-empty id (uuid), password and email
// (inbound.go:1209-1219, :1714-1721).
// ---------------------------------------------------------------------------

type InboundTuicSettingsModel struct {
	Server  *InboundTuicServerModel  `tfsdk:"server"`
	Clients []InboundTuicClientModel `tfsdk:"clients"`
}

// InboundTuicServerModel mirrors tuic.TuicServerSettings. certificate and
// private_key are inline PEM strings written straight into the sidecar config.
type InboundTuicServerModel struct {
	Certificate           types.String `tfsdk:"certificate"`
	PrivateKey            types.String `tfsdk:"private_key"`
	CongestionControl     types.String `tfsdk:"congestion_control"`
	ALPN                  types.List   `tfsdk:"alpn"` // list of string
	UDPRelayMode          types.String `tfsdk:"udp_relay_mode"`
	ZeroRTTHandshake      types.Bool   `tfsdk:"zero_rtt_handshake"`
	LogLevel              types.String `tfsdk:"log_level"`
	MaxIdleTime           types.Int64  `tfsdk:"max_idle_time"`
	AuthenticationTimeout types.Int64  `tfsdk:"authentication_timeout"`
	MaxUDPRelayPacketSize types.Int64  `tfsdk:"max_udp_relay_packet_size"`
	SNI                   types.String `tfsdk:"sni"`
}

// InboundTuicClientModel is one TUIC user. The id (uuid) and password are
// REQUIRED: the panel rejects a keyless or passwordless peer on inbound save,
// and derives them only on the /panel/api/clients endpoints, which do not own
// these peers (same rule as AmneziaWG's public_key). The bookkeeping fields
// are modelled because the inbound rewrites clients[] wholesale on every
// apply — a field the provider does not read is silently zeroed.
type InboundTuicClientModel struct {
	Email    types.String `tfsdk:"email"`
	ID       types.String `tfsdk:"id"`
	Password types.String `tfsdk:"password"`

	Enable     types.Bool   `tfsdk:"enable"`
	LimitIP    types.Int64  `tfsdk:"limit_ip"`
	TotalGB    types.Int64  `tfsdk:"total_gb"`
	ExpiryTime types.Int64  `tfsdk:"expiry_time"`
	TgID       types.Int64  `tfsdk:"tg_id"`
	SubID      types.String `tfsdk:"sub_id"`
	Comment    types.String `tfsdk:"comment"`
	Reset      types.Int64  `tfsdk:"reset"`

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

// tuicString builds the Optional+Computed shape every TUIC field uses: the
// panel applies its defaults at read time, so an omitted field must survive a
// plan without being re-planned as unknown.
func tuicString(description string, validators ...validator.String) schema.StringAttribute {
	return schema.StringAttribute{
		Optional: true, Computed: true,
		Description: description,
		Validators:  validators,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

func tuicInt(description string, validators ...validator.Int64) schema.Int64Attribute {
	return schema.Int64Attribute{
		Optional: true, Computed: true,
		Description: description,
		Validators:  validators,
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
}

func tuicBool(description string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional: true, Computed: true,
		Description: description,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	}
}

func tuicSettingsBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Settings for the TUIC v5 protocol (3x-ui v3.8.0+). TUIC is terminated by a bundled tuic-server sidecar that relays into Xray. " +
			"Users are managed here through `clients`, not through separate `threexui_inbound_client` resources.",
		Blocks: map[string]schema.Block{
			"server": schema.SingleNestedBlock{
				Description: "TUIC server parameters. The certificate/private_key inline PEM pair is operator-supplied; every other field falls back to the panel's read-time defaults (bbr, native, info, 15, 3, 1500, [\"h3\",\"spdy/3.1\"]).",
				Attributes: map[string]schema.Attribute{
					"certificate": tuicString("Server certificate (inline PEM). Required for the tuic-server sidecar to actually serve; the panel accepts a save without it but the instance will not come up."),
					"private_key": func() schema.StringAttribute {
						a := tuicString("Server private key (inline PEM).", nonEmpty()...)
						a.Sensitive = true
						return a
					}(),
					"congestion_control": tuicString("Congestion control algorithm.",
						stringvalidator.OneOf("bbr", "cubic", "new_reno")),
					"alpn": schema.ListAttribute{
						Optional: true, Computed: true,
						ElementType: types.StringType,
						Description: "ALPN protocols (panel default [\"h3\", \"spdy/3.1\"]).",
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
					},
					"udp_relay_mode": tuicString("UDP relay mode.",
						stringvalidator.OneOf("native", "quic")),
					"zero_rtt_handshake": tuicBool("Enable 0-RTT handshake."),
					"log_level": tuicString("Sidecar log level.",
						stringvalidator.OneOf("info", "warn", "error", "debug")),
					"max_idle_time":             tuicInt("Max idle time in seconds (panel default 15).", int64validator.AtLeast(1)),
					"authentication_timeout":    tuicInt("Authentication timeout in seconds (panel default 3).", int64validator.AtLeast(1)),
					"max_udp_relay_packet_size": tuicInt("Max UDP relay packet size in bytes (panel default 1500).", int64validator.AtLeast(1)),
					"sni":                       tuicString("Server name indication override."),
				},
			},
			"clients": tuicClientsBlock(),
		},
	}
}

func tuicClientsBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "TUIC users. id (uuid) and password are required: the panel refuses a keyless or passwordless peer on inbound save.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"email": schema.StringAttribute{
					Required:    true,
					Description: "Client email; keys the traffic counters (unique per panel).",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"id": schema.StringAttribute{
					Required:    true,
					Description: "Client UUID.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"password": func() schema.StringAttribute {
					a := schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "Client password.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					}
					return a
				}(),
				"enable":            tuicBool("Enable the client."),
				"limit_ip":          tuicInt("IP limit (0 = unlimited)."),
				"total_gb":          tuicInt("Traffic quota in bytes (0 = unlimited)."),
				"expiry_time":       tuicInt("Expiry as a unix timestamp (0 = never)."),
				"tg_id":             tuicInt("Telegram ID."),
				"sub_id":            tuicString("Subscription ID."),
				"comment":           tuicString("Free-form comment."),
				"reset":             tuicInt("Traffic reset cycle in days (0 = none)."),
				"reset_day":         tuicInt("Calendar renewal day (0 keeps the rolling interval)."),
				"reset_max":         tuicInt("Cap on auto-renewals (0 = unlimited)."),
				"traffic_reset":     tuicString("Per-client traffic reset cycle."),
				"traffic_reset_day": tuicInt("Per-client traffic reset day (1-31)."),
				"created_at":        tuicInt("Row creation timestamp (panel-managed)."),
				"updated_at":        tuicInt("Row update timestamp (panel-managed)."),
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Typed model -> untyped map (snake_case wire keys, forwarded verbatim)
// ---------------------------------------------------------------------------

func expandTuicInboundSettings(m *InboundTuicSettingsModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	if server := expandTuicServerFromModel(m.Server); len(server) > 0 {
		out["server"] = server
	}
	if len(m.Clients) > 0 {
		out["clients"] = expandTuicClientsFromModel(m.Clients)
	}
	return out
}

func expandTuicServerFromModel(m *InboundTuicServerModel) map[string]any {
	if m == nil {
		return nil
	}
	out := map[string]any{}
	tuicPutString(out, "certificate", m.Certificate)
	tuicPutString(out, "private_key", m.PrivateKey)
	tuicPutString(out, "congestion_control", m.CongestionControl)
	if !m.ALPN.IsNull() && !m.ALPN.IsUnknown() && len(m.ALPN.Elements()) > 0 {
		out["alpn"] = typesListToAnySlice(m.ALPN)
	}
	tuicPutString(out, "udp_relay_mode", m.UDPRelayMode)
	if !m.ZeroRTTHandshake.IsNull() && !m.ZeroRTTHandshake.IsUnknown() {
		out["zero_rtt_handshake"] = m.ZeroRTTHandshake.ValueBool()
	}
	tuicPutString(out, "log_level", m.LogLevel)
	tuicPutInt(out, "max_idle_time", m.MaxIdleTime)
	tuicPutInt(out, "authentication_timeout", m.AuthenticationTimeout)
	tuicPutInt(out, "max_udp_relay_packet_size", m.MaxUDPRelayPacketSize)
	tuicPutString(out, "sni", m.SNI)
	return out
}

func expandTuicClientsFromModel(list []InboundTuicClientModel) []any {
	out := make([]any, 0, len(list))
	for _, c := range list {
		entry := map[string]any{}
		tuicPutString(entry, "email", c.Email)
		// The wire accepts both `uuid` and `id`; the panel's own client
		// serialisation (model.Client) writes `id`, and the save-path
		// validation reads client.ID — write `id`.
		tuicPutString(entry, "id", c.ID)
		if !c.Password.IsNull() && !c.Password.IsUnknown() {
			entry["password"] = c.Password.ValueString()
		}
		if !c.Enable.IsNull() && !c.Enable.IsUnknown() {
			entry["enable"] = c.Enable.ValueBool()
		}
		tuicPutInt(entry, "limitIp", c.LimitIP)
		tuicPutInt(entry, "totalGB", c.TotalGB)
		tuicPutInt(entry, "expiryTime", c.ExpiryTime)
		tuicPutInt(entry, "tgId", c.TgID)
		tuicPutString(entry, "subId", c.SubID)
		tuicPutString(entry, "comment", c.Comment)
		tuicPutInt(entry, "reset", c.Reset)
		tuicPutInt(entry, "resetDay", c.ResetDay)
		tuicPutInt(entry, "resetMax", c.ResetMax)
		tuicPutString(entry, "trafficReset", c.TrafficReset)
		tuicPutInt(entry, "trafficResetDay", c.TrafficResetDay)
		tuicPutInt(entry, "created_at", c.CreatedAt)
		tuicPutInt(entry, "updated_at", c.UpdatedAt)
		out = append(out, entry)
	}
	return out
}

func tuicPutString(out map[string]any, key string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		out[key] = v.ValueString()
	}
}

func tuicPutInt(out map[string]any, key string, v types.Int64) {
	if !v.IsNull() && !v.IsUnknown() && v.ValueInt64() != 0 {
		out[key] = int(v.ValueInt64())
	}
}

// ---------------------------------------------------------------------------
// Untyped map -> typed model
// ---------------------------------------------------------------------------

func flattenTuicInboundSettings(data map[string]any) *InboundTuicSettingsModel {
	if len(data) == 0 {
		return nil
	}
	out := &InboundTuicSettingsModel{}
	if server, ok := data["server"].(map[string]any); ok {
		out.Server = flattenTuicServerToModel(server)
	}
	if clients, ok := data["clients"].([]any); ok {
		out.Clients = flattenTuicClientsToModel(clients)
	}
	if out.Server == nil && len(out.Clients) == 0 {
		return nil
	}
	return out
}

func flattenTuicServerToModel(raw map[string]any) *InboundTuicServerModel {
	if len(raw) == 0 {
		return nil
	}
	// A zero-value types.List carries no element type and cannot be set into
	// state — absent list attributes must be explicit nulls.
	m := &InboundTuicServerModel{ALPN: types.ListNull(types.StringType)}
	if v, ok := raw["certificate"].(string); ok && v != "" {
		m.Certificate = types.StringValue(v)
	}
	if v, ok := raw["private_key"].(string); ok && v != "" {
		m.PrivateKey = types.StringValue(v)
	}
	if v, ok := raw["congestion_control"].(string); ok && v != "" {
		m.CongestionControl = types.StringValue(v)
	}
	if v, ok := raw["alpn"].([]any); ok {
		m.ALPN = anySliceToTypesList(v)
	}
	if v, ok := raw["udp_relay_mode"].(string); ok && v != "" {
		m.UDPRelayMode = types.StringValue(v)
	}
	if v, ok := raw["zero_rtt_handshake"].(bool); ok {
		m.ZeroRTTHandshake = types.BoolValue(v)
	}
	if v, ok := raw["log_level"].(string); ok && v != "" {
		m.LogLevel = types.StringValue(v)
	}
	if v, ok := raw["max_idle_time"]; ok {
		m.MaxIdleTime = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := raw["authentication_timeout"]; ok {
		m.AuthenticationTimeout = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := raw["max_udp_relay_packet_size"]; ok {
		m.MaxUDPRelayPacketSize = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := raw["sni"].(string); ok && v != "" {
		m.SNI = types.StringValue(v)
	}
	return m
}

func flattenTuicClientsToModel(list []any) []InboundTuicClientModel {
	out := make([]InboundTuicClientModel, 0, len(list))
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		// The panel serialises a TUIC client inconsistently between the
		// create read-back (empty optional keys present) and the update
		// read-back (omitempty-stripped), so every optional field flattens
		// to a concrete zero value — never null — or the apply fails with
		// an inconsistent-result error on the first update.
		c := InboundTuicClientModel{
			Email:    tuicReadString(raw, "email"),
			ID:       tuicReadClientID(raw),
			Password: tuicReadString(raw, "password"),
			Enable:   tuicReadBool(raw, "enable"),

			LimitIP:    tuicReadInt(raw, "limitIp"),
			TotalGB:    tuicReadInt(raw, "totalGB"),
			ExpiryTime: tuicReadInt(raw, "expiryTime"),
			TgID:       tuicReadInt(raw, "tgId"),
			SubID:      tuicReadString(raw, "subId"),
			Comment:    tuicReadString(raw, "comment"),
			Reset:      tuicReadInt(raw, "reset"),

			ResetDay:        tuicReadInt(raw, "resetDay"),
			ResetMax:        tuicReadInt(raw, "resetMax"),
			TrafficReset:    tuicReadString(raw, "trafficReset"),
			TrafficResetDay: tuicReadInt(raw, "trafficResetDay"),
			CreatedAt:       tuicReadInt(raw, "created_at"),
			UpdatedAt:       tuicReadInt(raw, "updated_at"),
		}
		out = append(out, c)
	}
	return out
}

// tuicReadString returns "" (not null) for an absent or empty key.
func tuicReadString(raw map[string]any, key string) types.String {
	if v, ok := raw[key].(string); ok {
		return types.StringValue(v)
	}
	return types.StringValue("")
}

// tuicReadInt returns 0 (not null) for an absent key.
func tuicReadInt(raw map[string]any, key string) types.Int64 {
	if v, ok := raw[key]; ok {
		return types.Int64Value(int64(intValue(v)))
	}
	return types.Int64Value(0)
}

func tuicReadBool(raw map[string]any, key string) types.Bool {
	if v, ok := raw[key].(bool); ok {
		return types.BoolValue(v)
	}
	return types.BoolValue(false)
}

// tuicReadClientID accepts both wire spellings (uuid preferred, id fallback).
func tuicReadClientID(raw map[string]any) types.String {
	if v, ok := raw["uuid"].(string); ok && v != "" {
		return types.StringValue(v)
	}
	return tuicReadString(raw, "id")
}
