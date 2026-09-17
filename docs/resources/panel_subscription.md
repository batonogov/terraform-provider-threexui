---
page_title: "threexui_panel_subscription Resource - 3x-ui"
subcategory: "Panel Settings"
description: |-
  Manages subscription settings in the 3x-ui panel.
---

# threexui_panel_subscription (Resource)

Manages the subscription service settings of the 3x-ui panel.

This is a singleton resource -- only one instance should exist per provider. Deleting this resource only removes it from Terraform state; it does not reset the settings.

~> **Note:** Almost every field on this resource triggers a **panel restart**, and therefore brief panel downtime, when it changes. The 3x-ui subscription server reads its settings once, inside `(*sub.Server).initRouter()`, and freezes them into the running server; `initRouter` only runs when the panel starts. That covers the server-binding fields (`sub_enable`, `sub_listen`, `sub_domain`, `sub_port`, `sub_path`, `sub_cert_file`, `sub_key_file`), the route switches (`sub_json_enable`, `sub_json_path`, `sub_clash_enable`, `sub_clash_path`), the JSON/Clash body settings (`sub_json_mux`, `sub_json_rules`, `sub_json_final_mask`, `sub_json_observatory`, `sub_clash_rules`, …) and the page-presentation fields (`sub_title`, `sub_support_url`, `sub_profile_url`, `sub_announce`, `sub_hide_settings`, …). Without the restart the change would apply to the panel database and to Terraform state while every served subscription kept the old value — a silent no-op ([#291](https://github.com/batonogov/terraform-provider-threexui/issues/291), [#443](https://github.com/batonogov/terraform-provider-threexui/issues/443)). Only the three link-generation URIs — `sub_uri`, `sub_json_uri`, `sub_clash_uri` — are read per request and do **not** trigger a restart. A restart fires only on an actual value change: re-applying identical configuration does nothing.

~> **Note:** When `sub_port` (default `2096`) differs from the main panel port and the panel runs behind a reverse proxy, the proxy must be configured to forward subscription path requests to the subscription port. Without this, subscription URLs will return 404.

**Caddy** example:

```caddy
handle /sub/* {
    reverse_proxy 3x-ui:2096
}
```

**Nginx** example:

```nginx
location /sub/ {
    proxy_pass http://3x-ui:2096;
}
```

## Example Usage

```hcl
resource "threexui_panel_subscription" "settings" {
  sub_enable      = true
  sub_json_enable = true
  sub_port        = 2096
  sub_path        = "/sub/"
  sub_domain      = "sub.example.com"
}
```

## Argument Reference

### General

- `sub_enable` (Optional, Boolean) - Enable subscription service.
- `sub_json_enable` (Optional, Boolean) - Enable JSON subscription format.
- `sub_title` (Optional, String) - Subscription title.
- `sub_support_url` (Optional, String) - Support URL shown to clients.
- `sub_profile_url` (Optional, String) - Profile URL.
- `sub_announce` (Optional, String) - Announcement text.

### Routing

- `sub_enable_routing` (Optional, Boolean) - Enable routing in subscriptions.
- `sub_routing_rules` (Optional, String) - Routing rules for subscriptions.
- `sub_incy_enable_routing` (Optional, Boolean) - Enable routing injection for the Incy subscription client (3x-ui v3.4.1+).
- `sub_incy_routing_rules` (Optional, String) - Incy routing deep-link injected into the subscription body (3x-ui v3.4.1+).

### Server

- `sub_listen` (Optional, String) - Listen address.
- `sub_port` (Optional, Number) - Subscription server port.
- `sub_path` (Optional, String) - Subscription URL path.
- `sub_domain` (Optional, String) - Subscription domain.
- `sub_cert_file` (Optional, String) - TLS certificate file path.
- `sub_key_file` (Optional, String) - TLS key file path.
- `sub_updates` (Optional, Number) - Update interval in hours.
- `sub_encrypt` (Optional, Boolean) - Encrypt subscription data.
- `sub_show_info` (Optional, Boolean, Deprecated) - Show info in subscription. **Deprecated:** Removed from 3x-ui v3.4.0; accepted but has no effect on v3.4.0+ panels.
- `sub_email_in_remark` (Optional, Boolean, Deprecated) - Include the client email in subscription profile names. Default is `true` on 3x-ui v3.0.2+. **Deprecated:** Removed from 3x-ui v3.4.0; accepted but has no effect on v3.4.0+ panels.
- `sub_theme_dir` (Optional, String) - Absolute path to a folder containing a custom subscription page template. Added in 3x-ui v3.3.0; ignored by older panels.
- `remark_template` (Optional, String) - Subscription remark template (`{{VAR}}` tokens rendered per client, e.g. Jalali date/transport/status). Added in 3x-ui v3.4.0; ignored by older panels.
- `sub_hide_settings` (Optional, Boolean) - Hide server settings in happ subscription (Happ only). Added in 3x-ui v3.4.0; ignored by older panels.

### URI

- `sub_uri` (Optional, String) - Subscription URI.
- `sub_json_path` (Optional, String) - JSON subscription path.
- `sub_json_uri` (Optional, String) - JSON subscription URI.
- `sub_json_fragment` (Optional, String) - JSON fragment settings. The expected format depends on the 3x-ui version:
  - **v2.9.2+:** only the fragment parameters object, e.g. `{"packets":"tlshello","length":"100-200","interval":"10-20","maxSplit":"300-400"}`.
  - **v2.9.1 and earlier:** full outbound object with `tag`, `protocol`, `settings`, and `streamSettings`.
  - **Deprecated in 3x-ui v3.2.8** — replaced by `sub_clash_enable_routing`.
- `sub_json_noises` (Optional, String) - JSON noise settings. The expected format depends on the 3x-ui version:
  - **v2.9.2+:** only the noises array, e.g. `[{"type":"rand","packet":"10-20","delay":"10-16","applyTo":"ip"}]`.
  - **v2.9.1 and earlier:** full outbound object with `tag`, `protocol`, `settings`, and `streamSettings`.
  - **Deprecated in 3x-ui v3.2.8** — replaced by `sub_clash_rules`.
- `sub_json_mux` (Optional, String) - JSON mux settings.
- `sub_json_rules` (Optional, String) - JSON rules.
- `sub_json_observatory` (Optional, String) - Observatory block injected into the JSON subscription for client-side balancers, as a JSON string. Added in 3x-ui v3.7.0. Older panels do not carry the key at all, so the attribute reads back as null there — leave it unset on a pre-v3.7.0 panel rather than configuring an empty value. Like `sub_json_mux`, `sub_json_rules` and `sub_json_final_mask`, this value is read once when the subscription server starts, so changing it restarts the panel (see the note at the top).
- `sub_json_auto_detect` (Optional, Boolean) - Auto-detect JSON subscription format by User-Agent. Added in 3x-ui v3.6.0; ignored by older panels.
- `sub_json_always_array` (Optional, Boolean) - Always output the JSON subscription as an array. Added in 3x-ui v3.6.0; ignored by older panels.
- `sub_json_user_agent_regex` (Optional, String) - User-Agent regex for JSON subscription auto-detection. Added in 3x-ui v3.6.0; ignored by older panels.

### Clash / Mihomo

- `sub_clash_enable` (Optional, Boolean) - Enable Clash/Mihomo subscription endpoint.
- `sub_clash_path` (Optional, String) - Path for Clash/Mihomo subscription endpoint.
- `sub_clash_uri` (Optional, String) - Clash/Mihomo subscription server URI.
- `sub_clash_enable_routing` (Optional, Boolean) - Enable global routing rules for Clash/Mihomo subscriptions. Available since 3x-ui v3.2.8.
- `sub_clash_rules` (Optional, String) - Clash/Mihomo global routing rules. Available since 3x-ui v3.2.8.
- `sub_clash_auto_detect` (Optional, Boolean) - Auto-detect Clash subscription format by User-Agent. Added in 3x-ui v3.6.0; ignored by older panels.
- `sub_clash_user_agent_regex` (Optional, String) - User-Agent regex for Clash subscription auto-detection. Added in 3x-ui v3.6.0; ignored by older panels.
- `sub_json_final_mask` (Optional, String) - JSON subscription global finalmask (tcp/udp masks and quicParams). Available since 3x-ui v3.2.8.

## Attribute Reference

All arguments are also exported as attributes.

## Import

```shell
terraform import threexui_panel_subscription.settings settings
```

## v3.8.0+ subscription settings

All attributes in this section require 3x-ui **v3.8.0+**; older panels ignore the values and read them back empty. The one exception is `sub_profile_mode`, which landed in **v3.8.5** (#6538 was merged between v3.8.0 and v3.8.5) — on v3.8.0 the panel does not know the key and the attribute reads back null. `sub_profile_mode`, `sub_json_routing_rules`, `sub_json_dns` and the whole `sub_happ_*` / Happ block are frozen into the subscription server at startup — changing them triggers a **panel restart** (see the note at the top); `sub_info_node_enable`, `sub_calendar_expire_inclusive`, `sub_expired_template`, `sub_traffic_depleted_template` and `happ_link_enable` are read per request and do not restart.

- `sub_profile_mode` (Optional, String) - Subscription profile page mode: `none`, `builtin`, or `custom`. Requires 3x-ui v3.8.5+: the panel turned the built-in profile page off by default and stopped advertising it in subscription responses ([3x-ui #6538](https://github.com/MHSanaei/3x-ui/pull/6538)); an existing `sub_profile_url` maps to `custom`. Empty is treated as `none` by the panel.
- `sub_info_node_enable` (Optional, Boolean) - Add a dummy info/status node to client node lists.
- `sub_calendar_expire_inclusive` (Optional, Boolean) - Present expiry at month end instead of a rolling interval.
- `sub_expired_template` (Optional, String) - Template for expired-client subscription pages.
- `sub_traffic_depleted_template` (Optional, String) - Template for depleted-client subscription pages.
- `sub_json_routing_rules` (Optional, String) - Client routing rules baked into JSON subscriptions, as a JSON string.
- `sub_json_dns` (Optional, String) - Panel-chosen DNS servers baked into JSON subscriptions, as a JSON string.
- `happ_link_enable` (Optional, Boolean) - Enable the Happ app link integration.

### Happ client customization

- `sub_happ_auto_detect` (Optional, Boolean) - Auto-detect the client app.
- `sub_happ_provider_id` (Optional, String) - Provider ID.
- `sub_happ_new_url` (Optional, String) - New-user URL.
- `sub_happ_fallback_url` (Optional, String) - Fallback URL.
- `sub_happ_sub_info_color` (Optional, String) - Subscription info color.
- `sub_happ_sub_info_text` (Optional, String) - Subscription info text.
- `sub_happ_sub_info_button_text` (Optional, String) - Subscription info button text.
- `sub_happ_sub_info_button_link` (Optional, String) - Subscription info button link.
- `sub_happ_sub_expire` (Optional, Boolean) - Show subscription expiry.
- `sub_happ_sub_expire_button_link` (Optional, String) - Expiry button link.
- `sub_happ_notification_expire` (Optional, Boolean) - Expiry notification.
- `sub_happ_no_limit` (Optional, Boolean) - No-limit profile.
- `sub_happ_always_hwid` (Optional, Boolean) - Always use HWID limits.
- `sub_happ_tun_mode` (Optional, String) - Tun mode.
- `sub_happ_tun_type` (Optional, String) - Tun type.
- `sub_happ_exclude_routes` (Optional, String) - Excluded routes.
- `sub_happ_exclude_apns` (Optional, Boolean) - Exclude APNs.
- `sub_happ_color_profile` (Optional, String) - Color profile.
- `sub_happ_ping_type` (Optional, String) - Connectivity-check ping type.
- `sub_happ_auto_connect` (Optional, Boolean) - Auto-connect.
- `sub_happ_auto_connect_type` (Optional, String) - Auto-connect type.
- `sub_happ_per_app_mode` (Optional, String) - Per-app proxy mode.
- `sub_happ_per_app_list` (Optional, String) - Per-app list.
