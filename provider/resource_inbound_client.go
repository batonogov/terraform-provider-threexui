package provider

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var inboundClientMu sync.Mutex

// ---------------------------------------------------------------------------
// Interface checks
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &InboundClientResource{}
	_ resource.ResourceWithImportState = &InboundClientResource{}
	_ resource.ResourceWithModifyPlan  = &InboundClientResource{}
)

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type InboundClientResourceModel struct {
	ID              types.String `tfsdk:"id"`
	InboundID       types.Int64  `tfsdk:"inbound_id"`
	ClientID        types.String `tfsdk:"client_id"`
	Email           types.String `tfsdk:"email"`
	Security        types.String `tfsdk:"security"`
	Password        types.String `tfsdk:"password"`
	Flow            types.String `tfsdk:"flow"`
	ReverseTag      types.String `tfsdk:"reverse_tag"`
	Auth            types.String `tfsdk:"auth"`
	LimitIP         types.Int64  `tfsdk:"limit_ip"`
	TotalGB         types.Int64  `tfsdk:"total_gb"`
	ExpiryTime      types.Int64  `tfsdk:"expiry_time"`
	Enable          types.Bool   `tfsdk:"enable"`
	TgID            types.Int64  `tfsdk:"tg_id"`
	SubID           types.String `tfsdk:"sub_id"`
	Comment         types.String `tfsdk:"comment"`
	Reset           types.Int64  `tfsdk:"reset"`
	Group           types.String `tfsdk:"group"`
	ResetDay        types.Int64  `tfsdk:"reset_day"`
	ResetMax        types.Int64  `tfsdk:"reset_max"`
	ResetWeekday    types.Int64  `tfsdk:"reset_weekday"`
	TrafficReset    types.String `tfsdk:"traffic_reset"`
	TrafficResetDay types.Int64  `tfsdk:"traffic_reset_day"`
	Secret          types.String `tfsdk:"secret"`
	AdTag           types.String `tfsdk:"ad_tag"`
	RestartXray     types.Bool   `tfsdk:"restart_xray"`

	// Write-only secret variants (Strategy B — see resource_node.go).
	PasswordWO        types.String `tfsdk:"password_wo"`
	PasswordWOVersion types.Int64  `tfsdk:"password_wo_version"`
	SecretWO          types.String `tfsdk:"secret_wo"`
	SecretWOVersion   types.Int64  `tfsdk:"secret_wo_version"`
}

// ---------------------------------------------------------------------------
// Resource
// ---------------------------------------------------------------------------

type InboundClientResource struct {
	client *Client
}

func NewInboundClientResource() resource.Resource {
	return &InboundClientResource{}
}

func (r *InboundClientResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inbound_client"
}

func (r *InboundClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"inbound_id": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"client_id": schema.StringAttribute{
				Optional:  true,
				Computed:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					// client_id is Computed and usually omitted from config. Keep the prior
					// state value instead of going unknown on an unrelated change (e.g. a
					// comment edit); without this the planned client_id becomes "known after
					// apply", and RequiresReplace then recreates the client — rotating its
					// UUID and subId and breaking its vless link and subscription.
					stringplanmodifier.UseStateForUnknown(),
					// An explicit client_id change is still a replace (the UUID is identity).
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				Required: true,
			},
			"security": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				Optional:  true,
				Computed:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.PreferWriteOnlyAttribute(path.MatchRoot("password_wo")),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password_wo": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Description: "Write-only password for trojan/shadowsocks clients. Use password_wo_version to trigger updates.",
			},
			"password_wo_version": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Increment to trigger password update when using password_wo.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.AlsoRequires(path.MatchRoot("password_wo")),
				},
			},
			"flow": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"reverse_tag": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "VLESS reverse tag. Stored in 3x-ui as reverse.tag.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"auth": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Description: "Auth password for Hysteria clients.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"limit_ip": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"total_gb": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"expiry_time": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"enable": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"tg_id": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"sub_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"comment": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"reset": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(0),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"group": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Client group name (3x-ui v3.2.0+).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"reset_day": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Description: "Calendar day of month (1-31) on which this client renews. " +
					"0 keeps the rolling-interval behaviour driven by `reset`. " +
					"3x-ui v3.7.0+; older panels report 0 (unsupported).",
				Validators: []validator.Int64{
					int64validator.Between(0, 31),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"reset_max": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Description: "Maximum number of automatic renewals for this client. 0 means unlimited. " +
					"3x-ui v3.7.0+; older panels report 0 (unsupported).",
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"reset_weekday": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Description: "Calendar weekday (1-7, Monday-Sunday) on which this client renews weekly. " +
					"0 disables weekly renewal. Mutually exclusive with a positive reset or reset_day: " +
					"the panel rejects the combination (validateClientRenewal). " +
					"3x-ui v3.9.0+; older panels report 0 (unsupported).",
				Validators: []validator.Int64{
					int64validator.Between(0, 7),
					resetWeekdayExclusiveValidator{},
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"traffic_reset": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Per-client traffic reset cycle ('never', 'hourly', 'daily', 'weekly', 'monthly'), " +
					"independent of the inbound's own cycle. 3x-ui v3.7.0+; older panels report an empty value (unsupported).",
				Validators: trafficResetValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"traffic_reset_day": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Description: "Day of month (1-31) for this client's monthly traffic resets. " +
					"Only effective when traffic_reset = 'monthly'. 3x-ui v3.7.0+; older panels report 0 (unsupported). " +
					"Cannot be set to 0: the panel clamps any value below 1 up to 1 (normalizeClientTrafficReset), " +
					"so a configured 0 could never round-trip.",
				Validators: []validator.Int64{
					int64validator.Between(1, 31),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"secret": schema.StringAttribute{
				Optional:  true,
				Computed:  true,
				Sensitive: true,
				Description: "MTProto FakeTLS secret (3x-ui v3.5.0+, mtg-multi per-client). " +
					"Format: \"ee\" + 32 hex chars (random middle) + hex-encoded domain suffix. " +
					"The panel rebuilds the domain suffix from the inbound's fakeTlsDomain on save, " +
					"so only the random middle must be stable across applies. Setting a domain " +
					"suffix that differs from the inbound's fakeTlsDomain causes drift after the " +
					"first apply (the panel heals it) — leave unset to let the panel generate it. " +
					"Leave unset for non-MTProto clients.",
				Validators: []validator.String{
					stringvalidator.PreferWriteOnlyAttribute(path.MatchRoot("secret_wo")),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_wo": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Description: "Write-only MTProto FakeTLS secret (3x-ui v3.5.0+). Use secret_wo_version to trigger updates.",
			},
			"secret_wo_version": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Increment to trigger secret update when using secret_wo.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.AlsoRequires(path.MatchRoot("secret_wo")),
				},
			},
			"ad_tag": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "MTProto advertising tag from @MTProxybot (3x-ui v3.5.0+). " +
					"Must be exactly 32 hex characters (16 bytes). Leave unset for non-MTProto clients.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9a-fA-F]{32}$`),
						"ad_tag must be exactly 32 hexadecimal characters (16 bytes)"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"restart_xray": schema.BoolAttribute{
				Optional:    true,
				Description: "Restart Xray core after create, update, or delete operations. Default is false.",
			},
		},
	}
}

func (r *InboundClientResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ---------------------------------------------------------------------------
// ModifyPlan — mark plain secrets Unknown when _wo_version changes (inlined
// for two WO secrets, same pattern as resource_node.go).
// ---------------------------------------------------------------------------

func (r *InboundClientResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	var plan, state InboundClientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	changed := false
	if woVersionTriggered(plan.PasswordWOVersion, state.PasswordWOVersion) {
		plan.Password = types.StringUnknown()
		changed = true
	}
	if woVersionTriggered(plan.SecretWOVersion, state.SecretWOVersion) {
		plan.Secret = types.StringUnknown()
		changed = true
	}
	if changed {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

// resolveInboundClientSecretsWO copies _wo values into plain fields on Create.
func resolveInboundClientSecretsWO(plan *InboundClientResourceModel, config InboundClientResourceModel) {
	if !config.PasswordWO.IsNull() && !config.PasswordWO.IsUnknown() {
		plan.Password = config.PasswordWO
	}
	if !config.SecretWO.IsNull() && !config.SecretWO.IsUnknown() {
		plan.Secret = config.SecretWO
	}
}

// resolveInboundClientSecretsWOUpdate copies _wo values into plain fields on
// Update only when the _wo_version trigger changed.
func resolveInboundClientSecretsWOUpdate(plan *InboundClientResourceModel, state, config InboundClientResourceModel) {
	if !config.PasswordWO.IsNull() && !config.PasswordWO.IsUnknown() &&
		woVersionTriggered(plan.PasswordWOVersion, state.PasswordWOVersion) {
		plan.Password = config.PasswordWO
	}
	if !config.SecretWO.IsNull() && !config.SecretWO.IsUnknown() &&
		woVersionTriggered(plan.SecretWOVersion, state.SecretWOVersion) {
		plan.Secret = config.SecretWO
	}
}

// ---------------------------------------------------------------------------
// CRUD
// ---------------------------------------------------------------------------

func (r *InboundClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InboundClientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config InboundClientResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveInboundClientSecretsWO(&plan, config)

	// Ensure _wo_version is known (not Unknown) after Create — Terraform
	// requires all Computed values to be known after apply. If no _wo is
	// configured, default to 0.
	if plan.PasswordWOVersion.IsUnknown() {
		plan.PasswordWOVersion = types.Int64Value(0)
	}
	if plan.SecretWOVersion.IsUnknown() {
		plan.SecretWOVersion = types.Int64Value(0)
	}

	inboundID := int(plan.InboundID.ValueInt64())

	inboundClientMu.Lock()
	defer inboundClientMu.Unlock()

	if err := ensureInboundClientsKey(ctx, r.client, inboundID); err != nil {
		resp.Diagnostics.AddError("Failed to ensure clients key", err.Error())
		return
	}

	clientData := expandInboundClientFromModel(&plan)
	clientID := getClientIDFromModel(&plan, clientData)
	if clientID == "" {
		var err error
		clientID, err = newUUID()
		if err != nil {
			resp.Diagnostics.AddError("Failed to generate UUID", err.Error())
			return
		}
	}
	clientData["id"] = clientID

	// Generate subId if not present (API does not auto-generate it; the web UI does).
	if stringValue(clientData["subId"]) == "" {
		subID, err := newSubID()
		if err != nil {
			resp.Diagnostics.AddError("Failed to generate sub_id", err.Error())
			return
		}
		clientData["subId"] = subID
	}

	if err := r.client.AddInboundClient(ctx, inboundID, clientData); err != nil {
		resp.Diagnostics.AddError("Failed to add inbound client", err.Error())
		return
	}

	// 3x-ui v3.1.0 ClientService.Create forcibly overrides enable=false to
	// true. Issue a corrective update when the plan requests enable=false so
	// the post-create read returns the correct value.
	if !plan.Enable.IsNull() && !plan.Enable.IsUnknown() && !plan.Enable.ValueBool() {
		if err := r.client.UpdateInboundClient(ctx, inboundID, clientID, plan.Email.ValueString(), clientData); err != nil {
			resp.Diagnostics.AddError("Failed to correct enable field after create", err.Error())
			return
		}
	}

	// Read back from API to populate state. The just-added client may not
	// be visible to a subsequent GET if SQLite is contended (issue #157),
	// so poll until it appears or the budget is exhausted.
	state, err := r.readClientStateWithRetry(ctx, inboundID, clientID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read created inbound client", err.Error())
		return
	}
	if state == nil {
		return
	}
	// Preserve _wo_version from plan — inboundClientToModel builds a fresh
	// model and would lose the trigger, causing perpetual rotation.
	state.PasswordWOVersion = preserveWOVersion(state.PasswordWOVersion, plan.PasswordWOVersion)
	state.SecretWOVersion = preserveWOVersion(state.SecretWOVersion, plan.SecretWOVersion)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	r.maybeRestartXrayClient(ctx, &plan)
}

func (r *InboundClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var cur InboundClientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &cur)...)
	if resp.Diagnostics.HasError() {
		return
	}

	inboundID, clientID, err := splitInboundClientID(cur.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", err.Error())
		return
	}

	state := r.readClientState(ctx, &resp.Diagnostics, inboundID, clientID)
	if state == nil {
		// Client not found — remove from state.
		if !resp.Diagnostics.HasError() {
			resp.State.RemoveResource(ctx)
		}
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *InboundClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan InboundClientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var cur InboundClientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &cur)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config InboundClientResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveInboundClientSecretsWOUpdate(&plan, cur, config)

	inboundID, clientID, err := splitInboundClientID(cur.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", err.Error())
		return
	}

	inboundClientMu.Lock()
	defer inboundClientMu.Unlock()

	if err := ensureInboundClientsKey(ctx, r.client, inboundID); err != nil {
		resp.Diagnostics.AddError("Failed to ensure clients key", err.Error())
		return
	}

	clientData := expandInboundClientFromModel(&plan)
	clientData["id"] = clientID

	if err := r.client.UpdateInboundClient(ctx, inboundID, clientID, cur.Email.ValueString(), clientData); err != nil {
		resp.Diagnostics.AddError("Failed to update inbound client", err.Error())
		return
	}

	state, err := r.readClientStateWithRetry(ctx, inboundID, clientID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read updated inbound client", err.Error())
		return
	}
	if state == nil {
		return
	}
	// Preserve _wo_version from plan — inboundClientToModel builds a fresh
	// model and would lose the trigger, causing perpetual rotation.
	state.PasswordWOVersion = preserveWOVersion(state.PasswordWOVersion, plan.PasswordWOVersion)
	state.SecretWOVersion = preserveWOVersion(state.SecretWOVersion, plan.SecretWOVersion)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	r.maybeRestartXrayClient(ctx, &plan)
}

func (r *InboundClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var cur InboundClientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &cur)...)
	if resp.Diagnostics.HasError() {
		return
	}

	inboundID, clientID, err := splitInboundClientID(cur.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid resource ID", err.Error())
		return
	}

	inboundClientMu.Lock()
	defer inboundClientMu.Unlock()

	if err := ensureInboundClientsKey(ctx, r.client, inboundID); err != nil {
		resp.Diagnostics.AddError("Failed to ensure clients key", err.Error())
		return
	}

	if err := r.client.DeleteInboundClient(ctx, inboundID, clientID, cur.Email.ValueString()); err != nil {
		if strings.Contains(err.Error(), "no client remained in Inbound") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to delete inbound client", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
	r.maybeRestartXrayClient(ctx, &cur)
}

// maybeRestartXrayClient restarts the Xray core if restart_xray is true.
func (r *InboundClientResource) maybeRestartXrayClient(ctx context.Context, m *InboundClientResourceModel) {
	if m.RestartXray.ValueBool() {
		if err := r.client.RestartXrayService(ctx); err != nil {
			tflog.Warn(ctx, "restartXrayService failed", map[string]any{"error": err.Error()})
		}
	}
}

// ---------------------------------------------------------------------------
// Import
// ---------------------------------------------------------------------------

func (r *InboundClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	inboundID, clientID, err := splitInboundClientID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format inbound_id:client_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), makeInboundClientID(inboundID, clientID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("inbound_id"), int64(inboundID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("client_id"), clientID)...)
}

// ---------------------------------------------------------------------------
// Read helper
// ---------------------------------------------------------------------------

func (r *InboundClientResource) readClientState(ctx context.Context, diags *diag.Diagnostics, inboundID int, clientID string) *InboundClientResourceModel {
	inbound, err := r.client.GetInbound(ctx, inboundID)
	if err != nil {
		diags.AddError("Failed to read inbound", err.Error())
		return nil
	}

	settings, err := parseInboundSettings(inbound.Settings)
	if err != nil {
		diags.AddError("Failed to parse inbound settings", err.Error())
		return nil
	}

	found := findClientByID(settings.Clients, clientID)
	if found == nil {
		return nil
	}

	return inboundClientToModel(inboundID, clientID, found)
}

// readClientStateWithRetry is the post-write variant of readClientState: it
// treats "client not yet visible" as a transient condition and polls until
// the row appears or the budget is exhausted (issue #157). Use it only
// after a successful AddInboundClient/UpdateInboundClient — for plain Read
// the absence of a client is meaningful and should not be retried.
func (r *InboundClientResource) readClientStateWithRetry(ctx context.Context, inboundID int, clientID string) (*InboundClientResourceModel, error) {
	var state *InboundClientResourceModel
	err := r.client.WithReadAfterWriteRetry(ctx, fmt.Sprintf("read inbound %d client %s", inboundID, clientID), func() (bool, error) {
		inbound, getErr := r.client.GetInbound(ctx, inboundID)
		if getErr != nil {
			return false, getErr
		}
		settings, parseErr := parseInboundSettings(inbound.Settings)
		if parseErr != nil {
			return false, parseErr
		}
		found := findClientByID(settings.Clients, clientID)
		if found == nil {
			return false, nil
		}
		state = inboundClientToModel(inboundID, clientID, found)
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return state, nil
}

// ---------------------------------------------------------------------------
// Expand / Flatten
// ---------------------------------------------------------------------------

func expandInboundClientFromModel(m *InboundClientResourceModel) map[string]any {
	client := map[string]any{}

	if !m.Email.IsNull() {
		client["email"] = m.Email.ValueString()
	}
	if !m.Security.IsNull() && !m.Security.IsUnknown() {
		client["security"] = m.Security.ValueString()
	}
	if !m.Password.IsNull() && !m.Password.IsUnknown() {
		client["password"] = m.Password.ValueString()
	}
	if !m.Flow.IsNull() && !m.Flow.IsUnknown() {
		client["flow"] = m.Flow.ValueString()
	}
	if !m.ReverseTag.IsNull() && !m.ReverseTag.IsUnknown() {
		if tag := m.ReverseTag.ValueString(); tag != "" {
			client["reverse"] = map[string]any{"tag": tag}
		}
	}
	if !m.Auth.IsNull() && !m.Auth.IsUnknown() {
		client["auth"] = m.Auth.ValueString()
	}
	if !m.LimitIP.IsNull() && !m.LimitIP.IsUnknown() {
		client["limitIp"] = int(m.LimitIP.ValueInt64())
	}
	if !m.TotalGB.IsNull() && !m.TotalGB.IsUnknown() {
		client["totalGB"] = int(m.TotalGB.ValueInt64())
	}
	if !m.ExpiryTime.IsNull() && !m.ExpiryTime.IsUnknown() {
		client["expiryTime"] = int(m.ExpiryTime.ValueInt64())
	}
	if !m.Enable.IsNull() && !m.Enable.IsUnknown() {
		client["enable"] = m.Enable.ValueBool()
	}
	if !m.TgID.IsNull() && !m.TgID.IsUnknown() {
		client["tgId"] = int(m.TgID.ValueInt64())
	}
	if !m.SubID.IsNull() && !m.SubID.IsUnknown() {
		client["subId"] = m.SubID.ValueString()
	}
	if !m.Comment.IsNull() && !m.Comment.IsUnknown() {
		client["comment"] = m.Comment.ValueString()
	}
	if !m.Reset.IsNull() && !m.Reset.IsUnknown() {
		client["reset"] = int(m.Reset.ValueInt64())
	}
	if !m.Group.IsNull() && !m.Group.IsUnknown() {
		client["group"] = m.Group.ValueString()
	}
	if !m.ResetDay.IsNull() && !m.ResetDay.IsUnknown() {
		client["resetDay"] = int(m.ResetDay.ValueInt64())
	}
	if !m.ResetMax.IsNull() && !m.ResetMax.IsUnknown() {
		client["resetMax"] = int(m.ResetMax.ValueInt64())
	}
	if !m.ResetWeekday.IsNull() && !m.ResetWeekday.IsUnknown() {
		client["resetWeekday"] = int(m.ResetWeekday.ValueInt64())
	}
	if !m.TrafficReset.IsNull() && !m.TrafficReset.IsUnknown() {
		client["trafficReset"] = m.TrafficReset.ValueString()
	}
	if !m.TrafficResetDay.IsNull() && !m.TrafficResetDay.IsUnknown() {
		client["trafficResetDay"] = int(m.TrafficResetDay.ValueInt64())
	}
	if !m.Secret.IsNull() && !m.Secret.IsUnknown() {
		client["secret"] = m.Secret.ValueString()
	}
	if !m.AdTag.IsNull() && !m.AdTag.IsUnknown() {
		client["adTag"] = m.AdTag.ValueString()
	}
	if !m.ClientID.IsNull() && !m.ClientID.IsUnknown() {
		client["id"] = m.ClientID.ValueString()
	}

	return client
}

func inboundClientToModel(inboundID int, clientID string, client map[string]any) *InboundClientResourceModel {
	return &InboundClientResourceModel{
		ID:              types.StringValue(makeInboundClientID(inboundID, clientID)),
		InboundID:       types.Int64Value(int64(inboundID)),
		ClientID:        types.StringValue(clientID),
		Email:           types.StringValue(stringValue(client["email"])),
		Security:        types.StringValue(stringValue(client["security"])),
		Password:        types.StringValue(stringValue(client["password"])),
		Flow:            types.StringValue(stringValue(client["flow"])),
		ReverseTag:      types.StringValue(reverseTagValue(client["reverse"])),
		Auth:            types.StringValue(stringValue(client["auth"])),
		LimitIP:         types.Int64Value(int64(intValue(client["limitIp"]))),
		TotalGB:         types.Int64Value(int64(intValue(client["totalGB"]))),
		ExpiryTime:      types.Int64Value(int64(intValue(client["expiryTime"]))),
		Enable:          types.BoolValue(boolValue(client["enable"])),
		TgID:            types.Int64Value(int64(intValue(client["tgId"]))),
		SubID:           types.StringValue(stringValue(client["subId"])),
		Comment:         types.StringValue(stringValue(client["comment"])),
		Reset:           types.Int64Value(int64(intValue(client["reset"]))),
		Group:           stringValueOrNull(stringValue(client["group"])),
		ResetDay:        types.Int64Value(int64(intValue(client["resetDay"]))),
		ResetMax:        types.Int64Value(int64(intValue(client["resetMax"]))),
		ResetWeekday:    types.Int64Value(int64(intValue(client["resetWeekday"]))),
		TrafficReset:    stringValueOrNull(stringValue(client["trafficReset"])),
		TrafficResetDay: types.Int64Value(int64(intValue(client["trafficResetDay"]))),
		Secret:          stringValueOrNull(stringValue(client["secret"])),
		AdTag:           stringValueOrNull(stringValue(client["adTag"])),
	}
}

func reverseTagValue(raw any) string {
	m, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	return stringValue(m["tag"])
}

func getClientIDFromModel(m *InboundClientResourceModel, client map[string]any) string {
	if !m.ClientID.IsNull() && !m.ClientID.IsUnknown() && m.ClientID.ValueString() != "" {
		return m.ClientID.ValueString()
	}
	if v := stringValue(client["id"]); v != "" {
		return v
	}
	if v := stringValue(client["password"]); v != "" {
		return v
	}
	if v := stringValue(client["auth"]); v != "" {
		return v
	}
	return ""
}

// ---------------------------------------------------------------------------
// Helper types and functions
// ---------------------------------------------------------------------------

type inboundSettings struct {
	Clients []map[string]any `json:"clients"`
}

func parseInboundSettings(settings string) (*inboundSettings, error) {
	if strings.TrimSpace(settings) == "" {
		return &inboundSettings{}, nil
	}
	var out inboundSettings
	if err := json.Unmarshal([]byte(settings), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ensureInboundClientsKey writes an empty settings.clients array via
// UpdateInbound when the key is absent, so the legacy client endpoints have
// an array to append to.
//
// Since 3x-ui v3.9.0 the UpdateInbound call below is a silent no-op for the
// clients array (keepStoredClients replaces posted clients with the stored
// ones — 3x-ui-3.9.0/internal/web/service/inbound.go:1851-1862). It is left
// in place rather than gated: it only fires when the settings carry no
// clients key at all, which the v3.1.0+ /panel/api/clients/add endpoint does
// not require (its service layer marshals {"clients": [...]} into the payload
// itself), so the wasted no-op update is harmless, and the client has no
// cheap panel-version signal to gate on — its probes distinguish API-surface
// layouts, not versions.
func ensureInboundClientsKey(ctx context.Context, client *Client, inboundID int) error {
	if client == nil || inboundID == 0 {
		return nil
	}
	inbound, err := client.GetInbound(ctx, inboundID)
	if err != nil {
		return err
	}
	settings, err := ParseJSONField(inbound.Settings)
	if err != nil {
		return err
	}
	if settings == nil {
		settings = map[string]any{}
	}
	if _, ok := settings["clients"]; ok {
		return nil
	}
	settings["clients"] = []any{}
	updated, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	inbound.Settings = string(updated)
	_, err = client.UpdateInbound(ctx, inbound)
	return err
}

func findClientByID(clients []map[string]any, clientID string) map[string]any {
	for _, client := range clients {
		if stringValue(client["id"]) == clientID || stringValue(client["password"]) == clientID || stringValue(client["auth"]) == clientID || stringValue(client["email"]) == clientID {
			return client
		}
	}
	return nil
}

func makeInboundClientID(inboundID int, clientID string) string {
	return fmt.Sprintf("%d:%s", inboundID, clientID)
}

func newSubID() (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = chars[b[i]%byte(len(chars))]
	}
	return string(b), nil
}

func newUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	), nil
}

func splitInboundClientID(id string) (int, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid inbound_client id: %s", id)
	}
	inboundID, err := strconv.Atoi(parts[0])
	if err != nil || inboundID == 0 {
		return 0, "", fmt.Errorf("invalid inbound id: %s", id)
	}
	if parts[1] == "" {
		return 0, "", fmt.Errorf("invalid client id: %s", id)
	}
	return inboundID, parts[1], nil
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func stringValueOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func intValue(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case float32:
		return int(val)
	default:
		return 0
	}
}

// int64Value is intValue without the platform-width narrowing. `int` is 32 bits
// on the 386 and arm release targets, so intValue silently wraps values above
// 2^31-1 there; use this for any field the upstream schema models as a 64-bit
// integer. Note that a value decoded from JSON arrives as float64 and so is
// still only exact up to 2^53 — this removes the overflow, not that ceiling.
func int64Value(v any) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int64:
		return val
	case float64:
		return int64(val)
	case float32:
		return int64(val)
	default:
		return 0
	}
}

func boolValue(v any) bool {
	b, _ := v.(bool)
	return b
}
