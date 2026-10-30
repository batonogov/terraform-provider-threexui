package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Values are safe for a listener inside the container: mark/tproxy/mptcp need
// CAP_NET_ADMIN or kernel support and would take xray down for other tests.
func testAccInboundSockoptConfig(sockopt string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_inbound" "sockopt" {
  port     = 25036
  protocol = "vless"
  remark   = "acc-sockopt"
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
    sockopt {
%s
    }
  }
}
`, sockopt)
}

func TestAccInboundSockopt(t *testing.T) {
	const name = "threexui_inbound.sockopt"
	full := testAccInboundSockoptConfig(`
      trusted_x_forwarded_for = ["X-Forwarded-For", "CF-Connecting-IP"]
      accept_proxy_protocol   = false
      tfo_enable              = true
      mark                    = 0
      tcp_keep_alive_interval = 30
      tcp_keep_alive_idle     = 300
      tcp_user_timeout        = 10000
      tcp_congestion          = "cubic"
      custom_sockopt = [{
        level = "1"
        opt   = "9"
        type  = "int"
        value = "1"
      }]
`)
	cleared := testAccInboundSockoptConfig(`
      trusted_x_forwarded_for = []
      accept_proxy_protocol   = false
      tfo_enable              = false
      mark                    = 0
      tcp_keep_alive_interval = 0
      tcp_keep_alive_idle     = 0
      tcp_user_timeout        = 0
      tcp_congestion          = ""
      custom_sockopt          = []
`)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: full,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "stream_settings.sockopt.trusted_x_forwarded_for.#", "2"),
					resource.TestCheckResourceAttr(name, "stream_settings.sockopt.trusted_x_forwarded_for.0", "X-Forwarded-For"),
					resource.TestCheckResourceAttr(name, "stream_settings.sockopt.tcp_keep_alive_idle", "300"),
					resource.TestCheckResourceAttr(name, "stream_settings.sockopt.tcp_congestion", "cubic"),
					resource.TestCheckResourceAttr(name, "stream_settings.sockopt.custom_sockopt.0.opt", "9"),
					testAccCheckInboundStoredSockopt(name, map[string]any{
						"trustedXForwardedFor": []any{"X-Forwarded-For", "CF-Connecting-IP"},
						"acceptProxyProtocol":  false,
						"tcpFastOpen":          true,
						"mark":                 float64(0),
						"tcpKeepAliveInterval": float64(30),
						"tcpKeepAliveIdle":     float64(300),
						"tcpUserTimeout":       float64(10000),
						"tcpcongestion":        "cubic",
						"customSockopt": []any{map[string]any{
							"level": "1", "opt": "9", "type": "int", "value": "1",
						}},
					}),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:             full,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: cleared,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "stream_settings.sockopt.trusted_x_forwarded_for.#", "0"),
					resource.TestCheckResourceAttr(name, "stream_settings.sockopt.custom_sockopt.#", "0"),
					resource.TestCheckResourceAttr(name, "stream_settings.sockopt.tcp_congestion", ""),
					testAccCheckInboundStoredSockopt(name, map[string]any{
						"trustedXForwardedFor": []any{},
						"acceptProxyProtocol":  false,
						"tcpFastOpen":          false,
						"mark":                 float64(0),
						"tcpKeepAliveInterval": float64(0),
						"tcpKeepAliveIdle":     float64(0),
						"tcpUserTimeout":       float64(0),
						"tcpcongestion":        "",
						"customSockopt":        []any{},
					}),
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

// Checks the JSON the panel actually stored, not the provider's state.
func testAccCheckInboundStoredSockopt(resourceName string, want map[string]any) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s not found in state", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("parse id %q: %w", rs.Primary.ID, err)
		}
		client, err := testAccClientFromEnv()
		if err != nil {
			return err
		}
		inbound, err := client.GetInbound(context.Background(), id)
		if err != nil {
			return err
		}
		var stream struct {
			Sockopt map[string]any `json:"sockopt"`
		}
		if err := json.Unmarshal([]byte(inbound.StreamSettings), &stream); err != nil {
			return fmt.Errorf("parse streamSettings %q: %w", inbound.StreamSettings, err)
		}
		if !reflect.DeepEqual(stream.Sockopt, want) {
			return fmt.Errorf("stored sockopt = %v, want %v", stream.Sockopt, want)
		}
		return nil
	}
}
