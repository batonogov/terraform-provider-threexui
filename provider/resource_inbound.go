package provider

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &InboundResource{}
	_ resource.ResourceWithImportState = &InboundResource{}
	_ resource.ResourceWithModifyPlan  = &InboundResource{}
)

type InboundResource struct {
	client *Client
}

type InboundResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Up                   types.Int64  `tfsdk:"up"`
	Down                 types.Int64  `tfsdk:"down"`
	Total                types.Int64  `tfsdk:"total"`
	AllTime              types.Int64  `tfsdk:"all_time"`
	Remark               types.String `tfsdk:"remark"`
	Enable               types.Bool   `tfsdk:"enable"`
	ExpiryTime           types.Int64  `tfsdk:"expiry_time"`
	TrafficReset         types.String `tfsdk:"traffic_reset"`
	TrafficResetDay      types.Int64  `tfsdk:"traffic_reset_day"`
	LastTrafficResetTime types.Int64  `tfsdk:"last_traffic_reset_time"`
	Listen               types.String `tfsdk:"listen"`
	Port                 types.Int64  `tfsdk:"port"`
	Protocol             types.String `tfsdk:"protocol"`
	Tag                  types.String `tfsdk:"tag"`
	NodeID               types.Int64  `tfsdk:"node_id"`
	SubSortIndex         types.Int64  `tfsdk:"sub_sort_index"`
	ShareAddr            types.String `tfsdk:"share_addr"`
	ShareAddrStrategy    types.String `tfsdk:"share_addr_strategy"`
	DisableFlow          types.Bool   `tfsdk:"disable_flow"`
	RestartXray          types.Bool   `tfsdk:"restart_xray"`

	// Per-protocol settings (typed blocks)
	VlessSettings       *InboundVlessSettingsModel       `tfsdk:"vless_settings"`
	TrojanSettings      *InboundTrojanSettingsModel      `tfsdk:"trojan_settings"`
	ShadowsocksSettings *InboundShadowsocksSettingsModel `tfsdk:"shadowsocks_settings"`
	HTTPSettings        *InboundHTTPSettingsModel        `tfsdk:"http_settings"`
	SocksSettings       *InboundSocksSettingsModel       `tfsdk:"socks_settings"`
	MixedSettings       *InboundMixedSettingsModel       `tfsdk:"mixed_settings"`
	WireguardSettings   *InboundWireguardSettingsModel   `tfsdk:"wireguard_settings"`
	AmneziawgSettings   *InboundAmneziawgSettingsModel   `tfsdk:"amneziawg_settings"`
	TuicSettings        *InboundTuicSettingsModel        `tfsdk:"tuic_settings"`
	DokodemoSettings    *InboundDokodemoSettingsModel    `tfsdk:"dokodemo_settings"`
	HysteriaSettings    *InboundHysteriaSettingsModel    `tfsdk:"hysteria_settings"`
	MtprotoSettings     *InboundMtprotoSettingsModel     `tfsdk:"mtproto_settings"`

	// Stream settings (typed block)
	StreamSettings *InboundStreamSettingsModel `tfsdk:"stream_settings"`

	// Sniffing (typed block)
	Sniffing *InboundSniffingModel `tfsdk:"sniffing"`
}

func NewInboundResource() resource.Resource {
	return &InboundResource{}
}

func (r *InboundResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inbound"
}

func (r *InboundResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a 3x-ui inbound.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Numeric ID of the inbound.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"up": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Upload traffic (bytes).",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"down": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Download traffic (bytes).",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"total": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "Total traffic limit (bytes). 0 means unlimited.",
			},
			"all_time": schema.Int64Attribute{
				Computed: true,
				Description: "Deprecated: always `0` on every supported panel. 3x-ui carried an `allTime` field from " +
					"v2.6.7 until v3.1.0 removed it (upstream PR #4469, which also drops the `all_time` columns on " +
					"startup). No version this provider supports (v3.2.x+) sends it, so the attribute has no value to " +
					"report. Use `up`, `down`, or the `threexui_client_traffics` data source.",
				DeprecationMessage: "all_time is always 0: 3x-ui removed the allTime field in v3.1.0 (upstream PR #4469), " +
					"and no supported panel version (v3.2.x+) sends it. Use up, down, or the threexui_client_traffics " +
					"data source. The attribute will be removed in the next major release.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"remark": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "Remark / display name for the inbound.",
			},
			"enable": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the inbound is enabled.",
			},
			"expiry_time": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Expiry time in milliseconds since epoch. 0 means never.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"traffic_reset": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("never"),
				Description: "Traffic reset interval (e.g. 'never', 'hourly', 'daily', 'weekly', 'monthly').",
				Validators:  trafficResetValidators(),
			},
			"traffic_reset_day": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Description: "Day of month (1-31) for monthly traffic resets. Only effective when traffic_reset = 'monthly'. " +
					"3x-ui v3.6.0+; older panels report 0 (unsupported). Cannot be set to 0: the panel clamps any value " +
					"below 1 up to 1 (normalizeTrafficResetDay), so a configured 0 could never round-trip.",
				Validators: []validator.Int64{
					int64validator.Between(1, 31),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"last_traffic_reset_time": schema.Int64Attribute{
				Computed:    true,
				Description: "Last traffic reset time in milliseconds since epoch.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"disable_flow": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Opt this inbound out of the panel's automatic XTLS Vision flow assignment. 3x-ui v3.7.0+; older panels report false (unsupported).",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"listen": schema.StringAttribute{
				Optional:    true,
				Description: "Listen address.",
			},
			"port": schema.Int64Attribute{
				Required:    true,
				Description: "Listen port.",
				Validators:  portValidators(),
			},
			"protocol": schema.StringAttribute{
				Required:    true,
				Description: "Protocol (vless, vmess, trojan, shadowsocks, http, mixed, wireguard, amneziawg, tunnel, tun, hysteria, mtproto). socks and dokodemo-door are deprecated since 3x-ui v3.2.0 — use mixed and tunnel instead. tun is an alias for tunnel available since 3x-ui v3.2.7; mtproto is available since v3.3.0; amneziawg since v3.7.0.",
				Validators:  protocolValidators(),
			},
			"tag": schema.StringAttribute{
				Computed:    true,
				Description: "Xray inbound tag (auto-generated by the panel).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"node_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "3x-ui v3 node ID for multi-node deployments. Null means the inbound runs on the local panel.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"sub_sort_index": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description:   "1-based sort order of this inbound's links in subscription output (lower first; ties by id). Added in 3x-ui v3.3.1; ignored by older panels.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"share_addr": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Share address used in generated subscription links when share_addr_strategy is custom. Added in 3x-ui v3.3.1; ignored by older panels.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"share_addr_strategy": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Strategy for the share address in subscription links: node (inbound listen/node address), listen, or custom (uses share_addr). Added in 3x-ui v3.3.1; ignored by older panels.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"restart_xray": schema.BoolAttribute{
				Optional:    true,
				Description: "Restart Xray core after create, update, or delete operations. Default is false.",
			},
		},
		Blocks: func() map[string]schema.Block {
			blocks := inboundSettingsBlockSchemas()
			blocks["stream_settings"] = inboundStreamSettingsBlockSchema()
			blocks["sniffing"] = inboundSniffingBlockSchema()
			return blocks
		}(),
	}
}

// ConfigValidators enforces cross-attribute rules the schema alone cannot
// express — currently only the AmneziaWG server block, whose absence would let
// the panel rotate the server keypair on an unrelated update (#441).
func (r *InboundResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		amneziawgServerRequiredValidator{},
	}
}

func (r *InboundResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type",
			fmt.Sprintf("Expected *Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *InboundResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InboundResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	inbound := expandInboundFromModel(&plan)

	settingsJSON, err := ensureVlessEncFromAuth(ctx, r.client, inbound.Settings, inbound.Protocol)
	if err != nil {
		resp.Diagnostics.AddError("Failed to resolve VLESS encryption", err.Error())
		return
	}
	inbound.Settings = settingsJSON

	if err := applyDefaultInboundSettings(inbound); err != nil {
		resp.Diagnostics.AddError("Failed to apply default inbound settings", err.Error())
		return
	}
	if err := ensureRealityKeys(ctx, r.client, inbound, nil); err != nil {
		resp.Diagnostics.AddError("Failed to ensure Reality keys", err.Error())
		return
	}
	if err := ensureInboundClientIDs(inbound); err != nil {
		resp.Diagnostics.AddError("Failed to ensure inbound client IDs", err.Error())
		return
	}

	// AmneziaWG: hold the configured server fields back so the panel generates a
	// complete obfuscation set, then re-apply them below. See
	// splitAmneziawgServer for why a partial block silently disables obfuscation.
	var amneziawgServerOverrides map[string]any
	if inbound.Protocol == "amneziawg" {
		rest, overrides, splitErr := splitAmneziawgServer(inbound.Settings)
		if splitErr != nil {
			resp.Diagnostics.AddError("Failed to prepare AmneziaWG settings", splitErr.Error())
			return
		}
		inbound.Settings = rest
		amneziawgServerOverrides = overrides
	}

	// Acquire inboundClientMu if settings contains clients[] to serialise
	// with concurrent inbound_client RMW cycles on the same inbound (#343).
	if settingsHasClients(inbound.Settings) {
		inboundClientMu.Lock()
		defer inboundClientMu.Unlock()
	}

	created, err := r.client.AddInbound(ctx, inbound)
	if err != nil {
		if hint := deprecatedProtocolHint(inbound.Protocol); hint != "" {
			resp.Diagnostics.AddError("Failed to create inbound", err.Error()+"\n\n"+hint)
		} else {
			resp.Diagnostics.AddError("Failed to create inbound", err.Error())
		}
		return
	}

	// 3x-ui occasionally returns success with an empty obj when SQLite is
	// contended: the row is committed seconds later. Recover by polling the
	// list endpoint for the matching port (3x-ui enforces port uniqueness).
	// See issue #157.
	if created == nil || created.ID == 0 {
		var resolvedID int
		retryErr := r.client.WithReadAfterWriteRetry(ctx, "AddInbound resolve by port", func() (bool, error) {
			list, listErr := r.client.GetInbounds(ctx)
			if listErr != nil {
				return false, listErr
			}
			for i := range list {
				if list[i].Port == inbound.Port {
					resolvedID = list[i].ID
					return true, nil
				}
			}
			return false, nil
		})
		if retryErr != nil {
			resp.Diagnostics.AddError("Failed to resolve created inbound",
				fmt.Sprintf("AddInbound returned success but the row was not visible: %s", retryErr.Error()))
			return
		}
		created = &Inbound{ID: resolvedID}
	}

	// Re-read the inbound via GET to ensure consistent state (#131).
	// The add endpoint may return incomplete data under SQLite pressure.
	// Retry on transient errors — the row may not be committed yet (#223).
	if retryErr := r.client.WithReadAfterWriteRetry(ctx, fmt.Sprintf("read created inbound %d", created.ID), func() (bool, error) {
		got, getErr := r.client.GetInbound(ctx, created.ID)
		if getErr != nil {
			return false, getErr
		}
		created = got
		return true, nil
	}); retryErr != nil {
		resp.Diagnostics.AddError("Failed to read created inbound", retryErr.Error())
		return
	}

	// Second phase of the AmneziaWG create: the panel has now generated a full
	// server block, so the configured fields can be laid over it without wiping
	// the obfuscation parameters.
	if settled, err := applyAmneziawgServerPhaseTwo(ctx, r.client, created, amneziawgServerOverrides); err != nil {
		resp.Diagnostics.AddError("Failed to apply AmneziaWG server settings", err.Error())
		return
	} else {
		created = settled
	}

	state, diags := inboundToModel(created, false)
	resp.Diagnostics.Append(diags...)
	// Preserve plan block presence: if the user did not specify a block in
	// config, nil it out even if the API returned data for it.  This avoids
	// the "was absent, but now present" inconsistency error from Terraform.
	alignBlocksWithPlan(state, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	r.maybeRestartXray(ctx, &plan)
}

func (r *InboundResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InboundResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", err.Error())
		return
	}

	inbound, err := r.client.GetInbound(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read inbound", err.Error())
		return
	}

	newState, diags := inboundToModel(inbound, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Preserve block presence from prior state so that Read does not
	// introduce blocks the user never specified, which would cause
	// perpetual diffs.  Skip alignment when coming from import (where
	// protocol is not yet set in the prior state).
	if !state.Protocol.IsNull() && !state.Protocol.IsUnknown() {
		alignBlocksWithPlan(newState, &state)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// inboundReflectsSent reports whether got reflects the scalar fields we
// just wrote in sent. Used to detect the post-update SQLite visibility lag
// (issue #157): if these scalars don't yet match the request, the panel
// hasn't applied the update on the read side. Settings / StreamSettings /
// Sniffing are deliberately excluded — the panel may decorate them
// (Reality keys, defaults), so byte-equality is too strict for a
// visibility check.
func inboundReflectsSent(got, sent *Inbound) bool {
	if got == nil || sent == nil {
		return false
	}
	return got.Remark == sent.Remark &&
		got.Port == sent.Port &&
		got.Enable == sent.Enable &&
		got.Listen == sent.Listen &&
		got.Total == sent.Total &&
		got.ExpiryTime == sent.ExpiryTime &&
		got.Protocol == sent.Protocol
}

func (r *InboundResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan InboundResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state InboundResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", err.Error())
		return
	}

	existing, err := r.client.GetInbound(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read existing inbound", err.Error())
		return
	}

	inbound := expandInboundFromModel(&plan)

	settingsJSON, err := ensureVlessEncFromAuth(ctx, r.client, inbound.Settings, inbound.Protocol)
	if err != nil {
		resp.Diagnostics.AddError("Failed to resolve VLESS encryption", err.Error())
		return
	}
	inbound.Settings = settingsJSON

	if err := applyDefaultInboundSettings(inbound); err != nil {
		resp.Diagnostics.AddError("Failed to apply default inbound settings", err.Error())
		return
	}
	if err := ensureRealityKeys(ctx, r.client, inbound, existing); err != nil {
		resp.Diagnostics.AddError("Failed to ensure Reality keys", err.Error())
		return
	}
	if err := preserveInboundSettings(inbound, existing); err != nil {
		resp.Diagnostics.AddError("Failed to preserve inbound settings", err.Error())
		return
	}
	inbound.ID = id

	// Acquire inboundClientMu if settings contains clients[] to serialise
	// with concurrent inbound_client RMW cycles on the same inbound (#343).
	if settingsHasClients(inbound.Settings) {
		inboundClientMu.Lock()
		defer inboundClientMu.Unlock()
	}

	_, err = r.client.UpdateInbound(ctx, inbound)
	if err != nil {
		if hint := deprecatedProtocolHint(inbound.Protocol); hint != "" {
			resp.Diagnostics.AddError("Failed to update inbound", err.Error()+"\n\n"+hint)
		} else {
			resp.Diagnostics.AddError("Failed to update inbound", err.Error())
		}
		return
	}

	// Re-read the inbound via GET to ensure consistent state (#131).
	// Under SQLite contention the GET may briefly return the pre-update
	// snapshot, which Terraform then rejects as inconsistent. Poll until
	// scalar fields we just wrote are reflected, or the budget expires
	// (issue #157).
	var updated *Inbound
	if retryErr := r.client.WithReadAfterWriteRetry(ctx, fmt.Sprintf("read updated inbound %d", id), func() (bool, error) {
		got, getErr := r.client.GetInbound(ctx, id)
		if getErr != nil {
			return false, getErr
		}
		updated = got
		return inboundReflectsSent(got, inbound), nil
	}); retryErr != nil {
		resp.Diagnostics.AddError("Failed to read updated inbound", retryErr.Error())
		return
	}

	// A peer dropped from the configuration is removed by this update, but the
	// panel keeps its client row — see releasePeerEmailsRemovedByUpdate.
	releasePeerEmailsRemovedByUpdate(ctx, r.client, id, plan.Protocol.ValueString(), &state, &plan, &resp.Diagnostics)

	newState, diags := inboundToModel(updated, false)
	resp.Diagnostics.Append(diags...)
	alignBlocksWithPlan(newState, &plan)
	// Tag is Computed-only (auto-generated by panel). The panel may
	// regenerate the tag on update (e.g. port/listen change). Preserve
	// the planned value to avoid inconsistency; the new tag will be
	// picked up on the next refresh.
	if !plan.Tag.IsNull() && !plan.Tag.IsUnknown() {
		newState.Tag = plan.Tag
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
	r.maybeRestartXray(ctx, &plan)
}

func (r *InboundResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InboundResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.maybeRestartXray(ctx, &state)

	id, err := parseID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", err.Error())
		return
	}

	if err := r.client.DeleteInbound(ctx, id); err != nil {
		resp.Diagnostics.AddError("Failed to delete inbound", err.Error())
		return
	}

	// Free the peers' emails now that the inbound is gone. Deliberately after
	// the inbound delete, not before — see releaseInboundOwnedPeers.
	releaseInboundOwnedPeers(ctx, r.client, id, state.Protocol.ValueString(), &state, &resp.Diagnostics)

	// The DELETE was accepted by the API. 3x-ui's DelInbound is a multi-step
	// SQLite operation; under load the row may still be visible to a follow-up
	// list call for a short time. Poll the list to confirm absence so the
	// post-test sweep does not see a "dangling resource" (#136). Exhaustion is
	// reported as a Warning, not Error: the API has already accepted the
	// delete, so leaving the resource in TF state would be the worse failure
	// mode — the next refresh will reconcile.
	if err := r.waitForInboundDeletion(ctx, id); err != nil {
		resp.Diagnostics.AddWarning("Inbound deletion not confirmed within budget", err.Error())
	}
}

// waitForInboundDeletion polls the inbound list until id is absent. Errors
// from the list call are treated as retryable, not as success: a transient
// network blip must not be misread as confirmed deletion. DelInbound is NOT
// idempotent in 3x-ui (it calls GetInbound first and errors on a missing
// row), so we never re-issue DELETE — the original DELETE has already been
// accepted.
func (r *InboundResource) waitForInboundDeletion(ctx context.Context, id int) error {
	// Budget aligned with the client's read-after-write settings so the
	// delete-side visibility window matches the read-after-write side; both
	// observe the same SQLite contention pattern (issue #161).
	attempts, delay := r.client.ReadAfterWriteConfig()
	var lastErr error
	for i := 0; i < attempts; i++ {
		inbounds, err := r.client.GetInbounds(ctx)
		if err != nil {
			lastErr = err
		} else {
			lastErr = nil
			if !slices.ContainsFunc(inbounds, func(in Inbound) bool { return in.ID == id }) {
				return nil
			}
		}
		if i+1 == attempts {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	if lastErr != nil {
		return fmt.Errorf("inbound %d delete confirmation failed: %w", id, lastErr)
	}
	return fmt.Errorf("inbound %d still visible after delete", id)
}

func (r *InboundResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// trafficCounterPaths lists computed traffic-counter attributes that change
// continuously outside of Terraform.  During update ModifyPlan marks them as
// unknown so the framework accepts any value returned by Read, preventing
// "Provider produced inconsistent result after apply" errors (#202).
var trafficCounterPaths = []path.Path{
	path.Root("up"),
	path.Root("down"),
	path.Root("last_traffic_reset_time"),
	// all_time is deliberately absent: no supported panel sends `allTime`, so the
	// attribute is a constant 0 rather than a live counter. Marking it unknown
	// would plan it as "(known after apply)" on every update for nothing (#442).
}

func (r *InboundResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Create: no prior state — nothing to do.
	if req.State.Raw.IsNull() {
		return
	}

	// Destroy: plan is null — nothing to do.
	if req.Plan.Raw.IsNull() {
		return
	}

	// No-op plan: if state and plan are identical, don't touch traffic counters.
	if req.Plan.Raw.Equal(req.State.Raw) {
		return
	}

	// Real update: mark traffic counters as unknown so Terraform accepts
	// whatever Read returns (#202).
	for _, p := range trafficCounterPaths {
		resp.Plan.SetAttribute(ctx, p, types.Int64Unknown())
	}
}

// ---------------------------------------------------------------------------
// Model <-> Inbound conversion
// ---------------------------------------------------------------------------

func expandInboundFromModel(m *InboundResourceModel) *Inbound {
	protocol := m.Protocol.ValueString()

	// Settings: typed blocks -> untyped map -> JSON
	var settingsJSON string
	if settingsMap := expandSettingsFromModel(protocol, m); len(settingsMap) > 0 {
		settingsJSON = buildSettingsJSON(settingsMap, protocol)
	} else {
		settingsJSON = "{}"
	}

	// Stream settings: typed block -> untyped map -> JSON
	var streamSettingsJSON string
	if ssMap := expandStreamSettingsFromModel(m.StreamSettings); len(ssMap) > 0 {
		streamSettingsJSON = buildStreamSettingsJSON(ssMap)
	} else {
		streamSettingsJSON = "{}"
	}

	// Sniffing: typed block -> untyped map -> JSON
	var sniffingJSON string
	if snMap := expandSniffingFromModel(m.Sniffing); len(snMap) > 0 {
		sniffingJSON = buildSniffingJSON(snMap)
	} else {
		sniffingJSON = "{}"
	}

	inbound := &Inbound{
		Up:                   m.Up.ValueInt64(),
		Down:                 m.Down.ValueInt64(),
		Total:                m.Total.ValueInt64(),
		Remark:               m.Remark.ValueString(),
		Enable:               m.Enable.ValueBool(),
		ExpiryTime:           m.ExpiryTime.ValueInt64(),
		TrafficReset:         m.TrafficReset.ValueString(),
		TrafficResetDay:      int(m.TrafficResetDay.ValueInt64()),
		LastTrafficResetTime: m.LastTrafficResetTime.ValueInt64(),
		Listen:               m.Listen.ValueString(),
		Port:                 int(m.Port.ValueInt64()),
		Protocol:             protocol,
		Settings:             settingsJSON,
		StreamSettings:       streamSettingsJSON,
		Sniffing:             sniffingJSON,
	}
	if !m.NodeID.IsNull() && !m.NodeID.IsUnknown() {
		nodeID := int(m.NodeID.ValueInt64())
		inbound.NodeID = &nodeID
	}
	if !m.SubSortIndex.IsNull() && !m.SubSortIndex.IsUnknown() {
		inbound.SubSortIndex = int(m.SubSortIndex.ValueInt64())
	}
	if !m.ShareAddr.IsNull() && !m.ShareAddr.IsUnknown() {
		inbound.ShareAddr = m.ShareAddr.ValueString()
	}
	if !m.DisableFlow.IsNull() && !m.DisableFlow.IsUnknown() {
		inbound.DisableFlow = m.DisableFlow.ValueBool()
	}
	if !m.ShareAddrStrategy.IsNull() && !m.ShareAddrStrategy.IsUnknown() {
		inbound.ShareAddrStrategy = m.ShareAddrStrategy.ValueString()
	}
	return inbound
}

// inboundToModel converts an API Inbound into a Terraform model.
// When failHard is true (Read/Import), parse errors are reported as errors.
// When false (Create/Update), parse errors are reported as warnings so that
// the model with basic fields is still returned — this prevents leaving an
// unmanaged resource after a successful write to the API.
func inboundToModel(inbound *Inbound, failHard bool) (*InboundResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	m := &InboundResourceModel{
		ID:                   types.StringValue(fmt.Sprintf("%d", inbound.ID)),
		Up:                   types.Int64Value(inbound.Up),
		Down:                 types.Int64Value(inbound.Down),
		Total:                types.Int64Value(inbound.Total),
		AllTime:              types.Int64Value(inbound.AllTime),
		Remark:               types.StringValue(inbound.Remark),
		Enable:               types.BoolValue(inbound.Enable),
		ExpiryTime:           types.Int64Value(inbound.ExpiryTime),
		TrafficReset:         types.StringValue(inbound.TrafficReset),
		TrafficResetDay:      types.Int64Value(int64(inbound.TrafficResetDay)),
		LastTrafficResetTime: types.Int64Value(inbound.LastTrafficResetTime),
		Listen:               stringValueOrNull(inbound.Listen),
		Port:                 types.Int64Value(int64(inbound.Port)),
		Protocol:             types.StringValue(inbound.Protocol),
		Tag:                  types.StringValue(inbound.Tag),
		NodeID:               types.Int64Null(),
		SubSortIndex:         types.Int64Value(int64(inbound.SubSortIndex)),
		ShareAddr:            stringValueOrNull(inbound.ShareAddr),
		ShareAddrStrategy:    stringValueOrNull(inbound.ShareAddrStrategy),
		DisableFlow:          types.BoolValue(inbound.DisableFlow),
	}
	if inbound.NodeID != nil {
		m.NodeID = types.Int64Value(int64(*inbound.NodeID))
	}

	addDiag := diags.AddWarning
	if failHard {
		addDiag = diags.AddError
	}

	// Settings: JSON -> untyped map -> typed model
	settingsMap, err := flattenSettingsToMap(inbound.Settings, inbound.Protocol)
	if err != nil {
		addDiag("Failed to parse inbound settings", err.Error())
	} else if settingsMap != nil {
		flattenSettingsToModel(inbound.Protocol, settingsMap, m)
	}

	// Stream settings: JSON -> untyped map -> typed model
	ssMap, err := flattenStreamSettingsToMap(inbound.StreamSettings)
	if err != nil {
		addDiag("Failed to parse inbound stream_settings", err.Error())
	} else if ssMap != nil {
		m.StreamSettings = flattenStreamSettingsToModel(ssMap)
	}

	// Sniffing: JSON -> untyped map -> typed model
	snMap, err := flattenSniffingToMap(inbound.Sniffing)
	if err != nil {
		addDiag("Failed to parse inbound sniffing", err.Error())
	} else if snMap != nil {
		m.Sniffing = flattenSniffingToModel(snMap)
	}

	return m, diags
}

// alignBlocksWithPlan nils out blocks on the state that were not present in the
// plan.  This prevents the "was absent, but now present" inconsistency error
// that Terraform raises when a Computed block appears in the state but was not
// in the configuration.
func alignBlocksWithPlan(state *InboundResourceModel, plan *InboundResourceModel) {
	if plan.VlessSettings == nil {
		state.VlessSettings = nil
	}
	if plan.TrojanSettings == nil {
		state.TrojanSettings = nil
	}
	if plan.ShadowsocksSettings == nil {
		state.ShadowsocksSettings = nil
	}
	if plan.HTTPSettings == nil {
		state.HTTPSettings = nil
	}
	if plan.SocksSettings == nil {
		state.SocksSettings = nil
	}
	if plan.MixedSettings == nil {
		state.MixedSettings = nil
	}
	if plan.WireguardSettings == nil {
		state.WireguardSettings = nil
	}
	if plan.AmneziawgSettings == nil {
		state.AmneziawgSettings = nil
	} else if state.AmneziawgSettings != nil && plan.AmneziawgSettings.Server == nil {
		// The nested server block follows the same rule as the top-level ones:
		// a block absent from the configuration must stay absent in state, or the
		// framework reports "was absent, but now present".
		state.AmneziawgSettings.Server = nil
	}
	if plan.TuicSettings == nil {
		state.TuicSettings = nil
	}
	if plan.DokodemoSettings == nil {
		state.DokodemoSettings = nil
	}
	if plan.HysteriaSettings == nil {
		state.HysteriaSettings = nil
	}
	if plan.MtprotoSettings == nil {
		state.MtprotoSettings = nil
	}
	if plan.StreamSettings == nil {
		state.StreamSettings = nil
	} else if state.StreamSettings != nil {
		// Align nested stream_settings sub-blocks
		if plan.StreamSettings.RealitySettings == nil {
			state.StreamSettings.RealitySettings = nil
		}
		if plan.StreamSettings.TCPSettings == nil {
			state.StreamSettings.TCPSettings = nil
		}
		if plan.StreamSettings.WSSettings == nil {
			state.StreamSettings.WSSettings = nil
		}
		if plan.StreamSettings.GRPCSettings == nil {
			state.StreamSettings.GRPCSettings = nil
		}
		if plan.StreamSettings.HTTPUpgradeSettings == nil {
			state.StreamSettings.HTTPUpgradeSettings = nil
		}
		if plan.StreamSettings.XHTTPSettings == nil {
			state.StreamSettings.XHTTPSettings = nil
		}
		if plan.StreamSettings.KCPSettings == nil {
			state.StreamSettings.KCPSettings = nil
		}
		if plan.StreamSettings.HysteriaSettings == nil {
			state.StreamSettings.HysteriaSettings = nil
		}
		if plan.StreamSettings.Sockopt == nil {
			state.StreamSettings.Sockopt = nil
		}
	}
	if plan.Sniffing == nil {
		state.Sniffing = nil
	}
}

// ---------------------------------------------------------------------------
// ensureVlessEncFromAuth — resolves VLESS decryption/encryption from the
// panel's auth endpoint when selectedAuth is set but decryption/encryption
// are missing in the settings JSON.
// ---------------------------------------------------------------------------

func ensureVlessEncFromAuth(ctx context.Context, client *Client, settingsJSON string, protocol string) (string, error) {
	if protocol != "vless" || client == nil {
		return settingsJSON, nil
	}
	if strings.TrimSpace(settingsJSON) == "" {
		return settingsJSON, nil
	}

	settings, err := ParseJSONField(settingsJSON)
	if err != nil {
		return settingsJSON, err
	}

	selected := stringValue(settings["selectedAuth"])
	if selected == "" {
		return settingsJSON, nil
	}

	decryptionMissing := stringValue(settings["decryption"]) == ""
	encryptionMissing := stringValue(settings["encryption"]) == ""
	if !decryptionMissing && !encryptionMissing {
		return settingsJSON, nil
	}

	auths, err := client.GetNewVlessEnc(ctx)
	if err != nil {
		return settingsJSON, err
	}

	var match *VlessEncAuth
	for i := range auths {
		if auths[i].Label == selected {
			match = &auths[i]
			break
		}
	}
	if match == nil {
		return settingsJSON, fmt.Errorf("no auth block for selected_auth %q", selected)
	}

	if decryptionMissing {
		settings["decryption"] = match.Decryption
	}
	if encryptionMissing {
		settings["encryption"] = match.Encryption
	}

	updated, err := json.Marshal(settings)
	if err != nil {
		return settingsJSON, err
	}
	return string(updated), nil
}

// ---------------------------------------------------------------------------
// preserveInboundSettings / preserveSettingsKey — preserve clients and
// testseed from existing inbound during update (no SDK dependency)
// ---------------------------------------------------------------------------

func preserveInboundSettings(desired *Inbound, existing *Inbound) error {
	if desired == nil || existing == nil {
		return nil
	}
	if strings.TrimSpace(desired.Settings) == "" || strings.TrimSpace(existing.Settings) == "" {
		return nil
	}
	desiredSettings, err := ParseJSONField(desired.Settings)
	if err != nil {
		return err
	}
	existingSettings, err := ParseJSONField(existing.Settings)
	if err != nil {
		return err
	}
	// clients[] is only preserved for protocols whose clients belong to
	// threexui_inbound_client. WireGuard and AmneziaWG peers are owned by the
	// inbound itself, so copying the existing array back would make removing the
	// last peer impossible: the plan drops it, this puts it straight back, and
	// the apply fails with "block count changed from 0 to 1" while the peer keeps
	// connecting.
	if protocolOwnsClients(desired.Protocol) {
		_ = preserveSettingsKey(desiredSettings, existingSettings, "testseed")
		updated, err := json.Marshal(desiredSettings)
		if err != nil {
			return err
		}
		desired.Settings = string(updated)
		return nil
	}
	if !preserveSettingsKey(desiredSettings, existingSettings, "clients") {
		return nil
	}
	_ = preserveSettingsKey(desiredSettings, existingSettings, "testseed")
	updated, err := json.Marshal(desiredSettings)
	if err != nil {
		return err
	}
	desired.Settings = string(updated)
	return nil
}

func preserveSettingsKey(desired, existing map[string]any, key string) bool {
	if existing == nil {
		return false
	}
	existingVal, ok := existing[key]
	if !ok {
		return false
	}
	switch v := existingVal.(type) {
	case []any:
		if len(v) == 0 {
			return false
		}
	}
	if desired == nil {
		return false
	}
	if desiredVal, ok := desired[key]; ok {
		if list, ok := desiredVal.([]any); ok && len(list) > 0 {
			return false
		}
	}
	desired[key] = existingVal
	return true
}

// ---------------------------------------------------------------------------
// settingsHasClients returns true if the inbound settings JSON contains a
// non-empty clients array. Used to decide whether InboundResource should
// acquire inboundClientMu (serialising with inbound_client RMW cycles).
// ---------------------------------------------------------------------------
func settingsHasClients(settingsJSON string) bool {
	settings, err := ParseJSONField(settingsJSON)
	if err != nil || settings == nil {
		return false
	}
	clients, ok := settings["clients"].([]any)
	return ok && len(clients) > 0
}

// ---------------------------------------------------------------------------
// ensureInboundClientIDs — auto-generate UUIDs for clients without id
// (no SDK dependency)
// ---------------------------------------------------------------------------

func ensureInboundClientIDs(inbound *Inbound) error {
	if inbound == nil {
		return nil
	}
	settings, err := ParseJSONField(inbound.Settings)
	if err != nil {
		return err
	}
	clientsRaw, ok := settings["clients"]
	if !ok {
		return nil
	}
	clients, ok := clientsRaw.([]any)
	if !ok {
		return nil
	}
	changed := false
	for i := range clients {
		clientMap, ok := clients[i].(map[string]any)
		if !ok {
			continue
		}
		id, _ := clientMap["id"].(string)
		if id == "" {
			newID, err := newUUID()
			if err != nil {
				return err
			}
			clientMap["id"] = newID
			clients[i] = clientMap
			changed = true
		}
	}
	if !changed {
		return nil
	}
	settings["clients"] = clients
	updated, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	inbound.Settings = string(updated)
	return nil
}

// ---------------------------------------------------------------------------
// Reality key helpers (no SDK dependency)
// ---------------------------------------------------------------------------

func ensureRealityKeys(ctx context.Context, client *Client, inbound *Inbound, existing *Inbound) error {
	if inbound == nil || inbound.StreamSettings == "" {
		return nil
	}
	payload, err := ParseJSONField(inbound.StreamSettings)
	if err != nil {
		return err
	}
	security := stringValue(payload["security"])
	if security != "reality" {
		return nil
	}
	rs, _ := payload["realitySettings"].(map[string]any)
	if rs == nil {
		rs = map[string]any{}
	}
	mergeRealityFromExisting(existing, rs)
	ensureRealityDefaults(rs)
	if !hasRealityShortIDs(rs) {
		rs["shortIds"] = randomShortIDs()
	}
	if pk, ok := rs["privateKey"].(string); ok && pk != "" {
		return nil
	}
	cert, err := client.GetNewX25519Cert(ctx)
	if err != nil {
		return err
	}
	privateKey := stringValue(cert["privateKey"])
	publicKey := stringValue(cert["publicKey"])
	if privateKey == "" {
		return fmt.Errorf("generated reality privateKey is empty")
	}
	rs["privateKey"] = privateKey
	settings, _ := rs["settings"].(map[string]any)
	if settings == nil {
		settings = map[string]any{}
	}
	if settings["publicKey"] == nil || stringValue(settings["publicKey"]) == "" {
		settings["publicKey"] = publicKey
	}
	rs["settings"] = settings
	payload["realitySettings"] = rs
	updated, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	inbound.StreamSettings = string(updated)
	return nil
}

func ensureRealityDefaults(reality map[string]any) {
	if reality == nil {
		return
	}
	if hasStringListValues(reality["serverNames"]) {
		return
	}
	target := stringValue(reality["target"])
	if target != "" {
		host := strings.Split(target, ":")[0]
		if host != "" {
			reality["serverNames"] = []any{host}
			return
		}
	}
	reality["target"] = "www.amazon.com:443"
	reality["serverNames"] = []any{"www.amazon.com", "amazon.com"}
}

func mergeRealityFromExisting(existing *Inbound, reality map[string]any) {
	if existing == nil || existing.StreamSettings == "" {
		return
	}
	payload, err := ParseJSONField(existing.StreamSettings)
	if err != nil {
		return
	}
	rs, _ := payload["realitySettings"].(map[string]any)
	if rs == nil {
		return
	}
	if stringValue(reality["privateKey"]) == "" {
		if pk := stringValue(rs["privateKey"]); pk != "" {
			reality["privateKey"] = pk
		}
	}
	if !hasRealityShortIDs(reality) {
		if raw, ok := rs["shortIds"]; ok {
			if list, ok := raw.([]any); ok && len(list) > 0 {
				reality["shortIds"] = list
			}
		}
	}
	settings, _ := reality["settings"].(map[string]any)
	if settings == nil {
		settings = map[string]any{}
	}
	if stringValue(settings["publicKey"]) == "" {
		if s, ok := rs["settings"].(map[string]any); ok {
			if pk := stringValue(s["publicKey"]); pk != "" {
				settings["publicKey"] = pk
			}
		}
	}
	reality["settings"] = settings
}

func hasRealityShortIDs(reality map[string]any) bool {
	if reality == nil {
		return false
	}
	return hasStringListValues(reality["shortIds"])
}

func hasRealityServerNames(reality map[string]any) bool {
	if reality == nil {
		return false
	}
	return hasStringListValues(reality["serverNames"])
}

func hasStringListValues(raw any) bool {
	switch v := raw.(type) {
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				return true
			}
		}
	case []string:
		for _, s := range v {
			if s != "" {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Random helpers (no SDK dependency)
// ---------------------------------------------------------------------------

func randomHex(length int) string {
	if length <= 0 {
		return ""
	}
	buf := make([]byte, (length+1)/2)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	out := hex.EncodeToString(buf)
	if len(out) > length {
		out = out[:length]
	}
	return out
}

func randomShortIDs() []any {
	lengths := []int{2, 4, 6, 8, 10, 12, 14, 16}
	out := make([]any, 0, len(lengths))
	for _, l := range lengths {
		out = append(out, randomHex(l))
	}
	return out
}

// ---------------------------------------------------------------------------
// parseID — parses a numeric string ID (no SDK dependency)
// ---------------------------------------------------------------------------

func parseID(id string) (int, error) {
	var parsed int
	_, err := fmt.Sscanf(id, "%d", &parsed)
	if err != nil || parsed == 0 {
		return 0, fmt.Errorf("invalid id: %s", id)
	}
	return parsed, nil
}

// ---------------------------------------------------------------------------
// isSubset — standalone utility function (no SDK dependency)
// ---------------------------------------------------------------------------

func isSubset(desired, actual any) bool {
	switch dv := desired.(type) {
	case map[string]any:
		av, ok := actual.(map[string]any)
		if !ok {
			return false
		}
		for k, dval := range dv {
			aval, ok := av[k]
			if !ok {
				return false
			}
			if !isSubset(dval, aval) {
				return false
			}
		}
		return true
	case []any:
		av, ok := actual.([]any)
		if !ok {
			return false
		}
		if len(dv) == 0 {
			return true
		}
		if len(dv) > len(av) {
			return false
		}
		for i := range dv {
			found := false
			for j := range av {
				if isSubset(dv[i], av[j]) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(desired, actual)
	}
}

// maybeRestartXray restarts the Xray core if the resource's restart_xray
// attribute is set to true.
func (r *InboundResource) maybeRestartXray(ctx context.Context, plan *InboundResourceModel) {
	if plan.RestartXray.ValueBool() {
		if err := r.client.RestartXrayService(ctx); err != nil {
			tflog.Warn(ctx, "restartXrayService failed", map[string]any{"error": err.Error()})
		}
	}
}

// deprecatedProtocolHint returns a user-facing hint when a protocol
// was removed in 3x-ui v3.2.0.
func deprecatedProtocolHint(protocol string) string {
	switch protocol {
	case "socks":
		return `Protocol "socks" is not supported by 3x-ui v3.2.0+. Use protocol "mixed" instead.`
	case "dokodemo-door":
		return `Protocol "dokodemo-door" is not supported by 3x-ui v3.2.0+. Use protocol "tunnel" instead.`
	default:
		return ""
	}
}
