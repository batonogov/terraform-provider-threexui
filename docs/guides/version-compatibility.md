---
page_title: "Version compatibility with 3x-ui"
subcategory: "Guides"
description: |-
  Which 3x-ui releases are tested, what changed between them, and how the provider handles version-specific features.
---

# Version compatibility with 3x-ui

The 3x-ui panel evolves quickly, and some releases introduce breaking API changes that affect the provider. This guide documents which versions are supported, what changed between them, and how to pin the version you need.

## Support policy

The provider officially supports every released patch across all supported 3x-ui minor lines — see the compatibility table in the README. The acceptance matrix exercises each version on each push to `main` and every pull request. Older lines (2.9.x, 3.0.x and earlier) were dropped from the test matrix in provider v3.4.0; 3.1.x was dropped when 3.7.x was added; 3.2.x was dropped when 3.8.x was added.

## Compatibility table

<!-- sync-versions:begin -->
| 3x-ui version | Status | Notes |
| --- | --- | --- |
| v3.8.5 | Tested | Rebuilt subscription page; Xray survives a bad config (port-collision saves and enables are refused); large-fleet node sync; empty REALITY `min_client_ver` now means no minimum (xray-core v26.9.9). |
| v3.8.0 | Tested | TUIC v5 inbounds, AmneziaWG outbounds, a Discord notification bot (10 `discord*` settings), Happ client customization (`subHapp*`), `sub_profile_mode`, xray-core v26.9.9 with one-time template migrations, and hardened installs. |
| v3.7.0 | Tested | Native AmneziaWG inbounds, calendar-day client renewals with a per-client traffic reset cycle, inbound `disable_flow`, an IP-limit allowlist, and scoped API tokens. |
| v3.6.0 | Tested | Node `apiToken` becomes write-only ([3x-ui #5613](https://github.com/MHSanaei/3x-ui/pull/5613)); xray-core v26.7.28. |
| v3.5.0 | Tested | Host groups, MTProto multi-client support, Xray `env`, outbound `target_strategy`, and expanded balancer settings. |
| v3.4.2 | Tested | WireGuard multi-client support, `ldap_insecure_skip_verify`, and Xray Observatory/BurstObservatory. |
| v3.4.1 | Tested | Incy subscription routing injection settings. |
| v3.4.0 | Tested | SMTP notifications and expanded Telegram/subscription settings. |
| v3.3.1 | Tested | Live config apply; `panelProxy` replaced by the `panelOutbound` egress bridge. |
| v3.3.0 | Tested | `subThemeDir`, `warpUpdateInterval`, MTProto, and the node-sync surface. |
<!-- sync-versions:end -->

Older lines (3.2.x, 3.1.x, 3.0.x, 2.9.x and earlier) are no longer tested. The provider may still work, but compatibility is not guaranteed.

## Known issues

### v2.9.1 — `InstallXray` skips execution

In 3x-ui v2.9.1, the `InstallXray` API handler returns success without actually running the installation. This is an upstream bug fixed in later patches. The `threexui_xray_version` resource may report success while Xray is not updated. Upgrade to v2.9.2 or later to resolve this.

### v3.0.0+ — CSRF-protected requests

Starting with v3.0.0, 3x-ui requires a CSRF token for all unsafe HTTP methods (POST, PUT, DELETE). The provider handles this automatically:

1. On startup, it fetches an anonymous CSRF token from `GET /csrf-token` (v3.x only).
2. The token is sent as the `X-CSRF-Token` header on every mutating request.
3. On a 403 (stale token), the provider refreshes via `GET /panel/csrf-token` and retries once.

No configuration is needed on your end. The CSRF flow is transparent.

### v3.2.0+ — legacy inbound protocols removed upstream

3x-ui v3.2.0 removed the legacy `socks` and `dokodemo-door` protocol entries from the current upstream UI/API surface. Use `mixed` instead of `socks`, and `tunnel` instead of `dokodemo-door`. The provider keeps compatibility paths for older supported panels and imported state, but new configurations targeting v3.2.0+ should use the current protocol names.

## Breaking changes

### v2.9.0 — WireGuard MTU field

In 3x-ui v2.9.0, the WireGuard `mtu` attribute changed from a single integer to a list of two integers `[v4, v6]`. The provider schema exposes `mtu` as a list. If you are upgrading from v2.8.x, update your Terraform configuration:

```hcl
# Before (v2.8.x)
mtu = 1280

# After (v2.9.0+)
mtu = [1280, 1280]
```

WireGuard also gained `gateway` and `dns` list fields in v2.9.0.

### v2.9.0 — KCP congestion fields

The KCP congestion fields `congestion`, `read_buffer_size`, and `write_buffer_size` were replaced by `cwnd_multiplier` and `max_sending_window`. Update your KCP stream settings if you used these attributes.

### v2.9.0 — Sniffing exclusions

The sniffing block gained `ips_excluded` and `domains_excluded` fields. These are optional and backward-compatible — existing configurations are unaffected.

### v3.0.0 — CSRF tokens

See [v3.0.0+ CSRF-protected requests](#v300--csrf-protected-requests) above. This is a breaking change in the 3x-ui API surface, not in the Terraform schema. The provider abstracts it away, but custom scripts or direct API calls targeting v3.0.0+ must include CSRF handling.

### v3.0.0 — Inbound `nodeId`

Inbound objects gained a `nodeId` field for the multi-node feature introduced in v3.0.0. The provider ignores this field in single-panel setups.

### v3.6.0 — Node `apiToken` is write-only

Starting with 3x-ui v3.6.0 ([#5613](https://github.com/MHSanaei/3x-ui/pull/5613)), the node API token is **write-only**: the panel no longer returns `apiToken` on `GET /panel/api/nodes`. The token must be stored at creation time.

The provider handles this transparently for `terraform apply` — the configured value is preserved in state across reads. However, `terraform import` cannot recover the token because the panel cannot echo it back. After importing a `threexui_node` resource on v3.6.0+, set `api_token` (or `api_token_wo`) in your configuration and run `terraform apply` to persist it:

```hcl
import {
  to = threexui_node.example
  id = "1"
}

resource "threexui_node" "example" {
  name      = "de-fra-1"
  address   = "node1.example.com"
  port      = 2053
  api_token = "your-token-here" # must be provided — panel won't return it
}
```

## Version-aware test skipping

The provider's acceptance test suite uses `requireMinVersion(t, "vX.Y.Z")` to skip tests that rely on features not present in older 3x-ui versions. This is intentional: the same test binary runs against every supported version, and feature-gated tests are skipped automatically based on the `THREEXUI_VERSION` environment variable.

Current version gates:

- **v2.9.0+**: mixed protocol inbound, WireGuard `mtu` as list, `gateway`, `dns`, sniffing `ips_excluded`/`domains_excluded`, KCP `cwnd_multiplier`/`max_sending_window`.
- **v2.9.2+**: XHTTP padding fields and the compact subscription JSON fragment/noises format.
- **v2.9.4+**: outbound `final_rule` and VLESS `reverse_tag`.
- **v3.0.0+**: CSRF-protected unsafe requests, inbound `nodeId`, multi-node surface, API token endpoint.
- **v3.0.2+**: tunnel `rewrite_address`, `rewrite_port`, and `allowed_network`; default trusted proxy CIDRs; subscription email-in-remark default.
- **v3.2.0+**: the client API surface the provider auto-detects; `mixed`/`tunnel` replace legacy `socks`/`dokodemo-door`; client `group` and panel `panel_proxy` are available.
- **v3.2.7+**: `tun` is available as an alias for the tunnel inbound.
- **v3.3.0+** (matrix floor): `subThemeDir` and `warpUpdateInterval` settings; node-sync multi-node surface.
- **v3.3.1+**: inbound `subSortIndex`, `shareAddr`/`shareAddrStrategy`; `panelProxy` renamed to `panelOutbound` (outbound egress bridge).
- **v3.4.0+**: SMTP notifications; expanded Telegram and subscription settings.
- **v3.4.1+**: Incy client routing injection in subscription output.
- **v3.4.2+**: WireGuard multi-client fields, LDAP TLS verification control, and Xray Observatory/BurstObservatory.
- **v3.5.0+**: host groups, MTProto per-client FakeTLS fields, Xray environment variables, outbound target strategy, and expanded balancer settings.
- **v3.6.0+**: inbound `traffic_reset_day`, SMTP `From` header, subscription format auto-detection, and node `api_token_wo`.
- **v3.7.0+**: inbound `disable_flow`, per-client calendar renewals (`reset_day`, `reset_max`) and traffic reset cycle (`traffic_reset`, `traffic_reset_day`), panel `ip_limit_allowlist`, and subscription `sub_json_observatory`.

Tests without `requireMinVersion` run on all supported versions (v3.3.0+).

## Selecting a 3x-ui version

The provider communicates with whatever 3x-ui version is running on your host. To pin the panel version in Docker:

```bash
# Set the 3x-ui image tag
export THREEXUI_VERSION=v3.8.5

# Start the container
docker compose up -d
```

In `docker-compose.yaml`, the image tag is parameterized via `${THREEXUI_VERSION:-v3.8.5}`, so omitting the variable defaults to the latest tested version.

For the Terraform provider itself, use the latest release from the [Terraform Registry](https://registry.terraform.io/providers/batonogov/threexui). The single provider binary supports all 3x-ui versions listed in the compatibility table above.

## Related

- [Backup-as-code](backup-as-code.md) — version your panel state alongside your infrastructure.
- [Server migration](server-migration.md) — migrate between servers running the same or different 3x-ui versions.
- [Bulk client onboarding](bulk-clients.md) — add many clients at once, compatible with all supported versions.
