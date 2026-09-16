package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// --- Vmess: create + update remark ---

func TestAccInboundVmess(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "vmess" {
  port     = 25001
  protocol = "vmess"
  remark   = "acc-vmess-1"
  enable   = true
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.vmess", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.vmess", "protocol", "vmess"),
					resource.TestCheckResourceAttr("threexui_inbound.vmess", "remark", "acc-vmess-1"),
					resource.TestCheckResourceAttr("threexui_inbound.vmess", "port", "25001"),
				),
			},
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "vmess" {
  port     = 25001
  protocol = "vmess"
  remark   = "acc-vmess-2"
  enable   = true
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.vmess", "remark", "acc-vmess-2"),
				),
			},
		},
	})
}

// --- Trojan ---

func TestAccInboundTrojan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "trojan" {
  port     = 25002
  protocol = "trojan"
  remark   = "acc-trojan"
  enable   = true
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.trojan", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.trojan", "protocol", "trojan"),
					resource.TestCheckResourceAttr("threexui_inbound.trojan", "remark", "acc-trojan"),
				),
			},
		},
	})
}

// --- Shadowsocks with method/password/network ---

func TestAccInboundShadowsocks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "ss" {
  port     = 25003
  protocol = "shadowsocks"
  remark   = "acc-ss"
  enable   = true
  shadowsocks_settings {
    method   = "aes-256-gcm"
    password = "testpassword123"
    network  = "tcp,udp"
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.ss", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.ss", "protocol", "shadowsocks"),
				),
			},
		},
	})
}

// --- HTTP with auth/accounts/allow_transparent ---

func TestAccInboundHTTP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "http" {
  port     = 25004
  protocol = "http"
  remark   = "acc-http"
  enable   = true
  http_settings {
    auth              = "password"
    allow_transparent = false
    account {
      user = "testuser"
      pass = "testpass"
    }
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.http", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.http", "protocol", "http"),
				),
			},
		},
	})
}

// --- WireGuard with peers ---

func TestAccInboundWireguard(t *testing.T) {
	// Below the CI matrix floor (v3.2.0), so this only matters when an older
	// panel is pinned manually via THREEXUI_VERSION — kept for the same reason
	// as the requireBelowVersion gates: skip cleanly instead of hard-failing.
	requireMinVersion(t, "v2.9.0") // mtu changed from int to list [v4, v6] in v2.9.0
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "wg" {
  port     = 25005
  protocol = "wireguard"
  remark   = "acc-wireguard"
  enable   = true
  wireguard_settings {
    mtu = [1420, 1280]
    peer {
      public_key  = "dGVzdHB1YmxpY2tleXRlc3RwdWJsaWNrZXkxMjM0NQ=="
      allowed_ips = ["10.0.0.2/32"]
    }
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.wg", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.wg", "protocol", "wireguard"),
				),
			},
		},
	})
}

// --- Dokodemo-door (tunnel) ---

func TestAccInboundDokodemo(t *testing.T) {
	// dokodemo-door renamed to "tunnel" in v3.2.0. v3.3.0 is the oldest version in
	// the CI matrix, so this always skips there — it only runs when a panel older
	// than v3.2.0 is pointed at manually via THREEXUI_VERSION.
	requireBelowVersion(t, "v3.2.0")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "dokodemo" {
  port     = 25006
  protocol = "dokodemo-door"
  remark   = "acc-dokodemo"
  enable   = true
  dokodemo_settings {
    address          = "127.0.0.1"
    port             = 80
    network          = "tcp"
    follow_redirect  = false
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.dokodemo", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.dokodemo", "protocol", "dokodemo-door"),
				),
			},
		},
	})
}

// --- Tunnel (dokodemo-door alias) ---

func TestAccInboundTunnel(t *testing.T) {
	tunnelConfig := testAccProviderConfig() + `
resource "threexui_inbound" "tunnel" {
  port     = 25030
  protocol = "tunnel"
  remark   = "acc-tunnel"
  enable   = true
  dokodemo_settings {
    address         = "127.0.0.1"
    port            = 80
    network         = "tcp"
    follow_redirect = false
    port_map = {
      "80"  = "127.0.0.1:8080"
      "443" = "127.0.0.1:8443"
    }
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: tunnelConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.tunnel", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.tunnel", "protocol", "tunnel"),
					resource.TestCheckResourceAttr("threexui_inbound.tunnel", "dokodemo_settings.port_map.80", "127.0.0.1:8080"),
					resource.TestCheckResourceAttr("threexui_inbound.tunnel", "dokodemo_settings.port_map.443", "127.0.0.1:8443"),
				),
			},
			// Update remark
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "tunnel" {
  port     = 25030
  protocol = "tunnel"
  remark   = "acc-tunnel-updated"
  enable   = true
  dokodemo_settings {
    address         = "127.0.0.1"
    port            = 80
    network         = "tcp"
    follow_redirect = false
    port_map = {
      "80"  = "127.0.0.1:8080"
      "443" = "127.0.0.1:8443"
    }
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.tunnel", "remark", "acc-tunnel-updated"),
					resource.TestCheckResourceAttr("threexui_inbound.tunnel", "protocol", "tunnel"),
				),
			},
			// Import
			{
				ResourceName:      "threexui_inbound.tunnel",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// --- VLESS + Reality (auto keys) ---

func TestAccInboundReality(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "reality" {
  port     = 25007
  protocol = "vless"
  remark   = "acc-reality"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "reality"
    reality_settings {
      target       = "google.com:443"
      server_names = ["google.com"]
    }
  }
}
`,
				// Reality auto-generates keys — UseStateForUnknown plan modifiers
				// suppress the diff since auto-generated fields are preserved.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.reality", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.reality", "protocol", "vless"),
				),
			},
			// Add the client-version gate. "0.0.0" is the only way to remove the
			// lower bound — Xray 26.7.x substitutes its own 26.3.27 default for an
			// unset minClientVer.
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "reality" {
  port     = 25007
  protocol = "vless"
  remark   = "acc-reality"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "reality"
    reality_settings {
      target         = "google.com:443"
      server_names   = ["google.com"]
      min_client_ver = "0.0.0"
      max_client_ver = "255.255.255"
      max_timediff   = 60000
    }
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.reality",
						"stream_settings.reality_settings.min_client_ver", "0.0.0"),
					resource.TestCheckResourceAttr("threexui_inbound.reality",
						"stream_settings.reality_settings.max_client_ver", "255.255.255"),
					resource.TestCheckResourceAttr("threexui_inbound.reality",
						"stream_settings.reality_settings.max_timediff", "60000"),
				),
			},
			// Update in place: a concrete lower bound, and max_timediff back to the
			// disabling zero. The zero must survive as a concrete value rather than
			// collapsing to null, which would leave a permanent diff.
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "reality" {
  port     = 25007
  protocol = "vless"
  remark   = "acc-reality"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "reality"
    reality_settings {
      target         = "google.com:443"
      server_names   = ["google.com"]
      min_client_ver = "26.3.27"
      max_client_ver = "255.255.255"
      max_timediff   = 0
    }
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.reality",
						"stream_settings.reality_settings.min_client_ver", "26.3.27"),
					resource.TestCheckResourceAttr("threexui_inbound.reality",
						"stream_settings.reality_settings.max_timediff", "0"),
				),
			},
			// Dropping the attributes from the configuration is NOT a clear. These
			// are Optional+Computed with UseStateForUnknown — the same contract the
			// rest of the reality block relies on for import to stay driftless — so
			// an unconfigured value resolves to the prior state. Pin that down: the
			// plan must stay empty and the values must survive, which is why the
			// documented way to widen a bound is the extreme value (0.0.0 /
			// 255.255.255 / 0), not deletion.
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "reality" {
  port     = 25007
  protocol = "vless"
  remark   = "acc-reality"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "reality"
    reality_settings {
      target       = "google.com:443"
      server_names = ["google.com"]
    }
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.reality",
						"stream_settings.reality_settings.min_client_ver", "26.3.27"),
					resource.TestCheckResourceAttr("threexui_inbound.reality",
						"stream_settings.reality_settings.max_client_ver", "255.255.255"),
					resource.TestCheckResourceAttr("threexui_inbound.reality",
						"stream_settings.reality_settings.max_timediff", "0"),
				),
			},
		},
	})
}

// --- VLESS with fallbacks ---

func TestAccInboundFallbacks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "fallback" {
  port     = 25008
  protocol = "vless"
  remark   = "acc-fallback"
  enable   = true
  vless_settings {
    decryption = "none"
    fallback {
      name = "default"
      dest = "80"
      xver = 1
    }
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.fallback", "id"),
				),
			},
		},
	})
}

// --- Settings: decryption preserved ---

func TestAccInboundSettings(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "settings" {
  port     = 25009
  protocol = "vless"
  remark   = "acc-settings"
  enable   = true
  vless_settings {
    decryption = "none"
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.settings", "id"),
				),
			},
			// Re-apply same config: testseed should be preserved
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "settings" {
  port     = 25009
  protocol = "vless"
  remark   = "acc-settings"
  enable   = true
  vless_settings {
    decryption = "none"
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.settings", "id"),
				),
			},
		},
	})
}

// --- Stream settings + sniffing update ---

func TestAccInboundStreamSniffing(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "stream" {
  port     = 25010
  protocol = "vless"
  remark   = "acc-stream"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "none"
  }
  sniffing {
    enabled       = true
    dest_override = ["http", "tls"]
    metadata_only = false
    route_only    = false
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.stream", "id"),
				),
			},
			// Update sniffing
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "stream" {
  port     = 25010
  protocol = "vless"
  remark   = "acc-stream"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "none"
  }
  sniffing {
    enabled       = true
    dest_override = ["http", "tls", "quic", "fakedns"]
    metadata_only = true
    route_only    = false
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.stream", "id"),
				),
			},
		},
	})
}

// --- Port conflict -> error ---

func TestAccInboundPortConflict(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "conflict1" {
  port     = 25011
  protocol = "vless"
  remark   = "acc-conflict-1"
  enable   = true
  vless_settings {
    decryption = "none"
  }
}

resource "threexui_inbound" "conflict2" {
  port     = 25011
  protocol = "vless"
  remark   = "acc-conflict-2"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  depends_on = [threexui_inbound.conflict1]
}
`,
				ExpectError: regexp.MustCompile(`(?i)(port|exist|already|error|fail)`),
			},
		},
	})
}

// --- Update port, enable, remark ---

func TestAccInboundUpdateFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccInboundUpdateConfig("acc-update-1", 25012, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.update", "remark", "acc-update-1"),
					resource.TestCheckResourceAttr("threexui_inbound.update", "port", "25012"),
					resource.TestCheckResourceAttr("threexui_inbound.update", "enable", "true"),
				),
			},
			{
				Config: testAccProviderConfig() + testAccInboundUpdateConfig("acc-update-2", 25013, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.update", "remark", "acc-update-2"),
					resource.TestCheckResourceAttr("threexui_inbound.update", "port", "25013"),
					resource.TestCheckResourceAttr("threexui_inbound.update", "enable", "false"),
				),
			},
		},
	})
}

// --- Idempotency: no changes on re-apply ---

func TestAccInboundIdempotency(t *testing.T) {
	config := testAccProviderConfig() + `
resource "threexui_inbound" "idem" {
  port     = 25014
  protocol = "vless"
  remark   = "acc-idempotent"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "none"
    tcp_settings {
      accept_proxy_protocol = false
      header_type           = "none"
    }
  }
  sniffing {
    enabled       = true
    dest_override = ["http", "tls"]
    metadata_only = false
    route_only    = false
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.idem", "id"),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// --- Negative: duplicate client email ---

func TestAccInboundClientDuplicateEmail(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "dup_host" {
  port     = 25021
  protocol = "vless"
  remark   = "acc-dup-email-host"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "reality"
    reality_settings {
      target       = "google.com:443"
      server_names = ["google.com"]
    }
  }
}

resource "threexui_inbound_client" "dup1" {
  inbound_id = threexui_inbound.dup_host.id
  email      = "duplicate@test.com"
  enable     = true
  flow       = "xtls-rprx-vision"
}

resource "threexui_inbound_client" "dup2" {
  inbound_id = threexui_inbound.dup_host.id
  email      = "duplicate@test.com"
  enable     = true
  flow       = "xtls-rprx-vision"
  depends_on = [threexui_inbound_client.dup1]
}
`,
				ExpectError: regexp.MustCompile(`(?i)(error|fail|duplicate|exist|already)`),
			},
		},
	})
}

// --- Stream settings: WebSocket transport ---

func TestAccInboundWebSocket(t *testing.T) {
	config := testAccProviderConfig() + `
resource "threexui_inbound" "ws" {
  port     = 25022
  protocol = "vless"
  remark   = "acc-ws"
  enable   = true
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
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.ws", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.ws", "protocol", "vless"),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// --- Stream settings: gRPC transport ---

func TestAccInboundGRPC(t *testing.T) {
	config := testAccProviderConfig() + `
resource "threexui_inbound" "grpc" {
  port     = 25023
  protocol = "vless"
  remark   = "acc-grpc"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "grpc"
    security = "none"
    grpc_settings {
      service_name = "mygrpc"
    }
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.grpc", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.grpc", "protocol", "vless"),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// --- Mixed protocol + listen + settings ---

func TestAccInboundMixed(t *testing.T) {
	// Below the CI matrix floor (v3.2.0), so this only matters when an older
	// panel is pinned manually via THREEXUI_VERSION — kept for the same reason
	// as the requireBelowVersion gates: skip cleanly instead of hard-failing.
	requireMinVersion(t, "v2.9.0") // mixed protocol added in v2.9.0
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "mixed" {
  port     = 25024
  protocol = "mixed"
  remark   = "acc-mixed"
  listen   = "127.0.0.1"
  enable   = true
  mixed_settings {
    auth = "password"
    udp  = true
    ip   = "127.0.0.1"
    account {
      user = "mixeduser"
      pass = "mixedpass"
    }
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.mixed", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.mixed", "protocol", "mixed"),
					resource.TestCheckResourceAttr("threexui_inbound.mixed", "listen", "127.0.0.1"),
					resource.TestCheckResourceAttr("threexui_inbound.mixed", "mixed_settings.auth", "password"),
					resource.TestCheckResourceAttr("threexui_inbound.mixed", "mixed_settings.udp", "true"),
				),
			},
			// Update: change udp to false
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "mixed" {
  port     = 25024
  protocol = "mixed"
  remark   = "acc-mixed-updated"
  listen   = "127.0.0.1"
  enable   = true
  mixed_settings {
    auth = "password"
    udp  = false
    ip   = "127.0.0.1"
    account {
      user = "mixeduser"
      pass = "mixedpass"
    }
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.mixed", "remark", "acc-mixed-updated"),
					resource.TestCheckResourceAttr("threexui_inbound.mixed", "mixed_settings.udp", "false"),
				),
			},
		},
	})
}

// --- Listen field ---

func TestAccInboundListen(t *testing.T) {
	configNoListen := testAccProviderConfig() + `
resource "threexui_inbound" "listen" {
  port     = 25025
  protocol = "vless"
  remark   = "acc-listen"
  enable   = true
  vless_settings {
    decryption = "none"
  }
}
`
	configWithListen := testAccProviderConfig() + `
resource "threexui_inbound" "listen" {
  port     = 25025
  protocol = "vless"
  remark   = "acc-listen"
  listen   = "0.0.0.0"
  enable   = true
  vless_settings {
    decryption = "none"
  }
}
`
	configWithListenChanged := testAccProviderConfig() + `
resource "threexui_inbound" "listen" {
  port     = 25025
  protocol = "vless"
  remark   = "acc-listen"
  listen   = "127.0.0.1"
  enable   = true
  vless_settings {
    decryption = "none"
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			// Step 1: create without listen — should be null in state
			{
				Config: configNoListen,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.listen", "id"),
					resource.TestCheckNoResourceAttr("threexui_inbound.listen", "listen"),
				),
			},
			// Step 2: idempotency — no diff on re-apply
			{
				Config:             configNoListen,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 3: set listen to 0.0.0.0
			{
				Config: configWithListen,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.listen", "listen", "0.0.0.0"),
				),
			},
			// Step 4: idempotency with listen set
			{
				Config:             configWithListen,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 5: change listen to 127.0.0.1
			{
				Config: configWithListenChanged,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.listen", "listen", "127.0.0.1"),
				),
			},
			// Step 6: remove listen — back to null
			{
				Config: configNoListen,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("threexui_inbound.listen", "listen"),
				),
			},
			// Step 7: idempotency after removing listen
			{
				Config:             configNoListen,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// --- Total field (default 0 = unlimited) ---

func TestAccInboundTotalDefault(t *testing.T) {
	configNoTotal := testAccProviderConfig() + `
resource "threexui_inbound" "total" {
  port     = 25027
  protocol = "vless"
  remark   = "acc-total"
  enable   = true
  vless_settings {
    decryption = "none"
  }
}
`
	configWithTotal := testAccProviderConfig() + `
resource "threexui_inbound" "total" {
  port     = 25027
  protocol = "vless"
  remark   = "acc-total"
  total    = 1073741824
  enable   = true
  vless_settings {
    decryption = "none"
  }
}
`
	configWithTotalChanged := testAccProviderConfig() + `
resource "threexui_inbound" "total" {
  port     = 25027
  protocol = "vless"
  remark   = "acc-total"
  total    = 2147483648
  enable   = true
  vless_settings {
    decryption = "none"
  }
}
`
	configTotalZero := testAccProviderConfig() + `
resource "threexui_inbound" "total" {
  port     = 25027
  protocol = "vless"
  remark   = "acc-total"
  total    = 0
  enable   = true
  vless_settings {
    decryption = "none"
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			// Step 1: create without total — defaults to 0 (unlimited)
			{
				Config: configNoTotal,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.total", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.total", "total", "0"),
				),
			},
			// Step 2: idempotency
			{
				Config:             configNoTotal,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 3: set total to 1 GB
			{
				Config: configWithTotal,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.total", "total", "1073741824"),
				),
			},
			// Step 4: idempotency with total set
			{
				Config:             configWithTotal,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 5: change total to 2 GB
			{
				Config: configWithTotalChanged,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.total", "total", "2147483648"),
				),
			},
			// Step 6: explicitly set total to 0 (unlimited)
			{
				Config: configTotalZero,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.total", "total", "0"),
				),
			},
			// Step 7: remove total from config — back to default 0
			{
				Config: configNoTotal,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.total", "total", "0"),
				),
			},
			// Step 8: idempotency after removing total
			{
				Config:             configNoTotal,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// --- Remark omitted (default empty string), then set, then removed ---

func TestAccInboundRemarkDefault(t *testing.T) {
	configNoRemark := testAccProviderConfig() + `
resource "threexui_inbound" "noremark" {
  port     = 25026
  protocol = "vless"
  enable   = true
  vless_settings {
    decryption = "none"
  }
}
`
	configWithRemark := testAccProviderConfig() + `
resource "threexui_inbound" "noremark" {
  port     = 25026
  protocol = "vless"
  remark   = "now-has-remark"
  enable   = true
  vless_settings {
    decryption = "none"
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			// Step 1: create without remark — defaults to ""
			{
				Config: configNoRemark,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.noremark", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.noremark", "remark", ""),
				),
			},
			// Step 2: idempotency — no diff on re-apply
			{
				Config:             configNoRemark,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 3: set remark
			{
				Config: configWithRemark,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.noremark", "remark", "now-has-remark"),
				),
			},
			// Step 4: remove remark — back to default ""
			{
				Config: configNoRemark,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.noremark", "remark", ""),
				),
			},
			// Step 5: idempotency after removing remark
			{
				Config:             configNoRemark,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// --- Hysteria ---

func TestAccInboundHysteria(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "hysteria" {
  port     = 25501
  protocol = "hysteria"
  remark   = "acc-hysteria-1"
  enable   = true

  hysteria_settings {
    version = 2
  }

  stream_settings {
    network  = "hysteria"
  }
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.hysteria", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.hysteria", "protocol", "hysteria"),
					resource.TestCheckResourceAttr("threexui_inbound.hysteria", "remark", "acc-hysteria-1"),
					resource.TestCheckResourceAttr("threexui_inbound.hysteria", "port", "25501"),
				),
			},
		},
	})
}

// --- Import: no drift with minimal config ---
//
// After importing an existing inbound, planning with a config that omits
// Optional+Computed fields (show, xver, short_ids, metadata_only, route_only,
// encryption, selected_auth, reality settings attribute, etc.) must produce an
// empty plan.  This is the primary regression test for the UseStateForUnknown
// plan modifiers (including the object-level modifier on reality_settings.settings).

func TestAccInboundImportNoDrift_Reality(t *testing.T) {
	config := testAccProviderConfig() + `
resource "threexui_inbound" "import_reality" {
  port     = 25031
  protocol = "vless"
  remark   = "acc-import-reality"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "reality"
    reality_settings {
      target       = "google.com:443"
      server_names = ["google.com"]
    }
  }
  sniffing {
    enabled       = true
    dest_override = ["http", "tls"]
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			// Step 1: Create — server populates show, xver, short_ids,
			// reality settings (fingerprint, public_key, spider_x),
			// sniffing (metadata_only, route_only), vless (encryption).
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.import_reality", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.import_reality", "protocol", "vless"),
				),
			},
			// Step 2: Import — fresh state from the API.
			// reality_settings.settings is an Optional+Computed attribute
			// with UseStateForUnknown — both Create and Import populate it
			// from the API, so ImportStateVerify passes without ignores.
			{
				ResourceName:      "threexui_inbound.import_reality",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Plan with the same config — must be empty.
			// Without UseStateForUnknown modifiers this step fails because
			// Terraform plans show, xver, short_ids, metadata_only, etc.
			// as "(known after apply)".
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccInboundImportNoDrift_Sniffing(t *testing.T) {
	config := testAccProviderConfig() + `
resource "threexui_inbound" "import_sniffing" {
  port     = 25032
  protocol = "vless"
  remark   = "acc-import-sniffing"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "none"
  }
  sniffing {
    enabled       = true
    dest_override = ["http", "tls"]
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.import_sniffing", "id"),
				),
			},
			{
				ResourceName:      "threexui_inbound.import_sniffing",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// After import, metadata_only, route_only, ips_excluded,
			// domains_excluded are in state but not in config — must not drift.
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// --- XHTTP with xPadding fields ---

func TestAccInboundXHTTPPadding(t *testing.T) {
	// Below the CI matrix floor (v3.2.0), so this only matters when an older
	// panel is pinned manually via THREEXUI_VERSION — kept for the same reason
	// as the requireBelowVersion gates: skip cleanly instead of hard-failing.
	requireMinVersion(t, "v2.9.2") // xPaddingObfsMode, xPaddingKey added in v2.9.2
	config := testAccProviderConfig() + `
resource "threexui_inbound" "xhttp_pad" {
  port     = 25033
  protocol = "vless"
  remark   = "acc-xhttp-xpadding"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "xhttp"
    security = "none"
    xhttp_settings {
      path                = "/xhttp-padded"
      mode                = "auto"
      x_padding_bytes     = "100-1000"
      x_padding_obfs_mode = true
      x_padding_key       = "my-secret-key"
      x_padding_header    = "X-Padding"
      x_padding_placement = "header"
      x_padding_method    = "aes"
    }
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.xhttp_pad", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.xhttp_pad", "protocol", "vless"),
					resource.TestCheckResourceAttr("threexui_inbound.xhttp_pad", "stream_settings.xhttp_settings.path", "/xhttp-padded"),
					resource.TestCheckResourceAttr("threexui_inbound.xhttp_pad", "stream_settings.xhttp_settings.x_padding_bytes", "100-1000"),
					resource.TestCheckResourceAttr("threexui_inbound.xhttp_pad", "stream_settings.xhttp_settings.x_padding_obfs_mode", "true"),
					resource.TestCheckResourceAttr("threexui_inbound.xhttp_pad", "stream_settings.xhttp_settings.x_padding_key", "my-secret-key"),
					resource.TestCheckResourceAttr("threexui_inbound.xhttp_pad", "stream_settings.xhttp_settings.x_padding_header", "X-Padding"),
					resource.TestCheckResourceAttr("threexui_inbound.xhttp_pad", "stream_settings.xhttp_settings.x_padding_placement", "header"),
					resource.TestCheckResourceAttr("threexui_inbound.xhttp_pad", "stream_settings.xhttp_settings.x_padding_method", "aes"),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// --- Config helpers ---

func testAccInboundUpdateConfig(remark string, port int, enable bool) string {
	return fmt.Sprintf(`
resource "threexui_inbound" "update" {
  port     = %d
  protocol = "vless"
  remark   = %q
  enable   = %t
  vless_settings {
    decryption = "none"
  }
}
`, port, remark, enable)
}

// --- Inbound v3.3.1 multi-node subscription fields (sub_sort_index, share_addr, share_addr_strategy) ---

func TestAccInboundSubscriptionFields(t *testing.T) {
	requireMinVersion(t, "v3.3.1")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "sub" {
  port               = 25040
  protocol           = "trojan"
  remark             = "acc-sub-fields"
  enable             = true
  sub_sort_index     = 2
  share_addr         = "203.0.113.10"
  share_addr_strategy = "custom"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_inbound.sub", "id"),
					resource.TestCheckResourceAttr("threexui_inbound.sub", "sub_sort_index", "2"),
					resource.TestCheckResourceAttr("threexui_inbound.sub", "share_addr", "203.0.113.10"),
					resource.TestCheckResourceAttr("threexui_inbound.sub", "share_addr_strategy", "custom"),
				),
			},
		},
	})
}

// TestAccInboundDisableFlow_v370 covers the v3.7.0 inbound `disable_flow` flag,
// which opts the inbound out of the panel's automatic XTLS Vision assignment.
func TestAccInboundDisableFlow_v370(t *testing.T) {
	requireMinVersion(t, "v3.7.0")
	config := testAccProviderConfig() + `
resource "threexui_inbound" "disable_flow" {
  port         = 25132
  protocol     = "vless"
  remark       = "acc-inbound-disable-flow-v370"
  enable       = true
  disable_flow = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "reality"
    reality_settings {
      target       = "google.com:443"
      server_names = ["google.com"]
    }
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr("threexui_inbound.disable_flow", "disable_flow", "true"),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccInboundTrafficResetDay_v360 is the end-to-end guard for the v3.6.0
// `traffic_reset_day` write path. The field was present in the model, schema,
// expand and flatten but missing from inboundToForm, so every configured value
// was silently dropped and the panel kept its own default — invisible to a
// url.Values unit test, and exactly what the PlanOnly step below catches.
func TestAccInboundTrafficResetDay_v360(t *testing.T) {
	requireMinVersion(t, "v3.6.0")
	config := testAccProviderConfig() + `
resource "threexui_inbound" "reset_day" {
  port              = 25133
  protocol          = "vless"
  remark            = "acc-inbound-traffic-reset-day-v360"
  enable            = true
  traffic_reset     = "monthly"
  traffic_reset_day = 15
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "reality"
    reality_settings {
      target       = "google.com:443"
      server_names = ["google.com"]
    }
  }
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.reset_day", "traffic_reset", "monthly"),
					resource.TestCheckResourceAttr("threexui_inbound.reset_day", "traffic_reset_day", "15"),
				),
			},
			// A dropped write shows up here: the panel would report its own day,
			// leaving a permanent diff against the configured 15.
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Updating the day must also reach the panel, not just state.
			{
				Config: testAccProviderConfig() + `
resource "threexui_inbound" "reset_day" {
  port              = 25133
  protocol          = "vless"
  remark            = "acc-inbound-traffic-reset-day-v360"
  enable            = true
  traffic_reset     = "monthly"
  traffic_reset_day = 28
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "reality"
    reality_settings {
      target       = "google.com:443"
      server_names = ["google.com"]
    }
  }
}
`,
				Check: resource.TestCheckResourceAttr("threexui_inbound.reset_day", "traffic_reset_day", "28"),
			},
		},
	})
}
