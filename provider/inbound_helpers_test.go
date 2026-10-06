package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPreserveInboundSettings_Nil(t *testing.T) {
	err := preserveInboundSettings(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPreserveInboundSettings_NilExisting(t *testing.T) {
	desired := &Inbound{Settings: `{"decryption":"none"}`}
	err := preserveInboundSettings(desired, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPreserveInboundSettings_ClientsPreserved(t *testing.T) {
	existing := &Inbound{Settings: `{"clients":[{"id":"abc","email":"test@test.com"}],"testseed":[1,2,3]}`}
	desired := &Inbound{Settings: `{"decryption":"none"}`}
	err := preserveInboundSettings(desired, existing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(desired.Settings), &result); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	clients, ok := result["clients"].([]any)
	if !ok || len(clients) != 1 {
		t.Fatalf("expected clients preserved, got %v", result["clients"])
	}
	testseed, ok := result["testseed"].([]any)
	if !ok || len(testseed) != 3 {
		t.Fatalf("expected testseed preserved, got %v", result["testseed"])
	}
}

func TestPreserveInboundSettings_ClientsNotOverwritten(t *testing.T) {
	existing := &Inbound{Settings: `{"clients":[{"id":"old"}]}`}
	desired := &Inbound{Settings: `{"clients":[{"id":"new"}]}`}
	err := preserveInboundSettings(desired, existing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(desired.Settings), &result); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	clients := result["clients"].([]any)
	first := clients[0].(map[string]any)
	if first["id"] != "new" {
		t.Fatalf("expected desired clients kept, got %v", first["id"])
	}
}

func TestPreserveInboundSettings_InvalidJSON(t *testing.T) {
	desired := &Inbound{Settings: "{bad"}
	existing := &Inbound{Settings: `{"clients":[]}`}
	err := preserveInboundSettings(desired, existing)
	if err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
}

func TestPreserveInboundSettings_EmptySettings(t *testing.T) {
	desired := &Inbound{Settings: ""}
	existing := &Inbound{Settings: `{"clients":[]}`}
	err := preserveInboundSettings(desired, existing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPreserveSettingsKey_KeyPresent(t *testing.T) {
	desired := map[string]any{}
	existing := map[string]any{"clients": []any{map[string]any{"id": "abc"}}}
	ok := preserveSettingsKey(desired, existing, "clients")
	if !ok {
		t.Fatalf("expected true")
	}
	if _, exists := desired["clients"]; !exists {
		t.Fatalf("expected clients to be set")
	}
}

func TestPreserveSettingsKey_KeyAbsentInExisting(t *testing.T) {
	desired := map[string]any{}
	existing := map[string]any{}
	ok := preserveSettingsKey(desired, existing, "clients")
	if ok {
		t.Fatalf("expected false when key absent")
	}
}

func TestPreserveSettingsKey_DesiredAlreadyHasKey(t *testing.T) {
	desired := map[string]any{"clients": []any{map[string]any{"id": "new"}}}
	existing := map[string]any{"clients": []any{map[string]any{"id": "old"}}}
	ok := preserveSettingsKey(desired, existing, "clients")
	if ok {
		t.Fatalf("expected false when desired already has non-empty key")
	}
}

func TestPreserveSettingsKey_NilExisting(t *testing.T) {
	desired := map[string]any{}
	ok := preserveSettingsKey(desired, nil, "clients")
	if ok {
		t.Fatalf("expected false for nil existing")
	}
}

func TestPreserveSettingsKey_EmptyListInExisting(t *testing.T) {
	desired := map[string]any{}
	existing := map[string]any{"clients": []any{}}
	ok := preserveSettingsKey(desired, existing, "clients")
	if ok {
		t.Fatalf("expected false for empty list")
	}
}

func TestEnsureRealityDefaults_Nil(t *testing.T) {
	ensureRealityDefaults(nil)
}

func TestEnsureRealityDefaults_HasServerNames(t *testing.T) {
	reality := map[string]any{
		"serverNames": []any{"custom.com"},
		"target":      "other.com:443",
	}
	ensureRealityDefaults(reality)
	names := reality["serverNames"].([]any)
	if names[0] != "custom.com" {
		t.Fatalf("expected custom.com preserved")
	}
}

func TestEnsureRealityDefaults_DeriveFromTarget(t *testing.T) {
	reality := map[string]any{
		"target": "myhost.com:443",
	}
	ensureRealityDefaults(reality)
	names := reality["serverNames"].([]any)
	if names[0] != "myhost.com" {
		t.Fatalf("expected myhost.com, got %v", names[0])
	}
}

func TestEnsureRealityDefaults_NoTargetNoNames(t *testing.T) {
	reality := map[string]any{}
	ensureRealityDefaults(reality)
	if reality["target"] != "www.amazon.com:443" {
		t.Fatalf("expected default target")
	}
	names := reality["serverNames"].([]any)
	if len(names) != 2 {
		t.Fatalf("expected 2 default server names")
	}
}

func TestHasRealityShortIDs_Nil(t *testing.T) {
	if hasRealityShortIDs(nil) {
		t.Fatalf("expected false for nil")
	}
}

func TestHasRealityShortIDs_Present(t *testing.T) {
	reality := map[string]any{"shortIds": []any{"abc"}}
	if !hasRealityShortIDs(reality) {
		t.Fatalf("expected true")
	}
}

func TestHasRealityShortIDs_Empty(t *testing.T) {
	reality := map[string]any{"shortIds": []any{}}
	if hasRealityShortIDs(reality) {
		t.Fatalf("expected false for empty list")
	}
}

func TestHasRealityShortIDs_MissingKey(t *testing.T) {
	reality := map[string]any{"other": "val"}
	if hasRealityShortIDs(reality) {
		t.Fatalf("expected false for missing key")
	}
}

func TestHasRealityServerNames_Nil(t *testing.T) {
	if hasRealityServerNames(nil) {
		t.Fatalf("expected false for nil")
	}
}

func TestHasRealityServerNames_Present(t *testing.T) {
	reality := map[string]any{"serverNames": []any{"example.com"}}
	if !hasRealityServerNames(reality) {
		t.Fatalf("expected true")
	}
}

func TestHasRealityServerNames_Empty(t *testing.T) {
	reality := map[string]any{"serverNames": []any{}}
	if hasRealityServerNames(reality) {
		t.Fatalf("expected false for empty")
	}
}

func TestHasStringListValues_Nil(t *testing.T) {
	if hasStringListValues(nil) {
		t.Fatalf("expected false for nil")
	}
}

func TestHasStringListValues_NonEmpty(t *testing.T) {
	if !hasStringListValues([]any{"abc"}) {
		t.Fatalf("expected true")
	}
}

func TestHasStringListValues_EmptyStrings(t *testing.T) {
	if hasStringListValues([]any{""}) {
		t.Fatalf("expected false for empty strings")
	}
}

func TestHasStringListValues_WrongType(t *testing.T) {
	if hasStringListValues("not a list") {
		t.Fatalf("expected false for wrong type")
	}
}

func TestHasStringListValues_StringSlice(t *testing.T) {
	if !hasStringListValues([]string{"abc"}) {
		t.Fatalf("expected true for []string")
	}
}

func TestHasStringListValues_EmptyStringSlice(t *testing.T) {
	if hasStringListValues([]string{""}) {
		t.Fatalf("expected false for empty []string")
	}
}

func TestRandomHex_Zero(t *testing.T) {
	result := randomHex(0)
	if result != "" {
		t.Fatalf("expected empty, got %q", result)
	}
}

func TestRandomHex_Eight(t *testing.T) {
	result := randomHex(8)
	if len(result) != 8 {
		t.Fatalf("expected length 8, got %d", len(result))
	}
	matched, _ := regexp.MatchString("^[0-9a-f]+$", result)
	if !matched {
		t.Fatalf("expected hex chars, got %q", result)
	}
}

func TestRandomHex_Odd(t *testing.T) {
	result := randomHex(5)
	if len(result) != 5 {
		t.Fatalf("expected length 5, got %d", len(result))
	}
	matched, _ := regexp.MatchString("^[0-9a-f]+$", result)
	if !matched {
		t.Fatalf("expected hex chars, got %q", result)
	}
}

func TestRandomHex_Unique(t *testing.T) {
	a := randomHex(16)
	b := randomHex(16)
	if a == b {
		t.Fatalf("expected different values")
	}
}

func TestIsSubset_MapSubset(t *testing.T) {
	desired := map[string]any{"a": float64(1)}
	actual := map[string]any{"a": float64(1), "b": float64(2)}
	if !isSubset(desired, actual) {
		t.Fatalf("expected subset")
	}
}

func TestIsSubset_MapNotSubset(t *testing.T) {
	desired := map[string]any{"a": float64(1), "c": float64(3)}
	actual := map[string]any{"a": float64(1), "b": float64(2)}
	if isSubset(desired, actual) {
		t.Fatalf("expected not subset")
	}
}

func TestIsSubset_ArraySubset(t *testing.T) {
	desired := []any{float64(1)}
	actual := []any{float64(1), float64(2)}
	if !isSubset(desired, actual) {
		t.Fatalf("expected subset")
	}
}

func TestIsSubset_EmptyArraySubset(t *testing.T) {
	desired := []any{}
	actual := []any{float64(1)}
	if !isSubset(desired, actual) {
		t.Fatalf("expected empty array is subset")
	}
}

func TestIsSubset_Scalars(t *testing.T) {
	if !isSubset(float64(1), float64(1)) {
		t.Fatalf("expected equal scalars")
	}
	if isSubset(float64(1), float64(2)) {
		t.Fatalf("expected not equal")
	}
}

// TestExpandInboundFromModel_v331Fields covers the v3.3.1 subscription-
// sharing fields (sub_sort_index, share_addr, share_addr_strategy) flowing
// from the Terraform model into the API Inbound struct.
func TestExpandInboundFromModel_v331Fields(t *testing.T) {
	m := &InboundResourceModel{
		SubSortIndex:      types.Int64Value(3),
		ShareAddr:         types.StringValue("1.2.3.4"),
		ShareAddrStrategy: types.StringValue("custom"),
	}
	inb := expandInboundFromModel(m)
	if inb.SubSortIndex != 3 {
		t.Fatalf("SubSortIndex: %d", inb.SubSortIndex)
	}
	if inb.ShareAddr != "1.2.3.4" {
		t.Fatalf("ShareAddr: %q", inb.ShareAddr)
	}
	if inb.ShareAddrStrategy != "custom" {
		t.Fatalf("ShareAddrStrategy: %q", inb.ShareAddrStrategy)
	}
}

// TestInboundToModel_v331Fields covers the reverse direction: API Inbound →
// Terraform model for the v3.3.1 fields. Uses failHard=false so empty settings
// produce only warnings and the basic fields (incl. the v3.3.1 ones) are
// still populated.
func TestInboundToModel_v331Fields(t *testing.T) {
	inb := &Inbound{
		SubSortIndex:      7,
		ShareAddr:         "node.example.com",
		ShareAddrStrategy: "node",
	}
	m, diags := inboundToModel(inb, false)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags)
	}
	if m.SubSortIndex.ValueInt64() != 7 {
		t.Fatalf("SubSortIndex: %d", m.SubSortIndex.ValueInt64())
	}
	if m.ShareAddr.ValueString() != "node.example.com" {
		t.Fatalf("ShareAddr: %q", m.ShareAddr.ValueString())
	}
	if m.ShareAddrStrategy.ValueString() != "node" {
		t.Fatalf("ShareAddrStrategy: %q", m.ShareAddrStrategy.ValueString())
	}
}

// TestInboundDisableFlowExpandFlatten covers the v3.7.0 `disableFlow` field,
// which opts an inbound out of the panel's automatic XTLS Vision assignment.
// A pre-v3.7.0 panel omits the key entirely, which must read back as false
// rather than as an error.
func TestInboundDisableFlowExpandFlatten(t *testing.T) {
	inb := expandInboundFromModel(&InboundResourceModel{
		DisableFlow: types.BoolValue(true),
	})
	if !inb.DisableFlow {
		t.Fatalf("DisableFlow: %v", inb.DisableFlow)
	}

	m, diags := inboundToModel(&Inbound{DisableFlow: true}, false)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags)
	}
	if !m.DisableFlow.ValueBool() {
		t.Fatalf("DisableFlow: %v", m.DisableFlow.ValueBool())
	}

	// Unconfigured attribute (null) must not force a value onto the payload.
	if inb := expandInboundFromModel(&InboundResourceModel{DisableFlow: types.BoolNull()}); inb.DisableFlow {
		t.Fatalf("null DisableFlow must expand to false, got %v", inb.DisableFlow)
	}
	old, diags := inboundToModel(&Inbound{}, false)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags)
	}
	if old.DisableFlow.ValueBool() {
		t.Fatalf("pre-v3.7.0 panel must read back false, got %v", old.DisableFlow.ValueBool())
	}
}

// TestInboundExcludeFromSubExpandFlatten covers the v3.9.0 `excludeFromSub`
// field, which keeps an inbound operational while omitting it from
// subscription output. A pre-v3.9.0 panel omits the key entirely, which must
// read back as false rather than as an error.
func TestInboundExcludeFromSubExpandFlatten(t *testing.T) {
	inb := expandInboundFromModel(&InboundResourceModel{
		ExcludeFromSub: types.BoolValue(true),
	})
	if !inb.ExcludeFromSub {
		t.Fatalf("ExcludeFromSub: %v", inb.ExcludeFromSub)
	}

	m, diags := inboundToModel(&Inbound{ExcludeFromSub: true}, false)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags)
	}
	if !m.ExcludeFromSub.ValueBool() {
		t.Fatalf("ExcludeFromSub: %v", m.ExcludeFromSub.ValueBool())
	}

	// Unconfigured attribute (null) must not force a value onto the payload.
	if inb := expandInboundFromModel(&InboundResourceModel{ExcludeFromSub: types.BoolNull()}); inb.ExcludeFromSub {
		t.Fatalf("null ExcludeFromSub must expand to false, got %v", inb.ExcludeFromSub)
	}
	old, diags := inboundToModel(&Inbound{}, false)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags)
	}
	if old.ExcludeFromSub.ValueBool() {
		t.Fatalf("pre-v3.9.0 panel must read back false, got %v", old.ExcludeFromSub.ValueBool())
	}
}

// TestInboundToFormExcludeFromSub pins the form key the panel binds
// (form:"excludeFromSub" on model.Inbound, form-bindable on add and update).
func TestInboundToFormExcludeFromSub(t *testing.T) {
	form := inboundToForm(&Inbound{ExcludeFromSub: true})
	if got := form.Get("excludeFromSub"); got != "true" {
		t.Fatalf("excludeFromSub form value: %q", got)
	}
	form = inboundToForm(&Inbound{})
	if got := form.Get("excludeFromSub"); got != "false" {
		t.Fatalf("excludeFromSub form value: %q", got)
	}
}

func TestEnsureInboundEnableNoopWhenMatching(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Write(okResponse(nil))
	}))
	defer srv.Close()

	r := &InboundResource{client: newTestClient(t, srv.URL)}
	pushed := false
	if err := r.ensureInboundEnable(context.Background(), 7, true, true, &pushed); err != nil {
		t.Fatalf("ensureInboundEnable: %v", err)
	}
	if pushed {
		t.Fatalf("pushed must stay false when observed already matches")
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Fatalf("expected no HTTP call when observed matches, got %d", got)
	}
}

func TestEnsureInboundEnablePushesOnce(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/inbounds/setEnable/7" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&calls, 1)
		w.Write(okResponse(nil))
	}))
	defer srv.Close()

	r := &InboundResource{client: newTestClient(t, srv.URL)}
	pushed := false
	if err := r.ensureInboundEnable(context.Background(), 7, false, true, &pushed); err != nil {
		t.Fatalf("ensureInboundEnable: %v", err)
	}
	if !pushed {
		t.Fatalf("pushed must be set after the reconcile call")
	}
	// A second poll that still observes the pre-flip value must not re-issue
	// the call — the flip is committed and only its visibility is pending.
	if err := r.ensureInboundEnable(context.Background(), 7, false, true, &pushed); err != nil {
		t.Fatalf("ensureInboundEnable (second poll): %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly 1 setEnable call across polls, got %d", got)
	}
}

func TestEnsureInboundEnableSurfacesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(failResponse("port conflict"))
	}))
	defer srv.Close()

	r := &InboundResource{client: newTestClient(t, srv.URL)}
	pushed := false
	if err := r.ensureInboundEnable(context.Background(), 7, true, false, &pushed); err == nil {
		t.Fatalf("expected the setEnable failure to surface")
	}
	if pushed {
		t.Fatalf("pushed must stay false when the call fails")
	}
}
