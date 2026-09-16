package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// --- Panel Email: SMTP notifications (3x-ui v3.4.0+) ---

func TestAccPanelEmail(t *testing.T) {
	requireMinVersion(t, "v3.4.0")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_email" "test" {
  smtp_enable          = true
  smtp_host            = "smtp.example.com"
  smtp_port            = 587
  smtp_username        = "alerts@example.com"
  smtp_password        = "supersecret"
  smtp_to              = "admin@example.com"
  smtp_encryption_type = "starttls"
  smtp_enabled_events  = "login,backup"
  smtp_cpu             = 80
  smtp_memory          = 90
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_email.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_enable", "true"),
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_host", "smtp.example.com"),
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_port", "587"),
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_username", "alerts@example.com"),
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_encryption_type", "starttls"),
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_cpu", "80"),
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_memory", "90"),
				),
			},
			// Update
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_email" "test" {
  smtp_enable          = false
  smtp_host            = "mail.example.org"
  smtp_port            = 465
  smtp_username        = "noreply@example.org"
  smtp_password        = "supersecret"
  smtp_to              = "ops@example.org"
  smtp_encryption_type = "tls"
  smtp_enabled_events  = ""
  smtp_cpu             = 95
  smtp_memory          = 95
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_enable", "false"),
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_host", "mail.example.org"),
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_port", "465"),
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_encryption_type", "tls"),
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_cpu", "95"),
				),
			},
		},
	})
}

func TestAccPanelEmailWriteOnly(t *testing.T) {
	requireMinVersion(t, "v3.4.0")
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.11.0"))),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_email" "test" {
  smtp_enable               = true
  smtp_host                 = "smtp.example.com"
  smtp_port                 = 587
  smtp_username             = "alerts@example.com"
  smtp_password_wo          = "supersecret"
  smtp_password_wo_version  = 1
  smtp_to                   = "admin@example.com"
  smtp_encryption_type      = "starttls"
  smtp_cpu                  = 80
  smtp_memory               = 90
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_email.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_enable", "true"),
					resource.TestCheckResourceAttr("threexui_panel_email.test", "smtp_password_wo_version", "1"),
				),
			},
			// Idempotency: re-applying the same _wo config must produce an empty plan.
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_email" "test" {
  smtp_enable               = true
  smtp_host                 = "smtp.example.com"
  smtp_port                 = 587
  smtp_username             = "alerts@example.com"
  smtp_password_wo          = "supersecret"
  smtp_password_wo_version  = 1
  smtp_to                   = "admin@example.com"
  smtp_encryption_type      = "starttls"
  smtp_cpu                  = 80
  smtp_memory               = 90
}
`,
				PlanOnly: true,
			},
		},
	})
}

// --- Panel v3.4.0 attribute additions (telegram/general/subscription) ---

func TestAccPanelSettings_v340(t *testing.T) {
	requireMinVersion(t, "v3.4.0")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "g" {
  warp_update_interval = 24
}
resource "threexui_panel_telegram" "tg" {
  tg_bot_enable       = false
  tg_bot_chat_id      = ""
  tg_lang             = "en"
  tg_run_time         = "@daily"
  tg_bot_backup       = false
  tg_bot_login_notify = true
  tg_cpu              = 80
  tg_enabled_events   = "login,backup"
  tg_memory           = 90
}
resource "threexui_panel_subscription" "sub" {
  sub_enable        = false
  sub_listen        = ""
  sub_port          = 2096
  sub_theme_dir     = "/etc/3x-ui/sub"
  remark_template   = "{{email}} {{transport}}"
  sub_hide_settings = true
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_general.g", "warp_update_interval", "24"),
					resource.TestCheckResourceAttr("threexui_panel_telegram.tg", "tg_enabled_events", "login,backup"),
					resource.TestCheckResourceAttr("threexui_panel_telegram.tg", "tg_memory", "90"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.sub", "sub_theme_dir", "/etc/3x-ui/sub"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.sub", "remark_template", "{{email}} {{transport}}"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.sub", "sub_hide_settings", "true"),
				),
			},
		},
	})
}

// TestAccPanelSettings_v370 covers the two AllSetting fields added in 3x-ui
// v3.7.0 end-to-end: the per-client IP-limit allowlist on panel_general and the
// JSON-subscription observatory blob on panel_subscription.
func TestAccPanelSettings_v370(t *testing.T) {
	requireMinVersion(t, "v3.7.0")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "g" {
  ip_limit_allowlist = "10.0.0.0/8,192.0.2.7"
}
resource "threexui_panel_subscription" "sub" {
  sub_enable           = false
  sub_listen           = ""
  sub_port             = 2096
  sub_json_observatory = "{\"subjectSelector\":[\"out\"],\"probeInterval\":\"5m\"}"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_general.g", "ip_limit_allowlist", "10.0.0.0/8,192.0.2.7"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.sub", "sub_json_observatory",
						`{"subjectSelector":["out"],"probeInterval":"5m"}`),
				),
			},
			{
				// Clearing the allowlist must stick, not fall back to the old value.
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "g" {
  ip_limit_allowlist = ""
}
`,
				Check: resource.TestCheckResourceAttr("threexui_panel_general.g", "ip_limit_allowlist", ""),
			},
		},
	})
}

// --- Panel General: page_size, remark_model, time_location, update, idempotency ---

func TestAccPanelGeneral(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "test" {
  date_picker                    = "gregorian"
  expire_diff                    = 1
  external_traffic_inform_enable = false
  external_traffic_inform_uri    = ""
  ldap_auto_create               = false
  ldap_auto_delete               = false
  ldap_base_dn                   = ""
  ldap_bind_dn                   = ""
  ldap_default_expiry_days       = 0
  ldap_default_limit_ip          = 0
  ldap_default_total_gb          = 0
  ldap_enable                    = false
  ldap_flag_field                = ""
  ldap_host                      = ""
  ldap_inbound_tags              = ""
  ldap_invert_flag               = false
  ldap_password                  = ""
  ldap_port                      = 389
  ldap_sync_cron                 = "@every 1m"
  ldap_truthy_values             = "true,1,yes,on"
  ldap_use_tls                   = false
  ldap_user_attr                 = "mail"
  ldap_user_filter               = "(objectClass=person)"
  ldap_vless_field               = "vless_enabled"
  page_size                      = 50
  remark_model                   = "-ieo"
  session_max_age                = 360
  time_location                  = "Asia/Tehran"
  traffic_diff                   = 1
  web_base_path                  = "/"
  web_cert_file                  = ""
  web_domain                     = ""
  web_key_file                   = ""
  web_listen                     = ""
  web_port                       = 2053
  xray_outbound_test_url         = "https://example.com/generate_204"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_general.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_general.test", "page_size", "50"),
					resource.TestCheckResourceAttr("threexui_panel_general.test", "time_location", "Asia/Tehran"),
					resource.TestCheckResourceAttr("threexui_panel_general.test", "xray_outbound_test_url", "https://example.com/generate_204"),
				),
			},
			// Update
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "test" {
  date_picker                    = "gregorian"
  expire_diff                    = 0
  external_traffic_inform_enable = false
  external_traffic_inform_uri    = ""
  ldap_auto_create               = false
  ldap_auto_delete               = false
  ldap_base_dn                   = ""
  ldap_bind_dn                   = ""
  ldap_default_expiry_days       = 0
  ldap_default_limit_ip          = 0
  ldap_default_total_gb          = 0
  ldap_enable                    = false
  ldap_flag_field                = ""
  ldap_host                      = ""
  ldap_inbound_tags              = ""
  ldap_invert_flag               = false
  ldap_password                  = ""
  ldap_port                      = 389
  ldap_sync_cron                 = "@every 1m"
  ldap_truthy_values             = "true,1,yes,on"
  ldap_use_tls                   = false
  ldap_user_attr                 = "mail"
  ldap_user_filter               = "(objectClass=person)"
  ldap_vless_field               = "vless_enabled"
  page_size                      = 25
  remark_model                   = "-ieo"
  session_max_age                = 360
  time_location                  = "Local"
  traffic_diff                   = 0
  web_base_path                  = "/"
  web_cert_file                  = ""
  web_domain                     = ""
  web_key_file                   = ""
  web_listen                     = ""
  web_port                       = 2053
  xray_outbound_test_url         = "https://www.google.com/generate_204"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_general.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_general.test", "page_size", "25"),
					resource.TestCheckResourceAttr("threexui_panel_general.test", "time_location", "Local"),
					resource.TestCheckResourceAttr("threexui_panel_general.test", "xray_outbound_test_url", "https://www.google.com/generate_204"),
				),
			},
			// ImportState
			{
				ResourceName:            "threexui_panel_general.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "settings",
				ImportStateVerifyIgnore: []string{"ldap_password", "remark_model", "panel_proxy"},
			},
			// Idempotency
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "test" {
  date_picker                    = "gregorian"
  expire_diff                    = 0
  external_traffic_inform_enable = false
  external_traffic_inform_uri    = ""
  ldap_auto_create               = false
  ldap_auto_delete               = false
  ldap_base_dn                   = ""
  ldap_bind_dn                   = ""
  ldap_default_expiry_days       = 0
  ldap_default_limit_ip          = 0
  ldap_default_total_gb          = 0
  ldap_enable                    = false
  ldap_flag_field                = ""
  ldap_host                      = ""
  ldap_inbound_tags              = ""
  ldap_invert_flag               = false
  ldap_password                  = ""
  ldap_port                      = 389
  ldap_sync_cron                 = "@every 1m"
  ldap_truthy_values             = "true,1,yes,on"
  ldap_use_tls                   = false
  ldap_user_attr                 = "mail"
  ldap_user_filter               = "(objectClass=person)"
  ldap_vless_field               = "vless_enabled"
  page_size                      = 25
  remark_model                   = "-ieo"
  session_max_age                = 360
  time_location                  = "Local"
  traffic_diff                   = 0
  web_base_path                  = "/"
  web_cert_file                  = ""
  web_domain                     = ""
  web_key_file                   = ""
  web_listen                     = ""
  web_port                       = 2053
  xray_outbound_test_url         = "https://www.google.com/generate_204"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// --- Panel General: LDAP fields ---

func TestAccPanelGeneralLDAP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "ldap" {
  date_picker                    = "gregorian"
  expire_diff                    = 0
  external_traffic_inform_enable = false
  external_traffic_inform_uri    = ""
  ldap_auto_create               = false
  ldap_auto_delete               = false
  ldap_base_dn                   = "dc=example,dc=com"
  ldap_bind_dn                   = "cn=admin,dc=example,dc=com"
  ldap_default_expiry_days       = 0
  ldap_default_limit_ip          = 0
  ldap_default_total_gb          = 0
  ldap_enable                    = true
  ldap_flag_field                = ""
  ldap_host                      = "ldap.example.com"
  ldap_inbound_tags              = ""
  ldap_invert_flag               = false
  ldap_password                  = "ldappass"
  ldap_port                      = 636
  ldap_sync_cron                 = "@every 1m"
  ldap_truthy_values             = "true,1,yes,on"
  ldap_use_tls                   = true
  ldap_user_attr                 = "mail"
  ldap_user_filter               = "(objectClass=person)"
  ldap_vless_field               = "vless_enabled"
  page_size                      = 25
  remark_model                   = "-ieo"
  session_max_age                = 360
  time_location                  = "Local"
  traffic_diff                   = 0
  web_base_path                  = "/"
  web_cert_file                  = ""
  web_domain                     = ""
  web_key_file                   = ""
  web_listen                     = ""
  web_port                       = 2053
  xray_outbound_test_url         = "https://www.google.com/generate_204"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_general.ldap", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_general.ldap", "ldap_enable", "true"),
					resource.TestCheckResourceAttr("threexui_panel_general.ldap", "ldap_host", "ldap.example.com"),
					resource.TestCheckResourceAttr("threexui_panel_general.ldap", "ldap_port", "636"),
				),
			},
			// Disable LDAP (restore defaults)
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "ldap" {
  date_picker                    = "gregorian"
  expire_diff                    = 0
  external_traffic_inform_enable = false
  external_traffic_inform_uri    = ""
  ldap_auto_create               = false
  ldap_auto_delete               = false
  ldap_base_dn                   = "dc=example,dc=com"
  ldap_bind_dn                   = "cn=admin,dc=example,dc=com"
  ldap_default_expiry_days       = 0
  ldap_default_limit_ip          = 0
  ldap_default_total_gb          = 0
  ldap_enable                    = false
  ldap_flag_field                = ""
  ldap_host                      = ""
  ldap_inbound_tags              = ""
  ldap_invert_flag               = false
  ldap_password                  = "ldappass"
  ldap_port                      = 389
  ldap_sync_cron                 = "@every 1m"
  ldap_truthy_values             = "true,1,yes,on"
  ldap_use_tls                   = true
  ldap_user_attr                 = "mail"
  ldap_user_filter               = "(objectClass=person)"
  ldap_vless_field               = "vless_enabled"
  page_size                      = 25
  remark_model                   = "-ieo"
  session_max_age                = 360
  time_location                  = "Local"
  traffic_diff                   = 0
  web_base_path                  = "/"
  web_cert_file                  = ""
  web_domain                     = ""
  web_key_file                   = ""
  web_listen                     = ""
  web_port                       = 2053
  xray_outbound_test_url         = "https://www.google.com/generate_204"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_general.ldap", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_general.ldap", "ldap_enable", "false"),
				),
			},
		},
	})
}

// --- Panel Security: two_factor_enable + two_factor_token, update, idempotency ---

func TestAccPanelSecurity(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_security" "test" {
  two_factor_enable = false
  two_factor_token  = ""
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_security.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_security.test", "two_factor_enable", "false"),
				),
			},
			// Update: set a token value (but keep 2FA disabled to not block provider)
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_security" "test" {
  two_factor_enable = false
  two_factor_token  = "test-token-value"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_security.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_security.test", "two_factor_token", "test-token-value"),
				),
			},
			// ImportState
			{
				ResourceName:      "threexui_panel_security.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "settings",
				// two_factor_token is a secret: on 3x-ui v3.4.0 the panel no longer
				// returns it raw, so it does not round-trip through import.
				ImportStateVerifyIgnore: []string{"two_factor_token"},
			},
			// Idempotency
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_security" "test" {
  two_factor_enable = false
  two_factor_token  = "test-token-value"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Restore defaults
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_security" "test" {
  two_factor_enable = false
  two_factor_token  = ""
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_security.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_security.test", "two_factor_enable", "false"),
				),
			},
		},
	})
}

// --- Telegram: enable + token/chat_id/run_time/lang, update ---

func TestAccPanelTelegram(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_telegram" "test" {
  tg_bot_backup       = false
  tg_bot_chat_id      = "987654321"
  tg_bot_enable       = true
  tg_bot_login_notify = true
  tg_bot_proxy        = ""
  tg_bot_api_server   = ""
  tg_bot_token        = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
  tg_cpu              = 80
  tg_lang             = "en"
  tg_run_time         = "@daily"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "tg_bot_enable", "true"),
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "tg_bot_chat_id", "987654321"),
				),
			},
			// Update
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_telegram" "test" {
  tg_bot_backup       = false
  tg_bot_chat_id      = ""
  tg_bot_enable       = false
  tg_bot_login_notify = true
  tg_bot_proxy        = ""
  tg_bot_api_server   = ""
  tg_bot_token        = ""
  tg_cpu              = 80
  tg_lang             = "ru"
  tg_run_time         = "@daily"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "tg_bot_enable", "false"),
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "tg_lang", "ru"),
				),
			},
			// ImportState
			{
				ResourceName:            "threexui_panel_telegram.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "settings",
				ImportStateVerifyIgnore: []string{"tg_bot_token", "tg_bot_login_notify"},
			},
			// Idempotency
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_telegram" "test" {
  tg_bot_backup       = false
  tg_bot_chat_id      = ""
  tg_bot_enable       = false
  tg_bot_login_notify = true
  tg_bot_proxy        = ""
  tg_bot_api_server   = ""
  tg_bot_token        = ""
  tg_cpu              = 80
  tg_lang             = "ru"
  tg_run_time         = "@daily"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// --- Subscription: enable + port/path/title, update, idempotency ---

// TestAccPanelGeneralConcurrentSettings verifies that panel_general and
// panel_subscription can be applied in the same graph without lost updates.
// Terraform applies independent resources concurrently, so both paths compete
// for the settings API. The settingsMu mutex must serialize these operations.
func TestAccPanelGeneralConcurrentSettings(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "test" {
  date_picker                    = "gregorian"
  expire_diff                    = 1
  external_traffic_inform_enable = false
  external_traffic_inform_uri    = ""
  ldap_auto_create               = false
  ldap_auto_delete               = false
  ldap_base_dn                   = ""
  ldap_bind_dn                   = ""
  ldap_default_expiry_days       = 0
  ldap_default_limit_ip          = 0
  ldap_default_total_gb          = 0
  ldap_enable                    = false
  ldap_flag_field                = ""
  ldap_host                      = ""
  ldap_inbound_tags              = ""
  ldap_invert_flag               = false
  ldap_password                  = ""
  ldap_port                      = 389
  ldap_sync_cron                 = "@every 1m"
  ldap_truthy_values             = "true,1,yes,on"
  ldap_use_tls                   = false
  ldap_user_attr                 = "mail"
  ldap_user_filter               = "(objectClass=person)"
  ldap_vless_field               = "vless_enabled"
  page_size                      = 50
  remark_model                   = "-ieo"
  session_max_age                = 360
  time_location                  = "Asia/Tehran"
  traffic_diff                   = 1
  web_base_path                  = "/"
  web_cert_file                  = ""
  web_domain                     = ""
  web_key_file                   = ""
  web_listen                     = ""
  web_port                       = 2053
  xray_outbound_test_url         = "https://example.com/generate_204"
}

resource "threexui_panel_subscription" "test" {
  sub_announce       = ""
  sub_cert_file      = ""
  sub_domain         = ""
  sub_enable         = true
  sub_enable_routing = true
  sub_encrypt        = true
  sub_json_enable    = true
  sub_json_fragment  = ""
  sub_json_mux       = ""
  sub_json_noises    = ""
  sub_json_path      = "/json/"
  sub_json_rules     = ""
  sub_json_uri       = ""
  sub_key_file       = ""
  sub_listen         = ""
  sub_path           = "/sub/"
  sub_port           = 2096
  sub_profile_url    = ""
  sub_routing_rules  = ""
  sub_show_info      = true
  sub_support_url    = ""
  sub_title          = "concurrent-test"
  sub_updates        = 12
  sub_uri            = ""
}
`,
				Check: resource.ComposeTestCheckFunc(
					// panel_general values preserved
					resource.TestCheckResourceAttr("threexui_panel_general.test", "page_size", "50"),
					resource.TestCheckResourceAttr("threexui_panel_general.test", "time_location", "Asia/Tehran"),
					resource.TestCheckResourceAttr("threexui_panel_general.test", "xray_outbound_test_url", "https://example.com/generate_204"),
					// panel_subscription values preserved
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_enable", "true"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_title", "concurrent-test"),
				),
			},
			// Idempotency — both resources stable after concurrent apply
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "test" {
  date_picker                    = "gregorian"
  expire_diff                    = 1
  external_traffic_inform_enable = false
  external_traffic_inform_uri    = ""
  ldap_auto_create               = false
  ldap_auto_delete               = false
  ldap_base_dn                   = ""
  ldap_bind_dn                   = ""
  ldap_default_expiry_days       = 0
  ldap_default_limit_ip          = 0
  ldap_default_total_gb          = 0
  ldap_enable                    = false
  ldap_flag_field                = ""
  ldap_host                      = ""
  ldap_inbound_tags              = ""
  ldap_invert_flag               = false
  ldap_password                  = ""
  ldap_port                      = 389
  ldap_sync_cron                 = "@every 1m"
  ldap_truthy_values             = "true,1,yes,on"
  ldap_use_tls                   = false
  ldap_user_attr                 = "mail"
  ldap_user_filter               = "(objectClass=person)"
  ldap_vless_field               = "vless_enabled"
  page_size                      = 50
  remark_model                   = "-ieo"
  session_max_age                = 360
  time_location                  = "Asia/Tehran"
  traffic_diff                   = 1
  web_base_path                  = "/"
  web_cert_file                  = ""
  web_domain                     = ""
  web_key_file                   = ""
  web_listen                     = ""
  web_port                       = 2053
  xray_outbound_test_url         = "https://example.com/generate_204"
}

resource "threexui_panel_subscription" "test" {
  sub_announce       = ""
  sub_cert_file      = ""
  sub_domain         = ""
  sub_enable         = true
  sub_enable_routing = true
  sub_encrypt        = true
  sub_json_enable    = true
  sub_json_fragment  = ""
  sub_json_mux       = ""
  sub_json_noises    = ""
  sub_json_path      = "/json/"
  sub_json_rules     = ""
  sub_json_uri       = ""
  sub_key_file       = ""
  sub_listen         = ""
  sub_path           = "/sub/"
  sub_port           = 2096
  sub_profile_url    = ""
  sub_routing_rules  = ""
  sub_show_info      = true
  sub_support_url    = ""
  sub_title          = "concurrent-test"
  sub_updates        = 12
  sub_uri            = ""
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccPanelGeneralConcurrentXray verifies that panel_general
// (xray_outbound_test_url) and xray_outbounds can be applied in the
// same graph without lost updates. Both compete for the xray template
// endpoint; xrayTemplateMu must serialize them.
func TestAccPanelGeneralConcurrentXray(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "test" {
  date_picker                    = "gregorian"
  expire_diff                    = 0
  external_traffic_inform_enable = false
  external_traffic_inform_uri    = ""
  ldap_auto_create               = false
  ldap_auto_delete               = false
  ldap_base_dn                   = ""
  ldap_bind_dn                   = ""
  ldap_default_expiry_days       = 0
  ldap_default_limit_ip          = 0
  ldap_default_total_gb          = 0
  ldap_enable                    = false
  ldap_flag_field                = ""
  ldap_host                      = ""
  ldap_inbound_tags              = ""
  ldap_invert_flag               = false
  ldap_password                  = ""
  ldap_port                      = 389
  ldap_sync_cron                 = "@every 1m"
  ldap_truthy_values             = "true,1,yes,on"
  ldap_use_tls                   = false
  ldap_user_attr                 = "mail"
  ldap_user_filter               = "(objectClass=person)"
  ldap_vless_field               = "vless_enabled"
  page_size                      = 50
  remark_model                   = "-ieo"
  session_max_age                = 360
  time_location                  = "Asia/Tehran"
  traffic_diff                   = 0
  web_base_path                  = "/"
  web_cert_file                  = ""
  web_domain                     = ""
  web_key_file                   = ""
  web_listen                     = ""
  web_port                       = 2053
  xray_outbound_test_url         = "https://example.com/generate_204"
}

resource "threexui_xray_outbounds" "test" {
  outbound {
    tag      = "direct"
    protocol = "freedom"

    freedom_settings {
      domain_strategy = "AsIs"
    }
  }

  outbound {
    tag      = "blocked"
    protocol = "blackhole"

    blackhole_settings {
      response_type = "none"
    }
  }

  outbound {
    tag      = "dns-out"
    protocol = "dns"
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_general.test", "xray_outbound_test_url", "https://example.com/generate_204"),
					resource.TestCheckResourceAttr("threexui_xray_outbounds.test", "outbound.0.tag", "direct"),
					resource.TestCheckResourceAttr("threexui_xray_outbounds.test", "outbound.1.tag", "blocked"),
				),
			},
			// Idempotency
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "test" {
  date_picker                    = "gregorian"
  expire_diff                    = 0
  external_traffic_inform_enable = false
  external_traffic_inform_uri    = ""
  ldap_auto_create               = false
  ldap_auto_delete               = false
  ldap_base_dn                   = ""
  ldap_bind_dn                   = ""
  ldap_default_expiry_days       = 0
  ldap_default_limit_ip          = 0
  ldap_default_total_gb          = 0
  ldap_enable                    = false
  ldap_flag_field                = ""
  ldap_host                      = ""
  ldap_inbound_tags              = ""
  ldap_invert_flag               = false
  ldap_password                  = ""
  ldap_port                      = 389
  ldap_sync_cron                 = "@every 1m"
  ldap_truthy_values             = "true,1,yes,on"
  ldap_use_tls                   = false
  ldap_user_attr                 = "mail"
  ldap_user_filter               = "(objectClass=person)"
  ldap_vless_field               = "vless_enabled"
  page_size                      = 50
  remark_model                   = "-ieo"
  session_max_age                = 360
  time_location                  = "Asia/Tehran"
  traffic_diff                   = 0
  web_base_path                  = "/"
  web_cert_file                  = ""
  web_domain                     = ""
  web_key_file                   = ""
  web_listen                     = ""
  web_port                       = 2053
  xray_outbound_test_url         = "https://example.com/generate_204"
}

resource "threexui_xray_outbounds" "test" {
  outbound {
    tag      = "direct"
    protocol = "freedom"

    freedom_settings {
      domain_strategy = "AsIs"
    }
  }

  outbound {
    tag      = "blocked"
    protocol = "blackhole"

    blackhole_settings {
      response_type = "none"
    }
  }

  outbound {
    tag      = "dns-out"
    protocol = "dns"
  }
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccPanelSubscription(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_subscription" "test" {
  sub_announce       = ""
  sub_cert_file      = ""
  sub_domain         = ""
  sub_enable         = true
  sub_enable_routing = true
  sub_encrypt        = true
  sub_json_enable    = true
  sub_json_fragment  = ""
  sub_json_mux       = ""
  sub_json_noises    = ""
  sub_json_path      = "/json/"
  sub_json_rules     = ""
  sub_json_uri       = ""
  sub_key_file       = ""
  sub_listen         = ""
  sub_path           = "/sub/"
  sub_port           = 2096
  sub_profile_url    = ""
  sub_routing_rules  = ""
  sub_show_info      = true
  sub_support_url    = ""
  sub_title          = "acc-test-sub"
  sub_updates        = 12
  sub_uri             = ""
  sub_clash_enable_routing = null
  sub_clash_rules          = null
  sub_json_final_mask      = null
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_enable", "true"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_title", "acc-test-sub"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_port", "2096"),
				),
			},
			// Update
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_subscription" "test" {
  sub_announce       = ""
  sub_cert_file      = ""
  sub_domain         = ""
  sub_enable         = true
  sub_enable_routing = true
  sub_encrypt        = true
  sub_json_enable    = false
  sub_json_fragment  = ""
  sub_json_mux       = ""
  sub_json_noises    = ""
  sub_json_path      = "/json/"
  sub_json_rules     = ""
  sub_json_uri       = ""
  sub_key_file       = ""
  sub_listen         = ""
  sub_path           = "/newsub/"
  sub_port           = 2097
  sub_profile_url    = ""
  sub_routing_rules  = ""
  sub_show_info      = true
  sub_support_url    = ""
  sub_title          = "acc-test-sub-updated"
  sub_updates        = 12
  sub_uri             = ""
  sub_clash_enable_routing = null
  sub_clash_rules          = null
  sub_json_final_mask      = null
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_title", "acc-test-sub-updated"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_port", "2097"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_path", "/newsub/"),
				),
			},
			// ImportState
			{
				ResourceName:      "threexui_panel_subscription.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "settings",
				// sub_show_info / sub_email_in_remark were removed from AllSetting in
				// 3x-ui v3.4.0 and do not round-trip through import there.
				ImportStateVerifyIgnore: []string{"sub_show_info", "sub_email_in_remark"},
			},
			// Idempotency
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_subscription" "test" {
  sub_announce       = ""
  sub_cert_file      = ""
  sub_domain         = ""
  sub_enable         = true
  sub_enable_routing = true
  sub_encrypt        = true
  sub_json_enable    = false
  sub_json_fragment  = ""
  sub_json_mux       = ""
  sub_json_noises    = ""
  sub_json_path      = "/json/"
  sub_json_rules     = ""
  sub_json_uri       = ""
  sub_key_file       = ""
  sub_listen         = ""
  sub_path           = "/newsub/"
  sub_port           = 2097
  sub_profile_url    = ""
  sub_routing_rules  = ""
  sub_show_info      = true
  sub_support_url    = ""
  sub_title          = "acc-test-sub-updated"
  sub_updates        = 12
  sub_uri             = ""
  sub_clash_enable_routing = null
  sub_clash_rules          = null
  sub_json_final_mask      = null
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Disable subscription (restore)
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_subscription" "test" {
  sub_announce       = ""
  sub_cert_file      = ""
  sub_domain         = ""
  sub_enable         = false
  sub_enable_routing = true
  sub_encrypt        = true
  sub_json_enable    = false
  sub_json_fragment  = ""
  sub_json_mux       = ""
  sub_json_noises    = ""
  sub_json_path      = "/json/"
  sub_json_rules     = ""
  sub_json_uri       = ""
  sub_key_file       = ""
  sub_listen         = ""
  sub_path           = "/sub/"
  sub_port           = 2096
  sub_profile_url    = ""
  sub_routing_rules  = ""
  sub_show_info      = true
  sub_support_url    = ""
  sub_title          = "acc-test-sub-updated"
  sub_updates        = 12
  sub_uri             = ""
  sub_clash_enable_routing = null
  sub_clash_rules          = null
  sub_json_final_mask      = null
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_enable", "false"),
				),
			},
		},
	})
}

// TestAccPanelGeneralBasePathChange verifies that changing web_base_path and
// xray_outbound_test_url in the same operation succeeds. Before the fix, the
// Xray update after restart would fail because client.basePath was stale.
//
// This test drives the client directly (not through terraform-plugin-testing)
// because the framework re-configures the provider between apply and
// post-apply plan, which would create a new client with the old base_path.
func TestAccPanelGeneralBasePathChange(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	skipOnFlakyVersions(t,
		"3x-ui v3.0.0 applies the first panel SIGHUP restart after webBasePath changes, then stops processing later panel restarts; this test needs a second restart to restore the shared acceptance container (#176)",
		"v3.0.0")

	client, err := testAccClientFromEnv()
	if err != nil {
		t.Fatalf("client init: %v", err)
	}
	ctx := context.Background()

	// Read current settings so we can restore them later.
	original, err := client.GetSettings(ctx)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	originalTestURL, err := client.GetXrayOutboundTestURL(ctx)
	if err != nil {
		t.Fatalf("get test url: %v", err)
	}

	// Ensure we restore the original base path regardless of outcome.
	restored := false
	t.Cleanup(func() {
		if restored {
			return
		}
		if err := restorePanelGeneralAfterBasePathChange(ctx, client, original, originalTestURL); err != nil {
			t.Logf("cleanup panel settings after base path change: %v", err)
		}
	})

	// --- Simulate what applyPanelGeneral does ---

	// 1. Update settings: change webBasePath.
	desired := map[string]any{"webBasePath": "/testbp/"}
	merged := mergeSettings(original, desired)
	if err := client.UpdateSettings(ctx, merged); err != nil {
		t.Fatalf("update settings: %v", err)
	}

	// 2. Send restart on the OLD path, then update basePath, then wait.
	if err := client.SendRestart(ctx); err != nil {
		t.Fatalf("send restart: %v", err)
	}
	client.SetBasePath("/testbp/")
	if err := client.WaitForReady(ctx); err != nil {
		t.Fatalf("wait for ready: %v", err)
	}

	// 3. Set xray outbound test URL on the NEW path.
	newTestURL := "https://example.com/generate_204"
	if err := client.SetXrayOutboundTestURL(ctx, newTestURL); err != nil {
		t.Fatalf("set xray outbound test url after base path change: %v", err)
	}

	// Verify the test URL was applied.
	got, err := client.GetXrayOutboundTestURL(ctx)
	if err != nil {
		t.Fatalf("get test url: %v", err)
	}
	if got != newTestURL {
		t.Fatalf("xray outbound test url = %q, want %q", got, newTestURL)
	}

	if err := restorePanelGeneralAfterBasePathChange(ctx, client, original, originalTestURL); err != nil {
		t.Fatalf("restore panel settings after base path change: %v", err)
	}
	restored = true
}

func restorePanelGeneralAfterBasePathChange(ctx context.Context, client *Client, original map[string]any, originalTestURL string) error {
	originalBasePath := normalizeBasePath(stringValue(original["webBasePath"]))
	restoreSettings := mergeSettings(original, map[string]any{"webBasePath": originalBasePath})
	candidates := []string{"/testbp/", originalBasePath, "/"}
	seen := make(map[string]bool, len(candidates))
	errs := make([]string, 0, len(candidates))

	for _, currentBasePath := range candidates {
		currentBasePath = normalizeBasePath(currentBasePath)
		if seen[currentBasePath] {
			continue
		}
		seen[currentBasePath] = true

		client.SetBasePath(currentBasePath)
		if err := client.Login(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("login on %s: %v", currentBasePath, err))
			continue
		}
		if err := client.UpdateSettings(ctx, restoreSettings); err != nil {
			errs = append(errs, fmt.Sprintf("update settings on %s: %v", currentBasePath, err))
			continue
		}
		restoredSettings, err := client.GetSettings(ctx)
		if err != nil {
			errs = append(errs, fmt.Sprintf("read settings on %s after restore update: %v", currentBasePath, err))
			continue
		}
		if got := normalizeBasePath(stringValue(restoredSettings["webBasePath"])); got != originalBasePath {
			errs = append(errs, fmt.Sprintf("webBasePath on %s after restore update = %q, want %q", currentBasePath, got, originalBasePath))
			continue
		}
		if err := client.SendRestart(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("restart panel on %s: %v", currentBasePath, err))
			continue
		}

		client.SetBasePath(originalBasePath)
		if err := client.WaitForReady(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("wait for %s: %v", originalBasePath, err))
			continue
		}
		if err := client.SetXrayOutboundTestURL(ctx, originalTestURL); err != nil {
			errs = append(errs, fmt.Sprintf("restore xray outbound test url on %s: %v", originalBasePath, err))
			continue
		}
		return nil
	}

	return fmt.Errorf("%s", strings.Join(errs, "; "))
}

// TestAccPanelSubscription_v328 verifies that v3.2.8 subscription fields
// (sub_clash_enable_routing, sub_clash_rules, sub_json_final_mask) round-trip
// through the 3x-ui API. Runs on every supported version (v3.2.8 predates the
// matrix floor), so no requireMinVersion gate is needed.
func TestAccPanelSubscription_v328(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_subscription" "test" {
  sub_enable              = true
  sub_json_enable         = true
  sub_path                = "/sub/"
  sub_port                = 2096
  sub_title               = "v328-test"
  sub_clash_enable        = true
  sub_clash_enable_routing = true
  sub_clash_rules         = "DIRECT,REJECT"
  sub_json_final_mask     = "{\"tcp\":\"mask\"}"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_clash_enable_routing", "true"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_clash_rules", "DIRECT,REJECT"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_json_final_mask", "{\"tcp\":\"mask\"}"),
				),
			},
			// Update — disable routing, change rules
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_subscription" "test" {
  sub_enable              = true
  sub_json_enable         = true
  sub_path                = "/sub/"
  sub_port                = 2096
  sub_title               = "v328-test-updated"
  sub_clash_enable        = true
  sub_clash_enable_routing = false
  sub_clash_rules         = ""
  sub_json_final_mask     = ""
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_clash_enable_routing", "false"),
					resource.TestCheckResourceAttr("threexui_panel_subscription.test", "sub_clash_rules", ""),
				),
			},
			// Import
			{
				ResourceName:      "threexui_panel_subscription.test",
				ImportState:       true,
				ImportStateVerify: true,
				// sub_show_info / sub_email_in_remark were removed from AllSetting in
				// 3x-ui v3.4.0 and do not round-trip through import there.
				ImportStateVerifyIgnore: []string{"sub_show_info", "sub_email_in_remark"},
			},
		},
	})
}

func TestAccPanelSecurityWriteOnly(t *testing.T) {
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.11.0"))),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_security" "test" {
  two_factor_enable           = false
  two_factor_token_wo         = "test-token-value"
  two_factor_token_wo_version = 1
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_security.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_security.test", "two_factor_enable", "false"),
					resource.TestCheckResourceAttr("threexui_panel_security.test", "two_factor_token_wo_version", "1"),
				),
			},
			// Idempotency: re-applying the same _wo config must produce an empty plan.
			// Verifies that mergeSettingsForUpdate replays the cached secret via
			// preserveCachedSettingSecrets when expand omits the field.
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_security" "test" {
  two_factor_enable           = false
  two_factor_token_wo         = "test-token-value"
  two_factor_token_wo_version = 1
}`,
				PlanOnly: true,
			},
			// Restore defaults
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_security" "test" {
  two_factor_enable = false
  two_factor_token  = ""
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_security.test", "id", "settings"),
				),
			},
		},
	})
}

func TestAccPanelSecurityWriteOnlyUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.11.0"))),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_security" "test" {
  two_factor_enable           = false
  two_factor_token_wo         = "token-v1"
  two_factor_token_wo_version = 1
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_security.test", "two_factor_token_wo_version", "1"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_security" "test" {
  two_factor_enable           = false
  two_factor_token_wo         = "token-v2"
  two_factor_token_wo_version = 2
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_security.test", "two_factor_token_wo_version", "2"),
				),
			},
		},
	})
}

// TestAccPanelSecurityWriteOnlyMigration verifies switching from the plain
// attribute to _wo: existing configs that migrate must continue to work.
func TestAccPanelSecurityWriteOnlyMigration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.11.0"))),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_security" "test" {
  two_factor_enable = false
  two_factor_token  = "plain-token"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_security.test", "two_factor_token", "plain-token"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_security" "test" {
  two_factor_enable           = false
  two_factor_token_wo         = "wo-token"
  two_factor_token_wo_version = 1
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_security.test", "two_factor_token_wo_version", "1"),
				),
			},
		},
	})
}

func TestAccPanelTelegramWriteOnly(t *testing.T) {
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.11.0"))),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_telegram" "test" {
  tg_bot_enable           = false
  tg_bot_token_wo         = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
  tg_bot_token_wo_version = 1
  tg_bot_chat_id          = ""
  tg_lang                 = "en"
  tg_run_time             = "@daily"
  tg_bot_backup           = false
  tg_bot_login_notify     = true
  tg_cpu                  = 80
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "tg_bot_enable", "false"),
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "tg_bot_token_wo_version", "1"),
				),
			},
			// Idempotency: re-applying the same _wo config must produce an empty plan.
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_telegram" "test" {
  tg_bot_enable           = false
  tg_bot_token_wo         = "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
  tg_bot_token_wo_version = 1
  tg_bot_chat_id          = ""
  tg_lang                 = "en"
  tg_run_time             = "@daily"
  tg_bot_backup           = false
  tg_bot_login_notify     = true
  tg_cpu                  = 80
}`,
				PlanOnly: true,
			},
			// Restore defaults
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_telegram" "test" {
  tg_bot_enable       = false
  tg_bot_token        = ""
  tg_bot_chat_id      = ""
  tg_lang             = "en"
  tg_run_time         = "@daily"
  tg_bot_backup       = false
  tg_bot_login_notify = true
  tg_cpu              = 80
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "id", "settings"),
				),
			},
		},
	})
}

func testAccPanelGeneralConfigWO(password string, version int) string {
	return fmt.Sprintf(`
resource "threexui_panel_general" "test" {
  date_picker                    = "gregorian"
  expire_diff                    = 0
  external_traffic_inform_enable = false
  external_traffic_inform_uri    = ""
  ldap_auto_create               = false
  ldap_auto_delete               = false
  ldap_base_dn                   = "dc=example,dc=com"
  ldap_bind_dn                   = "cn=admin,dc=example,dc=com"
  ldap_default_expiry_days       = 0
  ldap_default_limit_ip          = 0
  ldap_default_total_gb          = 0
  ldap_enable                    = true
  ldap_flag_field                = ""
  ldap_host                      = "ldap.example.com"
  ldap_inbound_tags              = ""
  ldap_invert_flag               = false
  ldap_password_wo               = %q
  ldap_password_wo_version       = %d
  ldap_port                      = 636
  ldap_sync_cron                 = "@every 1m"
  ldap_truthy_values             = "true,1,yes,on"
  ldap_use_tls                   = true
  ldap_user_attr                 = "mail"
  ldap_user_filter               = "(objectClass=person)"
  ldap_vless_field               = "vless_enabled"
  page_size                      = 25
  remark_model                   = "-ieo"
  session_max_age                = 360
  time_location                  = "Local"
  traffic_diff                   = 0
  web_base_path                  = "/"
  web_cert_file                  = ""
  web_domain                     = ""
  web_key_file                   = ""
  web_listen                     = ""
  web_port                       = 2053
  xray_outbound_test_url         = "https://www.google.com/generate_204"
}`, password, version)
}

func TestAccPanelGeneralWriteOnly(t *testing.T) {
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.11.0"))),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccPanelGeneralConfigWO("ldappass-wo", 1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_general.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_general.test", "ldap_enable", "true"),
					resource.TestCheckResourceAttr("threexui_panel_general.test", "ldap_password_wo_version", "1"),
				),
			},
			// Idempotency: re-applying the same _wo config must produce an empty plan.
			{
				Config:   testAccProviderConfig() + testAccPanelGeneralConfigWO("ldappass-wo", 1),
				PlanOnly: true,
			},
		},
	})
}

func TestAccPanelTelegramWriteOnlyUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.11.0"))),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_telegram" "test" {
  tg_bot_enable           = false
  tg_bot_token_wo         = "111:token-v1"
  tg_bot_token_wo_version = 1
  tg_bot_chat_id          = ""
  tg_lang                 = "en"
  tg_run_time             = "@daily"
  tg_bot_backup           = false
  tg_bot_login_notify     = true
  tg_cpu                  = 80
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "tg_bot_token_wo_version", "1"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_telegram" "test" {
  tg_bot_enable           = false
  tg_bot_token_wo         = "222:token-v2"
  tg_bot_token_wo_version = 2
  tg_bot_chat_id          = ""
  tg_lang                 = "en"
  tg_run_time             = "@daily"
  tg_bot_backup           = false
  tg_bot_login_notify     = true
  tg_cpu                  = 80
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "tg_bot_token_wo_version", "2"),
				),
			},
		},
	})
}

// TestAccPanelTelegramWriteOnlyMigration verifies switching from the plain
// tg_bot_token to tg_bot_token_wo.
func TestAccPanelTelegramWriteOnlyMigration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.11.0"))),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_telegram" "test" {
  tg_bot_enable       = false
  tg_bot_token        = "plain-token"
  tg_bot_chat_id      = ""
  tg_lang             = "en"
  tg_run_time         = "@daily"
  tg_bot_backup       = false
  tg_bot_login_notify = true
  tg_cpu              = 80
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "tg_bot_token", "plain-token"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_telegram" "test" {
  tg_bot_enable           = false
  tg_bot_token_wo         = "wo-token"
  tg_bot_token_wo_version = 1
  tg_bot_chat_id          = ""
  tg_lang                 = "en"
  tg_run_time             = "@daily"
  tg_bot_backup           = false
  tg_bot_login_notify     = true
  tg_cpu                  = 80
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_telegram.test", "tg_bot_token_wo_version", "1"),
				),
			},
		},
	})
}

// TestAccPanelGeneralOutbound verifies the panel_outbound attribute
// (Xray outbound egress bridge) introduced in 3x-ui v3.3.1.
func TestAccPanelGeneralOutbound(t *testing.T) {
	requireMinVersion(t, "v3.3.1")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "test" {
  panel_outbound = "test-egress"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_general.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_general.test", "panel_outbound", "test-egress"),
				),
			},
			// Update — clear the outbound tag.
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_general" "test" {
  panel_outbound = ""
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_general.test", "panel_outbound", ""),
				),
			},
			// ImportState
			{
				ResourceName:            "threexui_panel_general.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "settings",
				ImportStateVerifyIgnore: []string{"ldap_password", "remark_model", "panel_proxy"},
			},
		},
	})
}

// --- panel_discord (3x-ui v3.8.0+) ---

// TestAccPanelDiscord round-trips the Discord notifier settings. The enable
// flips exercise the provider-initiated panel restart (discordBotEnable feeds
// the startup-only alarm sampler registration, see panelSettingsNeedRestart).
func TestAccPanelDiscord(t *testing.T) {
	requireMinVersion(t, "v3.8.0")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_discord" "test" {
  discord_bot_enable      = true
  discord_bot_token       = "MTIzNDU2Nzg5MDEyMzQ1Njc4.Gabcde.fAkEtOkEnFaKeToKeNfAkE"
  discord_channel_id      = "1234567890123456789"
  discord_admin_ids       = "111111111111111111,222222222222222222"
  discord_run_time        = "@daily"
  discord_bot_backup      = false
  discord_cpu             = 80
  discord_memory          = 90
  discord_lang            = "en-US"
  discord_enabled_events  = "login,backup,cpu.high,memory.high"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_discord.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_discord.test", "discord_bot_enable", "true"),
					resource.TestCheckResourceAttr("threexui_panel_discord.test", "discord_channel_id", "1234567890123456789"),
					resource.TestCheckResourceAttr("threexui_panel_discord.test", "discord_admin_ids", "111111111111111111,222222222222222222"),
					resource.TestCheckResourceAttr("threexui_panel_discord.test", "discord_cpu", "80"),
					resource.TestCheckResourceAttr("threexui_panel_discord.test", "discord_enabled_events", "login,backup,cpu.high,memory.high"),
				),
			},
			// Update: disable the bot, move the language, nudge a threshold
			// within its alarm-registration class (80 -> 90 must not restart).
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_discord" "test" {
  discord_bot_enable      = false
  discord_bot_token       = ""
  discord_channel_id      = "1234567890123456789"
  discord_admin_ids       = "111111111111111111"
  discord_run_time        = "@every 6h"
  discord_bot_backup      = true
  discord_cpu             = 90
  discord_memory          = 90
  discord_lang            = "ru-RU"
  discord_enabled_events  = "login"
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_discord.test", "discord_bot_enable", "false"),
					resource.TestCheckResourceAttr("threexui_panel_discord.test", "discord_lang", "ru-RU"),
					resource.TestCheckResourceAttr("threexui_panel_discord.test", "discord_bot_backup", "true"),
				),
			},
			// ImportState: /setting/all blanks discordBotToken on v3.8.x, so
			// the token cannot be verified against state.
			{
				ResourceName:            "threexui_panel_discord.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "settings",
				ImportStateVerifyIgnore: []string{"discord_bot_token"},
			},
			// Idempotency
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_discord" "test" {
  discord_bot_enable      = false
  discord_bot_token       = ""
  discord_channel_id      = "1234567890123456789"
  discord_admin_ids       = "111111111111111111"
  discord_run_time        = "@every 6h"
  discord_bot_backup      = true
  discord_cpu             = 90
  discord_memory          = 90
  discord_lang            = "ru-RU"
  discord_enabled_events  = "login"
}`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccPanelDiscordWriteOnly mirrors TestAccPanelTelegramWriteOnly for the
// discord bot token (Terraform/OpenTofu 1.11+ write-only attributes).
func TestAccPanelDiscordWriteOnly(t *testing.T) {
	requireMinVersion(t, "v3.8.0")
	resource.Test(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(version.Must(version.NewVersion("1.11.0"))),
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_discord" "test" {
  discord_bot_enable         = false
  discord_bot_token_wo       = "MTIzNDU2Nzg5MDEyMzQ1Njc4.Gabcde.fAkEtOkEnFaKeToKeNfAkE"
  discord_bot_token_wo_version = 1
  discord_channel_id         = ""
  discord_admin_ids          = ""
  discord_run_time           = "@daily"
  discord_bot_backup         = false
  discord_cpu                = 80
  discord_memory             = 90
  discord_lang               = "en-US"
  discord_enabled_events     = ""
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_discord.test", "id", "settings"),
					resource.TestCheckResourceAttr("threexui_panel_discord.test", "discord_bot_enable", "false"),
					resource.TestCheckResourceAttr("threexui_panel_discord.test", "discord_bot_token_wo_version", "1"),
				),
			},
			// Idempotency: re-applying the same _wo config must produce an empty plan.
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_discord" "test" {
  discord_bot_enable         = false
  discord_bot_token_wo       = "MTIzNDU2Nzg5MDEyMzQ1Njc4.Gabcde.fAkEtOkEnFaKeToKeNfAkE"
  discord_bot_token_wo_version = 1
  discord_channel_id         = ""
  discord_admin_ids          = ""
  discord_run_time           = "@daily"
  discord_bot_backup         = false
  discord_cpu                = 80
  discord_memory             = 90
  discord_lang               = "en-US"
  discord_enabled_events     = ""
}`,
				PlanOnly: true,
			},
			// Restore defaults (plain attribute path).
			{
				Config: testAccProviderConfig() + `
resource "threexui_panel_discord" "test" {
  discord_bot_enable      = false
  discord_bot_token       = ""
  discord_channel_id      = ""
  discord_admin_ids       = ""
  discord_run_time        = "@daily"
  discord_bot_backup      = false
  discord_cpu             = 80
  discord_memory          = 90
  discord_lang            = "en-US"
  discord_enabled_events  = ""
}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_panel_discord.test", "id", "settings"),
				),
			},
		},
	})
}
