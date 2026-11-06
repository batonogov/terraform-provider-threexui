package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"golang.org/x/mod/semver"
)

const (
	testAccXmuxInbound     = "threexui_inbound.xmux"
	testAccXmuxPort        = 25037
	testAccXmuxSubPort     = 2096
	testAccXmuxSubJSONPath = "/json-xmux/"
	// Distinctive, so a body rendered before the sub server restarted cannot match.
	testAccXmuxSubJSONMux = `{"enabled":true,"concurrency":41}`
)

// testAccPanelOlderThan reports whether THREEXUI_VERSION names a panel older
// than version; unset means the latest panel, as in requireMinVersion.
func testAccPanelOlderThan(version string) bool {
	v := os.Getenv("THREEXUI_VERSION")
	return semver.IsValid(v) && semver.Compare(v, version) < 0
}

func testAccInboundXmuxConfig(xmux string) string {
	return testAccProviderConfig() + testAccInboundXmuxResource("", xmux)
}

func testAccInboundXmuxResource(dependsOn, xmux string) string {
	return fmt.Sprintf(`
resource "threexui_inbound" "xmux" {
  %s
  port     = %d
  protocol = "vless"
  remark   = "acc-xmux"
  enable   = true
  vless_settings {
    decryption = "none"
  }
  stream_settings {
    network  = "xhttp"
    security = "none"
    xhttp_settings {
      path = "/acc-xmux"
      mode = "stream-up"
%s
    }
  }
}
`, dependsOn, testAccXmuxPort, xmux)
}

// The subscription apply restarts the panel, so the inbound and the client
// wait for it instead of racing the restart window.
func testAccInboundXmuxSubscriptionConfig(xmux string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "threexui_panel_subscription" "sub" {
  sub_enable      = true
  sub_port        = %d
  sub_json_enable = true
  sub_json_path   = %q
  sub_json_mux    = %q
}

resource "threexui_inbound_client" "xmux" {
  depends_on = [threexui_panel_subscription.sub]

  inbound_id = threexui_inbound.xmux.id
  email      = "acc-xmux@test.com"
  enable     = true
}
`, testAccXmuxSubPort, testAccXmuxSubJSONPath, testAccXmuxSubJSONMux) +
		testAccInboundXmuxResource("depends_on = [threexui_panel_subscription.sub]", xmux)
}

func TestAccInboundXmux(t *testing.T) {
	const name = testAccXmuxInbound
	concurrency := testAccInboundXmuxConfig(`
      xmux {
        max_concurrency     = "16-32"
        c_max_reuse_times   = "64-128"
        h_max_request_times = "600-900"
        h_max_reusable_secs = "1800-3000"
        h_keep_alive_period = 30
      }
`)
	connections := testAccInboundXmuxConfig(`
      xmux {
        max_connections     = "2"
        h_max_request_times = "6000-9000"
        h_max_reusable_secs = "1800-3000"
      }
`)
	removed := testAccInboundXmuxConfig("")
	// JSON subscription: the global subJsonMux, or the inbound's xmux instead
	// of it (3x-ui v3.4.0+, internal/sub/json_service.go).
	subscriptionStepsUnsupported := func() (bool, error) { return testAccPanelOlderThan("v3.4.0"), nil }

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckInboundDestroyed,
		Steps: []resource.TestStep{
			{
				Config: concurrency,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "stream_settings.xhttp_settings.xmux.max_concurrency", "16-32"),
					resource.TestCheckResourceAttr(name, "stream_settings.xhttp_settings.xmux.h_keep_alive_period", "30"),
					testAccCheckInboundStoredXmux(map[string]any{
						"maxConcurrency":   "16-32",
						"cMaxReuseTimes":   "64-128",
						"hMaxRequestTimes": "600-900",
						"hMaxReusableSecs": "1800-3000",
						"hKeepAlivePeriod": float64(30),
					}),
					testAccCheckGeneratedInboundXmux(testAccXmuxPort),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config:             concurrency,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// Removed attributes must remove their keys: nothing is Computed.
				Config: connections,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "stream_settings.xhttp_settings.xmux.max_connections", "2"),
					resource.TestCheckNoResourceAttr(name, "stream_settings.xhttp_settings.xmux.max_concurrency"),
					testAccCheckInboundStoredXmux(map[string]any{
						"maxConnections":   "2",
						"hMaxRequestTimes": "6000-9000",
						"hMaxReusableSecs": "1800-3000",
					}),
				),
			},
			{
				// Saving the inbound in the 3x-ui form backfills the missing fields
				// with zeros and may store maxConnections as a number.
				PreConfig: func() {
					testAccStoreInboundXmux(t, map[string]any{
						"maxConcurrency":   "",
						"maxConnections":   float64(2),
						"cMaxReuseTimes":   float64(0),
						"hMaxRequestTimes": "6000-9000",
						"hMaxReusableSecs": "1800-3000",
						"hKeepAlivePeriod": float64(0),
					})
				},
				Config:             connections,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: removed,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(name, "stream_settings.xhttp_settings.xmux.max_connections"),
					testAccCheckInboundStoredXmux(nil),
				),
			},
			{
				Config:             removed,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				SkipFunc: subscriptionStepsUnsupported,
				Config:   testAccInboundXmuxSubscriptionConfig(""),
				Check:    testAccCheckXmuxSubscription(t, nil),
			},
			{
				// Even an empty block replaces subJsonMux.
				SkipFunc: subscriptionStepsUnsupported,
				Config: testAccInboundXmuxSubscriptionConfig(`
      xmux {}
`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckInboundStoredXmux(map[string]any{}),
					testAccCheckXmuxSubscription(t, map[string]any{}),
				),
			},
			{
				SkipFunc: subscriptionStepsUnsupported,
				Config: testAccInboundXmuxSubscriptionConfig(`
      xmux {}
`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccStoredXHTTPSettings(client *Client, id int) (map[string]any, error) {
	inbound, err := client.GetInbound(context.Background(), id)
	if err != nil {
		return nil, err
	}
	var stream struct {
		XHTTPSettings map[string]any `json:"xhttpSettings"`
	}
	if err := json.Unmarshal([]byte(inbound.StreamSettings), &stream); err != nil {
		return nil, fmt.Errorf("parse streamSettings %q: %w", inbound.StreamSettings, err)
	}
	return stream.XHTTPSettings, nil
}

// Checks the xmux object the panel stored, not the provider's state; nil
// means the key must be absent.
func testAccCheckInboundStoredXmux(want map[string]any) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[testAccXmuxInbound]
		if !ok {
			return fmt.Errorf("%s not found in state", testAccXmuxInbound)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("parse id %q: %w", rs.Primary.ID, err)
		}
		client, err := testAccClientFromEnv()
		if err != nil {
			return err
		}
		xhttp, err := testAccStoredXHTTPSettings(client, id)
		if err != nil {
			return err
		}
		got, present := xhttp["xmux"]
		switch {
		case want == nil && present:
			return fmt.Errorf("stored xhttpSettings still has xmux %v", got)
		case want != nil && !reflect.DeepEqual(got, any(want)):
			return fmt.Errorf("stored xmux = %#v, want %#v", got, want)
		}
		return nil
	}
}

// testAccStoreInboundXmux writes xmux through the panel API, bypassing the
// provider, and waits until a read returns it: a read right after a write can
// still serve the previous row (#157). PreConfig has no state, so the inbound
// is found by its port, which the panel keeps unique.
func testAccStoreInboundXmux(t *testing.T, xmux map[string]any) {
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
		if inbounds[i].Port == testAccXmuxPort {
			inbound = &inbounds[i]
		}
	}
	if inbound == nil {
		t.Fatalf("no inbound on port %d", testAccXmuxPort)
	}
	var stream map[string]any
	if err := json.Unmarshal([]byte(inbound.StreamSettings), &stream); err != nil {
		t.Fatalf("parse streamSettings: %v", err)
	}
	xhttp, ok := stream["xhttpSettings"].(map[string]any)
	if !ok {
		t.Fatalf("stored streamSettings has no xhttpSettings: %s", inbound.StreamSettings)
	}
	xhttp["xmux"] = xmux
	b, err := json.Marshal(stream)
	if err != nil {
		t.Fatal(err)
	}
	inbound.StreamSettings = string(b)
	if _, err := client.UpdateInbound(ctx, inbound); err != nil {
		t.Fatalf("storing xmux through the API: %v", err)
	}
	for attempt := 0; attempt < 30; attempt++ {
		if stored, err := testAccStoredXHTTPSettings(client, inbound.ID); err == nil && reflect.DeepEqual(stored["xmux"], any(xmux)) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("the panel does not return the xmux written through the API")
}

// testAccCheckGeneratedInboundXmux checks the config the panel generates for
// its own Xray: v3.4.0+ strips the client-only xmux (StripInboundXhttpClientFields),
// v3.3.x passes it through.
func testAccCheckGeneratedInboundXmux(port int) resource.TestCheckFunc {
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
			xhttp, _ := stream["xhttpSettings"].(map[string]any)
			_, present := xhttp["xmux"]
			if want := testAccPanelOlderThan("v3.4.0"); present != want {
				return fmt.Errorf("generated Xray config: xmux present = %t, want %t: %v", present, want, xhttp)
			}
			return nil
		}
		return fmt.Errorf("no inbound on port %d in the generated Xray config", port)
	}
}

// testAccCheckXmuxSubscription fetches the client's JSON subscription and checks
// its proxy outbound: with wantXmux nil it must carry the global subJsonMux and
// no xmux, otherwise the inbound's xmux and no mux.
func testAccCheckXmuxSubscription(t *testing.T, wantXmux map[string]any) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith("threexui_inbound_client.xmux", "sub_id", func(subID string) error {
		if subID == "" {
			return fmt.Errorf("client sub_id is empty")
		}
		var wantMux map[string]any
		if err := json.Unmarshal([]byte(testAccXmuxSubJSONMux), &wantMux); err != nil {
			return err
		}
		url := subscriptionTestURL(t, testAccXmuxSubPort, testAccXmuxSubJSONPath, subID)
		var lastErr error
		// The body is rendered from the database on each request; poll past the
		// read-after-write lag instead of trusting the first response.
		for attempt := 0; attempt < 20; attempt++ {
			body, err := waitForSubscriptionBody(t, url)
			if err != nil {
				return err
			}
			proxy, err := xmuxSubscriptionProxyOutbound(body)
			if err == nil {
				stream, _ := proxy["streamSettings"].(map[string]any)
				xhttp, _ := stream["xhttpSettings"].(map[string]any)
				xmux, hasXmux := xhttp["xmux"]
				mux, hasMux := proxy["mux"]
				switch {
				case wantXmux == nil && (hasXmux || !reflect.DeepEqual(mux, any(wantMux))):
					err = fmt.Errorf("want mux %v and no xmux, got mux %v, xmux %v", wantMux, mux, xmux)
				case wantXmux != nil && (hasMux || !reflect.DeepEqual(xmux, any(wantXmux))):
					err = fmt.Errorf("want xmux %v and no mux, got xmux %v (present %t), mux %v", wantXmux, xmux, hasXmux, mux)
				default:
					return nil
				}
			}
			lastErr = err
			time.Sleep(time.Second)
		}
		return fmt.Errorf("JSON subscription %s: %w", url, lastErr)
	})
}

// xmuxSubscriptionProxyOutbound returns the proxy outbound of the only config
// in a JSON subscription body (an object, or an array of one).
func xmuxSubscriptionProxyOutbound(body string) (map[string]any, error) {
	var parsed any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return nil, fmt.Errorf("parse body: %w: %s", err, truncateForError(body))
	}
	configs, isArray := parsed.([]any)
	if !isArray {
		configs = []any{parsed}
	}
	if len(configs) != 1 {
		return nil, fmt.Errorf("want one config, got %d: %s", len(configs), truncateForError(body))
	}
	config, _ := configs[0].(map[string]any)
	outbounds, _ := config["outbounds"].([]any)
	for _, item := range outbounds {
		if outbound, _ := item.(map[string]any); outbound["tag"] == "proxy" {
			return outbound, nil
		}
	}
	return nil, fmt.Errorf("no proxy outbound: %s", truncateForError(body))
}
