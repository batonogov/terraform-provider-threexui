package provider

import (
	"context"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// Panel Security model, schema, expand/flatten
// ---------------------------------------------------------------------------

type PanelSecurityModel struct {
	ID                      types.String `tfsdk:"id"`
	TwoFactorEnable         types.Bool   `tfsdk:"two_factor_enable"`
	TwoFactorToken          types.String `tfsdk:"two_factor_token"`
	TwoFactorTokenWO        types.String `tfsdk:"two_factor_token_wo"`
	TwoFactorTokenWOVersion types.Int64  `tfsdk:"two_factor_token_wo_version"`
}

func panelSecuritySchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"two_factor_enable": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"two_factor_token": schema.StringAttribute{
				Optional: true, Computed: true, Sensitive: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.PreferWriteOnlyAttribute(path.MatchRoot("two_factor_token_wo")),
				},
			},
			"two_factor_token_wo": schema.StringAttribute{
				Optional:  true,
				WriteOnly: true,
			},
			"two_factor_token_wo_version": schema.Int64Attribute{
				Optional: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.AlsoRequires(path.MatchRoot("two_factor_token_wo")),
				},
			},
		},
	}
}

func expandPanelSecurity(m *PanelSecurityModel) map[string]any {
	payload := map[string]any{}
	if !m.TwoFactorEnable.IsNull() && !m.TwoFactorEnable.IsUnknown() {
		payload["twoFactorEnable"] = m.TwoFactorEnable.ValueBool()
	}
	if !m.TwoFactorTokenWO.IsNull() && !m.TwoFactorTokenWO.IsUnknown() {
		payload["twoFactorToken"] = m.TwoFactorTokenWO.ValueString()
	} else if !m.TwoFactorToken.IsNull() && !m.TwoFactorToken.IsUnknown() {
		payload["twoFactorToken"] = m.TwoFactorToken.ValueString()
	}
	return payload
}

func flattenPanelSecurity(in map[string]any) *PanelSecurityModel {
	m := &PanelSecurityModel{
		ID: types.StringValue("settings"),
	}
	if v, ok := in["twoFactorEnable"]; ok {
		m.TwoFactorEnable = types.BoolValue(boolValue(v))
	}
	if v, ok := in["twoFactorToken"]; ok {
		m.TwoFactorToken = types.StringValue(stringValue(v))
	}
	return m
}

// ---------------------------------------------------------------------------
// Panel Telegram model, schema, expand/flatten
// ---------------------------------------------------------------------------

type PanelTelegramModel struct {
	ID                  types.String `tfsdk:"id"`
	TgBotEnable         types.Bool   `tfsdk:"tg_bot_enable"`
	TgBotToken          types.String `tfsdk:"tg_bot_token"`
	TgBotTokenWO        types.String `tfsdk:"tg_bot_token_wo"`
	TgBotTokenWOVersion types.Int64  `tfsdk:"tg_bot_token_wo_version"`
	TgBotProxy          types.String `tfsdk:"tg_bot_proxy"`
	TgBotAPIServer      types.String `tfsdk:"tg_bot_api_server"`
	TgBotChatID         types.String `tfsdk:"tg_bot_chat_id"`
	TgLang              types.String `tfsdk:"tg_lang"`
	TgRunTime           types.String `tfsdk:"tg_run_time"`
	TgBotBackup         types.Bool   `tfsdk:"tg_bot_backup"`
	TgBotLoginNotify    types.Bool   `tfsdk:"tg_bot_login_notify"`
	TgCPU               types.Int64  `tfsdk:"tg_cpu"`
	TgEnabledEvents     types.String `tfsdk:"tg_enabled_events"`
	TgMemory            types.Int64  `tfsdk:"tg_memory"`
}

func panelTelegramSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tg_bot_enable": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"tg_bot_token": schema.StringAttribute{
				Optional: true, Computed: true, Sensitive: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.PreferWriteOnlyAttribute(path.MatchRoot("tg_bot_token_wo")),
				},
			},
			"tg_bot_token_wo": schema.StringAttribute{
				Optional:  true,
				WriteOnly: true,
			},
			"tg_bot_token_wo_version": schema.Int64Attribute{
				Optional: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.AlsoRequires(path.MatchRoot("tg_bot_token_wo")),
				},
			},
			"tg_bot_proxy": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tg_bot_api_server": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tg_bot_chat_id": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tg_lang": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tg_run_time": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tg_bot_backup": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"tg_bot_login_notify": schema.BoolAttribute{
				Optional: true, Computed: true,
				DeprecationMessage: "Removed from 3x-ui v3.4.0; accepted but has no effect on v3.4.0+ panels.",
				PlanModifiers:      []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"tg_cpu": schema.Int64Attribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"tg_enabled_events": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Comma-separated event types to send via Telegram (e.g. login, backup, traffic threshold). Added in 3x-ui v3.4.0; ignored by older panels.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"tg_memory": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description:   "Memory usage threshold (%) for Telegram alerts (0-100). Added in 3x-ui v3.4.0; ignored by older panels.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func expandPanelTelegram(m *PanelTelegramModel) map[string]any {
	payload := map[string]any{}
	if !m.TgBotEnable.IsNull() && !m.TgBotEnable.IsUnknown() {
		payload["tgBotEnable"] = m.TgBotEnable.ValueBool()
	}
	if !m.TgBotTokenWO.IsNull() && !m.TgBotTokenWO.IsUnknown() {
		payload["tgBotToken"] = m.TgBotTokenWO.ValueString()
	} else if !m.TgBotToken.IsNull() && !m.TgBotToken.IsUnknown() {
		payload["tgBotToken"] = m.TgBotToken.ValueString()
	}
	if !m.TgBotProxy.IsNull() && !m.TgBotProxy.IsUnknown() {
		payload["tgBotProxy"] = m.TgBotProxy.ValueString()
	}
	if !m.TgBotAPIServer.IsNull() && !m.TgBotAPIServer.IsUnknown() {
		payload["tgBotAPIServer"] = m.TgBotAPIServer.ValueString()
	}
	if !m.TgBotChatID.IsNull() && !m.TgBotChatID.IsUnknown() {
		payload["tgBotChatId"] = m.TgBotChatID.ValueString()
	}
	if !m.TgLang.IsNull() && !m.TgLang.IsUnknown() {
		payload["tgLang"] = m.TgLang.ValueString()
	}
	if !m.TgRunTime.IsNull() && !m.TgRunTime.IsUnknown() {
		payload["tgRunTime"] = m.TgRunTime.ValueString()
	}
	if !m.TgBotBackup.IsNull() && !m.TgBotBackup.IsUnknown() {
		payload["tgBotBackup"] = m.TgBotBackup.ValueBool()
	}
	if !m.TgBotLoginNotify.IsNull() && !m.TgBotLoginNotify.IsUnknown() {
		payload["tgBotLoginNotify"] = m.TgBotLoginNotify.ValueBool()
	}
	if !m.TgCPU.IsNull() && !m.TgCPU.IsUnknown() {
		payload["tgCpu"] = int(m.TgCPU.ValueInt64())
	}
	if !m.TgEnabledEvents.IsNull() && !m.TgEnabledEvents.IsUnknown() {
		payload["tgEnabledEvents"] = m.TgEnabledEvents.ValueString()
	}
	if !m.TgMemory.IsNull() && !m.TgMemory.IsUnknown() {
		payload["tgMemory"] = int(m.TgMemory.ValueInt64())
	}
	return payload
}

func flattenPanelTelegram(in map[string]any) *PanelTelegramModel {
	m := &PanelTelegramModel{
		ID: types.StringValue("settings"),
	}
	if v, ok := in["tgBotEnable"]; ok {
		m.TgBotEnable = types.BoolValue(boolValue(v))
	}
	if v, ok := in["tgBotToken"]; ok {
		m.TgBotToken = types.StringValue(stringValue(v))
	}
	if v, ok := in["tgBotProxy"]; ok {
		m.TgBotProxy = types.StringValue(stringValue(v))
	}
	if v, ok := in["tgBotAPIServer"]; ok {
		m.TgBotAPIServer = types.StringValue(stringValue(v))
	}
	if v, ok := in["tgBotChatId"]; ok {
		m.TgBotChatID = types.StringValue(stringValue(v))
	}
	if v, ok := in["tgLang"]; ok {
		m.TgLang = types.StringValue(stringValue(v))
	}
	if v, ok := in["tgRunTime"]; ok {
		m.TgRunTime = types.StringValue(stringValue(v))
	}
	if v, ok := in["tgBotBackup"]; ok {
		m.TgBotBackup = types.BoolValue(boolValue(v))
	}
	if v, ok := in["tgBotLoginNotify"]; ok {
		m.TgBotLoginNotify = types.BoolValue(boolValue(v))
	}
	if v, ok := in["tgCpu"]; ok {
		m.TgCPU = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["tgEnabledEvents"]; ok {
		m.TgEnabledEvents = types.StringValue(stringValue(v))
	}
	if v, ok := in["tgMemory"]; ok {
		m.TgMemory = types.Int64Value(int64(intValue(v)))
	}
	return m
}

// ---------------------------------------------------------------------------
// Panel Email (SMTP notifications) model, schema, expand/flatten
// ---------------------------------------------------------------------------

// PanelEmailModel mirrors the SMTP notification fields in 3x-ui v3.4.0+ AllSetting.
// Older panels ignore these form values. smtp_password is a write-only secret
// (mirrors tg_bot_token_wo / panel_telegram).
type PanelEmailModel struct {
	ID                    types.String `tfsdk:"id"`
	SmtpEnable            types.Bool   `tfsdk:"smtp_enable"`
	SmtpHost              types.String `tfsdk:"smtp_host"`
	SmtpPort              types.Int64  `tfsdk:"smtp_port"`
	SmtpUsername          types.String `tfsdk:"smtp_username"`
	SmtpPassword          types.String `tfsdk:"smtp_password"`
	SmtpPasswordWO        types.String `tfsdk:"smtp_password_wo"`
	SmtpPasswordWOVersion types.Int64  `tfsdk:"smtp_password_wo_version"`
	SmtpTo                types.String `tfsdk:"smtp_to"`
	SmtpFrom              types.String `tfsdk:"smtp_from"`
	SmtpFromName          types.String `tfsdk:"smtp_from_name"`
	SmtpEncryptionType    types.String `tfsdk:"smtp_encryption_type"`
	SmtpEnabledEvents     types.String `tfsdk:"smtp_enabled_events"`
	SmtpCPU               types.Int64  `tfsdk:"smtp_cpu"`
	SmtpMemory            types.Int64  `tfsdk:"smtp_memory"`
}

func panelEmailSchema() schema.Schema {
	return schema.Schema{
		Description: "Manages the 3x-ui SMTP/email notification settings (3x-ui v3.4.0+). Older panels ignore these attributes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"smtp_enable": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Enable SMTP email notifications.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"smtp_host": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "SMTP server host.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"smtp_port": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description:   "SMTP server port (1-65535).",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"smtp_username": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "SMTP username.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"smtp_password": schema.StringAttribute{
				Optional: true, Computed: true, Sensitive: true,
				Description:   "SMTP password. Prefer smtp_password_wo on Terraform 1.11+ / OpenTofu 1.11+.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.PreferWriteOnlyAttribute(path.MatchRoot("smtp_password_wo")),
				},
			},
			"smtp_password_wo": schema.StringAttribute{
				Optional:    true,
				WriteOnly:   true,
				Description: "Write-only SMTP password (Terraform 1.11+ / OpenTofu 1.11+). Pair with smtp_password_wo_version to rotate.",
			},
			"smtp_password_wo_version": schema.Int64Attribute{
				Optional: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.AlsoRequires(path.MatchRoot("smtp_password_wo")),
				},
			},
			"smtp_to": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Comma-separated recipient email addresses.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"smtp_from": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "SMTP From address (RFC 5322). 3x-ui v3.6.0+.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"smtp_from_name": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "SMTP From display name. 3x-ui v3.6.0+.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"smtp_encryption_type": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "SMTP encryption: none, starttls, or tls.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"smtp_enabled_events": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Comma-separated event types to send via email (e.g. login, backup, traffic threshold).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"smtp_cpu": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description:   "CPU usage threshold (%) for email alerts (0-100).",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"smtp_memory": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description:   "Memory usage threshold (%) for email alerts (0-100).",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func expandPanelEmail(m *PanelEmailModel) map[string]any {
	payload := map[string]any{}
	if !m.SmtpEnable.IsNull() && !m.SmtpEnable.IsUnknown() {
		payload["smtpEnable"] = m.SmtpEnable.ValueBool()
	}
	if !m.SmtpHost.IsNull() && !m.SmtpHost.IsUnknown() {
		payload["smtpHost"] = m.SmtpHost.ValueString()
	}
	if !m.SmtpPort.IsNull() && !m.SmtpPort.IsUnknown() {
		payload["smtpPort"] = int(m.SmtpPort.ValueInt64())
	}
	if !m.SmtpUsername.IsNull() && !m.SmtpUsername.IsUnknown() {
		payload["smtpUsername"] = m.SmtpUsername.ValueString()
	}
	if !m.SmtpPasswordWO.IsNull() && !m.SmtpPasswordWO.IsUnknown() {
		payload["smtpPassword"] = m.SmtpPasswordWO.ValueString()
	} else if !m.SmtpPassword.IsNull() && !m.SmtpPassword.IsUnknown() {
		payload["smtpPassword"] = m.SmtpPassword.ValueString()
	}
	if !m.SmtpTo.IsNull() && !m.SmtpTo.IsUnknown() {
		payload["smtpTo"] = m.SmtpTo.ValueString()
	}
	if !m.SmtpFrom.IsNull() && !m.SmtpFrom.IsUnknown() {
		payload["smtpFrom"] = m.SmtpFrom.ValueString()
	}
	if !m.SmtpFromName.IsNull() && !m.SmtpFromName.IsUnknown() {
		payload["smtpFromName"] = m.SmtpFromName.ValueString()
	}
	if !m.SmtpEncryptionType.IsNull() && !m.SmtpEncryptionType.IsUnknown() {
		payload["smtpEncryptionType"] = m.SmtpEncryptionType.ValueString()
	}
	if !m.SmtpEnabledEvents.IsNull() && !m.SmtpEnabledEvents.IsUnknown() {
		payload["smtpEnabledEvents"] = m.SmtpEnabledEvents.ValueString()
	}
	if !m.SmtpCPU.IsNull() && !m.SmtpCPU.IsUnknown() {
		payload["smtpCpu"] = int(m.SmtpCPU.ValueInt64())
	}
	if !m.SmtpMemory.IsNull() && !m.SmtpMemory.IsUnknown() {
		payload["smtpMemory"] = int(m.SmtpMemory.ValueInt64())
	}
	return payload
}

func flattenPanelEmail(in map[string]any) *PanelEmailModel {
	m := &PanelEmailModel{
		ID: types.StringValue("settings"),
	}
	if v, ok := in["smtpEnable"]; ok {
		m.SmtpEnable = types.BoolValue(boolValue(v))
	}
	if v, ok := in["smtpHost"]; ok {
		m.SmtpHost = types.StringValue(stringValue(v))
	}
	if v, ok := in["smtpPort"]; ok {
		m.SmtpPort = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["smtpUsername"]; ok {
		m.SmtpUsername = types.StringValue(stringValue(v))
	}
	if v, ok := in["smtpPassword"]; ok {
		m.SmtpPassword = types.StringValue(stringValue(v))
	}
	if v, ok := in["smtpTo"]; ok {
		m.SmtpTo = types.StringValue(stringValue(v))
	}
	if v, ok := in["smtpFrom"]; ok {
		m.SmtpFrom = types.StringValue(stringValue(v))
	}
	if v, ok := in["smtpFromName"]; ok {
		m.SmtpFromName = types.StringValue(stringValue(v))
	}
	if v, ok := in["smtpEncryptionType"]; ok {
		m.SmtpEncryptionType = types.StringValue(stringValue(v))
	}
	if v, ok := in["smtpEnabledEvents"]; ok {
		m.SmtpEnabledEvents = types.StringValue(stringValue(v))
	}
	if v, ok := in["smtpCpu"]; ok {
		m.SmtpCPU = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["smtpMemory"]; ok {
		m.SmtpMemory = types.Int64Value(int64(intValue(v)))
	}
	return m
}

// ---------------------------------------------------------------------------
// Panel Discord (Discord notification bot) model, schema, expand/flatten
// ---------------------------------------------------------------------------

// PanelDiscordModel mirrors the Discord notification fields in 3x-ui v3.8.0+
// AllSetting (3x-ui #6486). The fields do not exist on older panels: reads
// surface nulls and writes are silently dropped by gin form binding, so the
// resource is effectively v3.8.0+ only. discord_bot_token is a write-only
// secret (mirrors tg_bot_token_wo / panel_telegram).
type PanelDiscordModel struct {
	ID                   types.String `tfsdk:"id"`
	DiscordBotEnable     types.Bool   `tfsdk:"discord_bot_enable"`
	DiscordBotToken      types.String `tfsdk:"discord_bot_token"`
	DiscordBotTokenWO    types.String `tfsdk:"discord_bot_token_wo"`
	DiscordBotTokenWOVer types.Int64  `tfsdk:"discord_bot_token_wo_version"`
	DiscordChannelID     types.String `tfsdk:"discord_channel_id"`
	DiscordAdminIDs      types.String `tfsdk:"discord_admin_ids"`
	DiscordRunTime       types.String `tfsdk:"discord_run_time"`
	DiscordBotBackup     types.Bool   `tfsdk:"discord_bot_backup"`
	DiscordCPU           types.Int64  `tfsdk:"discord_cpu"`
	DiscordMemory        types.Int64  `tfsdk:"discord_memory"`
	DiscordLang          types.String `tfsdk:"discord_lang"`
	DiscordEnabledEvents types.String `tfsdk:"discord_enabled_events"`
}

func panelDiscordSchema() schema.Schema {
	return schema.Schema{
		Description: "Discord notification bot settings (3x-ui v3.8.0+; the Discord settings tab). On older panels reads return nulls and writes have no effect.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"discord_bot_enable": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Enable the Discord notification bot. Requires 3x-ui v3.8.0+.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"discord_bot_token": schema.StringAttribute{
				Optional: true, Computed: true, Sensitive: true,
				Description: "Discord bot token. Requires 3x-ui v3.8.0+.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.PreferWriteOnlyAttribute(path.MatchRoot("discord_bot_token_wo")),
				},
			},
			"discord_bot_token_wo": schema.StringAttribute{
				Optional:  true,
				WriteOnly: true,
				Description: "Write-only Discord bot token (Terraform/OpenTofu 1.11+). " +
					"Use discord_bot_token_wo_version to rotate.",
			},
			"discord_bot_token_wo_version": schema.Int64Attribute{
				Optional: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.AlsoRequires(path.MatchRoot("discord_bot_token_wo")),
				},
			},
			"discord_channel_id": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Discord channel ID the bot posts to. Requires 3x-ui v3.8.0+.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"discord_admin_ids": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Comma-separated Discord admin/user IDs allowed to use bot commands. Requires 3x-ui v3.8.0+.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"discord_run_time": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Cron schedule for the periodic stats report (e.g. @daily). Requires 3x-ui v3.8.0+.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"discord_bot_backup": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Send database backups through the Discord bot. Requires 3x-ui v3.8.0+.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"discord_cpu": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description:   "CPU usage threshold (%) for Discord alerts (0-100). Requires 3x-ui v3.8.0+.",
				Validators:    []validator.Int64{int64validator.Between(0, 100)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"discord_memory": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description:   "Memory usage threshold (%) for Discord alerts (0-100). Requires 3x-ui v3.8.0+.",
				Validators:    []validator.Int64{int64validator.Between(0, 100)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"discord_lang": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Discord bot language. Requires 3x-ui v3.8.0+.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"discord_enabled_events": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "Comma-separated event types to send via Discord (e.g. login, backup, cpu.high, memory.high). " +
					"Requires 3x-ui v3.8.0+.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func expandPanelDiscord(m *PanelDiscordModel) map[string]any {
	payload := map[string]any{}
	if !m.DiscordBotEnable.IsNull() && !m.DiscordBotEnable.IsUnknown() {
		payload["discordBotEnable"] = m.DiscordBotEnable.ValueBool()
	}
	if !m.DiscordBotTokenWO.IsNull() && !m.DiscordBotTokenWO.IsUnknown() {
		payload["discordBotToken"] = m.DiscordBotTokenWO.ValueString()
	} else if !m.DiscordBotToken.IsNull() && !m.DiscordBotToken.IsUnknown() {
		payload["discordBotToken"] = m.DiscordBotToken.ValueString()
	}
	if !m.DiscordChannelID.IsNull() && !m.DiscordChannelID.IsUnknown() {
		payload["discordChannelId"] = m.DiscordChannelID.ValueString()
	}
	if !m.DiscordAdminIDs.IsNull() && !m.DiscordAdminIDs.IsUnknown() {
		payload["discordAdminIds"] = m.DiscordAdminIDs.ValueString()
	}
	if !m.DiscordRunTime.IsNull() && !m.DiscordRunTime.IsUnknown() {
		payload["discordRunTime"] = m.DiscordRunTime.ValueString()
	}
	if !m.DiscordBotBackup.IsNull() && !m.DiscordBotBackup.IsUnknown() {
		payload["discordBotBackup"] = m.DiscordBotBackup.ValueBool()
	}
	if !m.DiscordCPU.IsNull() && !m.DiscordCPU.IsUnknown() {
		payload["discordCpu"] = int(m.DiscordCPU.ValueInt64())
	}
	if !m.DiscordMemory.IsNull() && !m.DiscordMemory.IsUnknown() {
		payload["discordMemory"] = int(m.DiscordMemory.ValueInt64())
	}
	if !m.DiscordLang.IsNull() && !m.DiscordLang.IsUnknown() {
		payload["discordLang"] = m.DiscordLang.ValueString()
	}
	if !m.DiscordEnabledEvents.IsNull() && !m.DiscordEnabledEvents.IsUnknown() {
		payload["discordEnabledEvents"] = m.DiscordEnabledEvents.ValueString()
	}
	return payload
}

func flattenPanelDiscord(in map[string]any) *PanelDiscordModel {
	m := &PanelDiscordModel{
		ID: types.StringValue("settings"),
	}
	if v, ok := in["discordBotEnable"]; ok {
		m.DiscordBotEnable = types.BoolValue(boolValue(v))
	}
	if v, ok := in["discordBotToken"]; ok {
		m.DiscordBotToken = types.StringValue(stringValue(v))
	}
	if v, ok := in["discordChannelId"]; ok {
		m.DiscordChannelID = types.StringValue(stringValue(v))
	}
	if v, ok := in["discordAdminIds"]; ok {
		m.DiscordAdminIDs = types.StringValue(stringValue(v))
	}
	if v, ok := in["discordRunTime"]; ok {
		m.DiscordRunTime = types.StringValue(stringValue(v))
	}
	if v, ok := in["discordBotBackup"]; ok {
		m.DiscordBotBackup = types.BoolValue(boolValue(v))
	}
	if v, ok := in["discordCpu"]; ok {
		m.DiscordCPU = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["discordMemory"]; ok {
		m.DiscordMemory = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["discordLang"]; ok {
		m.DiscordLang = types.StringValue(stringValue(v))
	}
	if v, ok := in["discordEnabledEvents"]; ok {
		m.DiscordEnabledEvents = types.StringValue(stringValue(v))
	}
	return m
}

type PanelSubscriptionModel struct {
	ID                     types.String `tfsdk:"id"`
	SubEnable              types.Bool   `tfsdk:"sub_enable"`
	SubJsonEnable          types.Bool   `tfsdk:"sub_json_enable"`
	SubTitle               types.String `tfsdk:"sub_title"`
	SubSupportURL          types.String `tfsdk:"sub_support_url"`
	SubProfileURL          types.String `tfsdk:"sub_profile_url"`
	SubAnnounce            types.String `tfsdk:"sub_announce"`
	SubEnableRouting       types.Bool   `tfsdk:"sub_enable_routing"`
	SubRoutingRules        types.String `tfsdk:"sub_routing_rules"`
	SubListen              types.String `tfsdk:"sub_listen"`
	SubPort                types.Int64  `tfsdk:"sub_port"`
	SubPath                types.String `tfsdk:"sub_path"`
	SubDomain              types.String `tfsdk:"sub_domain"`
	SubCertFile            types.String `tfsdk:"sub_cert_file"`
	SubKeyFile             types.String `tfsdk:"sub_key_file"`
	SubUpdates             types.Int64  `tfsdk:"sub_updates"`
	SubEncrypt             types.Bool   `tfsdk:"sub_encrypt"`
	SubShowInfo            types.Bool   `tfsdk:"sub_show_info"`
	SubEmailInRemark       types.Bool   `tfsdk:"sub_email_in_remark"`
	SubURI                 types.String `tfsdk:"sub_uri"`
	SubJsonPath            types.String `tfsdk:"sub_json_path"`
	SubJsonURI             types.String `tfsdk:"sub_json_uri"`
	SubJsonFragment        types.String `tfsdk:"sub_json_fragment"`
	SubJsonNoises          types.String `tfsdk:"sub_json_noises"`
	SubJsonMux             types.String `tfsdk:"sub_json_mux"`
	SubJsonRules           types.String `tfsdk:"sub_json_rules"`
	SubJsonObservatory     types.String `tfsdk:"sub_json_observatory"`
	SubJsonAutoDetect      types.Bool   `tfsdk:"sub_json_auto_detect"`
	SubJsonAlwaysArray     types.Bool   `tfsdk:"sub_json_always_array"`
	SubJsonUserAgentRegex  types.String `tfsdk:"sub_json_user_agent_regex"`
	SubClashEnable         types.Bool   `tfsdk:"sub_clash_enable"`
	SubClashPath           types.String `tfsdk:"sub_clash_path"`
	SubClashURI            types.String `tfsdk:"sub_clash_uri"`
	SubClashEnableRouting  types.Bool   `tfsdk:"sub_clash_enable_routing"`
	SubClashRules          types.String `tfsdk:"sub_clash_rules"`
	SubClashAutoDetect     types.Bool   `tfsdk:"sub_clash_auto_detect"`
	SubClashUserAgentRegex types.String `tfsdk:"sub_clash_user_agent_regex"`
	SubJsonFinalMask       types.String `tfsdk:"sub_json_final_mask"`
	SubThemeDir            types.String `tfsdk:"sub_theme_dir"`
	RemarkTemplate         types.String `tfsdk:"remark_template"`
	SubHideSettings        types.Bool   `tfsdk:"sub_hide_settings"`
	SubIncyEnableRouting   types.Bool   `tfsdk:"sub_incy_enable_routing"`
	SubIncyRoutingRules    types.String `tfsdk:"sub_incy_routing_rules"`

	// v3.8.0 subscription additions (profile page mode, page state templates,
	// JSON-subscription routing/DNS, Happ client customization). The subHapp*
	// family and subProfileMode/subJsonRoutingRules/subJsonDns are frozen into
	// the sub server at startup (initRouter) — see panelSettingsNeedRestart.
	SubProfileMode             types.String `tfsdk:"sub_profile_mode"`
	SubInfoNodeEnable          types.Bool   `tfsdk:"sub_info_node_enable"`
	SubCalendarExpireInclusive types.Bool   `tfsdk:"sub_calendar_expire_inclusive"`
	SubExpiredTemplate         types.String `tfsdk:"sub_expired_template"`
	SubTrafficDepletedTemplate types.String `tfsdk:"sub_traffic_depleted_template"`
	SubJsonRoutingRules        types.String `tfsdk:"sub_json_routing_rules"`
	SubJsonDns                 types.String `tfsdk:"sub_json_dns"`
	HappLinkEnable             types.Bool   `tfsdk:"happ_link_enable"`
	SubHappAutoDetect          types.Bool   `tfsdk:"sub_happ_auto_detect"`
	SubHappProviderId          types.String `tfsdk:"sub_happ_provider_id"`
	SubHappNewUrl              types.String `tfsdk:"sub_happ_new_url"`
	SubHappFallbackUrl         types.String `tfsdk:"sub_happ_fallback_url"`
	SubHappSubInfoColor        types.String `tfsdk:"sub_happ_sub_info_color"`
	SubHappSubInfoText         types.String `tfsdk:"sub_happ_sub_info_text"`
	SubHappSubInfoButtonText   types.String `tfsdk:"sub_happ_sub_info_button_text"`
	SubHappSubInfoButtonLink   types.String `tfsdk:"sub_happ_sub_info_button_link"`
	SubHappSubExpire           types.Bool   `tfsdk:"sub_happ_sub_expire"`
	SubHappSubExpireButtonLink types.String `tfsdk:"sub_happ_sub_expire_button_link"`
	SubHappNotificationExpire  types.Bool   `tfsdk:"sub_happ_notification_expire"`
	SubHappNoLimit             types.Bool   `tfsdk:"sub_happ_no_limit"`
	SubHappAlwaysHwid          types.Bool   `tfsdk:"sub_happ_always_hwid"`
	SubHappTunMode             types.String `tfsdk:"sub_happ_tun_mode"`
	SubHappTunType             types.String `tfsdk:"sub_happ_tun_type"`
	SubHappExcludeRoutes       types.String `tfsdk:"sub_happ_exclude_routes"`
	SubHappExcludeApns         types.Bool   `tfsdk:"sub_happ_exclude_apns"`
	SubHappColorProfile        types.String `tfsdk:"sub_happ_color_profile"`
	SubHappPingType            types.String `tfsdk:"sub_happ_ping_type"`
	SubHappAutoConnect         types.Bool   `tfsdk:"sub_happ_auto_connect"`
	SubHappAutoConnectType     types.String `tfsdk:"sub_happ_auto_connect_type"`
	SubHappPerAppMode          types.String `tfsdk:"sub_happ_per_app_mode"`
	SubHappPerAppList          types.String `tfsdk:"sub_happ_per_app_list"`
}

func panelSubscriptionSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages subscription settings in the 3x-ui panel.\n\n" +
			"~> **Note:** When `sub_port` (default `2096`) differs from the main panel port and the panel " +
			"runs behind a reverse proxy, the proxy must be configured to forward subscription path requests " +
			"to the subscription port. Without this, subscription URLs will return 404. " +
			"For example, in Caddy: `handle /sub/* { reverse_proxy 3x-ui:2096 }`. " +
			"In Nginx: `location /sub/ { proxy_pass http://3x-ui:2096; }`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sub_enable": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_json_enable": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_title": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_support_url": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_profile_url": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_announce": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_enable_routing": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_routing_rules": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_incy_enable_routing": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_incy_routing_rules": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_profile_mode": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Subscription profile page mode: none, builtin, or custom. The built-in profile page is off by default on v3.8.0+ (upstream #6538); an existing sub_profile_url maps to custom. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				Validators:    []validator.String{stringvalidator.OneOf("none", "builtin", "custom")},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_info_node_enable": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Add a dummy info/status node to client node lists. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_calendar_expire_inclusive": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Present expiry at month end instead of a rolling interval. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_expired_template": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Template for expired-client subscription pages. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_traffic_depleted_template": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Template for depleted-client subscription pages. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_json_routing_rules": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Client routing rules baked into JSON subscriptions. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_json_dns": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Panel-chosen DNS servers baked into JSON subscriptions. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"happ_link_enable": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Enable the Happ app link integration. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_auto_detect": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: auto-detect the client app. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_provider_id": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: provider ID. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_new_url": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: new-user URL. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_fallback_url": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: fallback URL. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_sub_info_color": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: subscription info color. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_sub_info_text": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: subscription info text. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_sub_info_button_text": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: subscription info button text. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_sub_info_button_link": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: subscription info button link. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_sub_expire": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: show subscription expiry. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_sub_expire_button_link": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: expiry button link. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_notification_expire": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: expiry notification. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_no_limit": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: no-limit profile. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_always_hwid": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: always use HWID limits. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_tun_mode": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: tun mode. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_tun_type": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: tun type. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_exclude_routes": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: excluded routes. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_exclude_apns": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: exclude APNs. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_color_profile": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: color profile. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_ping_type": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: connectivity-check ping type. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_auto_connect": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: auto-connect. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_auto_connect_type": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: auto-connect type. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_per_app_mode": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: per-app proxy mode. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_happ_per_app_list": schema.StringAttribute{
				Optional: true, Computed: true,
				Description:   "Happ: per-app list. Requires 3x-ui v3.8.0+; older panels ignore it and read it back empty.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_listen": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_port": schema.Int64Attribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"sub_path": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_domain": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_cert_file": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_key_file": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_updates": schema.Int64Attribute{
				Optional: true, Computed: true,
				Validators:    subUpdatesValidators(),
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"sub_encrypt": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_show_info": schema.BoolAttribute{
				Optional: true, Computed: true,
				DeprecationMessage: "Removed from 3x-ui v3.4.0; accepted but has no effect on v3.4.0+ panels.",
				PlanModifiers:      []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_email_in_remark": schema.BoolAttribute{
				Optional:           true,
				Computed:           true,
				Description:        "Include the client email in subscription profile names.",
				DeprecationMessage: "Removed from 3x-ui v3.4.0; accepted but has no effect on v3.4.0+ panels.",
				PlanModifiers:      []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_uri": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_json_path": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_json_uri": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_json_fragment": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "JSON fragment settings for subscription. " +
					"**v2.9.2+:** only the fragment parameters object, e.g. " +
					"`{\"packets\":\"tlshello\",\"length\":\"100-200\",\"interval\":\"10-20\"}`. " +
					"**v2.9.1 and earlier:** full outbound object with tag, protocol, settings and streamSettings. " +
					"Deprecated in 3x-ui v3.2.8 — replaced by sub_clash_enable_routing.",
				PlanModifiers:      []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				DeprecationMessage: "Deprecated in 3x-ui v3.2.8. Use sub_clash_enable_routing instead.",
			},
			"sub_json_noises": schema.StringAttribute{
				Optional: true, Computed: true,
				Description: "JSON noise settings for subscription. " +
					"**v2.9.2+:** only the noises array, e.g. " +
					"`[{\"type\":\"rand\",\"packet\":\"10-20\",\"delay\":\"10-16\"}]`. " +
					"**v2.9.1 and earlier:** full outbound object with tag, protocol, settings and streamSettings. " +
					"Deprecated in 3x-ui v3.2.8 — replaced by sub_clash_rules.",
				PlanModifiers:      []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				DeprecationMessage: "Deprecated in 3x-ui v3.2.8. Use sub_clash_rules instead.",
			},
			"sub_json_mux": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_json_rules": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_json_observatory": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Observatory block injected into the JSON subscription for client-side balancers, as a JSON string. 3x-ui v3.7.0+; older panels report an empty string (unsupported).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_json_auto_detect": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Auto-detect JSON subscription format by User-Agent. 3x-ui v3.6.0+.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_json_always_array": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Always output JSON subscription as an array. 3x-ui v3.6.0+.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_json_user_agent_regex": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "User-Agent regex for JSON subscription auto-detection. 3x-ui v3.6.0+.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_clash_enable": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Enable Clash/Mihomo subscription endpoint.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_clash_path": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Path for Clash/Mihomo subscription endpoint.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_clash_uri": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Clash/Mihomo subscription server URI.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_clash_enable_routing": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Enable global routing rules for Clash/Mihomo subscriptions (3x-ui v3.2.8+).",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_clash_rules": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Clash/Mihomo global routing rules (3x-ui v3.2.8+).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_clash_auto_detect": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Auto-detect Clash subscription format by User-Agent. 3x-ui v3.6.0+.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sub_clash_user_agent_regex": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "User-Agent regex for Clash subscription auto-detection. 3x-ui v3.6.0+.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_json_final_mask": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "JSON subscription global finalmask — TCP mask type (fragment/sudoku/header-custom/xmc), UDP masks and quicParams (3x-ui v3.2.8+; xmc added in v3.5.0).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_theme_dir": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Absolute path to a folder containing a custom subscription page template. Added in 3x-ui v3.3.0; ignored by older panels.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"remark_template": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Subscription remark template ({{VAR}} tokens rendered per client, e.g. Jalali date/transport/status). Added in 3x-ui v3.4.0; ignored by older panels.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_hide_settings": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Hide server settings in happ subscription (Happ only). Added in 3x-ui v3.4.0; ignored by older panels.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func expandPanelSubscription(m *PanelSubscriptionModel) map[string]any {
	payload := map[string]any{}
	if !m.SubEnable.IsNull() && !m.SubEnable.IsUnknown() {
		payload["subEnable"] = m.SubEnable.ValueBool()
	}
	if !m.SubJsonEnable.IsNull() && !m.SubJsonEnable.IsUnknown() {
		payload["subJsonEnable"] = m.SubJsonEnable.ValueBool()
	}
	if !m.SubTitle.IsNull() && !m.SubTitle.IsUnknown() {
		payload["subTitle"] = m.SubTitle.ValueString()
	}
	if !m.SubSupportURL.IsNull() && !m.SubSupportURL.IsUnknown() {
		payload["subSupportUrl"] = m.SubSupportURL.ValueString()
	}
	if !m.SubProfileURL.IsNull() && !m.SubProfileURL.IsUnknown() {
		payload["subProfileUrl"] = m.SubProfileURL.ValueString()
	}
	if !m.SubAnnounce.IsNull() && !m.SubAnnounce.IsUnknown() {
		payload["subAnnounce"] = m.SubAnnounce.ValueString()
	}
	if !m.SubEnableRouting.IsNull() && !m.SubEnableRouting.IsUnknown() {
		payload["subEnableRouting"] = m.SubEnableRouting.ValueBool()
	}
	if !m.SubRoutingRules.IsNull() && !m.SubRoutingRules.IsUnknown() {
		payload["subRoutingRules"] = m.SubRoutingRules.ValueString()
	}
	if !m.SubIncyEnableRouting.IsNull() && !m.SubIncyEnableRouting.IsUnknown() {
		payload["subIncyEnableRouting"] = m.SubIncyEnableRouting.ValueBool()
	}
	if !m.SubIncyRoutingRules.IsNull() && !m.SubIncyRoutingRules.IsUnknown() {
		payload["subIncyRoutingRules"] = m.SubIncyRoutingRules.ValueString()
	}
	if !m.SubProfileMode.IsNull() && !m.SubProfileMode.IsUnknown() {
		payload["subProfileMode"] = m.SubProfileMode.ValueString()
	}
	if !m.SubInfoNodeEnable.IsNull() && !m.SubInfoNodeEnable.IsUnknown() {
		payload["subInfoNodeEnable"] = m.SubInfoNodeEnable.ValueBool()
	}
	if !m.SubCalendarExpireInclusive.IsNull() && !m.SubCalendarExpireInclusive.IsUnknown() {
		payload["subCalendarExpireInclusive"] = m.SubCalendarExpireInclusive.ValueBool()
	}
	if !m.SubExpiredTemplate.IsNull() && !m.SubExpiredTemplate.IsUnknown() {
		payload["subExpiredTemplate"] = m.SubExpiredTemplate.ValueString()
	}
	if !m.SubTrafficDepletedTemplate.IsNull() && !m.SubTrafficDepletedTemplate.IsUnknown() {
		payload["subTrafficDepletedTemplate"] = m.SubTrafficDepletedTemplate.ValueString()
	}
	if !m.SubJsonRoutingRules.IsNull() && !m.SubJsonRoutingRules.IsUnknown() {
		payload["subJsonRoutingRules"] = m.SubJsonRoutingRules.ValueString()
	}
	if !m.SubJsonDns.IsNull() && !m.SubJsonDns.IsUnknown() {
		payload["subJsonDns"] = m.SubJsonDns.ValueString()
	}
	if !m.HappLinkEnable.IsNull() && !m.HappLinkEnable.IsUnknown() {
		payload["happLinkEnable"] = m.HappLinkEnable.ValueBool()
	}
	if !m.SubHappAutoDetect.IsNull() && !m.SubHappAutoDetect.IsUnknown() {
		payload["subHappAutoDetect"] = m.SubHappAutoDetect.ValueBool()
	}
	if !m.SubHappProviderId.IsNull() && !m.SubHappProviderId.IsUnknown() {
		payload["subHappProviderId"] = m.SubHappProviderId.ValueString()
	}
	if !m.SubHappNewUrl.IsNull() && !m.SubHappNewUrl.IsUnknown() {
		payload["subHappNewUrl"] = m.SubHappNewUrl.ValueString()
	}
	if !m.SubHappFallbackUrl.IsNull() && !m.SubHappFallbackUrl.IsUnknown() {
		payload["subHappFallbackUrl"] = m.SubHappFallbackUrl.ValueString()
	}
	if !m.SubHappSubInfoColor.IsNull() && !m.SubHappSubInfoColor.IsUnknown() {
		payload["subHappSubInfoColor"] = m.SubHappSubInfoColor.ValueString()
	}
	if !m.SubHappSubInfoText.IsNull() && !m.SubHappSubInfoText.IsUnknown() {
		payload["subHappSubInfoText"] = m.SubHappSubInfoText.ValueString()
	}
	if !m.SubHappSubInfoButtonText.IsNull() && !m.SubHappSubInfoButtonText.IsUnknown() {
		payload["subHappSubInfoButtonText"] = m.SubHappSubInfoButtonText.ValueString()
	}
	if !m.SubHappSubInfoButtonLink.IsNull() && !m.SubHappSubInfoButtonLink.IsUnknown() {
		payload["subHappSubInfoButtonLink"] = m.SubHappSubInfoButtonLink.ValueString()
	}
	if !m.SubHappSubExpire.IsNull() && !m.SubHappSubExpire.IsUnknown() {
		payload["subHappSubExpire"] = m.SubHappSubExpire.ValueBool()
	}
	if !m.SubHappSubExpireButtonLink.IsNull() && !m.SubHappSubExpireButtonLink.IsUnknown() {
		payload["subHappSubExpireButtonLink"] = m.SubHappSubExpireButtonLink.ValueString()
	}
	if !m.SubHappNotificationExpire.IsNull() && !m.SubHappNotificationExpire.IsUnknown() {
		payload["subHappNotificationExpire"] = m.SubHappNotificationExpire.ValueBool()
	}
	if !m.SubHappNoLimit.IsNull() && !m.SubHappNoLimit.IsUnknown() {
		payload["subHappNoLimit"] = m.SubHappNoLimit.ValueBool()
	}
	if !m.SubHappAlwaysHwid.IsNull() && !m.SubHappAlwaysHwid.IsUnknown() {
		payload["subHappAlwaysHwid"] = m.SubHappAlwaysHwid.ValueBool()
	}
	if !m.SubHappTunMode.IsNull() && !m.SubHappTunMode.IsUnknown() {
		payload["subHappTunMode"] = m.SubHappTunMode.ValueString()
	}
	if !m.SubHappTunType.IsNull() && !m.SubHappTunType.IsUnknown() {
		payload["subHappTunType"] = m.SubHappTunType.ValueString()
	}
	if !m.SubHappExcludeRoutes.IsNull() && !m.SubHappExcludeRoutes.IsUnknown() {
		payload["subHappExcludeRoutes"] = m.SubHappExcludeRoutes.ValueString()
	}
	if !m.SubHappExcludeApns.IsNull() && !m.SubHappExcludeApns.IsUnknown() {
		payload["subHappExcludeApns"] = m.SubHappExcludeApns.ValueBool()
	}
	if !m.SubHappColorProfile.IsNull() && !m.SubHappColorProfile.IsUnknown() {
		payload["subHappColorProfile"] = m.SubHappColorProfile.ValueString()
	}
	if !m.SubHappPingType.IsNull() && !m.SubHappPingType.IsUnknown() {
		payload["subHappPingType"] = m.SubHappPingType.ValueString()
	}
	if !m.SubHappAutoConnect.IsNull() && !m.SubHappAutoConnect.IsUnknown() {
		payload["subHappAutoConnect"] = m.SubHappAutoConnect.ValueBool()
	}
	if !m.SubHappAutoConnectType.IsNull() && !m.SubHappAutoConnectType.IsUnknown() {
		payload["subHappAutoConnectType"] = m.SubHappAutoConnectType.ValueString()
	}
	if !m.SubHappPerAppMode.IsNull() && !m.SubHappPerAppMode.IsUnknown() {
		payload["subHappPerAppMode"] = m.SubHappPerAppMode.ValueString()
	}
	if !m.SubHappPerAppList.IsNull() && !m.SubHappPerAppList.IsUnknown() {
		payload["subHappPerAppList"] = m.SubHappPerAppList.ValueString()
	}
	if !m.SubListen.IsNull() && !m.SubListen.IsUnknown() {
		payload["subListen"] = m.SubListen.ValueString()
	}
	if !m.SubPort.IsNull() && !m.SubPort.IsUnknown() {
		payload["subPort"] = int(m.SubPort.ValueInt64())
	}
	if !m.SubPath.IsNull() && !m.SubPath.IsUnknown() {
		payload["subPath"] = m.SubPath.ValueString()
	}
	if !m.SubDomain.IsNull() && !m.SubDomain.IsUnknown() {
		payload["subDomain"] = m.SubDomain.ValueString()
	}
	if !m.SubCertFile.IsNull() && !m.SubCertFile.IsUnknown() {
		payload["subCertFile"] = m.SubCertFile.ValueString()
	}
	if !m.SubKeyFile.IsNull() && !m.SubKeyFile.IsUnknown() {
		payload["subKeyFile"] = m.SubKeyFile.ValueString()
	}
	if !m.SubUpdates.IsNull() && !m.SubUpdates.IsUnknown() {
		payload["subUpdates"] = int(m.SubUpdates.ValueInt64())
	}
	if !m.SubEncrypt.IsNull() && !m.SubEncrypt.IsUnknown() {
		payload["subEncrypt"] = m.SubEncrypt.ValueBool()
	}
	if !m.SubShowInfo.IsNull() && !m.SubShowInfo.IsUnknown() {
		payload["subShowInfo"] = m.SubShowInfo.ValueBool()
	}
	if !m.SubEmailInRemark.IsNull() && !m.SubEmailInRemark.IsUnknown() {
		payload["subEmailInRemark"] = m.SubEmailInRemark.ValueBool()
	}
	if !m.SubURI.IsNull() && !m.SubURI.IsUnknown() {
		payload["subURI"] = m.SubURI.ValueString()
	}
	if !m.SubJsonPath.IsNull() && !m.SubJsonPath.IsUnknown() {
		payload["subJsonPath"] = m.SubJsonPath.ValueString()
	}
	if !m.SubJsonURI.IsNull() && !m.SubJsonURI.IsUnknown() {
		payload["subJsonURI"] = m.SubJsonURI.ValueString()
	}
	if !m.SubJsonFragment.IsNull() && !m.SubJsonFragment.IsUnknown() {
		payload["subJsonFragment"] = m.SubJsonFragment.ValueString()
	}
	if !m.SubJsonNoises.IsNull() && !m.SubJsonNoises.IsUnknown() {
		payload["subJsonNoises"] = m.SubJsonNoises.ValueString()
	}
	if !m.SubJsonMux.IsNull() && !m.SubJsonMux.IsUnknown() {
		payload["subJsonMux"] = m.SubJsonMux.ValueString()
	}
	if !m.SubJsonRules.IsNull() && !m.SubJsonRules.IsUnknown() {
		payload["subJsonRules"] = m.SubJsonRules.ValueString()
	}
	if !m.SubJsonObservatory.IsNull() && !m.SubJsonObservatory.IsUnknown() {
		payload["subJsonObservatory"] = m.SubJsonObservatory.ValueString()
	}
	if !m.SubJsonAutoDetect.IsNull() && !m.SubJsonAutoDetect.IsUnknown() {
		payload["subJsonAutoDetect"] = m.SubJsonAutoDetect.ValueBool()
	}
	if !m.SubJsonAlwaysArray.IsNull() && !m.SubJsonAlwaysArray.IsUnknown() {
		payload["subJsonAlwaysArray"] = m.SubJsonAlwaysArray.ValueBool()
	}
	if !m.SubJsonUserAgentRegex.IsNull() && !m.SubJsonUserAgentRegex.IsUnknown() {
		payload["subJsonUserAgentRegex"] = m.SubJsonUserAgentRegex.ValueString()
	}
	if !m.SubClashEnable.IsNull() && !m.SubClashEnable.IsUnknown() {
		payload["subClashEnable"] = m.SubClashEnable.ValueBool()
	}
	if !m.SubClashPath.IsNull() && !m.SubClashPath.IsUnknown() {
		payload["subClashPath"] = m.SubClashPath.ValueString()
	}
	if !m.SubClashURI.IsNull() && !m.SubClashURI.IsUnknown() {
		payload["subClashURI"] = m.SubClashURI.ValueString()
	}
	if !m.SubClashEnableRouting.IsNull() && !m.SubClashEnableRouting.IsUnknown() {
		payload["subClashEnableRouting"] = m.SubClashEnableRouting.ValueBool()
	}
	if !m.SubClashRules.IsNull() && !m.SubClashRules.IsUnknown() {
		payload["subClashRules"] = m.SubClashRules.ValueString()
	}
	if !m.SubClashAutoDetect.IsNull() && !m.SubClashAutoDetect.IsUnknown() {
		payload["subClashAutoDetect"] = m.SubClashAutoDetect.ValueBool()
	}
	if !m.SubClashUserAgentRegex.IsNull() && !m.SubClashUserAgentRegex.IsUnknown() {
		payload["subClashUserAgentRegex"] = m.SubClashUserAgentRegex.ValueString()
	}
	if !m.SubJsonFinalMask.IsNull() && !m.SubJsonFinalMask.IsUnknown() {
		payload["subJsonFinalMask"] = m.SubJsonFinalMask.ValueString()
	}
	if !m.SubThemeDir.IsNull() && !m.SubThemeDir.IsUnknown() {
		payload["subThemeDir"] = m.SubThemeDir.ValueString()
	}
	if !m.RemarkTemplate.IsNull() && !m.RemarkTemplate.IsUnknown() {
		payload["remarkTemplate"] = m.RemarkTemplate.ValueString()
	}
	if !m.SubHideSettings.IsNull() && !m.SubHideSettings.IsUnknown() {
		payload["subHideSettings"] = m.SubHideSettings.ValueBool()
	}
	return payload
}

func flattenPanelSubscription(in map[string]any) *PanelSubscriptionModel {
	m := &PanelSubscriptionModel{
		ID: types.StringValue("settings"),
	}
	if v, ok := in["subEnable"]; ok {
		m.SubEnable = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subJsonEnable"]; ok {
		m.SubJsonEnable = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subTitle"]; ok {
		m.SubTitle = types.StringValue(stringValue(v))
	}
	if v, ok := in["subSupportUrl"]; ok {
		m.SubSupportURL = types.StringValue(stringValue(v))
	}
	if v, ok := in["subProfileUrl"]; ok {
		m.SubProfileURL = types.StringValue(stringValue(v))
	}
	if v, ok := in["subAnnounce"]; ok {
		m.SubAnnounce = types.StringValue(stringValue(v))
	}
	if v, ok := in["subEnableRouting"]; ok {
		m.SubEnableRouting = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subRoutingRules"]; ok {
		m.SubRoutingRules = types.StringValue(stringValue(v))
	}
	if v, ok := in["subIncyEnableRouting"]; ok {
		m.SubIncyEnableRouting = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subIncyRoutingRules"]; ok {
		m.SubIncyRoutingRules = types.StringValue(stringValue(v))
	}
	if v, ok := in["subProfileMode"]; ok {
		m.SubProfileMode = types.StringValue(stringValue(v))
	}
	if v, ok := in["subInfoNodeEnable"]; ok {
		m.SubInfoNodeEnable = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subCalendarExpireInclusive"]; ok {
		m.SubCalendarExpireInclusive = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subExpiredTemplate"]; ok {
		m.SubExpiredTemplate = types.StringValue(stringValue(v))
	}
	if v, ok := in["subTrafficDepletedTemplate"]; ok {
		m.SubTrafficDepletedTemplate = types.StringValue(stringValue(v))
	}
	if v, ok := in["subJsonRoutingRules"]; ok {
		m.SubJsonRoutingRules = types.StringValue(stringValue(v))
	}
	if v, ok := in["subJsonDns"]; ok {
		m.SubJsonDns = types.StringValue(stringValue(v))
	}
	if v, ok := in["happLinkEnable"]; ok {
		m.HappLinkEnable = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subHappAutoDetect"]; ok {
		m.SubHappAutoDetect = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subHappProviderId"]; ok {
		m.SubHappProviderId = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappNewUrl"]; ok {
		m.SubHappNewUrl = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappFallbackUrl"]; ok {
		m.SubHappFallbackUrl = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappSubInfoColor"]; ok {
		m.SubHappSubInfoColor = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappSubInfoText"]; ok {
		m.SubHappSubInfoText = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappSubInfoButtonText"]; ok {
		m.SubHappSubInfoButtonText = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappSubInfoButtonLink"]; ok {
		m.SubHappSubInfoButtonLink = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappSubExpire"]; ok {
		m.SubHappSubExpire = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subHappSubExpireButtonLink"]; ok {
		m.SubHappSubExpireButtonLink = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappNotificationExpire"]; ok {
		m.SubHappNotificationExpire = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subHappNoLimit"]; ok {
		m.SubHappNoLimit = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subHappAlwaysHwid"]; ok {
		m.SubHappAlwaysHwid = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subHappTunMode"]; ok {
		m.SubHappTunMode = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappTunType"]; ok {
		m.SubHappTunType = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappExcludeRoutes"]; ok {
		m.SubHappExcludeRoutes = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappExcludeApns"]; ok {
		m.SubHappExcludeApns = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subHappColorProfile"]; ok {
		m.SubHappColorProfile = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappPingType"]; ok {
		m.SubHappPingType = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappAutoConnect"]; ok {
		m.SubHappAutoConnect = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subHappAutoConnectType"]; ok {
		m.SubHappAutoConnectType = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappPerAppMode"]; ok {
		m.SubHappPerAppMode = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHappPerAppList"]; ok {
		m.SubHappPerAppList = types.StringValue(stringValue(v))
	}
	if v, ok := in["subListen"]; ok {
		m.SubListen = types.StringValue(stringValue(v))
	}
	if v, ok := in["subPort"]; ok {
		m.SubPort = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["subPath"]; ok {
		m.SubPath = types.StringValue(stringValue(v))
	}
	if v, ok := in["subDomain"]; ok {
		m.SubDomain = types.StringValue(stringValue(v))
	}
	if v, ok := in["subCertFile"]; ok {
		m.SubCertFile = types.StringValue(stringValue(v))
	}
	if v, ok := in["subKeyFile"]; ok {
		m.SubKeyFile = types.StringValue(stringValue(v))
	}
	if v, ok := in["subUpdates"]; ok {
		m.SubUpdates = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["subEncrypt"]; ok {
		m.SubEncrypt = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subShowInfo"]; ok {
		m.SubShowInfo = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subEmailInRemark"]; ok {
		m.SubEmailInRemark = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subURI"]; ok {
		m.SubURI = types.StringValue(stringValue(v))
	}
	if v, ok := in["subJsonPath"]; ok {
		m.SubJsonPath = types.StringValue(stringValue(v))
	}
	if v, ok := in["subJsonURI"]; ok {
		m.SubJsonURI = types.StringValue(stringValue(v))
	}
	if v, ok := in["subJsonFragment"]; ok {
		m.SubJsonFragment = types.StringValue(stringValue(v))
	} else {
		m.SubJsonFragment = types.StringValue("")
	}
	if v, ok := in["subJsonNoises"]; ok {
		m.SubJsonNoises = types.StringValue(stringValue(v))
	} else {
		m.SubJsonNoises = types.StringValue("")
	}
	if v, ok := in["subJsonMux"]; ok {
		m.SubJsonMux = types.StringValue(stringValue(v))
	}
	if v, ok := in["subJsonRules"]; ok {
		m.SubJsonRules = types.StringValue(stringValue(v))
	}
	if v, ok := in["subJsonObservatory"]; ok {
		m.SubJsonObservatory = types.StringValue(stringValue(v))
	}
	if v, ok := in["subJsonAutoDetect"]; ok {
		m.SubJsonAutoDetect = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subJsonAlwaysArray"]; ok {
		m.SubJsonAlwaysArray = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subJsonUserAgentRegex"]; ok {
		m.SubJsonUserAgentRegex = types.StringValue(stringValue(v))
	}
	if v, ok := in["subClashEnable"]; ok {
		m.SubClashEnable = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subClashPath"]; ok {
		m.SubClashPath = types.StringValue(stringValue(v))
	} else {
		m.SubClashPath = types.StringNull()
	}
	if v, ok := in["subClashURI"]; ok {
		m.SubClashURI = types.StringValue(stringValue(v))
	} else {
		m.SubClashURI = types.StringNull()
	}
	if v, ok := in["subClashEnableRouting"]; ok {
		m.SubClashEnableRouting = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subClashRules"]; ok {
		m.SubClashRules = types.StringValue(stringValue(v))
	}
	if v, ok := in["subClashAutoDetect"]; ok {
		m.SubClashAutoDetect = types.BoolValue(boolValue(v))
	}
	if v, ok := in["subClashUserAgentRegex"]; ok {
		m.SubClashUserAgentRegex = types.StringValue(stringValue(v))
	}
	if v, ok := in["subJsonFinalMask"]; ok {
		m.SubJsonFinalMask = types.StringValue(stringValue(v))
	}
	if v, ok := in["subThemeDir"]; ok {
		m.SubThemeDir = types.StringValue(stringValue(v))
	}
	if v, ok := in["remarkTemplate"]; ok {
		m.RemarkTemplate = types.StringValue(stringValue(v))
	}
	if v, ok := in["subHideSettings"]; ok {
		m.SubHideSettings = types.BoolValue(boolValue(v))
	}
	return m
}

// modifyPlanWOVersion marks the plain secret attribute as Unknown during plan
// when the *_wo_version trigger changes (or is set for the first time). This
// tells Terraform to accept a new sensitive value from Apply instead of
// rejecting it as "inconsistent values for sensitive" — the prior-state value
// is otherwise carried forward by UseStateForUnknown and blocks the update.
//
// On no-op plans (version unchanged, no _wo in config) it does nothing.
func modifyPlanWOVersion[T any](
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
	woVersion func(T) types.Int64,
	setPlain func(*T, types.String),
) {
	if req.Plan.Raw.IsNull() {
		return
	}
	if req.State.Raw.IsNull() {
		return
	}

	var plan, state T
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !woVersionTriggered(woVersion(plan), woVersion(state)) {
		return
	}

	setPlain(&plan, types.StringUnknown())
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// woVersionTriggered reports whether the *_wo_version value represents a
// change relative to the prior state: the values differ, or the prior state
// has none while the plan introduces one.
func woVersionTriggered(plan, state types.Int64) bool {
	if plan.IsNull() || plan.IsUnknown() {
		return false
	}
	if state.IsNull() || state.IsUnknown() {
		return true
	}
	return plan.ValueInt64() != state.ValueInt64()
}

// ---------------------------------------------------------------------------
// Panel General model, schema, expand/flatten
// ---------------------------------------------------------------------------

type PanelGeneralModel struct {
	ID                          types.String `tfsdk:"id"`
	WebListen                   types.String `tfsdk:"web_listen"`
	WebDomain                   types.String `tfsdk:"web_domain"`
	WebPort                     types.Int64  `tfsdk:"web_port"`
	WebBasePath                 types.String `tfsdk:"web_base_path"`
	SessionMaxAge               types.Int64  `tfsdk:"session_max_age"`
	TrustedProxyCIDRs           types.String `tfsdk:"trusted_proxy_cidrs"`
	WarpUpdateInterval          types.Int64  `tfsdk:"warp_update_interval"`
	PageSize                    types.Int64  `tfsdk:"page_size"`
	RemarkModel                 types.String `tfsdk:"remark_model"`
	DatePicker                  types.String `tfsdk:"date_picker"`
	TimeLocation                types.String `tfsdk:"time_location"`
	ExpireDiff                  types.Int64  `tfsdk:"expire_diff"`
	TrafficDiff                 types.Int64  `tfsdk:"traffic_diff"`
	WebCertFile                 types.String `tfsdk:"web_cert_file"`
	WebKeyFile                  types.String `tfsdk:"web_key_file"`
	ExternalTrafficInformEnable types.Bool   `tfsdk:"external_traffic_inform_enable"`
	ExternalTrafficInformURI    types.String `tfsdk:"external_traffic_inform_uri"`
	SubShowIdentityOnAllLinks   types.Bool   `tfsdk:"sub_show_identity_on_all_links"`
	OutboundDownThreshold       types.Int64  `tfsdk:"outbound_down_threshold"`
	RestartXrayOnClientDisable  types.Bool   `tfsdk:"restart_xray_on_client_disable"`
	IPLimitAllowlist            types.String `tfsdk:"ip_limit_allowlist"`
	RealityScanCandidates       types.String `tfsdk:"reality_scan_candidates"`
	LDAPEnable                  types.Bool   `tfsdk:"ldap_enable"`
	LDAPHost                    types.String `tfsdk:"ldap_host"`
	LDAPPort                    types.Int64  `tfsdk:"ldap_port"`
	LDAPUseTLS                  types.Bool   `tfsdk:"ldap_use_tls"`
	LDAPInsecureSkipVerify      types.Bool   `tfsdk:"ldap_insecure_skip_verify"`
	LDAPBindDN                  types.String `tfsdk:"ldap_bind_dn"`
	LDAPPassword                types.String `tfsdk:"ldap_password"`
	LDAPPasswordWO              types.String `tfsdk:"ldap_password_wo"`
	LDAPPasswordWOVersion       types.Int64  `tfsdk:"ldap_password_wo_version"`
	LDAPBaseDN                  types.String `tfsdk:"ldap_base_dn"`
	LDAPUserFilter              types.String `tfsdk:"ldap_user_filter"`
	LDAPUserAttr                types.String `tfsdk:"ldap_user_attr"`
	LDAPVlessField              types.String `tfsdk:"ldap_vless_field"`
	LDAPSyncCron                types.String `tfsdk:"ldap_sync_cron"`
	LDAPFlagField               types.String `tfsdk:"ldap_flag_field"`
	LDAPTruthyValues            types.String `tfsdk:"ldap_truthy_values"`
	LDAPInvertFlag              types.Bool   `tfsdk:"ldap_invert_flag"`
	LDAPInboundTags             types.String `tfsdk:"ldap_inbound_tags"`
	LDAPAutoCreate              types.Bool   `tfsdk:"ldap_auto_create"`
	LDAPAutoDelete              types.Bool   `tfsdk:"ldap_auto_delete"`
	LDAPDefaultTotalGB          types.Int64  `tfsdk:"ldap_default_total_gb"`
	LDAPDefaultExpiryDays       types.Int64  `tfsdk:"ldap_default_expiry_days"`
	LDAPDefaultLimitIP          types.Int64  `tfsdk:"ldap_default_limit_ip"`
	XrayOutboundTestURL         types.String `tfsdk:"xray_outbound_test_url"`
	PanelProxy                  types.String `tfsdk:"panel_proxy"`
	PanelOutbound               types.String `tfsdk:"panel_outbound"`
}

func panelGeneralSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"web_listen": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"web_domain": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"web_port": schema.Int64Attribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"web_base_path": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"session_max_age": schema.Int64Attribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"trusted_proxy_cidrs": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Comma-separated trusted reverse proxy IPs/CIDRs used by 3x-ui when honoring " +
					"X-Forwarded-For, X-Forwarded-Host, and X-Real-IP headers.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"warp_update_interval": schema.Int64Attribute{
				Optional: true, Computed: true,
				Description:   "Interval (hours) between Cloudflare WARP / geo auto-updates via Xray-core (0 disables). Added in 3x-ui v3.3.0; ignored by older panels.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"page_size": schema.Int64Attribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"remark_model": schema.StringAttribute{
				Optional: true, Computed: true,
				DeprecationMessage: "Removed from 3x-ui v3.4.0 (superseded by remark_template). Use remark_template on v3.4.0+; accepted but has no effect on v3.4.0+ panels.",
				PlanModifiers:      []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"date_picker": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"time_location": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"expire_diff": schema.Int64Attribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"traffic_diff": schema.Int64Attribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"web_cert_file": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"web_key_file": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"external_traffic_inform_enable": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"external_traffic_inform_uri": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_show_identity_on_all_links": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Add identity tokens to every subscription link. 3x-ui v3.6.0+.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"outbound_down_threshold": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Description:   "Consecutive-failure threshold before outbound.down alert fires (1-100). 3x-ui v3.6.0+; older panels report 0 (unsupported).",
				Validators:    []validator.Int64{int64validator.Between(0, 100)},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"restart_xray_on_client_disable": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Restart Xray when clients are automatically disabled by expiry or traffic limit.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"ip_limit_allowlist": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Comma-separated addresses or CIDRs exempt from the per-client IP limit. " +
					"3x-ui v3.7.0+; older panels report an empty string (unsupported).",
				Validators: addrOrPrefixListValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"reality_scan_candidates": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "REALITY target scan candidates (comma-separated host:port entries the panel scans when picking a dest). " +
					"3x-ui v3.8.0+; older panels report an empty string (unsupported).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ldap_enable": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"ldap_host": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ldap_port": schema.Int64Attribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"ldap_use_tls": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"ldap_insecure_skip_verify": schema.BoolAttribute{
				Optional: true, Computed: true,
				Description:   "Skip verification of the LDAP server's TLS certificate (3x-ui v3.4.2+; ignored by older panels).",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"ldap_bind_dn": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ldap_password": schema.StringAttribute{
				Optional: true, Computed: true, Sensitive: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.PreferWriteOnlyAttribute(path.MatchRoot("ldap_password_wo")),
				},
			},
			"ldap_password_wo": schema.StringAttribute{
				Optional:  true,
				WriteOnly: true,
			},
			"ldap_password_wo_version": schema.Int64Attribute{
				Optional: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.AlsoRequires(path.MatchRoot("ldap_password_wo")),
				},
			},
			"ldap_base_dn": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ldap_user_filter": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ldap_user_attr": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ldap_vless_field": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ldap_sync_cron": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ldap_flag_field": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ldap_truthy_values": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ldap_invert_flag": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"ldap_inbound_tags": schema.StringAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"ldap_auto_create": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"ldap_auto_delete": schema.BoolAttribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"ldap_default_total_gb": schema.Int64Attribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"ldap_default_expiry_days": schema.Int64Attribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"ldap_default_limit_ip": schema.Int64Attribute{
				Optional: true, Computed: true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"xray_outbound_test_url": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "URL used for testing outbound connectivity (default: https://www.google.com/generate_204).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"panel_proxy": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "HTTP/SOCKS5 proxy URL for the panel's own outbound requests (xray version " +
					"checks, Telegram bot, outbound testing). Available on 3x-ui v3.2.0 through v3.3.0; " +
					"superseded by panel_outbound (outbound egress bridge) on v3.3.1+. " +
					"Ignored by v3.3.1+ panels.",
				PlanModifiers:      []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				DeprecationMessage: "Superseded by panel_outbound on 3x-ui v3.3.1+. Use panel_outbound for new configurations.",
			},
			"panel_outbound": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Xray outbound tag (or balancer tag) used for the panel's own outbound HTTP (update checks/downloads, Telegram, geo updates, outbound-subscription fetches). Available on 3x-ui v3.3.1+. Ignored by older panels; use panel_proxy on v3.2.0 through v3.3.0.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func expandPanelGeneral(m *PanelGeneralModel) map[string]any {
	payload := map[string]any{}
	if !m.WebListen.IsNull() && !m.WebListen.IsUnknown() {
		payload["webListen"] = m.WebListen.ValueString()
	}
	if !m.WebDomain.IsNull() && !m.WebDomain.IsUnknown() {
		payload["webDomain"] = m.WebDomain.ValueString()
	}
	if !m.WebPort.IsNull() && !m.WebPort.IsUnknown() {
		payload["webPort"] = int(m.WebPort.ValueInt64())
	}
	if !m.WebBasePath.IsNull() && !m.WebBasePath.IsUnknown() {
		payload["webBasePath"] = m.WebBasePath.ValueString()
	}
	if !m.SessionMaxAge.IsNull() && !m.SessionMaxAge.IsUnknown() {
		payload["sessionMaxAge"] = int(m.SessionMaxAge.ValueInt64())
	}
	if !m.TrustedProxyCIDRs.IsNull() && !m.TrustedProxyCIDRs.IsUnknown() {
		payload["trustedProxyCIDRs"] = m.TrustedProxyCIDRs.ValueString()
	}
	if !m.WarpUpdateInterval.IsNull() && !m.WarpUpdateInterval.IsUnknown() {
		payload["warpUpdateInterval"] = int(m.WarpUpdateInterval.ValueInt64())
	}
	if !m.PageSize.IsNull() && !m.PageSize.IsUnknown() {
		payload["pageSize"] = int(m.PageSize.ValueInt64())
	}
	if !m.RemarkModel.IsNull() && !m.RemarkModel.IsUnknown() {
		payload["remarkModel"] = m.RemarkModel.ValueString()
	}
	if !m.DatePicker.IsNull() && !m.DatePicker.IsUnknown() {
		payload["datepicker"] = m.DatePicker.ValueString()
	}
	if !m.TimeLocation.IsNull() && !m.TimeLocation.IsUnknown() {
		payload["timeLocation"] = m.TimeLocation.ValueString()
	}
	if !m.ExpireDiff.IsNull() && !m.ExpireDiff.IsUnknown() {
		payload["expireDiff"] = int(m.ExpireDiff.ValueInt64())
	}
	if !m.TrafficDiff.IsNull() && !m.TrafficDiff.IsUnknown() {
		payload["trafficDiff"] = int(m.TrafficDiff.ValueInt64())
	}
	if !m.WebCertFile.IsNull() && !m.WebCertFile.IsUnknown() {
		payload["webCertFile"] = m.WebCertFile.ValueString()
	}
	if !m.WebKeyFile.IsNull() && !m.WebKeyFile.IsUnknown() {
		payload["webKeyFile"] = m.WebKeyFile.ValueString()
	}
	if !m.ExternalTrafficInformEnable.IsNull() && !m.ExternalTrafficInformEnable.IsUnknown() {
		payload["externalTrafficInformEnable"] = m.ExternalTrafficInformEnable.ValueBool()
	}
	if !m.ExternalTrafficInformURI.IsNull() && !m.ExternalTrafficInformURI.IsUnknown() {
		payload["externalTrafficInformURI"] = m.ExternalTrafficInformURI.ValueString()
	}
	if !m.SubShowIdentityOnAllLinks.IsNull() && !m.SubShowIdentityOnAllLinks.IsUnknown() {
		payload["subShowIdentityOnAllLinks"] = m.SubShowIdentityOnAllLinks.ValueBool()
	}
	if !m.OutboundDownThreshold.IsNull() && !m.OutboundDownThreshold.IsUnknown() {
		payload["outboundDownThreshold"] = int(m.OutboundDownThreshold.ValueInt64())
	}
	if !m.RestartXrayOnClientDisable.IsNull() && !m.RestartXrayOnClientDisable.IsUnknown() {
		payload["restartXrayOnClientDisable"] = m.RestartXrayOnClientDisable.ValueBool()
	}
	if !m.IPLimitAllowlist.IsNull() && !m.IPLimitAllowlist.IsUnknown() {
		payload["ipLimitAllowlist"] = m.IPLimitAllowlist.ValueString()
	}
	if !m.RealityScanCandidates.IsNull() && !m.RealityScanCandidates.IsUnknown() {
		payload["realityScanCandidates"] = m.RealityScanCandidates.ValueString()
	}
	if !m.LDAPEnable.IsNull() && !m.LDAPEnable.IsUnknown() {
		payload["ldapEnable"] = m.LDAPEnable.ValueBool()
	}
	if !m.LDAPHost.IsNull() && !m.LDAPHost.IsUnknown() {
		payload["ldapHost"] = m.LDAPHost.ValueString()
	}
	if !m.LDAPPort.IsNull() && !m.LDAPPort.IsUnknown() {
		payload["ldapPort"] = int(m.LDAPPort.ValueInt64())
	}
	if !m.LDAPUseTLS.IsNull() && !m.LDAPUseTLS.IsUnknown() {
		payload["ldapUseTLS"] = m.LDAPUseTLS.ValueBool()
	}
	if !m.LDAPInsecureSkipVerify.IsNull() && !m.LDAPInsecureSkipVerify.IsUnknown() {
		payload["ldapInsecureSkipVerify"] = m.LDAPInsecureSkipVerify.ValueBool()
	}
	if !m.LDAPBindDN.IsNull() && !m.LDAPBindDN.IsUnknown() {
		payload["ldapBindDN"] = m.LDAPBindDN.ValueString()
	}
	if !m.LDAPPasswordWO.IsNull() && !m.LDAPPasswordWO.IsUnknown() {
		payload["ldapPassword"] = m.LDAPPasswordWO.ValueString()
	} else if !m.LDAPPassword.IsNull() && !m.LDAPPassword.IsUnknown() {
		payload["ldapPassword"] = m.LDAPPassword.ValueString()
	}
	if !m.LDAPBaseDN.IsNull() && !m.LDAPBaseDN.IsUnknown() {
		payload["ldapBaseDN"] = m.LDAPBaseDN.ValueString()
	}
	if !m.LDAPUserFilter.IsNull() && !m.LDAPUserFilter.IsUnknown() {
		payload["ldapUserFilter"] = m.LDAPUserFilter.ValueString()
	}
	if !m.LDAPUserAttr.IsNull() && !m.LDAPUserAttr.IsUnknown() {
		payload["ldapUserAttr"] = m.LDAPUserAttr.ValueString()
	}
	if !m.LDAPVlessField.IsNull() && !m.LDAPVlessField.IsUnknown() {
		payload["ldapVlessField"] = m.LDAPVlessField.ValueString()
	}
	if !m.LDAPSyncCron.IsNull() && !m.LDAPSyncCron.IsUnknown() {
		payload["ldapSyncCron"] = m.LDAPSyncCron.ValueString()
	}
	if !m.LDAPFlagField.IsNull() && !m.LDAPFlagField.IsUnknown() {
		payload["ldapFlagField"] = m.LDAPFlagField.ValueString()
	}
	if !m.LDAPTruthyValues.IsNull() && !m.LDAPTruthyValues.IsUnknown() {
		payload["ldapTruthyValues"] = m.LDAPTruthyValues.ValueString()
	}
	if !m.LDAPInvertFlag.IsNull() && !m.LDAPInvertFlag.IsUnknown() {
		payload["ldapInvertFlag"] = m.LDAPInvertFlag.ValueBool()
	}
	if !m.LDAPInboundTags.IsNull() && !m.LDAPInboundTags.IsUnknown() {
		payload["ldapInboundTags"] = m.LDAPInboundTags.ValueString()
	}
	if !m.LDAPAutoCreate.IsNull() && !m.LDAPAutoCreate.IsUnknown() {
		payload["ldapAutoCreate"] = m.LDAPAutoCreate.ValueBool()
	}
	if !m.LDAPAutoDelete.IsNull() && !m.LDAPAutoDelete.IsUnknown() {
		payload["ldapAutoDelete"] = m.LDAPAutoDelete.ValueBool()
	}
	if !m.LDAPDefaultTotalGB.IsNull() && !m.LDAPDefaultTotalGB.IsUnknown() {
		payload["ldapDefaultTotalGB"] = int(m.LDAPDefaultTotalGB.ValueInt64())
	}
	if !m.LDAPDefaultExpiryDays.IsNull() && !m.LDAPDefaultExpiryDays.IsUnknown() {
		payload["ldapDefaultExpiryDays"] = int(m.LDAPDefaultExpiryDays.ValueInt64())
	}
	if !m.LDAPDefaultLimitIP.IsNull() && !m.LDAPDefaultLimitIP.IsUnknown() {
		payload["ldapDefaultLimitIP"] = int(m.LDAPDefaultLimitIP.ValueInt64())
	}
	if !m.PanelProxy.IsNull() && !m.PanelProxy.IsUnknown() {
		payload["panelProxy"] = m.PanelProxy.ValueString()
	}
	if !m.PanelOutbound.IsNull() && !m.PanelOutbound.IsUnknown() {
		payload["panelOutbound"] = m.PanelOutbound.ValueString()
	}
	return payload
}

func flattenPanelGeneral(in map[string]any) *PanelGeneralModel {
	m := &PanelGeneralModel{
		ID: types.StringValue("settings"),
	}
	if v, ok := in["webListen"]; ok {
		m.WebListen = types.StringValue(stringValue(v))
	}
	if v, ok := in["webDomain"]; ok {
		m.WebDomain = types.StringValue(stringValue(v))
	}
	if v, ok := in["webPort"]; ok {
		m.WebPort = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["webBasePath"]; ok {
		m.WebBasePath = types.StringValue(stringValue(v))
	}
	if v, ok := in["sessionMaxAge"]; ok {
		m.SessionMaxAge = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["trustedProxyCIDRs"]; ok {
		m.TrustedProxyCIDRs = types.StringValue(stringValue(v))
	}
	if v, ok := in["warpUpdateInterval"]; ok {
		m.WarpUpdateInterval = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["pageSize"]; ok {
		m.PageSize = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["remarkModel"]; ok {
		m.RemarkModel = types.StringValue(stringValue(v))
	}
	if v, ok := in["datepicker"]; ok {
		m.DatePicker = types.StringValue(stringValue(v))
	}
	if v, ok := in["timeLocation"]; ok {
		m.TimeLocation = types.StringValue(stringValue(v))
	}
	if v, ok := in["expireDiff"]; ok {
		m.ExpireDiff = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["trafficDiff"]; ok {
		m.TrafficDiff = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["webCertFile"]; ok {
		m.WebCertFile = types.StringValue(stringValue(v))
	}
	if v, ok := in["webKeyFile"]; ok {
		m.WebKeyFile = types.StringValue(stringValue(v))
	}
	if v, ok := in["externalTrafficInformEnable"]; ok {
		m.ExternalTrafficInformEnable = types.BoolValue(boolValue(v))
	}
	if v, ok := in["externalTrafficInformURI"]; ok {
		m.ExternalTrafficInformURI = types.StringValue(stringValue(v))
	}
	if v, ok := in["subShowIdentityOnAllLinks"]; ok {
		m.SubShowIdentityOnAllLinks = types.BoolValue(boolValue(v))
	}
	if v, ok := in["outboundDownThreshold"]; ok {
		m.OutboundDownThreshold = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["restartXrayOnClientDisable"]; ok {
		m.RestartXrayOnClientDisable = types.BoolValue(boolValue(v))
	}
	if v, ok := in["ipLimitAllowlist"]; ok {
		m.IPLimitAllowlist = types.StringValue(stringValue(v))
	}
	if v, ok := in["realityScanCandidates"]; ok {
		m.RealityScanCandidates = types.StringValue(stringValue(v))
	}
	if v, ok := in["ldapEnable"]; ok {
		m.LDAPEnable = types.BoolValue(boolValue(v))
	}
	if v, ok := in["ldapHost"]; ok {
		m.LDAPHost = types.StringValue(stringValue(v))
	}
	if v, ok := in["ldapPort"]; ok {
		m.LDAPPort = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["ldapUseTLS"]; ok {
		m.LDAPUseTLS = types.BoolValue(boolValue(v))
	}
	if v, ok := in["ldapInsecureSkipVerify"]; ok {
		m.LDAPInsecureSkipVerify = types.BoolValue(boolValue(v))
	}
	if v, ok := in["ldapBindDN"]; ok {
		m.LDAPBindDN = types.StringValue(stringValue(v))
	}
	if v, ok := in["ldapPassword"]; ok {
		m.LDAPPassword = types.StringValue(stringValue(v))
	}
	if v, ok := in["ldapBaseDN"]; ok {
		m.LDAPBaseDN = types.StringValue(stringValue(v))
	}
	if v, ok := in["ldapUserFilter"]; ok {
		m.LDAPUserFilter = types.StringValue(stringValue(v))
	}
	if v, ok := in["ldapUserAttr"]; ok {
		m.LDAPUserAttr = types.StringValue(stringValue(v))
	}
	if v, ok := in["ldapVlessField"]; ok {
		m.LDAPVlessField = types.StringValue(stringValue(v))
	}
	if v, ok := in["ldapSyncCron"]; ok {
		m.LDAPSyncCron = types.StringValue(stringValue(v))
	}
	if v, ok := in["ldapFlagField"]; ok {
		m.LDAPFlagField = types.StringValue(stringValue(v))
	}
	if v, ok := in["ldapTruthyValues"]; ok {
		m.LDAPTruthyValues = types.StringValue(stringValue(v))
	}
	if v, ok := in["ldapInvertFlag"]; ok {
		m.LDAPInvertFlag = types.BoolValue(boolValue(v))
	}
	if v, ok := in["ldapInboundTags"]; ok {
		m.LDAPInboundTags = types.StringValue(stringValue(v))
	}
	if v, ok := in["ldapAutoCreate"]; ok {
		m.LDAPAutoCreate = types.BoolValue(boolValue(v))
	}
	if v, ok := in["ldapAutoDelete"]; ok {
		m.LDAPAutoDelete = types.BoolValue(boolValue(v))
	}
	if v, ok := in["ldapDefaultTotalGB"]; ok {
		m.LDAPDefaultTotalGB = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["ldapDefaultExpiryDays"]; ok {
		m.LDAPDefaultExpiryDays = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["ldapDefaultLimitIP"]; ok {
		m.LDAPDefaultLimitIP = types.Int64Value(int64(intValue(v)))
	}
	if v, ok := in["panelProxy"]; ok {
		m.PanelProxy = types.StringValue(stringValue(v))
	}
	if v, ok := in["panelOutbound"]; ok {
		m.PanelOutbound = types.StringValue(stringValue(v))
	}
	return m
}

// ---------------------------------------------------------------------------
// Shared typed settings helper: apply settings to API and read back
// ---------------------------------------------------------------------------

var settingsMu sync.Mutex

// panelRestartMu serializes provider-initiated panel restarts. It is separate
// from settingsMu on purpose: the restart is issued *after* the settings lock is
// released (a restart while holding it would block every other settings read for
// the whole restart window). Without its own lock, panel_general and
// panel_subscription — which Terraform applies concurrently when both are in the
// graph — can each SendRestart and then WaitForReady against a panel the other
// one is bouncing, so both wait on a moving target.
var panelRestartMu sync.Mutex

var panelSettingSecretKeys = []string{
	"ldapPassword",
	"twoFactorToken",
	"tgBotToken",
	"smtpPassword",
	"discordBotToken",
}

func settingsApplyTyped(
	ctx context.Context,
	desired map[string]any,
	diags *diag.Diagnostics,
	client *Client,
) {
	if len(desired) == 0 {
		return
	}

	settingsMu.Lock()

	existing, err := client.GetSettings(ctx)
	if err != nil {
		settingsMu.Unlock()
		diags.AddError("Failed to get settings", err.Error())
		return
	}

	// Computed before the write, while `existing` still describes the panel.
	needRestart := panelSettingsNeedRestart(existing, desired)

	merged := mergeSettingsForUpdate(client, existing, desired)
	if err := client.UpdateSettings(ctx, merged); err != nil {
		settingsMu.Unlock()
		diags.AddError("Failed to update settings", err.Error())
		return
	}
	client.rememberConfiguredSettingSecrets(desired)
	settingsMu.Unlock()

	// The panel registers its notifier cron jobs once, at startup, from
	// settings owned by panel_telegram and panel_email — so those two resources
	// need the same restart gate panel_general and panel_subscription have
	// (#449). Restart outside settingsMu, and serialized against the other
	// resources' restarts, exactly as they do.
	if !needRestart {
		return
	}
	panelRestartMu.Lock()
	defer panelRestartMu.Unlock()
	if err := client.SendRestart(ctx); err != nil {
		diags.AddError("Failed to restart panel", err.Error())
		return
	}
	if err := client.WaitForReady(ctx); err != nil {
		diags.AddError("Panel did not become ready after restart", err.Error())
		return
	}
}

func settingsReadTyped(
	ctx context.Context,
	diags *diag.Diagnostics,
	client *Client,
) map[string]any {
	settings, err := client.GetSettings(ctx)
	if err != nil {
		diags.AddError("Failed to get settings", err.Error())
		return nil
	}
	return settings
}

func mergeSettingsForUpdate(client *Client, existing, desired map[string]any) map[string]any {
	if client != nil {
		existing = client.preserveCachedSettingSecrets(existing, desired)
	}
	return mergeSettings(existing, desired)
}

func (c *Client) rememberConfiguredSettingSecrets(settings map[string]any) {
	if c == nil || len(settings) == 0 {
		return
	}

	c.settingsSecretMu.Lock()
	defer c.settingsSecretMu.Unlock()

	for _, key := range panelSettingSecretKeys {
		value, ok := settings[key]
		if !ok {
			continue
		}

		secret := stringValue(value)
		if secret == "" || isRedactedSettingSecretValue(secret) {
			delete(c.settingsSecrets, key)
			continue
		}

		if c.settingsSecrets == nil {
			c.settingsSecrets = make(map[string]string)
		}
		c.settingsSecrets[key] = secret
	}
}

func (c *Client) preserveCachedSettingSecrets(existing, desired map[string]any) map[string]any {
	if c == nil || len(existing) == 0 {
		return existing
	}

	c.settingsSecretMu.Lock()
	defer c.settingsSecretMu.Unlock()

	if len(c.settingsSecrets) == 0 {
		return existing
	}

	out := existing
	copied := false
	for _, key := range panelSettingSecretKeys {
		if _, configured := desired[key]; configured {
			continue
		}

		cached := c.settingsSecrets[key]
		if cached == "" || !isRedactedSettingSecret(existing[key]) {
			continue
		}

		if !copied {
			out = make(map[string]any, len(existing))
			for k, v := range existing {
				out[k] = v
			}
			copied = true
		}
		out[key] = cached
	}
	return out
}

func isRedactedSettingSecret(value any) bool {
	secret, ok := value.(string)
	if !ok {
		return value == nil
	}
	return isRedactedSettingSecretValue(secret)
}

func isRedactedSettingSecretValue(secret string) bool {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return true
	}
	if strings.EqualFold(trimmed, "redacted") || strings.EqualFold(trimmed, "<redacted>") {
		return true
	}
	if len(trimmed) >= 3 && strings.Trim(trimmed, "*") == "" {
		return true
	}
	return false
}

func preserveSettingSecret(observed, configured types.String) types.String {
	if configured.IsNull() || configured.IsUnknown() {
		return observed
	}

	configuredValue := configured.ValueString()
	if observed.IsNull() || observed.IsUnknown() {
		return configured
	}

	observedValue := observed.ValueString()
	if configuredValue == "" && observedValue != "" {
		return configured
	}
	if configuredValue != "" && isRedactedSettingSecretValue(observedValue) {
		return configured
	}

	return observed
}

// preserveWOVersion carries the *_wo_version trigger into state during Apply.
// UseStateForUnknown on the schema handles the Plan phase; this handles Apply,
// where flatten* (which has no _wo_version field) would otherwise overwrite
// the planned value with null.
func preserveWOVersion(observed, configured types.Int64) types.Int64 {
	if observed.IsNull() || observed.IsUnknown() {
		return configured
	}
	return observed
}

// preserveRemovedString / preserveRemovedBool echo the configured (Apply) or
// prior-state (Read) value into state when the panel no longer returns the
// field (it was dropped upstream, e.g. remarkModel/subShowInfo/tgBotLoginNotify
// in 3x-ui v3.4.0, panelProxy in v3.3.1). Without this, a non-null planned
// value would collide with a null read-back and surface as "Provider produced
// inconsistent result after apply".
//
// Crucially, when there is no concrete value to echo (configured is null or
// unknown — the normal case for a Computed attr the user did not set), we
// return null, NOT unknown. Returning unknown would violate Terraform's
// "all values must be known after apply" rule.
func preserveRemovedString(observed, configured types.String) types.String {
	if observed.IsNull() || observed.IsUnknown() {
		if configured.IsNull() || configured.IsUnknown() {
			return types.StringNull()
		}
		return configured
	}
	return observed
}

func preserveRemovedBool(observed, configured types.Bool) types.Bool {
	if observed.IsNull() || observed.IsUnknown() {
		if configured.IsNull() || configured.IsUnknown() {
			return types.BoolNull()
		}
		return configured
	}
	return observed
}

func preservePanelGeneralSecrets(state, configured *PanelGeneralModel) {
	if state == nil || configured == nil {
		return
	}
	state.LDAPPassword = preserveSettingSecret(state.LDAPPassword, configured.LDAPPassword)
	state.LDAPPasswordWOVersion = preserveWOVersion(state.LDAPPasswordWOVersion, configured.LDAPPasswordWOVersion)
	// remarkModel was removed from AllSetting in 3x-ui v3.4.0 (superseded by
	// remarkTemplate). v3.4.0 panels accept but don't store/return it, so echo the
	// configured value back to keep state consistent; older panels return it.
	state.RemarkModel = preserveRemovedString(state.RemarkModel, configured.RemarkModel)
	// panelProxy was removed from AllSetting in 3x-ui v3.3.1 (replaced by
	// panelOutbound). Same echo needed so a user still setting panel_proxy on
	// v3.3.1+ does not hit "inconsistent result after apply".
	state.PanelProxy = preserveRemovedString(state.PanelProxy, configured.PanelProxy)
}

func preservePanelSecuritySecrets(state, configured *PanelSecurityModel) {
	if state == nil || configured == nil {
		return
	}
	state.TwoFactorToken = preserveSettingSecret(state.TwoFactorToken, configured.TwoFactorToken)
	state.TwoFactorTokenWOVersion = preserveWOVersion(state.TwoFactorTokenWOVersion, configured.TwoFactorTokenWOVersion)
}

func preservePanelTelegramSecrets(state, configured *PanelTelegramModel) {
	if state == nil || configured == nil {
		return
	}
	state.TgBotToken = preserveSettingSecret(state.TgBotToken, configured.TgBotToken)
	state.TgBotTokenWOVersion = preserveWOVersion(state.TgBotTokenWOVersion, configured.TgBotTokenWOVersion)
	// tgBotLoginNotify was removed from AllSetting in 3x-ui v3.4.0. Echo the
	// configured value on v3.4.0 (panel accepts but doesn't return it).
	state.TgBotLoginNotify = preserveRemovedBool(state.TgBotLoginNotify, configured.TgBotLoginNotify)
}

func preservePanelEmailSecrets(state, configured *PanelEmailModel) {
	if state == nil || configured == nil {
		return
	}
	state.SmtpPassword = preserveSettingSecret(state.SmtpPassword, configured.SmtpPassword)
	state.SmtpPasswordWOVersion = preserveWOVersion(state.SmtpPasswordWOVersion, configured.SmtpPasswordWOVersion)
}

func preservePanelDiscordSecrets(state, configured *PanelDiscordModel) {
	if state == nil || configured == nil {
		return
	}
	state.DiscordBotToken = preserveSettingSecret(state.DiscordBotToken, configured.DiscordBotToken)
	state.DiscordBotTokenWOVer = preserveWOVersion(state.DiscordBotTokenWOVer, configured.DiscordBotTokenWOVer)
}

// preservePanelSubscriptionRemoved echoes configured values for fields that
// 3x-ui v3.4.0 dropped from AllSetting (subShowInfo, subEmailInRemark). v3.4.0
// panels accept but don't store/return them; echoing keeps Apply consistent.
func preservePanelSubscriptionRemoved(state, configured *PanelSubscriptionModel) {
	if state == nil || configured == nil {
		return
	}
	state.SubShowInfo = preserveRemovedBool(state.SubShowInfo, configured.SubShowInfo)
	state.SubEmailInRemark = preserveRemovedBool(state.SubEmailInRemark, configured.SubEmailInRemark)
}

// ---------------------------------------------------------------------------
// PanelGeneralResource (threexui_panel_general)
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &PanelGeneralResource{}
	_ resource.ResourceWithConfigure   = &PanelGeneralResource{}
	_ resource.ResourceWithImportState = &PanelGeneralResource{}
	_ resource.ResourceWithModifyPlan  = &PanelGeneralResource{}
)

type PanelGeneralResource struct {
	client *Client
}

func NewPanelGeneralResource() resource.Resource {
	return &PanelGeneralResource{}
}

func (r *PanelGeneralResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_panel_general"
}

func (r *PanelGeneralResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = panelGeneralSchema()
}

func (r *PanelGeneralResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PanelGeneralResource) readPanelGeneralState(ctx context.Context, diags *diag.Diagnostics) *PanelGeneralModel {
	settings := settingsReadTyped(ctx, diags, r.client)
	if settings == nil {
		return nil
	}
	state := flattenPanelGeneral(settings)

	// xrayOutboundTestUrl is served via the xray endpoint, not the settings API.
	testURL, err := r.client.GetXrayOutboundTestURL(ctx)
	if err != nil {
		diags.AddError("Failed to get xray outbound test URL", err.Error())
		return nil
	}
	if testURL != "" {
		state.XrayOutboundTestURL = types.StringValue(testURL)
	}
	return state
}

func (r *PanelGeneralResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PanelGeneralModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config PanelGeneralModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveGeneralLDAPPasswordWO(&plan, config)

	r.applyPanelGeneral(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	state := r.readPanelGeneralState(ctx, &resp.Diagnostics)
	if state == nil {
		return
	}
	preservePanelGeneralSecrets(state, &plan)
	r.client.rememberConfiguredSettingSecrets(expandPanelGeneral(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelGeneralResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var prior PanelGeneralModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := r.readPanelGeneralState(ctx, &resp.Diagnostics)
	if state == nil {
		return
	}
	preservePanelGeneralSecrets(state, &prior)
	r.client.rememberConfiguredSettingSecrets(expandPanelGeneral(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelGeneralResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PanelGeneralModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var priorState PanelGeneralModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config PanelGeneralModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveGeneralLDAPPasswordWOUpdate(&plan, priorState, config)

	r.applyPanelGeneral(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	state := r.readPanelGeneralState(ctx, &resp.Diagnostics)
	if state == nil {
		return
	}
	preservePanelGeneralSecrets(state, &plan)
	r.client.rememberConfiguredSettingSecrets(expandPanelGeneral(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelGeneralResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *PanelGeneralResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	modifyPlanWOVersion(
		ctx, req, resp,
		func(m PanelGeneralModel) types.Int64 { return m.LDAPPasswordWOVersion },
		func(m *PanelGeneralModel, v types.String) { m.LDAPPassword = v },
	)
}

func (r *PanelGeneralResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	state := r.readPanelGeneralState(ctx, &resp.Diagnostics)
	if state == nil {
		return
	}
	r.client.rememberConfiguredSettingSecrets(expandPanelGeneral(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func resolveGeneralLDAPPasswordWO(plan *PanelGeneralModel, config PanelGeneralModel) {
	if !config.LDAPPasswordWO.IsNull() {
		plan.LDAPPassword = config.LDAPPasswordWO
	}
}

func resolveGeneralLDAPPasswordWOUpdate(plan *PanelGeneralModel, state PanelGeneralModel, config PanelGeneralModel) {
	if config.LDAPPasswordWO.IsNull() {
		return
	}
	if woVersionTriggered(plan.LDAPPasswordWOVersion, state.LDAPPasswordWOVersion) {
		plan.LDAPPassword = config.LDAPPasswordWO
	}
}

func (r *PanelGeneralResource) applyPanelGeneral(ctx context.Context, plan *PanelGeneralModel, diags *diag.Diagnostics) {
	desired := expandPanelGeneral(plan)

	// Warn about web_base_path change.
	if _, hasBasePath := desired["webBasePath"]; hasBasePath {
		diags.AddWarning(
			"Changing web_base_path requires updating provider config",
			"The provider's base_path must match the panel's web_base_path. After this change, update the provider configuration to use the new base_path, otherwise the provider will not be able to connect.",
		)
	}

	if len(desired) > 0 {
		settingsMu.Lock()
		existing, err := r.client.GetSettings(ctx)
		if err != nil {
			settingsMu.Unlock()
			diags.AddError("Failed to get settings", err.Error())
			return
		}

		needRestart := panelSettingsNeedRestart(existing, desired)
		merged := mergeSettingsForUpdate(r.client, existing, desired)
		if err := r.client.UpdateSettings(ctx, merged); err != nil {
			settingsMu.Unlock()
			diags.AddError("Failed to update settings", err.Error())
			return
		}
		r.client.rememberConfiguredSettingSecrets(desired)
		settingsMu.Unlock()

		if needRestart {
			panelRestartMu.Lock()
			// Send the restart request while basePath still points to the
			// old path (where the panel is currently listening).
			if err := r.client.SendRestart(ctx); err != nil {
				panelRestartMu.Unlock()
				diags.AddError("Failed to restart panel", err.Error())
				return
			}

			// Now update basePath so waitForReady polls the new path.
			if newPath, ok := desired["webBasePath"]; ok {
				r.client.SetBasePath(stringValue(newPath))
			}

			err := r.client.WaitForReady(ctx)
			panelRestartMu.Unlock()
			if err != nil {
				diags.AddError("Panel did not become ready after restart", err.Error())
				return
			}
		} else if newPath, ok := desired["webBasePath"]; ok {
			// No restart needed, but basePath still needs updating.
			r.client.SetBasePath(stringValue(newPath))
		}
	}

	// xrayOutboundTestUrl is managed via xray endpoint, not settings API.
	if !plan.XrayOutboundTestURL.IsNull() && !plan.XrayOutboundTestURL.IsUnknown() {
		xrayTemplateMu.Lock()
		if err := r.client.SetXrayOutboundTestURL(ctx, plan.XrayOutboundTestURL.ValueString()); err != nil {
			xrayTemplateMu.Unlock()
			diags.AddError("Failed to set xray outbound test URL", err.Error())
			return
		}
		xrayTemplateMu.Unlock()
	}
}

// ---------------------------------------------------------------------------
// PanelSecurityResource (threexui_panel_security)
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &PanelSecurityResource{}
	_ resource.ResourceWithConfigure   = &PanelSecurityResource{}
	_ resource.ResourceWithImportState = &PanelSecurityResource{}
	_ resource.ResourceWithModifyPlan  = &PanelSecurityResource{}
)

type PanelSecurityResource struct {
	client *Client
}

func NewPanelSecurityResource() resource.Resource {
	return &PanelSecurityResource{}
}

func (r *PanelSecurityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_panel_security"
}

func (r *PanelSecurityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = panelSecuritySchema()
}

func (r *PanelSecurityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PanelSecurityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PanelSecurityModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config PanelSecurityModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveSecurityTokenWO(&plan, config)

	r.warnIfTwoFactor(&plan, &resp.Diagnostics)

	desired := expandPanelSecurity(&plan)
	settingsApplyTyped(ctx, desired, &resp.Diagnostics, r.client)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelSecurity(settings)
	preservePanelSecuritySecrets(state, &plan)
	r.client.rememberConfiguredSettingSecrets(expandPanelSecurity(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelSecurityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var prior PanelSecurityModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelSecurity(settings)
	preservePanelSecuritySecrets(state, &prior)
	r.client.rememberConfiguredSettingSecrets(expandPanelSecurity(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelSecurityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PanelSecurityModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var priorState PanelSecurityModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config PanelSecurityModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveSecurityTokenWOUpdate(&plan, priorState, config)

	r.warnIfTwoFactor(&plan, &resp.Diagnostics)

	desired := expandPanelSecurity(&plan)
	settingsApplyTyped(ctx, desired, &resp.Diagnostics, r.client)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelSecurity(settings)
	preservePanelSecuritySecrets(state, &plan)
	r.client.rememberConfiguredSettingSecrets(expandPanelSecurity(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelSecurityResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *PanelSecurityResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	modifyPlanWOVersion(
		ctx, req, resp,
		func(m PanelSecurityModel) types.Int64 { return m.TwoFactorTokenWOVersion },
		func(m *PanelSecurityModel, v types.String) { m.TwoFactorToken = v },
	)
}

func (r *PanelSecurityResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelSecurity(settings)
	r.client.rememberConfiguredSettingSecrets(expandPanelSecurity(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func resolveSecurityTokenWO(plan *PanelSecurityModel, config PanelSecurityModel) {
	if !config.TwoFactorTokenWO.IsNull() {
		plan.TwoFactorToken = config.TwoFactorTokenWO
	}
}

func resolveSecurityTokenWOUpdate(plan *PanelSecurityModel, state PanelSecurityModel, config PanelSecurityModel) {
	if config.TwoFactorTokenWO.IsNull() {
		return
	}
	if woVersionTriggered(plan.TwoFactorTokenWOVersion, state.TwoFactorTokenWOVersion) {
		plan.TwoFactorToken = config.TwoFactorTokenWO
	}
}

func (r *PanelSecurityResource) warnIfTwoFactor(plan *PanelSecurityModel, diags *diag.Diagnostics) {
	if !plan.TwoFactorEnable.IsNull() && !plan.TwoFactorEnable.IsUnknown() && plan.TwoFactorEnable.ValueBool() {
		diags.AddWarning(
			"2FA enabled — automatic re-login will not work",
			"The provider can send a TOTP code with the initial login (via the two_factor_code provider attribute), but TOTP codes expire every 30 seconds. Automatic re-login on session expiry will fail once the code is no longer valid. Ensure you supply a fresh two_factor_code for each run, or disable 2FA to allow unattended operation.",
		)
	}
}

// ---------------------------------------------------------------------------
// PanelTelegramResource (threexui_panel_telegram)
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &PanelTelegramResource{}
	_ resource.ResourceWithConfigure   = &PanelTelegramResource{}
	_ resource.ResourceWithImportState = &PanelTelegramResource{}
	_ resource.ResourceWithModifyPlan  = &PanelTelegramResource{}
)

type PanelTelegramResource struct {
	client *Client
}

func NewPanelTelegramResource() resource.Resource {
	return &PanelTelegramResource{}
}

func (r *PanelTelegramResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_panel_telegram"
}

func (r *PanelTelegramResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = panelTelegramSchema()
}

func (r *PanelTelegramResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PanelTelegramResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PanelTelegramModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config PanelTelegramModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveTelegramTokenWO(&plan, config)

	desired := expandPanelTelegram(&plan)
	settingsApplyTyped(ctx, desired, &resp.Diagnostics, r.client)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelTelegram(settings)
	preservePanelTelegramSecrets(state, &plan)
	r.client.rememberConfiguredSettingSecrets(expandPanelTelegram(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelTelegramResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var prior PanelTelegramModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelTelegram(settings)
	preservePanelTelegramSecrets(state, &prior)
	r.client.rememberConfiguredSettingSecrets(expandPanelTelegram(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelTelegramResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PanelTelegramModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var priorState PanelTelegramModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config PanelTelegramModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveTelegramTokenWOUpdate(&plan, priorState, config)

	desired := expandPanelTelegram(&plan)
	settingsApplyTyped(ctx, desired, &resp.Diagnostics, r.client)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelTelegram(settings)
	preservePanelTelegramSecrets(state, &plan)
	r.client.rememberConfiguredSettingSecrets(expandPanelTelegram(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func resolveTelegramTokenWO(plan *PanelTelegramModel, config PanelTelegramModel) {
	if !config.TgBotTokenWO.IsNull() {
		plan.TgBotToken = config.TgBotTokenWO
	}
}

func resolveTelegramTokenWOUpdate(plan *PanelTelegramModel, state PanelTelegramModel, config PanelTelegramModel) {
	if config.TgBotTokenWO.IsNull() {
		return
	}
	if woVersionTriggered(plan.TgBotTokenWOVersion, state.TgBotTokenWOVersion) {
		plan.TgBotToken = config.TgBotTokenWO
	}
}

func (r *PanelTelegramResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *PanelTelegramResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	modifyPlanWOVersion(
		ctx, req, resp,
		func(m PanelTelegramModel) types.Int64 { return m.TgBotTokenWOVersion },
		func(m *PanelTelegramModel, v types.String) { m.TgBotToken = v },
	)
}

func (r *PanelTelegramResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelTelegram(settings)
	r.client.rememberConfiguredSettingSecrets(expandPanelTelegram(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// ---------------------------------------------------------------------------
// PanelEmailResource (threexui_panel_email)
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &PanelEmailResource{}
	_ resource.ResourceWithConfigure   = &PanelEmailResource{}
	_ resource.ResourceWithImportState = &PanelEmailResource{}
	_ resource.ResourceWithModifyPlan  = &PanelEmailResource{}
)

type PanelEmailResource struct {
	client *Client
}

func NewPanelEmailResource() resource.Resource {
	return &PanelEmailResource{}
}

func (r *PanelEmailResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_panel_email"
}

func (r *PanelEmailResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = panelEmailSchema()
}

func (r *PanelEmailResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PanelEmailResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PanelEmailModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config PanelEmailModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveEmailPasswordWO(&plan, config)

	desired := expandPanelEmail(&plan)
	settingsApplyTyped(ctx, desired, &resp.Diagnostics, r.client)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelEmail(settings)
	preservePanelEmailSecrets(state, &plan)
	r.client.rememberConfiguredSettingSecrets(expandPanelEmail(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelEmailResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var prior PanelEmailModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelEmail(settings)
	preservePanelEmailSecrets(state, &prior)
	r.client.rememberConfiguredSettingSecrets(expandPanelEmail(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelEmailResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PanelEmailModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var priorState PanelEmailModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config PanelEmailModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveEmailPasswordWOUpdate(&plan, priorState, config)

	desired := expandPanelEmail(&plan)
	settingsApplyTyped(ctx, desired, &resp.Diagnostics, r.client)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelEmail(settings)
	preservePanelEmailSecrets(state, &plan)
	r.client.rememberConfiguredSettingSecrets(expandPanelEmail(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func resolveEmailPasswordWO(plan *PanelEmailModel, config PanelEmailModel) {
	if !config.SmtpPasswordWO.IsNull() {
		plan.SmtpPassword = config.SmtpPasswordWO
	}
}

func resolveEmailPasswordWOUpdate(plan *PanelEmailModel, state PanelEmailModel, config PanelEmailModel) {
	if config.SmtpPasswordWO.IsNull() {
		return
	}
	if woVersionTriggered(plan.SmtpPasswordWOVersion, state.SmtpPasswordWOVersion) {
		plan.SmtpPassword = config.SmtpPasswordWO
	}
}

func (r *PanelEmailResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *PanelEmailResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	modifyPlanWOVersion(
		ctx, req, resp,
		func(m PanelEmailModel) types.Int64 { return m.SmtpPasswordWOVersion },
		func(m *PanelEmailModel, v types.String) { m.SmtpPassword = v },
	)
}

func (r *PanelEmailResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelEmail(settings)
	r.client.rememberConfiguredSettingSecrets(expandPanelEmail(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// ---------------------------------------------------------------------------
// PanelDiscordResource (threexui_panel_discord)
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &PanelDiscordResource{}
	_ resource.ResourceWithConfigure   = &PanelDiscordResource{}
	_ resource.ResourceWithImportState = &PanelDiscordResource{}
	_ resource.ResourceWithModifyPlan  = &PanelDiscordResource{}
)

type PanelDiscordResource struct {
	client *Client
}

func NewPanelDiscordResource() resource.Resource {
	return &PanelDiscordResource{}
}

func (r *PanelDiscordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_panel_discord"
}

func (r *PanelDiscordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = panelDiscordSchema()
}

func (r *PanelDiscordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PanelDiscordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PanelDiscordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config PanelDiscordModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveDiscordTokenWO(&plan, config)

	desired := expandPanelDiscord(&plan)
	settingsApplyTyped(ctx, desired, &resp.Diagnostics, r.client)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelDiscord(settings)
	preservePanelDiscordSecrets(state, &plan)
	r.client.rememberConfiguredSettingSecrets(expandPanelDiscord(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelDiscordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var prior PanelDiscordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelDiscord(settings)
	preservePanelDiscordSecrets(state, &prior)
	r.client.rememberConfiguredSettingSecrets(expandPanelDiscord(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelDiscordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PanelDiscordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var priorState PanelDiscordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var config PanelDiscordModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resolveDiscordTokenWOUpdate(&plan, priorState, config)

	desired := expandPanelDiscord(&plan)
	settingsApplyTyped(ctx, desired, &resp.Diagnostics, r.client)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelDiscord(settings)
	preservePanelDiscordSecrets(state, &plan)
	r.client.rememberConfiguredSettingSecrets(expandPanelDiscord(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func resolveDiscordTokenWO(plan *PanelDiscordModel, config PanelDiscordModel) {
	if !config.DiscordBotTokenWO.IsNull() {
		plan.DiscordBotToken = config.DiscordBotTokenWO
	}
}

func resolveDiscordTokenWOUpdate(plan *PanelDiscordModel, state PanelDiscordModel, config PanelDiscordModel) {
	if config.DiscordBotTokenWO.IsNull() {
		return
	}
	if woVersionTriggered(plan.DiscordBotTokenWOVer, state.DiscordBotTokenWOVer) {
		plan.DiscordBotToken = config.DiscordBotTokenWO
	}
}

func (r *PanelDiscordResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *PanelDiscordResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	modifyPlanWOVersion(
		ctx, req, resp,
		func(m PanelDiscordModel) types.Int64 { return m.DiscordBotTokenWOVer },
		func(m *PanelDiscordModel, v types.String) { m.DiscordBotToken = v },
	)
}

func (r *PanelDiscordResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelDiscord(settings)
	r.client.rememberConfiguredSettingSecrets(expandPanelDiscord(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// ---------------------------------------------------------------------------
// PanelSubscriptionResource (threexui_panel_subscription)
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = &PanelSubscriptionResource{}
	_ resource.ResourceWithConfigure   = &PanelSubscriptionResource{}
	_ resource.ResourceWithImportState = &PanelSubscriptionResource{}
)

type PanelSubscriptionResource struct {
	client *Client
}

func NewPanelSubscriptionResource() resource.Resource {
	return &PanelSubscriptionResource{}
}

func (r *PanelSubscriptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_panel_subscription"
}

func (r *PanelSubscriptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = panelSubscriptionSchema()
}

func (r *PanelSubscriptionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PanelSubscriptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PanelSubscriptionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applySubscription(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelSubscription(settings)
	preservePanelSubscriptionRemoved(state, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelSubscriptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var prior PanelSubscriptionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelSubscription(settings)
	preservePanelSubscriptionRemoved(state, &prior)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelSubscriptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PanelSubscriptionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.applySubscription(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelSubscription(settings)
	preservePanelSubscriptionRemoved(state, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *PanelSubscriptionResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.State.RemoveResource(ctx)
}

func (r *PanelSubscriptionResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	settings := settingsReadTyped(ctx, &resp.Diagnostics, r.client)
	if settings == nil {
		return
	}
	state := flattenPanelSubscription(settings)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// applySubscription applies subscription settings twice to work around a 3x-ui
// bug where subJsonEnable is not persisted when subEnable changes in the same
// request.
func (r *PanelSubscriptionResource) applySubscription(ctx context.Context, plan *PanelSubscriptionModel, diags *diag.Diagnostics) {
	desired := expandPanelSubscription(plan)
	if len(desired) == 0 {
		return
	}

	settingsMu.Lock()

	// First apply.
	existing, err := r.client.GetSettings(ctx)
	if err != nil {
		settingsMu.Unlock()
		diags.AddError("Failed to get settings", err.Error())
		return
	}
	// Detect whether a server-binding key changed (subEnable/subListen/subDomain/
	// subPort/subPath/subCertFile/subKeyFile). The 3x-ui subscription server is only
	// (re)initialised at panel startup (see 3x-ui/internal/sub/sub.go Start()+initRouter()),
	// so without a panel restart the subscription port stays closed / the URL 404s
	// until the panel is restarted by hand (#291).
	needRestart := panelSettingsNeedRestart(existing, desired)
	merged := mergeSettingsForUpdate(r.client, existing, desired)
	if err := r.client.UpdateSettings(ctx, merged); err != nil {
		settingsMu.Unlock()
		diags.AddError("Failed to update settings", err.Error())
		return
	}
	r.client.rememberConfiguredSettingSecrets(desired)

	// Second apply (workaround for 3x-ui bug).
	existing2, err := r.client.GetSettings(ctx)
	if err != nil {
		settingsMu.Unlock()
		diags.AddError("Failed to get settings (second apply)", err.Error())
		return
	}
	merged2 := mergeSettingsForUpdate(r.client, existing2, desired)
	if err := r.client.UpdateSettings(ctx, merged2); err != nil {
		settingsMu.Unlock()
		diags.AddError("Failed to update settings (second apply)", err.Error())
		return
	}
	r.client.rememberConfiguredSettingSecrets(desired)
	settingsMu.Unlock()

	// Restart the panel (outside the settings lock) so the subscription server
	// rebinds with the new settings. Mirrors applyPanelGeneral. No base-path
	// handling here — the subscription server does not share the panel's webBasePath.
	if needRestart {
		panelRestartMu.Lock()
		if err := r.client.SendRestart(ctx); err != nil {
			panelRestartMu.Unlock()
			diags.AddError("Failed to restart panel", err.Error())
			return
		}
		err := r.client.WaitForReady(ctx)
		panelRestartMu.Unlock()
		if err != nil {
			diags.AddError("Panel did not become ready after restart", err.Error())
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Helper functions (no SDK dependency)
// ---------------------------------------------------------------------------

func panelSettingsNeedRestart(existing, desired map[string]any) bool {
	restartKeys := []string{
		"webListen",
		"webDomain",
		"webPort",
		"webBasePath",
		"webCertFile",
		"webKeyFile",
		"sessionMaxAge",
		// Read once in web.Server.Start()/startTask registration, not per request:
		// GetTimeLocation seeds every cron job's location (internal/web/web.go:503),
		// GetLdapEnable/GetLdapSyncCron decide whether the LDAP sync job is registered
		// at all and on what schedule (web.go:376-383). Changing them without a restart
		// leaves the old schedule running.
		"timeLocation",
		"ldapEnable",
		"ldapSyncCron",
		// Notifier cron registration (panel_telegram / panel_email). web.Server.Start()
		// decides ONCE whether to register the periodic stats report and the CPU and
		// memory alarm jobs, and on what schedule:
		//   - tgRunTime is the stats-notify schedule (internal/web/web.go:389).
		//   - tgBotEnable gates both the stats-notify job and the callback-hash
		//     cleanup job (web.go:387-403). The bot process itself IS hot-reloaded
		//     (controller/setting.go:165-172 → web.go:650), so a changed token or
		//     chat id needs no restart — but toggling the bot without one leaves it
		//     running with no periodic report.
		//   - tgEnabledEvents/tgCpu/tgMemory and smtpEnable/smtpEnabledEvents/
		//     smtpCpu/smtpMemory feed cpuAlarmWanted()/memoryAlarmWanted()
		//     (web.go:408-412, :439-448, :469-478), which decide whether the alarm
		//     jobs are registered at all — for either notifier.
		"tgRunTime",
		"tgBotEnable",
		"tgEnabledEvents",
		"tgCpu",
		"tgMemory",
		"smtpEnable",
		"smtpEnabledEvents",
		"smtpCpu",
		"smtpMemory",
		// Discord notifier (panel_discord, 3x-ui v3.8.0+). Unlike Telegram, the
		// notify cron AND the gateway are hot-reloaded in-process on settings
		// update (controller.SetReloadDiscordFunc, 3x-ui-3.8.5/internal/web/
		// controller/setting.go:182-189 → web.go:693-719), so discordRunTime,
		// discordBotToken and discordChannelId do NOT restart. discordBotEnable
		// does: cpuAlarmWanted()/memoryAlarmWanted() — which decide whether the
		// CPU/memory alarm samplers are registered at all — read it once at
		// startup (web.go:439-448, :478-482) and the reload func never re-checks
		// alarm registration, so enabling the bot with cpu.high/memory.high
		// configured must restart or the samplers never appear. The thresholds
		// and event list follow the same derived rules as tg/smtp below.
		"discordBotEnable",
		"discordEnabledEvents",
		"discordCpu",
		"discordMemory",
		// Subscription server binding — parallels the web* keys above. The sub server is
		// (re)initialised at panel startup, so changing whether/where it listens needs a
		// panel restart; without it the subscription URL 404s until the panel is restarted.
		"subEnable",
		"subListen",
		"subDomain",
		"subPort",
		"subPath",
		"subCertFile",
		"subKeyFile",
		// Subscription server body/route settings. Every one of these is read inside
		// (*sub.Server).initRouter() and frozen into the SUBController it builds
		// (3x-ui-3.7.0/internal/sub/sub.go:50-301), and initRouter runs only from
		// Start(), which main.go calls at boot and on SIGHUP. A change that is not
		// followed by a restart applies to the panel DB and to Terraform state while
		// every served subscription keeps the old value — the same silent no-op as the
		// binding keys above (#291), reported as #443.
		//
		// Route registration: these decide which paths the gin engine even serves, so
		// a change without a restart 404s on the new path and keeps serving the old one.
		"subJsonPath",
		"subClashPath",
		"subJsonEnable",
		"subClashEnable",
		// JSON/Clash body content.
		"subJsonMux",
		"subJsonRules",
		"subJsonFinalMask",
		"subJsonObservatory",
		"subClashRules",
		"subClashEnableRouting",
		// Client-detection and body-shape switches.
		"subJsonAutoDetect",
		"subJsonAlwaysArray",
		"subJsonUserAgentRegex",
		"subClashAutoDetect",
		"subClashUserAgentRegex",
		"subEncrypt",
		"subUpdates",
		"remarkTemplate",
		// Page/link presentation. These look like the per-request link-generation
		// fields (subURI, subJsonURI, subClashURI — deliberately NOT listed here), but
		// they are not: initRouter reads them once and hands them to the controller.
		"subTitle",
		"subSupportUrl",
		"subProfileUrl",
		"subAnnounce",
		"subHideSettings",
		"subEnableRouting",
		"subRoutingRules",
		"subIncyEnableRouting",
		"subIncyRoutingRules",
		// v3.8.0 additions frozen by (*sub.Server).initRouter():
		// subProfileMode decides whether the profile-page route is registered
		// at all (sub.go:197), subJsonRoutingRules/subJsonDns are baked into
		// the JSON-subscription controller (sub.go:153-161), and the whole
		// Happ customization block is captured into happCfg (sub.go:233-256).
		// The per-request exceptions (NOT here): subCalendarExpireInclusive
		// (PrepareForRequest, service.go:116), subInfoNodeEnable and the
		// expired/depleted templates (loadRemarkSettings, service.go:251-257),
		// happLinkEnable (HappService gate, web/service/happ.go:89).
		"subProfileMode",
		"subJsonRoutingRules",
		"subJsonDns",
		"subHappAutoDetect",
		"subHappProviderId",
		"subHappNewUrl",
		"subHappFallbackUrl",
		"subHappSubInfoColor",
		"subHappSubInfoText",
		"subHappSubInfoButtonText",
		"subHappSubInfoButtonLink",
		"subHappSubExpire",
		"subHappSubExpireButtonLink",
		"subHappNotificationExpire",
		"subHappNoLimit",
		"subHappAlwaysHwid",
		"subHappTunMode",
		"subHappTunType",
		"subHappExcludeRoutes",
		"subHappExcludeApns",
		"subHappColorProfile",
		"subHappPingType",
		"subHappAutoConnect",
		"subHappAutoConnectType",
		"subHappPerAppMode",
		"subHappPerAppList",
	}
	for _, key := range restartKeys {
		newVal, ok := desired[key]
		if !ok {
			continue
		}
		oldVal, ok := existing[key]
		if !ok {
			// The panel does not report this key at all, which means its version
			// predates the setting: /setting/update will drop it just as
			// /setting/all omitted it. Restarting would bounce the panel on every
			// apply for a value that is never stored — 7 of the 13 versions in
			// compat-versions.json predate the smtp*/tg* keys below.
			continue
		}
		if changed, ok := restartKeyRules[key]; ok {
			if changed(oldVal, newVal) {
				return true
			}
			continue
		}
		if !settingsValueEqual(oldVal, newVal) {
			return true
		}
	}
	return false
}

// restartKeyRules holds the keys where a changed value does not necessarily
// change what the panel wired up at startup. Restarting on every edit of these
// would take the panel down for a change it cannot observe.
var restartKeyRules = map[string]func(oldVal, newVal any) bool{
	// The alarm thresholds decide only WHETHER the sampler job is registered:
	// cpuAlarmWanted()/memoryAlarmWanted() test `threshold <= 0` and nothing else
	// (3x-ui-3.7.0/internal/web/web.go:428-431, :458-461). The job itself
	// publishes a raw metric and never reads the threshold
	// (internal/web/job/check_cpu_usage.go:20-33); the comparison happens per
	// event in the notifier. So 80 → 90 changes nothing about registration and
	// takes effect immediately, while 0 → 80 has to register the job.
	"tgCpu":      alarmThresholdCrossesZero,
	"tgMemory":   alarmThresholdCrossesZero,
	"smtpCpu":    alarmThresholdCrossesZero,
	"smtpMemory": alarmThresholdCrossesZero,
	// Same shape for the event lists: registration turns on the membership of
	// cpu.high / memory.high (web.go:432-436, :462-466). Every other event is
	// filtered per event at delivery time, so adding "backup" to the list must
	// not bounce the panel.
	"tgEnabledEvents":   alarmEventMembershipChanged,
	"smtpEnabledEvents": alarmEventMembershipChanged,
	// Discord mirrors the tg/smtp alarm shape (web.go:478-482 and the memory
	// analog): the sampler jobs test threshold <= 0 and cpu.high/memory.high
	// membership once, at startup.
	"discordCpu":           alarmThresholdCrossesZero,
	"discordMemory":        alarmThresholdCrossesZero,
	"discordEnabledEvents": alarmEventMembershipChanged,
}

// alarmThresholdCrossesZero reports whether a threshold change flips the
// setting between "off" (<= 0) and "on".
func alarmThresholdCrossesZero(oldVal, newVal any) bool {
	return intValue(oldVal) <= 0 != (intValue(newVal) <= 0)
}

// alarmEventMembershipChanged reports whether the comma-separated event list
// gained or lost one of the two events that gate a cron job.
func alarmEventMembershipChanged(oldVal, newVal any) bool {
	for _, event := range []string{"cpu.high", "memory.high"} {
		if eventListContains(oldVal, event) != eventListContains(newVal, event) {
			return true
		}
	}
	return false
}

func eventListContains(value any, event string) bool {
	list, ok := value.(string)
	if !ok {
		return false
	}
	for _, candidate := range strings.Split(list, ",") {
		if strings.TrimSpace(candidate) == event {
			return true
		}
	}
	return false
}

func settingsValueEqual(a, b any) bool {
	switch av := a.(type) {
	case nil:
		return b == nil
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case float64:
		return numberValueEqual(av, b)
	case float32:
		return numberValueEqual(float64(av), b)
	case int:
		return numberValueEqual(float64(av), b)
	case int8:
		return numberValueEqual(float64(av), b)
	case int16:
		return numberValueEqual(float64(av), b)
	case int32:
		return numberValueEqual(float64(av), b)
	case int64:
		return numberValueEqual(float64(av), b)
	case uint:
		return numberValueEqual(float64(av), b)
	case uint8:
		return numberValueEqual(float64(av), b)
	case uint16:
		return numberValueEqual(float64(av), b)
	case uint32:
		return numberValueEqual(float64(av), b)
	case uint64:
		return numberValueEqual(float64(av), b)
	default:
		return false
	}
}

func numberValueEqual(a float64, b any) bool {
	switch bv := b.(type) {
	case float64:
		return a == bv
	case float32:
		return a == float64(bv)
	case int:
		return a == float64(bv)
	case int8:
		return a == float64(bv)
	case int16:
		return a == float64(bv)
	case int32:
		return a == float64(bv)
	case int64:
		return a == float64(bv)
	case uint:
		return a == float64(bv)
	case uint8:
		return a == float64(bv)
	case uint16:
		return a == float64(bv)
	case uint32:
		return a == float64(bv)
	case uint64:
		return a == float64(bv)
	default:
		return false
	}
}
