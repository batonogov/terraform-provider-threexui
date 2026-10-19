package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// Typed models
// ---------------------------------------------------------------------------

type XrayRoutingModel struct {
	Script         types.String      `tfsdk:"script"`
	ID             types.String      `tfsdk:"id"`
	DomainStrategy types.String      `tfsdk:"domain_strategy"`
	DomainMatcher  types.String      `tfsdk:"domain_matcher"`
	Rule           []XrayRoutingRule `tfsdk:"rule"`
}

type XrayRoutingRule struct {
	Type        types.String `tfsdk:"type"`
	Domain      types.List   `tfsdk:"domain"`
	IP          types.List   `tfsdk:"ip"`
	Port        types.String `tfsdk:"port"`
	SourcePort  types.String `tfsdk:"source_port"`
	Network     types.String `tfsdk:"network"`
	Source      types.List   `tfsdk:"source"`
	User        types.List   `tfsdk:"user"`
	InboundTag  types.List   `tfsdk:"inbound_tag"`
	Protocol    types.List   `tfsdk:"protocol"`
	Attrs       types.String `tfsdk:"attrs"`
	OutboundTag types.String `tfsdk:"outbound_tag"`
	BalancerTag types.String `tfsdk:"balancer_tag"`
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

func xrayRoutingSchema() schema.Schema {
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
			"domain_strategy": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_matcher": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"rule": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"domain": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
						},
						"ip": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
						},
						"port": schema.StringAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"source_port": schema.StringAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"network": schema.StringAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"source": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
						},
						"user": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
						},
						"inbound_tag": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
						},
						"protocol": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
						},
						"attrs": schema.StringAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"outbound_tag": schema.StringAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"balancer_tag": schema.StringAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Typed model <-> untyped map conversion
// ---------------------------------------------------------------------------

// expandXrayRouting converts the typed model to the untyped map format that
// buildXrayRoutingJSON expects.
func expandXrayRouting(m *XrayRoutingModel) map[string]any {
	out := map[string]any{}
	if !m.Script.IsNull() && !m.Script.IsUnknown() {
		out["script"] = m.Script.ValueString()
	}

	if !m.DomainStrategy.IsNull() && !m.DomainStrategy.IsUnknown() {
		out["domain_strategy"] = m.DomainStrategy.ValueString()
	}
	if !m.DomainMatcher.IsNull() && !m.DomainMatcher.IsUnknown() {
		out["domain_matcher"] = m.DomainMatcher.ValueString()
	}

	if len(m.Rule) > 0 {
		rules := make([]any, 0, len(m.Rule))
		for _, r := range m.Rule {
			entry := map[string]any{}

			if !r.Type.IsNull() && !r.Type.IsUnknown() {
				entry["type"] = r.Type.ValueString()
			}
			if !r.Domain.IsNull() && !r.Domain.IsUnknown() {
				entry["domain"] = typesListToAnySlice(r.Domain)
			}
			if !r.IP.IsNull() && !r.IP.IsUnknown() {
				entry["ip"] = typesListToAnySlice(r.IP)
			}
			if !r.Port.IsNull() && !r.Port.IsUnknown() {
				entry["port"] = r.Port.ValueString()
			}
			if !r.SourcePort.IsNull() && !r.SourcePort.IsUnknown() {
				entry["source_port"] = r.SourcePort.ValueString()
			}
			if !r.Network.IsNull() && !r.Network.IsUnknown() {
				entry["network"] = r.Network.ValueString()
			}
			if !r.Source.IsNull() && !r.Source.IsUnknown() {
				entry["source"] = typesListToAnySlice(r.Source)
			}
			if !r.User.IsNull() && !r.User.IsUnknown() {
				entry["user"] = typesListToAnySlice(r.User)
			}
			if !r.InboundTag.IsNull() && !r.InboundTag.IsUnknown() {
				entry["inbound_tag"] = typesListToAnySlice(r.InboundTag)
			}
			if !r.Protocol.IsNull() && !r.Protocol.IsUnknown() {
				entry["protocol"] = typesListToAnySlice(r.Protocol)
			}
			if !r.Attrs.IsNull() && !r.Attrs.IsUnknown() {
				entry["attrs"] = r.Attrs.ValueString()
			}
			if !r.OutboundTag.IsNull() && !r.OutboundTag.IsUnknown() {
				entry["outbound_tag"] = r.OutboundTag.ValueString()
			}
			if !r.BalancerTag.IsNull() && !r.BalancerTag.IsUnknown() {
				entry["balancer_tag"] = r.BalancerTag.ValueString()
			}

			rules = append(rules, entry)
		}
		out["rule"] = rules
	}

	return out
}

// flattenXrayRouting converts the output of flattenXrayRoutingToMap back to
// the typed model. Input has keys like domain_strategy, domain_matcher, rule.
func flattenXrayRouting(data map[string]any) *XrayRoutingModel {
	m := &XrayRoutingModel{
		ID: types.StringValue("xray_routing"),
	}

	if v, ok := data["script"].(string); ok {
		m.Script = types.StringValue(v)
	}

	if v, ok := data["domain_strategy"].(string); ok && v != "" {
		m.DomainStrategy = types.StringValue(v)
	} else {
		m.DomainStrategy = types.StringNull()
	}

	if v, ok := data["domain_matcher"].(string); ok && v != "" {
		m.DomainMatcher = types.StringValue(v)
	} else {
		m.DomainMatcher = types.StringNull()
	}

	if v, ok := data["rule"].([]any); ok && len(v) > 0 {
		m.Rule = make([]XrayRoutingRule, 0, len(v))
		for _, item := range v {
			rm, ok := item.(map[string]any)
			if !ok {
				continue
			}
			rule := XrayRoutingRule{}

			if s, ok := rm["type"].(string); ok && s != "" {
				rule.Type = types.StringValue(s)
			} else {
				rule.Type = types.StringNull()
			}

			rule.Domain = anySliceToTypesList(rm["domain"])
			rule.IP = anySliceToTypesList(rm["ip"])

			if s, ok := rm["port"].(string); ok && s != "" {
				rule.Port = types.StringValue(s)
			} else {
				rule.Port = types.StringNull()
			}
			if s, ok := rm["source_port"].(string); ok && s != "" {
				rule.SourcePort = types.StringValue(s)
			} else {
				rule.SourcePort = types.StringNull()
			}
			if s, ok := rm["network"].(string); ok && s != "" {
				rule.Network = types.StringValue(s)
			} else {
				rule.Network = types.StringNull()
			}

			rule.Source = anySliceToTypesList(rm["source"])
			rule.User = anySliceToTypesList(rm["user"])
			rule.InboundTag = anySliceToTypesList(rm["inbound_tag"])
			rule.Protocol = anySliceToTypesList(rm["protocol"])

			if s, ok := rm["attrs"].(string); ok && s != "" {
				rule.Attrs = types.StringValue(s)
			} else {
				rule.Attrs = types.StringNull()
			}
			if s, ok := rm["outbound_tag"].(string); ok && s != "" {
				rule.OutboundTag = types.StringValue(s)
			} else {
				rule.OutboundTag = types.StringNull()
			}
			if s, ok := rm["balancer_tag"].(string); ok && s != "" {
				rule.BalancerTag = types.StringValue(s)
			} else {
				rule.BalancerTag = types.StringNull()
			}

			m.Rule = append(m.Rule, rule)
		}
	}

	return m
}

// ---------------------------------------------------------------------------
// Existing build/flatten functions (untyped map <-> Xray JSON)
// ---------------------------------------------------------------------------

func buildXrayRoutingJSON(d map[string]any) any {
	payload := map[string]any{}
	if v, ok := d["script"].(string); ok {
		payload["script"] = v
	}

	if v, ok := d["domain_strategy"].(string); ok && v != "" {
		payload["domainStrategy"] = v
	}
	if v, ok := d["domain_matcher"].(string); ok && v != "" {
		payload["domainMatcher"] = v
	}
	if v, ok := d["rule"]; ok {
		if list, ok := v.([]any); ok {
			rules := expandRoutingRules(list)
			apiRule := map[string]any{
				"type":        "field",
				"inboundTag":  []string{"api"},
				"outboundTag": "api",
			}
			payload["rules"] = append([]any{apiRule}, rules...)
		}
	}

	return payload
}

func expandRoutingRules(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if isInternalAPIRoutingRule(m) || isManagedDnsAllowRule(m) {
			continue
		}
		entry := map[string]any{}

		if v, ok := m["type"].(string); ok && v != "" {
			entry["type"] = v
		}
		if v, ok := m["domain"].([]any); ok && len(v) > 0 {
			entry["domain"] = expandStringList(v)
		}
		if v, ok := m["ip"].([]any); ok && len(v) > 0 {
			entry["ip"] = expandStringList(v)
		}
		if v, ok := m["port"].(string); ok && v != "" {
			entry["port"] = v
		}
		if v, ok := m["source_port"].(string); ok && v != "" {
			entry["sourcePort"] = v
		}
		if v, ok := m["network"].(string); ok && v != "" {
			entry["network"] = v
		}
		if v, ok := m["source"].([]any); ok && len(v) > 0 {
			entry["source"] = expandStringList(v)
		}
		if v, ok := m["user"].([]any); ok && len(v) > 0 {
			entry["user"] = expandStringList(v)
		}
		if v, ok := m["inbound_tag"].([]any); ok && len(v) > 0 {
			entry["inboundTag"] = expandStringList(v)
		}
		if v, ok := m["protocol"].([]any); ok && len(v) > 0 {
			entry["protocol"] = expandStringList(v)
		}
		if v, ok := m["attrs"].(string); ok && v != "" {
			entry["attrs"] = v
		}
		if v, ok := m["outbound_tag"].(string); ok && v != "" {
			entry["outboundTag"] = v
		}
		if v, ok := m["balancer_tag"].(string); ok && v != "" {
			entry["balancerTag"] = v
		}

		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func flattenXrayRoutingToMap(data any) map[string]any {
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

	if v, ok := payload["domainStrategy"].(string); ok {
		out["domain_strategy"] = v
	}
	if v, ok := payload["domainMatcher"].(string); ok {
		out["domain_matcher"] = v
	}
	if v, ok := payload["rules"].([]any); ok {
		out["rule"] = flattenRoutingRules(v)
	}

	return out
}

func flattenRoutingRules(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if isInternalAPIRoutingRule(m) || isManagedDnsAllowRule(m) {
			continue
		}
		entry := map[string]any{}

		if v, ok := m["type"].(string); ok {
			entry["type"] = v
		}
		if v, ok := m["domain"].([]any); ok {
			entry["domain"] = v
		}
		if v, ok := m["ip"].([]any); ok {
			entry["ip"] = v
		}
		if v, ok := m["port"].(string); ok {
			entry["port"] = v
		}
		if v, ok := m["sourcePort"].(string); ok {
			entry["source_port"] = v
		}
		if v, ok := m["network"].(string); ok {
			entry["network"] = v
		}
		if v, ok := m["source"].([]any); ok {
			entry["source"] = v
		}
		if v, ok := m["user"].([]any); ok {
			entry["user"] = v
		}
		if v, ok := m["inboundTag"].([]any); ok {
			entry["inbound_tag"] = v
		}
		if v, ok := m["protocol"].([]any); ok {
			entry["protocol"] = v
		}
		if v, ok := m["attrs"].(string); ok {
			entry["attrs"] = v
		}
		if v, ok := m["outboundTag"].(string); ok {
			entry["outbound_tag"] = v
		}
		if v, ok := m["balancerTag"].(string); ok {
			entry["balancer_tag"] = v
		}

		out = append(out, entry)
	}
	return out
}

func isInternalAPIRoutingRule(m map[string]any) bool {
	outboundTag, _ := m["outboundTag"].(string)
	if outboundTag == "" {
		outboundTag, _ = m["outbound_tag"].(string)
	}
	if outboundTag != "api" {
		return false
	}
	return routingValueContainsString(m["inboundTag"], "api") ||
		routingValueContainsString(m["inbound_tag"], "api")
}

// isManagedDnsAllowRule mirrors 3x-ui v3.5.0's service.dnsAllowRuleShape: a plain
// "type=field, ip=[...], port=..., outboundTag=direct" rule with no other
// matchers (an "enabled":true key is tolerated). The panel inserts these
// before the geoip:private block rule (service.EnsureDnsServerRouting) to keep
// private/internal DNS servers reachable, and rebuilds them on every xray-
// template save. Filtering them out of Read avoids a permanent drift — without
// this the rules reappear on every `terraform plan` because the server re-adds
// them after each apply. A rule carrying any extra matcher is left untouched
// (the upstream shape check is strict, so hand-written allow-rules survive).
func isManagedDnsAllowRule(m map[string]any) bool {
	if t, _ := m["type"].(string); t != "field" {
		return false
	}
	outboundTag, _ := m["outboundTag"].(string)
	if outboundTag == "" {
		outboundTag, _ = m["outbound_tag"].(string)
	}
	if outboundTag != "direct" {
		return false
	}
	if _, ok := m["ip"]; !ok {
		return false
	}
	if _, ok := m["port"]; !ok {
		return false
	}
	for key := range m {
		switch key {
		case "type", "outboundTag", "outbound_tag", "ip", "port":
			continue
		case "enabled":
			if enabled, ok := m[key].(bool); !ok || !enabled {
				return false
			}
			continue
		default:
			return false
		}
	}
	return true
}

func routingValueContainsString(raw any, value string) bool {
	switch v := raw.(type) {
	case string:
		return v == value
	case []string:
		for _, item := range v {
			if item == value {
				return true
			}
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == value {
				return true
			}
		}
	}
	return false
}

func validateNoAPIRoutingRules(rules []XrayRoutingRule) string {
	for _, r := range rules {
		if r.OutboundTag.ValueString() != "api" {
			continue
		}
		if r.InboundTag.IsNull() || r.InboundTag.IsUnknown() {
			continue
		}
		var tags []string
		r.InboundTag.ElementsAs(context.Background(), &tags, false)
		for _, tag := range tags {
			if tag == "api" {
				return "API routing rules (inbound_tag containing \"api\" with outbound_tag \"api\") are automatically managed by the `api` block in `threexui_xray_basics`. Remove this rule from `threexui_xray_routing`."
			}
		}
	}
	return ""
}

func ensureNoAPIRoutingRules(rules []XrayRoutingRule, diags *diag.Diagnostics) {
	if msg := validateNoAPIRoutingRules(rules); msg != "" {
		diags.AddError("Invalid routing rule", msg)
	}
}
