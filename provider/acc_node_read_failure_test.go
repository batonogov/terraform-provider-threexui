package provider

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// Fail only the read immediately after a successful update, not the refresh
// before it. Keeping the fake's row allows the next Terraform refresh to recover.
type nodeReadFailurePanel struct {
	panel   fakeNodePanel
	mu      sync.Mutex
	failure string
	pending bool
	updates int
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
		p.pending = true
		p.updates++
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
				// The framework rejects a null Update result without diagnostics.
				// Preserve that existing behavior; the following refresh recreates.
				ExpectError: regexp.MustCompile("Missing Resource State After Update"),
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
