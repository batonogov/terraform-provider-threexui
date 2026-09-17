package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// -------------------------------------------------------------------------
// Contract JSON — committed to repo, used as fallback when snapshots absent
// -------------------------------------------------------------------------

// upstreamContract mirrors the structure of testdata/upstream_contract.json.
type upstreamContract struct {
	Version           string   `json:"version"`
	ProtocolsGoModel  []string `json:"protocols_go_model"`
	ProtocolsJS       []string `json:"protocols_js"`
	ProtocolForms     []string `json:"protocol_forms"`
	StreamForms       []string `json:"stream_forms"`
	InboundFields     []string `json:"inbound_fields"`
	ClientFields      []string `json:"client_fields"`
	AllSettingFields  []string `json:"all_setting_fields"`
	XraySettingsPages []string `json:"xray_settings_pages"`
}

// contractPath returns the absolute path to testdata/upstream_contract.json
// relative to the current test file (provider/ directory).
func contractPath(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	return filepath.Join(root, "provider", "testdata", "upstream_contract.json")
}

func loadContract(t *testing.T) *upstreamContract {
	t.Helper()
	data, err := os.ReadFile(contractPath(t))
	if err != nil {
		t.Fatalf("cannot read upstream_contract.json: %v", err)
	}
	var c upstreamContract
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("cannot parse upstream_contract.json: %v", err)
	}
	return &c
}

// -------------------------------------------------------------------------
// Repository helpers
// -------------------------------------------------------------------------

// repoRoot returns the absolute path of the repository root.
// It walks up from the current working directory looking for go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get cwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot find repo root (go.mod not found)")
		}
		dir = parent
	}
}

// -------------------------------------------------------------------------
// Snapshot directory helpers (semver-aware sorting)
// -------------------------------------------------------------------------

// parseSemver extracts numeric components from a version string like "2.9.0".
// Returns the parts as a slice of ints. Non-numeric segments are treated as 0.
func parseSemver(version string) []int {
	parts := strings.Split(version, ".")
	result := make([]int, len(parts))
	for i, p := range parts {
		n, _ := strconv.Atoi(p)
		result[i] = n
	}
	return result
}

// compareSemver returns -1, 0, or 1 comparing two semver slices.
func compareSemver(a, b []int) int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	for i := 0; i < maxLen; i++ {
		va, vb := 0, 0
		if i < len(a) {
			va = a[i]
		}
		if i < len(b) {
			vb = b[i]
		}
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
	}
	return 0
}

// findSnapshotDirs returns 3x-ui-* directories found in the given root,
// sorted by semver (ascending).
func findSnapshotDirs(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "3x-ui-") {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Slice(dirs, func(i, j int) bool {
		vi := parseSemver(strings.TrimPrefix(dirs[i], "3x-ui-"))
		vj := parseSemver(strings.TrimPrefix(dirs[j], "3x-ui-"))
		return compareSemver(vi, vj) < 0
	})
	return dirs
}

// upstreamGoPath resolves a Go-package source path inside a 3x-ui snapshot.
// 3x-ui v3.3.1 moved the Go packages (config, database, web, xray, ...) under
// internal/. Try the internal/ layout first, then fall back to the legacy root
// layout so older snapshots keep working.
func upstreamGoPath(dir string, parts ...string) string {
	rel := filepath.Join(parts...)
	if p := filepath.Join(dir, "internal", rel); pathExists(p) {
		return p
	}
	return filepath.Join(dir, rel)
}

// pathExists reports whether a file or directory exists at p.
func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// latestSnapshotDir returns the path to the latest 3x-ui-<version> directory.
// It checks the THREEXUI_SNAPSHOT_DIR env var first, then the repo root.
// Returns "" if no snapshots are found.
func latestSnapshotDir(t *testing.T) string {
	t.Helper()

	// Allow explicit override via environment variable.
	if envDir := os.Getenv("THREEXUI_SNAPSHOT_DIR"); envDir != "" {
		if info, err := os.Stat(envDir); err == nil && info.IsDir() {
			return envDir
		}
	}

	root := repoRoot(t)
	dirs := findSnapshotDirs(root)
	if len(dirs) > 0 {
		return filepath.Join(root, dirs[len(dirs)-1])
	}
	return ""
}

// -------------------------------------------------------------------------
// Auto-extraction of provider-supported protocols
// -------------------------------------------------------------------------

// providerProtocolsFromSwitch extracts protocol names from the switch-case
// in expandSettingsFromModel (inbound_settings_schema.go). This avoids
// hardcoding protocol lists in tests.
func providerProtocolsFromSwitch(t *testing.T) map[string]bool {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "provider", "inbound_settings_schema.go"))
	if err != nil {
		t.Fatalf("cannot read inbound_settings_schema.go: %v", err)
	}

	content := string(data)
	// Find the expandSettingsFromModel function body.
	fnStart := strings.Index(content, "func expandSettingsFromModel(")
	if fnStart == -1 {
		t.Fatal("cannot find expandSettingsFromModel in inbound_settings_schema.go")
	}

	// Extract from fnStart to end of function (find matching brace).
	braceStart := strings.Index(content[fnStart:], "{")
	if braceStart == -1 {
		t.Fatal("cannot find opening brace of expandSettingsFromModel")
	}
	pos := fnStart + braceStart + 1
	depth := 1
	for pos < len(content) && depth > 0 {
		switch content[pos] {
		case '{':
			depth++
		case '}':
			depth--
		}
		pos++
	}
	fnBody := content[fnStart:pos]

	// Extract case "xxx": and case "xxx", "yyy": patterns.
	re := regexp.MustCompile(`case\s+"([^"]+)"(?:\s*,\s*"([^"]+)")?`)
	matches := re.FindAllStringSubmatch(fnBody, -1)
	if len(matches) == 0 {
		t.Fatal("no case clauses found in expandSettingsFromModel")
	}

	protocols := make(map[string]bool)
	for _, m := range matches {
		protocols[m[1]] = true
		if m[2] != "" {
			protocols[m[2]] = true
		}
	}
	return protocols
}

// providerSettingsBlockProtocols extracts protocol names from the keys of
// inboundSettingsBlockSchemas() — e.g. "vless_settings" → "vless".
func providerSettingsBlockProtocols() map[string]bool {
	blocks := inboundSettingsBlockSchemas()
	result := make(map[string]bool, len(blocks))
	for key := range blocks {
		proto := strings.TrimSuffix(key, "_settings")
		result[proto] = true
	}
	return result
}

// -------------------------------------------------------------------------
// Auto-extraction of provider struct JSON tags via reflection
// -------------------------------------------------------------------------

// providerInboundJSONTags returns json tags from the provider's Inbound struct.
func providerInboundJSONTags() map[string]bool {
	return extractReflectJSONTags(reflect.TypeOf(Inbound{}))
}

// extractReflectJSONTags extracts json tag names from a struct type using reflection.
func extractReflectJSONTags(t reflect.Type) map[string]bool {
	tags := make(map[string]bool)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		// Take only the name part (before comma).
		name := strings.SplitN(tag, ",", 2)[0]
		if name != "" && name != "-" {
			tags[name] = true
		}
	}
	return tags
}

// -------------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------------

// extractStructJSONTags extracts json tag values from a Go struct definition
// in source code. It finds "type <name> struct {" and collects all json:"xxx"
// tags until the closing "}".
func extractStructJSONTags(t *testing.T, source, structName string) []string {
	t.Helper()

	// Find struct start.
	structRe := regexp.MustCompile(`type\s+` + regexp.QuoteMeta(structName) + `\s+struct\s*\{`)
	loc := structRe.FindStringIndex(source)
	if loc == nil {
		t.Fatalf("struct %s not found in source", structName)
	}

	// Find matching closing brace.
	depth := 1
	pos := loc[1]
	for pos < len(source) && depth > 0 {
		switch source[pos] {
		case '{':
			depth++
		case '}':
			depth--
		}
		pos++
	}
	block := source[loc[1]:pos]

	// Extract json tags.
	jsonRe := regexp.MustCompile(`json:"([^",]+)`)
	matches := jsonRe.FindAllStringSubmatch(block, -1)

	var tags []string
	for _, m := range matches {
		tags = append(tags, m[1])
	}
	return tags
}

// toSortedSlice returns a sorted slice of keys from a bool map.
func toSortedSlice(m map[string]bool) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	sort.Strings(s)
	return s
}

// toSet converts a string slice to a bool map.
func toSet(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}

func extractInboundJSProtocols(t *testing.T, jsPath string) []string {
	t.Helper()
	data, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatalf("cannot read inbound model JS: %v", err)
	}

	content := string(data)
	start := strings.Index(content, "const Protocols = {")
	if start == -1 {
		start = strings.Index(content, "export const Protocols = {")
	}
	if start == -1 {
		start = strings.Index(content, "export const Protocols = Object.freeze({")
	}
	if start == -1 {
		t.Fatal("cannot find Protocols object in inbound model JS")
	}
	// Object.freeze({...}); ends with });, plain objects end with };
	sep := "};"
	if strings.HasPrefix(content[start:], "export const Protocols = Object.freeze(") {
		sep = "})"
	}
	end := strings.Index(content[start:], sep)
	if end == -1 {
		t.Fatal("cannot find end of Protocols object in inbound model JS")
	}
	block := content[start : start+end]

	re := regexp.MustCompile(`\w+:\s*['"]([^'"]+)['"]`)
	matches := re.FindAllStringSubmatch(block, -1)
	if len(matches) == 0 {
		t.Fatal("no protocols found in Protocols object")
	}

	seen := make(map[string]bool)
	for _, m := range matches {
		seen[m[1]] = true
	}
	return toSortedSlice(seen)
}

func upstreamInboundJSPath(dir string) string {
	// v3.2.0+: TypeScript migration moved the Protocols definition.
	ts := filepath.Join(dir, "frontend", "src", "schemas", "primitives", "protocol.ts")
	if _, err := os.Stat(ts); err == nil {
		return ts
	}
	legacy := filepath.Join(dir, "web", "assets", "js", "model", "inbound.js")
	if _, err := os.Stat(legacy); err == nil {
		return legacy
	}
	return filepath.Join(dir, "frontend", "src", "models", "inbound.js")
}

func upstreamProtocolForms(t *testing.T, dir string) []string {
	t.Helper()
	formDir := upstreamGoPath(dir, "web", "html", "form", "protocol")
	if entries, err := os.ReadDir(formDir); err == nil {
		var upstream []string
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
				continue
			}
			upstream = append(upstream, strings.TrimSuffix(e.Name(), ".html"))
		}
		sort.Strings(upstream)
		return upstream
	}

	// v3.2.0+: TypeScript schemas in frontend/src/schemas/protocols/inbound/
	tsDir := filepath.Join(dir, "frontend", "src", "schemas", "protocols", "inbound")
	if entries, err := os.ReadDir(tsDir); err == nil {
		var upstream []string
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".ts") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".ts")
			if name == "index" {
				continue
			}
			upstream = append(upstream, name)
		}
		sort.Strings(upstream)
		return upstream
	}

	return extractInboundJSProtocols(t, upstreamInboundJSPath(dir))
}

func upstreamStreamForms(t *testing.T, dir string) []string {
	t.Helper()
	streamDir := upstreamGoPath(dir, "web", "html", "form", "stream")
	if entries, err := os.ReadDir(streamDir); err == nil {
		var upstream []string
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
				continue
			}
			upstream = append(upstream, strings.TrimSuffix(e.Name(), ".html"))
		}
		sort.Strings(upstream)
		return upstream
	}

	// v3.2.0+: TypeScript schemas in frontend/src/schemas/protocols/stream/
	tsDir := filepath.Join(dir, "frontend", "src", "schemas", "protocols", "stream")
	if entries, err := os.ReadDir(tsDir); err == nil {
		seen := map[string]bool{"stream_settings": true}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".ts") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".ts")
			if name == "index" {
				continue
			}
			seen["stream_"+strings.ReplaceAll(name, "-", "_")] = true
		}
		return toSortedSlice(seen)
	}

	// Legacy: extract from inbound.js StreamSettings class.
	jsPath := filepath.Join(dir, "frontend", "src", "models", "inbound.js")
	if _, err := os.Stat(jsPath); err != nil {
		jsPath = filepath.Join(dir, "web", "assets", "js", "model", "inbound.js")
	}
	if _, err := os.Stat(jsPath); err != nil {
		t.Fatalf("no stream form source found in %s (neither TS schemas nor legacy inbound.js)", dir)
	}
	data, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatalf("cannot read inbound model JS: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "class StreamSettings") {
		t.Fatal("cannot find StreamSettings class in inbound model JS")
	}

	seen := map[string]bool{"stream_settings": true}
	re := regexp.MustCompile(`\w+Settings:\s*network\s*===\s*['"]([^'"]+)['"]`)
	for _, m := range re.FindAllStringSubmatch(content, -1) {
		seen["stream_"+m[1]] = true
	}
	if strings.Contains(content, "externalProxy") {
		seen["external_proxy"] = true
	}
	if strings.Contains(content, "finalmask") {
		seen["stream_finalmask"] = true
	}
	if strings.Contains(content, "sockopt") {
		seen["stream_sockopt"] = true
	}
	return toSortedSlice(seen)
}

func upstreamXraySettingsPages(t *testing.T, dir string) []string {
	t.Helper()
	xrayDir := upstreamGoPath(dir, "web", "html", "settings", "xray")
	if entries, err := os.ReadDir(xrayDir); err == nil {
		var upstream []string
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
				continue
			}
			upstream = append(upstream, strings.TrimSuffix(e.Name(), ".html"))
		}
		sort.Strings(upstream)
		return upstream
	}

	vuePath := filepath.Join(dir, "frontend", "src", "pages", "xray", "XrayPage.vue")
	tsxPath := filepath.Join(dir, "frontend", "src", "pages", "xray", "XrayPage.tsx")
	pagePath := vuePath
	data, err := os.ReadFile(pagePath)
	if err != nil {
		// v3.1.0+ migrated from Vue to React (TSX)
		pagePath = tsxPath
		data, err = os.ReadFile(pagePath)
		if err != nil {
			t.Fatalf("cannot read XrayPage.vue or XrayPage.tsx: %v", err)
		}
	}
	content := string(data)
	componentPages := map[string]string{
		"BasicsTab":    "basics",
		"RoutingTab":   "routing",
		"OutboundsTab": "outbounds",
		"BalancersTab": "balancers",
		"DnsTab":       "dns",
	}
	seen := make(map[string]bool)
	for component, page := range componentPages {
		if strings.Contains(content, component) {
			seen[page] = true
		}
	}
	if strings.Contains(content, "tpl-advanced") || strings.Contains(content, "advancedText") {
		seen["advanced"] = true
	}
	return toSortedSlice(seen)
}

// checkMissing reports upstream items not found in provider set.
func checkMissing(t *testing.T, upstream []string, provider map[string]bool, skip map[string]bool, msgFmt string) {
	t.Helper()
	var missing []string
	for _, item := range upstream {
		if skip[item] || provider[item] {
			continue
		}
		missing = append(missing, item)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf(msgFmt, missing)
	}
}

// checkRemoved reports provider items not present in upstream (reverse check).
func checkRemoved(t *testing.T, provider map[string]bool, upstream map[string]bool, skip map[string]bool, msgFmt string) {
	t.Helper()
	var removed []string
	for item := range provider {
		if skip[item] || upstream[item] {
			continue
		}
		removed = append(removed, item)
	}
	if len(removed) > 0 {
		sort.Strings(removed)
		t.Fatalf(msgFmt, removed)
	}
}

// -------------------------------------------------------------------------
// Test: upstream inbound protocols vs provider protocol mappings (Go model)
// -------------------------------------------------------------------------

func TestDriftInboundProtocols_GoModel(t *testing.T) {
	// Auto-extract provider protocols from switch-case.
	providerHandled := providerProtocolsFromSwitch(t)

	// Additional protocols the provider handles but not via explicit switch case.
	// "vmess" and "mixed" have no per-protocol settings block (handled via default).
	// "socks" is available via UI. "tun" is UI alias for tunnel/dokodemo-door.
	// "dokodemo-door" is the xray-level name; upstream model.go only exposes "tunnel".
	// "hysteria2" removed from upstream in v3.2.0; switch-case kept for reading v3.1.0 data.
	providerExtras := map[string]bool{
		"vmess":         true,
		"mixed":         true,
		"socks":         true,
		"tun":           true,
		"dokodemo-door": true,
		"hysteria2":     true,
	}
	for k := range providerExtras {
		providerHandled[k] = true
	}

	dir := latestSnapshotDir(t)
	var upstream []string

	if dir != "" {
		// Parse upstream model.go for protocol constants.
		modelPath := upstreamGoPath(dir, "database", "model", "model.go")
		data, err := os.ReadFile(modelPath)
		if err != nil {
			t.Fatalf("cannot read model.go: %v", err)
		}

		// Restrict regex to const/var blocks to avoid matching unrelated strings.
		re := regexp.MustCompile(`(?:const|var)\s+\w*[Pp]rotocol\w*\s*=\s*"([^"]+)"`)
		matches := re.FindAllStringSubmatch(string(data), -1)
		if len(matches) == 0 {
			// Fallback: look inside const blocks for Protocol = "xxx".
			blockRe := regexp.MustCompile(`(?s)const\s*\(([^)]+)\)`)
			blocks := blockRe.FindAllStringSubmatch(string(data), -1)
			valRe := regexp.MustCompile(`\b\w*Protocol\w*\s*=\s*"([^"]+)"`)
			for _, block := range blocks {
				bm := valRe.FindAllStringSubmatch(block[1], -1)
				matches = append(matches, bm...)
			}
		}
		if len(matches) == 0 {
			t.Fatal("no protocol constants found in model.go")
		}

		seen := make(map[string]bool)
		for _, m := range matches {
			seen[m[1]] = true
		}
		upstream = toSortedSlice(seen)
	} else {
		// CI fallback: use committed contract.
		c := loadContract(t)
		upstream = c.ProtocolsGoModel
	}

	upstreamSet := toSet(upstream)
	// "tuic" (v3.8.0, TUIC v5) is terminated by a bundled tuic-server sidecar
	// rather than xray-core; provider support is tracked separately.
	upstreamSkipped := map[string]bool{"tuic": true}
	checkMissing(t, upstream, providerHandled, upstreamSkipped,
		"upstream model.go has protocols not handled by provider: %v")
	checkRemoved(t, providerHandled, upstreamSet, providerExtras,
		"provider handles protocols no longer in upstream model.go: %v")
}

// -------------------------------------------------------------------------
// Test: upstream inbound protocols vs provider (JS)
// -------------------------------------------------------------------------

func TestDriftInboundProtocols_JS(t *testing.T) {
	providerHandled := providerProtocolsFromSwitch(t)
	// "dokodemo-door" is the xray-level name; UI uses "tunnel".
	// "hysteria2" removed from upstream in v3.2.0; switch-case kept for reading v3.1.0 data.
	providerExtras := map[string]bool{
		"vmess":         true,
		"mixed":         true,
		"socks":         true,
		"tun":           true,
		"dokodemo-door": true,
		"hysteria2":     true,
	}
	for k := range providerExtras {
		providerHandled[k] = true
	}

	dir := latestSnapshotDir(t)
	var upstream []string

	if dir != "" {
		upstream = extractInboundJSProtocols(t, upstreamInboundJSPath(dir))
	} else {
		c := loadContract(t)
		upstream = c.ProtocolsJS
	}

	upstreamSet := toSet(upstream)
	// "tuic" (v3.8.0, TUIC v5 sidecar) — see TestDriftInboundProtocols_GoModel.
	upstreamSkipped := map[string]bool{"tuic": true}
	checkMissing(t, upstream, providerHandled, upstreamSkipped,
		"upstream inbound.js Protocols has entries not handled by provider: %v")
	checkRemoved(t, providerHandled, upstreamSet, providerExtras,
		"provider handles protocols not in upstream inbound.js: %v")
}

// -------------------------------------------------------------------------
// Test: upstream protocol HTML forms vs provider settings blocks
// -------------------------------------------------------------------------

func TestDriftProtocolForms(t *testing.T) {
	// Auto-extract from inboundSettingsBlockSchemas() keys.
	providerBlocks := providerSettingsBlockProtocols()
	// Additional mappings not derived from block names.
	providerExtras := map[string]bool{
		"vmess":    true, // vmess has no settings block but is handled
		"tun":      true, // alias for tunnel/dokodemo-door
		"tunnel":   true, // v3 UI name for the dokodemo settings block
		"mixed":    true, // mixed reuses the socks form in 3x-ui
		"socks":    true, // legacy UI protocol kept for existing configs
		"dokodemo": true, // legacy form name; v3 UI uses tunnel
	}
	for k := range providerExtras {
		providerBlocks[k] = true
	}

	dir := latestSnapshotDir(t)
	var upstream []string

	if dir != "" {
		upstream = upstreamProtocolForms(t, dir)
	} else {
		c := loadContract(t)
		upstream = c.ProtocolForms
	}

	upstreamSet := toSet(upstream)
	// "tuic" (v3.8.0, TUIC v5 sidecar) — see TestDriftInboundProtocols_GoModel.
	upstreamSkipped := map[string]bool{"tuic": true}
	checkMissing(t, upstream, providerBlocks, upstreamSkipped,
		"upstream protocol form files not handled by provider: %v")
	checkRemoved(t, providerBlocks, upstreamSet, providerExtras,
		"provider has settings blocks for protocols without upstream forms: %v")
}

// -------------------------------------------------------------------------
// Test: upstream stream/transport HTML forms vs provider stream_settings
// -------------------------------------------------------------------------

func TestDriftStreamSettingsForms(t *testing.T) {
	providerHandled := map[string]bool{
		"stream_tcp":            true,
		"stream_ws":             true,
		"stream_grpc":           true,
		"stream_httpupgrade":    true,
		"stream_xhttp":          true,
		"stream_kcp":            true,
		"stream_hysteria":       true,
		"stream_sockopt":        true,
		"stream_settings":       true, // main stream_settings container
		"stream_finalmask":      true, // finalmask is part of xhttp settings
		"stream_external_proxy": true, // TS schema name for external_proxy
	}

	dir := latestSnapshotDir(t)
	var upstream []string

	if dir != "" {
		upstream = upstreamStreamForms(t, dir)
	} else {
		c := loadContract(t)
		upstream = c.StreamForms
	}

	upstreamSet := toSet(upstream)
	checkMissing(t, upstream, providerHandled, nil,
		"upstream stream form files not handled by provider: %v")
	checkRemoved(t, providerHandled, upstreamSet, nil,
		"provider handles stream forms not in upstream: %v")
}

// -------------------------------------------------------------------------
// Test: upstream Inbound model fields vs provider Inbound struct
// -------------------------------------------------------------------------

func TestDriftInboundFields(t *testing.T) {
	// Auto-extract from provider's Inbound struct via reflection.
	providerFields := providerInboundJSONTags()

	// Fields intentionally skipped (internal DB fields, not part of API contract).
	skip := map[string]bool{
		"-":              true, // UserId uses json:"-"
		"fallbackParent": true, // v3.1.0 frontend-only, not persisted
		"originNodeGuid": true, // multi-hop node attribution, not managed by provider
	}

	dir := latestSnapshotDir(t)
	var upstream []string

	if dir != "" {
		modelPath := upstreamGoPath(dir, "database", "model", "model.go")
		data, err := os.ReadFile(modelPath)
		if err != nil {
			t.Fatalf("cannot read model.go: %v", err)
		}
		upstream = extractStructJSONTags(t, string(data), "Inbound")
	} else {
		c := loadContract(t)
		upstream = c.InboundFields
	}

	upstreamSet := toSet(upstream)
	checkMissing(t, upstream, providerFields, skip,
		"upstream Inbound struct has json fields not in provider: %v")
	// `allTime` is provider-only within the supported range. Upstream added it to
	// model.Inbound and xray.ClientTraffic in v2.6.7 (PR #3387) and removed it in
	// v3.1.0 (PR #4469), which also drops the `all_time` columns at startup — the
	// drop migration is still present in every snapshot here
	// (3x-ui-3.7.0/internal/web/service/inbound_migration.go:55-63). No panel this
	// provider supports (v3.2.x+) sends the field, so threexui_inbound.all_time
	// always reads 0. The attribute now carries a DeprecationMessage; dropping it
	// is a breaking change held for the next major release (#442).
	checkRemoved(t, providerFields, upstreamSet, map[string]bool{"allTime": true},
		"provider Inbound struct has json fields not in upstream: %v")
}

// -------------------------------------------------------------------------
// Test: upstream Client model fields vs provider awareness
// -------------------------------------------------------------------------

func TestDriftClientFields(t *testing.T) {
	// Client fields known to provider (from resource_inbound_client.go expand/flatten).
	// These map tfsdk attributes to upstream JSON keys — must stay hardcoded because
	// the provider uses different naming than upstream (e.g. tfsdk:"limit_ip" → "limitIp").
	providerKnown := map[string]bool{
		"id": true, "security": true, "password": true, "flow": true,
		"auth": true, "email": true, "limitIp": true, "totalGB": true,
		"expiryTime": true, "enable": true, "tgId": true, "subId": true,
		"comment": true, "reset": true, "created_at": true, "updated_at": true,
		"reverse": true, "group": true,
		// v3.5.0 MTProto multi-client (mtg-multi): per-client FakeTLS secret + ad-tag.
		"secret": true, "adTag": true,
		// v3.7.0 calendar-day renewals + per-client traffic reset cycle.
		"resetDay": true, "resetMax": true,
		"trafficReset": true, "trafficResetDay": true,
	}

	dir := latestSnapshotDir(t)
	var upstream []string

	if dir != "" {
		modelPath := upstreamGoPath(dir, "database", "model", "model.go")
		data, err := os.ReadFile(modelPath)
		if err != nil {
			t.Fatalf("cannot read model.go: %v", err)
		}
		upstream = extractStructJSONTags(t, string(data), "Client")
	} else {
		c := loadContract(t)
		upstream = c.ClientFields
	}

	upstreamSet := toSet(upstream)
	// The Client struct gained 5 WireGuard multi-client fields in v3.4.2
	// (privateKey/publicKey/allowedIPs/preSharedKey/keepAlive). They are not
	// exposed on the generic threexui_inbound_client resource (which RMWs the
	// shared settings.clients[]); instead they are surfaced on the WireGuard
	// inbound via wireguard_settings.peer[] and wireguard_settings.clients[]
	// (InboundWireguardPeerModel / InboundWireguardClientModel). Kept here so
	// the generic-client drift gate stays green.
	clientIntentionallySkipped := map[string]bool{
		"privateKey": true, "publicKey": true, "allowedIPs": true,
		"preSharedKey": true, "keepAlive": true,
		// v3.7.0 AmneziaWG surface. forwardedPorts is the per-client DNAT spec;
		// it IS implemented, but on amneziawg_settings.clients[] rather than on
		// the generic threexui_inbound_client resource this gate covers (#441).
		// allowedIPsByInbound is deliberately unimplemented: the panel accepts it
		// only on the /panel/api/clients CRUD endpoints, never echoes it back on
		// a GET (the read side is the separate tunnelAllowedIPs projection), so
		// it cannot round-trip through Terraform state.
		"forwardedPorts": true, "allowedIPsByInbound": true,
	}
	checkMissing(t, upstream, providerKnown, clientIntentionallySkipped,
		"upstream Client struct has json fields not known to provider: %v")
	checkRemoved(t, providerKnown, upstreamSet, clientIntentionallySkipped,
		"provider knows Client fields not in upstream: %v")
}

// -------------------------------------------------------------------------
// Test: upstream AllSetting fields vs provider settings tabs
// -------------------------------------------------------------------------

func TestDriftAllSettingFields(t *testing.T) {
	// All fields handled across panel_general, panel_security, panel_telegram,
	// panel_subscription resources. Collected from expand/flatten functions.
	// Must stay hardcoded because the provider uses different naming convention
	// (tfsdk snake_case → upstream camelCase JSON keys).
	providerKnown := map[string]bool{
		// General
		"webListen": true, "webDomain": true, "webPort": true,
		"webBasePath": true, "sessionMaxAge": true, "pageSize": true,
		"remarkModel": true, "datepicker": true, "timeLocation": true,
		"expireDiff": true, "trafficDiff": true, "webCertFile": true,
		"webKeyFile": true, "trustedProxyCIDRs": true, "externalTrafficInformEnable": true,
		"externalTrafficInformURI": true, "restartXrayOnClientDisable": true,
		// LDAP
		"ldapEnable": true, "ldapHost": true, "ldapPort": true,
		"ldapUseTLS": true, "ldapInsecureSkipVerify": true, "ldapBindDN": true, "ldapPassword": true,
		"ldapBaseDN": true, "ldapUserFilter": true, "ldapUserAttr": true,
		"ldapVlessField": true, "ldapSyncCron": true,
		"ldapFlagField": true, "ldapTruthyValues": true, "ldapInvertFlag": true,
		"ldapInboundTags": true, "ldapAutoCreate": true, "ldapAutoDelete": true,
		"ldapDefaultTotalGB": true, "ldapDefaultExpiryDays": true, "ldapDefaultLimitIP": true,
		// Security
		"twoFactorEnable": true, "twoFactorToken": true,
		// Telegram
		"tgBotEnable": true, "tgBotToken": true, "tgBotProxy": true,
		"tgBotAPIServer": true, "tgBotChatId": true, "tgLang": true,
		"tgRunTime": true, "tgBotBackup": true, "tgBotLoginNotify": true,
		"tgCpu": true,
		// v3.4.0 Telegram event/threshold additions
		"tgEnabledEvents": true, "tgMemory": true,
		// Subscription
		"subEnable": true, "subJsonEnable": true, "subTitle": true,
		"subSupportUrl": true, "subProfileUrl": true, "subAnnounce": true,
		"subEnableRouting": true, "subRoutingRules": true,
		"subListen": true, "subPort": true, "subPath": true,
		"subDomain": true, "subCertFile": true, "subKeyFile": true,
		"subUpdates": true, "subEncrypt": true, "subShowInfo": true,
		"subEmailInRemark": true, "subURI": true, "subJsonPath": true, "subJsonURI": true,
		"subJsonFragment": true, "subJsonNoises": true, "subJsonMux": true,
		"subJsonRules": true, "subClashEnable": true, "subClashPath": true,
		"subClashURI": true, "subClashEnableRouting": true, "subClashRules": true,
		"subJsonFinalMask": true,
		// v3.3.0/v3.4.0 subscription additions
		"subThemeDir": true, "remarkTemplate": true, "subHideSettings": true,
		// v3.4.1 subscription additions (Incy client routing injection)
		"subIncyEnableRouting": true, "subIncyRoutingRules": true,
		// v3.2.0
		"panelProxy":    true,
		"panelOutbound": true,
		// v3.3.0 panel_general
		"warpUpdateInterval": true,
		// v3.4.0 SMTP/email notifications (threexui_panel_email)
		"smtpEnable": true, "smtpHost": true, "smtpPort": true,
		"smtpUsername": true, "smtpPassword": true, "smtpTo": true,
		"smtpEncryptionType": true, "smtpEnabledEvents": true,
		"smtpCpu": true, "smtpMemory": true,
		// v3.8.0 Discord notification bot (threexui_panel_discord)
		"discordBotEnable": true, "discordBotToken": true,
		"discordChannelId": true, "discordAdminIds": true,
		"discordRunTime": true, "discordBotBackup": true,
		"discordCpu": true, "discordMemory": true,
		"discordLang": true, "discordEnabledEvents": true,
		// v3.6.0 SMTP From header (threexui_panel_email)
		"smtpFrom": true, "smtpFromName": true,
		// v3.6.0 subscription format auto-detection (threexui_panel_subscription)
		"subJsonAutoDetect": true, "subJsonAlwaysArray": true,
		"subJsonUserAgentRegex": true, "subClashAutoDetect": true,
		"subClashUserAgentRegex": true,
		// v3.6.0 panel_general
		"subShowIdentityOnAllLinks": true, "outboundDownThreshold": true,
		// v3.7.0 IP-limit allowlist (threexui_panel_general) and JSON-subscription
		// observatory blob for client-side balancers (threexui_panel_subscription)
		"ipLimitAllowlist": true, "subJsonObservatory": true,
		// v3.8.0 subscription page / JSON-subscription / Happ additions
		// (threexui_panel_subscription) and REALITY scan candidates
		// (threexui_panel_general)
		"subProfileMode": true, "subInfoNodeEnable": true,
		"subCalendarExpireInclusive": true, "subExpiredTemplate": true,
		"subTrafficDepletedTemplate": true,
		"subJsonRoutingRules":        true, "subJsonDns": true,
		"happLinkEnable": true, "realityScanCandidates": true,
		"subHappAutoDetect": true, "subHappProviderId": true,
		"subHappNewUrl": true, "subHappFallbackUrl": true,
		"subHappSubInfoColor": true, "subHappSubInfoText": true,
		"subHappSubInfoButtonText": true, "subHappSubInfoButtonLink": true,
		"subHappSubExpire": true, "subHappSubExpireButtonLink": true,
		"subHappNotificationExpire": true, "subHappNoLimit": true,
		"subHappAlwaysHwid": true, "subHappTunMode": true,
		"subHappTunType": true, "subHappExcludeRoutes": true,
		"subHappExcludeApns": true, "subHappColorProfile": true,
		"subHappPingType": true, "subHappAutoConnect": true,
		"subHappAutoConnectType": true, "subHappPerAppMode": true,
		"subHappPerAppList": true,
	}

	// Fields intentionally not managed by the provider.
	intentionallySkipped := map[string]bool{
		// Removed in 3x-ui v3.2.8 — kept in provider for backward compat.
		"subJsonFragment": true,
		"subJsonNoises":   true,

		// Removed in 3x-ui v3.4.0 — kept in provider for backward compat
		// with v3.1.x–v3.3.x panels (gin silently ignores unknown form
		// values on v3.4.0). remarkModel was superseded by remarkTemplate.
		"remarkModel":      true,
		"subEmailInRemark": true,
		"subShowInfo":      true,
		"tgBotLoginNotify": true,
		// v3.3.1 renamed panelProxy → panelOutbound (outbound egress bridge).
		// The provider keeps panelProxy for backward compat with v3.2.0–v3.3.0
		// panels even though v3.3.1's AllSetting struct dropped it.
		"panelProxy": true,
	}

	dir := latestSnapshotDir(t)
	var upstream []string

	if dir != "" {
		entityPath := upstreamGoPath(dir, "web", "entity", "entity.go")
		data, err := os.ReadFile(entityPath)
		if err != nil {
			t.Fatalf("cannot read entity.go: %v", err)
		}
		upstream = extractStructJSONTags(t, string(data), "AllSetting")
	} else {
		c := loadContract(t)
		upstream = c.AllSettingFields
	}

	upstreamSet := toSet(upstream)
	checkMissing(t, upstream, providerKnown, intentionallySkipped,
		"upstream AllSetting struct has json fields not handled by provider settings: %v\n"+
			"Add them to the appropriate provider settings resource or to intentionallySkipped in this test.")
	checkRemoved(t, providerKnown, upstreamSet, intentionallySkipped,
		"provider settings handle fields not in upstream AllSetting: %v")
}

// -------------------------------------------------------------------------
// Test: upstream xray settings HTML pages vs provider xray resources
// -------------------------------------------------------------------------

func TestDriftXraySettingsPages(t *testing.T) {
	providerResources := map[string]bool{
		"basics":    true, // threexui_xray_basics
		"dns":       true, // threexui_xray_dns
		"routing":   true, // threexui_xray_routing
		"outbounds": true, // threexui_xray_outbounds
		"balancers": true, // threexui_xray_balancers
		"reverse":   true, // threexui_xray_reverse
		"advanced":  true, // advanced is raw JSON editing, no dedicated resource needed
	}

	// Intentionally skipped: "advanced" is not a real resource, just raw JSON.
	providerExtras := map[string]bool{
		"advanced": true,
		"reverse":  true, // legacy threexui_xray_reverse; upstream UI page was removed in v2.9.4
	}

	dir := latestSnapshotDir(t)
	var upstream []string

	if dir != "" {
		upstream = upstreamXraySettingsPages(t, dir)
	} else {
		c := loadContract(t)
		upstream = c.XraySettingsPages
	}

	upstreamSet := toSet(upstream)
	checkMissing(t, upstream, providerResources, nil,
		"upstream xray settings pages not handled by provider: %v")
	checkRemoved(t, providerResources, upstreamSet, providerExtras,
		"provider has xray resources without upstream settings pages: %v")
}

// -------------------------------------------------------------------------
// Test: semver sorting
// -------------------------------------------------------------------------

func TestSemverSorting(t *testing.T) {
	dirs := []string{"3x-ui-2.10.0", "3x-ui-2.9.0", "3x-ui-2.8.9", "3x-ui-2.11.1"}
	sort.Slice(dirs, func(i, j int) bool {
		vi := parseSemver(strings.TrimPrefix(dirs[i], "3x-ui-"))
		vj := parseSemver(strings.TrimPrefix(dirs[j], "3x-ui-"))
		return compareSemver(vi, vj) < 0
	})
	expected := []string{"3x-ui-2.8.9", "3x-ui-2.9.0", "3x-ui-2.10.0", "3x-ui-2.11.1"}
	for i, d := range dirs {
		if d != expected[i] {
			t.Fatalf("semver sort failed: got %v, want %v", dirs, expected)
		}
	}
}
