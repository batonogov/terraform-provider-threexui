package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestAccNodeResource_GUIDChangesDuringUpdate delays the first heartbeat's GUID
// learning (or replacement of a previously learned GUID) until after planning.
// The existing live-heartbeat fake learns its GUID on the create re-read, which
// cannot exercise either transition between the update plan and its re-read.
func TestAccNodeResource_GUIDChangesDuringUpdate(t *testing.T) {
	for _, tc := range []struct {
		name        string
		initialGUID string
	}{
		{name: "first_heartbeat", initialGUID: ""},
		{name: "replacement", initialGUID: "previous-node-guid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			panel := httptest.NewServer(&guidChangingNodePanel{
				panel:       fakeNodePanel{now: 1_700_000_000},
				initialGUID: tc.initialGUID,
			})
			defer panel.Close()

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: testAccNodeFakePanelConfig(panel.URL, "first"),
						Check:  resource.TestCheckResourceAttr("threexui_node.test", "guid", tc.initialGUID),
					},
					{
						Config:   testAccNodeFakePanelConfig(panel.URL, "first"),
						PlanOnly: true,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PostApplyPostRefresh: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction("threexui_node.test", plancheck.ResourceActionNoop),
								plancheck.ExpectKnownValue("threexui_node.test", tfjsonpath.New("guid"), knownvalue.StringExact(tc.initialGUID)),
							},
						},
					},
					{
						Config: testAccNodeFakePanelConfig(panel.URL, "second"),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction("threexui_node.test", plancheck.ResourceActionUpdate),
								plancheck.ExpectUnknownValue("threexui_node.test", tfjsonpath.New("guid")),
								plancheck.ExpectKnownValue("threexui_node.test", tfjsonpath.New("id"), knownvalue.StringExact("1")),
							},
						},
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("threexui_node.test", "id", "1"),
							resource.TestCheckResourceAttr("threexui_node.test", "guid", fakeNodeGUID),
							resource.TestCheckResourceAttr("threexui_node.test", "remark", "second"),
						),
					},
					{
						Config:   testAccNodeFakePanelConfig(panel.URL, "second"),
						PlanOnly: true,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PostApplyPostRefresh: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction("threexui_node.test", plancheck.ResourceActionNoop),
								plancheck.ExpectKnownValue("threexui_node.test", tfjsonpath.New("guid"), knownvalue.StringExact(fakeNodeGUID)),
							},
						},
					},
				},
			})
		})
	}
}

// guidChangingNodePanel reuses the heartbeat fake and all its routes, but keeps
// the initial observed GUID until an update. Serializing delegation and response
// rewriting makes the transition deterministic without sleeps or background jobs.
type guidChangingNodePanel struct {
	mu          sync.Mutex
	panel       fakeNodePanel
	initialGUID string
	updated     bool
}

func (p *guidChangingNodePanel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if r.URL.Path == "/panel/api/nodes/update/1" {
		p.updated = true
	}
	recorded := httptest.NewRecorder()
	p.panel.ServeHTTP(recorded, r)
	if r.URL.Path == "/panel/api/nodes/get/1" && !p.updated {
		var response struct {
			Success bool `json:"success"`
			Obj     Node `json:"obj"`
		}
		if err := json.Unmarshal(recorded.Body.Bytes(), &response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if response.Success {
			response.Obj.Guid = p.initialGUID
			recorded.Body.Reset()
			recorded.Body.Write(okResponse(&response.Obj))
		}
	}
	for key, values := range recorded.Header() {
		w.Header()[key] = values
	}
	w.WriteHeader(recorded.Code)
	_, _ = w.Write(recorded.Body.Bytes())
}

func TestGUIDChangingNodePanelPreservesMissingNode(t *testing.T) {
	panel := &guidChangingNodePanel{initialGUID: "not-yet-learned"}
	response := httptest.NewRecorder()
	panel.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panel/api/nodes/get/1", nil))
	if response.Code != http.StatusOK || response.Body.String() != string(failResponse("obtain (record not found)")) {
		t.Fatal("GUID fake must preserve a missing-node API failure")
	}
}
