package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccXrayObservatory(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccXrayObservatoryConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "id", "xray_observatory"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.#", "1"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.tag", "obs_default"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.subject_selector.#", "1"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.subject_selector.0", "proxy-*"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.probe_url", "https://www.google.com/generate_204"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.probe_interval", "1m"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.enable_concurrency", "true"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.#", "1"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.tag", "burst_default"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.subject_selector.#", "1"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.subject_selector.0", "proxy-*"),
				),
			},
			// Update
			{
				Config: testAccProviderConfig() + testAccXrayObservatoryConfigUpdated(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "id", "xray_observatory"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.#", "1"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.tag", "obs_updated"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.subject_selector.#", "2"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.subject_selector.0", "proxy-*"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.subject_selector.1", "direct-*"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.probe_interval", "30s"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.enable_concurrency", "false"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.#", "1"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.tag", "burst_updated"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.ping_config.0.destination", "https://www.cloudflare.com/cdn-cgi/trace"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.ping_config.0.interval", "1m"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.ping_config.0.connect_timeout", "5s"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.ping_config.0.timeout", "10s"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.ping_config.0.samples", "3"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.ping_config.0.sampling_count", "2"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.ping_config.0.lazy", "true"),
				),
			},
			// Idempotency
			{
				Config:             testAccProviderConfig() + testAccXrayObservatoryConfigUpdated(),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccXrayObservatoryImport(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccXrayObservatoryConfigUpdated(),
			},
			{
				ResourceName:            "threexui_xray_observatory.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "xray_observatory",
				ImportStateVerifyIgnore: []string{},
			},
		},
	})
}

// TestAccXrayObservatoryFlat migrates a tag-keyed configuration to the flat
// Xray singleton form (tag "default") and requires an empty plan afterwards,
// proving read-back convergence on a live panel.
func TestAccXrayObservatoryFlat(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + testAccXrayObservatoryConfig(),
			},
			{
				Config: testAccProviderConfig() + testAccXrayObservatoryFlatConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.#", "1"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.tag", "default"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.subject_selector.0", "proxy-*"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "observatory.0.probe_interval", "1m"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.#", "1"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.tag", "default"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.subject_selector.0", "proxy-*"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.ping_config.0.destination", "https://www.cloudflare.com/cdn-cgi/trace"),
					resource.TestCheckResourceAttr("threexui_xray_observatory.test", "burst_observatory.0.ping_config.0.samples", "3"),
					checkFlatWireAndCoreAlive(t),
				),
			},
			{
				Config:             testAccProviderConfig() + testAccXrayObservatoryFlatConfig(),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccXrayObservatoryFlatConfig() string {
	return `
resource "threexui_xray_observatory" "test" {
  observatory {
    tag                = "default"
    subject_selector   = ["proxy-*"]
    probe_url          = "https://www.google.com/generate_204"
    probe_interval     = "1m"
    enable_concurrency = true
  }

  burst_observatory {
    tag              = "default"
    subject_selector = ["proxy-*"]

    ping_config {
      destination = "https://www.cloudflare.com/cdn-cgi/trace"
      interval    = "1m"
      timeout     = "5s"
      samples     = 3
    }
  }
}
`
}

// checkFlatWireAndCoreAlive verifies that the flat singleton shape actually
// reached the panel template and that the Xray core is running with it.
// xray.version alone is not a liveness signal: it comes from
// `xray -version` and survives a core that crashed on an invalid config.
func checkFlatWireAndCoreAlive(t *testing.T) resource.TestCheckFunc {
	t.Helper()
	return func(*terraform.State) error {
		client, err := testAccClientFromEnv()
		if err != nil {
			return fmt.Errorf("test client init: %w", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		// Restart so the new template is loaded even if the core already
		// died from an earlier (invalid) configuration.
		if err := client.RestartXrayService(ctx); err != nil {
			return fmt.Errorf("RestartXrayService: %w", err)
		}

		tpl, err := client.GetXrayTemplate(ctx)
		if err != nil {
			return fmt.Errorf("GetXrayTemplate: %w", err)
		}
		for _, key := range []string{"observatory", "burstObservatory"} {
			section, ok := tpl[key].(map[string]any)
			if !ok {
				return fmt.Errorf("template key %q missing or not an object: %#v", key, tpl[key])
			}
			if _, ok := section["subjectSelector"]; !ok {
				return fmt.Errorf("template key %q is not the flat singleton form: %#v", key, section)
			}
			if _, ok := section["default"]; ok {
				return fmt.Errorf("template key %q is still tag-keyed: %#v", key, section)
			}
		}

		// Two consecutive running polls, 3s apart, rule out the brief
		// "running" state between process start and config-load failure.
		deadline := time.Now().Add(60 * time.Second)
		streak := 0
		var lastStatus any
		for time.Now().Before(deadline) {
			status, err := client.GetServerStatus(ctx)
			if err != nil {
				return fmt.Errorf("GetServerStatus: %w", err)
			}
			lastStatus = status
			xray, _ := status["xray"].(map[string]any)
			if state, _ := xray["state"].(string); state == "running" {
				streak++
				if streak >= 2 {
					if msg, _ := xray["errorMsg"].(string); msg != "" {
						t.Logf("xray.errorMsg present while running (informational): %s", msg)
					}
					return nil
				}
			} else {
				streak = 0
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("xray liveness deadline exceeded, last status: %#v", lastStatus)
			case <-time.After(3 * time.Second):
			}
		}
		return fmt.Errorf("xray core did not stay running, last status: %#v", lastStatus)
	}
}

func testAccXrayObservatoryConfig() string {
	return `
resource "threexui_xray_observatory" "test" {
  observatory {
    tag                = "obs_default"
    subject_selector   = ["proxy-*"]
    probe_url          = "https://www.google.com/generate_204"
    probe_interval     = "1m"
    enable_concurrency = true
  }

  burst_observatory {
    tag              = "burst_default"
    subject_selector = ["proxy-*"]
  }
}
`
}

func testAccXrayObservatoryConfigUpdated() string {
	return `
resource "threexui_xray_observatory" "test" {
  observatory {
    tag                = "obs_updated"
    subject_selector   = ["proxy-*", "direct-*"]
    probe_url          = "https://www.google.com/generate_204"
    probe_interval     = "30s"
    enable_concurrency = false
  }

  burst_observatory {
    tag              = "burst_updated"
    subject_selector = ["proxy-*"]

    ping_config {
      destination     = "https://www.cloudflare.com/cdn-cgi/trace"
      interval        = "1m"
      connect_timeout = "5s"
      timeout         = "10s"
      samples         = 3
      sampling_count  = 2
      lazy            = true
    }
  }
}
`
}
