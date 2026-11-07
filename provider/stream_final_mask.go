package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// finalMaskSalamanderMinPassword is xray-core's smPSKMinLen, which 3x-ui
// v3.10.0+ enforces on save (internal/util/maskcompat/salamander.go). Older
// panels store a shorter password and the core then refuses to open the socket.
const finalMaskSalamanderMinPassword = 4

// streamFinalMaskAttribute is stream_settings.final_mask: xray's finalmask
// object kept as JSON text, because its tcp/udp mask lists and quicParams grow
// new mask types with every xray-core release.
func streamFinalMaskAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Optional:   true,
		Sensitive:  true,
		CustomType: jsonObjectType{},
		Description: "xray `streamSettings.finalmask` as a JSON object (`tcp`/`udp` mask lists, `quicParams`), " +
			"e.g. `jsonencode({ udp = [{ type = \"salamander\", settings = { password = var.obfs } }] })`. " +
			"Compared as JSON, so formatting and key order do not cause a diff. Sensitive because masks carry passwords. " +
			"Removing it removes `finalmask` from the stream settings.",
		Validators: []validator.String{finalMaskValidator{}},
	}
}

// canonicalJSON re-encodes a JSON document the way Terraform's jsonencode
// does (sorted keys, no whitespace), keeping numbers verbatim, so an imported
// value matches a jsonencode configuration byte for byte.
func canonicalJSON(raw []byte) (string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return "", err
	}
	if _, err := dec.Token(); err != io.EOF {
		return "", fmt.Errorf("unexpected data after the JSON value")
	}
	out, err := json.Marshal(v)
	return string(out), err
}

// ---------------------------------------------------------------------------
// Custom type: JSON text compared by value
// ---------------------------------------------------------------------------

var (
	_ basetypes.StringTypable                    = jsonObjectType{}
	_ basetypes.StringValuableWithSemanticEquals = jsonObjectValue{}
)

type jsonObjectType struct {
	basetypes.StringType
}

func (t jsonObjectType) Equal(o attr.Type) bool {
	_, ok := o.(jsonObjectType)
	return ok
}

func (t jsonObjectType) String() string {
	return "jsonObjectType"
}

func (t jsonObjectType) ValueType(context.Context) attr.Value {
	return jsonObjectValue{}
}

func (t jsonObjectType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return jsonObjectValue{StringValue: in}, nil
}

func (t jsonObjectType) ValueFromTerraform(_ context.Context, in tftypes.Value) (attr.Value, error) {
	if !in.IsKnown() {
		return jsonObjectValue{StringValue: types.StringUnknown()}, nil
	}
	if in.IsNull() {
		return jsonObjectNull(), nil
	}
	var s string
	if err := in.As(&s); err != nil {
		return nil, fmt.Errorf("expected a string value: %w", err)
	}
	return newJSONObjectValue(s), nil
}

// jsonObjectValue lets the framework keep the configured spelling when the
// panel returns the same document in another layout: semantic equality runs
// after Create, Update and Read. With CustomType declared, every model field
// for the attribute must hold this type, not a plain types.String.
type jsonObjectValue struct {
	basetypes.StringValue
}

func newJSONObjectValue(s string) jsonObjectValue {
	return jsonObjectValue{StringValue: types.StringValue(s)}
}

func jsonObjectNull() jsonObjectValue {
	return jsonObjectValue{StringValue: types.StringNull()}
}

func (v jsonObjectValue) Type(context.Context) attr.Type {
	return jsonObjectType{}
}

func (v jsonObjectValue) Equal(o attr.Value) bool {
	other, ok := o.(jsonObjectValue)
	return ok && v.StringValue.Equal(other.StringValue)
}

func (v jsonObjectValue) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	newValue, d := newValuable.ToStringValue(ctx)
	diags.Append(d...)
	if diags.HasError() || v.IsNull() || v.IsUnknown() || newValue.IsNull() || newValue.IsUnknown() {
		return false, diags
	}
	var oldDoc, newDoc any
	if json.Unmarshal([]byte(v.ValueString()), &oldDoc) != nil ||
		json.Unmarshal([]byte(newValue.ValueString()), &newDoc) != nil {
		return false, diags
	}
	return reflect.DeepEqual(oldDoc, newDoc), diags
}

// ---------------------------------------------------------------------------
// Validator
// ---------------------------------------------------------------------------

type finalMaskValidator struct{}

func (finalMaskValidator) Description(context.Context) string {
	return fmt.Sprintf("must be a JSON object; salamander passwords must be at least %d bytes", finalMaskSalamanderMinPassword)
}

func (v finalMaskValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (finalMaskValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(req.ConfigValue.ValueString()), &doc); err != nil || doc == nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid finalmask",
			"final_mask must be a JSON object such as jsonencode({ udp = [...] }).")
		return
	}
	// Mirrors maskcompat.CheckSalamanderPasswords: the type match is
	// case-insensitive there, and a missing password counts as empty.
	masks, _ := doc["udp"].([]any)
	for i, entry := range masks {
		mask, _ := entry.(map[string]any)
		if maskType, _ := mask["type"].(string); !strings.EqualFold(maskType, "salamander") {
			continue
		}
		settings, _ := mask["settings"].(map[string]any)
		if password, _ := settings["password"].(string); len(password) < finalMaskSalamanderMinPassword {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid finalmask",
				fmt.Sprintf("udp[%d]: the salamander password must be at least %d bytes. xray-core loads a shorter one "+
					"and then refuses to open the socket, which takes the listener (or every dial) down; "+
					"3x-ui v3.10.0+ rejects it on save.", i, finalMaskSalamanderMinPassword))
		}
	}
}
