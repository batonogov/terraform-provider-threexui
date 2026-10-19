---
page_title: "threexui_xray_dns Resource - 3x-ui"
subcategory: "Xray Settings"
description: |-
  Manages Xray DNS configuration in the 3x-ui panel.
---

# threexui_xray_dns (Resource)

Manages the DNS section of the Xray template configuration. Uses a **set path** strategy -- the provided configuration completely replaces the `dns` key in the Xray template.

This is a singleton resource. Deleting this resource only removes it from Terraform state; it does not reset the DNS configuration.

-> **Note:** A server block with only `address` is serialized as a plain string in the Xray JSON. A server with additional fields (port, domains, etc.) is serialized as an object.

## Example Usage

```hcl
resource "threexui_xray_dns" "config" {
  server {
    address = "https+local://1.1.1.1/dns-query"
  }

  server {
    address        = "localhost"
    port           = 53
    domains        = ["geosite:cn"]
    expect_ips     = ["geoip:cn"]
    skip_fallback  = false
    query_strategy = "UseIP"
  }

  hosts = {
    "example.com" = "1.2.3.4"
  }

  query_strategy     = "UseIP"
  tag                = "dns-out"
  disable_cache      = false
  disable_fallback   = false
  client_ip          = ""
}
```

## Argument Reference

### Top-level attributes

- `script` (String, Optional) - Path to a Lua script on the panel host (3x-ui v3.10.0+, Xray-core v26.10.10+). The file must already exist and be readable by Xray; the provider does not upload it. Set `script = ""` to disable it; omission preserves the previous value.

- `query_strategy` (String, Optional) - DNS query strategy (e.g. `UseIP`, `UseIPv4`, `UseIPv6`).
- `tag` (String, Optional) - DNS outbound tag.
- `disable_cache` (Bool, Optional) - Disable DNS cache.
- `disable_fallback` (Bool, Optional) - Disable DNS fallback.
- `disable_fallback_if_match` (Bool, Optional) - Disable fallback if match found.
- `client_ip` (String, Optional) - Client IP for EDNS.
- `enable_parallel_query` (Bool, Optional) - Enable parallel DNS query across all configured servers.
- `use_system_hosts` (Bool, Optional) - Use system hosts file for DNS resolution.
- `hosts` (Map of String, Optional) - Static DNS host mappings.

### server (Block, Optional, List)

- `id` (String, Optional) - Resolver identifier used by Lua scripts (3x-ui v3.10.0+). An ID forces object encoding even when `address` is the only other field. Include it in each configured server that needs it: server blocks are authoritative on updates.

- `address` (String, Required) - DNS server address (e.g. `8.8.8.8`, `https+local://1.1.1.1/dns-query`, `localhost`).
- `port` (Int, Optional) - DNS server port.
- `domains` (List of String, Optional) - Domains to route to this server.
- `expect_ips` (List of String, Optional) - Expected IP ranges for validation.
- `unexpected_ips` (List of String, Optional) - Unexpected IP ranges to filter.
- `skip_fallback` (Bool, Optional) - Skip this server during fallback.
- `query_strategy` (String, Optional) - Per-server query strategy.
- `disable_cache` (Bool, Optional) - Disable cache for this server.
- `final_query` (Bool, Optional) - Mark as final query server.

## Attribute Reference

- `id` - The resource identifier (`xray_dns`).

## Import

```shell
terraform import threexui_xray_dns.config xray_dns
```
