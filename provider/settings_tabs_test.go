package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestPanelGeneralSchemaAttributeShape asserts that the v3.2.0/v3.3.1 panel
// egress attributes exist with the expected Optional+Computed shape. It also
// exercises panelGeneralSchema() so the schema literal counts towards unit
// coverage (otherwise every panel_* attribute reads as uncovered by Codecov
// because no unit test instantiates the resource Schema()).
func TestPanelGeneralSchemaAttributeShape(t *testing.T) {
	s := panelGeneralSchema()
	requireStringAttr := func(name string) {
		t.Helper()
		attr, ok := s.Attributes[name]
		if !ok {
			t.Fatalf("attribute %q missing on panel_general schema", name)
		}
		str, ok := attr.(schema.StringAttribute)
		if !ok {
			t.Fatalf("attribute %q is not a StringAttribute", name)
		}
		if !str.Optional {
			t.Errorf("attribute %q must be Optional", name)
		}
		if !str.Computed {
			t.Errorf("attribute %q must be Computed", name)
		}
	}

	// Sanity-check that a well-known string attribute is present (confirms
	// the schema loaded, not just the two egress attrs).
	requireStringAttr("web_domain")

	// v3.2.0–v3.3.0 proxy URL (kept for backward compat).
	requireStringAttr("panel_proxy")

	// v3.3.1 outbound egress bridge (Xray outbound / balancer tag).
	requireStringAttr("panel_outbound")
}

func TestPanelSettingsNeedRestart_ChangedKey(t *testing.T) {
	existing := map[string]any{"webPort": float64(2053)}
	desired := map[string]any{"webPort": float64(8080)}
	if !panelSettingsNeedRestart(existing, desired) {
		t.Fatalf("expected restart needed")
	}
}

func TestPanelSettingsNeedRestart_UnchangedKey(t *testing.T) {
	existing := map[string]any{"webPort": float64(2053)}
	desired := map[string]any{"webPort": float64(2053)}
	if panelSettingsNeedRestart(existing, desired) {
		t.Fatalf("expected no restart")
	}
}

func TestPanelSettingsNeedRestart_NonRestartKey(t *testing.T) {
	existing := map[string]any{"pageSize": float64(25)}
	desired := map[string]any{"pageSize": float64(50)}
	if panelSettingsNeedRestart(existing, desired) {
		t.Fatalf("expected no restart for pageSize")
	}
}

// A key the panel does not report is a key the panel does not have:
// /setting/all serialises AllSetting whole, so an absent key means this version
// predates the setting and /setting/update will drop it exactly as the read
// omitted it. Restarting for a value that is never stored would bounce the
// panel on every apply — and 7 of the 13 versions in compat-versions.json
// predate the notifier keys (#449). This deliberately reverses the behaviour
// added in #443.
func TestPanelSettingsNeedRestart_KeyUnknownToPanel(t *testing.T) {
	if panelSettingsNeedRestart(map[string]any{}, map[string]any{"webBasePath": "/new/"}) {
		t.Error("a key absent from the panel must not trigger a restart: the value cannot be stored either")
	}
	if panelSettingsNeedRestart(map[string]any{}, map[string]any{"smtpCpu": 80}) {
		t.Error("a v3.4.0+ key on a v3.2.x panel must not restart on every apply")
	}
	// A key the panel does report still restarts when it changes.
	if !panelSettingsNeedRestart(map[string]any{"webBasePath": "/old/"}, map[string]any{"webBasePath": "/new/"}) {
		t.Error("expected restart when a known key changes")
	}
}

func TestPanelSettingsNeedRestart_NoRestartKeys(t *testing.T) {
	existing := map[string]any{"webPort": float64(2053)}
	desired := map[string]any{"remarkModel": "-ieo"}
	if panelSettingsNeedRestart(existing, desired) {
		t.Fatalf("expected no restart")
	}
}

func TestPanelSettingsNeedRestart_SubscriptionServerKey(t *testing.T) {
	// Toggling/changing the subscription server must trigger a panel restart — the sub
	// server is initialised at panel startup, so otherwise the subscription URL 404s.
	if !panelSettingsNeedRestart(map[string]any{"subEnable": false}, map[string]any{"subEnable": true}) {
		t.Fatalf("expected restart when the subscription server is enabled")
	}
	if !panelSettingsNeedRestart(map[string]any{"subPath": "/old/"}, map[string]any{"subPath": "/new/"}) {
		t.Fatalf("expected restart when the subscription path changes")
	}
	if !panelSettingsNeedRestart(map[string]any{"subPort": float64(2096)}, map[string]any{"subPort": float64(2097)}) {
		t.Fatalf("expected restart when the subscription port changes")
	}
}

func TestPanelSettingsNeedRestart_SubscriptionLinkKeyNoRestart(t *testing.T) {
	// The three *URI settings are the only subscription keys read at request time
	// (internal/sub/service.go:2704-2706); everything else initRouter touches is
	// frozen at startup and IS gated below.
	for _, key := range []string{"subURI", "subJsonURI", "subClashURI"} {
		if panelSettingsNeedRestart(map[string]any{key: "https://a/"}, map[string]any{key: "https://b/"}) {
			t.Errorf("expected no restart for %s (link-generation only)", key)
		}
	}
}

// Every setting (*sub.Server).initRouter() reads is frozen into the SUBController
// until the panel restarts (3x-ui-3.7.0/internal/sub/sub.go:50-301, reached only
// from Start()). Changing one without a restart is a silent no-op: state matches
// the panel while served subscriptions keep the old value (#443, same class as
// #291).
//
// This is a PIN, not a verification: it restates the list rather than deriving it
// from upstream, so it catches a key being dropped from restartKeys but cannot
// catch a key that upstream adds to initRouter and nobody adds here. The
// derived-from-upstream check for that is the drift gate over AllSetting; when
// adding a panel_subscription attribute, check initRouter by hand.
func TestPanelSettingsNeedRestart_SubServerStartupKeys(t *testing.T) {
	stringKeys := []string{
		"subJsonPath", "subClashPath",
		"subJsonMux", "subJsonRules", "subJsonFinalMask", "subJsonObservatory",
		"subClashRules", "subJsonUserAgentRegex", "subClashUserAgentRegex",
		"subUpdates", "remarkTemplate",
		"subTitle", "subSupportUrl", "subProfileUrl", "subAnnounce",
		"subRoutingRules", "subIncyRoutingRules",
	}
	for _, key := range stringKeys {
		if !panelSettingsNeedRestart(map[string]any{key: "old"}, map[string]any{key: "new"}) {
			t.Errorf("expected restart when %s changes", key)
		}
	}

	boolKeys := []string{
		"subJsonEnable", "subClashEnable",
		"subJsonAutoDetect", "subJsonAlwaysArray", "subClashAutoDetect",
		"subClashEnableRouting", "subEncrypt", "subHideSettings",
		"subEnableRouting", "subIncyEnableRouting",
	}
	for _, key := range boolKeys {
		if !panelSettingsNeedRestart(map[string]any{key: false}, map[string]any{key: true}) {
			t.Errorf("expected restart when %s changes", key)
		}
	}
}

// The panel's own cron wiring is read once in web.Server.Start(): the timezone
// every job is scheduled in (internal/web/web.go:503) and whether the LDAP sync
// job is registered at all, on what schedule (web.go:376-383).
func TestPanelSettingsNeedRestart_PanelStartupKeys(t *testing.T) {
	if !panelSettingsNeedRestart(map[string]any{"timeLocation": "UTC"}, map[string]any{"timeLocation": "Europe/Moscow"}) {
		t.Error("expected restart when timeLocation changes")
	}
	if !panelSettingsNeedRestart(map[string]any{"ldapEnable": false}, map[string]any{"ldapEnable": true}) {
		t.Error("expected restart when ldapEnable changes")
	}
	if !panelSettingsNeedRestart(map[string]any{"ldapSyncCron": "@every 1m"}, map[string]any{"ldapSyncCron": "@hourly"}) {
		t.Error("expected restart when ldapSyncCron changes")
	}
}

// The panel registers its notifier cron jobs once, in web.Server.Start(): the
// stats-notify schedule (internal/web/web.go:389), whether the Telegram bot's
// jobs exist at all (web.go:387-403), and whether the CPU/memory alarm jobs are
// registered for either notifier (web.go:408-412, :439-448, :469-478). Changing
// any of them without a restart leaves the old schedule running (#449).
func TestPanelSettingsNeedRestart_NotifierCronKeys(t *testing.T) {
	for key, change := range map[string][2]any{
		"tgRunTime":         {"@daily", "@every 6h"},
		"tgEnabledEvents":   {"client.expiry", "client.expiry,cpu.high"},
		"smtpEnabledEvents": {"", "memory.high"},
	} {
		if !panelSettingsNeedRestart(map[string]any{key: change[0]}, map[string]any{key: change[1]}) {
			t.Errorf("expected restart when %s changes", key)
		}
	}

	// Only cpu.high / memory.high membership gates a cron job; every other event
	// is filtered at delivery time, so the list changing around them is not worth
	// a restart.
	for _, key := range []string{"tgEnabledEvents", "smtpEnabledEvents"} {
		if panelSettingsNeedRestart(
			map[string]any{key: "login.attempt,cpu.high"},
			map[string]any{key: "login.attempt,backup,cpu.high"},
		) {
			t.Errorf("adding an unrelated event to %s must not restart the panel", key)
		}
		if !panelSettingsNeedRestart(
			map[string]any{key: "login.attempt,cpu.high"},
			map[string]any{key: "login.attempt"},
		) {
			t.Errorf("dropping cpu.high from %s must restart: the sampler job is deregistered", key)
		}
		if panelSettingsNeedRestart(
			map[string]any{key: "cpu.high,memory.high"},
			map[string]any{key: " cpu.high , memory.high "},
		) {
			t.Errorf("whitespace-only differences in %s must not restart the panel", key)
		}
	}
	for _, key := range []string{"tgCpu", "tgMemory", "smtpCpu", "smtpMemory"} {
		if !panelSettingsNeedRestart(map[string]any{key: float64(0)}, map[string]any{key: 80}) {
			t.Errorf("expected restart when %s changes", key)
		}
	}
	for _, key := range []string{"tgBotEnable", "smtpEnable"} {
		if !panelSettingsNeedRestart(map[string]any{key: false}, map[string]any{key: true}) {
			t.Errorf("expected restart when %s changes", key)
		}
	}
}

// The Telegram bot process is hot-reloaded by the panel, so its credentials must
// NOT bounce the panel — only the settings that gate cron registration do.
func TestPanelSettingsNeedRestart_HotReloadedKeysStayQuiet(t *testing.T) {
	for key, change := range map[string][2]any{
		"tgBotToken":     {"old", "new"},
		"tgBotChatId":    {"1", "2"},
		"tgBotAPIServer": {"", "https://api.example"},
		"smtpHost":       {"old.example", "new.example"},
		"smtpUser":       {"a@example", "b@example"},
	} {
		if panelSettingsNeedRestart(map[string]any{key: change[0]}, map[string]any{key: change[1]}) {
			t.Errorf("%s is applied without a restart and must not trigger one", key)
		}
	}
}

// A restart is only justified by a real change: re-applying identical values for
// the newly gated keys must stay quiet, or every no-op apply bounces the panel.
func TestPanelSettingsNeedRestart_UnchangedSubServerKeysStayQuiet(t *testing.T) {
	settings := map[string]any{
		"subJsonMux":         `{"enabled":true}`,
		"subJsonRules":       "[]",
		"subJsonObservatory": "",
		"subTitle":           "My subscription",
		"subJsonEnable":      true,
		"subJsonPath":        "/json/",
		"timeLocation":       "UTC",
		"ldapEnable":         false,
	}
	same := make(map[string]any, len(settings))
	for k, v := range settings {
		same[k] = v
	}
	if panelSettingsNeedRestart(settings, same) {
		t.Fatal("expected no restart when nothing changed")
	}
}

func TestSettingsValueEqual_Strings(t *testing.T) {
	if !settingsValueEqual("abc", "abc") {
		t.Fatalf("expected equal")
	}
	if settingsValueEqual("abc", "def") {
		t.Fatalf("expected not equal")
	}
}

func TestSettingsValueEqual_Bools(t *testing.T) {
	if !settingsValueEqual(true, true) {
		t.Fatalf("expected equal")
	}
	if settingsValueEqual(true, false) {
		t.Fatalf("expected not equal")
	}
}

func TestSettingsValueEqual_FloatAndInt(t *testing.T) {
	if !settingsValueEqual(float64(42), int(42)) {
		t.Fatalf("expected equal")
	}
	if !settingsValueEqual(float64(42), int64(42)) {
		t.Fatalf("expected equal for int64")
	}
}

func TestSettingsValueEqual_IntAndFloat(t *testing.T) {
	if !settingsValueEqual(int(42), float64(42)) {
		t.Fatalf("expected equal")
	}
}

func TestSettingsValueEqual_Nil(t *testing.T) {
	if !settingsValueEqual(nil, nil) {
		t.Fatalf("expected nil == nil")
	}
	if settingsValueEqual(nil, "x") {
		t.Fatalf("expected nil != string")
	}
}

func TestSettingsValueEqual_DifferentTypes(t *testing.T) {
	if settingsValueEqual("42", float64(42)) {
		t.Fatalf("expected not equal for string vs number")
	}
}

func TestNumberValueEqual_Float64(t *testing.T) {
	if !numberValueEqual(3.14, float64(3.14)) {
		t.Fatalf("expected equal")
	}
}

func TestNumberValueEqual_Int(t *testing.T) {
	if !numberValueEqual(42.0, int(42)) {
		t.Fatalf("expected equal")
	}
}

func TestNumberValueEqual_Int64(t *testing.T) {
	if !numberValueEqual(100.0, int64(100)) {
		t.Fatalf("expected equal")
	}
}

func TestNumberValueEqual_Uint(t *testing.T) {
	if !numberValueEqual(10.0, uint(10)) {
		t.Fatalf("expected equal")
	}
}

func TestNumberValueEqual_NonNumeric(t *testing.T) {
	if numberValueEqual(42.0, "42") {
		t.Fatalf("expected not equal for string")
	}
}

func TestMergeSettings_BothNil(t *testing.T) {
	result := mergeSettings(nil, nil)
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %v", result)
	}
}

func TestMergeSettings_NilExisting(t *testing.T) {
	desired := map[string]any{"a": "1"}
	result := mergeSettings(nil, desired)
	if result["a"] != "1" {
		t.Fatalf("expected desired, got %v", result)
	}
}

func TestMergeSettings_Overlap(t *testing.T) {
	existing := map[string]any{"a": "old", "b": "keep"}
	desired := map[string]any{"a": "new"}
	result := mergeSettings(existing, desired)
	if result["a"] != "new" {
		t.Fatalf("expected override, got %v", result["a"])
	}
	if result["b"] != "keep" {
		t.Fatalf("expected keep, got %v", result["b"])
	}
}

func TestMergeSettings_Union(t *testing.T) {
	existing := map[string]any{"a": "1"}
	desired := map[string]any{"b": "2"}
	result := mergeSettings(existing, desired)
	if result["a"] != "1" || result["b"] != "2" {
		t.Fatalf("expected union, got %v", result)
	}
}

func TestFlattenPanelSecurity(t *testing.T) {
	in := map[string]any{
		"twoFactorEnable": true,
		"twoFactorToken":  "secret",
	}
	m := flattenPanelSecurity(in)
	if m.TwoFactorEnable.ValueBool() != true {
		t.Fatalf("unexpected two_factor_enable: %v", m.TwoFactorEnable)
	}
	if m.TwoFactorToken.ValueString() != "secret" {
		t.Fatalf("unexpected two_factor_token: %v", m.TwoFactorToken)
	}
}

func TestFlattenPanelTelegram(t *testing.T) {
	in := map[string]any{
		"tgBotEnable":      true,
		"tgBotToken":       "tok",
		"tgBotProxy":       "proxy",
		"tgBotAPIServer":   "api",
		"tgBotChatId":      "123",
		"tgLang":           "en",
		"tgRunTime":        "@daily",
		"tgBotBackup":      true,
		"tgBotLoginNotify": false,
		"tgCpu":            float64(80),
	}
	m := flattenPanelTelegram(in)
	if m.TgBotEnable.ValueBool() != true {
		t.Fatalf("unexpected tg_bot_enable: %v", m.TgBotEnable)
	}
	if m.TgBotToken.ValueString() != "tok" {
		t.Fatalf("unexpected tg_bot_token: %v", m.TgBotToken)
	}
	if m.TgCPU.ValueInt64() != 80 {
		t.Fatalf("unexpected tg_cpu: %v", m.TgCPU)
	}
}

func TestFlattenPanelSubscription(t *testing.T) {
	in := map[string]any{
		"subEnable":        true,
		"subJsonEnable":    false,
		"subTitle":         "My Sub",
		"subPort":          float64(443),
		"subEncrypt":       true,
		"subEmailInRemark": false,
	}
	m := flattenPanelSubscription(in)
	if m.SubEnable.ValueBool() != true {
		t.Fatalf("unexpected sub_enable: %v", m.SubEnable)
	}
	if m.SubTitle.ValueString() != "My Sub" {
		t.Fatalf("unexpected sub_title: %v", m.SubTitle)
	}
	if m.SubPort.ValueInt64() != 443 {
		t.Fatalf("unexpected sub_port: %v", m.SubPort)
	}
	if m.SubEmailInRemark.ValueBool() {
		t.Fatalf("unexpected sub_email_in_remark: %v", m.SubEmailInRemark)
	}
}

func TestFlattenPanelGeneral(t *testing.T) {
	in := map[string]any{
		"webListen":                  "0.0.0.0",
		"webPort":                    float64(2053),
		"webBasePath":                "/panel/",
		"trustedProxyCIDRs":          "127.0.0.1/32,10.0.0.0/8",
		"pageSize":                   float64(25),
		"restartXrayOnClientDisable": true,
	}
	m := flattenPanelGeneral(in)
	if m.WebListen.ValueString() != "0.0.0.0" {
		t.Fatalf("unexpected web_listen: %v", m.WebListen)
	}
	if m.WebPort.ValueInt64() != 2053 {
		t.Fatalf("unexpected web_port: %v", m.WebPort)
	}
	if m.WebBasePath.ValueString() != "/panel/" {
		t.Fatalf("unexpected web_base_path: %v", m.WebBasePath)
	}
	if m.TrustedProxyCIDRs.ValueString() != "127.0.0.1/32,10.0.0.0/8" {
		t.Fatalf("unexpected trusted_proxy_cidrs: %v", m.TrustedProxyCIDRs)
	}
	if !m.RestartXrayOnClientDisable.ValueBool() {
		t.Fatalf("unexpected restart_xray_on_client_disable: %v", m.RestartXrayOnClientDisable)
	}
	expanded := expandPanelGeneral(m)
	if expanded["trustedProxyCIDRs"] != "127.0.0.1/32,10.0.0.0/8" {
		t.Fatalf("unexpected expanded trustedProxyCIDRs: %v", expanded["trustedProxyCIDRs"])
	}
	if expanded["restartXrayOnClientDisable"] != true {
		t.Fatalf("unexpected expanded restartXrayOnClientDisable: %v", expanded["restartXrayOnClientDisable"])
	}
}

func TestPanelProxyExpandFlatten(t *testing.T) {
	t.Run("proxy set", func(t *testing.T) {
		m := &PanelGeneralModel{
			PanelProxy: types.StringValue("socks5://proxy:1080"),
		}
		expanded := expandPanelGeneral(m)
		if expanded["panelProxy"] != "socks5://proxy:1080" {
			t.Fatalf("expected panelProxy=socks5://proxy:1080, got %v", expanded["panelProxy"])
		}
	})

	t.Run("proxy null", func(t *testing.T) {
		m := &PanelGeneralModel{
			PanelProxy: types.StringNull(),
		}
		expanded := expandPanelGeneral(m)
		if _, ok := expanded["panelProxy"]; ok {
			t.Fatalf("expected no panelProxy key, got %v", expanded["panelProxy"])
		}
	})

	t.Run("flatten with value", func(t *testing.T) {
		in := map[string]any{"panelProxy": "http://proxy:8080"}
		m := flattenPanelGeneral(in)
		if m.PanelProxy.ValueString() != "http://proxy:8080" {
			t.Fatalf("expected http://proxy:8080, got %q", m.PanelProxy)
		}
	})

	t.Run("flatten empty string", func(t *testing.T) {
		in := map[string]any{"panelProxy": ""}
		m := flattenPanelGeneral(in)
		if m.PanelProxy.ValueString() != "" {
			t.Fatalf("expected empty string for empty panelProxy, got %q", m.PanelProxy)
		}
	})

	t.Run("flatten missing key", func(t *testing.T) {
		in := map[string]any{}
		m := flattenPanelGeneral(in)
		if !m.PanelProxy.IsNull() {
			t.Fatalf("expected null for missing panelProxy, got %q", m.PanelProxy)
		}
	})
}

func TestPanelOutboundExpandFlatten(t *testing.T) {
	t.Run("outbound set", func(t *testing.T) {
		m := &PanelGeneralModel{
			PanelOutbound: types.StringValue("proxy-out"),
		}
		expanded := expandPanelGeneral(m)
		if expanded["panelOutbound"] != "proxy-out" {
			t.Fatalf("expected panelOutbound=proxy-out, got %v", expanded["panelOutbound"])
		}
	})

	t.Run("outbound null", func(t *testing.T) {
		m := &PanelGeneralModel{
			PanelOutbound: types.StringNull(),
		}
		expanded := expandPanelGeneral(m)
		if _, ok := expanded["panelOutbound"]; ok {
			t.Fatalf("expected no panelOutbound key, got %v", expanded["panelOutbound"])
		}
	})

	t.Run("flatten with value", func(t *testing.T) {
		in := map[string]any{"panelOutbound": "proxy-out"}
		m := flattenPanelGeneral(in)
		if m.PanelOutbound.ValueString() != "proxy-out" {
			t.Fatalf("expected proxy-out, got %q", m.PanelOutbound)
		}
	})

	t.Run("flatten empty string", func(t *testing.T) {
		in := map[string]any{"panelOutbound": ""}
		m := flattenPanelGeneral(in)
		if m.PanelOutbound.ValueString() != "" {
			t.Fatalf("expected empty string for empty panelOutbound, got %q", m.PanelOutbound)
		}
	})

	t.Run("flatten missing key", func(t *testing.T) {
		in := map[string]any{}
		m := flattenPanelGeneral(in)
		if !m.PanelOutbound.IsNull() {
			t.Fatalf("expected null for missing panelOutbound, got %q", m.PanelOutbound)
		}
	})
}

func TestLDAPInsecureSkipVerifyExpandFlatten(t *testing.T) {
	t.Run("set true", func(t *testing.T) {
		m := &PanelGeneralModel{
			LDAPInsecureSkipVerify: types.BoolValue(true),
		}
		expanded := expandPanelGeneral(m)
		if expanded["ldapInsecureSkipVerify"] != true {
			t.Fatalf("expected ldapInsecureSkipVerify=true, got %v", expanded["ldapInsecureSkipVerify"])
		}
	})

	t.Run("null omits key", func(t *testing.T) {
		m := &PanelGeneralModel{
			LDAPInsecureSkipVerify: types.BoolNull(),
		}
		expanded := expandPanelGeneral(m)
		if _, ok := expanded["ldapInsecureSkipVerify"]; ok {
			t.Fatalf("expected no ldapInsecureSkipVerify key, got %v", expanded["ldapInsecureSkipVerify"])
		}
	})

	t.Run("flatten with value", func(t *testing.T) {
		in := map[string]any{"ldapInsecureSkipVerify": true}
		m := flattenPanelGeneral(in)
		if !m.LDAPInsecureSkipVerify.ValueBool() {
			t.Fatalf("expected true, got %v", m.LDAPInsecureSkipVerify)
		}
	})

	t.Run("flatten missing key stays null", func(t *testing.T) {
		// Old panels (v3.4.1 and earlier) never return this key — the
		// Optional+Computed attr must stay null until the panel sends it.
		in := map[string]any{}
		m := flattenPanelGeneral(in)
		if !m.LDAPInsecureSkipVerify.IsNull() {
			t.Fatalf("expected null for missing key on old panels, got %v", m.LDAPInsecureSkipVerify)
		}
	})
}

func TestPreserveSettingSecret_ConfiguredEmptyObservedNonEmpty(t *testing.T) {
	got := preserveSettingSecret(types.StringValue("existing-secret"), types.StringValue(""))
	if got.ValueString() != "" {
		t.Fatalf("expected configured empty string, got %q", got.ValueString())
	}
}

func TestPreserveSettingSecret_ConfiguredNonEmptyObservedEmpty(t *testing.T) {
	got := preserveSettingSecret(types.StringValue(""), types.StringValue("configured-secret"))
	if got.ValueString() != "configured-secret" {
		t.Fatalf("expected configured secret, got %q", got.ValueString())
	}
}

func TestPreserveSettingSecret_ConfiguredNonEmptyObservedRedacted(t *testing.T) {
	got := preserveSettingSecret(types.StringValue("********"), types.StringValue("configured-secret"))
	if got.ValueString() != "configured-secret" {
		t.Fatalf("expected configured secret, got %q", got.ValueString())
	}
}

func TestPreserveSettingSecret_ObservedDifferentNonEmpty(t *testing.T) {
	got := preserveSettingSecret(types.StringValue("remote-secret"), types.StringValue("state-secret"))
	if got.ValueString() != "remote-secret" {
		t.Fatalf("expected observed secret, got %q", got.ValueString())
	}
}

// preserveRemoved* echoes the configured value when the panel returns null for
// a field it dropped upstream (remarkModel, tgBotLoginNotify, subShowInfo,
// panelProxy, ...). It must return the configured value when observed is
// null/unknown (the v3.4.0 / v3.3.1+ case) and the observed value otherwise
// (older panels that still return the field).

func TestPreserveRemovedString_ObservedNull(t *testing.T) {
	got := preserveRemovedString(types.StringNull(), types.StringValue("-ieo"))
	if got.ValueString() != "-ieo" {
		t.Fatalf("expected configured echoed, got %q", got.ValueString())
	}
}

// Regression: when there is nothing concrete to echo (configured unknown, the
// normal case for a Computed attr the user did not set), we must return null,
// NOT unknown — otherwise Terraform errors "all values must be known after apply".
func TestPreserveRemovedString_ObservedNullConfiguredUnknown(t *testing.T) {
	got := preserveRemovedString(types.StringNull(), types.StringUnknown())
	if !got.IsNull() {
		t.Fatalf("expected null (not unknown) when nothing to echo, got %v", got)
	}
}

func TestPreserveRemovedString_ObservedNullConfiguredNull(t *testing.T) {
	got := preserveRemovedString(types.StringNull(), types.StringNull())
	if !got.IsNull() {
		t.Fatalf("expected null when both null, got %v", got)
	}
}

func TestPreserveRemovedString_ObservedUnknown(t *testing.T) {
	got := preserveRemovedString(types.StringUnknown(), types.StringValue("-ieo"))
	if got.ValueString() != "-ieo" {
		t.Fatalf("expected configured echoed, got %q", got.ValueString())
	}
}

func TestPreserveRemovedString_ObservedNonEmpty(t *testing.T) {
	// older panels still return the field — observed wins, no drift
	got := preserveRemovedString(types.StringValue("remote"), types.StringValue("state"))
	if got.ValueString() != "remote" {
		t.Fatalf("expected observed to win on panels that return it, got %q", got.ValueString())
	}
}

func TestPreserveRemovedBool_ObservedNull(t *testing.T) {
	got := preserveRemovedBool(types.BoolNull(), types.BoolValue(true))
	if !got.ValueBool() {
		t.Fatalf("expected configured true echoed, got false")
	}
}

// Regression: unknown configured must yield null, not unknown (see string variant).
func TestPreserveRemovedBool_ObservedNullConfiguredUnknown(t *testing.T) {
	got := preserveRemovedBool(types.BoolNull(), types.BoolUnknown())
	if !got.IsNull() {
		t.Fatalf("expected null (not unknown) when nothing to echo, got %v", got)
	}
}

func TestPreserveRemovedBool_ObservedUnknown(t *testing.T) {
	got := preserveRemovedBool(types.BoolUnknown(), types.BoolValue(true))
	if !got.ValueBool() {
		t.Fatalf("expected configured true echoed, got false")
	}
}

func TestPreserveRemovedBool_ObservedSet(t *testing.T) {
	got := preserveRemovedBool(types.BoolValue(false), types.BoolValue(true))
	if got.ValueBool() {
		t.Fatalf("expected observed to win on panels that return it, got true")
	}
}

func TestMergeSettingsForUpdate_PreservesCachedRedactedSecret(t *testing.T) {
	client := &Client{}
	client.rememberConfiguredSettingSecrets(map[string]any{"tgBotToken": "configured-token"})

	existing := map[string]any{
		"pageSize":     float64(25),
		"tgBotToken":   "",
		"ldapPassword": "********",
	}
	client.rememberConfiguredSettingSecrets(map[string]any{"ldapPassword": "configured-password"})

	merged := mergeSettingsForUpdate(client, existing, map[string]any{"pageSize": 50})
	if merged["tgBotToken"] != "configured-token" {
		t.Fatalf("expected cached tgBotToken, got %v", merged["tgBotToken"])
	}
	if merged["ldapPassword"] != "configured-password" {
		t.Fatalf("expected cached ldapPassword, got %v", merged["ldapPassword"])
	}
	if existing["tgBotToken"] != "" {
		t.Fatalf("mergeSettingsForUpdate mutated existing tgBotToken: %v", existing["tgBotToken"])
	}
}

func TestMergeSettingsForUpdate_DoesNotOverrideDesiredSecret(t *testing.T) {
	client := &Client{}
	client.rememberConfiguredSettingSecrets(map[string]any{"tgBotToken": "configured-token"})

	merged := mergeSettingsForUpdate(
		client,
		map[string]any{"pageSize": float64(25), "tgBotToken": ""},
		map[string]any{"pageSize": 50, "tgBotToken": ""},
	)
	if merged["tgBotToken"] != "" {
		t.Fatalf("expected desired empty tgBotToken, got %v", merged["tgBotToken"])
	}
}

func TestExpandPanelSecurity(t *testing.T) {
	m := &PanelSecurityModel{}
	m.TwoFactorEnable = typeBoolValue(false)
	m.TwoFactorToken = typeStringValue("tok")
	result := expandPanelSecurity(m)
	if result["twoFactorEnable"] != false {
		t.Fatalf("unexpected twoFactorEnable: %v", result["twoFactorEnable"])
	}
	if result["twoFactorToken"] != "tok" {
		t.Fatalf("unexpected twoFactorToken: %v", result["twoFactorToken"])
	}
}

func TestExpandPanelTelegram(t *testing.T) {
	m := &PanelTelegramModel{}
	m.TgBotEnable = typeBoolValue(true)
	m.TgBotToken = typeStringValue("token")
	m.TgCPU = typeInt64Value(80)
	result := expandPanelTelegram(m)
	if result["tgBotEnable"] != true {
		t.Fatalf("unexpected tgBotEnable: %v", result["tgBotEnable"])
	}
	if result["tgCpu"] != 80 {
		t.Fatalf("unexpected tgCpu: %v", result["tgCpu"])
	}
}

func TestFlattenPanelSubscription_ClashFields(t *testing.T) {
	in := map[string]any{
		"subEnable":             true,
		"subJsonEnable":         false,
		"subClashEnable":        true,
		"subClashPath":          "/clash/",
		"subClashURI":           "https://example.com/clash/",
		"subClashEnableRouting": true,
		"subClashRules":         "rule1,rule2",
		"subJsonFinalMask":      `{"tcp":"mask1"}`,
	}
	m := flattenPanelSubscription(in)
	if !m.SubClashEnable.ValueBool() {
		t.Fatalf("expected sub_clash_enable true")
	}
	if m.SubClashPath.ValueString() != "/clash/" {
		t.Fatalf("unexpected sub_clash_path: %s", m.SubClashPath.ValueString())
	}
	if m.SubClashURI.ValueString() != "https://example.com/clash/" {
		t.Fatalf("unexpected sub_clash_uri: %s", m.SubClashURI.ValueString())
	}
	if !m.SubClashEnableRouting.ValueBool() {
		t.Fatalf("expected sub_clash_enable_routing true")
	}
	if m.SubClashRules.ValueString() != "rule1,rule2" {
		t.Fatalf("unexpected sub_clash_rules: %s", m.SubClashRules.ValueString())
	}
	if m.SubJsonFinalMask.ValueString() != `{"tcp":"mask1"}` {
		t.Fatalf("unexpected sub_json_final_mask: %s", m.SubJsonFinalMask.ValueString())
	}
}

func TestExpandPanelSubscription_v328Fields(t *testing.T) {
	m := &PanelSubscriptionModel{
		SubEnable:             typeBoolValue(true),
		SubClashEnableRouting: typeBoolValue(true),
		SubClashRules:         typeStringValue("rule1,rule2"),
		SubJsonFinalMask:      typeStringValue(`{"tcp":"mask1"}`),
	}
	result := expandPanelSubscription(m)
	if result["subClashEnableRouting"] != true {
		t.Fatalf("expected subClashEnableRouting true, got %v", result["subClashEnableRouting"])
	}
	if result["subClashRules"] != "rule1,rule2" {
		t.Fatalf("unexpected subClashRules: %v", result["subClashRules"])
	}
	if result["subJsonFinalMask"] != `{"tcp":"mask1"}` {
		t.Fatalf("unexpected subJsonFinalMask: %v", result["subJsonFinalMask"])
	}
}

// ---------------------------------------------------------------------------
// v3.4.0 coverage: PanelEmail expand/flatten/schema + new TG/subscription
// fields. These follow the same round-trip convention as the Telegram/
// subscription tests above; they also call panelEmailSchema() so the schema
// literal counts towards unit coverage (see TestPanelGeneralSchemaAttributeShape).
// ---------------------------------------------------------------------------

// TestPanelEmailSchemaAttributeShape asserts the v3.4.0 SMTP attributes exist
// with the expected Optional+Computed shape, and that the password pair is
// Sensitive / WriteOnly as documented in AGENTS.md. Exercises panelEmailSchema()
// so the schema literal is covered.
func TestPanelEmailSchemaAttributeShape(t *testing.T) {
	s := panelEmailSchema()

	requireStringOptComp := func(name string) {
		t.Helper()
		attr, ok := s.Attributes[name]
		if !ok {
			t.Fatalf("attribute %q missing on panel_email schema", name)
		}
		str, ok := attr.(schema.StringAttribute)
		if !ok {
			t.Fatalf("attribute %q is not a StringAttribute", name)
		}
		if !str.Optional || !str.Computed {
			t.Errorf("attribute %q must be Optional+Computed", name)
		}
	}

	requireIntOptComp := func(name string) {
		t.Helper()
		attr, ok := s.Attributes[name]
		if !ok {
			t.Fatalf("attribute %q missing on panel_email schema", name)
		}
		if _, ok := attr.(schema.Int64Attribute); !ok {
			t.Fatalf("attribute %q is not an Int64Attribute", name)
		}
	}

	// Core SMTP fields added in 3x-ui v3.4.0.
	requireStringOptComp("smtp_host")
	requireStringOptComp("smtp_username")
	requireStringOptComp("smtp_to")
	requireStringOptComp("smtp_encryption_type")
	requireStringOptComp("smtp_enabled_events")
	requireIntOptComp("smtp_port")
	requireIntOptComp("smtp_cpu")
	requireIntOptComp("smtp_memory")

	// smtp_enable is a bool Optional+Computed.
	if attr, ok := s.Attributes["smtp_enable"]; !ok {
		t.Fatalf("attribute %q missing", "smtp_enable")
	} else if _, ok := attr.(schema.BoolAttribute); !ok {
		t.Fatalf("smtp_enable is not a BoolAttribute")
	}

	// smtp_password is the legacy Sensitive attribute.
	pwdAttr, ok := s.Attributes["smtp_password"]
	if !ok {
		t.Fatalf("smtp_password missing")
	}
	pwd, ok := pwdAttr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("smtp_password is not a StringAttribute")
	}
	if !pwd.Sensitive {
		t.Errorf("smtp_password must be Sensitive")
	}

	// smtp_password_wo is the write-only alternative.
	woAttr, ok := s.Attributes["smtp_password_wo"]
	if !ok {
		t.Fatalf("smtp_password_wo missing")
	}
	wo, ok := woAttr.(schema.StringAttribute)
	if !ok {
		t.Fatalf("smtp_password_wo is not a StringAttribute")
	}
	if !wo.WriteOnly {
		t.Errorf("smtp_password_wo must be WriteOnly")
	}
}

func TestExpandPanelEmail(t *testing.T) {
	m := &PanelEmailModel{
		SmtpEnable:         typeBoolValue(true),
		SmtpHost:           typeStringValue("smtp.example.com"),
		SmtpPort:           typeInt64Value(587),
		SmtpUsername:       typeStringValue("user"),
		SmtpPassword:       typeStringValue("plain-secret"),
		SmtpTo:             typeStringValue("ops@example.com"),
		SmtpFrom:           typeStringValue("noreply@example.com"),
		SmtpFromName:       typeStringValue("3x-ui Bot"),
		SmtpEncryptionType: typeStringValue("starttls"),
		SmtpEnabledEvents:  typeStringValue("login,backup"),
		SmtpCPU:            typeInt64Value(90),
		SmtpMemory:         typeInt64Value(80),
	}
	got := expandPanelEmail(m)
	if got["smtpEnable"] != true {
		t.Fatalf("smtpEnable: %v", got["smtpEnable"])
	}
	if got["smtpHost"] != "smtp.example.com" {
		t.Fatalf("smtpHost: %v", got["smtpHost"])
	}
	if got["smtpPort"] != 587 {
		t.Fatalf("smtpPort: %v", got["smtpPort"])
	}
	if got["smtpUsername"] != "user" {
		t.Fatalf("smtpUsername: %v", got["smtpUsername"])
	}
	// No _wo set → plain password used.
	if got["smtpPassword"] != "plain-secret" {
		t.Fatalf("smtpPassword (plain fallback): %v", got["smtpPassword"])
	}
	if got["smtpTo"] != "ops@example.com" {
		t.Fatalf("smtpTo: %v", got["smtpTo"])
	}
	if got["smtpFrom"] != "noreply@example.com" {
		t.Fatalf("smtpFrom: %v", got["smtpFrom"])
	}
	if got["smtpFromName"] != "3x-ui Bot" {
		t.Fatalf("smtpFromName: %v", got["smtpFromName"])
	}
	if got["smtpEncryptionType"] != "starttls" {
		t.Fatalf("smtpEncryptionType: %v", got["smtpEncryptionType"])
	}
	if got["smtpEnabledEvents"] != "login,backup" {
		t.Fatalf("smtpEnabledEvents: %v", got["smtpEnabledEvents"])
	}
	if got["smtpCpu"] != 90 {
		t.Fatalf("smtpCpu: %v", got["smtpCpu"])
	}
	if got["smtpMemory"] != 80 {
		t.Fatalf("smtpMemory: %v", got["smtpMemory"])
	}
}

// TestExpandPanelEmail_WriteOnlyPrecedence asserts that when both the plain and
// write-only password are set, the write-only value wins (matches the
// expandPanelEmail branch order). This is the core of the write-only contract.
func TestExpandPanelEmail_WriteOnlyPrecedence(t *testing.T) {
	m := &PanelEmailModel{
		SmtpPassword:   typeStringValue("plain-secret"),
		SmtpPasswordWO: typeStringValue("wo-secret"),
	}
	got := expandPanelEmail(m)
	if got["smtpPassword"] != "wo-secret" {
		t.Fatalf("write-only password must take precedence, got %v", got["smtpPassword"])
	}
}

func TestFlattenPanelEmail(t *testing.T) {
	in := map[string]any{
		"smtpEnable":         true,
		"smtpHost":           "smtp.example.com",
		"smtpPort":           float64(587),
		"smtpUsername":       "user",
		"smtpPassword":       "returned-secret",
		"smtpTo":             "ops@example.com",
		"smtpFrom":           "noreply@example.com",
		"smtpFromName":       "3x-ui Bot",
		"smtpEncryptionType": "starttls",
		"smtpEnabledEvents":  "login,backup",
		"smtpCpu":            float64(90),
		"smtpMemory":         float64(80),
	}
	m := flattenPanelEmail(in)
	if m.ID.ValueString() != "settings" {
		t.Fatalf("id: %s", m.ID.ValueString())
	}
	if !m.SmtpEnable.ValueBool() {
		t.Fatalf("smtpEnable")
	}
	if m.SmtpHost.ValueString() != "smtp.example.com" {
		t.Fatalf("smtpHost: %s", m.SmtpHost.ValueString())
	}
	if m.SmtpPort.ValueInt64() != 587 {
		t.Fatalf("smtpPort: %d", m.SmtpPort.ValueInt64())
	}
	if m.SmtpUsername.ValueString() != "user" {
		t.Fatalf("smtpUsername: %s", m.SmtpUsername.ValueString())
	}
	if m.SmtpPassword.ValueString() != "returned-secret" {
		t.Fatalf("smtpPassword: %s", m.SmtpPassword.ValueString())
	}
	if m.SmtpTo.ValueString() != "ops@example.com" {
		t.Fatalf("smtpTo: %s", m.SmtpTo.ValueString())
	}
	if m.SmtpFrom.ValueString() != "noreply@example.com" {
		t.Fatalf("smtpFrom: %s", m.SmtpFrom.ValueString())
	}
	if m.SmtpFromName.ValueString() != "3x-ui Bot" {
		t.Fatalf("smtpFromName: %s", m.SmtpFromName.ValueString())
	}
	if m.SmtpEncryptionType.ValueString() != "starttls" {
		t.Fatalf("smtpEncryptionType: %s", m.SmtpEncryptionType.ValueString())
	}
	if m.SmtpEnabledEvents.ValueString() != "login,backup" {
		t.Fatalf("smtpEnabledEvents: %s", m.SmtpEnabledEvents.ValueString())
	}
	if m.SmtpCPU.ValueInt64() != 90 {
		t.Fatalf("smtpCpu: %d", m.SmtpCPU.ValueInt64())
	}
	if m.SmtpMemory.ValueInt64() != 80 {
		t.Fatalf("smtpMemory: %d", m.SmtpMemory.ValueInt64())
	}
}

// TestExpandPanelTelegram_v340Fields covers the v3.4.0 telegram fields added
// alongside the v3.4.0 SMTP work (tgEnabledEvents, tgMemory).
func TestExpandPanelTelegram_v340Fields(t *testing.T) {
	m := &PanelTelegramModel{
		TgEnabledEvents: typeStringValue("login,backup"),
		TgMemory:        typeInt64Value(85),
	}
	got := expandPanelTelegram(m)
	if got["tgEnabledEvents"] != "login,backup" {
		t.Fatalf("tgEnabledEvents: %v", got["tgEnabledEvents"])
	}
	if got["tgMemory"] != 85 {
		t.Fatalf("tgMemory: %v", got["tgMemory"])
	}
}

func TestFlattenPanelTelegram_v340Fields(t *testing.T) {
	in := map[string]any{
		"tgEnabledEvents": "login,backup",
		"tgMemory":        float64(85),
	}
	m := flattenPanelTelegram(in)
	if m.TgEnabledEvents.ValueString() != "login,backup" {
		t.Fatalf("tgEnabledEvents: %s", m.TgEnabledEvents.ValueString())
	}
	if m.TgMemory.ValueInt64() != 85 {
		t.Fatalf("tgMemory: %d", m.TgMemory.ValueInt64())
	}
}

// TestExpandPanelSubscription_v340Fields covers the v3.4.0 subscription fields
// (remarkTemplate, subHideSettings, subThemeDir).
func TestExpandPanelSubscription_v340Fields(t *testing.T) {
	m := &PanelSubscriptionModel{
		RemarkTemplate:  typeStringValue("#REMARK#-#EMAIL#"),
		SubHideSettings: typeBoolValue(true),
		SubThemeDir:     typeStringValue("/sub/theme/"),
	}
	got := expandPanelSubscription(m)
	if got["remarkTemplate"] != "#REMARK#-#EMAIL#" {
		t.Fatalf("remarkTemplate: %v", got["remarkTemplate"])
	}
	if got["subHideSettings"] != true {
		t.Fatalf("subHideSettings: %v", got["subHideSettings"])
	}
	if got["subThemeDir"] != "/sub/theme/" {
		t.Fatalf("subThemeDir: %v", got["subThemeDir"])
	}
}

func TestFlattenPanelSubscription_v340Fields(t *testing.T) {
	in := map[string]any{
		"remarkTemplate":  "#REMARK#-#EMAIL#",
		"subHideSettings": true,
		"subThemeDir":     "/sub/theme/",
	}
	m := flattenPanelSubscription(in)
	if m.RemarkTemplate.ValueString() != "#REMARK#-#EMAIL#" {
		t.Fatalf("remarkTemplate: %s", m.RemarkTemplate.ValueString())
	}
	if !m.SubHideSettings.ValueBool() {
		t.Fatalf("subHideSettings")
	}
	if m.SubThemeDir.ValueString() != "/sub/theme/" {
		t.Fatalf("subThemeDir: %s", m.SubThemeDir.ValueString())
	}
}

// TestExpandPanelSubscription_v341Fields covers the v3.4.1 subscription
// Incy routing-injection fields (subIncyEnableRouting, subIncyRoutingRules).
func TestExpandPanelSubscription_v341Fields(t *testing.T) {
	m := &PanelSubscriptionModel{
		SubIncyEnableRouting: typeBoolValue(true),
		SubIncyRoutingRules:  typeStringValue("vless://incy-rule"),
	}
	got := expandPanelSubscription(m)
	if got["subIncyEnableRouting"] != true {
		t.Fatalf("subIncyEnableRouting: %v", got["subIncyEnableRouting"])
	}
	if got["subIncyRoutingRules"] != "vless://incy-rule" {
		t.Fatalf("subIncyRoutingRules: %v", got["subIncyRoutingRules"])
	}
}

func TestFlattenPanelSubscription_v341Fields(t *testing.T) {
	in := map[string]any{
		"subIncyEnableRouting": true,
		"subIncyRoutingRules":  "vless://incy-rule",
	}
	m := flattenPanelSubscription(in)
	if !m.SubIncyEnableRouting.ValueBool() {
		t.Fatalf("subIncyEnableRouting")
	}
	if m.SubIncyRoutingRules.ValueString() != "vless://incy-rule" {
		t.Fatalf("subIncyRoutingRules: %s", m.SubIncyRoutingRules.ValueString())
	}
}

// TestExpandPanelSubscription_v360Fields covers the v3.6.0 subscription
// auto-detection fields.
func TestExpandPanelSubscription_v360Fields(t *testing.T) {
	m := &PanelSubscriptionModel{
		SubJsonAutoDetect:      typeBoolValue(true),
		SubJsonAlwaysArray:     typeBoolValue(true),
		SubJsonUserAgentRegex:  typeStringValue("v2ray.*"),
		SubClashAutoDetect:     typeBoolValue(true),
		SubClashUserAgentRegex: typeStringValue("clash.*"),
	}
	got := expandPanelSubscription(m)
	if got["subJsonAutoDetect"] != true {
		t.Fatalf("subJsonAutoDetect: %v", got["subJsonAutoDetect"])
	}
	if got["subJsonAlwaysArray"] != true {
		t.Fatalf("subJsonAlwaysArray: %v", got["subJsonAlwaysArray"])
	}
	if got["subJsonUserAgentRegex"] != "v2ray.*" {
		t.Fatalf("subJsonUserAgentRegex: %v", got["subJsonUserAgentRegex"])
	}
	if got["subClashAutoDetect"] != true {
		t.Fatalf("subClashAutoDetect: %v", got["subClashAutoDetect"])
	}
	if got["subClashUserAgentRegex"] != "clash.*" {
		t.Fatalf("subClashUserAgentRegex: %v", got["subClashUserAgentRegex"])
	}
}

func TestFlattenPanelSubscription_v360Fields(t *testing.T) {
	in := map[string]any{
		"subJsonAutoDetect":      true,
		"subJsonAlwaysArray":     true,
		"subJsonUserAgentRegex":  "v2ray.*",
		"subClashAutoDetect":     true,
		"subClashUserAgentRegex": "clash.*",
	}
	m := flattenPanelSubscription(in)
	if !m.SubJsonAutoDetect.ValueBool() {
		t.Fatalf("subJsonAutoDetect")
	}
	if !m.SubJsonAlwaysArray.ValueBool() {
		t.Fatalf("subJsonAlwaysArray")
	}
	if m.SubJsonUserAgentRegex.ValueString() != "v2ray.*" {
		t.Fatalf("subJsonUserAgentRegex: %s", m.SubJsonUserAgentRegex.ValueString())
	}
	if !m.SubClashAutoDetect.ValueBool() {
		t.Fatalf("subClashAutoDetect")
	}
	if m.SubClashUserAgentRegex.ValueString() != "clash.*" {
		t.Fatalf("subClashUserAgentRegex: %s", m.SubClashUserAgentRegex.ValueString())
	}
}

// TestExpandPanelGeneral_v360Fields covers the v3.6.0 general fields.
func TestExpandPanelGeneral_v360Fields(t *testing.T) {
	m := &PanelGeneralModel{
		SubShowIdentityOnAllLinks: typeBoolValue(true),
		OutboundDownThreshold:     typeInt64Value(5),
	}
	got := expandPanelGeneral(m)
	if got["subShowIdentityOnAllLinks"] != true {
		t.Fatalf("subShowIdentityOnAllLinks: %v", got["subShowIdentityOnAllLinks"])
	}
	if got["outboundDownThreshold"] != 5 {
		t.Fatalf("outboundDownThreshold: %v", got["outboundDownThreshold"])
	}
}

func TestFlattenPanelGeneral_v360Fields(t *testing.T) {
	in := map[string]any{
		"subShowIdentityOnAllLinks": true,
		"outboundDownThreshold":     float64(5),
	}
	m := flattenPanelGeneral(in)
	if !m.SubShowIdentityOnAllLinks.ValueBool() {
		t.Fatalf("subShowIdentityOnAllLinks")
	}
	if m.OutboundDownThreshold.ValueInt64() != 5 {
		t.Fatalf("outboundDownThreshold: %d", m.OutboundDownThreshold.ValueInt64())
	}
}

// TestPanelSettings_v370Fields covers the two AllSetting fields added in 3x-ui
// v3.7.0: the IP-limit allowlist (panel_general) and the JSON-subscription
// observatory blob for client-side balancers (panel_subscription). The empty
// case mirrors a pre-v3.7.0 panel, which reports "" for both.
func TestPanelSettings_v370Fields(t *testing.T) {
	general := expandPanelGeneral(&PanelGeneralModel{
		IPLimitAllowlist: typeStringValue("10.0.0.0/8,192.0.2.7"),
	})
	if general["ipLimitAllowlist"] != "10.0.0.0/8,192.0.2.7" {
		t.Fatalf("ipLimitAllowlist: %v", general["ipLimitAllowlist"])
	}
	gm := flattenPanelGeneral(map[string]any{"ipLimitAllowlist": "10.0.0.0/8,192.0.2.7"})
	if gm.IPLimitAllowlist.ValueString() != "10.0.0.0/8,192.0.2.7" {
		t.Fatalf("ipLimitAllowlist: %s", gm.IPLimitAllowlist.ValueString())
	}
	if empty := flattenPanelGeneral(map[string]any{"ipLimitAllowlist": ""}); empty.IPLimitAllowlist.ValueString() != "" {
		t.Fatalf("ipLimitAllowlist on a pre-v3.7.0 panel: %s", empty.IPLimitAllowlist.ValueString())
	}

	observatory := `{"subjectSelector":["out"],"probeInterval":"5m"}`
	sub := expandPanelSubscription(&PanelSubscriptionModel{
		SubJsonObservatory: typeStringValue(observatory),
	})
	if sub["subJsonObservatory"] != observatory {
		t.Fatalf("subJsonObservatory: %v", sub["subJsonObservatory"])
	}
	sm := flattenPanelSubscription(map[string]any{"subJsonObservatory": observatory})
	if sm.SubJsonObservatory.ValueString() != observatory {
		t.Fatalf("subJsonObservatory: %s", sm.SubJsonObservatory.ValueString())
	}
	if empty := flattenPanelSubscription(map[string]any{"subJsonObservatory": ""}); empty.SubJsonObservatory.ValueString() != "" {
		t.Fatalf("subJsonObservatory on a pre-v3.7.0 panel: %s", empty.SubJsonObservatory.ValueString())
	}
}

// TestPreservePanelEmailSecrets verifies the SMTP password replay path: when
// the panel returns a redacted/masked sentinel for smtpPassword, the provider
// replays the user-configured secret so state stays consistent (defensive —
// 3x-ui v3.0.2–v3.3.1 actually return secrets raw, so this mainly fires when
// the panel genuinely has no secret stored). Also covers the write-only
// version trigger carry-through.
func TestPreservePanelEmailSecrets(t *testing.T) {
	state := &PanelEmailModel{
		// Panel returned a masked sentinel for the password.
		SmtpPassword:          typeStringValue("********"),
		SmtpPasswordWOVersion: typeInt64Value(2),
	}
	configured := &PanelEmailModel{
		// User still configures the real secret.
		SmtpPassword:          typeStringValue("real-secret"),
		SmtpPasswordWOVersion: typeInt64Value(0),
	}
	preservePanelEmailSecrets(state, configured)
	// Redacted observed + non-empty configured → configured wins (replay).
	if state.SmtpPassword.ValueString() != "real-secret" {
		t.Fatalf("expected configured secret to win over redacted observed, got %q", state.SmtpPassword.ValueString())
	}
	// preserveWOVersion echoes the observed version when it is set.
	if state.SmtpPasswordWOVersion.ValueInt64() != 2 {
		t.Fatalf("expected wo version preserved, got %d", state.SmtpPasswordWOVersion.ValueInt64())
	}
}

// TestPreservePanelEmailSecrets_NilGuards asserts the nil-safety guard.
func TestPreservePanelEmailSecrets_NilGuards(t *testing.T) {
	// Neither nil panics; both are no-ops.
	preservePanelEmailSecrets(nil, &PanelEmailModel{})
	preservePanelEmailSecrets(&PanelEmailModel{}, nil)
}

// TestPreservePanelSubscriptionRemoved verifies that the v3.4.0-removed
// subscription fields (subShowInfo, subEmailInRemark) are echoed from state so
// Apply stays consistent on v3.4.0 panels (which accept but don't return them).
func TestPreservePanelSubscriptionRemoved(t *testing.T) {
	state := &PanelSubscriptionModel{
		SubShowInfo:      typeBoolValue(true),
		SubEmailInRemark: typeBoolValue(false),
	}
	configured := &PanelSubscriptionModel{
		SubShowInfo:      types.BoolNull(),
		SubEmailInRemark: types.BoolNull(),
	}
	preservePanelSubscriptionRemoved(state, configured)
	if !state.SubShowInfo.ValueBool() {
		t.Fatalf("expected sub_show_info preserved from state")
	}
	if state.SubEmailInRemark.ValueBool() {
		t.Fatalf("expected sub_email_in_remark preserved from state")
	}
}

// Helper functions for creating typed values in tests
func typeBoolValue(v bool) types.Bool       { return types.BoolValue(v) }
func typeStringValue(v string) types.String { return types.StringValue(v) }
func typeInt64Value(v int64) types.Int64    { return types.Int64Value(v) }

// TestAddrOrPrefixListValidator mirrors upstream CheckNetipAddrOrPrefixList,
// which backs ip_limit_allowlist. netip (unlike net) rejects zero-padded
// prefixes such as "10.0.0.0/024", and the panel refuses the save when it does —
// so the validator has to reject exactly what the panel rejects.
func TestAddrOrPrefixListValidator(t *testing.T) {
	if got := (addrOrPrefixListValidator{}).Description(t.Context()); got == "" {
		t.Error("Description must not be empty — it is what Terraform shows on a validation failure")
	}
	if got := (addrOrPrefixListValidator{}).MarkdownDescription(t.Context()); got == "" {
		t.Error("MarkdownDescription must not be empty")
	}

	valid := []string{
		"",
		"10.0.0.1",
		"10.0.0.0/8",
		"10.0.0.0/8, 192.0.2.7",
		"  10.0.0.0/8 ,, 2001:db8::1  ",
		"2001:db8::/32",
	}
	for _, in := range valid {
		resp := &validator.StringResponse{}
		addrOrPrefixListValidator{}.ValidateString(t.Context(), validator.StringRequest{
			Path:        path.Root("ip_limit_allowlist"),
			ConfigValue: typeStringValue(in),
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("%q should be valid: %v", in, resp.Diagnostics.Errors())
		}
	}

	invalid := []string{
		"10.0.0.0/024",
		"192.0.2.999",
		"not-an-address",
		"10.0.0.0/33",
		"10.0.0.0/8, garbage",
	}
	for _, in := range invalid {
		resp := &validator.StringResponse{}
		addrOrPrefixListValidator{}.ValidateString(t.Context(), validator.StringRequest{
			Path:        path.Root("ip_limit_allowlist"),
			ConfigValue: typeStringValue(in),
		}, resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("%q should be rejected", in)
		}
	}

	// Null and unknown must be skipped, not rejected.
	for _, v := range []types.String{types.StringNull(), types.StringUnknown()} {
		resp := &validator.StringResponse{}
		addrOrPrefixListValidator{}.ValidateString(t.Context(), validator.StringRequest{
			Path:        path.Root("ip_limit_allowlist"),
			ConfigValue: v,
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("%v should be skipped: %v", v, resp.Diagnostics.Errors())
		}
	}
}

// --- panel_discord (3x-ui v3.8.0+) ---

// TestPanelDiscordSchemaAttributeShape asserts the discord secret attribute
// trio exists with the expected shape, and exercises panelDiscordSchema() so
// the schema literal counts towards unit coverage.
func TestPanelDiscordSchemaAttributeShape(t *testing.T) {
	s := panelDiscordSchema()
	token, ok := s.Attributes["discord_bot_token"].(schema.StringAttribute)
	if !ok {
		t.Fatal("discord_bot_token missing or not a StringAttribute")
	}
	if !token.Optional || !token.Computed || !token.Sensitive {
		t.Error("discord_bot_token must be Optional+Computed+Sensitive")
	}
	wo, ok := s.Attributes["discord_bot_token_wo"].(schema.StringAttribute)
	if !ok {
		t.Fatal("discord_bot_token_wo missing or not a StringAttribute")
	}
	if !wo.WriteOnly {
		t.Error("discord_bot_token_wo must be WriteOnly")
	}
	for _, name := range []string{"discord_cpu", "discord_memory"} {
		attr, ok := s.Attributes[name].(schema.Int64Attribute)
		if !ok {
			t.Fatalf("%s missing or not an Int64Attribute", name)
		}
		if len(attr.Validators) == 0 {
			t.Errorf("%s must carry the panel's 0-100 validator", name)
		}
	}
	for _, name := range []string{
		"discord_bot_enable", "discord_channel_id", "discord_admin_ids",
		"discord_run_time", "discord_bot_backup", "discord_lang",
		"discord_enabled_events",
	} {
		if _, ok := s.Attributes[name]; !ok {
			t.Errorf("attribute %q missing on panel_discord schema", name)
		}
	}
}

// TestPanelDiscordExpandFlatten round-trips every managed discord key through
// expand/flatten, including the write-only token precedence.
func TestPanelDiscordExpandFlatten(t *testing.T) {
	m := &PanelDiscordModel{
		DiscordBotEnable:     types.BoolValue(true),
		DiscordBotToken:      types.StringValue("plain-token"),
		DiscordChannelID:     types.StringValue("1234567890"),
		DiscordAdminIDs:      types.StringValue("111,222"),
		DiscordRunTime:       types.StringValue("@daily"),
		DiscordBotBackup:     types.BoolValue(true),
		DiscordCPU:           types.Int64Value(80),
		DiscordMemory:        types.Int64Value(90),
		DiscordLang:          types.StringValue("en-US"),
		DiscordEnabledEvents: types.StringValue("login,backup,cpu.high"),
	}
	out := expandPanelDiscord(m)
	want := map[string]any{
		"discordBotEnable":     true,
		"discordBotToken":      "plain-token",
		"discordChannelId":     "1234567890",
		"discordAdminIds":      "111,222",
		"discordRunTime":       "@daily",
		"discordBotBackup":     true,
		"discordCpu":           80,
		"discordMemory":        90,
		"discordLang":          "en-US",
		"discordEnabledEvents": "login,backup,cpu.high",
	}
	if len(out) != len(want) {
		t.Fatalf("expand produced %d keys, want %d: %v", len(out), len(want), out)
	}
	for k, v := range want {
		got, ok := out[k]
		if !ok {
			t.Fatalf("expand missing key %q", k)
		}
		if got != v {
			t.Errorf("expand[%q] = %v (%T), want %v (%T)", k, got, got, v, v)
		}
	}

	back := flattenPanelDiscord(out)
	if back.ID.ValueString() != "settings" {
		t.Errorf("flatten id = %q, want settings", back.ID.ValueString())
	}
	if !back.DiscordBotEnable.ValueBool() {
		t.Error("flatten discord_bot_enable")
	}
	if back.DiscordBotToken.ValueString() != "plain-token" {
		t.Error("flatten discord_bot_token")
	}
	if back.DiscordCPU.ValueInt64() != 80 || back.DiscordMemory.ValueInt64() != 90 {
		t.Error("flatten thresholds")
	}
	if back.DiscordEnabledEvents.ValueString() != want["discordEnabledEvents"] {
		t.Error("flatten discord_enabled_events")
	}

	// The write-only token wins over the plain attribute when both are set
	// (mirrors expandPanelTelegram).
	m.DiscordBotTokenWO = types.StringValue("wo-token")
	if got := expandPanelDiscord(m)["discordBotToken"]; got != "wo-token" {
		t.Errorf("expand with _wo set = %v, want wo-token", got)
	}

	// A redacted/blank token from /setting/all on v3.8.x must survive
	// round-trip without inventing state.
	blank := flattenPanelDiscord(map[string]any{"discordBotToken": ""})
	if blank.DiscordBotToken.ValueString() != "" {
		t.Errorf("flatten must surface the blank token as an empty string, got %q", blank.DiscordBotToken.ValueString())
	}
}

// TestResolveDiscordTokenWO mirrors the telegram write-only resolution:
// create copies the _wo value unconditionally, update only when the version
// trigger moved.
func TestResolveDiscordTokenWO(t *testing.T) {
	plan := &PanelDiscordModel{DiscordBotToken: types.StringValue("old")}
	config := PanelDiscordModel{DiscordBotTokenWO: types.StringValue("wo")}
	resolveDiscordTokenWO(plan, config)
	if plan.DiscordBotToken.ValueString() != "wo" {
		t.Errorf("create: token = %q, want wo", plan.DiscordBotToken.ValueString())
	}

	// Update with unchanged version: keep the planned value.
	plan = &PanelDiscordModel{
		DiscordBotToken:      types.StringValue("planned"),
		DiscordBotTokenWOVer: types.Int64Value(3),
	}
	state := PanelDiscordModel{DiscordBotTokenWOVer: types.Int64Value(3)}
	resolveDiscordTokenWOUpdate(plan, state, config)
	if plan.DiscordBotToken.ValueString() != "planned" {
		t.Errorf("update without version bump: token = %q, want planned", plan.DiscordBotToken.ValueString())
	}

	// Update with a version bump: copy the _wo value.
	state.DiscordBotTokenWOVer = types.Int64Value(2)
	resolveDiscordTokenWOUpdate(plan, state, config)
	if plan.DiscordBotToken.ValueString() != "wo" {
		t.Errorf("update with version bump: token = %q, want wo", plan.DiscordBotToken.ValueString())
	}

	// First use on existing state without a version: treated as a change.
	plan = &PanelDiscordModel{
		DiscordBotToken:      types.StringValue("planned"),
		DiscordBotTokenWOVer: types.Int64Value(1),
	}
	resolveDiscordTokenWOUpdate(plan, PanelDiscordModel{}, config)
	if plan.DiscordBotToken.ValueString() != "wo" {
		t.Errorf("first _wo use: token = %q, want wo", plan.DiscordBotToken.ValueString())
	}
}

// TestPanelSettingsNeedRestart_DiscordKeys pins the discord restart surface:
// the notify cron and gateway hot-reload in-process (3x-ui v3.8.0+
// controller/setting.go:182-189 → web.go:693-719), so only the keys feeding
// startup-only alarm sampler registration restart — plus discordBotEnable,
// because cpuAlarmWanted()/memoryAlarmWanted() read it once at boot
// (web.go:478-482) and the reload func never re-checks alarm registration.
func TestPanelSettingsNeedRestart_DiscordKeys(t *testing.T) {
	// Hot-reloaded keys: schedule, credentials, per-delivery settings.
	for key, change := range map[string][2]any{
		"discordRunTime":   {"@daily", "@every 6h"},
		"discordBotToken":  {"old", "new"},
		"discordChannelId": {"1", "2"},
		"discordAdminIds":  {"111", "222"},
		"discordLang":      {"en-US", "fa-IR"},
		"discordBotBackup": {false, true},
	} {
		if panelSettingsNeedRestart(map[string]any{key: change[0]}, map[string]any{key: change[1]}) {
			t.Errorf("%s is hot-reloaded by reloadDiscordFunc and must not restart the panel", key)
		}
	}

	// Enable flip restarts (alarm sampler registration reads it at startup).
	if !panelSettingsNeedRestart(
		map[string]any{"discordBotEnable": false},
		map[string]any{"discordBotEnable": true},
	) {
		t.Error("discordBotEnable flip must restart")
	}

	// Thresholds only matter when they cross zero.
	if !panelSettingsNeedRestart(map[string]any{"discordCpu": float64(0)}, map[string]any{"discordCpu": 80}) {
		t.Error("discordCpu 0 → 80 must restart")
	}
	if panelSettingsNeedRestart(map[string]any{"discordCpu": float64(80)}, map[string]any{"discordCpu": 90}) {
		t.Error("discordCpu 80 → 90 must not restart")
	}

	// Event lists only matter for cpu.high / memory.high membership.
	if !panelSettingsNeedRestart(
		map[string]any{"discordEnabledEvents": "login,cpu.high"},
		map[string]any{"discordEnabledEvents": "login"},
	) {
		t.Error("dropping cpu.high from discordEnabledEvents must restart")
	}
	if panelSettingsNeedRestart(
		map[string]any{"discordEnabledEvents": "login,cpu.high"},
		map[string]any{"discordEnabledEvents": "login,backup,cpu.high"},
	) {
		t.Error("adding an unrelated discord event must not restart")
	}

	// On panels older than v3.8.0 the keys do not exist at all: never restart.
	old := map[string]any{"webPort": float64(2053)}
	desired := map[string]any{
		"discordBotEnable": true, "discordRunTime": "@daily",
		"discordCpu": 80, "discordEnabledEvents": "cpu.high",
	}
	if panelSettingsNeedRestart(old, desired) {
		t.Error("keys unknown to the panel must never restart")
	}
}

// --- v3.8.0 subscription additions (Happ / subProfileMode / page templates) ---

// TestPanelSubscriptionV38ExpandFlatten round-trips a representative slice of
// the 31 v3.8.0 subscription fields through expand/flatten.
func TestPanelSubscriptionV38ExpandFlatten(t *testing.T) {
	m := &PanelSubscriptionModel{
		SubProfileMode:             types.StringValue("custom"),
		SubInfoNodeEnable:          types.BoolValue(true),
		SubCalendarExpireInclusive: types.BoolValue(false),
		SubExpiredTemplate:         types.StringValue("expired {{email}}"),
		SubTrafficDepletedTemplate: types.StringValue("depleted {{email}}"),
		SubJsonRoutingRules:        types.StringValue(`[{"outboundTag":"direct"}]`),
		SubJsonDns:                 types.StringValue(`{"servers":["1.1.1.1"]}`),
		HappLinkEnable:             types.BoolValue(true),
		SubHappAutoDetect:          types.BoolValue(true),
		SubHappProviderId:          types.StringValue("prov-1"),
		SubHappSubInfoText:         types.StringValue("info"),
		SubHappTunMode:             types.StringValue("auto"),
		SubHappPerAppList:          types.StringValue("com.a,com.b"),
	}
	out := expandPanelSubscription(m)
	for key, want := range map[string]any{
		"subProfileMode":             "custom",
		"subInfoNodeEnable":          true,
		"subCalendarExpireInclusive": false,
		"subExpiredTemplate":         "expired {{email}}",
		"subTrafficDepletedTemplate": "depleted {{email}}",
		"subJsonRoutingRules":        `[{"outboundTag":"direct"}]`,
		"subJsonDns":                 `{"servers":["1.1.1.1"]}`,
		"happLinkEnable":             true,
		"subHappAutoDetect":          true,
		"subHappProviderId":          "prov-1",
		"subHappSubInfoText":         "info",
		"subHappTunMode":             "auto",
		"subHappPerAppList":          "com.a,com.b",
	} {
		if got := out[key]; got != want {
			t.Errorf("expand[%s] = %#v, want %#v", key, got, want)
		}
	}

	back := flattenPanelSubscription(out)
	if back.SubProfileMode.ValueString() != "custom" {
		t.Errorf("flatten sub_profile_mode: %#v", back.SubProfileMode)
	}
	if !back.SubInfoNodeEnable.ValueBool() || back.HappLinkEnable.ValueBool() != true {
		t.Errorf("flatten bools: %#v / %#v", back.SubInfoNodeEnable, back.HappLinkEnable)
	}
	if back.SubHappPerAppList.ValueString() != "com.a,com.b" {
		t.Errorf("flatten sub_happ_per_app_list: %#v", back.SubHappPerAppList)
	}
}

// TestPanelSettingsNeedRestart_SubServerV38Keys: the v3.8.0 keys frozen by
// (*sub.Server).initRouter() must restart; the per-request ones must not
// (3x-ui-3.8.5/internal/sub/sub.go:153-256 vs service.go:116/:251-257 and
// web/service/happ.go:89).
func TestPanelSettingsNeedRestart_SubServerV38Keys(t *testing.T) {
	for key, change := range map[string][2]any{
		"subProfileMode":      {"none", "builtin"},
		"subJsonRoutingRules": {`[{"a":1}]`, `[{"a":2}]`},
		"subJsonDns":          {`{"servers":[]}`, `{"servers":["1.1.1.1"]}`},
		"subHappProviderId":   {"old", "new"},
		"subHappTunMode":      {"auto", "on"},
		"subHappPerAppList":   {"com.a", "com.b"},
		"subHappAutoDetect":   {false, true},
		"subHappAlwaysHwid":   {false, true},
	} {
		if !panelSettingsNeedRestart(map[string]any{key: change[0]}, map[string]any{key: change[1]}) {
			t.Errorf("%s is frozen by initRouter and must restart the panel", key)
		}
	}
	for key, change := range map[string][2]any{
		"subCalendarExpireInclusive": {false, true},
		"subInfoNodeEnable":          {false, true},
		"subExpiredTemplate":         {"a", "b"},
		"subTrafficDepletedTemplate": {"a", "b"},
		"happLinkEnable":             {false, true},
	} {
		if panelSettingsNeedRestart(map[string]any{key: change[0]}, map[string]any{key: change[1]}) {
			t.Errorf("%s is read per request and must not restart the panel", key)
		}
	}
}

// TestPanelGeneralRealityScanCandidates round-trips the v3.8.0 REALITY scan
// candidate list through expand/flatten.
func TestPanelGeneralRealityScanCandidates(t *testing.T) {
	m := &PanelGeneralModel{RealityScanCandidates: types.StringValue("www.cloudflare.com:443,www.bing.com:443")}
	out := expandPanelGeneral(m)
	if out["realityScanCandidates"] != "www.cloudflare.com:443,www.bing.com:443" {
		t.Fatalf("expand: %#v", out["realityScanCandidates"])
	}
	back := flattenPanelGeneral(out)
	if back.RealityScanCandidates.ValueString() != "www.cloudflare.com:443,www.bing.com:443" {
		t.Fatalf("flatten: %#v", back.RealityScanCandidates)
	}
}
