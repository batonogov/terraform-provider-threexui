package provider

import (
	"encoding/json"

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
// Typed models
// ---------------------------------------------------------------------------

type XrayDNSModel struct {
	Script                 types.String    `tfsdk:"script"`
	ID                     types.String    `tfsdk:"id"`
	Server                 []XrayDNSServer `tfsdk:"server"`
	Hosts                  types.Map       `tfsdk:"hosts"`
	QueryStrategy          types.String    `tfsdk:"query_strategy"`
	Tag                    types.String    `tfsdk:"tag"`
	DisableCache           types.Bool      `tfsdk:"disable_cache"`
	DisableFallback        types.Bool      `tfsdk:"disable_fallback"`
	DisableFallbackIfMatch types.Bool      `tfsdk:"disable_fallback_if_match"`
	ClientIP               types.String    `tfsdk:"client_ip"`
	EnableParallelQuery    types.Bool      `tfsdk:"enable_parallel_query"`
	UseSystemHosts         types.Bool      `tfsdk:"use_system_hosts"`
}

type XrayDNSServer struct {
	ID            types.String `tfsdk:"id"`
	Address       types.String `tfsdk:"address"`
	Port          types.Int64  `tfsdk:"port"`
	Domains       types.List   `tfsdk:"domains"`
	ExpectIPs     types.List   `tfsdk:"expect_ips"`
	UnexpectedIPs types.List   `tfsdk:"unexpected_ips"`
	SkipFallback  types.Bool   `tfsdk:"skip_fallback"`
	QueryStrategy types.String `tfsdk:"query_strategy"`
	DisableCache  types.Bool   `tfsdk:"disable_cache"`
	FinalQuery    types.Bool   `tfsdk:"final_query"`
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

func xrayDNSSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"script": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Path to a Lua script on the panel host (3x-ui v3.10.0+). Set an empty string to disable; omission preserves the prior value.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"query_strategy": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tag": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"disable_cache": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"disable_fallback": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"disable_fallback_if_match": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"client_ip": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enable_parallel_query": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"use_system_hosts": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"hosts": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"server": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Optional: true, Computed: true,
							Description:   "DNS server identifier used by Lua scripts (3x-ui v3.10.0+).",
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},

						"address": schema.StringAttribute{
							Required: true,
						},
						"port": schema.Int64Attribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
						"domains": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
						},
						"expect_ips": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
						},
						"unexpected_ips": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
						},
						"skip_fallback": schema.BoolAttribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"query_strategy": schema.StringAttribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"disable_cache": schema.BoolAttribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"final_query": schema.BoolAttribute{
							Optional: true, Computed: true,
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Typed model -> untyped map  (for buildXrayDNSJSON)
// ---------------------------------------------------------------------------

func expandXrayDNS(m *XrayDNSModel) map[string]any {
	out := map[string]any{}
	if !m.Script.IsNull() && !m.Script.IsUnknown() {
		out["script"] = m.Script.ValueString()
	}

	if servers := expandDNSServerList(m.Server); len(servers) > 0 {
		out["server"] = servers
	}
	if !m.Hosts.IsNull() && !m.Hosts.IsUnknown() {
		hosts := map[string]any{}
		for k, v := range m.Hosts.Elements() {
			if sv, ok := v.(types.String); ok && !sv.IsNull() && !sv.IsUnknown() {
				hosts[k] = sv.ValueString()
			}
		}
		if len(hosts) > 0 {
			out["hosts"] = hosts
		}
	}
	if !m.QueryStrategy.IsNull() && !m.QueryStrategy.IsUnknown() {
		out["query_strategy"] = m.QueryStrategy.ValueString()
	}
	if !m.Tag.IsNull() && !m.Tag.IsUnknown() {
		out["tag"] = m.Tag.ValueString()
	}
	if !m.DisableCache.IsNull() && !m.DisableCache.IsUnknown() {
		out["disable_cache"] = m.DisableCache.ValueBool()
	}
	if !m.DisableFallback.IsNull() && !m.DisableFallback.IsUnknown() {
		out["disable_fallback"] = m.DisableFallback.ValueBool()
	}
	if !m.DisableFallbackIfMatch.IsNull() && !m.DisableFallbackIfMatch.IsUnknown() {
		out["disable_fallback_if_match"] = m.DisableFallbackIfMatch.ValueBool()
	}
	if !m.ClientIP.IsNull() && !m.ClientIP.IsUnknown() {
		out["client_ip"] = m.ClientIP.ValueString()
	}
	if !m.EnableParallelQuery.IsNull() && !m.EnableParallelQuery.IsUnknown() {
		out["enable_parallel_query"] = m.EnableParallelQuery.ValueBool()
	}
	if !m.UseSystemHosts.IsNull() && !m.UseSystemHosts.IsUnknown() {
		out["use_system_hosts"] = m.UseSystemHosts.ValueBool()
	}

	return out
}

func expandDNSServerList(servers []XrayDNSServer) []any {
	if len(servers) == 0 {
		return nil
	}
	out := make([]any, 0, len(servers))
	for _, s := range servers {
		entry := map[string]any{}
		if !s.ID.IsNull() && !s.ID.IsUnknown() {
			entry["id"] = s.ID.ValueString()
		}
		if !s.Address.IsNull() && !s.Address.IsUnknown() {
			entry["address"] = s.Address.ValueString()
		}
		if !s.Port.IsNull() && !s.Port.IsUnknown() {
			entry["port"] = s.Port.ValueInt64()
		}
		if !s.Domains.IsNull() && !s.Domains.IsUnknown() {
			entry["domains"] = expandAttrStringList(s.Domains.Elements())
		}
		if !s.ExpectIPs.IsNull() && !s.ExpectIPs.IsUnknown() {
			entry["expect_ips"] = expandAttrStringList(s.ExpectIPs.Elements())
		}
		if !s.UnexpectedIPs.IsNull() && !s.UnexpectedIPs.IsUnknown() {
			entry["unexpected_ips"] = expandAttrStringList(s.UnexpectedIPs.Elements())
		}
		if !s.SkipFallback.IsNull() && !s.SkipFallback.IsUnknown() {
			entry["skip_fallback"] = s.SkipFallback.ValueBool()
		}
		if !s.QueryStrategy.IsNull() && !s.QueryStrategy.IsUnknown() {
			entry["query_strategy"] = s.QueryStrategy.ValueString()
		}
		if !s.DisableCache.IsNull() && !s.DisableCache.IsUnknown() {
			entry["disable_cache"] = s.DisableCache.ValueBool()
		}
		if !s.FinalQuery.IsNull() && !s.FinalQuery.IsUnknown() {
			entry["final_query"] = s.FinalQuery.ValueBool()
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

// expandAttrStringList converts []attr.Value (from types.List) to []any of strings.
func expandAttrStringList(elems []attr.Value) []any {
	out := make([]any, 0, len(elems))
	for _, e := range elems {
		if sv, ok := e.(types.String); ok && !sv.IsNull() && !sv.IsUnknown() {
			out = append(out, sv.ValueString())
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Untyped map -> typed model  (from flattenXrayDNSToMap output)
// ---------------------------------------------------------------------------

func flattenXrayDNS(data map[string]any) *XrayDNSModel {
	m := &XrayDNSModel{
		ID:                     types.StringValue(xraySectionDNS.id),
		Hosts:                  types.MapNull(types.StringType),
		QueryStrategy:          types.StringNull(),
		Tag:                    types.StringNull(),
		DisableCache:           types.BoolNull(),
		DisableFallback:        types.BoolNull(),
		DisableFallbackIfMatch: types.BoolNull(),
		ClientIP:               types.StringNull(),
		EnableParallelQuery:    types.BoolNull(),
		UseSystemHosts:         types.BoolNull(),
	}

	if v, ok := data["script"].(string); ok {
		m.Script = types.StringValue(v)
	}

	if v, ok := data["server"].([]any); ok {
		m.Server = flattenDNSServerList(v)
	}
	if v, ok := data["hosts"].(map[string]string); ok && len(v) > 0 {
		elems := map[string]attr.Value{}
		for k, val := range v {
			elems[k] = types.StringValue(val)
		}
		m.Hosts = types.MapValueMust(types.StringType, elems)
	}
	if v, ok := data["query_strategy"].(string); ok {
		m.QueryStrategy = types.StringValue(v)
	}
	if v, ok := data["tag"].(string); ok {
		m.Tag = types.StringValue(v)
	}
	if v, ok := data["disable_cache"].(bool); ok {
		m.DisableCache = types.BoolValue(v)
	}
	if v, ok := data["disable_fallback"].(bool); ok {
		m.DisableFallback = types.BoolValue(v)
	}
	if v, ok := data["disable_fallback_if_match"].(bool); ok {
		m.DisableFallbackIfMatch = types.BoolValue(v)
	}
	if v, ok := data["client_ip"].(string); ok {
		m.ClientIP = types.StringValue(v)
	}
	if v, ok := data["enable_parallel_query"].(bool); ok {
		m.EnableParallelQuery = types.BoolValue(v)
	}
	if v, ok := data["use_system_hosts"].(bool); ok {
		m.UseSystemHosts = types.BoolValue(v)
	}

	return m
}

func flattenDNSServerList(list []any) []XrayDNSServer {
	if len(list) == 0 {
		return nil
	}
	out := make([]XrayDNSServer, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := XrayDNSServer{
			Address:       types.StringNull(),
			Port:          types.Int64Null(),
			Domains:       types.ListNull(types.StringType),
			ExpectIPs:     types.ListNull(types.StringType),
			UnexpectedIPs: types.ListNull(types.StringType),
			SkipFallback:  types.BoolNull(),
			QueryStrategy: types.StringNull(),
			DisableCache:  types.BoolNull(),
			FinalQuery:    types.BoolNull(),
		}
		if v, ok := m["id"].(string); ok {
			entry.ID = types.StringValue(v)
		}
		if v, ok := m["address"].(string); ok {
			entry.Address = types.StringValue(v)
		}
		if v, ok := m["port"]; ok {
			entry.Port = types.Int64Value(int64(intValue(v)))
		}
		if v, ok := m["domains"].([]any); ok && len(v) > 0 {
			entry.Domains = flattenToStringList(v)
		}
		if v, ok := m["expect_ips"].([]any); ok && len(v) > 0 {
			entry.ExpectIPs = flattenToStringList(v)
		}
		if v, ok := m["unexpected_ips"].([]any); ok && len(v) > 0 {
			entry.UnexpectedIPs = flattenToStringList(v)
		}
		if v, ok := m["skip_fallback"].(bool); ok {
			entry.SkipFallback = types.BoolValue(v)
		}
		if v, ok := m["query_strategy"].(string); ok {
			entry.QueryStrategy = types.StringValue(v)
		}
		if v, ok := m["disable_cache"].(bool); ok {
			entry.DisableCache = types.BoolValue(v)
		}
		if v, ok := m["final_query"].(bool); ok {
			entry.FinalQuery = types.BoolValue(v)
		}
		out = append(out, entry)
	}
	return out
}

// flattenToStringList converts []any of strings to types.List of StringType.
func flattenToStringList(list []any) types.List {
	elems := make([]attr.Value, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			elems = append(elems, types.StringValue(s))
		}
	}
	return types.ListValueMust(types.StringType, elems)
}

// ---------------------------------------------------------------------------
// Legacy untyped build / flatten (used by CRUD layer)
// ---------------------------------------------------------------------------

func buildXrayDNSJSON(d map[string]any) any {
	payload := map[string]any{}
	if v, ok := d["script"].(string); ok {
		payload["script"] = v
	}

	if v, ok := d["server"]; ok {
		if list, ok := v.([]any); ok {
			payload["servers"] = expandDNSServers(list)
		}
	}
	if v, ok := d["hosts"]; ok {
		if m, ok := v.(map[string]any); ok {
			payload["hosts"] = expandStringMap(m)
		}
	}
	if v, ok := d["query_strategy"].(string); ok && v != "" {
		payload["queryStrategy"] = v
	}
	if v, ok := d["tag"].(string); ok && v != "" {
		payload["tag"] = v
	}
	if v, ok := d["disable_cache"]; ok {
		payload["disableCache"] = boolValue(v)
	}
	if v, ok := d["disable_fallback"]; ok {
		payload["disableFallback"] = boolValue(v)
	}
	if v, ok := d["disable_fallback_if_match"]; ok {
		payload["disableFallbackIfMatch"] = boolValue(v)
	}
	if v, ok := d["client_ip"].(string); ok && v != "" {
		payload["clientIp"] = v
	}
	if v, ok := d["enable_parallel_query"]; ok {
		payload["enableParallelQuery"] = boolValue(v)
	}
	if v, ok := d["use_system_hosts"]; ok {
		payload["useSystemHosts"] = boolValue(v)
	}

	return payload
}

// expandDNSServers converts TF server blocks to JSON.
// A server with only address is serialized as a plain string.
// A server with additional fields is serialized as an object.
func expandDNSServers(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		address, _ := m["address"].(string)
		if address == "" {
			continue
		}

		// Check if server has any fields beyond address
		_, hasExtra := m["id"].(string)
		if v, ok := m["port"]; ok {
			if intValue(v) != 0 {
				hasExtra = true
			}
		}
		if v, ok := m["domains"].([]any); ok && len(v) > 0 {
			hasExtra = true
		}
		if v, ok := m["expect_ips"].([]any); ok && len(v) > 0 {
			hasExtra = true
		}
		if v, ok := m["unexpected_ips"].([]any); ok && len(v) > 0 {
			hasExtra = true
		}
		if v, ok := m["skip_fallback"]; ok && boolValue(v) {
			hasExtra = true
		}
		if v, ok := m["query_strategy"].(string); ok && v != "" {
			hasExtra = true
		}
		if v, ok := m["disable_cache"]; ok && boolValue(v) {
			hasExtra = true
		}
		if v, ok := m["final_query"]; ok && boolValue(v) {
			hasExtra = true
		}

		if !hasExtra {
			out = append(out, address)
			continue
		}

		entry := map[string]any{
			"address": address,
		}
		if v, ok := m["id"].(string); ok {
			entry["id"] = v
		}
		if v, ok := m["port"]; ok {
			if p := intValue(v); p != 0 {
				entry["port"] = p
			}
		}
		if v, ok := m["domains"].([]any); ok && len(v) > 0 {
			entry["domains"] = expandStringList(v)
		}
		if v, ok := m["expect_ips"].([]any); ok && len(v) > 0 {
			entry["expectedIPs"] = expandStringList(v)
		}
		if v, ok := m["unexpected_ips"].([]any); ok && len(v) > 0 {
			entry["unexpectedIPs"] = expandStringList(v)
		}
		if v, ok := m["skip_fallback"]; ok {
			entry["skipFallback"] = boolValue(v)
		}
		if v, ok := m["query_strategy"].(string); ok && v != "" {
			entry["queryStrategy"] = v
		}
		if v, ok := m["disable_cache"]; ok {
			entry["disableCache"] = boolValue(v)
		}
		if v, ok := m["final_query"]; ok {
			entry["finalQuery"] = boolValue(v)
		}

		out = append(out, entry)
	}
	return out
}

func flattenXrayDNSToMap(data any) map[string]any {
	out := map[string]any{}
	if data == nil {
		return out
	}

	var payload map[string]any
	switch v := data.(type) {
	case map[string]any:
		payload = v
	case string:
		if err := json.Unmarshal([]byte(v), &payload); err != nil {
			return out
		}
	default:
		return out
	}

	if v, ok := payload["script"].(string); ok {
		out["script"] = v
	}

	if v, ok := payload["servers"].([]any); ok {
		out["server"] = flattenDNSServers(v)
	}
	if v, ok := payload["hosts"].(map[string]any); ok {
		out["hosts"] = flattenStringMap(v)
	}
	if v, ok := payload["queryStrategy"].(string); ok {
		out["query_strategy"] = v
	}
	if v, ok := payload["tag"].(string); ok {
		out["tag"] = v
	}
	if v, ok := payload["disableCache"].(bool); ok {
		out["disable_cache"] = v
	}
	if v, ok := payload["disableFallback"].(bool); ok {
		out["disable_fallback"] = v
	}
	if v, ok := payload["disableFallbackIfMatch"].(bool); ok {
		out["disable_fallback_if_match"] = v
	}
	if v, ok := payload["clientIp"].(string); ok {
		out["client_ip"] = v
	}
	if v, ok := payload["enableParallelQuery"].(bool); ok {
		out["enable_parallel_query"] = v
	}
	if v, ok := payload["useSystemHosts"].(bool); ok {
		out["use_system_hosts"] = v
	}

	return out
}

// flattenDNSServers handles both string servers and object servers from API.
func flattenDNSServers(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		switch v := item.(type) {
		case string:
			out = append(out, map[string]any{
				"address": v,
			})
		case map[string]any:
			entry := map[string]any{}
			if id, ok := v["id"].(string); ok {
				entry["id"] = id
			}
			if addr, ok := v["address"].(string); ok {
				entry["address"] = addr
			}
			if p, ok := v["port"]; ok {
				entry["port"] = intValue(p)
			}
			if d, ok := v["domains"].([]any); ok {
				entry["domains"] = d
			}
			if e, ok := v["expectedIPs"].([]any); ok {
				entry["expect_ips"] = e
			}
			if u, ok := v["unexpectedIPs"].([]any); ok {
				entry["unexpected_ips"] = u
			}
			if s, ok := v["skipFallback"].(bool); ok {
				entry["skip_fallback"] = s
			}
			if q, ok := v["queryStrategy"].(string); ok {
				entry["query_strategy"] = q
			}
			if dc, ok := v["disableCache"].(bool); ok {
				entry["disable_cache"] = dc
			}
			if f, ok := v["finalQuery"].(bool); ok {
				entry["final_query"] = f
			}
			out = append(out, entry)
		}
	}
	return out
}
