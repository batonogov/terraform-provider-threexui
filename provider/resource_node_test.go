package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ---------------------------------------------------------------------------
// Resource surface (Metadata / Configure / Schema / Delete / ImportState)
// ---------------------------------------------------------------------------

func TestNodeResource_Metadata(t *testing.T) {
	r := &NodeResource{}
	var resp resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "threexui",
	}, &resp)
	if resp.TypeName != "threexui_node" {
		t.Fatalf("expected threexui_node, got %s", resp.TypeName)
	}
}

func TestNodeResource_Schema(t *testing.T) {
	r := &NodeResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema errors: %v", resp.Diagnostics)
	}
	// Required managed attributes must be present.
	required := []string{"name", "address", "port"}
	for _, attr := range required {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Fatalf("required attribute %q missing from schema", attr)
		}
	}
	// Sensitive managed attributes must stay Sensitive (security rule).
	for _, attr := range []string{"api_token", "pinned_cert_sha256"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Fatalf("sensitive attribute %q missing", attr)
		}
		if !a.IsSensitive() {
			t.Fatalf("attribute %q must be Sensitive", attr)
		}
	}
}

func TestNodeResource_Configure(t *testing.T) {
	client := newTestClient(t, "http://localhost:0")
	r := &NodeResource{}

	// nil ProviderData — no-op.
	var resp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on nil ProviderData: %v", resp.Diagnostics)
	}

	// Valid client.
	resp = resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: client}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error with valid client: %v", resp.Diagnostics)
	}
	if r.client == nil {
		t.Fatal("client not set after Configure")
	}

	// Wrong type.
	resp = resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: "not a client"}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error with wrong ProviderData type")
	}
}

func TestNodeResource_Delete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/del/7":
			w.Write(okResponse(nil))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	r := &NodeResource{client: newTestClient(t, srv.URL)}
	state := nodeResourceDeleteState(t, r, "7")
	var resp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on Delete: %v", resp.Diagnostics)
	}
}

func TestNodeResource_Delete_InboundsAttached(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/del/7":
			// DB-002 guard: panel refuses with success:false mentioning inbounds.
			w.Write(failResponse("cannot delete node: 2 inbound(s) still attached to it"))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	r := &NodeResource{client: newTestClient(t, srv.URL)}
	state := nodeResourceDeleteState(t, r, "7")
	var resp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error when inbounds are still attached")
	}
}

// TestNodeResource_Delete_ToleratesNotFound verifies the defensive record-
// not-found branch in Delete. Note: the real 3x-ui panel returns SUCCESS for
// deleting an already-absent node (service/node.go Delete ignores the First
// error and gorm Delete with WHERE does not error on zero rows), so the
// realistic 'already gone' path is covered by TestNodeResource_Delete_Success.
// This test pins resilience to a hypothetical record-not-found response so
// the guard at resource_node.go does not regress into a hard error.
func TestNodeResource_Delete_ToleratesNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/del/7":
			w.Write(failResponse("obtain (record not found)"))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	r := &NodeResource{client: newTestClient(t, srv.URL)}
	state := nodeResourceDeleteState(t, r, "7")
	var resp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	// A record-not-found on delete must be tolerated (treated as already gone).
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on record-not-found Delete: %v", resp.Diagnostics)
	}
}

func TestNodeResource_Read_RemovesOnNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		}
		// The panel signals a missing node with HTTP 200 + success:false carrying
		// a gorm "record not found" message — NOT HTTP 404 (util.go jsonMsgObj).
		if r.URL.Path == "/panel/api/nodes/get/42" {
			w.Write(failResponse("obtain (record not found)"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	r := &NodeResource{client: newTestClient(t, srv.URL)}

	// State carries an id pointing at a node the panel reports as missing.
	state := nodeResourceReadState(t, r, "42")
	resp := newNodeResourceReadResponse(t, r)
	r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on not-found Read: %v", resp.Diagnostics)
	}
}

// ---------------------------------------------------------------------------
// Pure helpers: parseNodeID / nodeFromModel / flattenNodeToModel round-trip
// ---------------------------------------------------------------------------

func TestParseNodeID(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"1", 1, false},
		{"42", 42, false},
		{"0", 0, true},   // zero is invalid
		{"abc", 0, true}, // non-numeric
		{"", 0, true},
	}
	for _, tc := range cases {
		got, err := parseNodeID(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("parseNodeID(%q) expected error, got %d", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parseNodeID(%q) unexpected error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("parseNodeID(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestNodeFromModel_FlattenRoundtrip(t *testing.T) {
	plan := &NodeResourceModel{
		Name:                types.StringValue("de-fra-1"),
		Remark:              types.StringValue("Frankfurt"),
		Scheme:              types.StringValue("https"),
		Address:             types.StringValue("node1.example.com"),
		Port:                types.Int64Value(2053),
		BasePath:            types.StringValue("/"),
		ApiToken:            types.StringValue("tok-123"),
		Enable:              types.BoolValue(true),
		AllowPrivateAddress: types.BoolValue(false),
		TlsVerifyMode:       types.StringValue("pin"),
		PinnedCertSha256:    types.StringValue("abcdef"),
		InboundSyncMode:     types.StringValue("selected"),
		InboundTags:         []types.String{types.StringValue("tag-a"), types.StringValue("tag-b")},
		OutboundTag:         types.StringValue("proxy-out"),
	}

	n := nodeFromModel(plan)
	if n.Name != "de-fra-1" || n.Address != "node1.example.com" || n.Port != 2053 {
		t.Fatalf("nodeFromModel wrong managed fields: %+v", n)
	}
	if n.ApiToken != "tok-123" || n.PinnedCertSha256 != "abcdef" {
		t.Fatalf("nodeFromModel wrong sensitive fields: %+v", n)
	}
	if n.TlsVerifyMode != "pin" || n.InboundSyncMode != "selected" {
		t.Fatalf("nodeFromModel wrong mode fields: %+v", n)
	}
	if len(n.InboundTags) != 2 || n.InboundTags[0] != "tag-a" {
		t.Fatalf("nodeFromModel wrong inbound tags: %v", n.InboundTags)
	}
	if n.OutboundTag != "proxy-out" {
		t.Fatalf("nodeFromModel wrong outbound tag: %q", n.OutboundTag)
	}
	if !n.Enable {
		t.Fatal("Enable should default/propagate to true")
	}

	// Flatten back and assert the managed fields survive the round-trip.
	var got NodeResourceModel
	got.ApiToken = types.StringValue("tok-123") // preserve (flatten keeps non-empty remote)
	got.PinnedCertSha256 = types.StringValue("abcdef")
	flattenNodeToModel(n, &got)
	if got.Name.ValueString() != "de-fra-1" {
		t.Fatalf("flatten Name mismatch: %s", got.Name.ValueString())
	}
	if got.Port.ValueInt64() != 2053 {
		t.Fatalf("flatten Port mismatch: %d", got.Port.ValueInt64())
	}
	if got.TlsVerifyMode.ValueString() != "pin" {
		t.Fatalf("flatten TlsVerifyMode mismatch: %s", got.TlsVerifyMode.ValueString())
	}
	if len(got.InboundTags) != 2 || got.InboundTags[1].ValueString() != "tag-b" {
		t.Fatalf("flatten InboundTags mismatch: %v", got.InboundTags)
	}
}

// flattenNodeToModel must NOT clobber sensitive attrs with empty remote
// values (defensive; per #314 R1 the panel returns them raw, but an empty
// remote means "unset", not "erase").
func TestFlattenNodeToModel_PreservesEmptySensitive(t *testing.T) {
	n := &Node{Name: "n", Address: "a", Port: 1, ApiToken: "", PinnedCertSha256: ""}
	m := &NodeResourceModel{
		ApiToken:         types.StringValue("kept-tok"),
		PinnedCertSha256: types.StringValue("kept-pin"),
	}
	flattenNodeToModel(n, m)
	if m.ApiToken.ValueString() != "kept-tok" {
		t.Fatalf("ApiToken was clobbered with empty remote: %q", m.ApiToken.ValueString())
	}
	if m.PinnedCertSha256.ValueString() != "kept-pin" {
		t.Fatalf("PinnedCertSha256 was clobbered with empty remote: %q", m.PinnedCertSha256.ValueString())
	}
}

// ---------------------------------------------------------------------------
// Client methods: CreateNode / GetNode
// ---------------------------------------------------------------------------

func TestCreateNode(t *testing.T) {
	var receivedForm string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/add":
			r.ParseForm()
			receivedForm = r.Form.Encode()
			w.Write(okResponse(map[string]any{
				"id":         7,
				"name":       "de-fra-1",
				"address":    "node1.example.com",
				"port":       2053,
				"apiToken":   "tok-123",
				"enable":     true,
				"guid":       "g-7",
				"status":     "unknown",
				"transitive": false,
			}))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	created, err := client.CreateNode(context.Background(), &Node{
		Name: "de-fra-1", Scheme: "https", Address: "node1.example.com",
		Port: 2053, ApiToken: "tok-123", Enable: true,
		InboundTags: []string{},
	})
	if err != nil {
		t.Fatalf("CreateNode error: %v", err)
	}
	if created.Id != 7 {
		t.Fatalf("expected created id 7, got %d", created.Id)
	}
	// The form must carry the managed fields with upstream names.
	for _, want := range []string{"name=de-fra-1", "address=node1.example.com", "port=2053", "apiToken=tok-123", "inboundTags=%5B%5D"} {
		if !contains(receivedForm, want) {
			t.Fatalf("create form missing %q in %q", want, receivedForm)
		}
	}
}

func TestCreateNode_Nil(t *testing.T) {
	client := newTestClient(t, "http://localhost:0")
	if _, err := client.CreateNode(context.Background(), nil); err == nil {
		t.Fatal("expected error on nil node")
	}
}

func TestGetNode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/get/7":
			w.Write(okResponse(map[string]any{
				"id": 7, "name": "de-fra-1", "address": "node1.example.com",
				"port": 2053, "status": "online", "guid": "g-7",
			}))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	got, err := client.GetNode(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetNode error: %v", err)
	}
	if got.Id != 7 || got.Name != "de-fra-1" || got.Status != "online" {
		t.Fatalf("unexpected node: %+v", got)
	}
}

func TestGetNode_ZeroID(t *testing.T) {
	client := newTestClient(t, "http://localhost:0")
	if _, err := client.GetNode(context.Background(), 0); err == nil {
		t.Fatal("expected error on zero id")
	}
}

func TestNodeToForm(t *testing.T) {
	form := nodeToForm(&Node{
		Name: "n", Scheme: "https", Address: "a", Port: 2053,
		ApiToken: "tok", Enable: true, InboundTags: []string{"x", "y"},
	})
	if form.Get("name") != "n" {
		t.Fatalf("name: %q", form.Get("name"))
	}
	if form.Get("port") != "2053" {
		t.Fatalf("port: %q", form.Get("port"))
	}
	// inboundTags serialized as JSON array string.
	if form.Get("inboundTags") != "[\"x\",\"y\"]" {
		t.Fatalf("inboundTags: %q", form.Get("inboundTags"))
	}
	if form.Get("enable") != "true" {
		t.Fatalf("enable: %q", form.Get("enable"))
	}
}

func TestNodeToForm_Nil(t *testing.T) {
	if form := nodeToForm(nil); len(form) != 0 {
		t.Fatalf("expected empty form for nil, got %v", form)
	}
}

// nodeResourceReadState builds a tfsdk.State for the node resource with the
// given id, so Read can be exercised in unit tests.
func nodeResourceReadState(t *testing.T, r *NodeResource, id string) tfsdk.State {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	ctx := context.Background()
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	vals := map[string]tftypes.Value{}
	for k := range schemaResp.Schema.Attributes {
		vals[k] = tftypes.NewValue(objType.AttributeTypes[k], nil)
	}
	// Only the id is needed for Read; everything else stays null.
	vals["id"] = tftypes.NewValue(tftypes.String, id)
	return tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(objType, vals),
	}
}

// newNodeResourceReadResponse builds a ReadResponse with a properly
// initialised State so that resp.State.RemoveResource works in unit tests.
func newNodeResourceReadResponse(t *testing.T, r *NodeResource) resource.ReadResponse {
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

// TestNodeResource_Create exercises the full Create path against a mock panel:
// plan -> CreateNode -> re-read GetNode -> state. Covers the create flow that
// acc-tests normally cover, so it counts toward Codecov patch coverage.
func TestNodeResource_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/add":
			r.ParseForm()
			if r.Form.Get("name") != "de-fra-1" {
				w.Write(failResponse("bad name"))
				return
			}
			w.Write(okResponse(map[string]any{
				"id": 7, "name": "de-fra-1", "address": "node1.example.com",
				"port": 2053, "enable": true,
			}))
			return
		case "/panel/api/nodes/get/7":
			w.Write(okResponse(map[string]any{
				"id": 7, "name": "de-fra-1", "address": "node1.example.com",
				"port": 2053, "enable": true, "status": "online", "guid": "g-7",
				"latencyMs": 12,
			}))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	r := &NodeResource{client: newTestClient(t, srv.URL)}
	plan := nodeResourcePlan(t, r, map[string]any{
		"name":    "de-fra-1",
		"address": "node1.example.com",
		"port":    int64(2053),
		"scheme":  "https",
		"enable":  true,
	})

	resp := newNodeResourceCreateResponse(t, r)
	r.Create(context.Background(), resource.CreateRequest{Plan: plan, Config: planToConfig(plan)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on Create: %v", resp.Diagnostics)
	}

	var state NodeResourceModel
	resp.State.Get(context.Background(), &state)
	if state.ID.ValueString() != "7" {
		t.Fatalf("expected id 7, got %s", state.ID.ValueString())
	}
	if state.Status.ValueString() != "online" {
		t.Fatalf("expected observed status online (from re-read), got %s", state.Status.ValueString())
	}
}

// TestNodeResource_Create_ReachabilityError verifies the create error path
// surfaces the ensureReachability hint.
func TestNodeResource_Create_ReachabilityError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/add":
			w.Write(failResponse("node unreachable"))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	r := &NodeResource{client: newTestClient(t, srv.URL)}
	plan := nodeResourcePlan(t, r, map[string]any{
		"name": "x", "address": "unreachable.example", "port": int64(2053),
	})
	resp := newNodeResourceCreateResponse(t, r)
	r.Create(context.Background(), resource.CreateRequest{Plan: plan, Config: planToConfig(plan)}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error on unreachable node create")
	}
}

// nodeResourcePlan builds a tfsdk.Plan with the given managed attribute values.
func nodeResourcePlan(t *testing.T, r *NodeResource, vals map[string]any) tfsdk.Plan {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	ctx := context.Background()
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	out := map[string]tftypes.Value{}
	for k := range schemaResp.Schema.Attributes {
		switch k {
		case "port", "latency_ms", "last_heartbeat", "uptime_secs", "net_up",
			"net_down", "config_dirty_at", "inbound_count", "client_count",
			"online_count", "active_count", "disabled_count", "depleted_count",
			"created_at", "updated_at",
			"api_token_wo_version", "pinned_cert_sha256_wo_version":
			out[k] = tftypes.NewValue(tftypes.Number, nil)
		case "enable", "allow_private_address", "config_dirty", "transitive":
			out[k] = tftypes.NewValue(tftypes.Bool, nil)
		case "cpu_pct", "mem_pct":
			out[k] = tftypes.NewValue(tftypes.Number, nil)
		case "inbound_tags", "unavailable_metrics":
			out[k] = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil)
		default:
			out[k] = tftypes.NewValue(tftypes.String, nil)
		}
	}
	stringFields := []string{"name", "address", "scheme", "id", "remark", "base_path",
		"api_token", "api_token_wo", "pinned_cert_sha256", "pinned_cert_sha256_wo",
		"tls_verify_mode", "inbound_sync_mode", "outbound_tag"}
	for _, k := range stringFields {
		if v, ok := vals[k]; ok {
			out[k] = tftypes.NewValue(tftypes.String, v)
		}
	}
	if v, ok := vals["port"]; ok {
		out["port"] = tftypes.NewValue(tftypes.Number, v)
	}
	if v, ok := vals["enable"]; ok {
		out["enable"] = tftypes.NewValue(tftypes.Bool, v)
	}
	_ = objType
	return tfsdk.Plan{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object), out),
	}
}

func newNodeResourceCreateResponse(t *testing.T, r *NodeResource) resource.CreateResponse {
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

// TestNodeResource_Update covers the real M3 Update: form-POST
// /panel/api/nodes/update/:id then re-read GET /get/:id. The server restarts
// xray itself on outbound_tag change, so the provider does not.
func TestNodeResource_Update(t *testing.T) {
	var updatedForm string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/update/7":
			r.ParseForm()
			updatedForm = r.Form.Encode()
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/get/7":
			w.Write(okResponse(map[string]any{
				"id": 7, "name": "de-fra-1", "address": "node1.example.com",
				"port": 2053, "enable": false, "remark": "renamed",
			}))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	r := &NodeResource{client: newTestClient(t, srv.URL)}
	plan := nodeResourcePlan(t, r, map[string]any{
		"name": "de-fra-1", "address": "node1.example.com", "port": int64(2053),
		"id": "7", "remark": "renamed",
	})

	resp := newNodeResourceUpdateResponse(t, r)
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, Config: planToConfig(plan), State: tfsdk.State(plan)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on Update: %v", resp.Diagnostics)
	}
	// The update form must carry the managed field changes.
	if !contains(updatedForm, "remark=renamed") {
		t.Fatalf("update form missing remark: %q", updatedForm)
	}

	var state NodeResourceModel
	resp.State.Get(context.Background(), &state)
	if state.Remark.ValueString() != "renamed" {
		t.Fatalf("expected re-read remark 'renamed', got %q", state.Remark.ValueString())
	}
}

func TestNodeResource_Update_RemovesOnNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/update/7":
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/get/7":
			// Node vanished between update and re-read.
			w.Write(failResponse("obtain (record not found)"))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	r := &NodeResource{client: newTestClient(t, srv.URL)}
	plan := nodeResourcePlan(t, r, map[string]any{
		"name": "de-fra-1", "address": "node1.example.com", "port": int64(2053), "id": "7",
	})
	resp := newNodeResourceUpdateResponse(t, r)
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, Config: planToConfig(plan), State: tfsdk.State(plan)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on Update re-read not-found: %v", resp.Diagnostics)
	}
}

func TestNewNodeResource(t *testing.T) {
	if NewNodeResource() == nil {
		t.Fatal("NewNodeResource returned nil")
	}
}

func newNodeResourceUpdateResponse(t *testing.T, r *NodeResource) resource.UpdateResponse {
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

// nodeResourceDeleteState builds a tfsdk.State with just the id, for Delete.
func nodeResourceDeleteState(t *testing.T, r *NodeResource, id string) tfsdk.State {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	ctx := context.Background()
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	vals := map[string]tftypes.Value{}
	for k := range schemaResp.Schema.Attributes {
		vals[k] = tftypes.NewValue(objType.AttributeTypes[k], nil)
	}
	vals["id"] = tftypes.NewValue(tftypes.String, id)
	return tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(objType, vals),
	}
}

// ---------------------------------------------------------------------------
// Client methods: UpdateNode / DeleteNode
// ---------------------------------------------------------------------------

func TestUpdateNode(t *testing.T) {
	var receivedForm string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/update/7":
			r.ParseForm()
			receivedForm = r.Form.Encode()
			w.Write(okResponse(nil))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	err := client.UpdateNode(context.Background(), 7, &Node{
		Name: "de-fra-1", Scheme: "https", Address: "node1.example.com",
		Port: 2053, Remark: "updated-remark", OutboundTag: "proxy-out",
		InboundTags: []string{},
	})
	if err != nil {
		t.Fatalf("UpdateNode error: %v", err)
	}
	for _, want := range []string{"remark=updated-remark", "outboundTag=proxy-out", "address=node1.example.com"} {
		if !contains(receivedForm, want) {
			t.Fatalf("update form missing %q in %q", want, receivedForm)
		}
	}
}

func TestUpdateNode_NilAndZeroID(t *testing.T) {
	client := newTestClient(t, "http://localhost:0")
	if err := client.UpdateNode(context.Background(), 7, nil); err == nil {
		t.Fatal("expected error on nil node")
	}
	if err := client.UpdateNode(context.Background(), 0, &Node{Name: "x"}); err == nil {
		t.Fatal("expected error on zero id")
	}
}

func TestDeleteNode(t *testing.T) {
	var delPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/del/7":
			delPath = r.URL.Path
			// Gin uses POST for delete (not DELETE); confirm the method too.
			if r.Method != http.MethodPost {
				w.Write(failResponse("method not allowed"))
				return
			}
			w.Write(okResponse(nil))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.DeleteNode(context.Background(), 7); err != nil {
		t.Fatalf("DeleteNode error: %v", err)
	}
	if delPath != "/panel/api/nodes/del/7" {
		t.Fatalf("expected del hit on /panel/api/nodes/del/7, got %q", delPath)
	}
}

func TestDeleteNode_ZeroID(t *testing.T) {
	client := newTestClient(t, "http://localhost:0")
	if err := client.DeleteNode(context.Background(), 0); err == nil {
		t.Fatal("expected error on zero id")
	}
}

// planToConfig wraps a tfsdk.Plan's Schema/Raw into a tfsdk.Config so that
// Create/Update requests in unit tests carry the write-only _wo values the
// resource reads from req.Config.
func planToConfig(plan tfsdk.Plan) tfsdk.Config {
	return tfsdk.Config(plan)
}

// ---------------------------------------------------------------------------
// Write-only secrets (M4): resolve + ModifyPlan
// ---------------------------------------------------------------------------

// resolveNodeSecretsWO must copy the _wo value from config into the plain
// field on Create. This is the settings-style strategy (#314 R1).
func TestResolveNodeSecretsWO_Create(t *testing.T) {
	plan := &NodeResourceModel{}
	config := NodeResourceModel{
		ApiToken:           types.StringValue("from-state"),
		ApiTokenWO:         types.StringValue("wo-secret"),
		PinnedCertSha256:   types.StringValue("old-pin"),
		PinnedCertSha256WO: types.StringValue("wo-pin"),
	}
	resolveNodeSecretsWO(plan, config)
	if plan.ApiToken.ValueString() != "wo-secret" {
		t.Fatalf("expected api_token copied from _wo, got %q", plan.ApiToken.ValueString())
	}
	if plan.PinnedCertSha256.ValueString() != "wo-pin" {
		t.Fatalf("expected pinned_cert_sha256 copied from _wo, got %q", plan.PinnedCertSha256.ValueString())
	}
}

// resolveNodeSecretsWO must be a no-op when no _wo value is configured
// (plain path: state value survives).
func TestResolveNodeSecretsWO_NoWOConfigured(t *testing.T) {
	plan := &NodeResourceModel{ApiToken: types.StringValue("keep-me")}
	config := NodeResourceModel{}
	resolveNodeSecretsWO(plan, config)
	if plan.ApiToken.ValueString() != "keep-me" {
		t.Fatalf("plain api_token should be preserved, got %q", plan.ApiToken.ValueString())
	}
}

// resolveNodeSecretsWOUpdate only copies the secret when the _wo_version
// trigger changed (version bump OR first use when state has none).
func TestResolveNodeSecretsWOUpdate_TriggerBumped(t *testing.T) {
	plan := &NodeResourceModel{ApiTokenWOVersion: types.Int64Value(2)}
	state := NodeResourceModel{ApiTokenWOVersion: types.Int64Value(1)}
	config := NodeResourceModel{ApiTokenWO: types.StringValue("rotated")}
	resolveNodeSecretsWOUpdate(plan, state, config)
	if plan.ApiToken.ValueString() != "rotated" {
		t.Fatalf("expected api_token rotated on version bump, got %q", plan.ApiToken.ValueString())
	}
}

func TestResolveNodeSecretsWOUpdate_FirstUse(t *testing.T) {
	plan := &NodeResourceModel{ApiTokenWOVersion: types.Int64Value(1)}
	state := NodeResourceModel{ApiTokenWOVersion: types.Int64Null()}
	config := NodeResourceModel{ApiTokenWO: types.StringValue("first")}
	resolveNodeSecretsWOUpdate(plan, state, config)
	if plan.ApiToken.ValueString() != "first" {
		t.Fatalf("expected api_token set on first use, got %q", plan.ApiToken.ValueString())
	}
}

func TestResolveNodeSecretsWOUpdate_NoTrigger(t *testing.T) {
	// Version unchanged -> _wo value must NOT be resent (no-op update).
	plan := &NodeResourceModel{
		ApiToken:          types.StringValue("existing"),
		ApiTokenWOVersion: types.Int64Value(1),
	}
	state := NodeResourceModel{ApiTokenWOVersion: types.Int64Value(1)}
	config := NodeResourceModel{ApiTokenWO: types.StringValue("ignored")}
	resolveNodeSecretsWOUpdate(plan, state, config)
	if plan.ApiToken.ValueString() != "existing" {
		t.Fatalf("expected api_token unchanged on no-trigger update, got %q", plan.ApiToken.ValueString())
	}
}

// TestNodeResource_Create_WithWriteOnlyToken exercises the full Create path
// where the secret is supplied via api_token_wo: the resource must send it to
// the panel (the form carries the resolved value, not empty).
func TestNodeResource_Create_WithWriteOnlyToken(t *testing.T) {
	var receivedForm string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
			return
		case "/panel/api/nodes/add":
			r.ParseForm()
			receivedForm = r.Form.Encode()
			w.Write(okResponse(map[string]any{"id": 9, "name": "n", "address": "a", "port": 2053}))
			return
		case "/panel/api/nodes/get/9":
			w.Write(okResponse(map[string]any{"id": 9, "name": "n", "address": "a", "port": 2053}))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer srv.Close()

	r := &NodeResource{client: newTestClient(t, srv.URL)}
	plan := nodeResourcePlan(t, r, map[string]any{
		"name": "n", "address": "a", "port": int64(2053),
	})
	// Supply the secret only via write-only in config.
	planWO := setPlanValue(t, r, plan, "api_token_wo", "wo-create-secret")

	resp := newNodeResourceCreateResponse(t, r)
	r.Create(context.Background(), resource.CreateRequest{Plan: planWO, Config: planToConfig(planWO)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
	if !contains(receivedForm, "apiToken=wo-create-secret") {
		t.Fatalf("create form must carry the resolved write-only token; got %q", receivedForm)
	}
}

// setPlanValue returns a copy of the plan with the given string attribute set.
// Used to inject write-only values that nodeResourcePlan does not set by default.
func setPlanValue(t *testing.T, r *NodeResource, plan tfsdk.Plan, key, val string) tfsdk.Plan {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	ctx := context.Background()
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	vals := map[string]tftypes.Value{}
	_ = plan.Raw.As(&vals)
	for k, v := range vals {
		vals[k] = v
	}
	vals[key] = tftypes.NewValue(tftypes.String, val)
	return tfsdk.Plan{
		Schema: plan.Schema,
		Raw:    tftypes.NewValue(objType, vals),
	}
}

// NOTE: ModifyPlan itself is exercised end-to-end by the write-only acc
// tests (M4 follow-up). Its core logic lives in the generic
// modifyPlanWOVersion helper (already covered by the settings resources)
// plus resolveNodeSecretsWO/Update (covered above). Directly unit-testing
// ModifyPlan requires fully initialising resp.Plan's Schema/Raw, which is
// fragile; the resolve + woVersionTriggered unit tests below cover the
// behaviour that matters.

// ---------------------------------------------------------------------------
// ModifyPlan (M4, inlined for dual-secret support)
// ---------------------------------------------------------------------------

// modifyPlanFixture builds a Plan/State pair for ModifyPlan tests, with the
// given *_wo_version values for both secrets and a concrete api_token.
func modifyPlanFixture(t *testing.T, r *NodeResource, apiWOVer, pinWOVer int64) (tfsdk.Plan, tfsdk.State) {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	ctx := context.Background()
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	build := func(apiV, pinV int64) tfsdk.Plan {
		vals := map[string]tftypes.Value{}
		for k := range schemaResp.Schema.Attributes {
			switch k {
			case "port":
				vals[k] = tftypes.NewValue(tftypes.Number, 2053)
			case "enable", "allow_private_address", "config_dirty", "transitive":
				vals[k] = tftypes.NewValue(tftypes.Bool, nil)
			case "inbound_tags", "unavailable_metrics":
				vals[k] = tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil)
			case "api_token_wo_version":
				vals[k] = tftypes.NewValue(tftypes.Number, apiV)
			case "pinned_cert_sha256_wo_version":
				vals[k] = tftypes.NewValue(tftypes.Number, pinV)
			case "cpu_pct", "mem_pct", "latency_ms", "last_heartbeat", "uptime_secs",
				"net_up", "net_down", "config_dirty_at", "inbound_count", "client_count",
				"online_count", "active_count", "disabled_count", "depleted_count",
				"created_at", "updated_at":
				vals[k] = tftypes.NewValue(tftypes.Number, nil)
			default:
				vals[k] = tftypes.NewValue(tftypes.String, nil)
			}
		}
		vals["id"] = tftypes.NewValue(tftypes.String, "1")
		vals["api_token"] = tftypes.NewValue(tftypes.String, "prior-token")
		return tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, vals)}
	}

	plan := build(apiWOVer, pinWOVer)
	state := build(1, 1)
	return plan, tfsdk.State(state)
}

// TestNodeResource_ModifyPlan_BothSecretsTriggered is the regression test for
// the dual-secret bug the M4 review caught: when both *_wo_version change at
// once, BOTH plain secrets must be marked Unknown (a naive double call to the
// shared modifyPlanWOVersion generic would erase the first mark).
func TestNodeResource_ModifyPlan_BothSecretsTriggered(t *testing.T) {
	r := &NodeResource{}
	ctx := context.Background()
	plan, st := modifyPlanFixture(t, r, 2, 2) // both bumped

	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := &resource.ModifyPlanResponse{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)},
	}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
	var out NodeResourceModel
	resp.Plan.Get(ctx, &out)
	if !out.ApiToken.IsUnknown() {
		t.Fatalf("expected api_token marked Unknown on simultaneous rotation, got %v", out.ApiToken)
	}
	if !out.PinnedCertSha256.IsUnknown() {
		t.Fatalf("expected pinned_cert_sha256 marked Unknown on simultaneous rotation, got %v", out.PinnedCertSha256)
	}
}

func TestNodeResource_ModifyPlan_UnknownInboundTagsWithBothSecretsTriggered(t *testing.T) {
	r := &NodeResource{}
	ctx := context.Background()
	plan, st := modifyPlanFixture(t, r, 2, 2)

	diags := plan.SetAttribute(ctx, path.Root("inbound_tags"), types.ListUnknown(types.StringType))
	if diags.HasError() {
		t.Fatalf("failed to build unknown inbound_tags plan: %v", diags)
	}

	resp := &resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ModifyPlan must accept an unknown inbound_tags list: %v", resp.Diagnostics)
	}

	var inboundTags types.List
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("inbound_tags"), &inboundTags)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("failed to read inbound_tags from the modified plan: %v", resp.Diagnostics)
	}
	if !inboundTags.IsUnknown() {
		t.Fatalf("inbound_tags must remain unknown, got %v", inboundTags)
	}

	var apiToken, pinnedCertSha256 types.String
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("api_token"), &apiToken)...)
	resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("pinned_cert_sha256"), &pinnedCertSha256)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("failed to read rotated secrets from the modified plan: %v", resp.Diagnostics)
	}
	if !apiToken.IsUnknown() {
		t.Fatalf("api_token must be unknown on simultaneous rotation, got %v", apiToken)
	}
	if !pinnedCertSha256.IsUnknown() {
		t.Fatalf("pinned_cert_sha256 must be unknown on simultaneous rotation, got %v", pinnedCertSha256)
	}
}

func TestNodeResource_ModifyPlan_SingleSecretTriggered(t *testing.T) {
	r := &NodeResource{}
	ctx := context.Background()
	// Only api_token_wo_version bumps; pinned stays the same.
	plan, st := modifyPlanFixture(t, r, 2, 1)

	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := &resource.ModifyPlanResponse{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)},
	}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
	var out NodeResourceModel
	resp.Plan.Get(ctx, &out)
	if !out.ApiToken.IsUnknown() {
		t.Fatalf("expected api_token Unknown, got %v", out.ApiToken)
	}
	if out.PinnedCertSha256.IsUnknown() {
		t.Fatal("pinned_cert_sha256 must NOT be Unknown when its version is unchanged")
	}
}

func TestNodeResource_ModifyPlan_NoTrigger(t *testing.T) {
	r := &NodeResource{}
	ctx := context.Background()
	// No version change at all — ModifyPlan must be a no-op.
	plan, st := modifyPlanFixture(t, r, 1, 1)

	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := &resource.ModifyPlanResponse{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)},
	}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}
	var out NodeResourceModel
	resp.Plan.Get(ctx, &out)
	if out.ApiToken.IsUnknown() {
		t.Fatal("api_token must not be Unknown when no version changed")
	}
	if out.PinnedCertSha256.IsUnknown() {
		t.Fatal("pinned_cert_sha256 must not be Unknown when no version changed")
	}
}

// TestResolveNodeSecretsWOUpdate_PinnedCert rounds out the symmetric coverage
// for the second write-only secret on the update path (api_token is covered by
// the _TriggerBumped/_FirstUse/_NoTrigger tests above).
func TestResolveNodeSecretsWOUpdate_PinnedCert(t *testing.T) {
	plan := &NodeResourceModel{
		PinnedCertSha256WOVersion: types.Int64Value(3),
	}
	state := NodeResourceModel{PinnedCertSha256WOVersion: types.Int64Value(2)}
	config := NodeResourceModel{PinnedCertSha256WO: types.StringValue("rotated-pin")}
	resolveNodeSecretsWOUpdate(plan, state, config)
	if plan.PinnedCertSha256.ValueString() != "rotated-pin" {
		t.Fatalf("expected pinned_cert_sha256 rotated, got %q", plan.PinnedCertSha256.ValueString())
	}
}
