package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccXrayV310Fields(t *testing.T) {
	requireMinVersion(t, "v3.10.0")
	config := func(id string, systemCA bool) string {
		return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_xray_dns" "v310" {
 script = ""
 server {
  id = %q
  address = "8.8.8.8"
 }
}
resource "threexui_xray_routing" "v310" {
 script = ""
 domain_strategy = "AsIs"
}
resource "threexui_xray_outbounds" "v310" {
 outbound {
  tag = "direct"
  protocol = "freedom"
 }
 outbound {
  tag = "blocked"
  protocol = "blackhole"
 }
 outbound {
  tag = "v310-upstream"
  protocol = "trojan"
  trojan_settings {
   address = "127.0.0.1"
   port = 443
   password = "acceptance-test-password"
  }
  stream_settings {
   network = "tcp"
   security = "tls"
   tls_settings {
    server_name = "localhost"
    use_system_ca = %t
   }
  }
 }
}
`, id, systemCA)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: config("resolver-a", true), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("threexui_xray_dns.v310", "script", ""),
				resource.TestCheckResourceAttr("threexui_xray_dns.v310", "server.0.id", "resolver-a"),
				resource.TestCheckResourceAttr("threexui_xray_routing.v310", "script", ""),
				resource.TestCheckResourceAttr("threexui_xray_outbounds.v310", "outbound.2.stream_settings.tls_settings.use_system_ca", "true"),
			)},
			{ResourceName: "threexui_xray_dns.v310", ImportState: true, ImportStateId: "xray_dns", ImportStateVerify: true},
			{ResourceName: "threexui_xray_routing.v310", ImportState: true, ImportStateId: "xray_routing", ImportStateVerify: true},
			{ResourceName: "threexui_xray_outbounds.v310", ImportState: true, ImportStateId: "xray_outbounds", ImportStateVerify: true},
			{Config: config("resolver-b", false), Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("threexui_xray_dns.v310", "server.0.id", "resolver-b"),
				resource.TestCheckResourceAttr("threexui_xray_outbounds.v310", "outbound.2.stream_settings.tls_settings.use_system_ca", "false"),
			)},
			{Config: config("resolver-b", false), PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}
