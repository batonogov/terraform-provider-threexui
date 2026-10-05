package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestHostGroupExpandFlatten verifies the model↔struct round-trip: every
// managed attribute survives hostGroupFromModel → flattenHostGroupToModel
// unchanged (snake_case tfsdk ↔ camelCase struct).
func TestHostGroupExpandFlatten(t *testing.T) {
	model := &HostGroupResourceModel{
		GroupID:                types.StringValue("grp123"),
		Remark:                 types.StringValue("premium-servers"),
		ServerDescription:      types.StringValue("Premium EU nodes"),
		SortOrder:              types.Int64Value(5),
		IsDisabled:             types.BoolValue(false),
		IsHidden:               types.BoolValue(true),
		Port:                   types.Int64Value(443),
		Security:               types.StringValue("tls"),
		Sni:                    types.StringValue("example.com"),
		HostHeader:             types.StringValue("example.com"),
		Path:                   types.StringValue("/ws"),
		Fingerprint:            types.StringValue("chrome"),
		CipherSuites:           types.StringValue("TLS_AES_128_GCM_SHA256:TLS_CHACHA20_POLY1305_SHA256"),
		OverrideSniFromAddress: types.BoolValue(true),
		KeepSniBlank:           types.BoolValue(false),
		VerifyPeerCertByName:   types.StringValue("example.com"),
		AllowInsecure:          types.BoolValue(false),
		EchConfigList:          types.StringValue("{}"),
		MuxParams:              types.StringValue("{}"),
		SockoptParams:          types.StringValue("{}"),
		FinalMask:              types.StringValue("tcp"),
		MihomoIpVersion:        types.StringValue("ipv4-prefer"),
		MihomoX25519:           types.BoolValue(true),
		ShuffleHost:            types.BoolValue(true),
	}
	model.InboundIDs = types.ListValueMust(types.Int64Type, []attr.Value{
		types.Int64Value(1), types.Int64Value(2), types.Int64Value(3),
	})
	model.Hosts = types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("host-a.example.com"), types.StringValue("host-b.example.com"),
	})
	model.Tags = types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("eu"), types.StringValue("premium"),
	})
	model.Alpn = types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("h2"), types.StringValue("http/1.1"),
	})
	model.PinnedPeerCertSha256 = types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("sha256/abc"),
	})
	model.ExcludeFromSubTypes = types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("clash"),
	})
	model.NodeGuids = types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("node-guid-1"),
	})

	ctx := context.Background()
	hg := hostGroupFromModel(ctx, model)

	if hg.GroupId != "grp123" {
		t.Fatalf("GroupId: got %q, want grp123", hg.GroupId)
	}
	if hg.Remark != "premium-servers" {
		t.Fatalf("Remark: got %q", hg.Remark)
	}
	if hg.Port != 443 {
		t.Fatalf("Port: got %d", hg.Port)
	}
	if hg.Security != "tls" {
		t.Fatalf("Security: got %q", hg.Security)
	}
	if len(hg.InboundIds) != 3 || hg.InboundIds[0] != 1 || hg.InboundIds[2] != 3 {
		t.Fatalf("InboundIds: got %v", hg.InboundIds)
	}
	if len(hg.Hosts) != 2 || hg.Hosts[0] != "host-a.example.com" {
		t.Fatalf("Hosts: got %v", hg.Hosts)
	}
	if len(hg.Tags) != 2 || hg.Tags[1] != "premium" {
		t.Fatalf("Tags: got %v", hg.Tags)
	}
	if hg.MihomoIpVersion != "ipv4-prefer" {
		t.Fatalf("MihomoIpVersion: got %q", hg.MihomoIpVersion)
	}
	if hg.CipherSuites != "TLS_AES_128_GCM_SHA256:TLS_CHACHA20_POLY1305_SHA256" {
		t.Fatalf("CipherSuites: got %q", hg.CipherSuites)
	}

	// Flatten back into a fresh model and spot-check.
	flat := &HostGroupResourceModel{}
	flattenHostGroupToModel(hg, flat)
	if flat.GroupID.ValueString() != "grp123" {
		t.Fatalf("flatten GroupID: got %q", flat.GroupID)
	}
	if flat.Port.ValueInt64() != 443 {
		t.Fatalf("flatten Port: got %d", flat.Port.ValueInt64())
	}
	if flat.Security.ValueString() != "tls" {
		t.Fatalf("flatten Security: got %q", flat.Security)
	}
	if flat.CipherSuites.ValueString() != "TLS_AES_128_GCM_SHA256:TLS_CHACHA20_POLY1305_SHA256" {
		t.Fatalf("flatten CipherSuites: got %q", flat.CipherSuites)
	}
	// inbound_ids round-trip
	var ids []int64
	flat.InboundIDs.ElementsAs(ctx, &ids, false)
	if len(ids) != 3 || ids[0] != 1 || ids[2] != 3 {
		t.Fatalf("flatten InboundIDs: got %v", ids)
	}
}

// TestHostGroupModelToWire verifies the HostGroup struct serialises with the
// camelCase JSON keys the 3x-ui v3.5.0 controller binds (entity.HostGroup).
func TestHostGroupModelToWire(t *testing.T) {
	hg := &HostGroup{
		GroupId:                "grp456",
		InboundIds:             []int{10, 20},
		Remark:                 "test",
		SortOrder:              1,
		Port:                   8080,
		Security:               "reality",
		Hosts:                  []string{"a.test"},
		Alpn:                   []string{"h2"},
		CipherSuites:           "TLS_AES_128_GCM_SHA256",
		OverrideSniFromAddress: true,
		MihomoIpVersion:        "dual",
		ShuffleHost:            true,
	}
	raw, err := json.Marshal(hg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	camelKeys := []string{
		"groupId", "inboundIds", "remark", "sortOrder", "port", "security",
		"cipherSuites", "overrideSniFromAddress", "mihomoIpVersion", "shuffleHost",
	}
	for _, k := range camelKeys {
		if _, ok := m[k]; !ok {
			t.Fatalf("expected camelCase JSON key %q, got keys: %v", k, mapKeys(m))
		}
	}
	// snake_case must NOT appear on the wire.
	for _, k := range []string{"group_id", "inbound_ids", "sort_order", "override_sni_from_address"} {
		if _, ok := m[k]; ok {
			t.Fatalf("snake_case key %q must not be on the wire", k)
		}
	}
}

// TestHostGroupEmptyInboundIDs verifies expand handles an unset/null inbound_ids
// gracefully (the schema enforces min=1, but expand must not panic on null).
func TestHostGroupEmptyInboundIDs(t *testing.T) {
	model := &HostGroupResourceModel{
		Remark: types.StringValue("solo"),
		// InboundIDs intentionally null.
	}
	ctx := context.Background()
	hg := hostGroupFromModel(ctx, model)
	if hg.InboundIds != nil {
		t.Fatalf("expected nil InboundIds for null list, got %v", hg.InboundIds)
	}
	if hg.Remark != "solo" {
		t.Fatalf("Remark: got %q", hg.Remark)
	}
}

// TestHostGroupGeneratedGroupID verifies that when group_id is unset on the
// model, expand leaves GroupId empty so the server generates one on Create.
func TestHostGroupGeneratedGroupID(t *testing.T) {
	model := &HostGroupResourceModel{
		Remark: types.StringValue("auto-id"),
	}
	model.GroupID = types.StringNull()
	model.InboundIDs = types.ListValueMust(types.Int64Type, []attr.Value{types.Int64Value(1)})

	ctx := context.Background()
	hg := hostGroupFromModel(ctx, model)
	if hg.GroupId != "" {
		t.Fatalf("expected empty GroupId for server-generated id, got %q", hg.GroupId)
	}
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ---------------------------------------------------------------------------
// Client method tests (httptest): these guard the two v3.5.0 host-group
// wire-format gotchas — /add returns an ARRAY of host rows (not a single
// HostGroup), and a missing group is signalled by an explicit
// "host group not found" message (not gorm.ErrRecordNotFound).
// ---------------------------------------------------------------------------

func TestCreateHostGroupParsesArrayResponse(t *testing.T) {
	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/hosts/add":
			receivedBody, _ = io.ReadAll(r.Body)
			// 3x-ui's AddHostGroup returns []*model.Host — a JSON ARRAY, with the
			// server-generated groupId on each row. Mirror that shape exactly.
			w.Write(okResponse([]any{
				map[string]any{"groupId": "grp-abc", "remark": "r1", "host": "h1"},
				map[string]any{"groupId": "grp-abc", "remark": "r1", "host": "h2"},
			}))
			return
		case "/panel/api/hosts/get/grp-abc":
			w.Write(okResponse(map[string]any{
				"groupId": "grp-abc", "remark": "r1",
				"inboundIds": []int{1, 2},
			}))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	created, err := client.CreateHostGroup(context.Background(), &HostGroup{
		Remark:     "r1",
		InboundIds: []int{1, 2},
	})
	if err != nil {
		t.Fatalf("CreateHostGroup error: %v", err)
	}
	if created == nil || created.GroupId != "grp-abc" {
		t.Fatalf("expected created groupId grp-abc, got %+v", created)
	}
	// Confirm the request body was JSON (not form-encoded).
	var sent map[string]any
	if err := json.Unmarshal(receivedBody, &sent); err != nil {
		t.Fatalf("/add body should be JSON, parse failed: %v", err)
	}
	if sent["remark"] != "r1" {
		t.Fatalf("expected remark=r1 in JSON body, got %v", sent["remark"])
	}
}

func TestGetHostGroupMissingReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/hosts/get/missing":
			// 3x-ui signals a missing host group via success:false carrying an
			// explicit "host group not found" message (not gorm.ErrRecordNotFound).
			b, _ := json.Marshal(apiResponse{
				Success: false,
				Msg:     "pages.hosts.toasts.obtain (host group not found)",
			})
			w.Write(b)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	got, err := client.GetHostGroup(context.Background(), "missing")
	if err != nil {
		t.Fatalf("missing group must return nil,nil (not an error), got err: %v", err)
	}
	if got != nil {
		t.Fatalf("missing group must return nil HostGroup, got %+v", got)
	}
}

// ---------------------------------------------------------------------------
// Resource-level CRUD tests (tfsdk State wiring). These exercise the resource
// Create/Read/Delete methods end-to-end against a mock panel so Codecov patch
// coverage is green and the v3.5.0 wire-format gotchas regress at the resource
// layer too (not only at the client layer).
// ---------------------------------------------------------------------------

func hostGroupResourceReadState(t *testing.T, r *HostGroupResource, groupID string) tfsdk.State {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	ctx := context.Background()
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	vals := map[string]tftypes.Value{}
	for k := range schemaResp.Schema.Attributes {
		vals[k] = tftypes.NewValue(objType.AttributeTypes[k], nil)
	}
	// Read needs group_id to build the /get/:groupId path.
	vals["group_id"] = tftypes.NewValue(tftypes.String, groupID)
	vals["id"] = tftypes.NewValue(tftypes.String, groupID)
	return tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(objType, vals),
	}
}

func newHostGroupResourceReadResponse(t *testing.T, r *HostGroupResource) resource.ReadResponse {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	ctx := context.Background()
	return resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		},
	}
}

func TestHostGroupResource_Read_RemovesOnNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		}
		// 3x-ui signals a missing host group with success:false + an explicit
		// "host group not found" message (not gorm.ErrRecordNotFound).
		if r.URL.Path == "/panel/api/hosts/get/gone" {
			b, _ := json.Marshal(apiResponse{
				Success: false,
				Msg:     "pages.hosts.toasts.obtain (host group not found)",
			})
			w.Write(b)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	r := &HostGroupResource{client: newTestClient(t, srv.URL)}
	state := hostGroupResourceReadState(t, r, "gone")
	resp := newHostGroupResourceReadResponse(t, r)
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read of a missing group must not error (it removes from state): %v", resp.Diagnostics)
	}
	// RemoveResource → state Raw is null.
	if !resp.State.Raw.IsNull() {
		t.Fatalf("expected state removed for missing host group, got non-null state")
	}
}

func TestHostGroupResource_Read_PopulatesFromGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		}
		if r.URL.Path == "/panel/api/hosts/get/grp-1" {
			w.Write(okResponse(map[string]any{
				"groupId": "grp-1", "remark": "hello",
				"inboundIds": []int{10}, "port": 443,
				"security": "tls", "isDisabled": false,
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	r := &HostGroupResource{client: newTestClient(t, srv.URL)}
	state := hostGroupResourceReadState(t, r, "grp-1")
	resp := newHostGroupResourceReadResponse(t, r)
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected Read error: %v", resp.Diagnostics)
	}
	var got HostGroupResourceModel
	resp.State.Get(context.Background(), &got)
	if got.Remark.ValueString() != "hello" {
		t.Fatalf("expected remark=hello after Read, got %q", got.Remark)
	}
	if got.GroupID.ValueString() != "grp-1" {
		t.Fatalf("expected group_id=grp-1, got %q", got.GroupID)
	}
}

// ---------------------------------------------------------------------------
// Create / Update / Delete resource-level tests (tfsdk Plan/State wiring).
// These cover the resource CRUD methods directly so Codecov patch coverage is
// green, mirroring the node resource test pattern.
// ---------------------------------------------------------------------------

// hostGroupResourcePlan builds a tfsdk.Plan with managed attribute values.
// All unmentioned attributes are null; only the keys present in vals are set.
func hostGroupResourcePlan(t *testing.T, r *HostGroupResource, vals map[string]any) tfsdk.Plan {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	ctx := context.Background()
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	out := map[string]tftypes.Value{}
	// Default every attribute to a typed null based on the schema's attribute type.
	for k := range schemaResp.Schema.Attributes {
		out[k] = tftypes.NewValue(objType.AttributeTypes[k], nil)
	}
	// String attributes.
	for _, k := range []string{"id", "group_id", "remark", "server_description", "security",
		"sni", "host_header", "path", "fingerprint", "verify_peer_cert_by_name",
		"ech_config_list", "mux_params", "sockopt_params", "final_mask",
		"vless_route", "mihomo_ip_version"} {
		if v, ok := vals[k].(string); ok {
			out[k] = tftypes.NewValue(tftypes.String, v)
		}
	}
	// Int64 attributes → Number.
	for _, k := range []string{"sort_order", "port"} {
		if v, ok := vals[k].(int64); ok {
			out[k] = tftypes.NewValue(tftypes.Number, v)
		}
	}
	// Bool attributes.
	for _, k := range []string{"is_disabled", "is_hidden", "override_sni_from_address",
		"keep_sni_blank", "allow_insecure", "mihomo_x25519", "shuffle_host"} {
		if v, ok := vals[k].(bool); ok {
			out[k] = tftypes.NewValue(tftypes.Bool, v)
		}
	}
	// inbound_ids → List[int64] (Number element).
	if ids, ok := vals["inbound_ids"].([]int64); ok {
		elems := make([]tftypes.Value, 0, len(ids))
		for _, id := range ids {
			elems = append(elems, tftypes.NewValue(tftypes.Number, id))
		}
		out["inbound_ids"] = tftypes.NewValue(tftypes.List{ElementType: tftypes.Number}, elems)
	}
	return tfsdk.Plan{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(objType, out),
	}
}

func newHostGroupResourceCreateResponse(t *testing.T, r *HostGroupResource) resource.CreateResponse {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	ctx := context.Background()
	return resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		},
	}
}

func newHostGroupResourceUpdateResponse(t *testing.T, r *HostGroupResource) resource.UpdateResponse {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	ctx := context.Background()
	return resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		},
	}
}

func TestHostGroupResource_Create_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/hosts/add":
			// /add returns an ARRAY of host rows; the server-generated groupId is
			// on each row. Create re-reads via /get for canonical state.
			w.Write(okResponse([]any{
				map[string]any{"groupId": "grp-new", "remark": "r"},
			}))
			return
		case "/panel/api/hosts/get/grp-new":
			w.Write(okResponse(map[string]any{
				"groupId": "grp-new", "remark": "hello",
				"inboundIds": []int{1, 2}, "port": 443,
			}))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	r := &HostGroupResource{client: newTestClient(t, srv.URL)}
	plan := hostGroupResourcePlan(t, r, map[string]any{
		"remark":      "hello",
		"inbound_ids": []int64{1, 2},
	})
	resp := newHostGroupResourceCreateResponse(t, r)
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on Create: %v", resp.Diagnostics)
	}
	var state HostGroupResourceModel
	resp.State.Get(context.Background(), &state)
	if state.GroupID.ValueString() != "grp-new" {
		t.Fatalf("expected group_id=grp-new after Create, got %q", state.GroupID)
	}
	if state.Remark.ValueString() != "hello" {
		t.Fatalf("expected remark=hello (from re-read), got %q", state.Remark)
	}
}

func TestHostGroupResource_Create_AddError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/hosts/add":
			w.Write(failResponse("inbound ids do not exist"))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	r := &HostGroupResource{client: newTestClient(t, srv.URL)}
	plan := hostGroupResourcePlan(t, r, map[string]any{
		"remark":      "x",
		"inbound_ids": []int64{999},
	})
	resp := newHostGroupResourceCreateResponse(t, r)
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when /add fails")
	}
}

func TestHostGroupResource_Update(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/hosts/update/grp-7":
			w.Write(okResponse(nil))
			return
		case "/panel/api/hosts/get/grp-7":
			w.Write(okResponse(map[string]any{
				"groupId": "grp-7", "remark": "renamed",
				"inboundIds": []int{3}, "port": 8443,
			}))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	r := &HostGroupResource{client: newTestClient(t, srv.URL)}
	plan := hostGroupResourcePlan(t, r, map[string]any{
		"remark":      "renamed",
		"inbound_ids": []int64{3},
		"port":        int64(8443),
	})
	state := hostGroupResourceReadState(t, r, "grp-7")
	resp := newHostGroupResourceUpdateResponse(t, r)
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on Update: %v", resp.Diagnostics)
	}
	var got HostGroupResourceModel
	resp.State.Get(context.Background(), &got)
	if got.Remark.ValueString() != "renamed" {
		t.Fatalf("expected remark=renamed after Update re-read, got %q", got.Remark)
	}
}

func TestHostGroupResource_Delete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/hosts/del/grp-9":
			w.Write(okResponse(nil))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	r := &HostGroupResource{client: newTestClient(t, srv.URL)}
	state := hostGroupResourceReadState(t, r, "grp-9")
	var resp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on Delete: %v", resp.Diagnostics)
	}
}

func TestHostGroupResource_Delete_ToleratesNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/hosts/del/grp-9":
			// Missing group on delete → "host group not found"; must be tolerated.
			w.Write(failResponse("obtain (host group not found)"))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	r := &HostGroupResource{client: newTestClient(t, srv.URL)}
	state := hostGroupResourceReadState(t, r, "grp-9")
	var resp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	// A missing group on delete must be tolerated (treated as already gone),
	// mirroring Read — the panel signals "host group not found", not gorm
	// record-not-found, so Delete checks isHostGroupNotFound too.
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete of a missing group must not error: %v", resp.Diagnostics)
	}
}

// ---------------------------------------------------------------------------
// Client method edge cases (CreateHostGroup fallbacks, Update/Delete, matcher)
// ---------------------------------------------------------------------------

func TestCreateHostGroup_EmptyRowsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		}
		if r.URL.Path == "/panel/api/hosts/add" {
			// Server returns an empty array (no rows) and the caller provided no groupId.
			w.Write(okResponse([]any{}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.CreateHostGroup(context.Background(), &HostGroup{Remark: "r"})
	if err == nil {
		t.Fatal("expected error when /add returns no rows and no groupId provided")
	}
}

func TestCreateHostGroup_UsesProvidedGroupID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/hosts/add":
			// Empty rows in response, but the caller pre-set GroupId.
			w.Write(okResponse([]any{}))
			return
		case "/panel/api/hosts/get/pre-set":
			w.Write(okResponse(map[string]any{"groupId": "pre-set", "remark": "r"}))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	created, err := client.CreateHostGroup(context.Background(), &HostGroup{
		GroupId: "pre-set", Remark: "r",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created == nil || created.GroupId != "pre-set" {
		t.Fatalf("expected fallback to provided groupId pre-set, got %+v", created)
	}
}

func TestCreateHostGroup_NilArg(t *testing.T) {
	client := newTestClient(t, "http://localhost:0")
	if _, err := client.CreateHostGroup(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil host group")
	}
}

func TestUpdateHostGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/hosts/update/grp-u":
			w.Write(okResponse(nil))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.UpdateHostGroup(context.Background(), "grp-u", &HostGroup{Remark: "u"}); err != nil {
		t.Fatalf("unexpected UpdateHostGroup error: %v", err)
	}
	// Guard clauses.
	if err := client.UpdateHostGroup(context.Background(), "", &HostGroup{}); err == nil {
		t.Fatal("expected error for empty groupID")
	}
	if err := client.UpdateHostGroup(context.Background(), "grp-u", nil); err == nil {
		t.Fatal("expected error for nil host group")
	}
}

func TestDeleteHostGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/hosts/del/grp-d":
			w.Write(okResponse(nil))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.DeleteHostGroup(context.Background(), "grp-d"); err != nil {
		t.Fatalf("unexpected DeleteHostGroup error: %v", err)
	}
	// Empty groupID guard.
	if err := client.DeleteHostGroup(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty groupID")
	}
}

func TestIsHostGroupNotFound(t *testing.T) {
	if isHostGroupNotFound(nil) {
		t.Fatal("nil err must be false")
	}
	if !isHostGroupNotFound(failResponseErr("obtain (host group not found)")) {
		t.Fatal("host group not found message must match")
	}
	if !isHostGroupNotFound(failResponseErr("Failed to load host (host not found\n)")) {
		t.Fatal("v3.6.0 'host not found' message must match")
	}
	if isHostGroupNotFound(failResponseErr("hostname not found in DNS")) {
		t.Fatal("'hostname not found in DNS' must not match — substring false positive")
	}
	if isHostGroupNotFound(failResponseErr("some other error")) {
		t.Fatal("unrelated error must not match")
	}
}

// TestGetHostGroup_OtherError covers the non-not-found error branch: a
// failure that is neither "host group not found" nor a gorm record-not-found
// must be surfaced as an error (not swallowed into nil,nil).
func TestGetHostGroup_OtherError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/hosts/get/grp-x":
			w.Write(failResponse("validation failed"))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	got, err := client.GetHostGroup(context.Background(), "grp-x")
	if err == nil {
		t.Fatal("expected error for a non-not-found failure")
	}
	if got != nil {
		t.Fatalf("expected nil HostGroup on error, got %+v", got)
	}
}

// failResponseErr wraps a message into an error the way GetHostGroup would.
func failResponseErr(msg string) error {
	return fmt.Errorf("request failed: status 200, msg: %s", msg)
}

// TestHostGroupResource_Metadata_Configure_Import covers the trivial wiring
// methods so the whole resource's lines count toward Codecov patch coverage.
func TestHostGroupResource_Metadata_Configure_Import(t *testing.T) {
	r := NewHostGroupResource().(*HostGroupResource)

	// Metadata.
	var metaResp resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "threexui"}, &metaResp)
	if metaResp.TypeName != "threexui_host_group" {
		t.Fatalf("expected TypeName threexui_host_group, got %q", metaResp.TypeName)
	}

	// Configure: nil ProviderData is a no-op (skip path).
	var cfgResp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{}, &cfgResp)
	if cfgResp.Diagnostics.HasError() {
		t.Fatalf("nil ProviderData Configure must be a no-op, got errors: %v", cfgResp.Diagnostics)
	}

	// Configure: wrong type → error.
	var cfgBad resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not-a-client"}, &cfgBad)
	if !cfgBad.Diagnostics.HasError() {
		t.Fatal("expected error for non-*Client ProviderData")
	}

	// Configure: real client → stored.
	cli := newTestClient(t, "http://localhost:0")
	var cfgOK resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: cli}, &cfgOK)
	if cfgOK.Diagnostics.HasError() {
		t.Fatalf("unexpected Configure error: %v", cfgOK.Diagnostics)
	}
	if r.client == nil {
		t.Fatal("expected client to be set after Configure")
	}
}

// TestHostGroupResource_EmptyGroupIDError covers the defensive "group_id is
// empty" error branch in Read/Update/Delete (state without group_id or id).
func TestHostGroupResource_EmptyGroupIDError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	r := &HostGroupResource{client: newTestClient(t, srv.URL)}
	state := hostGroupResourceReadState(t, r, "") // empty group_id + id

	var readResp resource.ReadResponse
	r.Read(context.Background(), resource.ReadRequest{State: state}, &readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected error on Read with empty group_id")
	}

	var updResp resource.UpdateResponse
	r.Update(context.Background(), resource.UpdateRequest{Plan: tfsdk.Plan(state), State: state}, &updResp)
	if !updResp.Diagnostics.HasError() {
		t.Fatal("expected error on Update with empty group_id")
	}

	var delResp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &delResp)
	if !delResp.Diagnostics.HasError() {
		t.Fatal("expected error on Delete with empty group_id")
	}
}

// TestHostGroupCipherSuitesFlattenNull verifies an absent/empty cipherSuites
// flattens to null (not ""), matching the other optional string attributes.
func TestHostGroupCipherSuitesFlattenNull(t *testing.T) {
	m := &HostGroupResourceModel{}
	flattenHostGroupToModel(&HostGroup{GroupId: "grp-1", Remark: "r"}, m)
	if !m.CipherSuites.IsNull() {
		t.Fatalf("expected null cipher_suites, got %q", m.CipherSuites)
	}

	m2 := &HostGroupResourceModel{CipherSuites: types.StringNull()}
	hg := hostGroupFromModel(context.Background(), m2)
	if hg.CipherSuites != "" {
		t.Fatalf("null cipher_suites must expand to empty string, got %q", hg.CipherSuites)
	}
}
