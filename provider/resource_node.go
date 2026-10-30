package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &NodeResource{}
	_ resource.ResourceWithConfigure   = &NodeResource{}
	_ resource.ResourceWithImportState = &NodeResource{}
	_ resource.ResourceWithModifyPlan  = &NodeResource{}
)

type NodeResource struct {
	client *Client
}

func NewNodeResource() resource.Resource {
	return &NodeResource{}
}

func (r *NodeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_node"
}

func (r *NodeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = nodeResourceSchema()
}

func (r *NodeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan marks the plain secret attributes (api_token, pinned_cert_sha256)
// as Unknown when their *_wo_version trigger changes, so Terraform accepts the
// new sensitive value from Apply instead of rejecting it as "inconsistent
// values for sensitive" (the write-only value is read from req.Config and
// copied into the plain field by resolveNodeSecretsWO at Apply time).
//
// Inlined rather than calling the shared modifyPlanWOVersion generic twice:
// that generic re-reads req.Plan and overwrites the whole resp.Plan on each
// call, so a second call would erase the Unknown mark set by the first
// (simultaneous rotation of both secrets). threexui_node is the first
// resource with two write-only secrets; the settings resources have one each
// and use the generic directly.
func (r *NodeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var planAPITokenVersion, stateAPITokenVersion types.Int64
	var planPinnedCertVersion, statePinnedCertVersion types.Int64
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_token_wo_version"), &planAPITokenVersion)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("api_token_wo_version"), &stateAPITokenVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("pinned_cert_sha256_wo_version"), &planPinnedCertVersion)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("pinned_cert_sha256_wo_version"), &statePinnedCertVersion)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if woVersionTriggered(planAPITokenVersion, stateAPITokenVersion) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("api_token"), types.StringUnknown())...)
	}
	if woVersionTriggered(planPinnedCertVersion, statePinnedCertVersion) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("pinned_cert_sha256"), types.StringUnknown())...)
	}
}

func (r *NodeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NodeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config NodeResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveNodeSecretsWO(&plan, config)

	in := nodeFromModel(&plan)

	created, err := r.client.CreateNode(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create cluster node", err.Error()+
			"\n\nNote: the central panel probes the node for reachability (ensureReachable) "+
			"before persisting it. Ensure the node's web API is reachable from the central panel.")
		return
	}

	// The add endpoint echoes back the bound model.Node, but observed state
	// (guid, status, ...) is populated by heartbeat probes. Re-read to get
	// the freshest view, tolerating the not-yet-probed case.
	if created != nil && created.Id != 0 {
		if got, getErr := r.client.GetNode(ctx, created.Id); getErr == nil && got != nil {
			created = got
		}
	}

	plan.ID = types.StringValue(strconv.Itoa(created.Id))
	flattenNodeToModel(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *NodeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NodeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseNodeID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid node id", err.Error())
		return
	}

	got, err := r.client.GetNode(ctx, id)
	if err != nil {
		// The panel signals a missing node with HTTP 200 + success:false
		// carrying a gorm "record not found" message, not HTTP 404
		// (controller/node.go get → jsonMsg). Treat that as out-of-band
		// deletion and drop the resource from state.
		if isAPIRecordNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read cluster node", err.Error())
		return
	}
	if got == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	flattenNodeToModel(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies in-place changes to a cluster node via form-POST
// /panel/api/nodes/update/:id. The 3x-ui controller restarts the Xray core
// itself when outbound_tag changes (controller/node.go:180-181) and runs
// ensureReachable, so this method does NOT call RestartXrayService.
func (r *NodeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NodeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NodeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var config NodeResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveNodeSecretsWOUpdate(&plan, state, config)

	id, err := parseNodeID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid node id", err.Error())
		return
	}

	if err := r.client.UpdateNode(ctx, id, nodeFromModel(&plan)); err != nil {
		hint := ""
		if strings.Contains(strings.ToLower(err.Error()), "unreachable") {
			hint = "\n\nNote: the central panel probes the node for reachability (ensureReachable) during update. Ensure the node's web API is reachable from the central panel."
		}
		resp.Diagnostics.AddError("Failed to update cluster node", err.Error()+hint)
		return
	}

	// The update handler returns only a status message (no object), so re-read
	// to refresh observed state and detect drift.
	got, getErr := r.client.GetNode(ctx, id)
	if getErr != nil {
		if isAPIRecordNotFound(getErr) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddWarning("Failed to re-read node after update", getErr.Error())
	} else if got != nil {
		flattenNodeToModel(got, &plan)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete unregisters a cluster node from the central panel (POST
// /panel/api/nodes/del/:id). 3x-ui refuses to delete a node that still owns
// inbounds (DB-002, #314 R3); in that case the operator must detach/delete
// the inbounds first. Other transient failures return an error; on success
// the resource is removed from state.
func (r *NodeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NodeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseNodeID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid node id", err.Error())
		return
	}

	if err := r.client.DeleteNode(ctx, id); err != nil {
		if isAPIRecordNotFound(err) {
			// Already gone out-of-band; treat as deleted.
			return
		}
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "inbound") || strings.Contains(strings.ToLower(msg), "attached") {
			msg += "\n\n3x-ui refuses to delete a node that still owns inbounds (DB-002). " +
				"Detach or delete the inbounds referencing this node first, then re-run terraform destroy."
		}
		resp.Diagnostics.AddError("Failed to delete cluster node", msg)
		return
	}
}

func (r *NodeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// parseNodeID parses the stringified numeric node id.
func parseNodeID(id string) (int, error) {
	var parsed int
	if _, err := fmt.Sscanf(id, "%d", &parsed); err != nil || parsed == 0 {
		return 0, fmt.Errorf("invalid node id: %s", id)
	}
	return parsed, nil
}

// resolveNodeSecretsWO copies the write-only secret values from config into
// the plain fields before Create. Write-only values live only in req.Config
// (the framework nulls them in plan/state), so without this the panel would
// never receive the new secret on first apply. The settings-style strategy
// applies because the panel returns pinnedCertSha256 raw (#314 R1) and flatten
// reads it back, which would otherwise clash with a null plan value.
// apiToken is write-only since v3.6.0 (#5613) and is never echoed back, but
// the same preserve-on-empty logic handles it.
func resolveNodeSecretsWO(plan *NodeResourceModel, config NodeResourceModel) {
	if !config.ApiTokenWO.IsNull() && !config.ApiTokenWO.IsUnknown() {
		plan.ApiToken = config.ApiTokenWO
	}
	if !config.PinnedCertSha256WO.IsNull() && !config.PinnedCertSha256WO.IsUnknown() {
		plan.PinnedCertSha256 = config.PinnedCertSha256WO
	}
}

// resolveNodeSecretsWOUpdate copies the write-only secret into the plain field
// on Update only when the *_wo_version trigger changed (version bump, or first
// use when state has none). This avoids resending the secret on every update.
func resolveNodeSecretsWOUpdate(plan *NodeResourceModel, state, config NodeResourceModel) {
	if !config.ApiTokenWO.IsNull() && !config.ApiTokenWO.IsUnknown() &&
		woVersionTriggered(plan.ApiTokenWOVersion, state.ApiTokenWOVersion) {
		plan.ApiToken = config.ApiTokenWO
	}
	if !config.PinnedCertSha256WO.IsNull() && !config.PinnedCertSha256WO.IsUnknown() &&
		woVersionTriggered(plan.PinnedCertSha256WOVersion, state.PinnedCertSha256WOVersion) {
		plan.PinnedCertSha256 = config.PinnedCertSha256WO
	}
}

// nodeFromModel builds the API Node from the managed attributes of the model.
func nodeFromModel(m *NodeResourceModel) *Node {
	n := &Node{
		Name:                m.Name.ValueString(),
		Remark:              m.Remark.ValueString(),
		Scheme:              m.Scheme.ValueString(),
		Address:             m.Address.ValueString(),
		Port:                int(m.Port.ValueInt64()),
		BasePath:            m.BasePath.ValueString(),
		ApiToken:            m.ApiToken.ValueString(),
		Enable:              true,
		AllowPrivateAddress: m.AllowPrivateAddress.ValueBool(),
		TlsVerifyMode:       m.TlsVerifyMode.ValueString(),
		PinnedCertSha256:    m.PinnedCertSha256.ValueString(),
		InboundSyncMode:     m.InboundSyncMode.ValueString(),
		OutboundTag:         m.OutboundTag.ValueString(),
	}
	if !m.Enable.IsNull() {
		n.Enable = m.Enable.ValueBool()
	}
	for _, t := range m.InboundTags {
		n.InboundTags = append(n.InboundTags, t.ValueString())
	}
	return n
}

// flattenNodeToModel copies a Node into the model (managed + observed state).
// Managed attributes are overwritten from the remote so that drift is detected;
// sensitive managed attributes (api_token, pinned_cert_sha256) are preserved
// from the current model when the remote returns them empty. Since 3x-ui v3.6.0
// (#5613) the apiToken is write-only and always empty on read; the preserve-
// on-empty logic means the state value from Create/Update survives. An empty
// pinned_cert_sha256 also means "unset".
func flattenNodeToModel(n *Node, m *NodeResourceModel) {
	if n == nil {
		return
	}
	// Note: *_wo_version fields are intentionally NOT overwritten here — flatten
	// mutates the existing model in place (Create/Update pass &plan), so the
	// planned wo_version survives into state. (The settings resources instead
	// build a fresh model in flatten* and must call preserveWOVersion; node does
	// not because it mutates in place.)
	m.Name = types.StringValue(n.Name)
	if n.Remark != "" {
		m.Remark = types.StringValue(n.Remark)
	}
	if n.Scheme != "" {
		m.Scheme = types.StringValue(n.Scheme)
	}
	m.Address = types.StringValue(n.Address)
	m.Port = types.Int64Value(int64(n.Port))
	if n.BasePath != "" {
		m.BasePath = types.StringValue(n.BasePath)
	}
	if n.ApiToken != "" {
		m.ApiToken = types.StringValue(n.ApiToken)
	}
	m.Enable = types.BoolValue(n.Enable)
	m.AllowPrivateAddress = types.BoolValue(n.AllowPrivateAddress)
	if n.TlsVerifyMode != "" {
		m.TlsVerifyMode = types.StringValue(n.TlsVerifyMode)
	}
	if n.PinnedCertSha256 != "" {
		m.PinnedCertSha256 = types.StringValue(n.PinnedCertSha256)
	}
	if n.InboundSyncMode != "" {
		m.InboundSyncMode = types.StringValue(n.InboundSyncMode)
	}
	if n.InboundTags != nil {
		tags := make([]types.String, 0, len(n.InboundTags))
		for _, t := range n.InboundTags {
			tags = append(tags, types.StringValue(t))
		}
		m.InboundTags = tags
	}
	m.OutboundTag = types.StringValue(n.OutboundTag)

	// Observed state.
	m.Guid = types.StringValue(n.Guid)
	m.Status = types.StringValue(n.Status)
	m.LastHeartbeat = types.Int64Value(n.LastHeartbeat)
	m.LatencyMs = types.Int64Value(int64(n.LatencyMs))
	m.XrayVersion = types.StringValue(n.XrayVersion)
	m.PanelVersion = types.StringValue(n.PanelVersion)
	m.CpuPct = types.Float64Value(n.CpuPct)
	m.MemPct = types.Float64Value(n.MemPct)
	// Traffic/uptime come from the panel as uint64; clamping to int64 is safe in
	// practice (net bytes < 9.2 EB, uptime < ~292 years never overflow int64).
	m.UptimeSecs = types.Int64Value(int64(n.UptimeSecs)) //nolint:gosec // G115: see comment above
	m.UnavailableMetrics = sliceToStringList(n.UnavailableMetrics)
	m.NetUp = types.Int64Value(int64(n.NetUp))     //nolint:gosec // G115: see comment above
	m.NetDown = types.Int64Value(int64(n.NetDown)) //nolint:gosec // G115: see comment above
	m.LastError = types.StringValue(n.LastError)
	m.XrayState = types.StringValue(n.XrayState)
	m.XrayError = types.StringValue(n.XrayError)
	m.ConfigDirty = types.BoolValue(n.ConfigDirty)
	m.ConfigDirtyAt = types.Int64Value(n.ConfigDirtyAt)
	m.InboundCount = types.Int64Value(int64(n.InboundCount))
	m.ClientCount = types.Int64Value(int64(n.ClientCount))
	m.OnlineCount = types.Int64Value(int64(n.OnlineCount))
	m.ActiveCount = types.Int64Value(int64(n.ActiveCount))
	m.DisabledCount = types.Int64Value(int64(n.DisabledCount))
	m.DepletedCount = types.Int64Value(int64(n.DepletedCount))
	m.ParentGuid = types.StringValue(n.ParentGuid)
	m.Transitive = types.BoolValue(n.Transitive)
	m.CreatedAt = types.Int64Value(n.CreatedAt)
	m.UpdatedAt = types.Int64Value(n.UpdatedAt)
}
