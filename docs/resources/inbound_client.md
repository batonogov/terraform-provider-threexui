---
page_title: "threexui_inbound_client Resource - 3x-ui"
subcategory: "Inbound"
description: |-
  Manages a client within an inbound in the 3x-ui panel.
---

# threexui_inbound_client (Resource)

Manages a client within an existing inbound. Clients are users that can connect through the inbound proxy.

## Example Usage

```hcl
resource "threexui_inbound" "vless" {
  port     = 443
  protocol = "vless"
  enable   = true
  remark   = "VLESS Reality"

  vless_settings {
    decryption = "none"
  }

  stream_settings {
    network  = "tcp"
    security = "reality"
    reality_settings {
      target       = "www.amazon.com:443"
      server_names = ["www.amazon.com"]
    }
  }
}

resource "threexui_inbound_client" "user1" {
  inbound_id  = threexui_inbound.vless.id
  email       = "user1@example.com"
  enable      = true
  total_gb    = 10 * 1024 * 1024 * 1024 # 10 GiB; the API value is bytes
  expiry_time = 1735689600000
  comment     = "Main account"
}
```

### Hysteria Client

```hcl
resource "threexui_inbound" "hysteria" {
  port     = 8443
  protocol = "hysteria"
  enable   = true
  remark   = "Hysteria"

  hysteria_settings {
    version = 2
  }

  stream_settings {
    network  = "hysteria"
    security = "tls"
  }
}

resource "threexui_inbound_client" "hysteria_user" {
  inbound_id = threexui_inbound.hysteria.id
  email      = "hysteria-user@example.com"
  auth       = "my-secret-auth"
  enable     = true
}
```

## Argument Reference

- `inbound_id` (Required, Number, ForceNew) - The ID of the inbound this client belongs to. Changing this forces a new resource.
- `email` (Required, String) - Unique email identifier for the client. Required by 3x-ui to avoid database errors.
- `client_id` (Optional, String, Sensitive, ForceNew) - UUID credential for the client. Auto-generated if not provided. Changing this forces a new resource.
- `security` (Optional, String) - Security type.
- `password` (Optional, String, Sensitive) - Client password (used by trojan/shadowsocks).
- `password_wo` (Optional, String, Write-Only) - Write-only client password. Requires Terraform/OpenTofu 1.11+. Use `password_wo_version` to trigger later updates without storing the password in plan or state.
- `password_wo_version` (Optional, Number) - Increment this value when `password_wo` changes. It can only be configured together with `password_wo`.
- `flow` (Optional, String) - Flow control (e.g. `xtls-rprx-vision`). Leave unset when the parent `threexui_inbound` has `disable_flow = true`: the panel blanks `flow` on every client of such an inbound, so a configured value fails the apply as an inconsistent result.
- `reverse_tag` (Optional, String) - VLESS reverse tag. Stored in 3x-ui as `reverse.tag` and available on 3x-ui v2.9.4+.
- `auth` (Optional, String, Sensitive) - Auth password for Hysteria clients. Used as client identifier instead of UUID.
- `limit_ip` (Optional, Number) - Maximum concurrent connections.
- `total_gb` (Optional, Number) - Traffic quota in bytes. The name mirrors the 3x-ui API field, but the API does not convert gigabytes; for example, use `10 * 1024 * 1024 * 1024` for 10 GiB.
- `expiry_time` (Optional, Number) - Expiry time as Unix timestamp in milliseconds.
- `enable` (Optional, Boolean) - Whether the client is enabled.
- `tg_id` (Optional, Number) - Telegram user ID for bot notifications.
- `comment` (Optional, String) - Client description for administrative notes.
- `reset` (Optional, Number) - Traffic reset period in days. `0` means never (default).
- `group` (Optional, String) - Client group name. Available on 3x-ui v3.2.0+.
- `reset_day` (Optional, Number) - Calendar day of month (1-31) on which this client renews. `0` keeps the rolling-interval behaviour driven by `reset`. Added in 3x-ui v3.7.0; older panels report `0` (unsupported).
- `reset_max` (Optional, Number) - Maximum number of automatic renewals for this client. `0` means unlimited. Added in 3x-ui v3.7.0; older panels report `0` (unsupported).
- `reset_weekday` (Optional, Number) - Calendar weekday (1-7, Monday-Sunday) on which this client renews weekly. `0` disables weekly renewal (default). Mutually exclusive with a positive `reset` or `reset_day`: the panel rejects the combination, and the provider catches it at plan time. Added in 3x-ui v3.9.0; older panels report `0` (unsupported).
- `traffic_reset` (Optional, String) - Per-client traffic reset cycle, independent of the inbound's own cycle: `never`, `hourly`, `daily`, `weekly`, or `monthly`. Added in 3x-ui v3.7.0; older panels report an empty value (unsupported).
- `traffic_reset_day` (Optional, Number) - Day of month (1-31) for this client's monthly traffic resets. Only effective when `traffic_reset = "monthly"`. Added in 3x-ui v3.7.0; older panels report `0` (unsupported). `0` is rejected at plan time: the panel clamps any value below 1 up to 1, so a configured `0` could never round-trip. On v3.7.0+ the panel also normalizes an unset `traffic_reset` to `never`.
- `secret` (Optional, String, Sensitive) - MTProto FakeTLS secret, per-client (3x-ui v3.5.0+, `mtg-multi` engine). Format: `"ee"` + 32 hex chars (random middle) + hex-encoded domain suffix. The panel rebuilds the domain suffix from the inbound's `fakeTlsDomain` on save, so only the random middle must be stable across applies. Setting a domain suffix that differs from the inbound's `fakeTlsDomain` causes drift after the first apply (the panel heals it) — leave unset to let the panel generate it. Leave unset for non-MTProto clients.
- `secret_wo` (Optional, String, Write-Only) - Write-only MTProto FakeTLS secret. Requires Terraform/OpenTofu 1.11+. Use `secret_wo_version` to trigger later updates without storing the secret in plan or state.
- `secret_wo_version` (Optional, Number) - Increment this value when `secret_wo` changes. It can only be configured together with `secret_wo`.
- `ad_tag` (Optional, String) - MTProto advertising tag from @MTProxybot, per-client (3x-ui v3.5.0+). Must be exactly 32 hex characters. Leave unset for non-MTProto clients.
- `restart_xray` (Optional, Boolean) - Restart Xray core after create, update, or delete operations. Default is `false`.
- `sub_id` (Optional, String) - Subscription ID used for subscription URLs. Auto-generated if not provided. Set this to preserve existing subscription links when restoring from backup.

## Read-Only Attributes

- `id` (String, Sensitive) - Composite resource identifier in `inbound_id:client_id` format. It is sensitive because it embeds the client UUID credential.

## Usage: Subscription URLs

When the panel subscription service is enabled (`threexui_panel_subscription`),
each client receives a `sub_id`. Use it to build subscription
URLs that clients can import into v2rayN, Hiddify, Streisand, or any compatible app:

```hcl
resource "threexui_inbound_client" "user1" {
  inbound_id = threexui_inbound.vless.id
  email      = "user1@example.com"
  enable     = true
  comment    = "Main account"
}

output "user1_subscription_url" {
  value = "https://your-domain.com/sub/${threexui_inbound_client.user1.sub_id}"
}
```

For multiple clients:

```hcl
locals {
  clients = {
    user1 = threexui_inbound_client.user1
    user2 = threexui_inbound_client.user2
  }
}

output "subscription_urls" {
  value = { for name, client in local.clients : name => "https://your-domain.com/sub/${client.sub_id}" }
}
```

> **Note:** The subscription path (`/sub/`) and port (`2096`) depend on your
> `threexui_panel_subscription` configuration. Adjust the URL accordingly.

## Attribute Reference

All non-write-only arguments are also exported as attributes. Values supplied through `password_wo` or `secret_wo` are deliberately absent from plan and state; only their version triggers are persisted. On Terraform/OpenTofu versions before 1.11, configure the corresponding plain `password` or `secret` attribute instead.

## Import

Inbound clients can be imported using `inbound_id:client_id`:

```shell
terraform import threexui_inbound_client.example 1:550e8400-e29b-41d4-a716-446655440000
```
