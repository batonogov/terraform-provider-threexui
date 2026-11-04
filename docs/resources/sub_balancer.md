---
page_title: "threexui_sub_balancer Resource - 3x-ui"
subcategory: "Panel Settings"
description: |-
  Manages a client-side balancer of the 3x-ui JSON subscription (v3.7.0+).
---

# threexui_sub_balancer (Resource)

Manages a client-side balancer of the JSON subscription (**Settings → Subscription → Balancers** in the panel, `/panel/api/sub-balancers`). Each enabled balancer adds one extra profile to the JSON subscription: its outbounds are the proxy outbounds of the selected inbounds, and its `routing.balancers` entry picks between them on the client. `leastPing` and `leastLoad` profiles also carry a `burstObservatory` (probe settings: [`threexui_panel_subscription.sub_json_observatory`](panel_subscription.md)) and fall back to the first member when every probe fails.

This is not [`threexui_xray_balancers`](xray_balancers.md), which manages the server-side `routing.balancers` of the panel's Xray template.

~> **Note:** Requires 3x-ui **v3.7.0+**; `member_weights` requires **v3.8.0+**. On older panels every operation fails with an error naming the required version. On v3.7.x the panel silently discards `member_weights`, so the provider reports an error instead of applying the configuration without them: a create is rolled back, an update keeps the other changes and records what the panel stored.

Balancer profiles appear only in the JSON subscription (`threexui_panel_subscription.sub_json_enable = true`), and only for subscribers that have at least one of the selected inbounds. The panel reads the balancers on every subscription request, so changes take effect without a panel restart.

## Example Usage

```hcl
resource "threexui_sub_balancer" "auto" {
  remark      = "Auto · least load"
  strategy    = "leastLoad"
  inbound_ids = [threexui_inbound.de.id, threexui_inbound.nl.id]

  # Lower weight is picked more often; members without a weight count as 1.
  member_weights = {
    (threexui_inbound.nl.id) = 2
  }
}
```

## Argument Reference

- `remark` (Required, String) - Name of the profile shown to subscribers. Must be non-empty, at most 256 bytes and must not start or end with whitespace (the panel trims it, so an untrimmed value could never match the configuration).
- `inbound_ids` (Required, Set of Number) - IDs of the inbounds whose proxy outbounds become balancer members. At least one. A set: the members follow the subscription order of their inbounds (`sub_sort_index`, then id), not the order written in the configuration.
- `strategy` (Optional, String) - Balancing strategy: `leastLoad`, `leastPing`, `random` or `roundRobin`. Defaults to `random`.
- `member_weights` (Optional, Map of Number) - leastLoad cost per member, keyed by inbound id (map keys are strings, so `(threexui_inbound.nl.id) = 2` or `"3" = 2`). A lower weight is picked more often; members without a weight count as 1. Only valid with `strategy = "leastLoad"`; every key must be one of `inbound_ids`, without sign or leading zeros, and every value a positive float32. Omitting the attribute clears the weights on the panel. Requires 3x-ui v3.8.0+.
- `sort_order` (Optional, Number) - Position of the profile in the subscription, interleaved with the inbounds' `sub_sort_index`; on a tie the balancer comes after the inbound. Defaults to 1; must be at least 1 (the panel would raise a lower value to 1).
- `enabled` (Optional, Boolean) - Whether the profile is added to the subscription. Defaults to `true`.

The panel's own validation is repeated at plan time: a value the panel would reject, or silently rewrite (an untrimmed remark, `sort_order = 0`, a weight for an inbound outside `inbound_ids`), fails `terraform plan` instead of the apply. Checks that depend on values unknown at plan time (for example the ids of inbounds created in the same apply) run when Terraform validates the configuration again during the apply, still before anything is written.

## Attribute Reference

- `id` (String) - Numeric ID of the balancer.

## Deleted inbounds

When an inbound is deleted, the panel removes its id from every balancer but keeps its weight, and the next refresh shows both: while the configuration still lists the inbound, the plan puts its id back; once the configuration drops it, the plan removes a leftover weight, if there is one. A balancer left without members stays on the panel but adds nothing to the subscription.

## Import

Import is supported by numeric id:

```bash
terraform import threexui_sub_balancer.auto <id>
```
