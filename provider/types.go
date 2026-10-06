package provider

import (
	"encoding/json"
	"strings"
)

// Inbound represents a 3x-ui inbound (model.Inbound).
// Note: settings/streamSettings/sniffing are JSON strings internally.
// Custom UnmarshalJSON handles both the legacy format (escaped JSON strings)
// used by v2.9.x/v3.0.x and the modern format (nested JSON objects) used by v3.1.0+.
type Inbound struct {
	ID    int   `json:"id"`
	Up    int64 `json:"up"`
	Down  int64 `json:"down"`
	Total int64 `json:"total"`
	// AllTime is deprecated and always decodes to 0 on every supported panel.
	// Upstream carried `allTime` on model.Inbound from v2.6.7 (PR #3387) until
	// v3.1.0 removed it (PR #4469, which also drops the `all_time` columns on
	// startup), so the field is absent from every snapshot in the supported
	// v3.2.x+ range. Kept only to keep feeding the deprecated `all_time`
	// attribute until that attribute is dropped in the next major release (#442).
	AllTime              int64           `json:"allTime"`
	Remark               string          `json:"remark"`
	Enable               bool            `json:"enable"`
	ExpiryTime           int64           `json:"expiryTime"`
	TrafficReset         string          `json:"trafficReset"`
	TrafficResetDay      int             `json:"trafficResetDay"`
	LastTrafficResetTime int64           `json:"lastTrafficResetTime"`
	ClientStats          []ClientTraffic `json:"clientStats"`

	Listen            string `json:"listen"`
	Port              int    `json:"port"`
	Protocol          string `json:"protocol"`
	Settings          string `json:"settings"`
	StreamSettings    string `json:"streamSettings"`
	Tag               string `json:"tag"`
	SubSortIndex      int    `json:"subSortIndex,omitempty"`
	ShareAddr         string `json:"shareAddr,omitempty"`
	ShareAddrStrategy string `json:"shareAddrStrategy,omitempty"`
	Sniffing          string `json:"sniffing"`
	NodeID            *int   `json:"nodeId,omitempty"`
	DisableFlow       bool   `json:"disableFlow"`
	// ExcludeFromSub (v3.9.0+) keeps the inbound operational while omitting it
	// from subscription output. Older panels never send the key, so it decodes
	// to false there.
	ExcludeFromSub bool `json:"excludeFromSub"`
}

func (i *Inbound) UnmarshalJSON(data []byte) error {
	type Alias Inbound
	aux := &struct {
		Settings       json.RawMessage `json:"settings"`
		StreamSettings json.RawMessage `json:"streamSettings"`
		Sniffing       json.RawMessage `json:"sniffing"`
		*Alias
	}{
		Alias: (*Alias)(i),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	i.Settings = rawJSONToString(aux.Settings)
	i.StreamSettings = rawJSONToString(aux.StreamSettings)
	i.Sniffing = rawJSONToString(aux.Sniffing)
	return nil
}

// rawJSONToString normalises a JSON field that may be either a JSON string
// (legacy v2.9.x/v3.0.x: "settings":"{\"clients\":[]}") or a raw JSON
// object/array (v3.1.0+: "settings":{"clients":[]}) back to the plain
// string the provider uses internally.
func rawJSONToString(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	if len(trimmed) >= 2 && trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}
	}
	return trimmed
}

// ClientTraffic represents traffic statistics for a client (xray.ClientTraffic).
type ClientTraffic struct {
	ID         int    `json:"id"`
	InboundID  int    `json:"inboundId"`
	Enable     bool   `json:"enable"`
	Email      string `json:"email"`
	UUID       string `json:"uuid"`
	SubID      string `json:"subId"`
	Up         int64  `json:"up"`
	Down       int64  `json:"down"`
	ExpiryTime int64  `json:"expiryTime"`
	Total      int64  `json:"total"`
	Reset      int    `json:"reset"`
	LastOnline int64  `json:"lastOnline"`
}

// Node represents a 3x-ui cluster node (model.Node): a remote 3x-ui panel
// registered with the central panel for multi-node/cluster management.
//
// Managed fields (user-editable via the threexui_node resource, see M2) are the
// top block; the rest are observed state populated by the central panel's
// heartbeat probes. ApiToken and PinnedCertSha256 are sensitive. Since 3x-ui
// v3.6.0 (#5613) ApiToken is write-only — the API never returns it.
// PinnedCertSha256 is returned raw by the panel (no redaction layer), so callers
// must keep it marked Sensitive in Terraform schemas.
//
// Available since 3x-ui v3.0.2; /panel/api/nodes has no legacy fallback path.
type Node struct {
	// Managed (user-editable).
	Id                  int      `json:"id"`
	Name                string   `json:"name"`
	Remark              string   `json:"remark"`
	Scheme              string   `json:"scheme"`
	Address             string   `json:"address"`
	Port                int      `json:"port"`
	BasePath            string   `json:"basePath"`
	ApiToken            string   `json:"apiToken"`
	Enable              bool     `json:"enable"`
	AllowPrivateAddress bool     `json:"allowPrivateAddress"`
	TlsVerifyMode       string   `json:"tlsVerifyMode"`
	PinnedCertSha256    string   `json:"pinnedCertSha256"`
	InboundSyncMode     string   `json:"inboundSyncMode"`
	InboundTags         []string `json:"inboundTags"`
	OutboundTag         string   `json:"outboundTag"`

	// Observed identity / heartbeat state (read-only).
	Guid          string  `json:"guid"`
	Status        string  `json:"status"`
	LastHeartbeat int64   `json:"lastHeartbeat"`
	LatencyMs     int     `json:"latencyMs"`
	XrayVersion   string  `json:"xrayVersion"`
	PanelVersion  string  `json:"panelVersion"`
	CpuPct        float64 `json:"cpuPct"`
	MemPct        float64 `json:"memPct"`
	UptimeSecs    uint64  `json:"uptimeSecs"`
	NetUp         uint64  `json:"netUp"`
	NetDown       uint64  `json:"netDown"`
	LastError     string  `json:"lastError"`
	XrayState     string  `json:"xrayState"`
	XrayError     string  `json:"xrayError"`
	ConfigDirty   bool    `json:"configDirty"`
	ConfigDirtyAt int64   `json:"configDirtyAt"`
	InboundCount  int     `json:"inboundCount"`
	ClientCount   int     `json:"clientCount"`
	OnlineCount   int     `json:"onlineCount"`
	ActiveCount   int     `json:"activeCount"`
	DisabledCount int     `json:"disabledCount"`
	DepletedCount int     `json:"depletedCount"`
	ParentGuid    string  `json:"parentGuid,omitempty"`
	Transitive    bool    `json:"transitive,omitempty"`
	CreatedAt     int64   `json:"createdAt"`
	UpdatedAt     int64   `json:"updatedAt"`
}

// ParseJSONField decodes a JSON string into a generic map.
func ParseJSONField(value string) (map[string]any, error) {
	var out map[string]any
	if value == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return nil, err
	}
	return out, nil
}
