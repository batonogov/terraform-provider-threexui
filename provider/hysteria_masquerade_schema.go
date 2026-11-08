package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// InboundHysteriaMasqueradeModel mirrors xray-core's Masquerade
// (infra/conf/transport_method.go): what the Hysteria server answers an
// HTTP/3 request that does not authenticate.
type InboundHysteriaMasqueradeModel struct {
	Type        types.String `tfsdk:"type"`
	Dir         types.String `tfsdk:"dir"`
	URL         types.String `tfsdk:"url"`
	RewriteHost types.Bool   `tfsdk:"rewrite_host"`
	XForwarded  types.Bool   `tfsdk:"x_forwarded"`
	Insecure    types.Bool   `tfsdk:"insecure"`
	Content     types.String `tfsdk:"content"`
	Headers     types.Map    `tfsdk:"headers"`
	StatusCode  types.Int64  `tfsdk:"status_code"`
}

var masqueradeStringWireKeys = map[string]string{
	"type":    "type",
	"dir":     "dir",
	"url":     "url",
	"content": "content",
}

var masqueradeBoolWireKeys = map[string]string{
	"rewrite_host": "rewriteHost",
	"x_forwarded":  "xForwarded",
	"insecure":     "insecure",
}

func (m *InboundHysteriaMasqueradeModel) strings() map[string]*types.String {
	return map[string]*types.String{
		"type":    &m.Type,
		"dir":     &m.Dir,
		"url":     &m.URL,
		"content": &m.Content,
	}
}

func (m *InboundHysteriaMasqueradeModel) bools() map[string]*types.Bool {
	return map[string]*types.Bool{
		"rewrite_host": &m.RewriteHost,
		"x_forwarded":  &m.XForwarded,
		"insecure":     &m.Insecure,
	}
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

// Optional without Computed, like xmux: removing an attribute must remove its
// key. xray-core reads "", false, 0 and {} as unset, so reading them back as
// null loses nothing, and the schema rejects them so they cannot be planned.
func hysteriaMasqueradeBlockSchema() schema.SingleNestedBlock {
	nonEmpty := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			Optional:    true,
			Description: description,
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		}
	}
	trueOnly := func(description string) schema.BoolAttribute {
		return schema.BoolAttribute{
			Optional:    true,
			Description: description + " `false` is the default: omit the attribute instead.",
			Validators:  []validator.Bool{masqueradeTrueOnlyValidator{}},
		}
	}
	return schema.SingleNestedBlock{
		Description: "What the Hysteria server answers an HTTP/3 request without valid auth (xray `hysteriaSettings.masquerade`). " +
			"Without `type` it answers 404. An empty block is written as `{}`. " +
			"Removing the block, or an attribute, removes it from the stream settings.",
		Validators: []validator.Object{hysteriaMasqueradeValidator{}},
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "`file` serves `dir`, `proxy` reverse-proxies to `url`, `string` returns `content`. Omit for a 404.",
				Validators:  []validator.String{stringvalidator.OneOf("file", "proxy", "string")},
			},
			"dir":          nonEmpty("`file`: directory on the panel (or node) host to serve as a static site."),
			"url":          nonEmpty("`proxy`: upstream `http://` or `https://` URL, or the path of a unix socket."),
			"rewrite_host": trueOnly("`proxy`: send the upstream host as `Host` instead of the client's."),
			"x_forwarded":  trueOnly("`proxy`: add `X-Forwarded-*` headers (xray-core v26.9.9+, 3x-ui v3.8.0+; older cores ignore it)."),
			"insecure":     trueOnly("`proxy`: skip TLS verification of an `https://` upstream."),
			"content":      nonEmpty("`string`: response body."),
			"headers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "`string`: response headers.",
				Validators:  []validator.Map{mapvalidator.SizeAtLeast(1)},
			},
			"status_code": schema.Int64Attribute{
				Optional:    true,
				Description: "`string`: response status code; unset answers 200.",
				// net/http and quic-go panic on a status outside 100-999.
				Validators: []validator.Int64{int64validator.Between(100, 999)},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Validators
// ---------------------------------------------------------------------------

type masqueradeTrueOnlyValidator struct{}

func (masqueradeTrueOnlyValidator) Description(context.Context) string {
	return "must be true or omitted"
}

func (v masqueradeTrueOnlyValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (masqueradeTrueOnlyValidator) ValidateBool(_ context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueBool() {
		return
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid masquerade setting",
		"false is the xray-core default and reads back as unset, so omit the attribute instead.")
}

// hysteriaMasqueradeValidator rejects what makes xray-core's hysteria listener
// fail to start (transport/internet/hysteria/hub.go) or serve the wrong thing.
type hysteriaMasqueradeValidator struct{}

func (hysteriaMasqueradeValidator) Description(context.Context) string {
	return "type = \"file\" requires dir; type = \"proxy\" requires a url with an http, https or unix scheme"
}

func (v hysteriaMasqueradeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (hysteriaMasqueradeValidator) ValidateObject(_ context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	attrs := req.ConfigValue.Attributes()
	kind, _ := attrs["type"].(types.String)
	if kind.IsNull() || kind.IsUnknown() {
		return
	}
	switch kind.ValueString() {
	case "file":
		// http.Dir("") serves the working directory of the Xray process.
		if dir, _ := attrs["dir"].(types.String); dir.IsNull() {
			resp.Diagnostics.AddAttributeError(req.Path.AtName("dir"), "Missing masquerade directory",
				"type = \"file\" requires dir: without it xray-core serves its own working directory.")
		}
	case "proxy":
		u, _ := attrs["url"].(types.String)
		if u.IsUnknown() {
			return
		}
		if u.IsNull() {
			resp.Diagnostics.AddAttributeError(req.Path.AtName("url"), "Missing masquerade upstream",
				"type = \"proxy\" requires url.")
			return
		}
		if detail := masqueradeURLProblem(u.ValueString()); detail != "" {
			resp.Diagnostics.AddAttributeError(req.Path.AtName("url"), "Invalid masquerade upstream", detail)
		}
	}
}

// masqueradeURLProblem mirrors the scheme switch in xray-core's hysteria
// listener, which refuses to start on anything else.
func masqueradeURLProblem(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Sprintf("%q is not a URL: %v.", raw, err)
	}
	switch u.Scheme {
	case "http", "https":
		if u.Host == "" {
			return fmt.Sprintf("%q has no host.", raw)
		}
	case "", "unix":
		if u.Path == "" {
			return fmt.Sprintf("%q has no unix socket path.", raw)
		}
	default:
		return fmt.Sprintf("%q: xray-core accepts only http://, https:// or a unix socket path, and refuses to start the listener otherwise.", raw)
	}
	return ""
}

// ---------------------------------------------------------------------------
// Typed model <-> untyped map
// ---------------------------------------------------------------------------

func expandHysteriaMasqueradeFromModel(m *InboundHysteriaMasqueradeModel) map[string]any {
	out := map[string]any{}
	for key, v := range m.strings() {
		if !v.IsNull() && !v.IsUnknown() {
			out[key] = v.ValueString()
		}
	}
	for key, v := range m.bools() {
		if !v.IsNull() && !v.IsUnknown() {
			out[key] = v.ValueBool()
		}
	}
	if !m.Headers.IsNull() && !m.Headers.IsUnknown() {
		headers := map[string]any{}
		for k, v := range m.Headers.Elements() {
			if s, ok := v.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
				headers[k] = s.ValueString()
			}
		}
		out["headers"] = headers
	}
	if !m.StatusCode.IsNull() && !m.StatusCode.IsUnknown() {
		out["status_code"] = m.StatusCode.ValueInt64()
	}
	return out
}

func flattenHysteriaMasqueradeToModel(data map[string]any) *InboundHysteriaMasqueradeModel {
	m := &InboundHysteriaMasqueradeModel{StatusCode: types.Int64Null()}
	for key, dst := range m.strings() {
		if v, ok := data[key].(string); ok {
			*dst = types.StringValue(v)
		} else {
			*dst = types.StringNull()
		}
	}
	for key, dst := range m.bools() {
		if v, ok := data[key].(bool); ok {
			*dst = types.BoolValue(v)
		} else {
			*dst = types.BoolNull()
		}
	}
	if v, ok := data["headers"].(map[string]any); ok {
		m.Headers = anyMapToTypesMap(v)
	} else {
		m.Headers = types.MapNull(types.StringType)
	}
	if v, ok := data["status_code"]; ok {
		m.StatusCode = types.Int64Value(int64Value(v))
	}
	return m
}
