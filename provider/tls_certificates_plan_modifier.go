package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// tlsCertificatesNullOnCreate runs after UseStateForUnknown. The TLS certificate
// model uses a Go slice, which cannot decode a whole unknown list during Apply.
// An omitted list with no prior value means no certificates; imported/stored
// certificates are preserved by UseStateForUnknown before this modifier runs.
type tlsCertificatesNullOnCreate struct{}

func (tlsCertificatesNullOnCreate) Description(context.Context) string {
	return "Plans an omitted TLS certificate list as null when no prior certificates exist."
}

func (m tlsCertificatesNullOnCreate) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (tlsCertificatesNullOnCreate) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.ConfigValue.IsNull() && req.StateValue.IsNull() && req.PlanValue.IsUnknown() {
		resp.PlanValue = types.ListNull(req.PlanValue.ElementType(ctx))
	}
}
