package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The PanelDiscordResource CRUD bodies, driven against an httptest panel the
// same way resource_host_group_test.go drives HostGroupResource. The mock
// mirrors v3.8.x behaviour: /setting/all returns AllSettingView with
// discordBotToken blanked, and /setting/update accepts the posted keys.

// panelDiscordServer is a fake settings panel: GET /setting/all returns
// `settings` (merged over whatever /setting/update stored), and every update
// body is captured for assertions.
func panelDiscordServer(t *testing.T, settings map[string]any) (*httptest.Server, *[]map[string]any) {
	t.Helper()
	mu := sync.Mutex{}
	server := map[string]any{}
	for k, v := range settings {
		server[k] = v
	}
	updates := &[]map[string]any{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			_, _ = w.Write(okResponse(nil))
		case "/panel/api/setting/all":
			mu.Lock()
			snapshot := make(map[string]any, len(server))
			for k, v := range server {
				snapshot[k] = v
			}
			mu.Unlock()
			_, _ = w.Write(okResponse(snapshot))
		case "/panel/api/setting/restartPanel":
			_, _ = w.Write(okResponse(nil))
		case "/panel/api/setting/update":
			var in map[string]any
			_ = json.NewDecoder(r.Body).Decode(&in)
			mu.Lock()
			*updates = append(*updates, in)
			for k, v := range in {
				server[k] = v
			}
			mu.Unlock()
			_, _ = w.Write(okResponse(nil))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, updates
}

func panelDiscordClient(t *testing.T, url string) *Client {
	t.Helper()
	c := newTestClient(t, url)
	c.settingsAPIMu.Lock()
	v := true
	c.settingsUnderAPI = &v
	c.settingsAPIMu.Unlock()
	return c
}

// panelDiscordPlan builds a tfsdk.Plan (usable as Plan, Config or State) from
// a partial attribute map; everything unspecified is a typed null.
func panelDiscordPlan(t *testing.T, r *PanelDiscordResource, vals map[string]any) tfsdk.Plan {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	ctx := context.Background()
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	out := map[string]tftypes.Value{}
	for k := range schemaResp.Schema.Attributes {
		out[k] = tftypes.NewValue(objType.AttributeTypes[k], nil)
	}
	for k, v := range vals {
		switch v := v.(type) {
		case string:
			out[k] = tftypes.NewValue(tftypes.String, v)
		case bool:
			out[k] = tftypes.NewValue(tftypes.Bool, v)
		case int64:
			out[k] = tftypes.NewValue(tftypes.Number, float64(v))
		}
	}
	return tfsdk.Plan{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(objType, out),
	}
}

// panelDiscordConfig mirrors panelDiscordPlan for the Config field.
func panelDiscordConfig(t *testing.T, r *PanelDiscordResource, vals map[string]any) tfsdk.Config {
	t.Helper()
	return tfsdk.Config(panelDiscordPlan(t, r, vals))
}

func panelDiscordCreateResponse(t *testing.T, r *PanelDiscordResource) resource.CreateResponse {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	ctx := context.Background()
	return resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		},
	}
}

func TestPanelDiscordResource_Create(t *testing.T) {
	srv, updates := panelDiscordServer(t, map[string]any{
		"discordBotEnable": false,
		"discordBotToken":  "",
		"discordChannelId": "",
		"discordRunTime":   "@daily",
		"discordCpu":       float64(0),
	})
	r := &PanelDiscordResource{client: panelDiscordClient(t, srv.URL)}

	plan := panelDiscordPlan(t, r, map[string]any{
		"id":                     "settings",
		"discord_bot_enable":     true,
		"discord_bot_token":      "plain-token",
		"discord_channel_id":     "1234",
		"discord_run_time":       "@every 6h",
		"discord_cpu":            int64(80),
		"discord_memory":         int64(90),
		"discord_enabled_events": "login,cpu.high",
	})
	resp := panelDiscordCreateResponse(t, r)
	r.Create(context.Background(), resource.CreateRequest{Plan: plan, Config: panelDiscordConfig(t, r, map[string]any{
		"id":                     "settings",
		"discord_bot_enable":     true,
		"discord_bot_token":      "plain-token",
		"discord_channel_id":     "1234",
		"discord_run_time":       "@every 6h",
		"discord_cpu":            int64(80),
		"discord_memory":         int64(90),
		"discord_enabled_events": "login,cpu.high",
	})}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create failed: %v", resp.Diagnostics)
	}

	if len(*updates) != 1 {
		t.Fatalf("expected exactly one /setting/update, got %d", len(*updates))
	}
	sent := (*updates)[0]
	if sent["discordBotEnable"] != true || sent["discordChannelId"] != "1234" {
		t.Fatalf("update body wrong: %#v", sent)
	}
	if sent["discordCpu"] != float64(80) {
		t.Fatalf("discordCpu must serialise as a number: %#v", sent["discordCpu"])
	}

	var state PanelDiscordModel
	if diag := resp.State.Get(context.Background(), &state); diag.HasError() {
		t.Fatalf("state decode: %v", diag)
	}
	if !state.DiscordBotEnable.ValueBool() {
		t.Error("discord_bot_enable should echo the applied value")
	}
	// The panel blanks the token (AllSettingView); the provider must preserve
	// the configured one in state.
	if state.DiscordBotToken.ValueString() != "plain-token" {
		t.Errorf("discord_bot_token must be preserved from the plan, got %q", state.DiscordBotToken.ValueString())
	}
	if state.DiscordRunTime.ValueString() != "@every 6h" {
		t.Errorf("discord_run_time round-trip: %q", state.DiscordRunTime.ValueString())
	}
	// The secret replay cache must remember the configured token.
	r.client.settingsSecretMu.Lock()
	cached := r.client.settingsSecrets["discordBotToken"]
	r.client.settingsSecretMu.Unlock()
	if cached != "plain-token" {
		t.Errorf("settingsSecrets[discordBotToken] = %q, want plain-token", cached)
	}
}

func TestPanelDiscordResource_Create_WriteOnlyToken(t *testing.T) {
	srv, updates := panelDiscordServer(t, map[string]any{"discordBotEnable": false})
	r := &PanelDiscordResource{client: panelDiscordClient(t, srv.URL)}

	// The plan mirrors the config with the framework-nulled _wo attribute.
	plan := panelDiscordPlan(t, r, map[string]any{
		"id":                           "settings",
		"discord_bot_enable":           false,
		"discord_bot_token_wo_version": int64(1),
	})
	resp := panelDiscordCreateResponse(t, r)
	r.Create(context.Background(), resource.CreateRequest{Plan: plan, Config: panelDiscordConfig(t, r, map[string]any{
		"id":                           "settings",
		"discord_bot_enable":           false,
		"discord_bot_token_wo":         "wo-token",
		"discord_bot_token_wo_version": int64(1),
	})}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create failed: %v", resp.Diagnostics)
	}
	if got := (*updates)[0]["discordBotToken"]; got != "wo-token" {
		t.Fatalf("the _wo value must be sent, got %#v", got)
	}
}

func TestPanelDiscordResource_Read_BlankedTokenPreserved(t *testing.T) {
	srv, _ := panelDiscordServer(t, map[string]any{
		"discordBotEnable": true,
		"discordBotToken":  "", // AllSettingView blanks secrets
		"discordChannelId": "1234",
	})
	r := &PanelDiscordResource{client: panelDiscordClient(t, srv.URL)}

	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	ctx := context.Background()
	prior := panelDiscordPlan(t, r, map[string]any{
		"id":                 "settings",
		"discord_bot_enable": true,
		"discord_bot_token":  "prior-token",
	})
	resp := resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: prior.Raw.Copy()},
	}
	r.Read(context.Background(), resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: prior.Raw.Copy()}}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read failed: %v", resp.Diagnostics)
	}
	var state PanelDiscordModel
	if diag := resp.State.Get(ctx, &state); diag.HasError() {
		t.Fatalf("state decode: %v", diag)
	}
	if state.DiscordBotToken.ValueString() != "prior-token" {
		t.Errorf("a blanked panel token must not clobber the stored one, got %q", state.DiscordBotToken.ValueString())
	}
	if state.DiscordChannelID.ValueString() != "1234" {
		t.Errorf("discord_channel_id must read through, got %q", state.DiscordChannelID.ValueString())
	}
}

func TestPanelDiscordResource_Update(t *testing.T) {
	srv, updates := panelDiscordServer(t, map[string]any{
		"discordBotEnable": true,
		"discordLang":      "en-US",
	})
	r := &PanelDiscordResource{client: panelDiscordClient(t, srv.URL)}

	config := panelDiscordPlan(t, r, map[string]any{
		"id":                 "settings",
		"discord_bot_enable": true,
		"discord_lang":       "ru-RU",
		"discord_cpu":        int64(80),
	})
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	resp := resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(context.Background()), nil),
		},
	}
	req := resource.UpdateRequest{
		Plan:   config,
		State:  tfsdk.State{Schema: schemaResp.Schema, Raw: panelDiscordPlan(t, r, map[string]any{"id": "settings", "discord_bot_enable": true}).Raw.Copy()},
		Config: panelDiscordConfig(t, r, map[string]any{"id": "settings", "discord_bot_enable": true, "discord_lang": "ru-RU", "discord_cpu": int64(80)}),
	}
	r.Update(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update failed: %v", resp.Diagnostics)
	}
	if got := (*updates)[0]["discordLang"]; got != "ru-RU" {
		t.Fatalf("discordLang update missing, got %#v", (*updates)[0])
	}
}

func TestPanelDiscordResource_Delete(t *testing.T) {
	srv, _ := panelDiscordServer(t, nil)
	r := &PanelDiscordResource{client: panelDiscordClient(t, srv.URL)}
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.DeleteResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(context.Background()), nil),
		},
	}
	r.Delete(context.Background(), resource.DeleteRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete failed: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatal("Delete must remove the resource from state")
	}
}

func TestPanelDiscordResource_ImportState(t *testing.T) {
	srv, _ := panelDiscordServer(t, map[string]any{
		"discordBotEnable":     false,
		"discordAdminIds":      "111",
		"discordEnabledEvents": "login",
	})
	r := &PanelDiscordResource{client: panelDiscordClient(t, srv.URL)}
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(context.Background()), nil),
		},
	}
	r.ImportState(context.Background(), resource.ImportStateRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState failed: %v", resp.Diagnostics)
	}
	var state PanelDiscordModel
	if diag := resp.State.Get(context.Background(), &state); diag.HasError() {
		t.Fatalf("state decode: %v", diag)
	}
	if state.DiscordAdminIDs.ValueString() != "111" {
		t.Errorf("discord_admin_ids after import: %q", state.DiscordAdminIDs.ValueString())
	}
	if state.DiscordEnabledEvents.ValueString() != "login" {
		t.Errorf("discord_enabled_events after import: %q", state.DiscordEnabledEvents.ValueString())
	}
	if !state.DiscordBotToken.IsUnknown() && state.DiscordBotToken.ValueString() != "" {
		// A blanked token flattens as "" (the panel never returns it) — never
		// an invented value.
		if state.DiscordBotToken.IsNull() {
			t.Errorf("discord_bot_token after import should be an empty string, got null")
		}
	}
}
