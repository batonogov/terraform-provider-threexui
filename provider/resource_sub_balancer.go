package provider

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                     = &SubBalancerResource{}
	_ resource.ResourceWithConfigure        = &SubBalancerResource{}
	_ resource.ResourceWithImportState      = &SubBalancerResource{}
	_ resource.ResourceWithConfigValidators = &SubBalancerResource{}
)

const (
	subBalancerStrategyLeastLoad = "leastLoad"
	subBalancerDefaultStrategy   = "random"
	subBalancerRemarkMaxBytes    = 256
)

var (
	subBalancerStrategies = []string{subBalancerStrategyLeastLoad, "leastPing", subBalancerDefaultStrategy, "roundRobin"}

	// The panel decodes keys into map[int]float64 and answers with the
	// canonical spelling, so "05" or "+5" would come back as "5".
	subBalancerWeightKeyRe = regexp.MustCompile(`^[1-9]\d*$`)
)

// SubBalancer mirrors 3x-ui's model.SubBalancer (internal/database/model/model.go,
// v3.7.0+; memberWeights since v3.8.0). createdAt/updatedAt are not modeled.
type SubBalancer struct {
	Id            int64             `json:"id"`
	Remark        string            `json:"remark"`
	Strategy      string            `json:"strategy"`
	InboundIds    []int64           `json:"inboundIds"`
	MemberWeights map[int64]float64 `json:"memberWeights,omitempty"`
	SortOrder     int64             `json:"sortOrder"`
	Enabled       bool              `json:"enabled"`
}

type SubBalancerResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Remark        types.String `tfsdk:"remark"`
	Strategy      types.String `tfsdk:"strategy"`
	InboundIDs    types.Set    `tfsdk:"inbound_ids"`    // set of int64
	MemberWeights types.Map    `tfsdk:"member_weights"` // map of float64, keyed by inbound id
	SortOrder     types.Int64  `tfsdk:"sort_order"`
	Enabled       types.Bool   `tfsdk:"enabled"`
}

type SubBalancerResource struct {
	client *Client
}

func NewSubBalancerResource() resource.Resource {
	return &SubBalancerResource{}
}

func (r *SubBalancerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sub_balancer"
}

func (r *SubBalancerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = subBalancerResourceSchema()
}

func (r *SubBalancerResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{subBalancerWeightsValidator{}}
}

func (r *SubBalancerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *Client")
		return
	}
	r.client = client
}

func (r *SubBalancerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SubBalancerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in, diags := subBalancerFromModel(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateSubBalancer(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create sub balancer", err.Error())
		return
	}

	if subBalancerWeightsDropped(in, created) {
		// Roll back so the failed create leaves no untracked balancer behind.
		if delErr := r.client.DeleteSubBalancer(ctx, created.Id); delErr != nil && !isSubBalancerNotFound(delErr) {
			flattenSubBalancerToModel(created, &plan)
			resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
			resp.Diagnostics.AddAttributeError(path.Root("member_weights"), subBalancerWeightsUnsupportedSummary,
				subBalancerWeightsUnsupportedDetail+fmt.Sprintf(" Deleting the balancer without its weights (id %d) failed, so it stays in state as tainted: %s", created.Id, delErr))
			return
		}
		resp.Diagnostics.AddAttributeError(path.Root("member_weights"), subBalancerWeightsUnsupportedSummary,
			subBalancerWeightsUnsupportedDetail+" The balancer was deleted again, nothing was created.")
		return
	}

	flattenSubBalancerToModel(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SubBalancerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SubBalancerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := parseSubBalancerID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid sub balancer id", err.Error())
		return
	}

	got, err := r.client.GetSubBalancer(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read sub balancer", err.Error())
		return
	}
	if got == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	flattenSubBalancerToModel(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SubBalancerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SubBalancerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := parseSubBalancerID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid sub balancer id", err.Error())
		return
	}
	in, diags := subBalancerFromModel(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateSubBalancer(ctx, id, in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update sub balancer", err.Error())
		return
	}

	// State must reflect the panel even on error; the planned weights were not stored.
	flattenSubBalancerToModel(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if subBalancerWeightsDropped(in, updated) {
		resp.Diagnostics.AddAttributeError(path.Root("member_weights"), subBalancerWeightsUnsupportedSummary,
			subBalancerWeightsUnsupportedDetail+" The other changes were applied.")
	}
}

func (r *SubBalancerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SubBalancerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := parseSubBalancerID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid sub balancer id", err.Error())
		return
	}

	if err := r.client.DeleteSubBalancer(ctx, id); err != nil && !isSubBalancerNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete sub balancer", err.Error())
	}
}

func (r *SubBalancerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := parseSubBalancerID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

const (
	subBalancerWeightsUnsupportedSummary = "member_weights requires 3x-ui v3.8.0 or later"
	subBalancerWeightsUnsupportedDetail  = "The panel saved the balancer but discarded member_weights: the field was added in 3x-ui v3.8.0. " +
		"Remove member_weights or upgrade the panel."
)

// subBalancerWeightsDropped detects a v3.7.x panel, whose controller ignores
// the memberWeights form key: the write succeeds and the weights are gone.
// Stray keys cannot explain it: the validators reject keys outside inbound_ids.
func subBalancerWeightsDropped(sent, got *SubBalancer) bool {
	return len(sent.MemberWeights) > 0 && len(got.MemberWeights) == 0
}

func parseSubBalancerID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("expected the numeric id of a sub balancer, got %q", raw)
	}
	return id, nil
}

// ---------------------------------------------------------------------------
// Expand / Flatten
// ---------------------------------------------------------------------------

// subBalancerFromModel converts planned values. They have passed the schema
// validators: Terraform validates the configuration again during apply, once
// values that were unknown at plan time (e.g. new inbound ids) are known.
func subBalancerFromModel(ctx context.Context, m *SubBalancerResourceModel) (*SubBalancer, diag.Diagnostics) {
	var diags diag.Diagnostics
	b := &SubBalancer{
		Remark:    m.Remark.ValueString(),
		Strategy:  m.Strategy.ValueString(),
		SortOrder: m.SortOrder.ValueInt64(),
		Enabled:   m.Enabled.ValueBool(),
	}
	diags.Append(m.InboundIDs.ElementsAs(ctx, &b.InboundIds, false)...)
	slices.Sort(b.InboundIds)

	if !m.MemberWeights.IsNull() {
		elements := m.MemberWeights.Elements()
		b.MemberWeights = make(map[int64]float64, len(elements))
		for key, element := range elements {
			id, ok := parseSubBalancerWeightKey(key)
			if !ok {
				diags.AddAttributeError(path.Root("member_weights").AtMapKey(key), "Invalid member_weights key",
					fmt.Sprintf("%q is not an inbound id: keys must be positive integers without sign or leading zeros.", key))
				continue
			}
			// ElementsAs would reject a value with more digits than a float64
			// holds (e.g. 1/3 in HCL); the panel stores a float64 anyway.
			weight, _ := element.(types.Float64)
			b.MemberWeights[id] = weight.ValueFloat64()
		}
	}
	return b, diags
}

func parseSubBalancerWeightKey(key string) (int64, bool) {
	if !subBalancerWeightKeyRe.MatchString(key) {
		return 0, false
	}
	id, err := strconv.ParseInt(key, 10, 64)
	return id, err == nil
}

func subBalancerRemarkProblem(remark string) string {
	switch {
	case strings.TrimSpace(remark) == "":
		return "remark must not be empty"
	case strings.TrimSpace(remark) != remark:
		return "remark must not start or end with whitespace: the panel trims it, so the stored value would never match the configuration"
	case len(remark) > subBalancerRemarkMaxBytes:
		return fmt.Sprintf("remark must be at most %d bytes, got %d", subBalancerRemarkMaxBytes, len(remark))
	}
	return ""
}

// flattenSubBalancerToModel copies a SubBalancer into the model. Managed
// attributes are overwritten from the remote so drift is detected.
func flattenSubBalancerToModel(b *SubBalancer, m *SubBalancerResourceModel) {
	m.ID = types.StringValue(strconv.FormatInt(b.Id, 10))
	m.Remark = types.StringValue(b.Remark)
	m.Strategy = types.StringValue(b.Strategy)
	m.SortOrder = types.Int64Value(b.SortOrder)
	m.Enabled = types.BoolValue(b.Enabled)

	// The API stores repeated inboundIds as sent; a set must not repeat them.
	ids := make([]attr.Value, 0, len(b.InboundIds))
	seen := make(map[int64]bool, len(b.InboundIds))
	for _, id := range b.InboundIds {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, types.Int64Value(id))
		}
	}
	m.InboundIDs = types.SetValueMust(types.Int64Type, ids)

	// Weights of inbounds deleted on the panel stay: DelInbound only edits ids.
	m.MemberWeights = types.MapNull(types.Float64Type)
	if len(b.MemberWeights) > 0 {
		weights := make(map[string]attr.Value, len(b.MemberWeights))
		for id, weight := range b.MemberWeights {
			weights[strconv.FormatInt(id, 10)] = types.Float64Value(weight)
		}
		m.MemberWeights = types.MapValueMust(types.Float64Type, weights)
	}
}

// ---------------------------------------------------------------------------
// Validators
// ---------------------------------------------------------------------------

type subBalancerRemarkValidator struct{}

func (subBalancerRemarkValidator) Description(_ context.Context) string {
	return fmt.Sprintf("remark must be non-empty, at most %d bytes and must not start or end with whitespace", subBalancerRemarkMaxBytes)
}

func (v subBalancerRemarkValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (subBalancerRemarkValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if problem := subBalancerRemarkProblem(req.ConfigValue.ValueString()); problem != "" {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid remark", problem)
	}
}

// subBalancerWeightsValidator applies the panel's cross-attribute weight rules.
// Checks skipped for unknown values run when Terraform validates again at apply.
type subBalancerWeightsValidator struct{}

func (subBalancerWeightsValidator) Description(_ context.Context) string {
	return "member_weights requires strategy = \"leastLoad\" and keys taken from inbound_ids"
}

func (v subBalancerWeightsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (subBalancerWeightsValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config SubBalancerResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.MemberWeights.IsNull() || config.MemberWeights.IsUnknown() {
		return
	}

	strategy := subBalancerDefaultStrategy
	if !config.Strategy.IsNull() {
		strategy = config.Strategy.ValueString()
	}
	if !config.Strategy.IsUnknown() && strategy != subBalancerStrategyLeastLoad {
		resp.Diagnostics.AddAttributeError(path.Root("member_weights"), "member_weights requires strategy = \"leastLoad\"",
			fmt.Sprintf("The panel rejects weights for strategy %q: xray applies balancer costs only to leastLoad. "+
				"Remove member_weights or set strategy = \"leastLoad\".", strategy))
	}

	if config.InboundIDs.IsNull() || config.InboundIDs.IsUnknown() {
		return
	}
	ids := make(map[int64]bool, len(config.InboundIDs.Elements()))
	for _, element := range config.InboundIDs.Elements() {
		id, ok := element.(types.Int64)
		if !ok || id.IsUnknown() {
			return
		}
		ids[id.ValueInt64()] = true
	}
	keys := make([]string, 0, len(config.MemberWeights.Elements()))
	for key := range config.MemberWeights.Elements() {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		// A malformed key is reported by the attribute's own validator; one
		// beyond int64 matches the pattern but cannot be an inbound id.
		if !subBalancerWeightKeyRe.MatchString(key) {
			continue
		}
		if id, ok := parseSubBalancerWeightKey(key); !ok || !ids[id] {
			resp.Diagnostics.AddAttributeError(path.Root("member_weights").AtMapKey(key), "member_weights key is not in inbound_ids",
				fmt.Sprintf("Inbound %s is not a member of this balancer, and the panel silently drops weights for non-members. "+
					"Add it to inbound_ids or remove the weight.", key))
		}
	}
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

func subBalancerResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages a client-side balancer of the 3x-ui JSON subscription (Settings → Subscription → Balancers, " +
			"/panel/api/sub-balancers). Each enabled balancer adds one profile to the JSON subscription whose outbounds are the " +
			"selected inbounds' proxies, balanced on the client by routing.balancers. Requires 3x-ui v3.7.0+; member_weights " +
			"requires v3.8.0+. Server-side Xray balancers are managed by threexui_xray_balancers instead.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Numeric ID of the balancer.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"remark": schema.StringAttribute{
				Required: true,
				Description: "Name of the profile shown to subscribers. Must be non-empty, at most 256 bytes and must not start or " +
					"end with whitespace (the panel trims it).",
				Validators: []validator.String{subBalancerRemarkValidator{}},
			},
			"strategy": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(subBalancerDefaultStrategy),
				Description: "Balancing strategy: `leastLoad`, `leastPing`, `random` or `roundRobin`. Defaults to `random`. " +
					"`leastLoad` and `leastPing` add a burst observatory to the profile, configured panel-wide by " +
					"`threexui_panel_subscription.sub_json_observatory`.",
				Validators: []validator.String{stringvalidator.OneOf(subBalancerStrategies...)},
			},
			"inbound_ids": schema.SetAttribute{
				Required:    true,
				ElementType: types.Int64Type,
				Description: "IDs of the inbounds whose proxy outbounds become balancer members. A set: members follow the " +
					"subscription order of their inbounds (`sub_sort_index`, then id), not the order written here. Deleting an " +
					"inbound removes its id from every balancer on the panel.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.NoNullValues(),
					setvalidator.ValueInt64sAre(int64validator.AtLeast(1)),
				},
			},
			"member_weights": schema.MapAttribute{
				Optional:    true,
				ElementType: types.Float64Type,
				Description: "leastLoad cost per member, keyed by inbound id. A lower weight is picked more often; members " +
					"without a weight count as 1. Requires strategy = \"leastLoad\"; every key must be one of inbound_ids and " +
					"every value a positive float32. Omitting the attribute clears the weights. Requires 3x-ui v3.8.0+.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
					mapvalidator.NoNullValues(),
					mapvalidator.KeysAre(stringvalidator.RegexMatches(subBalancerWeightKeyRe,
						"must be an inbound id: a positive integer without sign or leading zeros")),
					mapvalidator.ValueFloat64sAre(float64validator.Between(math.SmallestNonzeroFloat32, math.MaxFloat32)),
				},
			},
			"sort_order": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(1),
				Description: "Position of the profile in the subscription, interleaved with the inbounds' `sub_sort_index`; on " +
					"a tie the balancer comes after the inbound. Defaults to 1; must be at least 1.",
				Validators: []validator.Int64{int64validator.AtLeast(1)},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the profile is added to the subscription. Defaults to true.",
			},
		},
	}
}
