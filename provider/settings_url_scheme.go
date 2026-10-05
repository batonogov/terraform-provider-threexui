package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// subSchemeNormalizedNote documents the panel-side rewrite shared by every
// subscription URL attribute (see ensureURLScheme below).
const subSchemeNormalizedNote = "A scheme-less non-empty value is normalized to `https://…` " +
	"(the panel rewrites it the same way on save via common.EnsureURLScheme), so write the full URL or accept the normalization; " +
	"values with an explicit scheme (`https://`, `tg://`, `mailto:`, `tel:`) and empty strings pass through untouched."

// subIncyDescription builds the shared description for the Incy client
// customization attributes (3x-ui v3.9.0+). An empty value is meaningful: it
// omits the header so the subscriber's own app setting is left alone
// (3x-ui-3.9.0/internal/web/entity/entity.go comment on the subIncy* block).
func subIncyDescription(field string) string {
	return "Incy: " + field + ". An empty string omits the header, leaving the subscriber's own app setting untouched. " +
		"Requires 3x-ui v3.9.0+; older panels ignore it and read it back empty."
}

// ensureURLScheme mirrors the panel's common.EnsureURLScheme
// (3x-ui-3.9.0/internal/util/common/url.go:14-25): a scheme-less non-empty
// value is prefixed with https:// so subscription apps don't resolve it
// against the panel's own domain; values with an explicit scheme and empty
// strings pass through. validateSettingsURLs
// (3x-ui-3.9.0/internal/web/service/setting.go:1789) applies this rewrite on
// save to subSupportUrl, subProfileUrl, the four subHapp* links and the four
// subIncy* URLs, so the provider must plan the same value the panel will
// store or every apply either fails as "inconsistent result" (the read-back
// differs from the plan) or diffs forever.
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

// ensureURLScheme returns a plan modifier that normalizes a configured
// subscription URL the same way the panel does on save, so a scheme-less
// config value (example.com) plans, applies and reads back as
// https://example.com instead of producing a perpetual diff.
func ensureURLScheme() planmodifier.String {
	return ensureURLSchemeModifier{}
}

type ensureURLSchemeModifier struct{}

func (m ensureURLSchemeModifier) Description(_ context.Context) string {
	return "Normalizes a scheme-less URL to https://, matching the panel's common.EnsureURLScheme rewrite on save."
}

func (m ensureURLSchemeModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m ensureURLSchemeModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if normalized := ensureURLSchemeValue(req.PlanValue.ValueString()); normalized != req.PlanValue.ValueString() {
		resp.PlanValue = types.StringValue(normalized)
	}
}
