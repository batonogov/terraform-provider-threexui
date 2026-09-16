package provider

import (
	"encoding/json"
	"strings"
)

func applyDefaultInboundSettings(inbound *Inbound) error {
	if inbound == nil {
		return nil
	}

	raw := strings.TrimSpace(inbound.Settings)
	if raw == "" {
		return setDefaultSettings(inbound)
	}

	settings, err := ParseJSONField(raw)
	if err != nil {
		return err
	}
	if len(settings) == 0 {
		return setDefaultSettings(inbound)
	}

	updated := false
	if inbound.Protocol == "vless" {
		if _, ok := settings["decryption"]; !ok {
			settings["decryption"] = "none"
			updated = true
		}
	}
	if protocolUsesClients(inbound.Protocol) {
		if _, ok := settings["clients"]; !ok {
			settings["clients"] = []any{}
			updated = true
		}
	}

	if !updated {
		return nil
	}

	encoded, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	inbound.Settings = string(encoded)
	return nil
}

func setDefaultSettings(inbound *Inbound) error {
	settings, err := defaultSettingsForProtocol(inbound.Protocol)
	if err != nil {
		return err
	}
	if protocolUsesClients(inbound.Protocol) {
		if settings == nil {
			settings = map[string]any{"clients": []any{}}
		} else if _, ok := settings["clients"]; !ok {
			settings["clients"] = []any{}
		}
	}
	if settings == nil {
		inbound.Settings = "{}"
		return nil
	}
	encoded, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	inbound.Settings = string(encoded)
	return nil
}

func protocolUsesClients(protocol string) bool {
	switch protocol {
	case "vmess", "vless", "trojan", "shadowsocks", "hysteria":
		return true
	default:
		return false
	}
}

func defaultSettingsForProtocol(protocol string) (map[string]any, error) { //nolint:unparam // error kept for future protocol support
	switch protocol {
	case "vless":
		return map[string]any{
			"decryption": "none",
			"encryption": "none",
			"testseed":   []any{900, 500, 900, 256},
		}, nil
	case "vmess":
		return nil, nil
	case "trojan":
		return nil, nil
	case "shadowsocks":
		return nil, nil
	case "hysteria", "hysteria2": // hysteria2 kept for reading v3.1.0 inbounds
		return map[string]any{
			"version": 2,
		}, nil
	default:
		return nil, nil
	}
}

// protocolOwnsClients reports whether settings.clients[] for this protocol is
// managed by threexui_inbound itself rather than by the separate
// threexui_inbound_client resource. WireGuard (v3.4.2+) and AmneziaWG (v3.7.0+)
// peers live in the inbound's own settings block; every other protocol's
// clients are owned by threexui_inbound_client.
func protocolOwnsClients(protocol string) bool {
	return protocol == "wireguard" || protocol == "amneziawg" || protocol == "tuic"
}
