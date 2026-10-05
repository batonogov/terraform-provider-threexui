package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccInboundAmneziawgKeysSurviveUnrelatedUpdate is the regression guard for
// the sharpest failure mode of the AmneziaWG surface (#441).
//
// 3x-ui regenerates the entire server block — a fresh keypair included —
// whenever an inbound is saved with settings that carry no `server` object
// (normalizeAmneziaWGSettings, internal/web/service/inbound_amneziawg.go:171-200).
// That runs on UpdateInbound too, so anything that lets the provider send an
// empty blob on a later apply silently rotates the server keys and invalidates
// every peer config already distributed. Confirmed against a live v3.7.0 panel:
// posting `settings = {}` on update returns a different publicKey than create
// did.
//
// The schema-level defence is amneziawgServerRequiredValidator, which refuses a
// configuration without the block. This test covers the other half: that when
// the block IS declared but every attribute is left to the panel, the values it
// generated on create survive an unrelated edit. That works only because the
// server attributes are Optional+Computed with UseStateForUnknown, so the plan
// replays what Read recorded — a regression in any of those three would rotate
// the keys here.
func TestAccInboundAmneziawgKeysSurviveUnrelatedUpdate(t *testing.T) {
	requireMinVersion(t, "v3.7.0")

	const port = 26012
	config := func(remark string) string {
		return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_inbound" "awg_keys" {
  port     = %d
  protocol = "amneziawg"
  remark   = %q
  enable   = true

  # Everything is left to the panel: this is the shape that regenerates the
  # server block if the provider ever sends an empty settings blob.
  amneziawg_settings {
    server {}
  }
}
`, port, remark)
	}

	var publicKeyAfterCreate, privateKeyAfterCreate string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config("acc-awg-keys"),
				Check: resource.ComposeTestCheckFunc(
					// The panel must have filled the block in.
					resource.TestCheckResourceAttrSet("threexui_inbound.awg_keys", "amneziawg_settings.server.public_key"),
					resource.TestCheckResourceAttrSet("threexui_inbound.awg_keys", "amneziawg_settings.server.private_key"),
					resource.TestCheckResourceAttrSet("threexui_inbound.awg_keys", "amneziawg_settings.server.jc"),
					resource.TestCheckResourceAttrWith("threexui_inbound.awg_keys", "amneziawg_settings.server.public_key", func(v string) error {
						publicKeyAfterCreate = v
						return nil
					}),
					resource.TestCheckResourceAttrWith("threexui_inbound.awg_keys", "amneziawg_settings.server.private_key", func(v string) error {
						privateKeyAfterCreate = v
						return nil
					}),
				),
			},
			{
				// Only the remark changes. Nothing about the tunnel should move.
				Config: config("acc-awg-keys-renamed"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.awg_keys", "remark", "acc-awg-keys-renamed"),
					resource.TestCheckResourceAttrWith("threexui_inbound.awg_keys", "amneziawg_settings.server.public_key", func(v string) error {
						if v != publicKeyAfterCreate {
							return fmt.Errorf("server public key rotated on an unrelated update: %q -> %q; "+
								"every peer configuration handed out before this apply is now invalid",
								publicKeyAfterCreate, v)
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("threexui_inbound.awg_keys", "amneziawg_settings.server.private_key", func(v string) error {
						if v != privateKeyAfterCreate {
							return fmt.Errorf("server private key rotated on an unrelated update")
						}
						return nil
					}),
				),
			},
			{
				// And the whole thing must be driftless afterwards.
				Config:   config("acc-awg-keys-renamed"),
				PlanOnly: true,
			},
		},
	})
}

// TestAccInboundAmneziawgPartialServerKeepsObfuscation guards the reason
// AmneziaWG exists at all.
//
// 3x-ui generates its randomised obfuscation set only when the settings it
// receives carry NO `server` object; a partial block is taken literally and
// every omitted field is stored as its zero value. An inbound configured with
// just a subnet therefore ends up with jc=0, blank h1-h4 and no header
// protection — plain WireGuard, trivially fingerprinted, with nothing in the
// panel or in Terraform reporting it. Measured directly against v3.7.0.
//
// The provider works around it by creating the inbound without the block and
// applying the configured fields afterwards (splitAmneziawgServer /
// applyAmneziawgServerOverrides). This test pins both halves: the configured
// values must win, and the generated ones must survive.
func TestAccInboundAmneziawgPartialServerKeepsObfuscation(t *testing.T) {
	requireMinVersion(t, "v3.7.0")

	const port = 26013
	config := testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_inbound" "awg_partial" {
  port     = %d
  protocol = "amneziawg"
  remark   = "acc-awg-partial"
  enable   = true

  amneziawg_settings {
    server {
      subnet_ip   = "10.9.2.0"
      subnet_cidr = 24
      primary_dns = "1.1.1.1"
    }
  }
}
`, port)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					// Configured fields win.
					resource.TestCheckResourceAttr("threexui_inbound.awg_partial", "amneziawg_settings.server.subnet_ip", "10.9.2.0"),
					resource.TestCheckResourceAttr("threexui_inbound.awg_partial", "amneziawg_settings.server.primary_dns", "1.1.1.1"),
					// Generated fields survive.
					resource.TestCheckResourceAttrWith("threexui_inbound.awg_partial", "amneziawg_settings.server.jc", nonZeroAttr("jc")),
					resource.TestCheckResourceAttrWith("threexui_inbound.awg_partial", "amneziawg_settings.server.jmin", nonZeroAttr("jmin")),
					resource.TestCheckResourceAttrWith("threexui_inbound.awg_partial", "amneziawg_settings.server.jmax", nonZeroAttr("jmax")),
					resource.TestCheckResourceAttrWith("threexui_inbound.awg_partial", "amneziawg_settings.server.s1", nonZeroAttr("s1")),
					resource.TestCheckResourceAttrWith("threexui_inbound.awg_partial", "amneziawg_settings.server.s2", nonZeroAttr("s2")),
					resource.TestCheckResourceAttrWith("threexui_inbound.awg_partial", "amneziawg_settings.server.h1", func(v string) error {
						if v == "" {
							return fmt.Errorf("h1 is blank: magic-header obfuscation was not generated")
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("threexui_inbound.awg_partial", "amneziawg_settings.server.header_protection_key", func(v string) error {
						if v == "" {
							return fmt.Errorf("header_protection_key is blank: header protection was not generated")
						}
						return nil
					}),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccInboundAmneziawgRemoveLastPeer covers peer removal, which is only
// possible because preserveInboundSettings skips protocols whose clients the
// inbound owns.
//
// For vmess/vless/… the provider re-injects the existing clients[] on update,
// since those peers belong to threexui_inbound_client and must not be clobbered
// by an inbound-level write. AmneziaWG peers are owned by the inbound, so the
// same behaviour would put a deleted peer straight back: the apply fails with
// "block count changed from 0 to 1" and the peer keeps connecting — a removal
// that reports failure while leaving access intact.
func TestAccInboundAmneziawgRemoveLastPeer(t *testing.T) {
	requireMinVersion(t, "v3.7.0")

	const port = 26014
	base := `
resource "threexui_inbound" "awg_peers" {
  port     = %d
  protocol = "amneziawg"
  remark   = "acc-awg-peers"
  enable   = true

  amneziawg_settings {
    server {}
%s
  }
}
`
	peer := `
    clients {
      email       = "awg-last-peer@test.com"
      enable      = true
      public_key  = "dGVzdHB1YmxpY2tleXRlc3RwdWJsaWNrZXkxMjM0NQ=="
      allowed_ips = ["10.8.1.5/32"]
    }`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(base, port, peer),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.awg_peers", "amneziawg_settings.clients.#", "1"),
					resource.TestCheckResourceAttr("threexui_inbound.awg_peers", "amneziawg_settings.clients.0.email", "awg-last-peer@test.com"),
				),
			},
			{
				// The only peer is removed. It must actually go away.
				Config: testAccProviderConfig() + fmt.Sprintf(base, port, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.awg_peers", "amneziawg_settings.clients.#", "0"),
				),
			},
			{
				// ...and its email must be free again. Dropping a peer is an
				// update, not a delete, and the panel keeps the client row just as
				// it does when the whole inbound goes — so a second inbound reusing
				// the address fails with "Duplicate email" unless the provider
				// released it (#452).
				Config: testAccProviderConfig() + fmt.Sprintf(base, port, "") + `
resource "threexui_inbound" "awg_peers_reuse" {
  port     = 26016
  protocol = "amneziawg"
  remark   = "acc-awg-peers-reuse"
  enable   = true

  amneziawg_settings {
    server {}
    clients {
      email       = "awg-last-peer@test.com"
      enable      = true
      public_key  = "dGVzdHB1YmxpY2tleXRlc3RwdWJsaWNrZXkxMjM0NQ=="
      allowed_ips = ["10.8.1.6/32"]
    }
  }
}
`,
				Check: resource.TestCheckResourceAttr("threexui_inbound.awg_peers_reuse",
					"amneziawg_settings.clients.0.email", "awg-last-peer@test.com"),
			},
		},
	})
}

// TestAccInboundAmneziawgRecreateWithSamePeerEmail is the regression guard for
// #452: destroy followed by apply, which is what a `-replace`, a moved resource
// or a rebuilt workspace does.
//
// 3x-ui's DelInbound drops the inbound-to-client links but keeps the `clients`
// rows, which carry the unique index on `email`. Peers owned by the inbound
// (AmneziaWG, and WireGuard `clients[]` since v3.4.2) are therefore left
// occupying their address, and the next create fails with
// "Duplicate email: <address>" naming a client that no longer appears under any
// inbound. The provider now deletes each peer through the per-client endpoint
// before removing the inbound.
//
// terraform-plugin-testing destroys only at the end of a test case, so the
// second step taints the resource to force a destroy+create in the middle. Both
// steps deliberately use the SAME peer email: the destroy has to leave it free
// for the create that follows.
func TestAccInboundAmneziawgRecreateWithSamePeerEmail(t *testing.T) {
	requireMinVersion(t, "v3.7.0")

	const port = 26015
	config := func(remark string) string {
		return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_inbound" "awg_recreate" {
  port     = %d
  protocol = "amneziawg"
  remark   = %q
  enable   = true

  amneziawg_settings {
    server {}
    clients {
      email       = "awg-recreated@test.com"
      enable      = true
      public_key  = "dGVzdHB1YmxpY2tleXRlc3RwdWJsaWNrZXkxMjM0NQ=="
      allowed_ips = ["10.8.1.7/32"]
    }
  }
}
`, port, remark)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config("acc-awg-recreate-first"),
				Check: resource.TestCheckResourceAttr("threexui_inbound.awg_recreate",
					"amneziawg_settings.clients.0.email", "awg-recreated@test.com"),
			},
			{
				// Forces destroy + create of the inbound with the same peer email.
				Taint:  []string{"threexui_inbound.awg_recreate"},
				Config: config("acc-awg-recreate-second"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.awg_recreate", "remark", "acc-awg-recreate-second"),
					resource.TestCheckResourceAttr("threexui_inbound.awg_recreate",
						"amneziawg_settings.clients.0.email", "awg-recreated@test.com"),
				),
			},
		},
	})
}

// TestAccInboundAmneziawgPeerLifecycle exercises the full peer lifecycle in
// one test — add, edit, partial removal and removal-to-zero — because each
// transition goes through a different code path since 3x-ui v3.9.0.
//
// Until v3.8.5 an UpdateInbound persisted the posted settings.clients
// wholesale; since v3.9.0 the panel silently replaces the posted array with
// the stored one (keepStoredClients,
// 3x-ui-3.9.0/internal/web/service/inbound.go:1851-1862), so the provider
// reconciles peers one by one through the /panel/api/clients/* endpoints
// (reconcileInboundOwnedPeers). On ≤ v3.8.5 that reconciliation is a no-op
// and the wholesale path does the work, so this test passes unchanged on
// every panel that has the protocol.
//
// The partial-removal step doubles as a regression test for a deadlock: the
// update holds inboundClientMu whenever the plan still has peers, and the
// orphan-email cleanup afterwards takes the same mutex — an earlier
// deferred-unlock structure held it across both and blocked forever.
func TestAccInboundAmneziawgPeerLifecycle(t *testing.T) {
	requireMinVersion(t, "v3.7.0")

	const port = 26020
	config := func(peers string) string {
		return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_inbound" "awg_lifecycle" {
  port     = %d
  protocol = "amneziawg"
  remark   = "acc-awg-lifecycle"
  enable   = true

  amneziawg_settings {
    server {}
%s
  }
}
`, port, peers)
	}

	peerA := func(comment string) string {
		return fmt.Sprintf(`
    clients {
      email       = "awg-life-a@test.com"
      enable      = true
      public_key  = "dGVzdHB1YmxpY2tleXRlc3RwdWJsaWNrZXkxMjM0NQ=="
      allowed_ips = ["10.8.1.10/32"]
      comment     = %q
    }`, comment)
	}
	peerB := `
    clients {
      email       = "awg-life-b@test.com"
      enable      = true
      public_key  = "cHB1YmxpY2tleXR3b3B1YmxpY2tleXR3b3B1YmxpY2tleQ=="
      allowed_ips = ["10.8.1.11/32"]
    }`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config(peerA("one")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.awg_lifecycle", "amneziawg_settings.clients.#", "1"),
					resource.TestCheckResourceAttr("threexui_inbound.awg_lifecycle", "amneziawg_settings.clients.0.comment", "one"),
				),
			},
			{
				// Edit peer a AND add peer b in one apply.
				Config: config(peerA("two") + peerB),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.awg_lifecycle", "amneziawg_settings.clients.#", "2"),
					resource.TestCheckResourceAttr("threexui_inbound.awg_lifecycle", "amneziawg_settings.clients.0.comment", "two"),
					resource.TestCheckResourceAttr("threexui_inbound.awg_lifecycle", "amneziawg_settings.clients.1.email", "awg-life-b@test.com"),
				),
			},
			{
				// Partial removal: peer b goes, peer a stays.
				Config: config(peerA("two")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.awg_lifecycle", "amneziawg_settings.clients.#", "1"),
					resource.TestCheckResourceAttr("threexui_inbound.awg_lifecycle", "amneziawg_settings.clients.0.email", "awg-life-a@test.com"),
				),
			},
			{
				// Removal-to-zero: the historically nasty case (see the
				// "block count changed from 0 to 1" gotcha).
				Config: config(""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.awg_lifecycle", "amneziawg_settings.clients.#", "0"),
				),
			},
			{
				// Peer b's email was freed by the removal, so reusing it must
				// not fail with "Duplicate email".
				Config: config(peerB),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.awg_lifecycle", "amneziawg_settings.clients.#", "1"),
					resource.TestCheckResourceAttr("threexui_inbound.awg_lifecycle", "amneziawg_settings.clients.0.email", "awg-life-b@test.com"),
				),
			},
			{
				Config:   config(peerB),
				PlanOnly: true,
			},
		},
	})
}

// TestAccInboundAmneziawgPeerEditBumpsUpdatedAt pins the v3.9.0-only shape of
// a peer edit: saved through the client endpoint, the panel stamps a fresh
// updated_at on the peer (client_inbound_apply.go) instead of preserving the
// posted one. The apply only succeeds because ModifyPlan plans updated_at as
// unknown for edited peers — without it Terraform rejects the bumped value as
// an inconsistent result. On ≤ v3.8.5 the wholesale update preserves the
// posted timestamp, so the assertion would not hold there.
func TestAccInboundAmneziawgPeerEditBumpsUpdatedAt(t *testing.T) {
	requireMinVersion(t, "v3.9.0")

	const port = 26021
	config := func(comment string) string {
		return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_inbound" "awg_stamps" {
  port     = %d
  protocol = "amneziawg"
  remark   = "acc-awg-stamps"
  enable   = true

  amneziawg_settings {
    server {}
    clients {
      email       = "awg-stamp@test.com"
      enable      = true
      public_key  = "dGVzdHB1YmxpY2tleXRlc3RwdWJsaWNrZXkxMjM0NQ=="
      allowed_ips = ["10.8.1.12/32"]
      comment     = %q
    }
  }
}
`, port, comment)
	}

	var updatedAtAfterCreate string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config("one"),
				Check: resource.TestCheckResourceAttrWith("threexui_inbound.awg_stamps", "amneziawg_settings.clients.0.updated_at", func(v string) error {
					if v == "" || v == "0" {
						return fmt.Errorf("expected the panel to stamp updated_at on create, got %q", v)
					}
					updatedAtAfterCreate = v
					return nil
				}),
			},
			{
				Config: config("two"),
				Check: resource.TestCheckResourceAttrWith("threexui_inbound.awg_stamps", "amneziawg_settings.clients.0.updated_at", func(v string) error {
					if v == updatedAtAfterCreate {
						return fmt.Errorf("expected the endpoint save to bump updated_at, still %q", v)
					}
					return nil
				}),
			},
			{
				Config:   config("two"),
				PlanOnly: true,
			},
		},
	})
}
