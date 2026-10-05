package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccInboundTuicClientLifecycle is the TUIC counterpart of
// TestAccInboundAmneziawgPeerLifecycle: users in tuic_settings.clients[] must
// be addable, editable and removable — including removal-to-zero — on every
// panel that has the protocol. Since 3x-ui v3.9.0 the inbound update no
// longer persists the posted clients (keepStoredClients), so the provider
// reconciles peers through the /panel/api/clients/* endpoints; on older
// panels the wholesale update does it and the reconciliation is a no-op.
//
// TUIC clients need id+password+email on every save — the client endpoints
// enforce the same triplet as the inbound path
// (client_inbound_apply.go:464-473) — so the fixture sets all three
// explicitly, plus a throwaway self-signed server certificate.
func TestAccInboundTuicClientLifecycle(t *testing.T) {
	requireMinVersion(t, "v3.8.0")

	const port = 26023
	certPEM, keyPEM := testAccSelfSignedCertPEM(t)
	config := func(peers string) string {
		return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_inbound" "tuic_peers" {
  port     = %d
  protocol = "tuic"
  remark   = "acc-tuic-peers"
  enable   = true

  tuic_settings {
    server {
      certificate = %q
      private_key = %q
    }
%s
  }
}
`, port, certPEM, keyPEM, peers)
	}

	peerA := func(comment string) string {
		return fmt.Sprintf(`
    clients {
      email    = "tuic-life-a@test.com"
      id       = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
      password = "tuic-life-pass-a"
      enable   = true
      comment  = %q
    }`, comment)
	}
	peerB := `
    clients {
      email    = "tuic-life-b@test.com"
      id       = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
      password = "tuic-life-pass-b"
      enable   = true
    }`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: config(peerA("one")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.tuic_peers", "tuic_settings.clients.#", "1"),
					resource.TestCheckResourceAttr("threexui_inbound.tuic_peers", "tuic_settings.clients.0.comment", "one"),
				),
			},
			{
				// Edit peer a AND add peer b in one apply.
				Config: config(peerA("two") + peerB),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.tuic_peers", "tuic_settings.clients.#", "2"),
					resource.TestCheckResourceAttr("threexui_inbound.tuic_peers", "tuic_settings.clients.0.comment", "two"),
					resource.TestCheckResourceAttr("threexui_inbound.tuic_peers", "tuic_settings.clients.1.email", "tuic-life-b@test.com"),
				),
			},
			{
				// Partial removal: peer b goes, peer a stays.
				Config: config(peerA("two")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.tuic_peers", "tuic_settings.clients.#", "1"),
					resource.TestCheckResourceAttr("threexui_inbound.tuic_peers", "tuic_settings.clients.0.email", "tuic-life-a@test.com"),
				),
			},
			{
				// Removal-to-zero.
				Config: config(""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_inbound.tuic_peers", "tuic_settings.clients.#", "0"),
				),
			},
			{
				Config:   config(""),
				PlanOnly: true,
			},
		},
	})
}

// TestAccInboundTuicClientEditMaterialisesSubIdAndUpdatedAt pins the
// v3.9.0-only shape of a TUIC peer edit. A peer created through the inbound
// path stores a blank subId; saving the edit through the client endpoint
// regenerates it (client_inbound_apply.go:825-830) and stamps a fresh
// updated_at (:847-853). The apply only stays consistent because ModifyPlan
// plans both attributes as unknown for edited peers. On ≤ v3.8.5 the
// wholesale update preserves the posted blank subId and timestamp, so these
// assertions would not hold there.
func TestAccInboundTuicClientEditMaterialisesSubIdAndUpdatedAt(t *testing.T) {
	requireMinVersion(t, "v3.9.0")

	const port = 26024
	certPEM, keyPEM := testAccSelfSignedCertPEM(t)
	config := func(comment string) string {
		return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_inbound" "tuic_stamps" {
  port     = %d
  protocol = "tuic"
  remark   = "acc-tuic-stamps"
  enable   = true

  tuic_settings {
    server {
      certificate = %q
      private_key = %q
    }
    clients {
      email    = "tuic-stamp@test.com"
      id       = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
      password = "tuic-stamp-pass"
      enable   = true
      comment  = %q
    }
  }
}
`, port, certPEM, keyPEM, comment)
	}

	var updatedAtAfterCreate string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config("one"),
				Check: resource.ComposeTestCheckFunc(
					// Created through the inbound path: subId stays blank.
					resource.TestCheckResourceAttr("threexui_inbound.tuic_stamps", "tuic_settings.clients.0.sub_id", ""),
					resource.TestCheckResourceAttrWith("threexui_inbound.tuic_stamps", "tuic_settings.clients.0.updated_at", func(v string) error {
						if v == "" || v == "0" {
							return fmt.Errorf("expected the panel to stamp updated_at on create, got %q", v)
						}
						updatedAtAfterCreate = v
						return nil
					}),
				),
			},
			{
				Config: config("two"),
				Check: resource.ComposeTestCheckFunc(
					// The endpoint save regenerated subId and bumped updated_at.
					resource.TestCheckResourceAttrWith("threexui_inbound.tuic_stamps", "tuic_settings.clients.0.sub_id", func(v string) error {
						if v == "" {
							return fmt.Errorf("expected the endpoint save to generate a sub_id, still blank")
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("threexui_inbound.tuic_stamps", "tuic_settings.clients.0.updated_at", func(v string) error {
						if v == updatedAtAfterCreate {
							return fmt.Errorf("expected the endpoint save to bump updated_at, still %q", v)
						}
						return nil
					}),
				),
			},
			{
				Config:   config("two"),
				PlanOnly: true,
			},
		},
	})
}
