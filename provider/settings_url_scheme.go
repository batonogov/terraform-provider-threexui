package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// subSchemeNormalizedNote documents the panel-side rewrite shared by every
// subscription URL attribute (see urlSchemeNormalizedType below).
const subSchemeNormalizedNote = "A scheme-less non-empty value is stored by the panel with an `https://` prefix " +
	"(common.EnsureURLScheme); the provider treats the two spellings as semantically equal, so writing either plans clean. " +
	"Values with an explicit scheme (`http://`, `https://`, `tg://`, `mailto:`, `tel:`) and empty strings pass through untouched."

// subIncyDescription builds the shared description for the Incy client
// customization attributes (3x-ui v3.9.0+). An empty value is meaningful: it
// omits the header so the subscriber's own app setting is left alone
// (3x-ui-3.9.0/internal/web/entity/entity.go comment on the subIncy* block).
func subIncyDescription(field string) string {
	return "Incy: " + field + ". An empty string omits the header, leaving the subscriber's own app setting untouched. " +
		"Requires 3x-ui v3.9.0+; older panels ignore it and read it back empty."
}

// ensureURLSchemeValue mirrors the panel's common.EnsureURLScheme
// (3x-ui-3.9.0/internal/util/common/url.go:14-25): a scheme-less non-empty
// value is prefixed with https:// so subscription apps don't resolve it
// against the panel's own domain; any value containing "://" — including
// http://, which is NOT upgraded to https — plus mailto:/tel: and empty
// strings pass through (trimmed). validateSettingsURLs
// (3x-ui-3.9.0/internal/web/service/setting.go:1789) applies this rewrite on
// save to subSupportUrl, subProfileUrl, the four subHapp* links and the four
// subIncy* URLs.
func ensureURLSchemeValue(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "://") ||
		strings.HasPrefix(trimmed, "mailto:") ||
		strings.HasPrefix(trimmed, "tel:") {
		return trimmed
	}
	return "https://" + trimmed
}

// urlSchemeNormalizedType is the custom string type for the subscription URL
// attributes the panel rewrites on save via common.EnsureURLScheme. A plan
// modifier cannot express this normalization — Terraform core rejects a plan
// whose value differs from an explicitly configured one ("planned value does
// not match config value") — so the normalization is modeled as semantic
// equality instead: a prior-state value of https://example.com/announce and a
// configured example.com/announce plan as no change, and the plan keeps the
// stored (prefixed) value. As a side effect that also keeps
// panelSettingsNeedRestart quiet on unrelated applies: the desired map it
// compares against the panel carries the same normalized spelling the panel
// reports.
type urlSchemeNormalizedType struct {
	basetypes.StringType
}

func (t urlSchemeNormalizedType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return urlSchemeNormalizedValue{StringValue: in}, nil
}

func (t urlSchemeNormalizedType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	v, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	sv, ok := v.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("expected basetypes.StringValue, got %T", v)
	}
	return urlSchemeNormalizedValue{StringValue: sv}, nil
}

// urlSchemeNormalizedValue carries the EnsureURLScheme-aware equality. The
// framework calls StringSemanticEquals on the PRIOR-STATE value with the
// planned (config) value; returning true keeps the state value in the plan.
// Because the schema attributes declare CustomType, every model field and
// flatten assignment for these attributes must use this exact type — a plain
// basetypes.StringValue fails State.Set with "Value Conversion Error".
type urlSchemeNormalizedValue struct {
	basetypes.StringValue
}

// newURLSchemeNormalizedValue wraps a known string in the custom value type.
func newURLSchemeNormalizedValue(s string) urlSchemeNormalizedValue {
	return urlSchemeNormalizedValue{StringValue: types.StringValue(s)}
}

func (v urlSchemeNormalizedValue) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	if v.IsNull() || v.IsUnknown() {
		return false, diags
	}

	newVal, ok := newValuable.(urlSchemeNormalizedValue)
	if !ok {
		plain, ok := newValuable.(basetypes.StringValue)
		if !ok {
			return false, diags
		}
		newVal = urlSchemeNormalizedValue{StringValue: plain}
	}
	if newVal.IsNull() || newVal.IsUnknown() {
		return false, diags
	}

	// The panel stores EnsureURLScheme(config), so the stored value is
	// semantically equal to the config exactly when normalizing the config
	// reproduces it. Because EnsureURLScheme leaves schemed values untouched
	// (http:// is not upgraded), config http://x never equals state https://x.
	return ensureURLSchemeValue(newVal.ValueString()) == v.ValueString(), diags
}
