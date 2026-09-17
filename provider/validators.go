package provider

import (
	"context"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// ---------------------------------------------------------------------------
// Provider config validators
// ---------------------------------------------------------------------------

// endpointValidators validates the provider endpoint URL format.
func endpointValidators() []validator.String {
	return []validator.String{
		stringvalidator.RegexMatches(
			regexp.MustCompile(`^https?://.+`),
			"must be a valid HTTP or HTTPS URL (e.g. http://localhost:2053)",
		),
	}
}

// basePathValidators validates the provider base_path.
func basePathValidators() []validator.String {
	return []validator.String{
		stringvalidator.RegexMatches(
			regexp.MustCompile(`^/.*`),
			"must start with '/'",
		),
	}
}

// requestTimeoutValidators validates the provider request_timeout duration string.
func requestTimeoutValidators() []validator.String {
	return []validator.String{
		durationValidator{},
	}
}

// maxRetriesValidators validates the provider max_retries integer range.
func maxRetriesValidators() []validator.Int64 {
	return []validator.Int64{
		int64validator.Between(0, int64(maxAllowedRetries)),
	}
}

// subUpdatesValidators validates the sub_updates range (0–525600, i.e. up to one year in minutes).
// The panel widened this from min(1).max(168) to min(0).max(525600) in 3x-ui v3.5.0.
func subUpdatesValidators() []validator.Int64 {
	return []validator.Int64{
		int64validator.Between(0, 525600),
	}
}

// ---------------------------------------------------------------------------
// Inbound resource validators
// ---------------------------------------------------------------------------

// portValidators validates inbound port range (1-65535).
func portValidators() []validator.Int64 {
	return []validator.Int64{
		int64validator.Between(1, 65535),
	}
}

// protocolValidators validates inbound protocol enum values.
func protocolValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"vless", "vmess", "trojan", "shadowsocks",
			"http", "socks", "mixed", "wireguard",
			"dokodemo-door", "tunnel", "tun",
			"hysteria", "hysteria2", "mtproto", "amneziawg", "tuic",
		),
	}
}

// trafficResetValidators validates inbound traffic_reset enum values.
func trafficResetValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf("never", "hourly", "daily", "weekly", "monthly"),
	}
}

// ---------------------------------------------------------------------------
// Stream settings validators
// ---------------------------------------------------------------------------

// networkValidators validates stream network enum values.
func networkValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf("tcp", "ws", "grpc", "httpupgrade", "xhttp", "kcp", "hysteria"),
	}
}

// securityValidators validates stream security enum values.
func securityValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf("none", "tls", "reality"),
	}
}

// tcpHeaderTypeValidators validates TCP header_type enum values.
func tcpHeaderTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf("none", "http"),
	}
}

// kcpHeaderTypeValidators validates KCP header_type enum values.
func kcpHeaderTypeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf("none", "srtp", "utp", "wechat-video", "dtls", "wireguard"),
	}
}

// xhttpModeValidators validates XHTTP mode enum values.
func xhttpModeValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf("auto", "packet-up", "stream-up", "stream-one"),
	}
}

// realityClientVerValidators validates REALITY min_client_ver / max_client_ver.
// Xray parses the value as up to three dot-separated bytes, zero-filling the
// missing ones, and rejects a component above 255
// (infra/conf/transport_security.go). An empty string is deliberately not
// accepted: Xray substitutes its own default for it, so "" would read as
// "no bound" while meaning the opposite. Widen a bound by setting the extreme
// value for it — 0.0.0 or 255.255.255 — rather than by removing the attribute,
// which keeps the last applied value.
func realityClientVerValidators() []validator.String {
	component := `(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])`
	return []validator.String{
		stringvalidator.RegexMatches(
			regexp.MustCompile(`^`+component+`(\.`+component+`){0,2}$`),
			"must be one to three dot-separated numbers in the range 0-255 "+
				"(e.g. 26.3.27); use 0.0.0 for no lower bound and "+
				"255.255.255 for no upper bound",
		),
	}
}

// realityMaxTimediffValidators validates REALITY max_timediff.
// Xray models the field as uint64 (transport/internet/reality/config.proto).
func realityMaxTimediffValidators() []validator.Int64 {
	return []validator.Int64{
		int64validator.AtLeast(0),
	}
}

// tproxyValidators validates sockopt tproxy enum values.
func tproxyValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf("off", "redirect", "tproxy"),
	}
}

// domainStrategyValidators validates sockopt domain_strategy enum values.
func domainStrategyValidators() []validator.String {
	return []validator.String{
		stringvalidator.OneOf(
			"AsIs", "UseIP", "UseIPv6v4", "UseIPv6", "UseIPv4v6", "UseIPv4",
			"ForceIP", "ForceIPv6v4", "ForceIPv6", "ForceIPv4v6", "ForceIPv4",
		),
	}
}

// ---------------------------------------------------------------------------
// Custom duration validator
// ---------------------------------------------------------------------------

// durationValidator validates that a string can be parsed as a Go duration.
type durationValidator struct{}

func (durationValidator) Description(_ context.Context) string {
	return "must be a valid Go duration string (e.g. 30s, 1m, 2m30s)"
}

func (durationValidator) MarkdownDescription(_ context.Context) string {
	return "must be a valid Go duration string (e.g. `30s`, `1m`, `2m30s`)"
}

func (durationValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := time.ParseDuration(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid duration",
			err.Error(),
		)
	}
}

// ---------------------------------------------------------------------------
// Address-list validators
// ---------------------------------------------------------------------------

// addrOrPrefixListValidator validates a comma-separated list of bare IP
// addresses and CIDR prefixes, mirroring upstream's CheckNetipAddrOrPrefixList
// (internal/web/entity/entity.go). It deliberately uses netip rather than net:
// the two disagree (net accepts "10.0.0.0/024", netip does not), and the panel
// rejects the save with an opaque error when they diverge. Validating here moves
// that failure to plan time.
type addrOrPrefixListValidator struct{}

func (addrOrPrefixListValidator) Description(_ context.Context) string {
	return "must be a comma-separated list of IP addresses or CIDR prefixes"
}

func (addrOrPrefixListValidator) MarkdownDescription(ctx context.Context) string {
	return addrOrPrefixListValidator{}.Description(ctx)
}

func (addrOrPrefixListValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, entry := range strings.Split(req.ConfigValue.ValueString(), ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if _, err := netip.ParseAddr(entry); err == nil {
			continue
		}
		if _, err := netip.ParsePrefix(entry); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid address list entry",
				"Entry "+strconv.Quote(entry)+" is neither an IP address nor a CIDR prefix. "+
					"3x-ui parses this list with netip, which — unlike net — rejects forms such as "+
					`"10.0.0.0/024".`,
			)
		}
	}
}

// addrOrPrefixListValidators validates a comma-separated IP/CIDR list.
func addrOrPrefixListValidators() []validator.String {
	return []validator.String{addrOrPrefixListValidator{}}
}
