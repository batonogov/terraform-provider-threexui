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
				if len(resp.Diagnostics) != 1 || resp.Diagnostics[0].Summary() != "Cluster node missing after update" || !resp.State.Raw.Equal(prior.Raw) {
					t.Fatal("missing node must return an actionable error and retain its prior identity until refresh")
				}
				read := resource.ReadResponse{State: resp.State}
				r.Read(ctx, resource.ReadRequest{State: resp.State}, &read)
				if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
					t.Fatal("next refresh must remove the missing node so Terraform can recreate it")
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

func TestAcknowledgedNodeSecret(t *testing.T) {
	known := types.StringValue("fixture-new")
	old := types.StringValue("fixture-old")
	one, two := types.Int64Value(1), types.Int64Value(2)
	for _, tc := range []struct {
		name        string
		planned, wo types.String
		version     types.Int64
		wantSecret  types.String
		wantVersion types.Int64
	}{
		{"rotation", known, known, two, known, two},
		{"legacy", known, types.StringNull(), types.Int64Null(), known, one},
		{"unknown-secret", types.StringUnknown(), known, two, old, one},
		{"null-secret", types.StringNull(), known, two, old, one},
		{"unknown-wo", known, types.StringUnknown(), two, known, one},
		{"null-wo", known, types.StringNull(), two, known, one},
		{"unknown-version", known, known, types.Int64Unknown(), known, one},
		{"null-version", known, known, types.Int64Null(), known, one},
		{"unchanged-version", old, known, one, old, one},
		{"unresolved-wo", old, known, two, old, one},
		{"known-empty-legacy", types.StringValue(""), types.StringNull(), two, old, one},
		{"blank-legacy", types.StringValue(" \t\n"), types.StringNull(), two, old, one},
		{"empty-wo", types.StringValue(""), types.StringValue(""), two, old, one},
		{"blank-wo", types.StringValue(" \t\n"), types.StringValue(" \t\n"), two, old, one},
	} {
		t.Run(tc.name, func(t *testing.T) {
			secret, version := acknowledgedNodeSecret(old, tc.planned, tc.wo, one, tc.version, true)
			if !secret.Equal(tc.wantSecret) || !version.Equal(tc.wantVersion) {
				t.Fatal("incorrect acknowledgement of submitted secret or rotation trigger")
			}
		})
	}
}

// A blank token is a no-op upstream, but a blank fingerprint is a real clear.
func TestAcknowledgedNodeSecret_EmptyPin(t *testing.T) {
	empty := types.StringValue("")
	secret, version := acknowledgedNodeSecret(types.StringValue("old-pin"), empty, empty, types.Int64Value(1), types.Int64Value(2), false)
	if !secret.Equal(empty) || version.ValueInt64() != 2 {
		t.Fatal("empty fingerprint clear must be acknowledged")
	}
}

func TestNodeResource_UpdateReadFailureSimultaneousSecrets(t *testing.T) {
	ctx := context.Background()
	fake := &nodeReadFailurePanel{panel: fakeNodePanel{now: 1_700_000_000, node: &Node{
		Id: 1, Name: "node", Address: "node.example.com", Port: 2053,
		Guid: fakeNodeGUID, Status: "online", LastHeartbeat: 1_700_000_000,
	}}, failure: "unavailable"}
	panel := httptest.NewServer(fake)
	defer panel.Close()
	r := &NodeResource{client: newTestClient(t, panel.URL)}
	prior := tfsdk.State(nodeResourcePlan(t, r, map[string]any{
		"id": "1", "name": "node", "address": "node.example.com", "port": int64(2053),
		"api_token": "fixture-old-token", "pinned_cert_sha256": "fixture-old-pin",
		"api_token_wo_version": int64(1), "pinned_cert_sha256_wo_version": int64(1),
	}))
	var state NodeResourceModel
	if diags := prior.Get(ctx, &state); diags.HasError() {
		t.Fatal(diags)
	}
	flattenNodeToModel(fake.panel.node, &state)
	if diags := prior.Set(ctx, &state); diags.HasError() {
		t.Fatal(diags)
	}
	plan := tfsdk.Plan(prior)
	for name, value := range map[string]any{
		"api_token": types.StringUnknown(), "pinned_cert_sha256": types.StringUnknown(),
		"api_token_wo_version": types.Int64Value(2), "pinned_cert_sha256_wo_version": types.Int64Value(2),
		"status": types.StringUnknown(), "last_heartbeat": types.Int64Unknown(), "config_dirty": types.BoolUnknown(),
	} {
		if diags := plan.SetAttribute(ctx, path.Root(name), value); diags.HasError() {
			t.Fatal(diags)
		}
	}
	configPlan := plan
	for name, value := range map[string]any{
		"api_token_wo":          types.StringValue("fixture-new-token"),
		"pinned_cert_sha256_wo": types.StringValue("fixture-new-pin"),
	} {
		if diags := configPlan.SetAttribute(ctx, path.Root(name), value); diags.HasError() {
			t.Fatal(diags)
		}
	}
	resp := resource.UpdateResponse{State: prior}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, Config: planToConfig(configPlan), State: prior}, &resp)
	if len(resp.Diagnostics) != 1 || resp.Diagnostics[0].Summary() != "Failed to re-read node after update" {
		t.Fatal("expected failed reread")
	}
	var got NodeResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatal(diags)
	}
	if got.ApiToken.ValueString() != "fixture-new-token" || got.PinnedCertSha256.ValueString() != "fixture-new-pin" ||
		got.ApiTokenWOVersion.ValueInt64() != 2 || got.PinnedCertSha256WOVersion.ValueInt64() != 2 {
		t.Fatal("successful simultaneous rotation must acknowledge both secrets and triggers")
	}
	if !got.ApiTokenWO.IsNull() || !got.PinnedCertSha256WO.IsNull() {
		t.Fatal("write-only attributes leaked into state")
	}
	// All remaining attributes must be exactly the prior known observed state.
	got.ApiToken, got.ApiTokenWOVersion = state.ApiToken, state.ApiTokenWOVersion
	got.PinnedCertSha256, got.PinnedCertSha256WOVersion = state.PinnedCertSha256, state.PinnedCertSha256WOVersion
	comparison := resp.State
	if diags := comparison.Set(ctx, &got); diags.HasError() {
		t.Fatal(diags)
	}
	if !comparison.Raw.Equal(prior.Raw) {
		t.Fatal("failed reread changed prior observed state")
	}
	read := resource.ReadResponse{State: resp.State}
	r.Read(ctx, resource.ReadRequest{State: resp.State}, &read)
	if read.Diagnostics.HasError() {
		t.Fatal(read.Diagnostics)
	}
	if diags := read.State.Get(ctx, &got); diags.HasError() {
		t.Fatal(diags)
	}
	if got.ApiToken.ValueString() != "fixture-new-token" || got.ApiTokenWOVersion.ValueInt64() != 2 || got.PinnedCertSha256WOVersion.ValueInt64() != 2 {
		t.Fatal("refresh lost non-echoed secret acknowledgement")
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.updates != 1 || fake.token != "fixture-new-token" || fake.pin != "fixture-new-pin" {
		t.Fatal("expected one POST with both resolved secrets")
	}
}
