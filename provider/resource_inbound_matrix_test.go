package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// ---------------------------------------------------------------------------
// Protocol round-trip acceptance matrix (#70)
//
// Each protocol entry defines HCL configs for create and update, plus
// protocol-specific checks. Every entry exercises the full lifecycle:
//   Create -> Read -> ImportStateVerify -> Update -> PlanOnly (idempotency)
//
// Client-bearing protocols (vless, vmess, trojan, shadowsocks, hysteria)
// additionally create an inbound_client resource and verify it survives
// the inbound update.
// ---------------------------------------------------------------------------

type protocolMatrixEntry struct {
	// protocol is the Terraform protocol value (e.g. "vless", "vmess").
	protocol string
	// tfName is the Terraform resource name suffix (e.g. "vless" -> threexui_inbound.vless).
	tfName string
	// port is a unique port for this protocol's test.
	port int
	// minVersion is the minimum 3x-ui version required (e.g. "v2.9.0"). Empty means all versions.
	minVersion string
	// maxVersion is the maximum 3x-ui version (exclusive). Empty means no upper bound.
	// Use for protocols removed upstream (e.g. "v3.2.0" means not available on v3.2.0+).
	maxVersion string
	// createHCL returns the HCL for the initial create step.
	createHCL func(port int) string
	// updateHCL returns the HCL for the update step (must change at least one
	// protocol-specific field beyond remark).
	updateHCL func(port int) string
	// createChecks returns TestCheckFuncs evaluated after create.
	createChecks func(tfAddr string) []resource.TestCheckFunc
	// updateChecks returns TestCheckFuncs evaluated after update.
	updateChecks func(tfAddr string) []resource.TestCheckFunc
	// hasClient indicates this protocol supports clients.
	hasClient bool
	// clientHCL returns an inbound_client resource referencing the inbound.
	// Only used when hasClient is true.
	clientHCL func(inboundTFAddr string) string
	// clientChecks returns TestCheckFuncs for the client resource.
	clientChecks func(clientTFAddr string) []resource.TestCheckFunc
	// importStateVerifyIgnore lists attributes to ignore on import verification.
	importStateVerifyIgnore []string
}

// protocolMatrix returns the full set of protocol test entries.
func protocolMatrix() []protocolMatrixEntry {
	return []protocolMatrixEntry{
		matrixVless(),
		matrixVmess(),
		matrixTrojan(),
		matrixShadowsocks(),
		matrixHTTP(),
		matrixSocks(),
		matrixMixed(),
		matrixWireguard(),
		matrixAmneziawg(),
		matrixTuic(),
		matrixDokodemo(),
		matrixHysteria(),
	}
}

// ---------------------------------------------------------------------------
// TestAccInboundProtocolMatrix — table-driven round-trip test
// ---------------------------------------------------------------------------

func TestAccInboundProtocolMatrix(t *testing.T) {
	for _, entry := range protocolMatrix() {
		t.Run(entry.protocol, func(t *testing.T) {
			if entry.minVersion != "" {
				requireMinVersion(t, entry.minVersion)
			}
			if entry.maxVersion != "" {
				requireBelowVersion(t, entry.maxVersion)
			}

			inboundAddr := fmt.Sprintf("threexui_inbound.%s", entry.tfName)
			createConfig := testAccProviderConfig() + entry.createHCL(entry.port)
			updateConfig := testAccProviderConfig() + entry.updateHCL(entry.port)

			// If protocol supports clients, append client resources.
			clientAddr := ""
			if entry.hasClient && entry.clientHCL != nil {
				clientAddr = fmt.Sprintf("threexui_inbound_client.%s_client", entry.tfName)
				clientHCL := entry.clientHCL(inboundAddr)
				createConfig += clientHCL
				updateConfig += clientHCL
			}

			steps := []resource.TestStep{
				// Step 1: Create
				{
					Config: createConfig,
					Check:  resource.ComposeTestCheckFunc(matrixCreateChecks(inboundAddr, entry, clientAddr)...),
				},
				// Step 2: Import
				{
					ResourceName:            inboundAddr,
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateVerifyIgnore: entry.importStateVerifyIgnore,
				},
				// Step 3: Update (protocol-specific change)
				{
					Config: updateConfig,
					Check:  resource.ComposeTestCheckFunc(matrixUpdateChecks(inboundAddr, entry, clientAddr)...),
				},
				// Step 4: Idempotency — no diff on re-apply
				{
					Config:             updateConfig,
					PlanOnly:           true,
					ExpectNonEmptyPlan: false,
				},
			}

			checkDestroy := testAccCheckInboundDestroyed
			if entry.hasClient {
				checkDestroy = func(state *terraform.State) error {
					if err := testAccCheckInboundDestroyed(state); err != nil {
						return err
					}
					return testAccCheckInboundClientDestroyed(state)
				}
			}

			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
				CheckDestroy:             checkDestroy,
				Steps:                    steps,
			})
		})
	}
}

// ---------------------------------------------------------------------------
// Check helpers
// ---------------------------------------------------------------------------

func matrixCreateChecks(inboundAddr string, e protocolMatrixEntry, clientAddr string) []resource.TestCheckFunc {
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttrSet(inboundAddr, "id"),
		resource.TestCheckResourceAttr(inboundAddr, "protocol", e.protocol),
		resource.TestCheckResourceAttr(inboundAddr, "port", fmt.Sprintf("%d", e.port)),
	}
	if e.createChecks != nil {
		checks = append(checks, e.createChecks(inboundAddr)...)
	}
	if clientAddr != "" && e.clientChecks != nil {
		checks = append(checks, e.clientChecks(clientAddr)...)
	}
	return checks
}

func matrixUpdateChecks(inboundAddr string, e protocolMatrixEntry, clientAddr string) []resource.TestCheckFunc {
	checks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttrSet(inboundAddr, "id"),
	}
	if e.updateChecks != nil {
		checks = append(checks, e.updateChecks(inboundAddr)...)
	}
	// After update, client must still exist.
	if clientAddr != "" && e.clientChecks != nil {
		checks = append(checks, e.clientChecks(clientAddr)...)
	}
	return checks
}

// ---------------------------------------------------------------------------
// Per-protocol definitions
// ---------------------------------------------------------------------------

func matrixVless() protocolMatrixEntry {
	return protocolMatrixEntry{
		protocol: "vless",
		tfName:   "mx_vless",
		port:     26001,
		createHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_vless" {
  port     = %d
  protocol = "vless"
  remark   = "matrix-vless-create"
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
    metadata_only = false
    route_only    = false
  }
}
`, port)
		},
		updateHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_vless" {
  port     = %d
  protocol = "vless"
  remark   = "matrix-vless-updated"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "tcp"
    security = "reality"
    reality_settings {
      target       = "github.com:443"
      server_names = ["github.com"]
    }
  }
  sniffing {
    enabled       = true
    dest_override = ["http", "tls"]
    metadata_only = false
    route_only    = false
  }
}
`, port)
		},
		createChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-vless-create"),
				resource.TestCheckResourceAttr(addr, "stream_settings.network", "tcp"),
				resource.TestCheckResourceAttr(addr, "stream_settings.security", "reality"),
				resource.TestCheckResourceAttr(addr, "stream_settings.reality_settings.target", "google.com:443"),
			}
		},
		updateChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-vless-updated"),
				resource.TestCheckResourceAttr(addr, "stream_settings.network", "tcp"),
				resource.TestCheckResourceAttr(addr, "stream_settings.security", "reality"),
				resource.TestCheckResourceAttr(addr, "stream_settings.reality_settings.target", "github.com:443"),
			}
		},
		hasClient: true,
		clientHCL: func(inboundAddr string) string {
			return fmt.Sprintf(`
resource "threexui_inbound_client" "mx_vless_client" {
  inbound_id = %s.id
  email      = "matrix-vless@test.com"
  enable     = true
  flow       = "xtls-rprx-vision"
}
`, inboundAddr)
		},
		clientChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttrSet(addr, "id"),
				resource.TestCheckResourceAttr(addr, "email", "matrix-vless@test.com"),
			}
		},
	}
}

func matrixVmess() protocolMatrixEntry {
	return protocolMatrixEntry{
		protocol: "vmess",
		tfName:   "mx_vmess",
		port:     26002,
		createHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_vmess" {
  port     = %d
  protocol = "vmess"
  remark   = "matrix-vmess-create"
  enable   = true
}
`, port)
		},
		updateHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_vmess" {
  port     = %d
  protocol = "vmess"
  remark   = "matrix-vmess-updated"
  enable   = false
  stream_settings {
    network  = "ws"
    security = "none"
    ws_settings {
      path = "/vmess-ws"
    }
  }
}
`, port)
		},
		createChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-vmess-create"),
				resource.TestCheckResourceAttr(addr, "enable", "true"),
			}
		},
		updateChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-vmess-updated"),
				resource.TestCheckResourceAttr(addr, "enable", "false"),
				resource.TestCheckResourceAttr(addr, "stream_settings.network", "ws"),
			}
		},
		hasClient: true,
		clientHCL: func(inboundAddr string) string {
			return fmt.Sprintf(`
resource "threexui_inbound_client" "mx_vmess_client" {
  inbound_id = %s.id
  email      = "matrix-vmess@test.com"
  enable     = true
  security   = "auto"
}
`, inboundAddr)
		},
		clientChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttrSet(addr, "id"),
				resource.TestCheckResourceAttr(addr, "email", "matrix-vmess@test.com"),
				resource.TestCheckResourceAttr(addr, "security", "auto"),
			}
		},
	}
}

func matrixTrojan() protocolMatrixEntry {
	return protocolMatrixEntry{
		protocol: "trojan",
		tfName:   "mx_trojan",
		port:     26003,
		createHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_trojan" {
  port     = %d
  protocol = "trojan"
  remark   = "matrix-trojan-create"
  enable   = true
}
`, port)
		},
		updateHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_trojan" {
  port     = %d
  protocol = "trojan"
  remark   = "matrix-trojan-updated"
  enable   = true
  trojan_settings {
    fallback {
      dest = "127.0.0.1:8080"
      xver = 0
    }
  }
  sniffing {
    enabled       = true
    dest_override = ["http", "tls"]
    metadata_only = false
    route_only    = false
  }
}
`, port)
		},
		createChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-trojan-create"),
			}
		},
		updateChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-trojan-updated"),
				resource.TestCheckResourceAttr(addr, "sniffing.enabled", "true"),
				resource.TestCheckResourceAttr(addr, "trojan_settings.fallback.0.dest", "127.0.0.1:8080"),
			}
		},
		hasClient: true,
		clientHCL: func(inboundAddr string) string {
			return fmt.Sprintf(`
resource "threexui_inbound_client" "mx_trojan_client" {
  inbound_id = %s.id
  email      = "matrix-trojan@test.com"
  enable     = true
  password   = "trojan-matrix-pass"
}
`, inboundAddr)
		},
		clientChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttrSet(addr, "id"),
				resource.TestCheckResourceAttr(addr, "email", "matrix-trojan@test.com"),
			}
		},
	}
}

func matrixShadowsocks() protocolMatrixEntry {
	return protocolMatrixEntry{
		protocol: "shadowsocks",
		tfName:   "mx_ss",
		port:     26004,
		createHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_ss" {
  port     = %d
  protocol = "shadowsocks"
  remark   = "matrix-ss-create"
  enable   = true
  shadowsocks_settings {
    method   = "aes-256-gcm"
    password = "matrix-ss-pass"
    network  = "tcp,udp"
  }
}
`, port)
		},
		updateHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_ss" {
  port     = %d
  protocol = "shadowsocks"
  remark   = "matrix-ss-updated"
  enable   = true
  shadowsocks_settings {
    method   = "chacha20-ietf-poly1305"
    password = "matrix-ss-pass"
    network  = "tcp,udp"
  }
}
`, port)
		},
		createChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-ss-create"),
				resource.TestCheckResourceAttr(addr, "shadowsocks_settings.method", "aes-256-gcm"),
			}
		},
		updateChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-ss-updated"),
				resource.TestCheckResourceAttr(addr, "shadowsocks_settings.method", "chacha20-ietf-poly1305"),
			}
		},
		hasClient: true,
		clientHCL: func(inboundAddr string) string {
			return fmt.Sprintf(`
resource "threexui_inbound_client" "mx_ss_client" {
  inbound_id = %s.id
  email      = "matrix-ss@test.com"
  enable     = true
  password   = "ss-client-pass"
}
`, inboundAddr)
		},
		clientChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttrSet(addr, "id"),
				resource.TestCheckResourceAttr(addr, "email", "matrix-ss@test.com"),
			}
		},
		importStateVerifyIgnore: []string{
			"shadowsocks_settings.password",
		},
	}
}

func matrixHTTP() protocolMatrixEntry {
	return protocolMatrixEntry{
		protocol: "http",
		tfName:   "mx_http",
		port:     26005,
		createHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_http" {
  port     = %d
  protocol = "http"
  remark   = "matrix-http-create"
  enable   = true
  http_settings {
    auth              = "password"
    allow_transparent = false
    account {
      user = "httpuser"
      pass = "httppass"
    }
  }
}
`, port)
		},
		updateHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_http" {
  port     = %d
  protocol = "http"
  remark   = "matrix-http-updated"
  enable   = true
  http_settings {
    auth              = "password"
    allow_transparent = true
    account {
      user = "httpuser"
      pass = "httppass"
    }
  }
}
`, port)
		},
		createChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-http-create"),
				resource.TestCheckResourceAttr(addr, "http_settings.allow_transparent", "false"),
			}
		},
		updateChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-http-updated"),
				resource.TestCheckResourceAttr(addr, "http_settings.allow_transparent", "true"),
				resource.TestCheckResourceAttr(addr, "http_settings.account.0.user", "httpuser"),
				resource.TestCheckResourceAttr(addr, "http_settings.account.0.pass", "httppass"),
			}
		},
	}
}

func matrixSocks() protocolMatrixEntry {
	return protocolMatrixEntry{
		protocol:   "socks",
		tfName:     "mx_socks",
		port:       26006,
		maxVersion: "v3.2.0", // socks removed upstream; use "mixed" instead
		createHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_socks" {
  port     = %d
  protocol = "socks"
  remark   = "matrix-socks-create"
  enable   = true
  socks_settings {
    auth = "password"
    udp  = true
    ip   = "127.0.0.1"
    account {
      user = "socksuser"
      pass = "sockspass"
    }
  }
}
`, port)
		},
		updateHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_socks" {
  port     = %d
  protocol = "socks"
  remark   = "matrix-socks-updated"
  enable   = true
  socks_settings {
    auth = "password"
    udp  = false
    ip   = "127.0.0.1"
    account {
      user = "socksuser"
      pass = "sockspass"
    }
  }
}
`, port)
		},
		createChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-socks-create"),
				resource.TestCheckResourceAttr(addr, "socks_settings.udp", "true"),
			}
		},
		updateChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-socks-updated"),
				resource.TestCheckResourceAttr(addr, "socks_settings.udp", "false"),
				resource.TestCheckResourceAttr(addr, "socks_settings.account.0.user", "socksuser"),
				resource.TestCheckResourceAttr(addr, "socks_settings.account.0.pass", "sockspass"),
			}
		},
	}
}

func matrixMixed() protocolMatrixEntry {
	return protocolMatrixEntry{
		protocol:   "mixed",
		tfName:     "mx_mixed",
		port:       26010,
		minVersion: "v2.9.0", // mixed protocol added in v2.9.0
		createHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_mixed" {
  port     = %d
  protocol = "mixed"
  remark   = "matrix-mixed-create"
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
`, port)
		},
		updateHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_mixed" {
  port     = %d
  protocol = "mixed"
  remark   = "matrix-mixed-updated"
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
`, port)
		},
		createChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-mixed-create"),
				resource.TestCheckResourceAttr(addr, "mixed_settings.udp", "true"),
			}
		},
		updateChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-mixed-updated"),
				resource.TestCheckResourceAttr(addr, "mixed_settings.udp", "false"),
				resource.TestCheckResourceAttr(addr, "mixed_settings.account.0.user", "mixeduser"),
				resource.TestCheckResourceAttr(addr, "mixed_settings.account.0.pass", "mixedpass"),
			}
		},
	}
}

func matrixWireguard() protocolMatrixEntry {
	return protocolMatrixEntry{
		protocol:   "wireguard",
		tfName:     "mx_wg",
		port:       26007,
		minVersion: "v2.9.0", // mtu changed from int to list [v4, v6] in v2.9.0
		createHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_wg" {
  port     = %d
  protocol = "wireguard"
  remark   = "matrix-wg-create"
  enable   = true
  wireguard_settings {
    mtu = [1420, 1280]
    peer {
      public_key  = "dGVzdHB1YmxpY2tleXRlc3RwdWJsaWNrZXkxMjM0NQ=="
      allowed_ips = ["10.0.0.2/32"]
    }
  }
}
`, port)
		},
		updateHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_wg" {
  port     = %d
  protocol = "wireguard"
  remark   = "matrix-wg-updated"
  enable   = true
  wireguard_settings {
    mtu = [1400, 1280]
    peer {
      public_key  = "dGVzdHB1YmxpY2tleXRlc3RwdWJsaWNrZXkxMjM0NQ=="
      allowed_ips = ["10.0.0.2/32"]
    }
  }
}
`, port)
		},
		createChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-wg-create"),
				resource.TestCheckResourceAttr(addr, "wireguard_settings.mtu.0", "1420"),
			}
		},
		updateChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-wg-updated"),
				resource.TestCheckResourceAttr(addr, "wireguard_settings.mtu.0", "1400"),
			}
		},
		importStateVerifyIgnore: []string{
			"wireguard_settings.secret_key",
		},
	}
}

// matrixAmneziawg covers the v3.7.0 AmneziaWG protocol (#441).
//
// The configuration deliberately sets only a handful of server fields: the panel
// generates the keypair, the subnet and the whole randomised obfuscation set on
// save (internal/web/service/inbound_amneziawg.go:114-148), and the provider
// reads them back into state. Anything the panel generates is therefore excluded
// from ImportStateVerify — not because it drifts, but because a create-time
// generated secret is not reproducible from configuration.
//
// mtu is the field the update step changes: it is plain, has no cross-field
// constraint, and is one of the omitempty keys, so a change also exercises the
// strip-and-restore path.
func matrixAmneziawg() protocolMatrixEntry {
	return protocolMatrixEntry{
		protocol:   "amneziawg",
		tfName:     "mx_awg",
		port:       26011,
		minVersion: "v3.7.0",
		createHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_awg" {
  port     = %d
  protocol = "amneziawg"
  remark   = "matrix-awg-create"
  enable   = true
  amneziawg_settings {
    server {
      subnet_ip   = "10.9.1.0"
      subnet_cidr = 24
      mtu         = 1380
      primary_dns = "1.1.1.1"
    }
    clients {
      email       = "matrix-awg-peer@test.com"
      enable      = true
      allowed_ips = ["10.9.1.2/32"]
      # Required: the panel rejects a keyless AmneziaWG peer and does not
      # generate one on the inbound path.
      public_key = "dGVzdHB1YmxpY2tleXRlc3RwdWJsaWNrZXkxMjM0NQ=="
    }
  }
}
`, port)
		},
		updateHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_awg" {
  port     = %d
  protocol = "amneziawg"
  remark   = "matrix-awg-updated"
  enable   = true
  amneziawg_settings {
    server {
      subnet_ip   = "10.9.1.0"
      subnet_cidr = 24
      mtu         = 1400
      primary_dns = "1.1.1.1"
    }
    clients {
      email       = "matrix-awg-peer@test.com"
      enable      = true
      allowed_ips = ["10.9.1.2/32"]
      # Required: the panel rejects a keyless AmneziaWG peer and does not
      # generate one on the inbound path.
      public_key = "dGVzdHB1YmxpY2tleXRlc3RwdWJsaWNrZXkxMjM0NQ=="
    }
  }
}
`, port)
		},
		createChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-awg-create"),
				resource.TestCheckResourceAttr(addr, "amneziawg_settings.server.mtu", "1380"),
				resource.TestCheckResourceAttr(addr, "amneziawg_settings.server.subnet_ip", "10.9.1.0"),
				resource.TestCheckResourceAttr(addr, "amneziawg_settings.clients.0.email", "matrix-awg-peer@test.com"),
				// Generated server-side. `jc` must be checked for a non-zero value,
				// not merely for being set: a partial server block used to come back
				// with the whole obfuscation set zeroed, which is plain WireGuard
				// wearing AmneziaWG's name, and TestCheckResourceAttrSet passes on
				// "0".
				resource.TestCheckResourceAttrSet(addr, "amneziawg_settings.server.public_key"),
				resource.TestCheckResourceAttrWith(addr, "amneziawg_settings.server.jc", nonZeroAttr("jc")),
				resource.TestCheckResourceAttrWith(addr, "amneziawg_settings.server.jmin", nonZeroAttr("jmin")),
				resource.TestCheckResourceAttrWith(addr, "amneziawg_settings.server.s1", nonZeroAttr("s1")),
				resource.TestCheckResourceAttrWith(addr, "amneziawg_settings.server.h1", func(v string) error {
					if v == "" {
						return fmt.Errorf("h1 is blank: the inbound has no magic-header obfuscation")
					}
					return nil
				}),
			}
		},
		updateChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-awg-updated"),
				resource.TestCheckResourceAttr(addr, "amneziawg_settings.server.mtu", "1400"),
				// The peer must survive an inbound update rather than being
				// dropped or re-keyed.
				resource.TestCheckResourceAttr(addr, "amneziawg_settings.clients.0.email", "matrix-awg-peer@test.com"),
			}
		},
		importStateVerifyIgnore: []string{
			"amneziawg_settings.server.private_key",
			"amneziawg_settings.server.header_protection_key",
			"amneziawg_settings.clients.0.private_key",
		},
	}
}

func matrixDokodemo() protocolMatrixEntry {
	return protocolMatrixEntry{
		protocol:   "dokodemo-door",
		tfName:     "mx_dokodemo",
		port:       26008,
		maxVersion: "v3.2.0", // dokodemo-door renamed to "tunnel" upstream
		createHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_dokodemo" {
  port     = %d
  protocol = "dokodemo-door"
  remark   = "matrix-dokodemo-create"
  enable   = true
  dokodemo_settings {
    address         = "127.0.0.1"
    port            = 80
    network         = "tcp"
    follow_redirect = false
  }
}
`, port)
		},
		updateHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_dokodemo" {
  port     = %d
  protocol = "dokodemo-door"
  remark   = "matrix-dokodemo-updated"
  enable   = true
  dokodemo_settings {
    address         = "127.0.0.1"
    port            = 443
    network         = "tcp,udp"
    follow_redirect = false
  }
}
`, port)
		},
		createChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-dokodemo-create"),
				resource.TestCheckResourceAttr(addr, "dokodemo_settings.network", "tcp"),
			}
		},
		updateChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-dokodemo-updated"),
				resource.TestCheckResourceAttr(addr, "dokodemo_settings.network", "tcp,udp"),
				resource.TestCheckResourceAttr(addr, "dokodemo_settings.port", "443"),
			}
		},
	}
}

func matrixHysteria() protocolMatrixEntry {
	return protocolMatrixEntry{
		protocol: "hysteria",
		tfName:   "mx_hysteria",
		port:     26009,
		createHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_hysteria" {
  port     = %d
  protocol = "hysteria"
  remark   = "matrix-hysteria-create"
  enable   = true
  hysteria_settings {
    version = 2
  }
  stream_settings {
    network  = "hysteria"
  }
}
`, port)
		},
		updateHCL: func(port int) string {
			return fmt.Sprintf(`
resource "threexui_inbound" "mx_hysteria" {
  port     = %d
  protocol = "hysteria"
  remark   = "matrix-hysteria-updated"
  enable   = true
  hysteria_settings {
    version = 2
  }
  stream_settings {
    network  = "hysteria"
  }
  sniffing {
    enabled       = true
    dest_override = ["http", "tls"]
    metadata_only = false
    route_only    = false
  }
}
`, port)
		},
		createChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-hysteria-create"),
				resource.TestCheckResourceAttr(addr, "hysteria_settings.version", "2"),
			}
		},
		updateChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-hysteria-updated"),
				resource.TestCheckResourceAttr(addr, "hysteria_settings.version", "2"),
				resource.TestCheckResourceAttr(addr, "sniffing.enabled", "true"),
			}
		},
		// Client creation for hysteria is tested by TestAccInboundClientHysteria.
		// Skipped here to reduce SQLite pressure at the end of a long test run.
		hasClient: false,
	}
}

// nonZeroAttr fails when an obfuscation parameter reads back as 0 — the shape a
// partial server block produces when the panel is not allowed to generate the
// set (#441).
func nonZeroAttr(name string) func(string) error {
	return func(v string) error {
		if v == "" || v == "0" {
			return fmt.Errorf("%s is %q: AmneziaWG obfuscation is disabled, the inbound is plain WireGuard", name, v)
		}
		return nil
	}
}

// matrixTuic covers TUIC v5 (3x-ui v3.8.0+), terminated by the bundled
// tuic-server sidecar rather than xray-core. The certificate/private_key
// inline PEM pair is operator-supplied (the panel generates nothing), so the
// entry uses a throwaway self-signed certificate. The update step changes
// congestion_control and the client comment; peers belong to the inbound
// (protocolOwnsClients) and require id+password+email on every save.
func matrixTuic() protocolMatrixEntry {
	certHCL := func(port int, remark, congestionControl, comment string) string {
		certPEM, keyPEM := mustSelfSignedCertPEM()
		return fmt.Sprintf(`
resource "threexui_inbound" "mx_tuic" {
  port     = %d
  protocol = "tuic"
  remark   = %q
  enable   = true

  tuic_settings {
    server {
      certificate         = %q
      private_key         = %q
      congestion_control  = %q
      udp_relay_mode      = "native"
      max_idle_time       = 15
    }
    clients {
      email    = "matrix-tuic-user@test.com"
      id       = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
      password = "matrix-tuic-pass"
      comment  = %q
    }
  }
}
`, port, remark, certPEM, keyPEM, congestionControl, comment)
	}
	return protocolMatrixEntry{
		protocol:   "tuic",
		tfName:     "mx_tuic",
		port:       26012,
		minVersion: "v3.8.0",
		createHCL: func(port int) string {
			return certHCL(port, "matrix-tuic-create", "bbr", "matrix")
		},
		updateHCL: func(port int) string {
			return certHCL(port, "matrix-tuic-updated", "cubic", "matrix-2")
		},
		createChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-tuic-create"),
				resource.TestCheckResourceAttr(addr, "tuic_settings.server.congestion_control", "bbr"),
				resource.TestCheckResourceAttr(addr, "tuic_settings.server.max_idle_time", "15"),
				resource.TestCheckResourceAttr(addr, "tuic_settings.clients.0.email", "matrix-tuic-user@test.com"),
				resource.TestCheckResourceAttr(addr, "tuic_settings.clients.0.comment", "matrix"),
			}
		},
		updateChecks: func(addr string) []resource.TestCheckFunc {
			return []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(addr, "remark", "matrix-tuic-updated"),
				resource.TestCheckResourceAttr(addr, "tuic_settings.server.congestion_control", "cubic"),
				resource.TestCheckResourceAttr(addr, "tuic_settings.clients.0.comment", "matrix-2"),
			}
		},
	}
}
