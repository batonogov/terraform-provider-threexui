package provider

import (
	"encoding/json"
	"fmt"
	"strings"
)

func buildStreamSettingsJSON(item map[string]any) string {
	if item == nil {
		return "{}"
	}

	payload := map[string]any{}
	if v, ok := item["network"].(string); ok && v != "" {
		payload["network"] = v
	}
	if v, ok := item["security"].(string); ok && v != "" {
		payload["security"] = v
	}
	if v, ok := item["external_proxy"]; ok {
		if list, ok := v.([]any); ok {
			payload["externalProxy"] = expandExternalProxy(list)
		}
	}
	if v, ok := item["reality_settings"]; ok {
		if list, ok := v.([]any); ok {
			if rs := expandRealitySettings(list); rs != nil {
				payload["realitySettings"] = rs
			}
		}
	}
	if v, ok := item["tls_settings"]; ok {
		if list, ok := v.([]any); ok {
			if ts := expandTLSSettings(list); ts != nil {
				payload["tlsSettings"] = ts
			}
		}
	}
	if v, ok := item["tcp_settings"]; ok {
		if list, ok := v.([]any); ok && len(list) > 0 {
			payload["tcpSettings"] = expandTCPSettings(list)
		}
	}
	if v, ok := item["ws_settings"]; ok {
		if list, ok := v.([]any); ok && len(list) > 0 {
			payload["wsSettings"] = expandWSSettings(list)
		}
	}
	if v, ok := item["grpc_settings"]; ok {
		if list, ok := v.([]any); ok && len(list) > 0 {
			payload["grpcSettings"] = expandGRPCSettings(list)
		}
	}
	if v, ok := item["httpupgrade_settings"]; ok {
		if list, ok := v.([]any); ok && len(list) > 0 {
			payload["httpupgradeSettings"] = expandHTTPUpgradeSettings(list)
		}
	}
	if v, ok := item["xhttp_settings"]; ok {
		if list, ok := v.([]any); ok && len(list) > 0 {
			payload["xhttpSettings"] = expandXHTTPSettings(list)
		}
	}
	if v, ok := item["kcp_settings"]; ok {
		if list, ok := v.([]any); ok && len(list) > 0 {
			payload["kcpSettings"] = expandKCPSettings(list)
		}
	}
	if v, ok := item["hysteria_settings"]; ok {
		if list, ok := v.([]any); ok && len(list) > 0 {
			payload["hysteriaSettings"] = expandHysteriaStreamSettings(list)
		}
	}
	if v, ok := item["sockopt"]; ok {
		if list, ok := v.([]any); ok && len(list) > 0 {
			payload["sockopt"] = expandSockopt(list)
		}
	}

	if len(payload) == 0 {
		return "{}"
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func flattenStreamSettings(stream string) ([]any, error) {
	if strings.TrimSpace(stream) == "" {
		return []any{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stream), &payload); err != nil {
		return nil, fmt.Errorf("failed to parse stream_settings JSON: %w", err)
	}
	out := map[string]any{}
	network, _ := payload["network"].(string)
	if v, ok := payload["network"].(string); ok {
		out["network"] = v
	}
	if v, ok := payload["security"].(string); ok {
		out["security"] = v
	}
	if v, ok := payload["externalProxy"].([]any); ok {
		out["external_proxy"] = flattenExternalProxy(v)
	}
	if v, ok := payload["realitySettings"].(map[string]any); ok {
		if rs := flattenRealitySettings(v); rs != nil {
			out["reality_settings"] = []any{rs}
		}
	}
	if v, ok := payload["tlsSettings"].(map[string]any); ok {
		if ts := flattenTLSSettings(v); ts != nil {
			out["tls_settings"] = []any{ts}
		}
	}
	if v, ok := payload["tcpSettings"].(map[string]any); ok {
		if ts := flattenTCPSettings(v); len(ts) > 0 || network == "tcp" {
			out["tcp_settings"] = []any{ts}
		}
	}
	if v, ok := payload["wsSettings"].(map[string]any); ok {
		if ws := flattenWSSettings(v); len(ws) > 0 || network == "ws" {
			out["ws_settings"] = []any{ws}
		}
	}
	if v, ok := payload["grpcSettings"].(map[string]any); ok {
		if gs := flattenGRPCSettings(v); len(gs) > 0 || network == "grpc" {
			out["grpc_settings"] = []any{gs}
		}
	}
	if v, ok := payload["httpupgradeSettings"].(map[string]any); ok {
		if hu := flattenHTTPUpgradeSettings(v); len(hu) > 0 || network == "httpupgrade" {
			out["httpupgrade_settings"] = []any{hu}
		}
	}
	if v, ok := payload["xhttpSettings"].(map[string]any); ok {
		if xh := flattenXHTTPSettings(v); len(xh) > 0 || network == "xhttp" {
			out["xhttp_settings"] = []any{xh}
		}
	}
	if v, ok := payload["kcpSettings"].(map[string]any); ok {
		if kcp := flattenKCPSettings(v); len(kcp) > 0 || network == "kcp" {
			out["kcp_settings"] = []any{kcp}
		}
	}
	if v, ok := payload["hysteriaSettings"].(map[string]any); ok {
		if h := flattenHysteriaStreamSettings(v); len(h) > 0 || network == "hysteria" || network == "hysteria2" {
			out["hysteria_settings"] = []any{h}
		}
	}
	if v, ok := payload["sockopt"].(map[string]any); ok {
		if so := flattenSockopt(v); len(so) > 0 {
			out["sockopt"] = []any{so}
		}
	}
	if len(out) == 0 {
		return []any{}, nil
	}
	return []any{out}, nil
}

func expandExternalProxy(list []any) []any {
	if len(list) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		if v, ok := m["dest"].(string); ok && v != "" {
			entry["dest"] = v
		}
		if v, ok := m["port"]; ok {
			if p := intValue(v); p != 0 {
				entry["port"] = p
			}
		}
		if v, ok := m["remark"].(string); ok && v != "" {
			entry["remark"] = v
		}
		if v, ok := m["force_tls"].(string); ok && v != "" {
			entry["forceTls"] = v
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func flattenExternalProxy(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		if v, ok := m["dest"].(string); ok {
			entry["dest"] = v
		}
		if v, ok := m["port"]; ok {
			entry["port"] = intValue(v)
		}
		if v, ok := m["remark"].(string); ok {
			entry["remark"] = v
		}
		if v, ok := m["forceTls"].(string); ok {
			entry["force_tls"] = v
		}
		out = append(out, entry)
	}
	return out
}

func expandRealitySettings(list []any) map[string]any {
	if len(list) == 0 {
		return nil
	}
	item, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	rs := map[string]any{}
	target := ""
	if v, ok := item["show"]; ok {
		rs["show"] = boolValue(v)
	}
	if v, ok := item["xver"]; ok {
		rs["xver"] = intValue(v)
	}
	if v, ok := item["target"].(string); ok && v != "" {
		target = v
		rs["target"] = v
	}
	if v, ok := item["server_names"]; ok {
		if list, ok := v.([]any); ok {
			rs["serverNames"] = expandStringList(list)
		}
	}
	if v, ok := item["private_key"].(string); ok && v != "" {
		rs["privateKey"] = v
	}
	if v, ok := item["min_client_ver"].(string); ok && v != "" {
		rs["minClientVer"] = v
	}
	if v, ok := item["max_client_ver"].(string); ok && v != "" {
		rs["maxClientVer"] = v
	}
	if v, ok := item["max_timediff"]; ok {
		rs["maxTimediff"] = int64Value(v)
	}
	if v, ok := item["short_ids"]; ok {
		if list, ok := v.([]any); ok {
			rs["shortIds"] = expandStringList(list)
		}
	}
	if v, ok := item["mldsa65_seed"].(string); ok && v != "" {
		rs["mldsa65Seed"] = v
	}
	if v, ok := item["settings"]; ok {
		if m, ok := v.(map[string]any); ok {
			if s := expandRealityInnerSettings(m); s != nil {
				rs["settings"] = s
			}
		}
	}
	if !hasRealityServerNames(rs) {
		if target != "" {
			host := strings.Split(target, ":")[0]
			if host != "" {
				rs["serverNames"] = []any{host}
			}
		}
	}
	if !hasRealityServerNames(rs) {
		rs["target"] = "www.amazon.com:443"
		rs["serverNames"] = []any{"www.amazon.com", "amazon.com"}
	}
	if len(rs) == 0 {
		return nil
	}
	return rs
}

func expandRealityInnerSettings(item map[string]any) map[string]any {
	if len(item) == 0 {
		return nil
	}
	out := map[string]any{}
	if v, ok := item["public_key"].(string); ok && v != "" {
		out["publicKey"] = v
	}
	if v, ok := item["fingerprint"].(string); ok && v != "" {
		out["fingerprint"] = v
	}
	if v, ok := item["server_name"].(string); ok && v != "" {
		out["serverName"] = v
	}
	if v, ok := item["spider_x"].(string); ok && v != "" {
		out["spiderX"] = v
	}
	if v, ok := item["mldsa65_verify"].(string); ok && v != "" {
		out["mldsa65Verify"] = v
	}
	return out
}

func flattenRealitySettings(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := map[string]any{}
	if v, ok := in["show"].(bool); ok {
		out["show"] = v
	}
	if v, ok := in["xver"]; ok {
		out["xver"] = intValue(v)
	}
	if v, ok := in["target"].(string); ok {
		out["target"] = v
	}
	if v, ok := in["serverNames"].([]any); ok {
		out["server_names"] = v
	}
	if v, ok := in["privateKey"].(string); ok {
		out["private_key"] = v
	}
	if v, ok := in["minClientVer"].(string); ok {
		out["min_client_ver"] = v
	}
	if v, ok := in["maxClientVer"].(string); ok {
		out["max_client_ver"] = v
	}
	// xray-core declares the field as `maxTimeDiff`
	// (infra/conf/transport_security.go); the panel's own schema writes
	// `maxTimediff` (frontend/src/schemas/protocols/security/reality.ts) and
	// binds the canonical spelling only because encoding/json falls back to a
	// case-insensitive match. 3x-ui stores streamSettings as opaque JSON text,
	// so an inbound authored by external tooling keeps whichever spelling it
	// was written with. Reading just one would import the other as null and
	// drop it from the JSON on the next update — silently disabling the gate.
	// Accept both, canonical first, mirroring the panel's own dest -> target
	// aliasing for the same class of problem.
	if v, ok := in["maxTimeDiff"]; ok {
		out["max_timediff"] = int64Value(v)
	} else if v, ok := in["maxTimediff"]; ok {
		out["max_timediff"] = int64Value(v)
	}
	if v, ok := in["shortIds"].([]any); ok {
		out["short_ids"] = v
	}
	if v, ok := in["mldsa65Seed"].(string); ok {
		out["mldsa65_seed"] = v
	}
	if v, ok := in["settings"].(map[string]any); ok {
		if s := flattenRealityInnerSettings(v); s != nil {
			out["settings"] = s
		}
	}
	return out
}

func flattenRealityInnerSettings(in map[string]any) map[string]any {
	out := map[string]any{}
	if v, ok := in["publicKey"].(string); ok {
		out["public_key"] = v
	}
	if v, ok := in["fingerprint"].(string); ok {
		out["fingerprint"] = v
	}
	if v, ok := in["serverName"].(string); ok {
		out["server_name"] = v
	}
	if v, ok := in["spiderX"].(string); ok {
		out["spider_x"] = v
	}
	if v, ok := in["mldsa65Verify"].(string); ok {
		out["mldsa65_verify"] = v
	}
	return out
}

// ---------------------------------------------------------------------------
// TLS
// ---------------------------------------------------------------------------

func expandTLSSettings(list []any) map[string]any {
	if len(list) == 0 {
		return nil
	}
	item, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	if v, ok := item["server_name"].(string); ok && v != "" {
		out["serverName"] = v
	}
	if v, ok := item["fingerprint"].(string); ok && v != "" {
		out["fingerprint"] = v
	}
	if v, ok := item["allow_insecure"]; ok {
		out["allowInsecure"] = boolValue(v)
	}
	if v, ok := item["alpn"]; ok {
		if list, ok := v.([]any); ok {
			out["alpn"] = expandStringList(list)
		}
	}
	if v, ok := item["min_version"].(string); ok && v != "" {
		out["minVersion"] = v
	}
	if v, ok := item["max_version"].(string); ok && v != "" {
		out["maxVersion"] = v
	}
	if v, ok := item["cipher"].(string); ok && v != "" {
		out["cipher"] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func flattenTLSSettings(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := map[string]any{}
	if v, ok := in["serverName"].(string); ok {
		out["server_name"] = v
	}
	if v, ok := in["fingerprint"].(string); ok {
		out["fingerprint"] = v
	}
	if v, ok := in["allowInsecure"].(bool); ok {
		out["allow_insecure"] = v
	}
	if v, ok := in["alpn"].([]any); ok {
		out["alpn"] = v
	}
	if v, ok := in["minVersion"].(string); ok {
		out["min_version"] = v
	}
	if v, ok := in["maxVersion"].(string); ok {
		out["max_version"] = v
	}
	if v, ok := in["cipher"].(string); ok {
		out["cipher"] = v
	}
	return out
}

func expandTCPSettings(list []any) map[string]any {
	if len(list) == 0 {
		return nil
	}
	item, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	if v, ok := item["accept_proxy_protocol"]; ok {
		out["acceptProxyProtocol"] = boolValue(v)
	}
	// Support both nested header block and flat header_type string
	if v, ok := item["header"]; ok {
		if list, ok := v.([]any); ok {
			if h := expandTCPHeader(list); h != nil {
				out["header"] = h
			}
		}
	}
	if v, ok := item["header_type"].(string); ok && v != "" {
		out["header"] = map[string]any{"type": v}
	}
	return out
}

func expandTCPHeader(list []any) map[string]any {
	if len(list) == 0 {
		return nil
	}
	item, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	if v, ok := item["type"].(string); ok && v != "" {
		out["type"] = v
	}
	return out
}

func flattenTCPSettings(in map[string]any) map[string]any {
	out := map[string]any{}
	if v, ok := in["acceptProxyProtocol"].(bool); ok {
		out["accept_proxy_protocol"] = v
	}
	if v, ok := in["header"].(map[string]any); ok {
		if h := flattenTCPHeader(v); h != nil {
			out["header"] = []any{h}
		}
	}
	return out
}

func flattenTCPHeader(in map[string]any) map[string]any {
	out := map[string]any{}
	if v, ok := in["type"].(string); ok {
		out["type"] = v
	}
	return out
}

// ---------------------------------------------------------------------------
// WebSocket
// ---------------------------------------------------------------------------

func expandWSSettings(list []any) map[string]any {
	if len(list) == 0 {
		return nil
	}
	item, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	if v, ok := item["path"].(string); ok && v != "" {
		out["path"] = v
	}
	if v, ok := item["headers"].(map[string]any); ok && len(v) > 0 {
		out["headers"] = v
	}
	return out
}

func flattenWSSettings(in map[string]any) map[string]any {
	out := map[string]any{}
	if v, ok := in["path"].(string); ok {
		out["path"] = v
	}
	if v, ok := in["headers"].(map[string]any); ok {
		out["headers"] = v
	}
	return out
}

// ---------------------------------------------------------------------------
// gRPC
// ---------------------------------------------------------------------------

func expandGRPCSettings(list []any) map[string]any {
	if len(list) == 0 {
		return nil
	}
	item, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	if v, ok := item["service_name"].(string); ok && v != "" {
		out["serviceName"] = v
	}
	if v, ok := item["multi_mode"]; ok {
		out["multiMode"] = boolValue(v)
	}
	if v, ok := item["idle_timeout"]; ok {
		if n := intValue(v); n != 0 {
			out["idle_timeout"] = n
		}
	}
	if v, ok := item["health_check_timeout"]; ok {
		if n := intValue(v); n != 0 {
			out["health_check_timeout"] = n
		}
	}
	if v, ok := item["permit_without_stream"]; ok {
		out["permitWithoutStream"] = boolValue(v)
	}
	if v, ok := item["initial_windows_size"]; ok {
		if n := intValue(v); n != 0 {
			out["initial_windows_size"] = n
		}
	}
	return out
}

func flattenGRPCSettings(in map[string]any) map[string]any {
	out := map[string]any{}
	if v, ok := in["serviceName"].(string); ok {
		out["service_name"] = v
	}
	if v, ok := in["multiMode"].(bool); ok {
		out["multi_mode"] = v
	}
	if v, ok := in["idle_timeout"]; ok {
		out["idle_timeout"] = intValue(v)
	}
	if v, ok := in["health_check_timeout"]; ok {
		out["health_check_timeout"] = intValue(v)
	}
	if v, ok := in["permitWithoutStream"].(bool); ok {
		out["permit_without_stream"] = v
	}
	if v, ok := in["initial_windows_size"]; ok {
		out["initial_windows_size"] = intValue(v)
	}
	return out
}

// ---------------------------------------------------------------------------
// HTTP Upgrade
// ---------------------------------------------------------------------------

func expandHTTPUpgradeSettings(list []any) map[string]any {
	if len(list) == 0 {
		return nil
	}
	item, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	if v, ok := item["path"].(string); ok && v != "" {
		out["path"] = v
	}
	if v, ok := item["host"].(string); ok && v != "" {
		out["host"] = v
	}
	return out
}

func flattenHTTPUpgradeSettings(in map[string]any) map[string]any {
	out := map[string]any{}
	if v, ok := in["path"].(string); ok {
		out["path"] = v
	}
	if v, ok := in["host"].(string); ok {
		out["host"] = v
	}
	return out
}

// ---------------------------------------------------------------------------
// XHTTP
// ---------------------------------------------------------------------------

func expandXHTTPSettings(list []any) map[string]any {
	if len(list) == 0 {
		return nil
	}
	item, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	if v, ok := item["path"].(string); ok && v != "" {
		out["path"] = v
	}
	if v, ok := item["mode"].(string); ok && v != "" {
		out["mode"] = v
	}
	if v, ok := item["no_sse_header"]; ok {
		out["noSSEHeader"] = boolValue(v)
	}
	if v, ok := item["keep_alive_interval"]; ok {
		if n := intValue(v); n != 0 {
			out["keepAliveInterval"] = n
		}
	}
	if v, ok := item["x_padding_bytes"].(string); ok && v != "" {
		out["xPaddingBytes"] = v
	}
	if v, ok := item["x_padding_obfs_mode"]; ok {
		out["xPaddingObfsMode"] = boolValue(v)
	}
	if v, ok := item["x_padding_key"].(string); ok && v != "" {
		out["xPaddingKey"] = v
	}
	if v, ok := item["x_padding_header"].(string); ok && v != "" {
		out["xPaddingHeader"] = v
	}
	if v, ok := item["x_padding_placement"].(string); ok && v != "" {
		out["xPaddingPlacement"] = v
	}
	if v, ok := item["x_padding_method"].(string); ok && v != "" {
		out["xPaddingMethod"] = v
	}
	return out
}

func flattenXHTTPSettings(in map[string]any) map[string]any {
	out := map[string]any{}
	if v, ok := in["path"].(string); ok {
		out["path"] = v
	}
	if v, ok := in["mode"].(string); ok {
		out["mode"] = v
	}
	if v, ok := in["noSSEHeader"].(bool); ok {
		out["no_sse_header"] = v
	}
	if v, ok := in["keepAliveInterval"]; ok {
		out["keep_alive_interval"] = intValue(v)
	}
	if v, ok := in["xPaddingBytes"].(string); ok {
		out["x_padding_bytes"] = v
	}
	if v, ok := in["xPaddingObfsMode"].(bool); ok {
		out["x_padding_obfs_mode"] = v
	}
	if v, ok := in["xPaddingKey"].(string); ok {
		out["x_padding_key"] = v
	}
	if v, ok := in["xPaddingHeader"].(string); ok {
		out["x_padding_header"] = v
	}
	if v, ok := in["xPaddingPlacement"].(string); ok {
		out["x_padding_placement"] = v
	}
	if v, ok := in["xPaddingMethod"].(string); ok {
		out["x_padding_method"] = v
	}
	return out
}

// ---------------------------------------------------------------------------
// KCP
// ---------------------------------------------------------------------------

func expandKCPSettings(list []any) map[string]any {
	if len(list) == 0 {
		return nil
	}
	item, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	if v, ok := item["mtu"]; ok {
		if n := intValue(v); n != 0 {
			out["mtu"] = n
		}
	}
	if v, ok := item["tti"]; ok {
		if n := intValue(v); n != 0 {
			out["tti"] = n
		}
	}
	if v, ok := item["uplink_capacity"]; ok {
		if n := intValue(v); n != 0 {
			out["uplinkCapacity"] = n
		}
	}
	if v, ok := item["downlink_capacity"]; ok {
		if n := intValue(v); n != 0 {
			out["downlinkCapacity"] = n
		}
	}
	if v, ok := item["cwnd_multiplier"]; ok {
		if n := intValue(v); n != 0 {
			out["cwndMultiplier"] = n
		}
	}
	if v, ok := item["max_sending_window"]; ok {
		if n := intValue(v); n != 0 {
			out["maxSendingWindow"] = n
		}
	}
	if v, ok := item["header_type"].(string); ok && v != "" {
		out["header"] = map[string]any{"type": v}
	}
	return out
}

func flattenKCPSettings(in map[string]any) map[string]any {
	out := map[string]any{}
	if v, ok := in["mtu"]; ok {
		out["mtu"] = intValue(v)
	}
	if v, ok := in["tti"]; ok {
		out["tti"] = intValue(v)
	}
	if v, ok := in["uplinkCapacity"]; ok {
		out["uplink_capacity"] = intValue(v)
	}
	if v, ok := in["downlinkCapacity"]; ok {
		out["downlink_capacity"] = intValue(v)
	}
	if v, ok := in["cwndMultiplier"]; ok {
		out["cwnd_multiplier"] = intValue(v)
	}
	if v, ok := in["maxSendingWindow"]; ok {
		out["max_sending_window"] = intValue(v)
	}
	if v, ok := in["header"].(map[string]any); ok {
		if t, ok := v["type"].(string); ok {
			out["header_type"] = t
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Hysteria
// ---------------------------------------------------------------------------

func expandHysteriaStreamSettings(list []any) map[string]any {
	if len(list) == 0 {
		return nil
	}
	item, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	if v, ok := item["protocol"].(string); ok && v != "" {
		out["protocol"] = v
	}
	if v, ok := item["version"]; ok {
		if n := intValue(v); n != 0 {
			out["version"] = n
		}
	}
	if v, ok := item["auth"].(string); ok && v != "" {
		out["auth"] = v
	}
	if v, ok := item["udp_idle_timeout"]; ok {
		if n := intValue(v); n != 0 {
			out["udpIdleTimeout"] = n
		}
	}
	return out
}

func flattenHysteriaStreamSettings(in map[string]any) map[string]any {
	out := map[string]any{}
	if v, ok := in["protocol"].(string); ok {
		out["protocol"] = v
	}
	if v, ok := in["version"]; ok {
		out["version"] = intValue(v)
	}
	if v, ok := in["auth"].(string); ok {
		out["auth"] = v
	}
	if v, ok := in["udpIdleTimeout"]; ok {
		out["udp_idle_timeout"] = intValue(v)
	}
	return out
}

// ---------------------------------------------------------------------------
// Sockopt
// ---------------------------------------------------------------------------

func expandSockopt(list []any) map[string]any {
	if len(list) == 0 {
		return nil
	}
	item, ok := list[0].(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	if v, ok := item["mark"]; ok {
		if n := intValue(v); n != 0 {
			out["mark"] = n
		}
	}
	if v, ok := item["tcp_keep_alive_interval"]; ok {
		if n := intValue(v); n != 0 {
			out["tcpKeepAliveInterval"] = n
		}
	}
	if v, ok := item["tcp_no_delay"]; ok {
		out["tcpNoDelay"] = boolValue(v)
	}
	if v, ok := item["tfo_enable"]; ok {
		out["tcpFastOpen"] = boolValue(v)
	}
	if v, ok := item["tproxy"].(string); ok && v != "" {
		out["tproxy"] = v
	}
	if v, ok := item["domain_strategy"].(string); ok && v != "" {
		out["domainStrategy"] = v
	}
	return out
}

func flattenSockopt(in map[string]any) map[string]any {
	out := map[string]any{}
	if v, ok := in["mark"]; ok {
		out["mark"] = intValue(v)
	}
	if v, ok := in["tcpKeepAliveInterval"]; ok {
		out["tcp_keep_alive_interval"] = intValue(v)
	}
	if v, ok := in["tcpNoDelay"].(bool); ok {
		out["tcp_no_delay"] = v
	}
	if v, ok := in["tcpFastOpen"].(bool); ok {
		out["tfo_enable"] = v
	}
	if v, ok := in["tproxy"].(string); ok {
		out["tproxy"] = v
	}
	if v, ok := in["domainStrategy"].(string); ok {
		out["domain_strategy"] = v
	}
	return out
}
