package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// fakeClientPanel models the v3.9.0+ routes the inbound_client resource uses
// for the limit_hwid / external_link lifecycle: dedicated client CRUD, the
// client record read and the replace-all external-links endpoint.
type fakeClientPanel struct {
	mu                sync.Mutex
	clients           []map[string]any
	limitHwid         int64
	externalLinks     []ClientExternalLink
	externalLinkPosts int
}

func (p *fakeClientPanel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch r.URL.Path {
	case "/login":
		w.Write(okResponse(nil))
	case "/panel/api/clients/list":
		w.Write(okResponse([]any{}))
	case "/panel/api/inbounds/get/9":
		settings, _ := json.Marshal(map[string]any{"clients": p.clients})
		w.Write(okResponse(map[string]any{"id": 9, "settings": string(settings)}))
	case "/panel/api/clients/add":
		var body struct {
			Client map[string]any `json:"client"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		p.clients = append(p.clients, body.Client)
		if v, ok := body.Client["limitHwid"].(float64); ok {
			p.limitHwid = int64(v)
		}
		w.Write(okResponse(nil))
	case "/panel/api/clients/update/a@b":
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if len(p.clients) > 0 {
			p.clients[0] = body
		}
		w.Write(okResponse(nil))
	case "/panel/api/clients/get/a@b":
		w.Write(okResponse(map[string]any{
			"client":        map[string]any{"email": "a@b", "limitHwid": p.limitHwid},
			"externalLinks": p.externalLinks,
		}))
	case "/panel/api/clients/a@b/externalLinks":
		var body struct {
			ExternalLinks []ClientExternalLink `json:"externalLinks"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		p.externalLinks = body.ExternalLinks
		p.externalLinkPosts++
		w.Write(okResponse(nil))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func inboundClientTestSchema(t *testing.T) resourceschema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	(&InboundClientResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// rawFromAttributes builds a tftypes object from a resource schema, filling
// every attribute not present in values with null.
func rawFromAttributes(t *testing.T, s resourceschema.Schema, values map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	attrTypes := map[string]tftypes.Type{}
	rawValues := map[string]tftypes.Value{}
	for name, attr := range s.Attributes {
		typ := attr.GetType().TerraformType(context.Background())
		attrTypes[name] = typ
		if v, ok := values[name]; ok {
			rawValues[name] = v
		} else {
			rawValues[name] = tftypes.NewValue(typ, nil)
		}
	}
	return tftypes.NewValue(tftypes.Object{AttributeTypes: attrTypes}, rawValues)
}

func externalLinkListValue(t *testing.T, s resourceschema.Schema, entries []map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	listType, ok := s.Attributes["external_link"].GetType().TerraformType(context.Background()).(tftypes.List)
	if !ok {
		t.Fatal("external_link is not a list attribute")
	}
	objType := listType.ElementType.(tftypes.Object)
	values := make([]tftypes.Value, 0, len(entries))
	for _, entry := range entries {
		full := map[string]tftypes.Value{}
		for name, typ := range objType.AttributeTypes {
			if v, ok := entry[name]; ok {
				full[name] = v
			} else {
				full[name] = tftypes.NewValue(typ, nil)
			}
		}
		values = append(values, tftypes.NewValue(objType, full))
	}
	return tftypes.NewValue(listType, values)
}

func TestInboundClientCreateWithExternalLinks(t *testing.T) {
	panel := &fakeClientPanel{}
	srv := httptest.NewServer(panel)
	defer srv.Close()

	s := inboundClientTestSchema(t)
	planRaw := rawFromAttributes(t, s, map[string]tftypes.Value{
		"inbound_id": tftypes.NewValue(tftypes.Number, 9),
		"email":      tftypes.NewValue(tftypes.String, "a@b"),
		"limit_hwid": tftypes.NewValue(tftypes.Number, 3),
		"external_link": externalLinkListValue(t, s, []map[string]tftypes.Value{{
			"kind":  tftypes.NewValue(tftypes.String, "subscription"),
			"value": tftypes.NewValue(tftypes.String, "https://example.com/sub"),
		}}),
	})
	plan := tfsdk.Plan{Schema: s, Raw: planRaw}

	r := &InboundClientResource{client: newTestClient(t, srv.URL)}
	resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan, Config: tfsdk.Config(plan)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %v", resp.Diagnostics)
	}

	if panel.externalLinkPosts != 1 {
		t.Fatalf("expected one replace-all external-links POST, got %d", panel.externalLinkPosts)
	}
	if len(panel.externalLinks) != 1 || panel.externalLinks[0].Kind != "subscription" {
		t.Fatalf("panel stored wrong links: %+v", panel.externalLinks)
	}

	var state InboundClientResourceModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("decoding state: %v", diags)
	}
	if state.LimitHwid.ValueInt64() != 3 {
		t.Fatalf("expected limit_hwid 3 in state, got %d", state.LimitHwid.ValueInt64())
	}
	if len(state.ExternalLink.Elements()) != 1 {
		t.Fatalf("expected 1 external link in state, got %d", len(state.ExternalLink.Elements()))
	}
}

func TestInboundClientReadMergesRecord(t *testing.T) {
	panel := &fakeClientPanel{
		clients:   []map[string]any{{"id": "uuid-1", "email": "a@b"}},
		limitHwid: 3,
		externalLinks: []ClientExternalLink{
			{Kind: "link", Value: "vless://uuid@host:443#name"},
		},
	}
	srv := httptest.NewServer(panel)
	defer srv.Close()

	s := inboundClientTestSchema(t)
	stateRaw := rawFromAttributes(t, s, map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, "9:uuid-1"),
		"inbound_id": tftypes.NewValue(tftypes.Number, 9),
		"client_id":  tftypes.NewValue(tftypes.String, "uuid-1"),
		"email":      tftypes.NewValue(tftypes.String, "a@b"),
	})

	r := &InboundClientResource{client: newTestClient(t, srv.URL)}
	resp := resource.ReadResponse{State: tfsdk.State{Schema: s}}
	r.Read(context.Background(), resource.ReadRequest{State: tfsdk.State{Schema: s, Raw: stateRaw}}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %v", resp.Diagnostics)
	}

	var state InboundClientResourceModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("decoding state: %v", diags)
	}
	if state.LimitHwid.ValueInt64() != 3 {
		t.Fatalf("expected limit_hwid 3 after refresh, got %d", state.LimitHwid.ValueInt64())
	}
	if len(state.ExternalLink.Elements()) != 1 {
		t.Fatalf("expected 1 external link after refresh, got %d", len(state.ExternalLink.Elements()))
	}
}

func TestInboundClientUpdateClearsExternalLinks(t *testing.T) {
	enable := true
	panel := &fakeClientPanel{
		clients:   []map[string]any{{"id": "uuid-1", "email": "a@b"}},
		limitHwid: 3,
		externalLinks: []ClientExternalLink{
			{Kind: "link", Value: "vless://uuid@host:443#name", Enable: &enable},
		},
	}
	srv := httptest.NewServer(panel)
	defer srv.Close()

	s := inboundClientTestSchema(t)
	base := map[string]tftypes.Value{
		"id":         tftypes.NewValue(tftypes.String, "9:uuid-1"),
		"inbound_id": tftypes.NewValue(tftypes.Number, 9),
		"client_id":  tftypes.NewValue(tftypes.String, "uuid-1"),
		"email":      tftypes.NewValue(tftypes.String, "a@b"),
		"limit_hwid": tftypes.NewValue(tftypes.Number, 3),
	}
	prior := map[string]tftypes.Value{}
	for k, v := range base {
		prior[k] = v
	}
	prior["external_link"] = externalLinkListValue(t, s, []map[string]tftypes.Value{{
		"value": tftypes.NewValue(tftypes.String, "vless://uuid@host:443#name"),
	}})

	// Plan clears the list: an explicit empty list must reach the panel.
	planned := map[string]tftypes.Value{}
	for k, v := range base {
		planned[k] = v
	}
	planned["external_link"] = externalLinkListValue(t, s, nil)
	plan := tfsdk.Plan{Schema: s, Raw: rawFromAttributes(t, s, planned)}

	r := &InboundClientResource{client: newTestClient(t, srv.URL)}
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: s, Raw: rawFromAttributes(t, s, prior)}}
	r.Update(context.Background(), resource.UpdateRequest{
		Plan:   plan,
		State:  tfsdk.State{Schema: s, Raw: rawFromAttributes(t, s, prior)},
		Config: tfsdk.Config(plan),
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %v", resp.Diagnostics)
	}

	if panel.externalLinkPosts != 1 {
		t.Fatalf("expected one external-links POST, got %d", panel.externalLinkPosts)
	}
	if len(panel.externalLinks) != 0 {
		t.Fatalf("clear did not reach the panel: %+v", panel.externalLinks)
	}

	var state InboundClientResourceModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("decoding state: %v", diags)
	}
	if len(state.ExternalLink.Elements()) != 0 {
		t.Fatalf("expected cleared links in state, got %d", len(state.ExternalLink.Elements()))
	}
}
