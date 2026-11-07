package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	testAccFinalMaskInbound = "threexui_inbound.final_mask"
	testAccFinalMaskPort    = 25041
)

var testAccSalamanderFinalMask = map[string]any{
	"udp": []any{map[string]any{"type": "salamander", "settings": map[string]any{"password": "acc-obfs-secret"}}},
}

func testAccInboundFinalMaskConfig(remark, hysteria, stream string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_inbound" "final_mask" {
  port     = %d
  protocol = "hysteria"
  remark   = %q
  enable   = true

  hysteria_settings {
    version = 2
  }

  stream_settings {
    network = "hysteria"
%s
    hysteria_settings {
      version = 2
%s
    }
  }
}
`, testAccFinalMaskPort, remark, stream, hysteria)
}

// Both fields used to be rebuilt away by every update of the inbound; the
// remark change in step 4 is that update.
func TestAccInboundFinalMaskMasquerade(t *testing.T) {
	const name = testAccFinalMaskInbound
	finalMask := `
    final_mask = jsonencode({
      udp = [{ type = "salamander", settings = { password = "acc-obfs-secret" } }]
    })
`
	masquerade := `
      masquerade {
        type = "file"
        dir  = "/var/www/acc-site"
      }
`
	full := testAccInboundFinalMaskConfig("acc-final-mask-1", masquerade, finalMask)
	renamed := testAccInboundFinalMaskConfig("acc-final-mask-2", masquerade, finalMask)
	removed := testAccInboundFinalMaskConfig("acc-final-mask-2", "", "")
	storedMasquerade := map[string]any{"type": "file", "dir": "/var/www/acc-site"}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				// xray-core loads a shorter password and then fails to open the
				// socket; the provider rejects it on every panel version. First,
				// because the post-test destroy runs with the last step's config.
				Config: testAccInboundFinalMaskConfig("acc-final-mask-1", "", `
    final_mask = jsonencode({ udp = [{ type = "salamander", settings = { password = "abc" } }] })
`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`salamander password must be at least 4 bytes`),
			},
			{
				Config: full,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "stream_settings.hysteria_settings.masquerade.type", "file"),
					resource.TestCheckResourceAttr(name, "stream_settings.hysteria_settings.masquerade.dir", "/var/www/acc-site"),
					resource.TestCheckNoResourceAttr(name, "stream_settings.hysteria_settings.masquerade.url"),
					testAccCheckInboundStoredStream(testAccSalamanderFinalMask, storedMasquerade),
					testAccCheckGeneratedInboundStream(testAccFinalMaskPort, testAccSalamanderFinalMask, storedMasquerade),
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
				Config: renamed,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "remark", "acc-final-mask-2"),
					testAccCheckInboundStoredStream(testAccSalamanderFinalMask, storedMasquerade),
				),
			},
			{
				// The 3x-ui form writes zeros for the masquerade keys a type does
				// not use and stores finalmask in its own layout: neither is a change.
				PreConfig: func() {
					testAccStoreInboundStream(t, testAccFinalMaskPort, func(stream map[string]any) {
						hy, _ := stream["hysteriaSettings"].(map[string]any)
						hy["masquerade"] = map[string]any{
							"type": "file", "dir": "/var/www/acc-site", "url": "", "rewriteHost": false,
							"insecure": false, "content": "", "headers": map[string]any{}, "statusCode": float64(0),
						}
						stream["finalmask"] = json.RawMessage(`{ "udp": [ { "settings": { "password": "acc-obfs-secret" }, "type": "salamander" } ] }`)
					})
				},
				Config:             renamed,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: removed,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(name, "stream_settings.final_mask"),
					resource.TestCheckNoResourceAttr(name, "stream_settings.hysteria_settings.masquerade.type"),
					testAccCheckInboundStoredStream(nil, nil),
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

func testAccStoredStream(client *Client, id int) (map[string]any, error) {
	inbound, err := client.GetInbound(context.Background(), id)
	if err != nil {
		return nil, err
	}
	var stream map[string]any
	if err := json.Unmarshal([]byte(inbound.StreamSettings), &stream); err != nil {
		return nil, fmt.Errorf("parse streamSettings %q: %w", inbound.StreamSettings, err)
	}
	return stream, nil
}

// checkStreamMasks compares finalmask and hysteriaSettings.masquerade; nil
// means the key must be absent.
func checkStreamMasks(where string, stream map[string]any, finalMask, masquerade map[string]any) error {
	hy, _ := stream["hysteriaSettings"].(map[string]any)
	for key, pair := range map[string]struct {
		got  any
		want map[string]any
	}{
		"finalmask":  {stream["finalmask"], finalMask},
		"masquerade": {hy["masquerade"], masquerade},
	} {
		switch {
		case pair.want == nil && pair.got != nil:
			return fmt.Errorf("%s still has %s %v", where, key, pair.got)
		case pair.want != nil && !reflect.DeepEqual(pair.got, any(pair.want)):
			return fmt.Errorf("%s %s = %#v, want %#v", where, key, pair.got, pair.want)
		}
	}
	return nil
}

// Checks what the panel stored, not the provider's state.
func testAccCheckInboundStoredStream(finalMask, masquerade map[string]any) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[testAccFinalMaskInbound]
		if !ok {
			return fmt.Errorf("%s not found in state", testAccFinalMaskInbound)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("parse id %q: %w", rs.Primary.ID, err)
		}
		client, err := testAccClientFromEnv()
		if err != nil {
			return err
		}
		stream, err := testAccStoredStream(client, id)
		if err != nil {
			return err
		}
		return checkStreamMasks("stored streamSettings", stream, finalMask, masquerade)
	}
}

// The panel must hand both to the Xray it runs, on every supported version.
func testAccCheckGeneratedInboundStream(port int, finalMask, masquerade map[string]any) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := testAccClientFromEnv()
		if err != nil {
			return err
		}
		config, err := client.GetXrayConfig(context.Background())
		if err != nil {
			return err
		}
		inbounds, _ := config["inbounds"].([]any)
		for _, item := range inbounds {
			inbound, _ := item.(map[string]any)
			if intValue(inbound["port"]) != port {
				continue
			}
			stream, _ := inbound["streamSettings"].(map[string]any)
			return checkStreamMasks("generated Xray config", stream, finalMask, masquerade)
		}
		return fmt.Errorf("no inbound on port %d in the generated Xray config", port)
	}
}

// testAccStoreInboundStream rewrites streamSettings through the panel API,
// bypassing the provider, and waits until a read returns the new value: a read
// right after a write can still serve the previous row (#157).
func testAccStoreInboundStream(t *testing.T, port int, edit func(map[string]any)) {
	t.Helper()
	client, err := testAccClientFromEnv()
	if err != nil {
		t.Fatalf("client init failed: %v", err)
	}
	ctx := context.Background()
	inbounds, err := client.GetInbounds(ctx)
	if err != nil {
		t.Fatalf("listing inbounds: %v", err)
	}
	var inbound *Inbound
	for i := range inbounds {
		if inbounds[i].Port == port {
			inbound = &inbounds[i]
		}
	}
	if inbound == nil {
		t.Fatalf("no inbound on port %d", port)
	}
	var stream map[string]any
	if err := json.Unmarshal([]byte(inbound.StreamSettings), &stream); err != nil {
		t.Fatalf("parse streamSettings: %v", err)
	}
	edit(stream)
	b, err := json.Marshal(stream)
	if err != nil {
		t.Fatal(err)
	}
	inbound.StreamSettings = string(b)
	var want map[string]any
	if err := json.Unmarshal(b, &want); err != nil {
		t.Fatal(err)
	}
	wantFinalMask, _ := want["finalmask"].(map[string]any)
	wantHysteria, _ := want["hysteriaSettings"].(map[string]any)
	wantMasquerade, _ := wantHysteria["masquerade"].(map[string]any)
	if _, err := client.UpdateInbound(ctx, inbound); err != nil {
		t.Fatalf("storing streamSettings through the API: %v", err)
	}
	for attempt := 0; attempt < 30; attempt++ {
		if stored, err := testAccStoredStream(client, inbound.ID); err == nil &&
			checkStreamMasks("", stored, wantFinalMask, wantMasquerade) == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("the panel does not return the streamSettings written through the API")
}
