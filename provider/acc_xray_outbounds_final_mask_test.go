package provider

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccXrayOutboundsFinalMaskConfig(finalMask string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_xray_outbounds" "final_mask" {
  outbound {
    tag      = "direct"
    protocol = "freedom"
  }
  outbound {
    tag      = "blocked"
    protocol = "blackhole"
  }
  outbound {
    tag      = "kcp-upstream"
    protocol = "vless"
    vless_settings {
      address    = "127.0.0.1"
      port       = 443
      id         = "6c1b8f4e-0a8e-4d55-9f5a-2b7c1e3d4a90"
      encryption = "none"
    }
    stream_settings {
      network  = "kcp"
      security = "tls"
      tls_settings {
        server_name = "localhost"
      }
%s
    }
  }
}

data "threexui_xray_config" "final_mask" {
  depends_on = [threexui_xray_outbounds.final_mask]
}
`, finalMask)
}

// On an outbound finalmask configures the panel's own Xray, so it must reach
// the config Xray runs with. mKCP rides UDP, so the salamander mask applies.
func TestAccXrayOutboundsFinalMask(t *testing.T) {
	const name = "threexui_xray_outbounds.final_mask"
	salamander := testAccXrayOutboundsFinalMaskConfig(`
      final_mask = jsonencode({
        udp = [{ type = "salamander", settings = { password = "acc-obfs-secret" } }]
      })
`)
	removed := testAccXrayOutboundsFinalMaskConfig("")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: salamander,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(name, "outbound.2.stream_settings.final_mask"),
					testAccCheckXrayOutboundFinalMask("data.threexui_xray_config.final_mask", "kcp-upstream", testAccSalamanderFinalMask),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateId:     "xray_outbounds",
				ImportStateVerify: true,
			},
			{
				Config:             salamander,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: removed,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(name, "outbound.2.stream_settings.final_mask"),
					testAccCheckXrayOutboundFinalMask("data.threexui_xray_config.final_mask", "kcp-upstream", nil),
				),
			},
			{
				Config:             removed,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// Checks finalmask in the generated xray config; nil means the key must be absent.
func testAccCheckXrayOutboundFinalMask(dataSourceName, tag string, want map[string]any) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("%s not found in state", dataSourceName)
		}
		var config struct {
			Outbounds []struct {
				Tag            string         `json:"tag"`
				StreamSettings map[string]any `json:"streamSettings"`
			} `json:"outbounds"`
		}
		if err := json.Unmarshal([]byte(rs.Primary.Attributes["json"]), &config); err != nil {
			return fmt.Errorf("parse %s.json: %w", dataSourceName, err)
		}
		for _, ob := range config.Outbounds {
			if ob.Tag != tag {
				continue
			}
			got, present := ob.StreamSettings["finalmask"]
			switch {
			case want == nil && present:
				return fmt.Errorf("outbound %q still has finalmask %v", tag, got)
			case want != nil && !reflect.DeepEqual(got, any(want)):
				return fmt.Errorf("outbound %q finalmask = %#v, want %#v", tag, got, want)
			}
			return nil
		}
		return fmt.Errorf("outbound %q not found in generated xray config", tag)
	}
}
