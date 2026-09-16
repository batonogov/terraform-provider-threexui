package provider

import (
	"encoding/json"
	"fmt"
	"strings"
)

func buildSettingsJSON(item map[string]any, protocol string) string {
	if item == nil {
		return "{}"
	}

	payload := map[string]any{}
	if v, ok := item["decryption"].(string); ok && v != "" {
		payload["decryption"] = v
	}
	if v, ok := item["encryption"].(string); ok && v != "" {
		payload["encryption"] = v
	}
	if v, ok := item["fallbacks"]; ok {
		if list, ok := v.([]any); ok {
			payload["fallbacks"] = expandFallbacks(list)
		}
	}
	if v, ok := item["selected_auth"].(string); ok && v != "" {
		payload["selectedAuth"] = v
	}
	if v, ok := item["testseed"]; ok {
		switch ts := v.(type) {
		case []any:
			payload["testseed"] = flattenIntList(ts)
		case []int:
			payload["testseed"] = ts
		}
	}
	if v, ok := item["method"].(string); ok && v != "" {
		payload["method"] = v
	}
	if v, ok := item["password"].(string); ok && v != "" {
		payload["password"] = v
	}
	if v, ok := item["network"].(string); ok && v != "" {
		payload["network"] = v
	}
	if v, ok := item["iv_check"]; ok {
		payload["ivCheck"] = boolValue(v)
	}
	if v, ok := item["allow_transparent"]; ok {
		payload["allowTransparent"] = boolValue(v)
	}
	if v, ok := item["auth"].(string); ok && v != "" {
		payload["auth"] = v
	}
	if v, ok := item["accounts"]; ok {
		if list, ok := v.([]any); ok {
			payload["accounts"] = expandAccounts(list)
		}
	}
	if v, ok := item["udp"]; ok {
		payload["udp"] = boolValue(v)
	}
	if v, ok := item["ip"].(string); ok && v != "" {
		payload["ip"] = v
	}
	if v, ok := item["address"].(string); ok && v != "" {
		payload["address"] = v
	}
	if v, ok := item["rewrite_address"].(string); ok && v != "" {
		payload["rewriteAddress"] = v
	}
	if v, ok := item["port"]; ok {
		if p := intValue(v); p != 0 {
			payload["port"] = p
		}
	}
	if v, ok := item["rewrite_port"]; ok {
		payload["rewritePort"] = intValue(v)
	}
	if v, ok := item["port_map"]; ok {
		switch pm := v.(type) {
		case map[string]any:
			payload["portMap"] = expandStringMap(pm)
		case map[string]string:
			payload["portMap"] = pm
		}
	}
	if v, ok := item["follow_redirect"]; ok {
		payload["followRedirect"] = boolValue(v)
	}
	if v, ok := item["allowed_network"].(string); ok && v != "" {
		payload["allowedNetwork"] = v
	}
	if v, ok := item["mtu"]; ok {
		switch val := v.(type) {
		case []any:
			if len(val) > 0 {
				payload["mtu"] = val
			}
		default:
			if m := intValue(v); m != 0 {
				payload["mtu"] = m
			}
		}
	}
	if v, ok := item["gateway"]; ok {
		if list, ok := v.([]any); ok && len(list) > 0 {
			payload["gateway"] = list
		}
	}
	if v, ok := item["dns"]; ok {
		if list, ok := v.([]any); ok && len(list) > 0 {
			payload["dns"] = list
		}
	}
	if v, ok := item["secret_key"].(string); ok && v != "" {
		payload["secretKey"] = v
	}
	if v, ok := item["no_kernel_tun"]; ok {
		payload["noKernelTun"] = boolValue(v)
	}
	if v, ok := item["peers"]; ok {
		if list, ok := v.([]any); ok {
			payload["peers"] = expandPeers(list)
		}
	}
	if v, ok := item["version"]; ok {
		if n := intValue(v); n != 0 {
			payload["version"] = n
		}
	}
	if v, ok := item["name"].(string); ok && v != "" {
		payload["name"] = v
	}
	if v, ok := item["user_level"]; ok {
		if ul := intValue(v); ul != 0 {
			payload["userLevel"] = ul
		}
	}
	// WireGuard multi-client (3x-ui v3.4.2+): the panel stores WireGuard peers
	// under `settings.clients[]` (the same key vmess/vless use), but unlike
	// vmess/vless they are managed via `threexui_inbound` itself — not via
	// `threexui_inbound_client`. Forward `clients[]` ONLY for wireguard; for
	// every other protocol it is deliberately stripped (managed by
	// threexui_inbound_client, surfaced there, not here).
	if protocol == "wireguard" || protocol == "amneziawg" || protocol == "tuic" {
		if v, ok := item["clients"]; ok {
			if list, ok := v.([]any); ok && len(list) > 0 {
				payload["clients"] = list
			}
		}
	}

	// AmneziaWG (3x-ui v3.7.0+) nests its server parameters under `server`
	// instead of spreading them over the top level, and the expander already
	// emits camelCase keys for it. Forward the object verbatim: folding it into
	// the flat key table above would collide on `mtu` (WireGuard), `privateKey`
	// / `publicKey` and `routeThroughXray` (MTProto).
	if protocol == "amneziawg" || protocol == "tuic" {
		if v, ok := item["server"]; ok {
			if server, ok := v.(map[string]any); ok && len(server) > 0 {
				payload["server"] = server
			}
		}
	}

	// MTProto settings (3x-ui v3.3.0+)
	if v, ok := item["fake_tls_domain"].(string); ok && v != "" {
		payload["fakeTlsDomain"] = v
	}
	if v, ok := item["proxy_protocol_listener"]; ok {
		payload["proxyProtocolListener"] = boolValue(v)
	}
	if v, ok := item["prefer_ip"].(string); ok && v != "" {
		payload["preferIp"] = v
	}
	if v, ok := item["debug"]; ok {
		payload["debug"] = boolValue(v)
	}
	if v, ok := item["domain_fronting"]; ok {
		if list, ok := v.([]any); ok && len(list) > 0 {
			payload["domainFronting"] = expandMtprotoDomainFronting(list)
		}
	}
	if v, ok := item["outbound_tag"].(string); ok && v != "" {
		payload["outboundTag"] = v
	}
	if v, ok := item["route_through_xray"]; ok {
		payload["routeThroughXray"] = boolValue(v)
	}
	if v, ok := item["route_xray_port"]; ok {
		if p := intValue(v); p != 0 {
			payload["routeXrayPort"] = p
		}
	}
	if v, ok := item["public_ipv4"].(string); ok && v != "" {
		payload["publicIpv4"] = v
	}
	if v, ok := item["public_ipv6"].(string); ok && v != "" {
		payload["publicIpv6"] = v
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

func flattenSettings(settings string, protocol string) ([]any, error) {
	if strings.TrimSpace(settings) == "" {
		return []any{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(settings), &payload); err != nil {
		return nil, fmt.Errorf("failed to parse settings JSON: %w", err)
	}
	out := map[string]any{}
	if v, ok := payload["decryption"].(string); ok {
		out["decryption"] = v
	}
	if v, ok := payload["encryption"].(string); ok {
		out["encryption"] = v
	}
	if v, ok := payload["fallbacks"].([]any); ok {
		out["fallbacks"] = flattenFallbacks(v)
	}
	if v, ok := payload["selectedAuth"].(string); ok {
		out["selected_auth"] = v
	}
	if v, ok := payload["testseed"].([]any); ok {
		out["testseed"] = flattenIntList(v)
	}
	if v, ok := payload["method"].(string); ok {
		out["method"] = v
	}
	if v, ok := payload["password"].(string); ok {
		out["password"] = v
	}
	if v, ok := payload["network"].(string); ok {
		out["network"] = v
	}
	if v, ok := payload["ivCheck"].(bool); ok {
		out["iv_check"] = v
	}
	if v, ok := payload["allowTransparent"].(bool); ok {
		out["allow_transparent"] = v
	}
	if v, ok := payload["auth"].(string); ok {
		out["auth"] = v
	}
	if v, ok := payload["accounts"].([]any); ok {
		out["accounts"] = flattenAccounts(v)
	}
	if v, ok := payload["udp"].(bool); ok {
		out["udp"] = v
	}
	if v, ok := payload["ip"].(string); ok {
		out["ip"] = v
	}
	if v, ok := payload["address"].(string); ok {
		out["address"] = v
	}
	if v, ok := payload["rewriteAddress"].(string); ok {
		out["rewrite_address"] = v
	}
	if v, ok := payload["port"]; ok {
		out["port"] = intValue(v)
	}
	if v, ok := payload["rewritePort"]; ok {
		out["rewrite_port"] = intValue(v)
	}
	if v, ok := payload["portMap"].(map[string]any); ok {
		out["port_map"] = flattenStringMap(v)
	}
	if v, ok := payload["followRedirect"].(bool); ok {
		out["follow_redirect"] = v
	}
	if v, ok := payload["allowedNetwork"].(string); ok {
		out["allowed_network"] = v
	}
	if v, ok := payload["mtu"]; ok {
		switch val := v.(type) {
		case []any:
			out["mtu"] = val
		default:
			n := intValue(v)
			out["mtu"] = []any{n, n}
		}
	}
	if v, ok := payload["gateway"].([]any); ok {
		out["gateway"] = v
	}
	if v, ok := payload["dns"].([]any); ok {
		out["dns"] = v
	}
	if v, ok := payload["secretKey"].(string); ok {
		out["secret_key"] = v
	}
	if v, ok := payload["noKernelTun"].(bool); ok {
		out["no_kernel_tun"] = v
	}
	if v, ok := payload["peers"].([]any); ok {
		out["peers"] = flattenPeers(v)
	}
	if v, ok := payload["version"]; ok {
		out["version"] = intValue(v)
	}
	if v, ok := payload["name"].(string); ok {
		out["name"] = v
	}
	if v, ok := payload["userLevel"]; ok {
		out["user_level"] = intValue(v)
	}
	// WireGuard multi-client: forward clients[] back ONLY for wireguard (see
	// buildSettingsJSON). For vmess/vless/trojan/SS/hysteria the clients array is
	// managed via threexui_inbound_client and must stay stripped here.
	if protocol == "wireguard" || protocol == "amneziawg" || protocol == "tuic" {
		if v, ok := payload["clients"].([]any); ok && len(v) > 0 {
			out["clients"] = v
		}
	}

	// AmneziaWG server block — passed through verbatim, mirroring buildSettingsJSON.
	if protocol == "amneziawg" || protocol == "tuic" {
		if v, ok := payload["server"].(map[string]any); ok && len(v) > 0 {
			out["server"] = v
		}
	}

	// MTProto settings (3x-ui v3.3.0+)
	if v, ok := payload["fakeTlsDomain"].(string); ok {
		out["fake_tls_domain"] = v
	}
	if v, ok := payload["proxyProtocolListener"].(bool); ok {
		out["proxy_protocol_listener"] = v
	}
	if v, ok := payload["preferIp"].(string); ok {
		out["prefer_ip"] = v
	}
	if v, ok := payload["debug"].(bool); ok {
		out["debug"] = v
	}
	if v, ok := payload["domainFronting"].([]any); ok {
		out["domain_fronting"] = flattenMtprotoDomainFronting(v)
	}
	if v, ok := payload["outboundTag"].(string); ok {
		out["outbound_tag"] = v
	}
	if v, ok := payload["routeThroughXray"].(bool); ok {
		out["route_through_xray"] = v
	}
	if v, ok := payload["routeXrayPort"]; ok {
		out["route_xray_port"] = intValue(v)
	}
	if v, ok := payload["publicIpv4"].(string); ok {
		out["public_ipv4"] = v
	}
	if v, ok := payload["publicIpv6"].(string); ok {
		out["public_ipv6"] = v
	}
	if len(out) == 0 {
		return []any{}, nil
	}
	return []any{out}, nil
}

func expandFallbacks(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		if v, ok := m["name"].(string); ok && v != "" {
			entry["name"] = v
		}
		if v, ok := m["alpn"].(string); ok && v != "" {
			entry["alpn"] = v
		}
		if v, ok := m["path"].(string); ok && v != "" {
			entry["path"] = v
		}
		if v, ok := m["dest"].(string); ok && v != "" {
			entry["dest"] = v
		}
		if v, ok := m["xver"]; ok {
			entry["xver"] = intValue(v)
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func flattenFallbacks(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		if v, ok := m["name"].(string); ok {
			entry["name"] = v
		}
		if v, ok := m["alpn"].(string); ok {
			entry["alpn"] = v
		}
		if v, ok := m["path"].(string); ok {
			entry["path"] = v
		}
		if v, ok := m["dest"].(string); ok {
			entry["dest"] = v
		}
		if v, ok := m["xver"]; ok {
			entry["xver"] = intValue(v)
		}
		out = append(out, entry)
	}
	return out
}

func expandAccounts(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		if v, ok := m["user"].(string); ok && v != "" {
			entry["user"] = v
		}
		if v, ok := m["pass"].(string); ok && v != "" {
			entry["pass"] = v
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func flattenAccounts(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		if v, ok := m["user"].(string); ok {
			entry["user"] = v
		}
		if v, ok := m["pass"].(string); ok {
			entry["pass"] = v
		}
		out = append(out, entry)
	}
	return out
}

func expandPeers(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		if v, ok := m["private_key"].(string); ok && v != "" {
			entry["privateKey"] = v
		}
		if v, ok := m["public_key"].(string); ok && v != "" {
			entry["publicKey"] = v
		}
		if v, ok := m["pre_shared_key"].(string); ok && v != "" {
			entry["preSharedKey"] = v
		}
		if v, ok := m["allowed_ips"]; ok {
			if list, ok := v.([]any); ok {
				entry["allowedIPs"] = expandStringList(list)
			}
		}
		if v, ok := m["keep_alive"]; ok {
			if ka := intValue(v); ka != 0 {
				entry["keepAlive"] = ka
			}
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func flattenPeers(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		if v, ok := m["privateKey"].(string); ok {
			entry["private_key"] = v
		}
		if v, ok := m["publicKey"].(string); ok {
			entry["public_key"] = v
		}
		if v, ok := m["preSharedKey"].(string); ok {
			entry["pre_shared_key"] = v
		}
		if v, ok := m["allowedIPs"].([]any); ok {
			entry["allowed_ips"] = v
		}
		if v, ok := m["keepAlive"]; ok {
			entry["keep_alive"] = intValue(v)
		}
		out = append(out, entry)
	}
	return out
}

func expandStringMap(in map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func flattenStringMap(in map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

func flattenIntList(list []any) []int {
	out := make([]int, 0, len(list))
	for _, v := range list {
		out = append(out, intValue(v))
	}
	return out
}

func expandMtprotoDomainFronting(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		if v, ok := m["ip"].(string); ok && v != "" {
			entry["ip"] = v
		}
		if v, ok := m["port"]; ok {
			if p := intValue(v); p != 0 {
				entry["port"] = p
			}
		}
		if v, ok := m["proxy_protocol"]; ok {
			entry["proxyProtocol"] = boolValue(v)
		}
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func flattenMtprotoDomainFronting(list []any) []any {
	out := make([]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{}
		if v, ok := m["ip"].(string); ok {
			entry["ip"] = v
		}
		if v, ok := m["port"]; ok {
			entry["port"] = intValue(v)
		}
		if v, ok := m["proxyProtocol"].(bool); ok {
			entry["proxy_protocol"] = v
		}
		out = append(out, entry)
	}
	return out
}
