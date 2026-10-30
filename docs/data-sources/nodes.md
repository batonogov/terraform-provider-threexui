---
page_title: "threexui_nodes Data Source - 3x-ui"
subcategory: "Cluster"
description: |-
  Retrieves the cluster node tree (3x-ui multi-node surface) registered with the central 3x-ui panel.
---

# threexui_nodes (Data Source)

Retrieves the cluster node tree registered with the central 3x-ui panel as a JSON string. Corresponds to `GET /panel/api/nodes/list` (the 3x-ui multi-node/cluster surface, available since 3x-ui v3.0.2).

## Example Usage

```hcl
data "threexui_nodes" "all" {}

locals {
  nodes = jsondecode(data.threexui_nodes.all.nodes)
}

# Node names and addresses are not secrets, so wrap with `nonsensitive()` to
# expose them in plan output. Sensitivity propagates from the source attribute
# because the payload also contains apiToken/pinnedCertSha256.
output "node_names" {
  value = nonsensitive([for n in local.nodes : n.name])
}
```

> **Sensitive payload:** The `nodes` attribute is marked sensitive because each node object contains `apiToken` and `pinnedCertSha256`, which the panel returns **raw without redaction**. Outputs that reference it (or values derived from it) must declare `sensitive = true` or be wrapped in `nonsensitive(...)` for fields that are safe to expose.
>
> **Node tree & transitive nodes:** The response is the full node tree. It includes **transitive sub-nodes** surfaced from downstream panels — read-only projections with `id == 0` and `transitive == true` that are not persisted on the central panel. Filter on `transitive != true` when you only want directly-managed nodes.

## Argument Reference

This data source has no arguments.

## Attribute Reference

- `id` (String) - ID derived from the first node (`0` when the cluster has no nodes).
- `nodes` (String, Sensitive) - JSON-encoded array of cluster node objects. Use `jsondecode()` to work with the data. Each object contains managed fields (`name`, `address`, `port`, `scheme`, `basePath`, `enable`, `tlsVerifyMode`, `inboundSyncMode`, `inboundTags`, `outboundTag`, …) and observed state (`status`, `lastHeartbeat`, `latencyMs`, `xrayVersion`, `panelVersion`, `cpuPct`, `memPct`, `xrayState`, `xrayError`, counts, `unavailableMetrics` (v3.11.0+), …), plus `apiToken`/`pinnedCertSha256` (sensitive, raw).
