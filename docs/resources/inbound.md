---
page_title: "threexui_inbound Resource - 3x-ui"
subcategory: "Inbound"
description: |-
  Manages an inbound in the 3x-ui panel.
---

# threexui_inbound (Resource)

Manages an inbound proxy in the 3x-ui panel. Supports protocols: vless, vmess, trojan, shadowsocks, http, mixed, wireguard, amneziawg, tunnel, tun, hysteria, and mtproto. For older panels, the provider also preserves legacy `socks`, `dokodemo-door`, and `hysteria2` values from imported state; on 3x-ui v3.2.0+ use `mixed`, `tunnel`, and `hysteria` with `version = 2` for new configurations. TUN requires v3.2.7+, MTProto requires v3.3.0+, and AmneziaWG requires v3.7.0+.

## Example Usage

### VLESS with Reality

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
    tcp_settings {
      accept_proxy_protocol = false
      header_type           = "none"
    }
  }

  sniffing {
    enabled       = true
    dest_override = ["http", "tls", "quic", "fakedns"]
  }
}
```

### VMess

```hcl
resource "threexui_inbound" "vmess" {
  port     = 10086
  protocol = "vmess"
  enable   = true
  remark   = "VMess"

  stream_settings {
    network  = "tcp"
    security = "none"
  }
}
```

### Shadowsocks

```hcl
resource "threexui_inbound" "ss" {
  port     = 8388
  protocol = "shadowsocks"
  enable   = true
  remark   = "Shadowsocks"

  shadowsocks_settings {
    method   = "chacha20-ietf-poly1305"
    password = "my-password"
    network  = "tcp,udp"
  }
}
```

### VLESS with WebSocket

```hcl
resource "threexui_inbound" "ws" {
  port     = 8443
  protocol = "vless"
  enable   = true
  remark   = "VLESS WS"

  vless_settings {
    decryption = "none"
  }

  stream_settings {
    network  = "ws"
    security = "none"
    ws_settings {
      path = "/ws"
    }
  }
}
```

### HTTP Proxy with Authentication

```hcl
resource "threexui_inbound" "http" {
  port     = 8080
  protocol = "http"
  enable   = true
  remark   = "HTTP Proxy"

  http_settings {
    auth              = "password"
    allow_transparent = false
    account {
      user = "myuser"
      pass = "mypass"
    }
  }
}
```

### WireGuard

```hcl
resource "threexui_inbound" "wg" {
  port     = 51820
  protocol = "wireguard"
  enable   = true
  remark   = "WireGuard"

  wireguard_settings {
    mtu = [1420, 1280]
    peer {
      public_key  = "BASE64_PUBLIC_KEY"
      allowed_ips = ["10.0.0.2/32"]
    }
  }
}
```

### AmneziaWG

AmneziaWG (3x-ui v3.7.0+) is WireGuard with DPI-resistant obfuscation, running in-process on an embedded userspace device — no kernel module is needed. Peers are declared inline as `clients` blocks; do **not** attach `threexui_inbound_client` resources to an AmneziaWG inbound.

Every attribute under `server` is optional, but the **block itself is required**: 3x-ui regenerates the whole server block — a fresh keypair included — whenever it saves an inbound whose settings carry no `server` object, on update as well as on create. Without the block in the configuration there would be nothing for the provider to send back, so an unrelated change such as a new remark would rotate the server keys and invalidate every peer configuration already distributed. `server {}` is enough: the panel fills the attributes in on create and the provider replays them on every later apply.

```hcl
resource "threexui_inbound" "awg" {
  port     = 51821
  protocol = "amneziawg"
  enable   = true
  remark   = "AmneziaWG"

  amneziawg_settings {
    server {
      subnet_ip   = "10.9.1.0"
      subnet_cidr = 24
      primary_dns = "1.1.1.1"
    }

    clients {
      email       = "phone@example.com"
      enable      = true
      public_key  = "BASE64_PEER_PUBLIC_KEY"
      allowed_ips = ["10.9.1.2/32"]
      # Publish two ports from the peer through the server.
      forwarded_ports = "80,443"
    }
  }
}
```

### Hysteria

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
```

### MTProto

```hcl
resource "threexui_inbound" "mtproto" {
  port     = 443
  protocol = "mtproto"
  enable   = true
  remark   = "MTProto FakeTLS"

  mtproto_settings {
    fake_tls_domain = "www.cloudflare.com"
    prefer_ip       = "prefer-ipv4"
  }
}
```

## Argument Reference

### Top-level

- `port` (Required, Number) - Port number for the inbound.
- `protocol` (Required, String) - Protocol type (`vless`, `vmess`, `trojan`, `shadowsocks`, `http`, `mixed`, `wireguard`, `amneziawg`, `tunnel`, `tun`, `hysteria`, `mtproto`). Legacy `socks` and `dokodemo-door` are available only on panels before 3x-ui v3.2.0; use `mixed` and `tunnel` on v3.2.0+. `tun` is a tunnel alias available on v3.2.7+, `mtproto` is available on v3.3.0+, and `amneziawg` on v3.7.0+.
- `enable` (Optional, Boolean) - Whether the inbound is enabled. Default is `true`.
- `remark` (Optional, String) - A label/name for the inbound.
- `listen` (Optional, String) - Listen address.
- `up` (Optional, Number) - Upload traffic counter in bytes.
- `down` (Optional, Number) - Download traffic counter in bytes.
- `total` (Optional, Number) - Total traffic limit in bytes.
- `expiry_time` (Optional, Number) - Expiry time as Unix timestamp in milliseconds.
- `traffic_reset` (Optional, String) - Traffic reset period. Default is `never`.
- `traffic_reset_day` (Optional, Number) - Day of month (1-31) for monthly traffic resets. Only effective when `traffic_reset = "monthly"`. Added in 3x-ui v3.6.0; older panels report `0` (unsupported). `0` is rejected at plan time: the panel clamps any value below 1 up to 1, so a configured `0` could never round-trip.
- `disable_flow` (Optional, Boolean) - Opt this inbound out of the panel's automatic XTLS Vision flow assignment. Added in 3x-ui v3.7.0; older panels report `false` (unsupported). **The panel blanks `flow` on every client of the inbound while this is `true`**, so do not combine it with a `threexui_inbound_client` that sets `flow` — see the note below.
- `node_id` (Optional, Number) - 3x-ui v3 node ID for multi-node deployments. Leave unset for the local panel. Changing this value recreates the inbound because 3x-ui v3 does not support moving an existing inbound between nodes.
- `restart_xray` (Optional, Boolean) - Restart Xray core after create, update, or delete operations. Default is `false`.
- `sub_sort_index` (Optional, Number) - 1-based sort order of this inbound's links in subscription output (lower first; ties by id). Added in 3x-ui v3.3.1; ignored by older panels.
- `share_addr` (Optional, String) - Share address used in generated subscription links when share_addr_strategy is custom. Added in 3x-ui v3.3.1; ignored by older panels.
- `share_addr_strategy` (Optional, String) - Strategy for the share address in subscription links: `node` (inbound listen/node address), `listen`, or `custom` (uses `share_addr`). Added in 3x-ui v3.3.1; ignored by older panels.

### Per-protocol Settings Blocks

Use the block matching your `protocol`. Only one should be specified.

#### `vless_settings`

- `decryption` (Optional, String) - Decryption method (usually `none`).
- `encryption` (Optional, String) - Encryption method.
- `selected_auth` (Optional, String) - Selected authentication type.
- `fallback` (Optional, Block List) - Fallback destinations.
  - `name` (Optional, String)
  - `alpn` (Optional, String)
  - `path` (Optional, String)
  - `dest` (Optional, String)
  - `xver` (Optional, Number)

#### `trojan_settings`

- `fallback` (Optional, Block List) - Same structure as vless fallback.

#### `shadowsocks_settings`

- `method` (Optional, String) - Encryption method (e.g. `chacha20-ietf-poly1305`, `2022-blake3-aes-256-gcm`). On 3x-ui v2.9.3+ the legacy `aes-128-gcm`/`aes-256-gcm` ciphers were dropped from the xray user switch and silently route through Shadowsocks-2022; pick a chacha20 variant or a `2022-blake3-*` method to stay compatible across the matrix.
- `password` (Optional, String, Sensitive) - Password.
- `network` (Optional, String) - Network type (e.g. `tcp,udp`).
- `iv_check` (Optional, Boolean) - Enable IV check.

#### `http_settings`

- `auth` (Optional, String) - Auth type (e.g. `password`).
- `allow_transparent` (Optional, Boolean) - Allow transparent proxy.
- `account` (Optional, Block List) - Authentication accounts.
  - `user` (Optional, String)
  - `pass` (Optional, String)

#### `socks_settings`

- `auth` (Optional, String) - Auth type.
- `udp` (Optional, Boolean) - Enable UDP.
- `ip` (Optional, String) - IP address.
- `account` (Optional, Block List) - Same structure as http account.

#### `mixed_settings`

Settings for mixed (HTTP+SOCKS) proxy protocol.

- `auth` (Optional, String) - Authentication type (e.g. `password`, `noauth`).
- `udp` (Optional, Boolean) - Enable UDP support.
- `ip` (Optional, String) - IP address for UDP.
- `account` (Optional, Block List) - Authentication accounts. Same structure as http account.

#### `wireguard_settings`

- `mtu` (Optional, List of Number) - MTU values `[IPv4, IPv6]`.
- `secret_key` (Optional, String, Sensitive) - Secret key.
- `no_kernel_tun` (Optional, Boolean) - Disable kernel TUN.
- `gateway` (Optional, List of String) - Gateway addresses.
- `dns` (Optional, List of String) - DNS server addresses.
- `peer` (Optional, Block List) - WireGuard peers.
  - `private_key` (Optional, String, Sensitive)
  - `public_key` (Optional, String)
  - `pre_shared_key` (Optional, String, Sensitive)
  - `allowed_ips` (Optional, List of String)
  - `keep_alive` (Optional, Number)
- `clients` (Optional, Block List) - WireGuard multi-client peers (3x-ui v3.4.2+). Absent/empty on older panels. Each entry is one client device the server accepts, with its own keypair and traffic limits. Use EITHER `clients` OR the legacy `peer` for an inbound, not both — the panel treats them as separate models and populating both yields undefined behavior.
  - `private_key` (Optional, String, Sensitive)
  - `public_key` (Optional, String)
  - `pre_shared_key` (Optional, String, Sensitive)
  - `allowed_ips` (Optional, List of String)
  - `keep_alive` (Optional, Number)
  - `email` (Optional, String) - the panel requires a non-empty unique email (keys traffic counters on it); set it even though it is Optional in the schema.
  - `limit_ip` (Optional, Number)
  - `total_gb` (Optional, Number) - traffic quota in bytes (the field name mirrors the 3x-ui API).
  - `expiry_time` (Optional, Number) - Unix timestamp in milliseconds.
  - `enable` (Optional, Boolean)
  - `tg_id` (Optional, Number)
  - `sub_id` (Optional, String)
  - `comment` (Optional, String)
  - `reset` (Optional, Number) - traffic-counter reset period in days (0 = no periodic reset).

#### `dokodemo_settings`

Used for both `tunnel` and `dokodemo-door` protocols.

- `address` (Optional, String) - Target address.
- `port` (Optional, Number) - Target port.
- `rewrite_address` (Optional, String) - Tunnel rewrite address on 3x-ui v3.0.2+. Mirrored with `address` for older panel compatibility.
- `rewrite_port` (Optional, Number) - Tunnel rewrite port on 3x-ui v3.0.2+. Mirrored with `port` for older panel compatibility.
- `port_map` (Optional, Map of String) - Port mapping.
- `network` (Optional, String) - Network type.
- `allowed_network` (Optional, String) - Tunnel allowed network on 3x-ui v3.0.2+. Mirrored with `network` for older panel compatibility.
- `follow_redirect` (Optional, Boolean) - Follow redirect.

#### `hysteria_settings`

- `version` (Optional, Number) - Hysteria version (1 or 2, default 2).

#### `amneziawg_settings` (Optional, Block)

Typed AmneziaWG settings, available on 3x-ui v3.7.0+. Unlike the client-carrying protocols (vmess/vless/trojan/shadowsocks/hysteria), whose clients are managed by `threexui_inbound_client`, AmneziaWG peers live in this block — the same rule as `wireguard_settings.clients`.

- `server` (Optional, Block) - Server parameters.
- `clients` (Optional, Block List) - Peers this server accepts.

##### `server` (Optional, Block)

~> **Required in practice.** The provider rejects an `amneziawg` inbound that does not declare this block, because a settings blob without a `server` object makes the panel regenerate the server keypair on every save. Declare `server {}` and leave the attributes to the panel if you do not want to manage them.

Every attribute is Optional+Computed: what the configuration omits, the panel generates on save and the provider records in state. Attributes the panel declares `omitempty` upstream reject an empty string or a zero — omit them instead, since a value the panel strips could not round-trip. The range-valued fields (`content_padding_addition` and the five timings) additionally reject spaces, because the panel stores them canonicalised: `"110 - 140"` would come back as `"110-140"` and fail the apply.

The provider creates an AmneziaWG inbound in two steps: first without this block, so the panel generates a complete randomised obfuscation set, then applying the configured fields on top. This matters because 3x-ui only generates that set when the settings carry no `server` object at all — a partial block is taken literally, and every field left out is stored as its zero value. Configuring just a subnet directly would produce `jc = 0`, blank `h1`-`h4` and no header protection: plain WireGuard under an AmneziaWG name, with nothing reporting it.

- `private_key` (Optional+Computed, String, Sensitive) - Server private key (base64). Generated when absent. Sending an empty value makes the panel generate a **new** keypair, invalidating every existing peer config.
- `public_key` (Optional+Computed, String) - Server public key (base64). The panel does not derive this from `private_key` for the server (only for clients), so set both or let it generate both.
- `subnet_ip` (Optional+Computed, String) - Tunnel subnet address. Panel default `10.8.1.0`.
- `subnet_cidr` (Optional+Computed, Number) - Tunnel prefix length, 1-32. Panel default `24`.
- `mtu` (Optional+Computed, Number) - Tunnel MTU, at least 1. Omit for the embedded device default of 1420 — `0` is rejected because the panel strips it.
- `primary_dns` (Optional+Computed, String) - Primary DNS handed to peers. Panel default `8.8.8.8`. An empty string is meaningful: it clears the entry rather than restoring the default.
- `secondary_dns` (Optional+Computed, String) - Secondary DNS. Panel default `8.8.4.4`. An empty string clears it.
- `external_interface` (Optional+Computed, String) - Host NIC used for egress NAT. Omit to auto-detect.
- `ipv6_enabled` (Optional+Computed, Boolean) - Enable the IPv6 tunnel. Requires `ipv6_subnet`.
- `ipv6_subnet` (Optional+Computed, String) - IPv6 tunnel prefix, e.g. `fd00:8:1::/64`. Required when `ipv6_enabled` is true.
- `ipv6_external_interface` (Optional+Computed, String) - Host NIC for IPv6 egress. Omit to reuse the IPv4 interface.
- `route_through_xray` (Optional+Computed, Boolean) - Vestigial upstream: the embedded relay is always on and nothing reads the flag. Round-tripped so the blob is not altered on save.
- `jc` (Optional+Computed, Number) - Junk packet count.
- `jmin` (Optional+Computed, Number) - Minimum junk packet size. Must not exceed `jmax`.
- `jmax` (Optional+Computed, Number) - Maximum junk packet size.
- `s1` (Optional+Computed, Number) - Init packet junk size. `s1 + 56` must not equal `s2`.
- `s2` (Optional+Computed, Number) - Response packet junk size.
- `s3` (Optional+Computed, Number) - Cookie reply packet junk size, 0-64.
- `s4` (Optional+Computed, Number) - Transport packet junk size, 0-32.
- `h1` (Optional+Computed, String) - Init packet magic header: an integer or a `lo-hi` range within 0-4294967295. Blank falls back to the classic WireGuard header `1` when a peer config is rendered.
- `h2` (Optional+Computed, String) - Response packet magic header. Blank falls back to `2`.
- `h3` (Optional+Computed, String) - Underload packet magic header. Blank falls back to `3`.
- `h4` (Optional+Computed, String) - Transport packet magic header. Blank falls back to `4`.
- `i1` (Optional+Computed, String) - Custom signature packet 1, e.g. `<r 64>`.
- `i2` (Optional+Computed, String) - Custom signature packet 2.
- `i3` (Optional+Computed, String) - Custom signature packet 3.
- `i4` (Optional+Computed, String) - Custom signature packet 4.
- `i5` (Optional+Computed, String) - Custom signature packet 5.
- `header_protection_key` (Optional+Computed, String, Sensitive) - Header-protection key: base64 of exactly 32 bytes. When set, all of `s1`-`s4` must be at least 12.
- `content_padding_addition` (Optional+Computed, String) - Extra content padding: an integer or a `lo-hi` range.
- `rekey_after_time` (Optional+Computed, String) - Rekey interval in seconds, integer or `lo-hi` range. Its maximum must be lower than the minimum of `reject_after_time`.
- `rekey_timeout` (Optional+Computed, String) - Rekey timeout in seconds.
- `reject_after_time` (Optional+Computed, String) - Session reject time in seconds.
- `keepalive_timeout` (Optional+Computed, String) - Keepalive timeout in seconds.
- `max_handshake_attempts` (Optional+Computed, String) - Maximum handshake attempts.
- `random_trailers` (Optional+Computed, Boolean) - Append random trailers to packets.
- `disable_cookies` (Optional+Computed, Boolean) - Disable the cookie-reply mechanism.

##### `clients` (Optional, Block List)

~> **Note:** removing a peer — whether by deleting it from this block or by destroying the whole inbound — deletes it individually, so its `email` is freed and can be reused. This is deliberate: 3x-ui rewrites `settings.clients[]` without the peer, and drops the inbound-to-client links on delete, but in both cases keeps the client row, which carries a unique index on `email`. Without that step the address would stay occupied by a client no longer visible under any inbound, and reusing it would fail with `Duplicate email: <address>`. Two cases still leave rows behind: removing the inbound **outside Terraform** (the panel UI, or the API directly), and a create that fails after the inbound was already made — Terraform records no state for it, so nothing ever destroys it. In both, delete the leftover clients in the panel before reusing their emails. `wireguard_settings.clients` is handled the same way.

- `email` (Optional+Computed, String) - Peer identifier. The panel keys traffic counters on it and requires a non-empty unique value, so set it even though the schema marks it Optional.
- `private_key` (Optional+Computed, String, Sensitive) - Peer private key. The panel stores it only to render a ready-made peer config, so it can be left out when the key is kept elsewhere; it is not generated on this path.
- `public_key` (**Required**, String) - Peer public key (base64). The panel rejects an AmneziaWG peer without one (`wireguard client requires a key`) and does not derive it from `private_key` on the inbound path — key generation lives only on the `/panel/api/clients` endpoints, which do not own these peers.
- `pre_shared_key` (Optional+Computed, String, Sensitive) - Optional pre-shared key.
- `allowed_ips` (Optional+Computed, List of String) - Peer tunnel addresses, e.g. `["10.9.1.2/32"]`. Set at least one: address allocation happens on the `/panel/api/clients` endpoints, which do not own these peers, so a peer declared here without addresses is saved unroutable. Normalised server-side — a bare address becomes `/32` and duplicates are dropped.
- `keep_alive` (Optional+Computed, Number) - Persistent keepalive in seconds, at least 1. Omit it to disable keepalive — the panel strips a zero, so `0` cannot round-trip.
- `forwarded_ports` (Optional+Computed, String) - Ports DNAT-forwarded to this peer, e.g. `80,443,8000-8100`. Expands to at most 100 ports. The panel rejects a spec colliding with the panel's own port, any enabled inbound's port, or this inbound's SOCKS relay port (65100 + inbound id); malformed tokens are dropped silently.
- `enable` (Optional+Computed, Boolean) - Whether the peer is enabled.
- `limit_ip` (Optional+Computed, Number) - Concurrent IP limit (0 = unlimited).
- `total_gb` (Optional+Computed, Number) - Traffic limit in bytes (0 = unlimited).
- `expiry_time` (Optional+Computed, Number) - Expiry timestamp in milliseconds since epoch (0 = never).
- `tg_id` (Optional+Computed, Number) - Telegram user id.
- `sub_id` (Optional+Computed, String) - Subscription id.
- `comment` (Optional+Computed, String) - Free-form comment.
- `reset` (Optional+Computed, Number) - Traffic reset interval in days (0 = never).
- `group` (Optional+Computed, String) - Logical grouping label.
- `reset_day` (Optional+Computed, Number) - Calendar day of month (1-31) the peer's traffic renews on. `0` keeps the rolling `reset` interval.
- `reset_max` (Optional+Computed, Number) - Maximum number of automatic renewals; `0` means unlimited.
- `traffic_reset` (Optional+Computed, String) - Per-peer traffic reset cycle: `never`, `hourly`, `daily`, `weekly` or `monthly`. Independent of the inbound's own cycle.
- `traffic_reset_day` (Optional+Computed, Number) - Day of month (1-31) for a monthly per-peer reset. The panel clamps anything below 1 up to 1, so `0` cannot round-trip.
- `created_at` (Optional+Computed, Number) - Creation timestamp in milliseconds since epoch, set by the panel.
- `updated_at` (Optional+Computed, Number) - Last-update timestamp in milliseconds since epoch, set by the panel.

#### `mtproto_settings` (Optional, Block)

Typed MTProto server settings available on 3x-ui v3.3.0+. On v3.5.0+, per-client FakeTLS `secret` and optional `ad_tag` values are managed by `threexui_inbound_client`; the panel heals their domain suffix from `fake_tls_domain`.

- `fake_tls_domain` (Optional+Computed, String) - FakeTLS domain. The panel default is `www.cloudflare.com`.
- `proxy_protocol_listener` (Optional+Computed, Boolean) - Enable the PROXY protocol listener.
- `prefer_ip` (Optional+Computed, String) - IP preference: `prefer-ipv6`, `prefer-ipv4`, `only-ipv6`, or `only-ipv4`.
- `debug` (Optional+Computed, Boolean) - Enable MTProto debug mode.
- `outbound_tag` (Optional+Computed, String) - Outbound tag used for routing.
- `route_through_xray` (Optional+Computed, Boolean) - Route MTProto traffic through Xray core.
- `route_xray_port` (Optional+Computed, Number) - Xray routing port.
- `public_ipv4` (Optional+Computed, String) - Public IPv4 address advertised by the service.
- `public_ipv6` (Optional+Computed, String) - Public IPv6 address advertised by the service.
- `domain_fronting` (Optional, Block List) - Domain-fronting listeners.

##### `domain_fronting` (Optional, Block List)

- `ip` (Optional+Computed, String) - Fronting IP address.
- `port` (Optional+Computed, Number) - Fronting port.
- `proxy_protocol` (Optional+Computed, Boolean) - Enable PROXY protocol for this listener.

### `stream_settings` Block

- `network` (Optional, String) - Transport network (`tcp`, `ws`, `grpc`, `httpupgrade`, `xhttp`, `kcp`, `hysteria`).
- `security` (Optional, String) - Security type (`none`, `tls`, `reality`).
- `tcp_settings` (Optional, Block) - TCP transport settings.
  - `accept_proxy_protocol` (Optional, Boolean)
  - `header_type` (Optional, String)
- `ws_settings` (Optional, Block) - WebSocket settings.
  - `path` (Optional, String)
  - `headers` (Optional, Map of String)
- `grpc_settings` (Optional, Block) - gRPC settings.
  - `service_name` (Optional, String)
  - `multi_mode` (Optional, Boolean)
  - `idle_timeout` (Optional, Number)
  - `health_check_timeout` (Optional, Number)
  - `permit_without_stream` (Optional, Boolean)
  - `initial_windows_size` (Optional, Number)
- `httpupgrade_settings` (Optional, Block) - HTTP Upgrade settings.
  - `path` (Optional, String)
  - `host` (Optional, String)
- `xhttp_settings` (Optional, Block) - XHTTP settings.
  - `path` (Optional, String)
  - `mode` (Optional, String)
  - `no_sse_header` (Optional, Boolean)
  - `keep_alive_interval` (Optional, Number)
  - `x_padding_bytes` (Optional, String) - xPadding bytes range (e.g. `100-1000`).
  - `x_padding_obfs_mode` (Optional, Boolean) - Enable xPadding obfuscation mode.
  - `x_padding_key` (Optional, String) - xPadding encryption key.
  - `x_padding_header` (Optional, String) - xPadding header name.
  - `x_padding_placement` (Optional, String) - xPadding placement (e.g. `header`, `body`).
  - `x_padding_method` (Optional, String) - xPadding method (e.g. `aes`).
- `kcp_settings` (Optional, Block) - mKCP settings.
  - `mtu` (Optional, Number)
  - `tti` (Optional, Number)
  - `uplink_capacity` (Optional, Number)
  - `downlink_capacity` (Optional, Number)
  - `cwnd_multiplier` (Optional, Number) - CWND multiplier.
  - `max_sending_window` (Optional, Number) - Maximum sending window size.
  - `header_type` (Optional, String)
- `hysteria_settings` (Optional, Block) - Hysteria transport settings.
  - `protocol` (Optional, String) - Hysteria transport protocol.
  - `version` (Optional, Number) - Hysteria version (default 2).
  - `auth` (Optional, String) - Hysteria auth string.
  - `udp_idle_timeout` (Optional, Number) - UDP idle timeout in seconds (default 60).
- `reality_settings` (Optional, Block) - Reality settings.
  - `show` (Optional, Boolean)
  - `xver` (Optional, Number)
  - `target` (Optional, String)
  - `server_names` (Optional, List of String)
  - `private_key` (Optional, String, Sensitive) - Auto-generated if not specified.
  - `min_client_ver` (Optional, String) - Minimum client Xray version the REALITY server accepts, as `major.minor.patch` (e.g. `26.3.27`). Never configuring it is **not** the same as disabling the gate: Xray 26.7.x substitutes `26.3.27` for an empty value, rejecting clients that report an older or absent version (e.g. sing-box). Set `0.0.0` to remove the lower bound — that is the canonical spelling, and `0` / `0.0` are zero-filled to the same version. One to three components, each in `0-255`; an empty string is rejected. See the note on widening a bound below.
  - `max_client_ver` (Optional, String) - Maximum client Xray version the REALITY server accepts, as `major.minor.patch`. Never configuring it means no upper bound; once set, `255.255.255` widens it back to every version.
  - `max_timediff` (Optional, Number) - Maximum allowed time difference with the client, in milliseconds. `0` disables the check, and is also how it is turned back off once set.
  - `short_ids` (Optional, List of String, Sensitive) - Auto-generated if not specified.
  - `mldsa65_seed` (Optional, String, Sensitive) - Private ML-DSA-65 seed material.
  - `settings` (Optional, Attribute) - Reality inner settings (client-side). Auto-populated if omitted.
    - `public_key` (Optional, String) - Auto-generated if not specified.
    - `fingerprint` (Optional, String)
    - `server_name` (Optional, String)
    - `spider_x` (Optional, String)
    - `mldsa65_verify` (Optional, String)

  > **Widening a REALITY version bound.** Every attribute in this block is
  > `Optional+Computed`, so an attribute removed from the configuration keeps its
  > last applied value — that is what lets an imported inbound stay driftless when
  > the configuration omits server-populated fields. Deleting `min_client_ver`
  > therefore does not restore the Xray default. Set the extreme value instead:
  > `min_client_ver = "0.0.0"` for no lower bound, `max_client_ver = "255.255.255"`
  > for no upper bound, `max_timediff = 0` for no time check.

- `tls_settings` (Optional, Block) - TLS client settings (used when `security = "tls"`).
  - `server_name` (Optional, String) - Server name (SNI) for the TLS handshake.
  - `fingerprint` (Optional, String) - Client fingerprint (e.g. `chrome`, `firefox`).
  - `allow_insecure` (Optional, Boolean) - Whether to allow insecure TLS connections.
  - `alpn` (Optional, List of String) - ALPN list (e.g. `["h2", "http/1.1"]`).
  - `min_version` (Optional, String) - Minimum TLS version (e.g. `1.2`).
  - `max_version` (Optional, String) - Maximum TLS version (e.g. `1.3`).
  - `cipher` (Optional, String) - TLS cipher suite.

- `external_proxy` (Optional, Block List) - External proxy entries.
  - `dest` (Optional, String)
  - `port` (Optional, Number)
  - `remark` (Optional, String)
  - `force_tls` (Optional, String)
- `sockopt` (Optional, Block) - Socket options.
  - `mark` (Optional, Number)
  - `tcp_keep_alive_interval` (Optional, Number)
  - `tcp_no_delay` (Optional, Boolean)
  - `tfo_enable` (Optional, Boolean)
  - `tproxy` (Optional, String)
  - `domain_strategy` (Optional, String)

### `sniffing` Block

- `enabled` (Optional, Boolean) - Whether sniffing is enabled.
- `dest_override` (Optional, List of String) - Destination override protocols (e.g. `http`, `tls`, `quic`, `fakedns`).
- `metadata_only` (Optional, Boolean) - Only sniff metadata.
- `route_only` (Optional, Boolean) - Only use sniffing for routing.
- `ips_excluded` (Optional, List of String) - IPs/CIDRs excluded from sniffing (e.g. `geoip:private`).
- `domains_excluded` (Optional, List of String) - Domains excluded from sniffing (e.g. `domain:example.com`).

## Attribute Reference

- `id` - The inbound ID (numeric).
- `all_time` (Read-only, Number, **Deprecated**) - Always `0` on every supported panel. 3x-ui carried an `allTime` field from v2.6.7 until [v3.1.0 removed it](https://github.com/MHSanaei/3x-ui/pull/4469) (that change also drops the `all_time` database columns on startup), and no version this provider supports (v3.2.x+) sends it. The attribute is kept for state compatibility and **will be removed in the next major release**. Use `up`, `down`, or the `threexui_client_traffics` data source instead.
- `last_traffic_reset_time` (Number) - Last traffic reset timestamp.
- `tag` (String) - Auto-generated inbound tag.

## Import

Inbounds can be imported using their numeric ID:

```shell
terraform import threexui_inbound.example 1
```

After import, you only need to specify the fields you want to manage in your configuration.
Server-populated fields (`show`, `xver`, `short_ids`, `fingerprint`, `public_key`, `private_key`,
`spider_x`, `metadata_only`, `route_only`, `encryption`, `selected_auth`, etc.) are automatically
preserved from the imported state — no need to copy them into your `.tf` file.

For example, a minimal configuration for an imported VLESS + Reality inbound:

```hcl
resource "threexui_inbound" "example" {
  port     = 443
  protocol = "vless"
  enable   = true
  remark   = "My VLESS"

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

  sniffing {
    enabled       = true
    dest_override = ["http", "tls", "quic", "fakedns"]
  }
}
```

## `disable_flow` and client `flow`

When `disable_flow = true`, 3x-ui strips `flow` from every client attached to the
inbound: the panel applies `clientWithInboundFlow` on both the add-client and
update-client paths, and re-runs `stripClientFlows` over the inbound settings on
every inbound add/update. A `threexui_inbound_client` on such an inbound that
sets `flow = "xtls-rprx-vision"` therefore fails the apply with
`Provider produced inconsistent result after apply: .flow: was
cty.StringVal("xtls-rprx-vision"), but now cty.StringVal("")`.

Flipping `disable_flow` to `true` on an inbound whose clients already carry a
`flow` has the same effect: the panel clears them, and the next `terraform plan`
shows drift on each client. Leave `flow` unset on clients of a `disable_flow`
inbound.
