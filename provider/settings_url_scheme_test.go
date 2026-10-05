package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestEnsureURLSchemeValue pins the provider-side mirror of the panel's
// common.EnsureURLScheme (3x-ui-3.9.0/internal/util/common/url.go:14-25),
// which validateSettingsURLs applies on save to the subscription URL fields.
func TestEnsureURLSchemeValue(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		in, want string
	}{
		{"", ""},
		{"   ", ""},
		{"example.com", "https://example.com"},
		{" example.com/path ", "https://example.com/path"},
		{"https://example.com", "https://example.com"},
		{" https://example.com ", "https://example.com"},
		{"http://example.com", "http://example.com"},
		{"tg://resolve?domain=x", "tg://resolve?domain=x"},
		{"mailto:support@example.com", "mailto:support@example.com"},
		{"tel:+123456", "tel:+123456"},
	} {
		if got := ensureURLSchemeValue(tt.in); got != tt.want {
			t.Errorf("ensureURLSchemeValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestEnsureURLSchemePlanModifier: the modifier rewrites only known,
// scheme-less non-empty plan values; null/unknown and already-schemed values
// pass through untouched so UseStateForUnknown and full URLs are unaffected.
func TestEnsureURLSchemePlanModifier(t *testing.T) {
	t.Parallel()
	mod := ensureURLScheme()
	for _, tt := range []struct {
		name string
		in   types.String
		want types.String
	}{
		{"null passes through", types.StringNull(), types.StringNull()},
		{"unknown passes through", types.StringUnknown(), types.StringUnknown()},
		{"scheme-less is prefixed", types.StringValue("example.com/announce"), types.StringValue("https://example.com/announce")},
		{"https is unchanged", types.StringValue("https://example.com"), types.StringValue("https://example.com")},
		{"mailto is unchanged", types.StringValue("mailto:a@b.c"), types.StringValue("mailto:a@b.c")},
		{"empty is unchanged", types.StringValue(""), types.StringValue("")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := planmodifier.StringRequest{PlanValue: tt.in}
			resp := &planmodifier.StringResponse{PlanValue: tt.in}
			mod.PlanModifyString(context.Background(), req, resp)
			if !resp.PlanValue.Equal(tt.want) {
				t.Fatalf("PlanValue = %#v, want %#v", resp.PlanValue, tt.want)
			}
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
		})
	}
}

// TestPanelSubscriptionSchemaURLSchemeModifier asserts that every subscription
// URL attribute the panel rewrites via common.EnsureURLScheme on save
// (3x-ui-3.9.0/internal/web/service/setting.go validateSettingsURLs) carries
// the matching plan modifier — a missing one resurfaces the
// "inconsistent result after apply" / perpetual-diff failure for scheme-less
// config values.
func TestPanelSubscriptionSchemaURLSchemeModifier(t *testing.T) {
	t.Parallel()
	attrs := []string{
		// Rewritten since v3.7.x (support/profile) and v3.8.0 (Happ links).
		"sub_support_url", "sub_profile_url",
		"sub_happ_new_url", "sub_happ_fallback_url",
		"sub_happ_sub_info_button_link", "sub_happ_sub_expire_button_link",
		// v3.9.0 Incy URLs (subIncyResolveDnsDomain is treated as a URL upstream).
		"sub_incy_announce_url", "sub_incy_premium_url",
		"sub_incy_banner_button_url", "sub_incy_resolve_dns_domain",
	}
	s := panelSubscriptionSchema()
	for _, name := range attrs {
		attr, ok := s.Attributes[name].(schema.StringAttribute)
		if !ok {
			t.Errorf("%s must be a StringAttribute", name)
			continue
		}
		found := false
		for _, pm := range attr.PlanModifiers {
			if _, ok := pm.(ensureURLSchemeModifier); ok {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s must carry the ensureURLScheme plan modifier", name)
		}
	}
}

// TestPanelSubscriptionSchemaV39Attributes asserts the v3.9.0 attributes exist
// with the expected types and Optional+Computed shape.
func TestPanelSubscriptionSchemaV39Attributes(t *testing.T) {
	t.Parallel()
	s := panelSubscriptionSchema()

	for _, name := range []string{
		"sub_happ_local_proxy_auth", "external_sub_user_agent",
		"sub_incy_profile_description", "sub_incy_sort_order", "sub_incy_support_email",
		"sub_incy_announce_url", "sub_incy_premium_url",
		"sub_incy_banner_text", "sub_incy_banner_button_text", "sub_incy_banner_button_url",
		"sub_incy_banner_bg_color", "sub_incy_banner_button_color",
		"sub_incy_hide_url", "sub_incy_hide_check",
		"sub_incy_no_limit_enabled", "sub_incy_per_app_enable", "sub_incy_per_app_mode", "sub_incy_per_app_list",
		"sub_incy_fragmentation_enable", "sub_incy_fragment_length", "sub_incy_fragment_interval", "sub_incy_fragment_packets",
		"sub_incy_noises_enable", "sub_incy_noises_type", "sub_incy_noises_packet", "sub_incy_noises_delay",
		"sub_incy_resolve_enable", "sub_incy_resolve_dns_domain", "sub_incy_resolve_dns_ip",
	} {
		attr, ok := s.Attributes[name].(schema.StringAttribute)
		if !ok {
			t.Errorf("%s must be a StringAttribute", name)
			continue
		}
		if !attr.IsOptional() || !attr.IsComputed() {
			t.Errorf("%s must be Optional+Computed", name)
		}
	}

	attr, ok := s.Attributes["sub_incy_app_auto_detect"].(schema.BoolAttribute)
	if !ok {
		t.Fatal("sub_incy_app_auto_detect must be a BoolAttribute")
	}
	if !attr.IsOptional() || !attr.IsComputed() {
		t.Fatal("sub_incy_app_auto_detect must be Optional+Computed")
	}
}
