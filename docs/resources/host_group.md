---
page_title: "threexui_host_group Resource - 3x-ui"
subcategory: "Cluster"
description: |-
  Manages a 3x-ui host group (bulk host management). Requires 3x-ui v3.5.0+.
---

# threexui_host_group (Resource)

Manages a 3x-ui host group — a bulk host-management primitive that applies a shared set of link/SNI/security overrides to one or more inbounds at once. Wraps the 3x-ui host group API (`/panel/api/hosts/*`), available since **3x-ui v3.5.0**.

> **Version requirement:** this resource only works against 3x-ui v3.5.0+ panels. On older panels the `/panel/api/hosts/*` routes do not exist and operations will fail.
>
> **Server-generated group id:** if `group_id` is omitted, the panel generates a 16-digit numeric id on create. Once created, changing `group_id` forces a replace.

## Example Usage

```hcl
resource "threexui_host_group" "premium_eu" {
  remark    = "Premium EU nodes"
  port      = 443
  security  = "tls"
  sni       = "eu.example.com"
  alpn      = ["h2", "http/1.1"]

  inbound_ids = [1, 2, 3]
}
```

Import an existing host group by its group id:

```bash
terraform import threexui_host_group.premium_eu 1234567890123456
```

## Argument Reference

- `group_id` (Optional, String) - Host group identifier. If omitted on create, the panel generates a 16-digit numeric string. Changing this forces a new resource.
- `inbound_ids` (Required, List of Number) - Inbound ids this host group applies to. At least one is required.
- `remark` (Required, String) - Remark / display name (max 256 chars).
- `server_description` (Optional, String) - Server description shown to clients (max 64 chars).
- `sort_order` (Optional, Number) - Sort order for display.
- `is_disabled` (Optional, Boolean) - Whether the group is disabled.
- `is_hidden` (Optional, Boolean) - Whether the group is hidden from client views.
- `port` (Optional, Number) - Override port for the generated share links (0–65535).
- `security` (Optional, String) - Link security scheme. One of: `same`, `tls`, `none`, `reality`.
- `sni` (Optional, String) - Server Name Indication override.
- `host_header` (Optional, String) - Host header override.
- `path` (Optional, String) - Path override for the generated share links.
- `alpn` (Optional, List of String) - ALPN protocol list (e.g. `["h2", "http/1.1"]`).
- `fingerprint` (Optional, String) - TLS fingerprint (uTLS).
- `cipher_suites` (Optional, String) - TLS cipher suites for the generated share links. Added in 3x-ui v3.9.0; empty means the panel default.
- `override_sni_from_address` (Optional, Boolean) - Derive the SNI from the host address.
- `keep_sni_blank` (Optional, Boolean) - Keep the SNI blank in share links.
- `pinned_peer_cert_sha256` (Optional, List of String) - Pinned peer certificate SHA-256 hashes.
- `verify_peer_cert_by_name` (Optional, String) - Verify peer certificate by name.
- `allow_insecure` (Optional, Boolean) - Allow insecure TLS connections.
- `ech_config_list` (Optional, String) - ECH config list (opaque JSON string).
- `mux_params` (Optional, String) - MUX parameters (opaque JSON string).
- `sockopt_params` (Optional, String) - Socket options parameters (opaque JSON string).
- `final_mask` (Optional, String) - Final mask type (e.g. `tcp`).
- `vless_route` (Optional, String) - VLESS route override.
- `exclude_from_sub_types` (Optional, List of String) - Subscription types to exclude this group from.
- `node_guids` (Optional, List of String) - Node GUIDs this group is scoped to.
- `mihomo_ip_version` (Optional, String) - Mihomo IP version preference. One of: `dual`, `ipv4`, `ipv6`, `ipv4-prefer`, `ipv6-prefer`.
- `mihomo_x25519` (Optional, Boolean) - Use Mihomo X25519 key exchange.
- `shuffle_host` (Optional, Boolean) - Shuffle the host order in generated links.
- `hosts` (Optional, List of String) - Explicit host list for this group.
- `tags` (Optional, List of String) - Tags assigned to this host group.

## Attribute Reference

- `id` - The host group identifier (`group_id`).

## Import

Host groups can be imported using the `group_id`:

```bash
terraform import threexui_host_group.example <group_id>
```
