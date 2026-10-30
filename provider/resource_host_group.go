package provider

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// Typed API struct (mirrors 3x-ui v3.5.0 entity.HostGroup)
// ---------------------------------------------------------------------------

// HostGroup mirrors 3x-ui v3.5.0's entity.HostGroup (internal/web/entity/entity.go).
// All fields use camelCase JSON keys matching the upstream struct tags.
type HostGroup struct {
	GroupId                string   `json:"groupId"`
	InboundIds             []int    `json:"inboundIds"`
	Hosts                  []string `json:"hosts"`
	SortOrder              int      `json:"sortOrder"`
	Remark                 string   `json:"remark"`
	ServerDescription      string   `json:"serverDescription"`
	IsDisabled             bool     `json:"isDisabled"`
	IsHidden               bool     `json:"isHidden"`
	Tags                   []string `json:"tags"`
	Port                   int      `json:"port"`
	Security               string   `json:"security"`
	Sni                    string   `json:"sni"`
	Snis                   []string `json:"snis"`
	SniSharedRandom        bool     `json:"sniSharedRandom"`
	HostHeader             string   `json:"hostHeader"`
	Path                   string   `json:"path"`
	Alpn                   []string `json:"alpn"`
	Fingerprint            string   `json:"fingerprint"`
	CipherSuites           string   `json:"cipherSuites"`
	OverrideSniFromAddress bool     `json:"overrideSniFromAddress"`
	KeepSniBlank           bool     `json:"keepSniBlank"`
	PinnedPeerCertSha256   []string `json:"pinnedPeerCertSha256"`
	VerifyPeerCertByName   string   `json:"verifyPeerCertByName"`
	AllowInsecure          bool     `json:"allowInsecure"`
	EchConfigList          string   `json:"echConfigList"`
	MuxParams              string   `json:"muxParams"`
	SockoptParams          string   `json:"sockoptParams"`
	FinalMask              string   `json:"finalMask"`
	VlessRoute             string   `json:"vlessRoute"`
	ExcludeFromSubTypes    []string `json:"excludeFromSubTypes"`
	NodeGuids              []string `json:"nodeGuids"`
	MihomoIpVersion        string   `json:"mihomoIpVersion"`
	MihomoX25519           bool     `json:"mihomoX25519"`
	ShuffleHost            bool     `json:"shuffleHost"`
	RandomCount            int      `json:"randomCount"`
}

// ---------------------------------------------------------------------------
// Typed Terraform model
// ---------------------------------------------------------------------------

// HostGroupResourceModel is the Terraform-level typed model for
// threexui_host_group. Managed attributes map 1:1 to entity.HostGroup; snake_case
// tfsdk tags convert to camelCase JSON keys via hostGroupFromModel /
// flattenHostGroupToModel.
type HostGroupResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	GroupID                types.String `tfsdk:"group_id"`
	InboundIDs             types.List   `tfsdk:"inbound_ids"` // list of int64
	Hosts                  types.List   `tfsdk:"hosts"`       // list of string
	SortOrder              types.Int64  `tfsdk:"sort_order"`
	Remark                 types.String `tfsdk:"remark"`
	ServerDescription      types.String `tfsdk:"server_description"`
	IsDisabled             types.Bool   `tfsdk:"is_disabled"`
	IsHidden               types.Bool   `tfsdk:"is_hidden"`
	Tags                   types.List   `tfsdk:"tags"` // list of string
	Port                   types.Int64  `tfsdk:"port"`
	Security               types.String `tfsdk:"security"`
	Sni                    types.String `tfsdk:"sni"`
	Snis                   types.List   `tfsdk:"snis"` // list of string
	SniSharedRandom        types.Bool   `tfsdk:"sni_shared_random"`
	HostHeader             types.String `tfsdk:"host_header"`
	Path                   types.String `tfsdk:"path"`
	Alpn                   types.List   `tfsdk:"alpn"` // list of string
	Fingerprint            types.String `tfsdk:"fingerprint"`
	CipherSuites           types.String `tfsdk:"cipher_suites"`
	OverrideSniFromAddress types.Bool   `tfsdk:"override_sni_from_address"`
	KeepSniBlank           types.Bool   `tfsdk:"keep_sni_blank"`
	PinnedPeerCertSha256   types.List   `tfsdk:"pinned_peer_cert_sha256"` // list of string
	VerifyPeerCertByName   types.String `tfsdk:"verify_peer_cert_by_name"`
	AllowInsecure          types.Bool   `tfsdk:"allow_insecure"`
	EchConfigList          types.String `tfsdk:"ech_config_list"`
	MuxParams              types.String `tfsdk:"mux_params"`
	SockoptParams          types.String `tfsdk:"sockopt_params"`
	FinalMask              types.String `tfsdk:"final_mask"`
	VlessRoute             types.String `tfsdk:"vless_route"`
	ExcludeFromSubTypes    types.List   `tfsdk:"exclude_from_sub_types"` // list of string
	NodeGuids              types.List   `tfsdk:"node_guids"`             // list of string
	MihomoIpVersion        types.String `tfsdk:"mihomo_ip_version"`
	MihomoX25519           types.Bool   `tfsdk:"mihomo_x25519"`
	ShuffleHost            types.Bool   `tfsdk:"shuffle_host"`
	RandomCount            types.Int64  `tfsdk:"random_count"`
}

// ---------------------------------------------------------------------------
// Resource
// ---------------------------------------------------------------------------

type HostGroupResource struct {
	client *Client
}

func NewHostGroupResource() resource.Resource {
	return &HostGroupResource{}
}

func (r *HostGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_group"
}

func (r *HostGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = hostGroupResourceSchema()
}

func (r *HostGroupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan, config HostGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !config.Sni.IsNull() {
		switch {
		case !config.Snis.IsNull() && !config.Snis.IsUnknown():
			if !hostGroupPoolHasUnknown(config.Snis) {
				configured := stringListToSlice(ctx, config.Snis)
				canonical := canonicalHostSniPool(config.Sni.ValueString(), configured)
				if !reflect.DeepEqual(canonical, configured) {
					resp.Diagnostics.AddError("Invalid SNI pool", "The configured sni and snis are not in the panel's canonical form: the primary SNI must be first and entries must be unique.")
				}
			}
		case config.Snis.IsNull() && !plan.Snis.IsNull() && !plan.Snis.IsUnknown() && !hostGroupPoolHasUnknown(plan.Snis):
			plan.Snis = sliceToStringList(canonicalHostSniPool(config.Sni.ValueString(), stringListToSlice(ctx, plan.Snis)))
		}
	} else if !config.Snis.IsNull() && !config.Snis.IsUnknown() {
		if hostGroupPoolHasUnknown(config.Snis) {
			plan.Sni = types.StringUnknown()
		} else if snis := stringListToSlice(ctx, config.Snis); len(snis) > 0 {
			plan.Sni = types.StringValue(snis[0])
		} else {
			plan.Sni = types.StringNull()
		}
	}

	if !config.SniSharedRandom.IsNull() && !config.SniSharedRandom.IsUnknown() && config.SniSharedRandom.ValueBool() &&
		!plan.Snis.IsNull() && !plan.Snis.IsUnknown() && !hostGroupPoolHasUnknown(plan.Snis) && len(plan.Snis.Elements()) < 2 {
		resp.Diagnostics.AddError("Invalid shared SNI randomization", "sni_shared_random requires at least two effective SNIs; the panel otherwise stores false.")
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *HostGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HostGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HostGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var config HostGroupResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateHostGroupPlanForApply(ctx, &plan, &config); err != nil {
		resp.Diagnostics.AddError("Invalid host group configuration", err.Error())
		return
	}
	hg := hostGroupFromModel(ctx, &plan)
	normalizeHostGroupSniPrimary(ctx, &plan, &config, hg)

	created, err := r.client.CreateHostGroup(ctx, hg)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create host group", err.Error())
		return
	}

	// The add endpoint generates the groupId server-side when omitted; re-read
	// to capture the canonical observed state (group_id + all fields).
	if created != nil && created.GroupId != "" {
		if got, getErr := r.client.GetHostGroup(ctx, created.GroupId); getErr == nil && got != nil {
			created = got
		}
	}

	plan.ID = types.StringValue(created.GroupId)
	plan.GroupID = types.StringValue(created.GroupId)
	flattenHostGroupToModel(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HostGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HostGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := state.GroupID.ValueString()
	if groupID == "" {
		groupID = state.ID.ValueString()
	}
	if groupID == "" {
		resp.Diagnostics.AddError("Invalid host group id", "group_id is empty")
		return
	}

	got, err := r.client.GetHostGroup(ctx, groupID)
	if err != nil {
		// The panel signals a missing host group with HTTP 200 + success:false
		// carrying a gorm "record not found" message, not HTTP 404. Treat that
		// as out-of-band deletion and drop the resource from state (same as nodes).
		if isAPIRecordNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read host group", err.Error())
		return
	}
	if got == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	flattenHostGroupToModel(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HostGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan HostGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state HostGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var config HostGroupResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := state.GroupID.ValueString()
	if groupID == "" {
		groupID = state.ID.ValueString()
	}
	if groupID == "" {
		resp.Diagnostics.AddError("Invalid host group id", "group_id is empty")
		return
	}

	if err := validateHostGroupPlanForApply(ctx, &plan, &config); err != nil {
		resp.Diagnostics.AddError("Invalid host group configuration", err.Error())
		return
	}
	hg := hostGroupFromModel(ctx, &plan)
	normalizeHostGroupSniPrimary(ctx, &plan, &config, hg)
	hg.GroupId = groupID

	if err := r.client.UpdateHostGroup(ctx, groupID, hg); err != nil {
		resp.Diagnostics.AddError("Failed to update host group", err.Error())
		return
	}

	// The update handler does a delete-then-recreate under a transaction;
	// re-read to refresh observed state and detect drift.
	got, getErr := r.client.GetHostGroup(ctx, groupID)
	if getErr != nil {
		if isAPIRecordNotFound(getErr) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddWarning("Failed to re-read host group after update", getErr.Error())
	} else if got != nil {
		flattenHostGroupToModel(got, &plan)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HostGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HostGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := state.GroupID.ValueString()
	if groupID == "" {
		groupID = state.ID.ValueString()
	}
	if groupID == "" {
		resp.Diagnostics.AddError("Invalid host group id", "group_id is empty")
		return
	}

	if err := r.client.DeleteHostGroup(ctx, groupID); err != nil {
		// Already gone out-of-band (either gorm record-not-found or the panel's
		// explicit "host group not found" sentinel); treat as deleted.
		if isAPIRecordNotFound(err) || isHostGroupNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Failed to delete host group", err.Error())
		return
	}
}

func (r *HostGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// Expand / Flatten
// ---------------------------------------------------------------------------

// hostGroupFromModel builds the API HostGroup from the managed attributes of
// the model. The GroupId is left empty on Create so the server generates it;
// Update sets it before calling (see HostGroupResource.Update).
func hostGroupFromModel(ctx context.Context, m *HostGroupResourceModel) *HostGroup {
	hg := &HostGroup{
		GroupId:                m.GroupID.ValueString(),
		Remark:                 m.Remark.ValueString(),
		ServerDescription:      m.ServerDescription.ValueString(),
		SortOrder:              int(m.SortOrder.ValueInt64()),
		Port:                   int(m.Port.ValueInt64()),
		Security:               m.Security.ValueString(),
		Sni:                    m.Sni.ValueString(),
		SniSharedRandom:        m.SniSharedRandom.ValueBool(),
		HostHeader:             m.HostHeader.ValueString(),
		Path:                   m.Path.ValueString(),
		Fingerprint:            m.Fingerprint.ValueString(),
		CipherSuites:           m.CipherSuites.ValueString(),
		OverrideSniFromAddress: m.OverrideSniFromAddress.ValueBool(),
		KeepSniBlank:           m.KeepSniBlank.ValueBool(),
		VerifyPeerCertByName:   m.VerifyPeerCertByName.ValueString(),
		AllowInsecure:          m.AllowInsecure.ValueBool(),
		EchConfigList:          m.EchConfigList.ValueString(),
		MuxParams:              m.MuxParams.ValueString(),
		SockoptParams:          m.SockoptParams.ValueString(),
		FinalMask:              m.FinalMask.ValueString(),
		VlessRoute:             m.VlessRoute.ValueString(),
		MihomoIpVersion:        m.MihomoIpVersion.ValueString(),
		MihomoX25519:           m.MihomoX25519.ValueBool(),
		ShuffleHost:            m.ShuffleHost.ValueBool(),
		RandomCount:            int(m.RandomCount.ValueInt64()),
	}
	if !m.IsDisabled.IsNull() {
		hg.IsDisabled = m.IsDisabled.ValueBool()
	}
	if !m.IsHidden.IsNull() {
		hg.IsHidden = m.IsHidden.ValueBool()
	}
	hg.InboundIds = int64ListToInts(ctx, m.InboundIDs)
	hg.Hosts = stringListToSlice(ctx, m.Hosts)
	hg.Snis = stringListToSlice(ctx, m.Snis)
	hg.Tags = stringListToSlice(ctx, m.Tags)
	hg.Alpn = stringListToSlice(ctx, m.Alpn)
	hg.PinnedPeerCertSha256 = stringListToSlice(ctx, m.PinnedPeerCertSha256)
	hg.ExcludeFromSubTypes = stringListToSlice(ctx, m.ExcludeFromSubTypes)
	hg.NodeGuids = stringListToSlice(ctx, m.NodeGuids)
	return hg
}

func hostGroupPoolHasUnknown(list types.List) bool {
	if list.IsUnknown() {
		return true
	}
	for _, element := range list.Elements() {
		if element.IsUnknown() {
			return true
		}
	}
	return false
}

func canonicalHostSniPool(primary string, pool []string) []string {
	seen := make(map[string]bool, len(pool))
	snis := make([]string, 0, len(pool)+1)
	for _, value := range pool {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			snis = append(snis, value)
		}
	}
	primary = strings.TrimSpace(primary)
	if primary == "" || (len(snis) > 0 && snis[0] == primary) {
		return snis
	}
	if len(snis) <= 1 {
		return []string{primary}
	}
	out := []string{primary}
	for _, value := range snis {
		if value != primary {
			out = append(out, value)
		}
	}
	return out
}

func validateHostGroupPlanForApply(ctx context.Context, plan, config *HostGroupResourceModel) error {
	if plan == nil || config == nil {
		return nil
	}
	if !config.Sni.IsNull() && !plan.Sni.IsUnknown() && !plan.Snis.IsNull() && !hostGroupPoolHasUnknown(plan.Snis) {
		planned := stringListToSlice(ctx, plan.Snis)
		if canonical := canonicalHostSniPool(plan.Sni.ValueString(), planned); !reflect.DeepEqual(canonical, planned) {
			return fmt.Errorf("configured sni and snis are not in the panel's canonical form")
		}
	}
	if plan.SniSharedRandom.ValueBool() && !plan.Snis.IsNull() && !hostGroupPoolHasUnknown(plan.Snis) && len(plan.Snis.Elements()) < 2 {
		return fmt.Errorf("sni_shared_random requires at least two effective SNIs")
	}
	return nil
}

func normalizeHostGroupSniPrimary(ctx context.Context, plan, config *HostGroupResourceModel, hg *HostGroup) {
	if plan == nil || config == nil || !config.Sni.IsNull() || plan.Snis.IsNull() || hostGroupPoolHasUnknown(plan.Snis) || hg == nil {
		return
	}
	snis := stringListToSlice(ctx, plan.Snis)
	if len(snis) > 0 {
		hg.Sni = snis[0]
	} else {
		hg.Sni = ""
	}
}

type hostGroupSnisValidator struct{}

func (hostGroupSnisValidator) Description(_ context.Context) string {
	return "snis must contain unique, non-empty, trimmed values"
}

func (v hostGroupSnisValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (hostGroupSnisValidator) ValidateList(_ context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	seen := make(map[string]bool, len(req.ConfigValue.Elements()))
	for _, element := range req.ConfigValue.Elements() {
		sni, ok := element.(types.String)
		if !ok || sni.IsNull() || sni.IsUnknown() {
			continue
		}
		value := sni.ValueString()
		if value == "" || strings.TrimSpace(value) != value || seen[value] {
			resp.Diagnostics.AddError("Invalid SNI pool", "Each SNI must be non-empty, trimmed, and unique.")
			return
		}
		seen[value] = true
	}
}

// flattenHostGroupToModel copies a HostGroup into the model. Managed attributes
// are overwritten from the remote so drift is detected.
func flattenHostGroupToModel(hg *HostGroup, m *HostGroupResourceModel) {
	if hg == nil {
		return
	}
	m.GroupID = types.StringValue(hg.GroupId)
	m.ID = types.StringValue(hg.GroupId)
	m.Remark = types.StringValue(hg.Remark)
	if hg.ServerDescription != "" {
		m.ServerDescription = types.StringValue(hg.ServerDescription)
	} else {
		m.ServerDescription = types.StringNull()
	}
	m.SortOrder = types.Int64Value(int64(hg.SortOrder))
	m.IsDisabled = types.BoolValue(hg.IsDisabled)
	m.IsHidden = types.BoolValue(hg.IsHidden)
	m.Port = types.Int64Value(int64(hg.Port))
	if hg.Security != "" {
		m.Security = types.StringValue(hg.Security)
	} else {
		m.Security = types.StringNull()
	}
	if hg.Sni != "" {
		m.Sni = types.StringValue(hg.Sni)
	} else {
		m.Sni = types.StringNull()
	}
	m.SniSharedRandom = types.BoolValue(hg.SniSharedRandom)
	if hg.HostHeader != "" {
		m.HostHeader = types.StringValue(hg.HostHeader)
	} else {
		m.HostHeader = types.StringNull()
	}
	if hg.Path != "" {
		m.Path = types.StringValue(hg.Path)
	} else {
		m.Path = types.StringNull()
	}
	if hg.Fingerprint != "" {
		m.Fingerprint = types.StringValue(hg.Fingerprint)
	} else {
		m.Fingerprint = types.StringNull()
	}
	if hg.CipherSuites != "" {
		m.CipherSuites = types.StringValue(hg.CipherSuites)
	} else {
		m.CipherSuites = types.StringNull()
	}
	m.OverrideSniFromAddress = types.BoolValue(hg.OverrideSniFromAddress)
	m.KeepSniBlank = types.BoolValue(hg.KeepSniBlank)
	if hg.VerifyPeerCertByName != "" {
		m.VerifyPeerCertByName = types.StringValue(hg.VerifyPeerCertByName)
	} else {
		m.VerifyPeerCertByName = types.StringNull()
	}
	m.AllowInsecure = types.BoolValue(hg.AllowInsecure)
	if hg.EchConfigList != "" {
		m.EchConfigList = types.StringValue(hg.EchConfigList)
	} else {
		m.EchConfigList = types.StringNull()
	}
	if hg.MuxParams != "" {
		m.MuxParams = types.StringValue(hg.MuxParams)
	} else {
		m.MuxParams = types.StringNull()
	}
	if hg.SockoptParams != "" {
		m.SockoptParams = types.StringValue(hg.SockoptParams)
	} else {
		m.SockoptParams = types.StringNull()
	}
	if hg.FinalMask != "" {
		m.FinalMask = types.StringValue(hg.FinalMask)
	} else {
		m.FinalMask = types.StringNull()
	}
	if hg.VlessRoute != "" {
		m.VlessRoute = types.StringValue(hg.VlessRoute)
	} else {
		m.VlessRoute = types.StringNull()
	}
	if hg.MihomoIpVersion != "" {
		m.MihomoIpVersion = types.StringValue(hg.MihomoIpVersion)
	} else {
		m.MihomoIpVersion = types.StringNull()
	}
	m.MihomoX25519 = types.BoolValue(hg.MihomoX25519)
	m.ShuffleHost = types.BoolValue(hg.ShuffleHost)
	if hg.RandomCount > 0 {
		m.RandomCount = types.Int64Value(int64(hg.RandomCount))
	} else {
		m.RandomCount = types.Int64Null()
	}

	m.InboundIDs = intsToInt64List(hg.InboundIds)
	m.Hosts = sliceToStringList(hg.Hosts)
	m.Snis = sliceToStringList(hg.Snis)
	m.Tags = sliceToStringList(hg.Tags)
	m.Alpn = sliceToStringList(hg.Alpn)
	m.PinnedPeerCertSha256 = sliceToStringList(hg.PinnedPeerCertSha256)
	m.ExcludeFromSubTypes = sliceToStringList(hg.ExcludeFromSubTypes)
	m.NodeGuids = sliceToStringList(hg.NodeGuids)
}

// int64ListToInts extracts []int from a types.List of int64 values.
func int64ListToInts(ctx context.Context, l types.List) []int {
	if l.IsNull() || l.IsUnknown() {
		return nil
	}
	var out []int64
	if diags := l.ElementsAs(ctx, &out, false); diags.HasError() {
		return nil
	}
	result := make([]int, 0, len(out))
	for _, v := range out {
		result = append(result, int(v))
	}
	return result
}

// stringListToSlice extracts []string from a types.List of string values.
func stringListToSlice(ctx context.Context, l types.List) []string {
	if l.IsNull() || l.IsUnknown() {
		return nil
	}
	var out []string
	if diags := l.ElementsAs(ctx, &out, false); diags.HasError() {
		return nil
	}
	return out
}

// intsToInt64List builds a types.List of int64 from a Go []int.
func intsToInt64List(in []int) types.List {
	if len(in) == 0 {
		return types.ListValueMust(types.Int64Type, nil)
	}
	vals := make([]attr.Value, 0, len(in))
	for _, v := range in {
		vals = append(vals, types.Int64Value(int64(v)))
	}
	return types.ListValueMust(types.Int64Type, vals)
}

// sliceToStringList builds a types.List of string from a Go []string.
func sliceToStringList(in []string) types.List {
	if len(in) == 0 {
		return types.ListValueMust(types.StringType, nil)
	}
	vals := make([]attr.Value, 0, len(in))
	for _, v := range in {
		vals = append(vals, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, vals)
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

func hostGroupResourceSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages a 3x-ui host group (bulk host management, /panel/api/hosts/*). " +
			"Requires 3x-ui v3.5.0+. On older panels the /panel/api/hosts/* routes " +
			"do not exist and operations will fail — gate this resource behind a v3.5.0+ panel.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Host group identifier. If omitted on create, the panel generates one " +
					"(random 16-digit numeric string). Once set (server-generated or explicit), " +
					"changing it forces a replace.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"inbound_ids": schema.ListAttribute{
				Required:    true,
				ElementType: types.Int64Type,
				Description: "Inbound ids this host group applies to. At least one is required.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"remark": schema.StringAttribute{
				Required:    true,
				Description: "Remark / display name (max 256 chars).",
			},
			"server_description": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Server description shown to clients (max 64 chars).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sort_order": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description: "Sort order for display.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"is_disabled": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_hidden": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"port": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description: "Override port for the generated share links (0–65535).",
				Validators: []validator.Int64{
					int64validator.Between(0, 65535),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"security": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Link security scheme.",
				Validators: []validator.String{
					stringvalidator.OneOf("same", "tls", "none", "reality"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sni": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"snis": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Pool of unique, trimmed SNIs for generated links. Requires 3x-ui v3.11.0+; [] clears the pool.",
				Validators:  []validator.List{hostGroupSnisValidator{}},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"sni_shared_random": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Pick one SNI per subscription fetch for the whole host group instead of per generated config. Requires at least two effective SNIs and 3x-ui v3.11.0+.",
			},
			"host_header": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"fingerprint": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cipher_suites": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "TLS cipher suites for the generated share links (3x-ui v3.9.0+). " +
					"Empty means the panel default.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"override_sni_from_address": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"keep_sni_blank": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"verify_peer_cert_by_name": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_insecure": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"ech_config_list": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mux_params": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sockopt_params": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"final_mask": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vless_route": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mihomo_ip_version": schema.StringAttribute{
				Optional: true, Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf("dual", "ipv4", "ipv6", "ipv4-prefer", "ipv6-prefer"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mihomo_x25519": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"shuffle_host": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"random_count": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Number of unique random addresses emitted for a CIDR host (1-100). Requires 3x-ui v3.11.0+; older panels read it back as null.",
				Validators: []validator.Int64{
					int64validator.Between(1, 100),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"hosts": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Explicit host list for this group.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"tags": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"alpn": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"pinned_peer_cert_sha256": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"exclude_from_sub_types": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"node_guids": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}
