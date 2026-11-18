package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func newProviderConfigureRequest(t *testing.T, values map[string]tftypes.Value) fwprovider.ConfigureRequest {
	t.Helper()
	p := &ThreeXUIProvider{}
	var schemaResp fwprovider.SchemaResponse
	p.Schema(context.Background(), fwprovider.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("provider schema diagnostics: %v", schemaResp.Diagnostics)
	}
	s := schemaResp.Schema

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
	raw := tftypes.NewValue(tftypes.Object{AttributeTypes: attrTypes}, rawValues)
	return fwprovider.ConfigureRequest{Config: tfsdk.Config{Schema: s, Raw: raw}}
}

func TestProviderConfigureWithAPIToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/panel/api/server/status" {
			gotAuth = r.Header.Get("Authorization")
			w.Write(okResponse(nil))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	req := newProviderConfigureRequest(t, map[string]tftypes.Value{
		"endpoint":  tftypes.NewValue(tftypes.String, srv.URL),
		"api_token": tftypes.NewValue(tftypes.String, "tok"),
	})
	var resp fwprovider.ConfigureResponse
	(&ThreeXUIProvider{}).Configure(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %v", resp.Diagnostics)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("probe did not carry the token: %q", gotAuth)
	}
	client, ok := resp.ResourceData.(*Client)
	if !ok || client.apiToken != "tok" {
		t.Fatalf("configured client lost the token: %+v", resp.ResourceData)
	}
}

func TestProviderConfigureWithAPITokenFromEnv(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write(okResponse(nil))
	}))
	defer srv.Close()

	t.Setenv("THREEXUI_API_TOKEN", "env-tok")
	req := newProviderConfigureRequest(t, map[string]tftypes.Value{
		"endpoint": tftypes.NewValue(tftypes.String, srv.URL),
	})
	var resp fwprovider.ConfigureResponse
	(&ThreeXUIProvider{}).Configure(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %v", resp.Diagnostics)
	}
	if gotAuth != "Bearer env-tok" {
		t.Fatalf("env token not used: %q", gotAuth)
	}
}

func TestProviderConfigureWithAPITokenRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	req := newProviderConfigureRequest(t, map[string]tftypes.Value{
		"endpoint":  tftypes.NewValue(tftypes.String, srv.URL),
		"api_token": tftypes.NewValue(tftypes.String, "bad"),
	})
	var resp fwprovider.ConfigureResponse
	(&ThreeXUIProvider{}).Configure(context.Background(), req, &resp)
	if !diagsContain(resp.Diagnostics, "API token authentication failed") {
		t.Fatalf("expected token failure diagnostic, got %v", resp.Diagnostics)
	}
}
