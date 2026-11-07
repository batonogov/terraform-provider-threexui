---
page_title: "threexui_node Resource - 3x-ui"
subcategory: "Cluster"
description: |-
  Manages a remote 3x-ui panel registered as a cluster node (3x-ui multi-node surface).
---

# threexui_node (Resource)

Manages a remote 3x-ui panel registered as a cluster node on the central panel. Wraps the 3x-ui multi-node API (`/panel/api/nodes/*`), available since 3x-ui v3.0.2.

> **Reachability constraint:** The central panel probes the node for reachability (`ensureReachable`) during create/update before persisting it. The node's web API **must be reachable from the central panel at apply time**, otherwise the create/update fails. There is no way to bypass this from the provider — it is inherent to the 3x-ui server flow.
>
> **Xray restart on `outbound_tag` change:** changing `outbound_tag` causes the 3x-ui server to restart the Xray core itself (`controller/node.go` update handler). The provider does not issue a separate restart — it relies on the server-side behaviour.

## Example Usage

```hcl
resource "threexui_node" "fra1" {
  name        = "de-fra-1"
  remark      = "Frankfurt edge"
  scheme      = "https"
  address     = "node1.example.com"
  port        = 2053
  base_path   = "/"
  api_token   = var.fra1_api_token
  enable      = true

  tls_verify_mode = "verify"
}
```

Import an existing node by its numeric id:

```bash
terraform import threexui_node.fra1 7
```

## Argument Reference

- `name` (Required, String) - Unique node name (upstream `uniqueIndex`). Changing this forces a new resource.
- `address` (Required, String) - Node host. Changing this forces a new resource.
- `port` (Required, Number) - Node web API port, 1-65535.
- `remark` (Optional, String) - Free-form note.
- `scheme` (Optional, String) - `http` or `https`. Defaults to `https`.
- `base_path` (Optional, String) - Node web API base path. Defaults to `/`.
- `api_token` (Optional, String, Sensitive) - Bearer API token the central panel uses to authenticate to the node. Required unless `tls_verify_mode` is `mtls`. Since 3x-ui v3.6.0 the panel no longer returns this value on read (write-only, 3x-ui #5613); the provider preserves the configured value in state, and after `terraform import` on v3.6.0+ you must set it manually because the panel cannot echo it back. Prefer the write-only `api_token_wo` on TF 1.11+.
- `api_token_wo` (Optional, String, Write-Only) - Write-only node API token (Terraform 1.11+ / OpenTofu 1.11+). Pair with `api_token_wo_version` to rotate.
- `api_token_wo_version` (Optional, Number) - Increment to rotate `api_token_wo`. Each change sends the current `_wo` value to the panel; requires `api_token_wo`.
- `enable` (Optional, Bool) - Whether the node is enabled. Defaults to `true`.
- `allow_private_address` (Optional, Bool) - Allow the node address to resolve to a private IP. Defaults to `false`.
- `tls_verify_mode` (Optional, String) - `verify`, `skip`, `pin`, or `mtls`. Defaults to `verify`.
- `pinned_cert_sha256` (Optional, String, Sensitive) - Pinned certificate fingerprint, required when `tls_verify_mode` is `pin`. Prefer the write-only `pinned_cert_sha256_wo` on TF 1.11+.
- `pinned_cert_sha256_wo` (Optional, String, Write-Only) - Write-only pinned certificate fingerprint (Terraform 1.11+ / OpenTofu 1.11+). Pair with `pinned_cert_sha256_wo_version` to rotate.
- `pinned_cert_sha256_wo_version` (Optional, Number) - Increment to rotate `pinned_cert_sha256_wo`; requires `pinned_cert_sha256_wo`.
- `inbound_sync_mode` (Optional, String) - `all` or `selected`. Defaults to `all`.
- `inbound_tags` (Optional, List of String) - Inbound tags to sync when `inbound_sync_mode` is `selected`.
- `outbound_tag` (Optional, String) - Xray outbound/balancer tag bridging this node.

## Attribute Reference

In addition to the arguments above, the following observed-state attributes are read-only (`Computed`):

- `id` (String) - Numeric node id (as a string). Import key.
- `guid` (String) - Remote panel stable GUID.
- `status` (String) - `online`, `offline`, or `unknown`.
- `last_heartbeat` (Number) - Unix seconds of the last successful heartbeat (0 = never).
- `latency_ms` (Number) - Last heartbeat latency in ms.
- `xray_version` (String) - Xray version reported by the node.
- `panel_version` (String) - 3x-ui panel version reported by the node.
- `cpu_pct` / `mem_pct` (Number) - Node CPU / memory usage percent.
- `uptime_secs` / `net_up` / `net_down` (Number) - Uptime (s) and network byte counters.
- `unavailable_metrics` (List of String) - Container metrics unavailable on the node (3x-ui v3.11.0+). A metric absent from this list can legitimately be zero.
- `last_error` (String) - Last heartbeat error message.
- `xray_state` / `xray_error` (String) - Xray core state / error from the node.
- `config_dirty` (Bool) / `config_dirty_at` (Number) - Whether the node config differs from the central panel.
- `inbound_count` / `client_count` / `online_count` / `active_count` / `disabled_count` / `depleted_count` (Number) - Client/inbound counts.
- `parent_guid` (String) / `transitive` (Bool) - Node-tree attribution (read-only transitive sub-nodes have `transitive = true`).
- `created_at` / `updated_at` (Number) - Unix millis.

The central panel rewrites most of these on every heartbeat probe, and every update marks the node config dirty, so an in-place update plans them as `(known after apply)`. Only `id`, `guid`, `created_at`, `parent_guid` and `transitive` keep their values in the plan.

## Delete behaviour

`terraform destroy` unregisters the node from the panel (`POST /panel/api/nodes/del/:id`). 3x-ui **refuses** to delete a node that still owns inbounds (DB-002): if any inbound references this node, the delete fails with an error mentioning the attached inbounds. Detach or delete those inbounds first, then re-run `terraform destroy`. If the node was already removed out-of-band, delete succeeds (treated as already gone).

## Recovery after an update error

If an update succeeds on the panel but the follow-up read fails, the provider returns an explicit error and retains the node identity and previous observed values. Known secret changes accepted by the successful update, including write-only rotation versions, are retained so a later refresh does not require repeating the rotation. Run Terraform again to refresh and reconcile state. If the panel explicitly reports that the node disappeared, the next refresh removes it from state and plans recreation.

## Import

Import is supported by numeric id:

```bash
terraform import threexui_node.NAME <id>
```

> **Scope note:** Set the write-only `_wo` value and bump its `_wo_version` to rotate (Terraform 1.11+ / OpenTofu 1.11+). The `_wo` attributes themselves are not stored in state, but this provider retains the resolved credentials in the corresponding plain `Sensitive` attributes to preserve them across panel reads and recover acknowledged updates. Sensitive values are hidden in normal CLI output, not encrypted or omitted from Terraform state; protect your state storage accordingly. Since 3x-ui v3.6.0 the panel does not echo `api_token` on read. ModifyPlan marks the plain attribute unknown on a version change so the new value can be recorded after apply. On Terraform >= 1.11, configuring the plain `Sensitive` attribute emits a `PreferWriteOnlyAttribute` warning.
