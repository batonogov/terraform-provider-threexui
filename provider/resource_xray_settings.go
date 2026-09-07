package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	resourcepath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var xrayTemplateMu sync.Mutex

type xraySectionMode int

const (
	xraySectionMergeRoot xraySectionMode = iota
	xraySectionSetPath
)

type xraySection struct {
	id   string
	mode xraySectionMode
	path []string
}

var (
	xraySectionBasics           = xraySection{id: "xray_basics", mode: xraySectionMergeRoot}
	xraySectionDNS              = xraySection{id: "xray_dns", mode: xraySectionSetPath, path: []string{"dns"}}
	xraySectionRouting          = xraySection{id: "xray_routing", mode: xraySectionSetPath, path: []string{"routing"}}
	xraySectionBalancers        = xraySection{id: "xray_balancers", mode: xraySectionSetPath, path: []string{"routing", "balancers"}}
	xraySectionReverse          = xraySection{id: "xray_reverse", mode: xraySectionSetPath, path: []string{"reverse"}}
	xraySectionOutbounds        = xraySection{id: "xray_outbounds", mode: xraySectionSetPath, path: []string{"outbounds"}}
	xraySectionObservatory      = xraySection{id: "xray_observatory", mode: xraySectionSetPath, path: []string{"observatory"}}
	xraySectionBurstObservatory = xraySection{id: "xray_observatory", mode: xraySectionSetPath, path: []string{"burstObservatory"}}
)

type xrayFlattenFunc func(data any) map[string]any

// ---------------------------------------------------------------------------
// Shared typed CRUD helpers
// ---------------------------------------------------------------------------

// xrayApplyTyped applies the desired value to the xray template.
// If desired is empty (empty map or empty/nil slice), the apply is skipped
// to avoid overwriting an existing section with an empty value.
//
// This means a user cannot "clear" a section by supplying an empty block in
// Terraform — the empty value is treated as a no-op. The Delete operation
// only removes the resource from Terraform state; it does not reset the
// corresponding section in the 3x-ui xray config. To clear a section
// manually, edit the xray template directly in the 3x-ui panel.
func xrayApplyTyped(
	ctx context.Context,
	desired any,
	diags *diag.Diagnostics,
	client *Client,
	section xraySection,
) {
	if isEmptyXrayValue(desired) {
		return
	}

	xrayTemplateMu.Lock()
	defer xrayTemplateMu.Unlock()

	current, err := client.GetXrayTemplate(ctx)
	if err != nil {
		diags.AddError("Failed to get xray template", err.Error())
		return
	}

	updated, err := applyXraySection(current, desired, section)
	if err != nil {
		diags.AddError("Failed to apply xray section", err.Error())
		return
	}

	if err := client.UpdateXrayTemplate(ctx, updated); err != nil {
		diags.AddError("Failed to update xray template", err.Error())
		return
	}
}

// xrayReadSection reads the xray template and extracts+flattens the specified section.
func xrayReadSection(
	ctx context.Context,
	diags *diag.Diagnostics,
	client *Client,
	section xraySection,
	flatten xrayFlattenFunc,
) map[string]any {
	current, err := client.GetXrayTemplate(ctx)
	if err != nil {
		diags.AddError("Failed to get xray template", err.Error())
		return nil
	}

	value := extractXraySection(current, section)
	return flatten(value)
}

// ---------------------------------------------------------------------------
// XrayBasicsResource
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &XrayBasicsResource{}
	_ resource.ResourceWithImportState = &XrayBasicsResource{}
	_ resource.ResourceWithModifyPlan  = &XrayBasicsResource{}
)

type XrayBasicsResource struct{ client *Client }

func NewXrayBasicsResource() resource.Resource { return &XrayBasicsResource{} }

func (r *XrayBasicsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_xray_basics"
}

func (r *XrayBasicsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = xrayBasicsSchema()
}

func (r *XrayBasicsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *Client")
		return
	}
	r.client = c
}

func (r *XrayBasicsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan XrayBasicsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := expandXrayBasics(&plan)
	desired := buildXrayBasicsJSON(input)
	xrayApplyTyped(ctx, desired, &resp.Diagnostics, r.client, xraySectionBasics)
	if resp.Diagnostics.HasError() {
		return
	}

	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionBasics, flattenXrayBasicsToMap)
	if flat == nil {
		return
	}
	state := flattenXrayBasics(flat)
	alignBasicsBlocksWithPlan(state, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayBasicsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var prior XrayBasicsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionBasics, flattenXrayBasicsToMap)
	if flat == nil {
		return
	}
	state := flattenXrayBasics(flat)
	// Skip alignment during import (prior state has no blocks set).
	if prior.ID.IsNull() || prior.ID.IsUnknown() {
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
		return
	}
	alignBasicsBlocksWithPlan(state, &prior)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayBasicsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan XrayBasicsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := expandXrayBasics(&plan)
	desired := buildXrayBasicsJSON(input)
	xrayApplyTyped(ctx, desired, &resp.Diagnostics, r.client, xraySectionBasics)
	if resp.Diagnostics.HasError() {
		return
	}

	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionBasics, flattenXrayBasicsToMap)
	if flat == nil {
		return
	}
	state := flattenXrayBasics(flat)
	alignBasicsBlocksWithPlan(state, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// ModifyPlan reconciles configured policy levels by the numeric ID used as the
// remote map key instead of by list index. Explicit config remains authoritative;
// omitted computed leaves come from state for the same ID (or remain unknown for
// a new ID), while configured list order is preserved for Terraform plan validity.
func (r *XrayBasicsResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var configuredPolicy types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, resourcepath.Root("policy"), &configuredPolicy)...)
	// The path and destination type are fixed by this resource's schema;
	// structural null and unknown values are handled by canonicalizeBasicsPolicy.

	priorPolicy := types.ListNull(configuredPolicy.ElementType(ctx))
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, resourcepath.Root("policy"), &priorPolicy)...)
	}

	canonicalPolicy, safe := canonicalizeBasicsPolicy(ctx, configuredPolicy, priorPolicy)
	if !safe {
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, resourcepath.Root("policy"), canonicalPolicy)...)
}

// canonicalizeBasicsPolicy returns a policy list whose configured levels are
// matched to prior state by known numeric ID without changing configured order. It intentionally
// operates on framework values so unknown leaves remain representable and keeps
// omitted non-level policy siblings from prior state. Structural unknowns,
// omitted level collections defer to the proposed plan. Unknown level objects
// and IDs stay in place while known siblings are still reconciled by ID.
func canonicalizeBasicsPolicy(ctx context.Context, policy, priorPolicy types.List) (types.List, bool) {
	if policy.IsNull() || policy.IsUnknown() {
		return policy, false
	}

	priorPoliciesByIndex := make(map[int]types.Object)
	priorLevelsByID := make(map[int64]types.Object)
	if !priorPolicy.IsNull() && !priorPolicy.IsUnknown() {
		for i, priorPolicyElement := range priorPolicy.Elements() {
			priorPolicyObject, ok := priorPolicyElement.(types.Object)
			if !ok || priorPolicyObject.IsNull() || priorPolicyObject.IsUnknown() {
				continue
			}
			priorPoliciesByIndex[i] = priorPolicyObject
			priorLevels, ok := priorPolicyObject.Attributes()["level"].(types.List)
			if !ok || priorLevels.IsNull() || priorLevels.IsUnknown() {
				continue
			}
			for _, priorLevelElement := range priorLevels.Elements() {
				priorLevel, ok := priorLevelElement.(types.Object)
				if !ok || priorLevel.IsNull() || priorLevel.IsUnknown() {
					continue
				}
				id, ok := priorLevel.Attributes()["id"].(types.Int64)
				if ok && !id.IsNull() && !id.IsUnknown() {
					priorLevelsByID[id.ValueInt64()] = priorLevel
				}
			}
		}
	}

	policyElements := policy.Elements()
	canonicalPolicies := make([]attr.Value, len(policyElements))
	for i, element := range policyElements {
		policyObject, ok := element.(types.Object)
		if !ok || policyObject.IsNull() || policyObject.IsUnknown() {
			return policy, false
		}

		policyAttributes := policyObject.Attributes()
		levels, ok := policyAttributes["level"].(types.List)
		if !ok || levels.IsNull() || levels.IsUnknown() {
			return policy, false
		}

		canonicalAttributes := make(map[string]attr.Value, len(policyAttributes))
		for name, value := range policyAttributes {
			canonicalAttributes[name] = value
			if name == "level" || !value.IsNull() {
				continue
			}
			if priorPolicyObject, ok := priorPoliciesByIndex[i]; ok {
				if priorValue, ok := priorPolicyObject.Attributes()[name]; ok {
					canonicalAttributes[name] = priorValue
				}
			}
		}

		levelElements := make([]attr.Value, len(levels.Elements()))
		for j, levelElement := range levels.Elements() {
			levelObject, ok := levelElement.(types.Object)
			if !ok {
				return policy, false
			}
			if levelObject.IsNull() || levelObject.IsUnknown() {
				levelElements[j] = levelElement
				continue
			}
			id, ok := levelObject.Attributes()["id"].(types.Int64)
			if !ok {
				return policy, false
			}
			idKnown := !id.IsNull() && !id.IsUnknown()

			levelAttributes := make(map[string]attr.Value, len(levelObject.Attributes()))
			for name, value := range levelObject.Attributes() {
				levelAttributes[name] = value
				if name == "id" || !value.IsNull() {
					continue
				}
				if idKnown {
					if priorLevel, ok := priorLevelsByID[id.ValueInt64()]; ok {
						if priorValue, ok := priorLevel.Attributes()[name]; ok {
							levelAttributes[name] = priorValue
							continue
						}
					}
				}
				switch value.(type) {
				case types.Int64:
					levelAttributes[name] = types.Int64Unknown()
				case types.Bool:
					levelAttributes[name] = types.BoolUnknown()
				default:
					return policy, false
				}
			}
			canonicalLevel := types.ObjectValueMust(levelObject.AttributeTypes(ctx), levelAttributes)
			levelElements[j] = canonicalLevel
		}

		canonicalLevels := types.ListValueMust(levels.ElementType(ctx), levelElements)
		canonicalAttributes["level"] = canonicalLevels
		canonicalPolicy := types.ObjectValueMust(policyObject.AttributeTypes(ctx), canonicalAttributes)
		canonicalPolicies[i] = canonicalPolicy
	}

	return types.ListValueMust(policy.ElementType(ctx), canonicalPolicies), true
}

func (r *XrayBasicsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *XrayBasicsResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionBasics, flattenXrayBasicsToMap)
	if flat == nil {
		return
	}
	state := flattenXrayBasics(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// ---------------------------------------------------------------------------
// XrayDNSResource
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &XrayDNSResource{}
	_ resource.ResourceWithImportState = &XrayDNSResource{}
	_ resource.ResourceWithModifyPlan  = &XrayDNSResource{}
)

type XrayDNSResource struct{ client *Client }

func NewXrayDNSResource() resource.Resource { return &XrayDNSResource{} }

func (r *XrayDNSResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_xray_dns"
}

func (r *XrayDNSResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = xrayDNSSchema()
}

func (r *XrayDNSResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *Client")
		return
	}
	r.client = c
}

func (r *XrayDNSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan XrayDNSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := expandXrayDNS(&plan)
	desired := buildXrayDNSJSON(input)
	xrayApplyTyped(ctx, desired, &resp.Diagnostics, r.client, xraySectionDNS)
	if resp.Diagnostics.HasError() {
		return
	}

	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionDNS, flattenXrayDNSToMap)
	if flat == nil {
		return
	}
	state := flattenXrayDNS(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayDNSResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionDNS, flattenXrayDNSToMap)
	if flat == nil {
		return
	}
	state := flattenXrayDNS(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayDNSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan XrayDNSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := expandXrayDNS(&plan)
	desired := buildXrayDNSJSON(input)
	xrayApplyTyped(ctx, desired, &resp.Diagnostics, r.client, xraySectionDNS)
	if resp.Diagnostics.HasError() {
		return
	}

	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionDNS, flattenXrayDNSToMap)
	if flat == nil {
		return
	}
	state := flattenXrayDNS(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayDNSResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *XrayDNSResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionDNS, flattenXrayDNSToMap)
	if flat == nil {
		return
	}
	state := flattenXrayDNS(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// ModifyPlan keeps configured DNS server blocks authoritative. Nested
// Optional+Computed attributes use UseStateForUnknown, which otherwise carries
// values from the prior occupant of the same list index when servers are
// reordered. Keep the collection as a framework types.List so unknown object
// elements and partial unknown leaves remain representable while known siblings
// are still reconciled.
func (r *XrayDNSResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var configured types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, resourcepath.Root("server"), &configured)...)
	if resp.Diagnostics.HasError() || configured.IsUnknown() {
		return
	}

	var planned types.List
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, resourcepath.Root("server"), &planned)...)
	if resp.Diagnostics.HasError() || planned.Equal(configured) {
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, resourcepath.Root("server"), configured)...)
}

// ---------------------------------------------------------------------------
// XrayRoutingResource
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &XrayRoutingResource{}
	_ resource.ResourceWithImportState = &XrayRoutingResource{}
	_ resource.ResourceWithModifyPlan  = &XrayRoutingResource{}
)

type XrayRoutingResource struct{ client *Client }

func NewXrayRoutingResource() resource.Resource { return &XrayRoutingResource{} }

func (r *XrayRoutingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_xray_routing"
}

func (r *XrayRoutingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = xrayRoutingSchema()
}

func (r *XrayRoutingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *Client")
		return
	}
	r.client = c
}

func (r *XrayRoutingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan XrayRoutingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ensureNoAPIRoutingRules(plan.Rule, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	input := expandXrayRouting(&plan)
	desired := buildXrayRoutingJSON(input)
	xrayApplyTyped(ctx, desired, &resp.Diagnostics, r.client, xraySectionRouting)
	if resp.Diagnostics.HasError() {
		return
	}

	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionRouting, flattenXrayRoutingToMap)
	if flat == nil {
		return
	}
	state := flattenXrayRouting(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayRoutingResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionRouting, flattenXrayRoutingToMap)
	if flat == nil {
		return
	}
	state := flattenXrayRouting(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayRoutingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan XrayRoutingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ensureNoAPIRoutingRules(plan.Rule, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	input := expandXrayRouting(&plan)
	desired := buildXrayRoutingJSON(input)
	xrayApplyTyped(ctx, desired, &resp.Diagnostics, r.client, xraySectionRouting)
	if resp.Diagnostics.HasError() {
		return
	}

	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionRouting, flattenXrayRoutingToMap)
	if flat == nil {
		return
	}
	state := flattenXrayRouting(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayRoutingResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *XrayRoutingResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionRouting, flattenXrayRoutingToMap)
	if flat == nil {
		return
	}
	state := flattenXrayRouting(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// ModifyPlan keeps the practitioner's configuration authoritative for the
// routing rule list, defeating a stale carry-forward bug in the schema plan
// modifiers.
//
// The nested `rule` block attributes are Optional + Computed with
// stringplanmodifier/listplanmodifier.UseStateForUnknown (added by #228 to
// silence "(known after apply)" drift after import). Because ListNestedBlock
// elements are matched by index, that modifier copies the prior rule's unset
// fields into the new rule occupying the same index whenever rules are
// removed or reordered — bleeding stale matchers across rules. On a reorder
// from [private→direct, RU-domains→direct, catch→proxy] to
// [RU-domains→direct, geoip:cn→direct, catch→proxy] the carried fields put
// `ip:[geoip:private]` on the RU-domains rule and `network:"tcp,udp"` on the
// geoip:cn rule, then those merged rules get written to the panel on apply.
//
// Overriding the planned `rule` list with the configured one removes the
// carry-forward while leaving truly unchanged rules (config == plan)
// untouched. The panel echoes the routing template verbatim on save and the
// provider filters the panel-managed `api` and `xui-dns-allow` rules on read,
// so there are no legitimate computed-only routing fields: configuration is
// always authoritative and this override never drops a real value.
//
// Create (no prior state) and Delete (null plan) have nothing to reconcile.
// When the configured `rule` collection is itself unknown (e.g. a computed
// `dynamic` block whose `for_each` is not yet known), defer to the schema plan
// modifiers — decoding an unknown list into `[]XrayRoutingRule` raises a
// Value Conversion Error (hashicorp/terraform-plugin-framework#1025).
func (r *XrayRoutingResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	// Read `rule` as types.List to tolerate an unknown collection. Decoding
	// into []XrayRoutingRule via req.Config.Get would fail with a Value
	// Conversion Error if the whole list is unknown.
	var ruleAttr types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, resourcepath.Root("rule"), &ruleAttr)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if ruleAttr.IsUnknown() {
		return
	}
	var plan, config XrayRoutingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if reconcileRoutingPlan(&plan, config) {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// reconcileRoutingPlan rewrites the planned routing rules to mirror the
// configuration, returning whether the plan changed. It is the unit-testable
// core of XrayRoutingResource.ModifyPlan: when the plan the schema plan
// modifiers produced carries stale per-index state that configuration does
// not, override it with the configured rule list.
func reconcileRoutingPlan(plan *XrayRoutingModel, config XrayRoutingModel) bool {
	if reflect.DeepEqual(plan.Rule, config.Rule) {
		return false
	}
	plan.Rule = config.Rule
	return true
}

// ---------------------------------------------------------------------------
// XrayBalancersResource
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &XrayBalancersResource{}
	_ resource.ResourceWithImportState = &XrayBalancersResource{}
)

type XrayBalancersResource struct{ client *Client }

func NewXrayBalancersResource() resource.Resource { return &XrayBalancersResource{} }

func (r *XrayBalancersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_xray_balancers"
}

func (r *XrayBalancersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = xrayBalancersSchema()
}

func (r *XrayBalancersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *Client")
		return
	}
	r.client = c
}

func (r *XrayBalancersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan XrayBalancersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := expandXrayBalancers(&plan)
	desired := buildXrayBalancersJSON(input)
	xrayApplyTyped(ctx, desired, &resp.Diagnostics, r.client, xraySectionBalancers)
	if resp.Diagnostics.HasError() {
		return
	}

	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionBalancers, flattenXrayBalancersToMap)
	if flat == nil {
		return
	}
	state := flattenXrayBalancers(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayBalancersResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionBalancers, flattenXrayBalancersToMap)
	if flat == nil {
		return
	}
	state := flattenXrayBalancers(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayBalancersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan XrayBalancersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := expandXrayBalancers(&plan)
	desired := buildXrayBalancersJSON(input)
	xrayApplyTyped(ctx, desired, &resp.Diagnostics, r.client, xraySectionBalancers)
	if resp.Diagnostics.HasError() {
		return
	}

	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionBalancers, flattenXrayBalancersToMap)
	if flat == nil {
		return
	}
	state := flattenXrayBalancers(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayBalancersResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *XrayBalancersResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionBalancers, flattenXrayBalancersToMap)
	if flat == nil {
		return
	}
	state := flattenXrayBalancers(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// ---------------------------------------------------------------------------
// XrayReverseResource
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &XrayReverseResource{}
	_ resource.ResourceWithImportState = &XrayReverseResource{}
)

type XrayReverseResource struct{ client *Client }

func NewXrayReverseResource() resource.Resource { return &XrayReverseResource{} }

func (r *XrayReverseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_xray_reverse"
}

func (r *XrayReverseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = xrayReverseSchema()
}

func (r *XrayReverseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *Client")
		return
	}
	r.client = c
}

func (r *XrayReverseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan XrayReverseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := expandXrayReverse(&plan)
	desired := buildXrayReverseJSON(input)
	xrayApplyTyped(ctx, desired, &resp.Diagnostics, r.client, xraySectionReverse)
	if resp.Diagnostics.HasError() {
		return
	}

	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionReverse, flattenXrayReverseToMap)
	if flat == nil {
		return
	}
	state := flattenXrayReverse(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayReverseResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionReverse, flattenXrayReverseToMap)
	if flat == nil {
		return
	}
	state := flattenXrayReverse(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayReverseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan XrayReverseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := expandXrayReverse(&plan)
	desired := buildXrayReverseJSON(input)
	xrayApplyTyped(ctx, desired, &resp.Diagnostics, r.client, xraySectionReverse)
	if resp.Diagnostics.HasError() {
		return
	}

	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionReverse, flattenXrayReverseToMap)
	if flat == nil {
		return
	}
	state := flattenXrayReverse(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayReverseResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *XrayReverseResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionReverse, flattenXrayReverseToMap)
	if flat == nil {
		return
	}
	state := flattenXrayReverse(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// ---------------------------------------------------------------------------
// XrayOutboundsResource
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &XrayOutboundsResource{}
	_ resource.ResourceWithImportState = &XrayOutboundsResource{}
	_ resource.ResourceWithModifyPlan  = &XrayOutboundsResource{}
)

type XrayOutboundsResource struct{ client *Client }

func NewXrayOutboundsResource() resource.Resource { return &XrayOutboundsResource{} }

func (r *XrayOutboundsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_xray_outbounds"
}

func (r *XrayOutboundsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = xrayOutboundsSchema()
}

func (r *XrayOutboundsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *Client")
		return
	}
	r.client = c
}

func (r *XrayOutboundsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan XrayOutboundsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := expandXrayOutbounds(&plan)
	desired := buildXrayOutboundsJSON(input)
	xrayApplyTyped(ctx, desired, &resp.Diagnostics, r.client, xraySectionOutbounds)
	if resp.Diagnostics.HasError() {
		return
	}

	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionOutbounds, flattenXrayOutboundsToMap)
	if flat == nil {
		return
	}
	state := flattenXrayOutbounds(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayOutboundsResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionOutbounds, flattenXrayOutboundsToMap)
	if flat == nil {
		return
	}
	state := flattenXrayOutbounds(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayOutboundsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan XrayOutboundsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := expandXrayOutbounds(&plan)
	desired := buildXrayOutboundsJSON(input)
	xrayApplyTyped(ctx, desired, &resp.Diagnostics, r.client, xraySectionOutbounds)
	if resp.Diagnostics.HasError() {
		return
	}

	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionOutbounds, flattenXrayOutboundsToMap)
	if flat == nil {
		return
	}
	state := flattenXrayOutbounds(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayOutboundsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *XrayOutboundsResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	flat := xrayReadSection(ctx, &resp.Diagnostics, r.client, xraySectionOutbounds, flattenXrayOutboundsToMap)
	if flat == nil {
		return
	}
	state := flattenXrayOutbounds(flat)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// ModifyPlan keeps each configured outbound object authoritative during an
// update. outbound is a ListNestedBlock, and every optional top-level field and
// every optional descendant of mux/protocol-specific blocks is also Computed
// with UseStateForUnknown. Terraform correlates those descendants by list index,
// so a reorder can otherwise copy values from the old occupant into a different
// outbound and Update will persist the polluted object.
//
// Work with the framework list value rather than []XrayOutboundEntry. Besides
// preserving partially unknown leaves, this safely represents both a wholly
// unknown outbound collection and known collections containing unknown object
// elements. A wholly unknown collection is left to the schema plan modifiers;
// otherwise the complete configured list replaces any index-polluted plan.
//
// stream_settings is Optional+Computed with UseStateForUnknown, but a
// practitioner who never declared it has no config value to plan from — the
// schema modifier restores the prior state object into the plan, and the
// wholesale configured-list reset below must not drop it. The configured
// elements are therefore re-merged with the prior state's stream_settings
// (matched by tag) whenever the configuration omits the block.
func (r *XrayOutboundsResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var configured types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, resourcepath.Root("outbound"), &configured)...)
	if resp.Diagnostics.HasError() || configured.IsUnknown() {
		return
	}

	var planned types.List
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, resourcepath.Root("outbound"), &planned)...)
	if resp.Diagnostics.HasError() || planned.Equal(configured) {
		return
	}

	var stateOutbound types.List
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, resourcepath.Root("outbound"), &stateOutbound)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if merged := mergeOutboundStreamSettingsFromState(configured, stateOutbound); !merged.Equal(configured) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, resourcepath.Root("outbound"), merged)...)
		return
	}

	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, resourcepath.Root("outbound"), configured)...)
}

// mergeOutboundStreamSettingsFromState copies the prior state's stream_settings
// object onto each configured outbound element (matched by tag) whose
// configuration omits the block. Without it, the "configured list is
// authoritative" reset above would strip a server-populated stream_settings —
// e.g. the TLS settings that make a vless outbound reachable — on every
// unrelated update, because the schema's UseStateForUnknown plan modifier can
// only restore the object into the plan, not into a configuration that declared
// nothing. Tag identifies an outbound across list reorders.
func mergeOutboundStreamSettingsFromState(configured, stateList types.List) types.List {
	if configured.IsNull() || configured.IsUnknown() || len(configured.Elements()) == 0 {
		return configured
	}
	if stateList.IsNull() || stateList.IsUnknown() || len(stateList.Elements()) == 0 {
		return configured
	}

	stateByTag := map[string]attr.Value{}
	for _, e := range stateList.Elements() {
		obj, ok := e.(types.Object)
		if !ok {
			continue
		}
		tag, ok := obj.Attributes()["tag"].(types.String)
		if !ok || tag.IsNull() || tag.IsUnknown() {
			continue
		}
		if ss, ok := obj.Attributes()["stream_settings"]; ok && !ss.IsNull() && !ss.IsUnknown() {
			stateByTag[tag.ValueString()] = ss
		}
	}
	if len(stateByTag) == 0 {
		return configured
	}

	ctx := context.Background()
	elems := make([]attr.Value, 0, len(configured.Elements()))
	changed := false
	for _, e := range configured.Elements() {
		obj, ok := e.(types.Object)
		if !ok {
			elems = append(elems, e)
			continue
		}
		attrs := obj.Attributes()
		tag, ok := attrs["tag"].(types.String)
		if !ok || tag.IsNull() || tag.IsUnknown() {
			elems = append(elems, e)
			continue
		}
		if ss, ok := attrs["stream_settings"]; ok && !ss.IsNull() && !ss.IsUnknown() {
			elems = append(elems, e)
			continue
		}
		saved, ok := stateByTag[tag.ValueString()]
		if !ok {
			elems = append(elems, e)
			continue
		}
		merged := make(map[string]attr.Value, len(attrs)+1)
		for k, v := range attrs {
			merged[k] = v
		}
		merged["stream_settings"] = saved
		elems = append(elems, types.ObjectValueMust(obj.AttributeTypes(ctx), merged))
		changed = true
	}
	if !changed {
		return configured
	}
	return types.ListValueMust(configured.ElementType(ctx), elems)
}

// ---------------------------------------------------------------------------
// XrayObservatoryResource
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &XrayObservatoryResource{}
	_ resource.ResourceWithImportState = &XrayObservatoryResource{}
)

type XrayObservatoryResource struct{ client *Client }

func NewXrayObservatoryResource() resource.Resource { return &XrayObservatoryResource{} }

func (r *XrayObservatoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_xray_observatory"
}

func (r *XrayObservatoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = xrayObservatorySchema()
}

func (r *XrayObservatoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *Client")
		return
	}
	r.client = c
}

// xrayObservatoryApply writes both the "observatory" and "burstObservatory"
// top-level keys. Unlike other xray resources that use a single xraySection
// path, this resource manages two independent top-level JSON keys. It performs
// two mutex-protected read-modify-write cycles (one per key), keeping the
// existing xrayTemplateMu serialization.
func (r *XrayObservatoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan XrayObservatoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applyObservatory(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	state := r.readObservatory(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || state == nil {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayObservatoryResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	state := r.readObservatory(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || state == nil {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayObservatoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan XrayObservatoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applyObservatory(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	state := r.readObservatory(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || state == nil {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *XrayObservatoryResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *XrayObservatoryResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	state := r.readObservatory(ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || state == nil {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// applyObservatory writes the observatory and burstObservatory sections.
// Each is applied as a set-path operation on the top-level JSON key.
//
// xrayApplyTyped stores its `desired` argument verbatim at the section path via
// setJSONPath(root, section.path, desired). For set-path sections the desired
// value must therefore be the section CONTENT (e.g. the tag-keyed object), not a
// map re-wrapped with the key — wrapping it again would double-nest the value
// (root["observatory"] = {"observatory": {...}}), which the read-back path then
// cannot decode, leaving subject_selector as an invalid zero-value types.List
// and raising a framework Value Conversion Error.
func (r *XrayObservatoryResource) applyObservatory(ctx context.Context, plan *XrayObservatoryModel, diags *diag.Diagnostics) {
	input := expandXrayObservatory(plan)
	desired := buildXrayObservatoryJSON(input).(map[string]any)

	// Apply observatory key. `obs` is already the tag-keyed object to store at
	// path ["observatory"], so pass it directly.
	if obs, ok := desired["observatory"]; ok && !isEmptyXrayValue(obs) {
		xrayApplyTyped(ctx, obs, diags, r.client, xraySectionObservatory)
	} else {
		// If user removed all observatories, clear the key
		r.clearObservatoryKey(ctx, "observatory", diags)
	}

	if diags.HasError() {
		return
	}

	// Apply burstObservatory key. `burst` is the tag-keyed object for path
	// ["burstObservatory"], passed directly (same rationale as above).
	if burst, ok := desired["burstObservatory"]; ok && !isEmptyXrayValue(burst) {
		xrayApplyTyped(ctx, burst, diags, r.client, xraySectionBurstObservatory)
	} else {
		r.clearObservatoryKey(ctx, "burstObservatory", diags)
	}
}

// clearObservatoryKey removes a top-level key from the xray template.
func (r *XrayObservatoryResource) clearObservatoryKey(ctx context.Context, key string, diags *diag.Diagnostics) {
	xrayTemplateMu.Lock()
	defer xrayTemplateMu.Unlock()

	current, err := r.client.GetXrayTemplate(ctx)
	if err != nil {
		diags.AddError("Failed to get xray template", err.Error())
		return
	}

	if _, exists := current[key]; exists {
		delete(current, key)
		if err := r.client.UpdateXrayTemplate(ctx, current); err != nil {
			diags.AddError("Failed to update xray template", err.Error())
		}
	}
}

// readObservatory reads both observatory and burstObservatory from the template.
func (r *XrayObservatoryResource) readObservatory(ctx context.Context, diags *diag.Diagnostics) *XrayObservatoryModel {
	current, err := r.client.GetXrayTemplate(ctx)
	if err != nil {
		diags.AddError("Failed to get xray template", err.Error())
		return nil
	}

	// Build a combined payload for flattenXrayObservatoryToMap
	payload := map[string]any{}
	if v, ok := current["observatory"]; ok {
		payload["observatory"] = v
	}
	if v, ok := current["burstObservatory"]; ok {
		payload["burstObservatory"] = v
	}

	flat := flattenXrayObservatoryToMap(payload)
	return flattenXrayObservatory(flat)
}

// ---------------------------------------------------------------------------
// JSON helpers (no SDK dependency)
// ---------------------------------------------------------------------------

func deepEqualJSON(a, b any) bool {
	ab, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(ab, bb)
}

func applyXraySection(current map[string]any, desired any, section xraySection) (map[string]any, error) {
	root := cloneJSONMap(current)
	switch section.mode {
	case xraySectionMergeRoot:
		desiredMap, ok := desired.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("value must be an object for %s", section.id)
		}
		root = deepMergeJSON(root, desiredMap)
	case xraySectionSetPath:
		if len(section.path) == 0 {
			return nil, fmt.Errorf("invalid section path for %s", section.id)
		}
		setJSONPath(root, section.path, desired)
	default:
		return nil, fmt.Errorf("unknown section mode for %s", section.id)
	}
	return root, nil
}

func extractXraySection(current map[string]any, section xraySection) any {
	switch section.mode {
	case xraySectionMergeRoot:
		out := map[string]any{}
		for _, key := range []string{"log", "policy", "api", "stats", "env"} {
			if v, ok := current[key]; ok {
				out[key] = v
			}
		}
		return out
	case xraySectionSetPath:
		return getJSONPath(current, section.path)
	default:
		return nil
	}
}

func cloneJSONMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func deepMergeJSON(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = map[string]any{}
	}
	for k, v := range src {
		if vMap, ok := v.(map[string]any); ok {
			if existing, ok := dst[k].(map[string]any); ok {
				dst[k] = deepMergeJSON(existing, vMap)
				continue
			}
		}
		dst[k] = v
	}
	return dst
}

func setJSONPath(root map[string]any, path []string, value any) {
	current := root
	for i := 0; i < len(path)-1; i++ {
		key := path[i]
		next, ok := current[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[key] = next
		}
		current = next
	}
	current[path[len(path)-1]] = value
}

func getJSONPath(root map[string]any, path []string) any {
	current := any(root)
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = m[key]
		if !ok {
			return nil
		}
	}
	return current
}

// isEmptyXrayValue returns true if v is nil, an empty map, or an empty/nil slice.
func isEmptyXrayValue(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case map[string]any:
		return len(val) == 0
	case []any:
		return len(val) == 0
	}
	return false
}
