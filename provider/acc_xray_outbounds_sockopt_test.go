package provider

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccXrayOutboundsSockoptConfig(sockopt string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_xray_outbounds" "sockopt" {
  outbound {
    tag      = "direct"
    protocol = "freedom"
  }
  outbound {
    tag      = "blocked"
    protocol = "blackhole"
  }
  outbound {
    tag      = "chained"
    protocol = "trojan"
    trojan_settings {
      address  = "127.0.0.1"
      port     = 443
      password = "acceptance-test-password"
    }
    stream_settings {
      network  = "tcp"
      security = "tls"
      tls_settings {
        server_name = "localhost"
      }
      sockopt {
%s
      }
    }
  }
}

data "threexui_xray_config" "sockopt" {
  depends_on = [threexui_xray_outbounds.sockopt]
}
`, sockopt)
}

func TestAccXrayOutboundsSockopt(t *testing.T) {
	const name = "threexui_xray_outbounds.sockopt"
	full := testAccXrayOutboundsSockoptConfig(`
        dialer_proxy        = "direct"
        domain_strategy     = "UseIPv4"
        tcp_keep_alive_idle = 60
        happy_eyeballs = {
          try_delay_ms       = 250
          prioritize_ipv6    = false
          interleave         = 1
          max_concurrent_try = 4
        }
`)
	// Review of #480: the outbound list is authoritative in ModifyPlan, which
	// used to delete every attribute a declared sockopt block omits.
	omitted := testAccXrayOutboundsSockoptConfig(`
        tcp_keep_alive_idle = 90
`)
	cleared := testAccXrayOutboundsSockoptConfig(`
        dialer_proxy        = ""
        domain_strategy     = "AsIs"
        tcp_keep_alive_idle = 0
        happy_eyeballs = {
          try_delay_ms       = 0
          prioritize_ipv6    = false
          interleave         = 1
          max_concurrent_try = 4
        }
`)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: full,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "outbound.2.stream_settings.sockopt.dialer_proxy", "direct"),
					resource.TestCheckResourceAttr(name, "outbound.2.stream_settings.sockopt.happy_eyeballs.try_delay_ms", "250"),
					testAccCheckXrayOutboundSockopt("data.threexui_xray_config.sockopt", "chained", map[string]any{
						"dialerProxy":      "direct",
						"domainStrategy":   "UseIPv4",
						"tcpKeepAliveIdle": float64(60),
						"happyEyeballs": map[string]any{
							"tryDelayMs": float64(250), "prioritizeIPv6": false,
							"interleave": float64(1), "maxConcurrentTry": float64(4),
						},
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
				Config:             full,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: omitted,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "outbound.2.stream_settings.sockopt.dialer_proxy", "direct"),
					resource.TestCheckResourceAttr(name, "outbound.2.stream_settings.sockopt.tcp_keep_alive_idle", "90"),
					testAccCheckXrayOutboundSockopt("data.threexui_xray_config.sockopt", "chained", map[string]any{
						"dialerProxy":      "direct",
						"domainStrategy":   "UseIPv4",
						"tcpKeepAliveIdle": float64(90),
						"happyEyeballs": map[string]any{
							"tryDelayMs": float64(250), "prioritizeIPv6": false,
							"interleave": float64(1), "maxConcurrentTry": float64(4),
						},
					}),
				),
			},
			{
				Config:             omitted,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: testAccXrayOutboundsSockoptConfig(`
        tcp_keep_alive_idle     = -1
        tcp_keep_alive_interval = 30
`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Invalid TCP keep-alive settings`),
			},
			{
				Config: cleared,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "outbound.2.stream_settings.sockopt.dialer_proxy", ""),
					resource.TestCheckResourceAttr(name, "outbound.2.stream_settings.sockopt.happy_eyeballs.try_delay_ms", "0"),
				),
			},
			{
				Config:             cleared,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// Checks sockopt in the generated xray config, not the provider's state.
func testAccCheckXrayOutboundSockopt(dataSourceName, tag string, want map[string]any) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("%s not found in state", dataSourceName)
		}
		var config struct {
			Outbounds []struct {
				Tag            string `json:"tag"`
				StreamSettings struct {
					Sockopt map[string]any `json:"sockopt"`
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
			if !reflect.DeepEqual(ob.StreamSettings.Sockopt, want) {
				return fmt.Errorf("outbound %q sockopt = %v, want %v", tag, ob.StreamSettings.Sockopt, want)
			}
			return nil
		}
		return fmt.Errorf("outbound %q not found in generated xray config", tag)
	}
}
