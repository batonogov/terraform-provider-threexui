---
page_title: "threexui_panel_general Resource - 3x-ui"
subcategory: "Panel Settings"
description: |-
  Manages general panel settings in the 3x-ui panel.
---

# threexui_panel_general (Resource)

Manages the general settings of the 3x-ui panel including web server configuration, LDAP integration, and display preferences.

This is a singleton resource -- only one instance should exist per provider. Deleting this resource only removes it from Terraform state; it does not reset the panel settings.

~> **Warning:** Changing `web_base_path` requires updating the provider's `base_path` to match. Otherwise the provider will lose connectivity to the panel.

~> **Note:** Changing any of the web-server binding fields — `web_listen`, `web_domain`, `web_port`, `web_base_path`, `web_cert_file`, `web_key_file`, `session_max_age` — or the scheduler wiring — `time_location`, `ldap_enable`, `ldap_sync_cron` — triggers a **panel restart** and therefore brief panel downtime. The panel binds its listener and registers its cron jobs once at startup: `time_location` fixes the timezone every scheduled job runs in, and `ldap_enable`/`ldap_sync_cron` decide whether the LDAP sync job is registered at all and on what schedule. Without the restart the change applies to the panel database and to Terraform state while the running panel keeps the old schedule. A restart fires only on an actual value change: re-applying identical configuration does nothing.

## Example Usage

```hcl
resource "threexui_panel_general" "settings" {
  web_port      = 2053
  web_base_path = "/panel/"
  page_size     = 50
  time_location = "UTC"
}
```

## Argument Reference

### Web Server

- `web_listen` (Optional, String) - Listen address. Default is `""` (all interfaces).
- `web_domain` (Optional, String) - Domain name. Default is `""`.
- `web_port` (Optional, Number) - Web panel port. Default is `2053`.
- `web_base_path` (Optional, String) - Base URL path. Default is `/`.
- `web_cert_file` (Optional, String) - TLS certificate file path. Default is `""`.
- `web_key_file` (Optional, String) - TLS key file path. Default is `""`.
- `session_max_age` (Optional, Number) - Session max age in minutes. Default is `360`.
- `trusted_proxy_cidrs` (Optional, String) - Comma-separated trusted reverse proxy IPs/CIDRs used for forwarded headers. Default is `127.0.0.1/32,::1/128` on 3x-ui v3.0.2+.
- `warp_update_interval` (Optional, Number) - Interval (hours) between Cloudflare WARP / geo auto-updates via Xray-core (0 disables). Added in 3x-ui v3.3.0; ignored by older panels.

### Display

- `page_size` (Optional, Number) - Items per page. Default is `25`.
- `remark_model` (Optional, String, Deprecated) - Remark display model. Default is `-ieo`. **Deprecated:** Removed from 3x-ui v3.4.0 (superseded by `remark_template`). Use `remark_template` on v3.4.0+; accepted but has no effect on v3.4.0+ panels.
- `date_picker` (Optional, String) - Date picker type. Default is `gregorian`.
- `time_location` (Optional, String) - Timezone. Default is `Local`.
- `expire_diff` (Optional, Number) - Expiry diff threshold. Default is `0`.
- `traffic_diff` (Optional, Number) - Traffic diff threshold. Default is `0`.

### External Traffic

- `external_traffic_inform_enable` (Optional, Boolean) - Enable external traffic notifications. Default is `false`.
- `external_traffic_inform_uri` (Optional, String) - External traffic notification URI. Default is `""`.
- `sub_show_identity_on_all_links` (Optional, Boolean) - Add identity tokens to every subscription link. Added in 3x-ui v3.6.0; ignored by older panels.
- `outbound_down_threshold` (Optional, Number) - Consecutive-failure threshold before the `outbound.down` notification fires (1-100). Added in 3x-ui v3.6.0; older panels report `0` (unsupported).
- `restart_xray_on_client_disable` (Optional, Boolean) - Restart Xray when clients are automatically disabled by expiry or traffic limit. Default is `true` on 3x-ui v2.9.4+.
- `ip_limit_allowlist` (Optional, String) - Comma-separated addresses or CIDRs exempt from the per-client IP limit. Added in 3x-ui v3.7.0; older panels report an empty string (unsupported). Entries are validated at plan time with the same rules the panel uses (`netip`), which rejects zero-padded prefixes such as `10.0.0.0/024`.
- `reality_scan_candidates` (Optional, String) - REALITY target scan candidates (comma-separated `host:port` entries the panel scans when picking a dest). Added in 3x-ui v3.8.0; older panels report an empty string (unsupported).

### LDAP

- `ldap_enable` (Optional, Boolean) - Enable LDAP. Default is `false`.
- `ldap_host` (Optional, String) - LDAP server host. Default is `""`.
- `ldap_port` (Optional, Number) - LDAP server port. Default is `389`.
- `ldap_use_tls` (Optional, Boolean) - Use TLS for LDAP. Default is `false`.
- `ldap_insecure_skip_verify` (Optional, Boolean) - Skip verification of the LDAP server's TLS certificate. 3x-ui v3.4.2+; ignored by older panels. Default is `false`.
- `ldap_bind_dn` (Optional, String) - Bind DN. Default is `""`.
- `ldap_password` (Optional, String, Sensitive) - Bind password. Default is `""`.
- `ldap_password_wo` (Optional, String, WriteOnly) - Write-only version of `ldap_password`. Not persisted in state. Terraform 1.11+ / OpenTofu 1.11+.
- `ldap_password_wo_version` (Optional, Number) - Increment to trigger re-send of `ldap_password_wo`. Must be set together with `ldap_password_wo`.
- `ldap_base_dn` (Optional, String) - Base DN. Default is `""`.
- `ldap_user_filter` (Optional, String) - User filter. Default is `(objectClass=person)`.
- `ldap_user_attr` (Optional, String) - User attribute. Default is `mail`.
- `ldap_vless_field` (Optional, String) - VLESS field name. Default is `vless_enabled`.
- `ldap_sync_cron` (Optional, String) - Sync cron expression. Default is `@every 1m`.
- `ldap_flag_field` (Optional, String) - Flag field name. Default is `""`.
- `ldap_truthy_values` (Optional, String) - Truthy values. Default is `true,1,yes,on`.
- `ldap_invert_flag` (Optional, Boolean) - Invert flag logic. Default is `false`.
- `ldap_inbound_tags` (Optional, String) - Inbound tags. Default is `""`.
- `ldap_auto_create` (Optional, Boolean) - Auto-create clients from LDAP. Default is `false`.
- `ldap_auto_delete` (Optional, Boolean) - Auto-delete clients from LDAP. Default is `false`.
- `ldap_default_total_gb` (Optional, Number) - Default traffic limit for LDAP clients (GB). Default is `0`.
- `ldap_default_expiry_days` (Optional, Number) - Default expiry days for LDAP clients. Default is `0`.
- `ldap_default_limit_ip` (Optional, Number) - Default IP limit for LDAP clients. Default is `0`.

### Xray

- `xray_outbound_test_url` (Optional, String) - URL used for testing outbound connectivity. Default is `https://www.google.com/generate_204`.

### Proxy

- `panel_proxy` (Optional, String, Deprecated) - HTTP/SOCKS5 proxy URL for the panel's own outbound requests (xray version checks, Telegram bot, outbound testing). Default is `""` (direct). Available on 3x-ui v3.2.0 through v3.3.0. **Deprecated:** superseded by `panel_outbound` on 3x-ui v3.3.1+; ignored by v3.3.1+ panels. Use `panel_outbound` for new configurations.
- `panel_outbound` (Optional, String) - Xray outbound tag (or balancer tag) used for the panel's own outbound HTTP (update checks/downloads, Telegram, geo updates, outbound-subscription fetches). Default is `""` (direct egress). Available on 3x-ui v3.3.1+. Ignored by older panels; use `panel_proxy` on 3x-ui v3.2.0 through v3.3.0.

## Attribute Reference

All arguments are also exported as attributes.

## Import

```shell
terraform import threexui_panel_general.settings settings
```
