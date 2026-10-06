package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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
		// http:// contains "://" and passes through untouched — the panel does
		// NOT upgrade http to https.
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

// foreignStringValuable implements basetypes.StringValuable via embedding but
// is neither urlSchemeNormalizedValue nor basetypes.StringValue — it exercises
// the StringSemanticEquals fallback for value implementations the framework
// itself never produces.
type foreignStringValuable struct {
	basetypes.StringValue
}

// TestURLSchemeNormalizedSemanticEquals: the prior-state value answers whether
// a planned (config) value is the same URL modulo the panel's
// common.EnsureURLScheme rewrite. This replaces a plan modifier — Terraform
// core rejects a plan modifier that changes an explicitly configured value
// ("planned value does not match config value", TestAccPanelSubscriptionV39).
func TestURLSchemeNormalizedSemanticEquals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mkVal := func(s string) urlSchemeNormalizedValue {
		return urlSchemeNormalizedValue{StringValue: types.StringValue(s)}
	}

	for _, tt := range []struct {
		name  string
		state urlSchemeNormalizedValue
		new   basetypes.StringValuable
		want  bool
	}{
		{
			name:  "scheme-less config equals https-prefixed state",
			state: mkVal("https://example.com/announce"),
			new:   mkVal("example.com/announce"),
			want:  true,
		},
		{
			name:  "identical prefixed values",
			state: mkVal("https://example.com"),
			new:   mkVal("https://example.com"),
			want:  true,
		},
		{
			name:  "config with surrounding whitespace equals state (panel trims)",
			state: mkVal("https://example.com"),
			new:   mkVal(" example.com "),
			want:  true,
		},
		{
			name:  "http config must NOT equal https state (no http→https upgrade upstream)",
			state: mkVal("https://example.com"),
			new:   mkVal("http://example.com"),
			want:  false,
		},
		{
			name:  "different path is a real change",
			state: mkVal("https://example.com/announce"),
			new:   mkVal("example.com/announce2"),
			want:  false,
		},
		{
			name:  "different host is a real change",
			state: mkVal("https://example.com"),
			new:   mkVal("other.com"),
			want:  false,
		},
		{
			name:  "non-http scheme passes through on both sides",
			state: mkVal("mailto:a@b.c"),
			new:   mkVal("mailto:a@b.c"),
			want:  true,
		},
		{
			name:  "empty on both sides",
			state: mkVal(""),
			new:   mkVal(""),
			want:  true,
		},
		{
			name:  "empty config vs prefixed state is a real change",
			state: mkVal("https://example.com"),
			new:   mkVal(""),
			want:  false,
		},
		{
			name:  "plain StringValue argument is tolerated",
			state: mkVal("https://example.com"),
			new:   basetypes.NewStringValue("example.com"),
			want:  true,
		},
		{
			name:  "an unrelated StringValuable implementation is never semantically equal",
			state: mkVal("https://example.com"),
			new:   foreignStringValuable{StringValue: basetypes.NewStringValue("example.com")},
			want:  false,
		},
		{
			name:  "null new value is never semantically equal",
			state: mkVal("https://example.com"),
			new:   urlSchemeNormalizedValue{StringValue: types.StringNull()},
			want:  false,
		},
		{
			name:  "unknown new value is never semantically equal",
			state: mkVal("https://example.com"),
			new:   urlSchemeNormalizedValue{StringValue: types.StringUnknown()},
			want:  false,
		},
		{
			name:  "null state is never semantically equal",
			state: urlSchemeNormalizedValue{StringValue: types.StringNull()},
			new:   mkVal("example.com"),
			want:  false,
		},
		{
			name:  "unknown state is never semantically equal",
			state: urlSchemeNormalizedValue{StringValue: types.StringUnknown()},
			new:   mkVal("example.com"),
			want:  false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, diags := tt.state.StringSemanticEquals(ctx, tt.new)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if got != tt.want {
				t.Fatalf("StringSemanticEquals(%#v, %#v) = %v, want %v", tt.state, tt.new, got, tt.want)
			}
		})
	}
}

// TestURLSchemeNormalizedTypeConversions covers the custom type's value
// constructors (the framework decodes config/state through them; without
// ValueFromTerraform the prior state would decode to a plain StringValue and
// semantic equality would never fire).
func TestURLSchemeNormalizedTypeConversions(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	typ := urlSchemeNormalizedType{}

	v, diags := typ.ValueFromString(ctx, types.StringValue("example.com"))
	if diags.HasError() {
		t.Fatalf("ValueFromString diagnostics: %v", diags)
	}
	sv, ok := v.(urlSchemeNormalizedValue)
	if !ok {
		t.Fatalf("ValueFromString returned %T, want urlSchemeNormalizedValue", v)
	}
	if sv.ValueString() != "example.com" {
		t.Fatalf("ValueFromString value = %q", sv.ValueString())
	}

	tv, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(tftypes.String, "https://example.com"))
	if err != nil {
		t.Fatalf("ValueFromTerraform: %v", err)
	}
	tvStr, ok := tv.(urlSchemeNormalizedValue)
	if !ok {
		t.Fatalf("ValueFromTerraform returned %T, want urlSchemeNormalizedValue", tv)
	}
	if tvStr.ValueString() != "https://example.com" {
		t.Fatalf("ValueFromTerraform value = %q", tvStr.ValueString())
	}

	for name, raw := range map[string]tftypes.Value{
		"unknown": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"null":    tftypes.NewValue(tftypes.String, nil),
	} {
		tv, err := typ.ValueFromTerraform(ctx, raw)
		if err != nil {
			t.Fatalf("ValueFromTerraform(%s): %v", name, err)
		}
		if _, ok := tv.(urlSchemeNormalizedValue); !ok {
			t.Fatalf("ValueFromTerraform(%s) returned %T, want urlSchemeNormalizedValue", name, tv)
		}
	}

	// A non-string Terraform value is a decode error.
	if _, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(tftypes.Bool, true)); err == nil {
		t.Fatal("ValueFromTerraform with a bool value must return an error")
	}
}

// TestPanelSubscriptionSchemaURLSchemeCustomType asserts that every
// subscription URL attribute the panel rewrites via common.EnsureURLScheme on
// save (3x-ui-3.9.0/internal/web/service/setting.go validateSettingsURLs)
// carries the semantic-equality custom type — a missing one resurfaces the
// "inconsistent result after apply" / perpetual-diff failure for scheme-less
// config values.
func TestPanelSubscriptionSchemaURLSchemeCustomType(t *testing.T) {
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
		if _, ok := attr.CustomType.(urlSchemeNormalizedType); !ok {
			t.Errorf("%s must carry the urlSchemeNormalizedType custom type, got %T", name, attr.CustomType)
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

// TestPanelSubscriptionCustomTypeStateRoundTrip is the regression guard for the
// live-panel failure "Cannot use attr.Value basetypes.StringValue, only
// provider.urlSchemeNormalizedValue is supported": with CustomType on the
// schema attribute, the flatten output AND the model decode must use the
// custom value type end to end. It drives the real framework conversion
// machinery (tfsdk.State.Set → Get) with the flatten-built model, so a
// regression of any of the 10 fields back to basetypes.StringValue fails here
// instead of in CI.
func TestPanelSubscriptionCustomTypeStateRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	model := flattenPanelSubscription(map[string]any{
		"subSupportUrl":              "https://example.com/support",
		"subProfileUrl":              "https://example.com/profile",
		"subHappNewUrl":              "https://example.com/new",
		"subHappFallbackUrl":         "https://example.com/fallback",
		"subHappSubInfoButtonLink":   "https://example.com/info",
		"subHappSubExpireButtonLink": "https://example.com/expire",
		"subIncyAnnounceUrl":         "https://example.com/announce",
		"subIncyPremiumUrl":          "https://example.com/premium",
		"subIncyBannerButtonUrl":     "https://example.com/buy",
		"subIncyResolveDnsDomain":    "https://dns.example.com",
	})

	// flatten must produce the custom value type for every CustomType attr.
	for name, v := range map[string]attr.Value{
		"sub_support_url":                 model.SubSupportURL,
		"sub_profile_url":                 model.SubProfileURL,
		"sub_happ_new_url":                model.SubHappNewUrl,
		"sub_happ_fallback_url":           model.SubHappFallbackUrl,
		"sub_happ_sub_info_button_link":   model.SubHappSubInfoButtonLink,
		"sub_happ_sub_expire_button_link": model.SubHappSubExpireButtonLink,
		"sub_incy_announce_url":           model.SubIncyAnnounceUrl,
		"sub_incy_premium_url":            model.SubIncyPremiumUrl,
		"sub_incy_banner_button_url":      model.SubIncyBannerButtonUrl,
		"sub_incy_resolve_dns_domain":     model.SubIncyResolveDnsDomain,
	} {
		if _, ok := v.(urlSchemeNormalizedValue); !ok {
			t.Errorf("flatten %s = %T, want urlSchemeNormalizedValue", name, v)
		}
	}

	sch := panelSubscriptionSchema()
	st := &tfsdk.State{
		Schema: sch,
		Raw:    tftypes.NewValue(sch.Type().TerraformType(ctx), nil),
	}
	if diags := st.Set(ctx, model); diags.HasError() {
		t.Fatalf("State.Set with the flatten-built model failed (the CI bug): %v", diags)
	}

	var back PanelSubscriptionModel
	if diags := st.Get(ctx, &back); diags.HasError() {
		t.Fatalf("State.Get into the model failed: %v", diags)
	}
	if got := back.SubIncyAnnounceUrl.ValueString(); got != "https://example.com/announce" {
		t.Fatalf("round-trip sub_incy_announce_url = %q", got)
	}
	if got := back.SubSupportURL.ValueString(); got != "https://example.com/support" {
		t.Fatalf("round-trip sub_support_url = %q", got)
	}
	// Keys absent from the panel payload stay null through the round-trip.
	if back.SubIncyPremiumUrl.ValueString() != "https://example.com/premium" {
		t.Fatalf("round-trip sub_incy_premium_url = %#v", back.SubIncyPremiumUrl)
	}
	if !back.SubIncyBannerText.IsNull() {
		t.Fatalf("absent key sub_incy_banner_text must stay null, got %#v", back.SubIncyBannerText)
	}
	if !back.SubHappLocalProxyAuth.IsNull() {
		t.Fatalf("absent key sub_happ_local_proxy_auth must stay null, got %#v", back.SubHappLocalProxyAuth)
	}
}
