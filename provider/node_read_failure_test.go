package provider

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNodeResource_UpdateReadFailure(t *testing.T) {
	for _, failure := range []string{"unavailable", "malformed", "api-error", "null", "missing", "empty", "not-found"} {
		t.Run(failure, func(t *testing.T) {
			ctx := context.Background()
			fake := &nodeReadFailurePanel{
				panel: fakeNodePanel{now: 1_700_000_000, node: &Node{
					Id: 1, Name: "node", Address: "node.example.com", Port: 2053,
					Remark: "first", Guid: fakeNodeGUID, Status: "online",
					LastHeartbeat: 1_700_000_000,
				}},
				failure: failure,
			}
			panel := httptest.NewServer(fake)
			defer panel.Close()
			r := &NodeResource{client: newTestClient(t, panel.URL)}
			prior := tfsdk.State(nodeResourcePlan(t, r, map[string]any{
				"id": "1", "name": "node", "address": "node.example.com", "port": int64(2053), "remark": "first",
			}))
			var model NodeResourceModel
			if diags := prior.Get(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			flattenNodeToModel(fake.panel.node, &model)
			if diags := prior.Set(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			plan := tfsdk.Plan(prior)
			for name, value := range map[string]any{
				"remark":         types.StringValue("second"),
				"status":         types.StringUnknown(),
				"last_heartbeat": types.Int64Unknown(),
				"config_dirty":   types.BoolUnknown(),
			} {
				if diags := plan.SetAttribute(ctx, path.Root(name), value); diags.HasError() {
					t.Fatal(diags)
				}
			}

			resp := resource.UpdateResponse{State: prior}
			r.Update(ctx, resource.UpdateRequest{Plan: plan, Config: planToConfig(plan), State: prior}, &resp)
			if failure == "not-found" {
				if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
					t.Fatalf("explicit record-not-found must remove state without errors: %v; state %s", resp.Diagnostics, resp.State.Raw)
				}
				return
			}
			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected reread error, got %v", resp.Diagnostics)
			}
			if len(resp.Diagnostics) != 1 || resp.Diagnostics[0].Summary() != "Failed to re-read node after update" ||
				!strings.Contains(resp.Diagnostics[0].Detail(), "was updated") {
				t.Fatalf("expected actionable read-after-update diagnostic: %v", resp.Diagnostics)
			}
			if !resp.State.Raw.Equal(prior.Raw) {
				t.Fatalf("reread failure must retain prior state, not unknown plan: %s", resp.State.Raw)
			}

			// The remote write succeeded. A later refresh must recover the new
			// managed and observed values using the retained identity alone.
			read := resource.ReadResponse{State: resp.State}
			r.Read(ctx, resource.ReadRequest{State: resp.State}, &read)
			if read.Diagnostics.HasError() {
				t.Fatal(read.Diagnostics)
			}
			if diags := read.State.Get(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			if model.ID.ValueString() != "1" || model.Remark.ValueString() != "second" ||
				model.LastHeartbeat.IsUnknown() || model.LastHeartbeat.ValueInt64() <= 1_700_000_000 ||
				!model.ConfigDirty.ValueBool() || model.Status.ValueString() != "online" {
				t.Fatalf("refresh did not reconcile the successful remote update: %+v", model)
			}
		})
	}
}
