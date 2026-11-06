package provider

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccXrayOutboundsXmuxConfig(xmux string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_xray_outbounds" "xmux" {
  outbound {
    tag      = "direct"
    protocol = "freedom"
  }
  outbound {
    tag      = "blocked"
    protocol = "blackhole"
  }
  outbound {
    tag      = "xhttp-upstream"
    protocol = "vless"
    vless_settings {
      address    = "127.0.0.1"
      port       = 443
      id         = "6c1b8f4e-0a8e-4d55-9f5a-2b7c1e3d4a90"
      encryption = "none"
    }
    stream_settings {
      network  = "xhttp"
      security = "tls"
      tls_settings {
        server_name = "localhost"
      }
      xhttp_settings {
        path = "/acc-xmux"
        mode = "stream-up"
%s
      }
    }
  }
}

data "threexui_xray_config" "xmux" {
  depends_on = [threexui_xray_outbounds.xmux]
}
`, xmux)
}

// On an outbound xmux configures the panel's own Xray, so it must reach the
// config Xray runs with, not only the template.
func TestAccXrayOutboundsXmux(t *testing.T) {
	const name = "threexui_xray_outbounds.xmux"
	connections := testAccXrayOutboundsXmuxConfig(`
        xmux {
          max_connections     = "2"
          h_max_request_times = "6000-9000"
          h_max_reusable_secs = "1800-3000"
          h_keep_alive_period = 30
        }
`)
	empty := testAccXrayOutboundsXmuxConfig(`
        xmux {}
`)
	removed := testAccXrayOutboundsXmuxConfig("")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: connections,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "outbound.2.stream_settings.xhttp_settings.xmux.max_connections", "2"),
					testAccCheckXrayOutboundXmux("data.threexui_xray_config.xmux", "xhttp-upstream", map[string]any{
						"maxConnections":   "2",
						"hMaxRequestTimes": "6000-9000",
						"hMaxReusableSecs": "1800-3000",
						"hKeepAlivePeriod": float64(30),
					}),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateId:     "xray_outbounds",
				ImportStateVerify: true,
			},
			{
				Config:             connections,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: empty,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(name, "outbound.2.stream_settings.xhttp_settings.xmux.max_connections"),
					testAccCheckXrayOutboundXmux("data.threexui_xray_config.xmux", "xhttp-upstream", map[string]any{}),
				),
			},
			{
				Config:             empty,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: removed,
				Check:  testAccCheckXrayOutboundXmux("data.threexui_xray_config.xmux", "xhttp-upstream", nil),
			},
			{
				Config:             removed,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// Checks xmux in the generated xray config; nil means the key must be absent.
func testAccCheckXrayOutboundXmux(dataSourceName, tag string, want map[string]any) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("%s not found in state", dataSourceName)
		}
		var config struct {
			Outbounds []struct {
				Tag            string `json:"tag"`
				StreamSettings struct {
					XHTTPSettings map[string]any `json:"xhttpSettings"`
				} `json:"streamSettings"`
			} `json:"outbounds"`
		}
		if err := json.Unmarshal([]byte(rs.Primary.Attributes["json"]), &config); err != nil {
			return fmt.Errorf("parse %s.json: %w", dataSourceName, err)
		}
		for _, ob := range config.Outbounds {
			if ob.Tag != tag {
				continue
			}
			got, present := ob.StreamSettings.XHTTPSettings["xmux"]
			switch {
			case want == nil && present:
				return fmt.Errorf("outbound %q still has xmux %v", tag, got)
			case want != nil && !reflect.DeepEqual(got, any(want)):
				return fmt.Errorf("outbound %q xmux = %#v, want %#v", tag, got, want)
			}
			return nil
		}
		return fmt.Errorf("outbound %q not found in generated xray config", tag)
	}
}
