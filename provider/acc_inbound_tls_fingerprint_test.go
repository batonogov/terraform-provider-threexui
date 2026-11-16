package provider

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccInboundTLSFingerprint(t *testing.T) {
	cert, key := testAccSelfSignedCertPEM(t)
	for _, protocol := range []string{"vless", "hysteria"} {
		t.Run(protocol, func(t *testing.T) {
			const name = "threexui_inbound.tls_fingerprint"
			config := func(fingerprint string) string {
				network, settings, transport := "tcp", `vless_settings { decryption = "none" }`, ""
				if protocol == "hysteria" {
					network = "hysteria"
					settings = "hysteria_settings { version = 2 }"
					transport = settings
				}
				return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_inbound" "tls_fingerprint" {
  port = 25522
  protocol = %q
  remark = "acc-tls-fingerprint"
  enable = true
  %s
  stream_settings {
    network = %q
    security = "tls"
    %s
    tls_settings {
      server_name = "acc-tls.example.com"
      fingerprint = %q
      certificates = [{
        certificate = %s
        key = %s
      }]
    }
  }
}
`, protocol, settings, network, transport, fingerprint,
					hclStringList(strings.Split(strings.TrimSpace(cert), "\n")),
					hclStringList(strings.Split(strings.TrimSpace(key), "\n")))
			}
			check := func(want string) resource.TestCheckFunc {
				return resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "stream_settings.tls_settings.fingerprint", want),
					func(s *terraform.State) error {
						id, err := strconv.Atoi(s.RootModule().Resources[name].Primary.ID)
						if err != nil {
							return err
						}
						client, err := testAccClientFromEnv()
						if err != nil {
							return err
						}
						stream, err := testAccStoredStream(client, id)
						if err != nil {
							return err
						}
						tls, _ := stream["tlsSettings"].(map[string]any)
						settings, _ := tls["settings"].(map[string]any)
						if settings["fingerprint"] != want {
							return fmt.Errorf("stored TLS client settings = %#v, want fingerprint %q", settings, want)
						}
						if _, ok := tls["fingerprint"]; ok {
							return fmt.Errorf("stored inbound has a root TLS fingerprint: %#v", tls)
						}
						return nil
					},
				)
			}
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
				CheckDestroy:             testAccCheckInboundDestroyed,
				Steps: []resource.TestStep{
					{Config: config("firefox"), Check: check("firefox")},
					{Config: config("firefox"), PlanOnly: true},
					{ResourceName: name, ImportState: true, ImportStateVerify: true},
					{
						// Simulate a resource stored by the old provider, then update it.
						PreConfig: func() {
							testAccStoreInboundStream(t, 25522, func(stream map[string]any) {
								tls := stream["tlsSettings"].(map[string]any)
								tls["fingerprint"] = "firefox"
								delete(tls, "settings")
							})
						},
						Config: config("chrome"),
						Check:  check("chrome"),
					},
					{
						// A panel-side edit must be read from the nested fingerprint.
						PreConfig: func() {
							testAccStoreInboundStream(t, 25522, func(stream map[string]any) {
								tls := stream["tlsSettings"].(map[string]any)
								tls["settings"] = map[string]any{"fingerprint": "safari"}
							})
						},
						Config:             config("chrome"),
						PlanOnly:           true,
						ExpectNonEmptyPlan: true,
					},
					{Config: config("chrome"), Check: check("chrome")},
					{Config: config("chrome"), PlanOnly: true},
				},
			})
		})
	}
}
