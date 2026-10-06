package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccCheckHostGroupDestroyed verifies that every threexui_host_group in
// the state has been deleted from the panel. GetHostGroup returns (nil, nil)
// when the group no longer exists, so a non-nil result means it survived.
func testAccCheckHostGroupDestroyed(state *terraform.State) error {
	client, err := testAccClientFromEnv()
	if err != nil {
		return fmt.Errorf("client init failed: %w", err)
	}
	for _, rs := range state.RootModule().Resources {
		if rs.Type != "threexui_host_group" {
			continue
		}
		groupID := rs.Primary.Attributes["group_id"]
		if groupID == "" {
			groupID = rs.Primary.ID
		}
		if groupID == "" {
			continue
		}
		hg, err := client.GetHostGroup(context.Background(), groupID)
		if err != nil {
			return fmt.Errorf("host group %s still exists or error reading: %w", groupID, err)
		}
		if hg != nil {
			return fmt.Errorf("host group %s still exists", groupID)
		}
	}
	return nil
}

// testAccHostGroupInboundConfig creates a minimal inbound so the host group has
// at least one inbound_id (Required, SizeAtLeast(1)).
func testAccHostGroupInboundConfig() string {
	return `
resource "threexui_inbound" "hg" {
  port     = 26001
  protocol = "vless"
  remark   = "acc-hg-inbound"
  enable   = true
  vless_settings {
    decryption = "none"
  }
}
`
}

// --- Create + update remark ---

func TestAccHostGroupBasic(t *testing.T) {
	requireMinVersion(t, "v3.5.0")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckHostGroupDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccHostGroupInboundConfig() + `
resource "threexui_host_group" "test" {
  remark      = "acc-host-group-1"
  hosts       = ["example.com"]
  sort_order  = 1
  inbound_ids = [threexui_inbound.hg.id]
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_host_group.test", "group_id"),
					resource.TestCheckResourceAttrSet("threexui_host_group.test", "id"),
					resource.TestCheckResourceAttr("threexui_host_group.test", "remark", "acc-host-group-1"),
					resource.TestCheckResourceAttr("threexui_host_group.test", "hosts.#", "1"),
					resource.TestCheckResourceAttr("threexui_host_group.test", "hosts.0", "example.com"),
					resource.TestCheckResourceAttr("threexui_host_group.test", "sort_order", "1"),
				),
			},
			// Update remark
			{
				Config: testAccProviderConfig() + testAccHostGroupInboundConfig() + `
resource "threexui_host_group" "test" {
  remark      = "acc-host-group-2"
  hosts       = ["example.com"]
  sort_order  = 1
  inbound_ids = [threexui_inbound.hg.id]
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_host_group.test", "remark", "acc-host-group-2"),
					resource.TestCheckResourceAttr("threexui_host_group.test", "hosts.0", "example.com"),
				),
			},
		},
	})
}

// --- Host group referencing an inbound, verify inbound_ids ---

func TestAccHostGroupWithInboundIDs(t *testing.T) {
	requireMinVersion(t, "v3.5.0")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckHostGroupDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccHostGroupInboundConfig() + `
resource "threexui_host_group" "test" {
  remark      = "acc-host-group-inbound"
  hosts       = ["node1.example.com", "node2.example.com"]
  sort_order  = 5
  inbound_ids = [threexui_inbound.hg.id]
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("threexui_host_group.test", "group_id"),
					resource.TestCheckResourceAttr("threexui_host_group.test", "remark", "acc-host-group-inbound"),
					resource.TestCheckResourceAttr("threexui_host_group.test", "inbound_ids.#", "1"),
				),
			},
		},
	})
}

// --- cipher_suites (v3.9.0+) ---

func TestAccHostGroupCipherSuites_v390(t *testing.T) {
	requireMinVersion(t, "v3.9.0")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckHostGroupDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccHostGroupInboundConfig() + `
resource "threexui_host_group" "test" {
  remark        = "acc-host-group-ciphers"
  hosts         = ["cipher.example.com"]
  security      = "tls"
  cipher_suites = "TLS_AES_128_GCM_SHA256:TLS_CHACHA20_POLY1305_SHA256"
  inbound_ids   = [threexui_inbound.hg.id]
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_host_group.test", "cipher_suites", "TLS_AES_128_GCM_SHA256:TLS_CHACHA20_POLY1305_SHA256"),
				),
			},
			{
				Config: testAccProviderConfig() + testAccHostGroupInboundConfig() + `
resource "threexui_host_group" "test" {
  remark        = "acc-host-group-ciphers"
  hosts         = ["cipher.example.com"]
  security      = "tls"
  cipher_suites = "TLS_AES_128_GCM_SHA256:TLS_CHACHA20_POLY1305_SHA256"
  inbound_ids   = [threexui_inbound.hg.id]
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// --- Import by group_id ---
func TestAccHostGroupImport(t *testing.T) {
	requireMinVersion(t, "v3.5.0")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckHostGroupDestroyed,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccHostGroupInboundConfig() + `
resource "threexui_host_group" "test" {
  remark      = "acc-host-group-import"
  hosts       = ["import.example.com"]
  sort_order  = 2
  inbound_ids = [threexui_inbound.hg.id]
}
`,
			},
			{
				ResourceName:      "threexui_host_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
