package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Fail only the read immediately after a successful update, not the refresh
// before it. Keeping the fake's row allows the next Terraform refresh to recover.
type nodeReadFailurePanel struct {
	panel   fakeNodePanel
	mu      sync.Mutex
	failure string
	pending bool
	updates int
	token   string
	pin     string
}

func (p *nodeReadFailurePanel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if r.URL.Path == "/panel/api/nodes/get/1" && p.pending {
		p.pending = false
		switch p.failure {
		case "not-found":
			p.panel.mu.Lock()
			p.panel.node = nil
			p.panel.mu.Unlock()
			w.Write(failResponse("obtain (record not found)"))
		case "null":
			w.Write(okResponse(nil))
		case "missing":
			w.Write([]byte(`{"success":true}`))
		case "empty":
			w.WriteHeader(http.StatusOK)
		case "malformed":
			w.Write([]byte(`{"success":`))
		case "api-error":
			w.Write(failResponse("database unavailable"))
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		return
	}
	p.panel.ServeHTTP(w, r)
	if r.URL.Path == "/panel/api/nodes/update/1" {
		p.pending = p.failure != ""
		p.updates++
		p.token = r.FormValue("apiToken")
		p.pin = r.FormValue("pinnedCertSha256")
	}
}

// An explicit record-not-found after update retains the existing deletion
// semantics: Terraform sees the disappearance and can recreate on the next run.
func TestAccNodeResource_UpdateReadNotFound(t *testing.T) {
	fake := &nodeReadFailurePanel{panel: fakeNodePanel{now: 1_700_000_000}, failure: "not-found"}
	panel := httptest.NewServer(fake)
	defer panel.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: testAccNodeFakePanelConfig(panel.URL, "first")},
			{
				Config: testAccNodeFakePanelConfig(panel.URL, "second"),
				// Report the disappearance explicitly, not a framework bug. The
				// following refresh removes the old identity and plans recreation.
				ExpectError: regexp.MustCompile("Cluster node missing after update"),
			},
			{
				Config: testAccNodeFakePanelConfig(panel.URL, "second"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("threexui_node.test", plancheck.ResourceActionCreate),
					},
				},
				Check: resource.TestCheckResourceAttr("threexui_node.test", "remark", "second"),
			},
		},
	})
}

func TestAccNodeResource_UpdateReadFailureRecovery(t *testing.T) {
	for _, failure := range []string{"unavailable", "null"} {
		t.Run(failure, func(t *testing.T) {
			fake := &nodeReadFailurePanel{panel: fakeNodePanel{now: 1_700_000_000}, failure: failure}
			panel := httptest.NewServer(fake)
			defer panel.Close()

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{Config: testAccNodeFakePanelConfig(panel.URL, "first")},
					{
						Config:      testAccNodeFakePanelConfig(panel.URL, "second"),
						ExpectError: regexp.MustCompile("Failed to re-read node after update"),
					},
					{
						Config: testAccNodeFakePanelConfig(panel.URL, "second"),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("threexui_node.test", "id", "1"),
							resource.TestCheckResourceAttr("threexui_node.test", "remark", "second"),
							resource.TestCheckResourceAttr("threexui_node.test", "status", "online"),
							resource.TestCheckResourceAttr("threexui_node.test", "config_dirty", "true"),
						),
					},
				},
			})
			fake.mu.Lock()
			defer fake.mu.Unlock()
			if fake.updates != 1 {
				t.Fatalf("refresh should recover without another update; got %d updates", fake.updates)
			}
		})
	}
}

// Exercise the real Terraform protocol: refresh cannot recover a non-echoed
// legacy token from the panel after a failed apply.
func TestAccNodeResource_SecretRotationReadFailureRecovery(t *testing.T) {
	for _, failure := range []string{"unavailable", "null"} {
		t.Run(failure, func(t *testing.T) {
			fake := &nodeReadFailurePanel{panel: fakeNodePanel{now: 1_700_000_000}, failure: failure}
			panel := httptest.NewServer(fake)
			defer panel.Close()
			config := func(version int) string {
				attrs := fmt.Sprintf("api_token = %q\n pinned_cert_sha256 = %q", fmt.Sprintf("fixture-token-%d", version), fmt.Sprintf("fixture-pin-%d", version))

				return strings.Replace(testAccNodeFakePanelConfig(panel.URL, "rotation"), `api_token    = "acc-node-token"`, attrs, 1)
			}
			check := func(s *terraform.State) error {
				attrs := s.RootModule().Resources["threexui_node.test"].Primary.Attributes
				if attrs["api_token"] != "fixture-token-2" || attrs["pinned_cert_sha256"] != "fixture-pin-2" {
					return fmt.Errorf("acknowledged sensitive values were not retained")
				}
				if attrs["api_token_wo"] != "" || attrs["pinned_cert_sha256_wo"] != "" {
					return fmt.Errorf("write-only attributes must not be persisted")
				}

				return nil
			}
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{Config: config(1)},
					{Config: config(2), ExpectError: regexp.MustCompile("Failed to re-read node after update")},
					{RefreshState: true, Check: check},
					{Config: config(2), ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}, Check: check},
				},
			})
			fake.mu.Lock()
			defer fake.mu.Unlock()
			if fake.updates != 1 {
				t.Fatalf("rotation recovery must issue exactly one update POST; got %d", fake.updates)
			}
			if fake.token != "fixture-token-2" || fake.pin != "fixture-pin-2" {
				t.Fatal("update POST did not carry both rotated secrets")
			}
		})
	}
}

// Use the production schema and real Terraform Core, including WO-only
// creation and migration from legacy plain secrets.
func TestAccNodeResource_WriteOnlyRotationReadFailureRecovery(t *testing.T) {
	for _, legacySeed := range []bool{false, true} {
		for _, failure := range []string{"unavailable", "null"} {
			t.Run(fmt.Sprintf("legacy-seed=%t/%s", legacySeed, failure), func(t *testing.T) {
				fake := &nodeReadFailurePanel{panel: fakeNodePanel{now: 1_700_000_000}, failure: failure}
				panel := httptest.NewServer(fake)
				defer panel.Close()
				config := func(version int, remark string) string {
					attrs := fmt.Sprintf("api_token_wo = %q\n api_token_wo_version = %d\n pinned_cert_sha256_wo = %q\n pinned_cert_sha256_wo_version = %d", fmt.Sprintf("fixture-token-%d", version), version, fmt.Sprintf("fixture-pin-%d", version), version)
					if legacySeed && version == 1 {
						attrs = `api_token = "fixture-token-1"
 pinned_cert_sha256 = "fixture-pin-1"`
					}
					return strings.Replace(testAccNodeFakePanelConfig(panel.URL, remark), `api_token    = "acc-node-token"`, attrs, 1)
				}
				check := func(version, updates int) resource.TestCheckFunc {
					return func(s *terraform.State) error {
						attrs := s.RootModule().Resources["threexui_node.test"].Primary.Attributes
						if attrs["api_token"] != fmt.Sprintf("fixture-token-%d", version) || attrs["pinned_cert_sha256"] != fmt.Sprintf("fixture-pin-%d", version) {
							return fmt.Errorf("acknowledged secrets were not retained")
						}
						for _, key := range []string{"api_token_wo", "pinned_cert_sha256_wo"} {
							if _, exists := attrs[key]; exists {
								return fmt.Errorf("write-only attribute %s persisted in state", key)
							}
							if !legacySeed || version != 1 {
								if attrs[key+"_version"] != fmt.Sprint(version) {
									return fmt.Errorf("rotation trigger %s was not acknowledged", key)
								}
							}
						}
						fake.mu.Lock()
						defer fake.mu.Unlock()
						if fake.updates != updates {
							return fmt.Errorf("expected %d update POSTs, got %d", updates, fake.updates)
						}
						if updates > 0 && (fake.token != "fixture-token-2" || fake.pin != "fixture-pin-2") {
							return fmt.Errorf("update did not preserve both rotated secrets")
						}
						return nil
					}
				}
				emptyPlan := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}
				resource.Test(t, resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
					Steps: []resource.TestStep{
						{Config: config(1, "rotation"), Check: check(1, 0)},
						{Config: config(2, "rotation"), ExpectError: regexp.MustCompile("Failed to re-read node after update")},
						{RefreshState: true, Check: check(2, 1)},
						{Config: config(2, "rotation"), ConfigPlanChecks: emptyPlan, Check: check(2, 1)},
						{PreConfig: func() { fake.mu.Lock(); fake.failure = ""; fake.mu.Unlock() }, Config: config(2, "unrelated"), Check: resource.ComposeTestCheckFunc(check(2, 2), resource.TestCheckResourceAttr("threexui_node.test", "remark", "unrelated"))},
						{Config: config(2, "unrelated"), ConfigPlanChecks: emptyPlan, Check: check(2, 2)},
					},
				})
			})
		}
	}
}
