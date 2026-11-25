package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
