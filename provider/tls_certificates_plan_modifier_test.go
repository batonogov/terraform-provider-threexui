package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTLSCertificatesNullOnCreate(t *testing.T) {
	ctx := context.Background()
	elementType := types.ObjectType{AttrTypes: map[string]attr.Type{"certificate_file": types.StringType}}
	null := types.ListNull(elementType)
	unknown := types.ListUnknown(elementType)
	known := types.ListValueMust(elementType, []attr.Value{types.ObjectValueMust(elementType.AttrTypes, map[string]attr.Value{"certificate_file": types.StringValue("/cert.pem")})})
	empty := types.ListValueMust(elementType, []attr.Value{})
	for _, tc := range []struct {
		name                      string
		config, state, plan, want types.List
	}{
		{"omitted on create", null, null, unknown, null},
		{"omitted without prior certificates", null, null, null, null},
		{"stored certificates", null, known, known, known},
		{"unknown with prior state", null, known, unknown, unknown},
		{"configured unknown", unknown, null, unknown, unknown},
		{"configured certificates", known, null, known, known},
		{"explicit clear", empty, known, empty, empty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.ListRequest{ConfigValue: tc.config, StateValue: tc.state, PlanValue: tc.plan}
			resp := planmodifier.ListResponse{PlanValue: tc.plan}
			m := tlsCertificatesNullOnCreate{}
			m.PlanModifyList(ctx, req, &resp)
			if resp.Diagnostics.HasError() || !resp.PlanValue.Equal(tc.want) {
				t.Fatalf("want %s, got %s (%v)", tc.want, resp.PlanValue, resp.Diagnostics)
			}
			if m.Description(ctx) == "" || m.MarkdownDescription(ctx) == "" {
				t.Fatal("missing modifier documentation")
			}
		})
	}
}
