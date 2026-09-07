---
page_title: "threexui_xray_outbounds Resource - 3x-ui"
subcategory: "Xray Settings"
description: |-
  Manages Xray outbounds configuration in the 3x-ui panel.
---

# threexui_xray_outbounds (Resource)

Manages the outbounds section of the Xray template configuration. Uses a **set path** strategy -- the provided configuration completely replaces the `outbounds` key in the Xray template.

This is a singleton resource. Deleting this resource only removes it from Terraform state; it does not reset the outbounds configuration.

## Example Usage

```hcl
resource "threexui_xray_outbounds" "config" {
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
    tag      = "proxy-wg"
    protocol = "wireguard"

    wireguard_settings {
      secret_key      = "your-secret-key"
      address         = ["10.0.0.2/32"]
      mtu             = 1420
      workers         = 2
      domain_strategy = "ForceIPv6v4"
      reserved        = [1, 2, 3]
      no_kernel_tun   = false

      peer {
        public_key  = "peer-public-key"
        endpoint    = "engage.cloudflareclient.com:2408"
        allowed_ips = ["0.0.0.0/0", "::/0"]
        keep_alive  = 30
      }
    }

    mux {
      enabled     = false
      concurrency = 8
    }
  }
}
```

## Argument Reference

### outbound (Block, Optional, List)

- `tag` (String, Optional) - Outbound tag name.
- `protocol` (String, Required) - Protocol type (`freedom`, `blackhole`, `dns`, `vmess`, `vless`, `trojan`, `shadowsocks`, `socks`, `http`, `wireguard`, `hysteria`, `hysteria2`).
- `send_through` (String, Optional) - Source IP address to bind.
- `target_strategy` (String, Optional) - Domain strategy for the outbound's destination (3x-ui v3.5.0+, xray-core v26.7.11+). One of: `AsIs`, `UseIP`, `UseIPv4`, `UseIPv6`, `UseIPv6v4`, `UseIPv4v6`, `ForceIPv6v4`. Empty/`AsIs` means xray resolves the destination as-is (the key is omitted on the wire when empty). Older xray cores silently ignore the unknown key; `freedom_settings.domain_strategy` is a separate, pre-existing field.

#### mux (Block, Optional, Max: 1)

- `enabled` (Bool, Optional) - Enable mux.
- `concurrency` (Int, Optional) - Number of concurrent connections.
- `xudp_concurrency` (Int, Optional) - XUDP concurrency.
- `xudp_proxy_udp443` (String, Optional) - XUDP proxy for UDP 443.

#### stream_settings (Block, Optional, Max: 1)

Transport settings for the outbound — the outgoing counterpart of an inbound's
`stream_settings`. Any protocol may carry one; vless/trojan outbounds typically
need it (with `security = "tls"` + `tls_settings`) to reach a TLS-protected
server. An omitted block is persisted from state on unrelated updates, and is
**not** written to the outbounds configuration when absent.

- `network` (String, Optional) - Transport network (`tcp`, `ws`, `grpc`, `httpupgrade`, `xhttp`, `kcp`, `hysteria`).
- `security` (String, Optional) - Security type (`none`, `tls`, `reality`).
- `tcp_settings` (Block, Optional, Max: 1) - TCP transport settings.
  - `accept_proxy_protocol` (Bool, Optional)
  - `header_type` (String, Optional)
- `ws_settings` (Block, Optional, Max: 1) - WebSocket settings.
  - `path` (String, Optional)
  - `headers` (Map of String, Optional)
- `grpc_settings` (Block, Optional, Max: 1) - gRPC settings.
  - `service_name` (String, Optional)
  - `multi_mode` (Bool, Optional)
  - `idle_timeout` (Int, Optional)
  - `health_check_timeout` (Int, Optional)
  - `permit_without_stream` (Bool, Optional)
  - `initial_windows_size` (Int, Optional)
- `httpupgrade_settings` (Block, Optional, Max: 1) - HTTP Upgrade settings.
  - `path` (String, Optional)
  - `host` (String, Optional)
- `xhttp_settings` (Block, Optional, Max: 1) - XHTTP settings.
  - `path` (String, Optional)
  - `mode` (String, Optional)
  - `no_sse_header` (Bool, Optional)
  - `keep_alive_interval` (Int, Optional)
  - `x_padding_bytes` (String, Optional) - xPadding bytes range (e.g. `100-1000`).
  - `x_padding_obfs_mode` (Bool, Optional)
  - `x_padding_key` (String, Optional)
  - `x_padding_header` (String, Optional)
  - `x_padding_placement` (String, Optional) - xPadding placement (e.g. `header`, `body`).
  - `x_padding_method` (String, Optional)
- `kcp_settings` (Block, Optional, Max: 1) - mKCP settings.
  - `mtu` (Int, Optional)
  - `tti` (Int, Optional)
  - `uplink_capacity` (Int, Optional)
  - `downlink_capacity` (Int, Optional)
  - `cwnd_multiplier` (Int, Optional)
  - `max_sending_window` (Int, Optional)
  - `header_type` (String, Optional)
- `hysteria_settings` (Block, Optional, Max: 1) - Hysteria transport settings.
  - `protocol` (String, Optional)
  - `version` (Int, Optional)
  - `auth` (String, Optional)
  - `udp_idle_timeout` (Int, Optional)
- `reality_settings` (Block, Optional, Max: 1) - Reality settings.
  - `show` (Bool, Optional)
  - `xver` (Int, Optional)
  - `target` (String, Optional)
  - `server_names` (List of String, Optional)
  - `private_key` (String, Optional, Sensitive)
  - `min_client_ver` (String, Optional)
  - `max_client_ver` (String, Optional)
  - `max_timediff` (Int, Optional)
  - `short_ids` (List of String, Optional, Sensitive)
  - `mldsa65_seed` (String, Optional, Sensitive)
  - `settings` (Attribute, Optional) - Reality inner settings (client-side).
    - `public_key` (String, Optional)
    - `fingerprint` (String, Optional)
    - `server_name` (String, Optional)
    - `spider_x` (String, Optional)
    - `mldsa65_verify` (String, Optional)
- `tls_settings` (Block, Optional, Max: 1) - TLS client settings (used when `security = "tls"`).
  - `server_name` (String, Optional) - Server name (SNI) for the TLS handshake.
  - `fingerprint` (String, Optional) - Client fingerprint (e.g. `chrome`, `firefox`).
  - `allow_insecure` (Bool, Optional) - Whether to allow insecure TLS connections.
  - `alpn` (List of String, Optional) - ALPN list (e.g. `["h2", "http/1.1"]`).
  - `min_version` (String, Optional) - Minimum TLS version (e.g. `1.2`).
  - `max_version` (String, Optional) - Maximum TLS version (e.g. `1.3`).
  - `cipher` (String, Optional) - TLS cipher suite.
- `sockopt` (Block, Optional, Max: 1) - Socket options.
  - `mark` (Int, Optional)
  - `tcp_keep_alive_interval` (Int, Optional)
  - `tcp_no_delay` (Bool, Optional)
  - `tfo_enable` (Bool, Optional)
  - `tproxy` (String, Optional)
  - `domain_strategy` (String, Optional)

### Per-protocol settings

Each outbound should have exactly one `*_settings` block matching its `protocol`.

#### freedom_settings (Block, Optional, Max: 1)

- `domain_strategy` (String, Optional) - Domain strategy (e.g. `AsIs`, `UseIP`).
- `redirect` (String, Optional) - Redirect address.
- `ips_blocked` (List of String, Optional) - Deprecated legacy list of IPs/CIDRs to block (e.g. `geoip:cn`). Use `final_rule` on 3x-ui v2.9.4+.

##### final_rule (Block, Optional, List)

- `action` (String, Optional) - Rule action, for example `block`.
- `network` (String, Optional) - Network selector, for example `tcp`, `udp`, or `tcp,udp`.
- `port` (String, Optional) - Port or port range.
- `ip` (List of String, Optional) - IP/CIDR/geosite entries.
- `block_delay` (String, Optional) - Block delay value stored as `blockDelay` in 3x-ui.

##### fragment (Block, Optional, Max: 1)

- `packets` (String, Optional) - Packet fragmentation mode.
- `length` (String, Optional) - Fragment length range.
- `interval` (String, Optional) - Fragment interval range.

##### noises (Block, Optional, List)

- `type` (String, Optional) - Noise type.
- `packet` (String, Optional) - Noise packet content.
- `delay` (String, Optional) - Noise delay.

#### blackhole_settings (Block, Optional, Max: 1)

- `response_type` (String, Optional) - Response type (`none` or `http`).

#### dns_settings (Block, Optional, Max: 1)

- `network` (String, Optional) - Network type.
- `address` (String, Optional) - DNS server address.
- `port` (Int, Optional) - DNS server port.
- `non_ip_query` (String, Optional) - Non-IP query handling.
- `block_types` (List of Int, Optional) - DNS record types to block.

#### vmess_settings (Block, Optional, Max: 1)

- `address` (String, Optional) - Server address.
- `port` (Int, Optional) - Server port.
- `id` (String, Optional, Sensitive) - VMess user ID (UUID credential).
- `security` (String, Optional) - Encryption method.

#### vless_settings (Block, Optional, Max: 1)

- `address` (String, Optional) - Server address.
- `port` (Int, Optional) - Server port.
- `id` (String, Optional, Sensitive) - VLESS user ID (UUID credential).
- `flow` (String, Optional) - Flow control (e.g. `xtls-rprx-vision`).
- `encryption` (String, Optional) - Encryption method.
- `reverse_tag` (String, Optional) - VLESS reverse tag. Stored in 3x-ui as `reverse.tag` and available on 3x-ui v2.9.4+.

#### trojan_settings (Block, Optional, Max: 1)

- `address` (String, Optional) - Server address.
- `port` (Int, Optional) - Server port.
- `password` (String, Optional, Sensitive) - Trojan password.

#### shadowsocks_settings (Block, Optional, Max: 1)

- `address` (String, Optional) - Server address.
- `port` (Int, Optional) - Server port.
- `password` (String, Optional, Sensitive) - Shadowsocks password.
- `method` (String, Optional) - Encryption method (e.g. `2022-blake3-aes-128-gcm`).
- `uot` (Bool, Optional) - Enable UDP over TCP.
- `uot_version` (Int, Optional) - UoT version.

#### socks_settings (Block, Optional, Max: 1)

- `address` (String, Optional) - SOCKS proxy address.
- `port` (Int, Optional) - SOCKS proxy port.
- `user` (String, Optional) - Username.
- `pass` (String, Optional, Sensitive) - Password.

#### http_settings (Block, Optional, Max: 1)

- `address` (String, Optional) - HTTP proxy address.
- `port` (Int, Optional) - HTTP proxy port.
- `user` (String, Optional) - Username.
- `pass` (String, Optional, Sensitive) - Password.

#### wireguard_settings (Block, Optional, Max: 1)

- `mtu` (Int, Optional) - MTU size.
- `secret_key` (String, Optional, Sensitive) - WireGuard private key.
- `address` (List of String, Optional) - Local addresses (e.g. `["10.0.0.2/32"]`).
- `workers` (Int, Optional) - Number of worker threads.
- `domain_strategy` (String, Optional) - Domain strategy.
- `reserved` (List of Int, Optional) - Reserved bytes.
- `no_kernel_tun` (Bool, Optional) - Disable kernel TUN.

##### peer (Block, Optional, List)

- `public_key` (String, Optional) - Peer public key.
- `pre_shared_key` (String, Optional, Sensitive) - Pre-shared key.
- `allowed_ips` (List of String, Optional) - Allowed IP ranges.
- `endpoint` (String, Optional) - Peer endpoint (host:port).
- `keep_alive` (Int, Optional) - Keep-alive interval in seconds.

#### hysteria_settings (Block, Optional, Max: 1)

- `address` (String, Optional) - Server address.
- `port` (Int, Optional) - Server port.
- `version` (Int, Optional) - Hysteria version.

## Attribute Reference

- `id` - The resource identifier (`xray_outbounds`).

## Import

```shell
terraform import threexui_xray_outbounds.config xray_outbounds
```
