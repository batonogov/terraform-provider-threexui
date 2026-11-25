package provider

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// Typed models
// ---------------------------------------------------------------------------

// XrayObservatoryModel is the Terraform state model for the
// threexui_xray_observatory resource. It manages both the "observatory" and
// "burstObservatory" top-level xray template keys (xray-core v26.6.27+,
// 3x-ui v3.4.2+).
type XrayObservatoryModel struct {
	ID               types.String           `tfsdk:"id"`
	Observatory      []XrayObservatoryEntry `tfsdk:"observatory"`
	BurstObservatory []XrayBurstObservatory `tfsdk:"burst_observatory"`
}

// XrayObservatoryEntry mirrors xray-core's ObservatoryConfig.
type XrayObservatoryEntry struct {
	Tag               types.String `tfsdk:"tag"`
	SubjectSelector   types.List   `tfsdk:"subject_selector"` // list of strings
	ProbeURL          types.String `tfsdk:"probe_url"`
	ProbeInterval     types.String `tfsdk:"probe_interval"`
	EnableConcurrency types.Bool   `tfsdk:"enable_concurrency"`
}

// XrayBurstObservatory mirrors xray-core's BurstObservatoryConfig.
type XrayBurstObservatory struct {
	Tag             types.String          `tfsdk:"tag"`
	SubjectSelector types.List            `tfsdk:"subject_selector"` // list of strings
	PingConfig      []XrayBurstPingConfig `tfsdk:"ping_config"`
}

// XrayBurstPingConfig mirrors xray-core's PingConfig used inside
// BurstObservatory.
type XrayBurstPingConfig struct {
	Destination    types.String `tfsdk:"destination"`
	Interval       types.String `tfsdk:"interval"`
	ConnectTimeout types.String `tfsdk:"connect_timeout"`
	Timeout        types.String `tfsdk:"timeout"`
	Samples        types.Int64  `tfsdk:"samples"`
	SamplingCount  types.Int64  `tfsdk:"sampling_count"`
	Lazy           types.Bool   `tfsdk:"lazy"`
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

func xrayObservatorySchema() schema.Schema {
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
			"observatory": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"tag": schema.StringAttribute{
							Required:    true,
							Description: "Tag identifying this observatory entry. Use default for a single flat Xray-core observatory.",
						},
						"subject_selector": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							Description: "List of outbound tag prefixes/patterns to probe.",
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
						},
						"probe_url": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "URL to probe (e.g. https://www.google.com/generate_204).",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"probe_interval": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Interval between probes (xray duration string, e.g. \"1m\").",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"enable_concurrency": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Probe all matching outbounds concurrently.",
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"burst_observatory": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"tag": schema.StringAttribute{
							Required:    true,
							Description: "Tag identifying this burst observatory entry. Use default for a single flat Xray-core burst observatory.",
						},
						"subject_selector": schema.ListAttribute{
							Optional:    true,
							Computed:    true,
							Description: "List of outbound tag prefixes/patterns to probe.",
							ElementType: types.StringType,
							PlanModifiers: []planmodifier.List{
								listplanmodifier.UseStateForUnknown(),
							},
						},
					},
					Blocks: map[string]schema.Block{
						"ping_config": singletonListNestedBlock(schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"destination": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "Probe destination URL.",
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"interval": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "Ping interval (xray duration string, e.g. \"1m\").",
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"connect_timeout": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "Connection timeout for each probe (xray duration string).",
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"timeout": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "Overall probe timeout (xray duration string).",
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"samples": schema.Int64Attribute{
										Optional:    true,
										Computed:    true,
										Description: "Number of samples per probe.",
										PlanModifiers: []planmodifier.Int64{
											int64planmodifier.UseStateForUnknown(),
										},
									},
									"sampling_count": schema.Int64Attribute{
										Optional:    true,
										Computed:    true,
										Description: "Number of sampling rounds per probe.",
										PlanModifiers: []planmodifier.Int64{
											int64planmodifier.UseStateForUnknown(),
										},
									},
									"lazy": schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Description: "Only probe when a connection request is received.",
										PlanModifiers: []planmodifier.Bool{
											boolplanmodifier.UseStateForUnknown(),
										},
									},
								},
							},
						}),
					},
				},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Typed expand: model → untyped map
// ---------------------------------------------------------------------------

func expandXrayObservatory(m *XrayObservatoryModel) map[string]any {
	out := map[string]any{}

	if entries := expandObservatoryEntryList(m.Observatory); len(entries) > 0 {
		out["observatory"] = entries
	}
	if entries := expandBurstObservatoryList(m.BurstObservatory); len(entries) > 0 {
		out["burst_observatory"] = entries
	}

	return out
}

func expandObservatoryEntryList(entries []XrayObservatoryEntry) []any {
	if len(entries) == 0 {
		return nil
	}
	out := make([]any, 0, len(entries))
	for _, e := range entries {
		entry := map[string]any{}
		if !e.Tag.IsNull() && !e.Tag.IsUnknown() {
			entry["tag"] = e.Tag.ValueString()
		}
		if !e.SubjectSelector.IsNull() && !e.SubjectSelector.IsUnknown() {
			entry["subjectSelector"] = expandAttrStringList(e.SubjectSelector.Elements())
		}
		if !e.ProbeURL.IsNull() && !e.ProbeURL.IsUnknown() {
			entry["probeURL"] = e.ProbeURL.ValueString()
		}
		if !e.ProbeInterval.IsNull() && !e.ProbeInterval.IsUnknown() {
			entry["probeInterval"] = e.ProbeInterval.ValueString()
		}
		if !e.EnableConcurrency.IsNull() && !e.EnableConcurrency.IsUnknown() {
			entry["enableConcurrency"] = e.EnableConcurrency.ValueBool()
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func expandBurstObservatoryList(entries []XrayBurstObservatory) []any {
	if len(entries) == 0 {
		return nil
	}
	out := make([]any, 0, len(entries))
	for _, e := range entries {
		entry := map[string]any{}
		if !e.Tag.IsNull() && !e.Tag.IsUnknown() {
			entry["tag"] = e.Tag.ValueString()
		}
		if !e.SubjectSelector.IsNull() && !e.SubjectSelector.IsUnknown() {
			entry["subjectSelector"] = expandAttrStringList(e.SubjectSelector.Elements())
		}
		if len(e.PingConfig) > 0 {
			if pc := expandBurstPingConfig(e.PingConfig); pc != nil {
				entry["pingConfig"] = pc
			}
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func expandBurstPingConfig(configs []XrayBurstPingConfig) map[string]any {
	if len(configs) == 0 {
		return nil
	}
	c := configs[0]
	out := map[string]any{}
	if !c.Destination.IsNull() && !c.Destination.IsUnknown() {
		out["destination"] = c.Destination.ValueString()
	}
	if !c.Interval.IsNull() && !c.Interval.IsUnknown() {
		out["interval"] = c.Interval.ValueString()
	}
	if !c.ConnectTimeout.IsNull() && !c.ConnectTimeout.IsUnknown() {
		out["connectTimeout"] = c.ConnectTimeout.ValueString()
	}
	if !c.Timeout.IsNull() && !c.Timeout.IsUnknown() {
		out["timeout"] = c.Timeout.ValueString()
	}
	if !c.Samples.IsNull() && !c.Samples.IsUnknown() {
		out["samples"] = int(c.Samples.ValueInt64())
	}
	if !c.SamplingCount.IsNull() && !c.SamplingCount.IsUnknown() {
		out["samplingCount"] = int(c.SamplingCount.ValueInt64())
	}
	if !c.Lazy.IsNull() && !c.Lazy.IsUnknown() {
		out["lazy"] = c.Lazy.ValueBool()
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ---------------------------------------------------------------------------
// Typed flatten: untyped map → model
// ---------------------------------------------------------------------------

func flattenXrayObservatory(data map[string]any) *XrayObservatoryModel {
	m := &XrayObservatoryModel{
		ID: types.StringValue("xray_observatory"),
	}

	if v, ok := data["observatory"].([]any); ok {
		m.Observatory = flattenObservatoryEntryList(v)
	}
	if v, ok := data["burst_observatory"].([]any); ok {
		m.BurstObservatory = flattenBurstObservatoryList(v)
	}

	return m
}

func flattenObservatoryEntryList(list []any) []XrayObservatoryEntry {
	if len(list) == 0 {
		return nil
	}
	out := make([]XrayObservatoryEntry, 0, len(list))
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := XrayObservatoryEntry{}
		if tag, ok := raw["tag"].(string); ok {
			entry.Tag = types.StringValue(tag)
		}
		if sel, ok := raw["subjectSelector"].([]any); ok {
			entry.SubjectSelector = flattenStringListToType(sel)
		} else {
			// Always store a valid types.List. A zero-value types.List{} cannot be
			// written to state (terraform-plugin-framework raises a Value
			// Conversion Error on State.Set); use an empty list when the panel
			// omits subjectSelector.
			entry.SubjectSelector = types.ListValueMust(types.StringType, nil)
		}
		if v, ok := raw["probeURL"].(string); ok && v != "" {
			entry.ProbeURL = types.StringValue(v)
		}
		if v, ok := raw["probeInterval"].(string); ok && v != "" {
			entry.ProbeInterval = types.StringValue(v)
		}
		if v, ok := raw["enableConcurrency"].(bool); ok {
			entry.EnableConcurrency = types.BoolValue(v)
		}
		out = append(out, entry)
	}
	return out
}

func flattenBurstObservatoryList(list []any) []XrayBurstObservatory {
	if len(list) == 0 {
		return nil
	}
	out := make([]XrayBurstObservatory, 0, len(list))
	for _, item := range list {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := XrayBurstObservatory{}
		if tag, ok := raw["tag"].(string); ok {
			entry.Tag = types.StringValue(tag)
		}
		if sel, ok := raw["subjectSelector"].([]any); ok {
			entry.SubjectSelector = flattenStringListToType(sel)
		} else {
			// Same zero-value guard as the observatory entry flatten path.
			entry.SubjectSelector = types.ListValueMust(types.StringType, nil)
		}
		if pcMap, ok := raw["pingConfig"].(map[string]any); ok && len(pcMap) > 0 {
			entry.PingConfig = flattenBurstPingConfig(pcMap)
		}
		out = append(out, entry)
	}
	return out
}

func flattenBurstPingConfig(in map[string]any) []XrayBurstPingConfig {
	st := XrayBurstPingConfig{}
	if v, ok := in["destination"].(string); ok && v != "" {
		st.Destination = types.StringValue(v)
	}
	if v, ok := in["interval"].(string); ok && v != "" {
		st.Interval = types.StringValue(v)
	}
	if v, ok := in["connectTimeout"].(string); ok && v != "" {
		st.ConnectTimeout = types.StringValue(v)
	}
	if v, ok := in["timeout"].(string); ok && v != "" {
		st.Timeout = types.StringValue(v)
	}
	if v, ok := toFloat64(in["samples"]); ok {
		st.Samples = types.Int64Value(int64(v))
	}
	if v, ok := toFloat64(in["samplingCount"]); ok {
		st.SamplingCount = types.Int64Value(int64(v))
	}
	if v, ok := in["lazy"].(bool); ok {
		st.Lazy = types.BoolValue(v)
	}
	return []XrayBurstPingConfig{st}
}

// ---------------------------------------------------------------------------
// Legacy untyped build / flatten (used by CRUD layer)
// ---------------------------------------------------------------------------

// buildXrayObservatoryJSON converts the expand* output map (snake_case keys)
// into the xray-core wire format. The key difference from other sections:
// observatory entries are keyed by tag in a JSON object, not a list.
func buildXrayObservatoryJSON(d map[string]any) any {
	payload := map[string]any{}

	if v, ok := d["observatory"]; ok {
		if list, ok := v.([]any); ok {
			obj := buildObservatoryObject(list)
			if len(obj) > 0 {
				payload["observatory"] = obj
			}
		}
	}
	if v, ok := d["burst_observatory"]; ok {
		if list, ok := v.([]any); ok {
			obj := buildBurstObservatoryObject(list)
			if len(obj) > 0 {
				payload["burstObservatory"] = obj
			}
		}
	}

	return payload
}

// flatObservatoryTag opts into the singleton format accepted by Xray-core.
const flatObservatoryTag = "default"

// buildObservatoryObject emits the Xray singleton or preserves tagged entries.
func buildObservatoryObject(list []any) map[string]any {
	obj := map[string]any{}
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tag, ok := m["tag"].(string)
		if !ok || tag == "" {
			continue
		}
		entry := map[string]any{}
		if v, ok := m["subjectSelector"].([]any); ok {
			entry["subjectSelector"] = expandStringList(v)
		}
		if v, ok := m["probeURL"].(string); ok && v != "" {
			entry["probeURL"] = v
		}
		if v, ok := m["probeInterval"].(string); ok && v != "" {
			entry["probeInterval"] = v
		}
		if v, ok := m["enableConcurrency"].(bool); ok {
			entry["enableConcurrency"] = v
		}
		if len(entry) > 0 {
			if tag == flatObservatoryTag && len(list) == 1 {
				return entry
			}
			obj[tag] = entry
		}
	}
	return obj
}

func buildBurstObservatoryObject(list []any) map[string]any {
	obj := map[string]any{}
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tag, ok := m["tag"].(string)
		if !ok || tag == "" {
			continue
		}
		entry := map[string]any{}
		if v, ok := m["subjectSelector"].([]any); ok {
			entry["subjectSelector"] = expandStringList(v)
		}
		if v, ok := m["pingConfig"].(map[string]any); ok && len(v) > 0 {
			entry["pingConfig"] = v
		}
		if len(entry) > 0 {
			if tag == flatObservatoryTag && len(list) == 1 {
				return entry
			}
			obj[tag] = entry
		}
	}
	return obj
}

// flattenXrayObservatoryToMap converts the xray-core wire data (JSON object
// keyed by tag) into a flat list of entries suitable for flattenXrayObservatory.
func flattenXrayObservatoryToMap(data any) map[string]any {
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

	if v, ok := payload["observatory"].(map[string]any); ok {
		out["observatory"] = flattenObservatoryObject(v)
	}
	if v, ok := payload["burstObservatory"].(map[string]any); ok {
		out["burst_observatory"] = flattenBurstObservatoryObject(v)
	}

	return out
}

// flattenObservatoryObject converts a wire JSON object keyed by tag into a
// flat list of entry maps with snake_case keys (including a "tag" field).
func flattenObservatoryObject(obj map[string]any) []any {
	for _, field := range []string{"subjectSelector", "probeURL", "probeInterval", "enableConcurrency"} {
		if value, ok := obj[field]; ok {
			if _, isTaggedEntry := value.(map[string]any); !isTaggedEntry {
				return flattenObservatoryObject(map[string]any{flatObservatoryTag: obj})
			}
		}
	}
	out := make([]any, 0, len(obj))
	for tag, raw := range obj {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		entry["tag"] = tag
		if v, ok := m["subjectSelector"].([]any); ok {
			entry["subjectSelector"] = v
		}
		if v, ok := m["probeURL"].(string); ok {
			entry["probeURL"] = v
		}
		if v, ok := m["probeInterval"].(string); ok {
			entry["probeInterval"] = v
		}
		if v, ok := m["enableConcurrency"].(bool); ok {
			entry["enableConcurrency"] = v
		}
		out = append(out, entry)
	}
	return out
}

// flattenBurstObservatoryObject converts a wire JSON object keyed by tag into a
// flat list of entry maps with snake_case keys (including a "tag" field).
// Xray's burstObservatory is a singleton {subjectSelector, pingConfig} object,
// not a tag-keyed map, so a flat object written for the "default" tag is
// mapped back onto that tag on read.
func flattenBurstObservatoryObject(obj map[string]any) []any {
	if v, ok := obj["subjectSelector"]; ok {
		if _, isTaggedEntry := v.(map[string]any); !isTaggedEntry {
			return flattenBurstObservatoryObject(map[string]any{flatObservatoryTag: obj})
		}
	}
	if v, ok := obj["pingConfig"].(map[string]any); ok {
		// A flat singleton without a subject selector is {"pingConfig":
		// {...}}. A tag-keyed entry always contains a subjectSelector and/or
		// pingConfig key, so a value with neither is the singleton's ping
		// config, not an entry.
		if _, hasSelector := v["subjectSelector"]; !hasSelector {
			if _, hasNested := v["pingConfig"]; !hasNested {
				return flattenBurstObservatoryObject(map[string]any{flatObservatoryTag: obj})
			}
		}
	}
	out := make([]any, 0, len(obj))
	for tag, raw := range obj {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		entry["tag"] = tag
		if v, ok := m["subjectSelector"].([]any); ok {
			entry["subjectSelector"] = v
		}
		if v, ok := m["pingConfig"].(map[string]any); ok {
			entry["pingConfig"] = v
		}
		out = append(out, entry)
	}
	return out
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// flattenStringListToType converts a []any of strings into a types.List.
func flattenStringListToType(list []any) types.List {
	vals := make([]attr.Value, 0, len(list))
	for _, s := range list {
		if str, ok := s.(string); ok {
			vals = append(vals, types.StringValue(str))
		}
	}
	return types.ListValueMust(types.StringType, vals)
}
