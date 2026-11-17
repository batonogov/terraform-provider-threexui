package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func newTestClientWithToken(t *testing.T, endpoint, token string) *Client {
	t.Helper()
	c, err := NewClient(ClientConfig{
		Endpoint:               endpoint,
		BasePath:               "/",
		ApiToken:               token,
		InsecureSkipVerify:     true,
		Timeout:                2 * time.Second,
		MaxRetries:             0,
		RetryBackoff:           1 * time.Millisecond,
		ReadAfterWriteAttempts: 3,
		ReadAfterWriteBackoff:  1 * time.Millisecond,
		VersionRetryAttempts:   2,
		VersionRetryBackoff:    1 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	return c
}

// TestClientLoginWithAPIToken verifies the bearer path: Login only probes the
// API (no /login POST, no CSRF), and every subsequent request carries the token.
func TestClientLoginWithAPIToken(t *testing.T) {
	var statusAuth, listAuth, loginPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panel/api/server/status":
			statusAuth = r.Header.Get("Authorization")
			w.Write(okResponse(nil))
		case "/panel/api/inbounds/list":
			listAuth = r.Header.Get("Authorization")
			w.Write(okResponse([]any{}))
		default:
			loginPath = r.URL.Path
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClientWithToken(t, srv.URL, "test-token")
	if err := client.Login(context.Background()); err != nil {
		t.Fatalf("Login with token failed: %v", err)
	}
	if statusAuth != "Bearer test-token" {
		t.Fatalf("status probe missing bearer header: %q", statusAuth)
	}
	if _, err := client.GetInbounds(context.Background()); err != nil {
		t.Fatalf("GetInbounds failed: %v", err)
	}
	if listAuth != "Bearer test-token" {
		t.Fatalf("API request missing bearer header: %q", listAuth)
	}
	if loginPath != "" {
		t.Fatalf("token auth must not call session login, got %s", loginPath)
	}
}

func TestClientLoginWithAPITokenRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := newTestClientWithToken(t, srv.URL, "bad-token")
	err := client.Login(context.Background())
	if err == nil || !strings.Contains(err.Error(), "API token authentication failed") {
		t.Fatalf("expected API token failure, got %v", err)
	}
}

func TestClientGetClientRecord(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write(okResponse(map[string]any{
			"client": map[string]any{"email": "a@b", "limitHwid": 3},
			"externalLinks": []map[string]any{
				{"kind": "subscription", "value": "https://example.com/sub", "enable": true, "expiryTime": 42, "remark": "r", "namePrefix": "p"},
			},
		}))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	rec, err := client.GetClientRecord(context.Background(), "a@b")
	if err != nil {
		t.Fatalf("GetClientRecord failed: %v", err)
	}
	if gotPath != "/panel/api/clients/get/a@b" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if rec == nil || rec.Client.LimitHwid != 3 {
		t.Fatalf("expected limitHwid 3, got %+v", rec)
	}
	if len(rec.ExternalLinks) != 1 || rec.ExternalLinks[0].Kind != "subscription" || rec.ExternalLinks[0].ExpiryTime != 42 {
		t.Fatalf("unexpected external links: %+v", rec.ExternalLinks)
	}
}

func TestClientGetClientRecordNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":false,"msg":"record not found","obj":null}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	rec, err := client.GetClientRecord(context.Background(), "gone@b")
	if err != nil {
		t.Fatalf("missing record must not error: %v", err)
	}
	if rec != nil {
		t.Fatalf("expected nil record, got %+v", rec)
	}
}

func TestClientSetClientExternalLinks(t *testing.T) {
	var gotPath string
	var gotBody map[string]json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Write(okResponse(nil))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	enable := false
	links := []ClientExternalLink{
		{Kind: "subscription", Value: "https://example.com/sub", Enable: &enable, ExpiryTime: 7},
		{Kind: "link", Value: "vless://uuid@host:443#name"},
	}
	if err := client.SetClientExternalLinks(context.Background(), "a@b", links); err != nil {
		t.Fatalf("SetClientExternalLinks failed: %v", err)
	}
	if gotPath != "/panel/api/clients/a@b/externalLinks" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	var sent []ClientExternalLink
	if err := json.Unmarshal(gotBody["externalLinks"], &sent); err != nil {
		t.Fatalf("decode sent links: %v", err)
	}
	if len(sent) != 2 || sent[0].Enable == nil || *sent[0].Enable {
		t.Fatalf("expected explicit enable=false, got %+v", sent)
	}
}

// TestClientSetClientExternalLinksEmptyClears pins the replace-all semantics:
// an empty slice is a clear, not an omit.
func TestClientSetClientExternalLinksEmptyClears(t *testing.T) {
	var sent []ClientExternalLink
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&body)
		json.Unmarshal(body["externalLinks"], &sent)
		w.Write(okResponse(nil))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.SetClientExternalLinks(context.Background(), "a@b", nil); err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if sent == nil || len(sent) != 0 {
		t.Fatalf("expected explicit empty array, got %+v", sent)
	}
}

func TestInboundClientLimitHwidExpand(t *testing.T) {
	m := &InboundClientResourceModel{LimitHwid: types.Int64Value(5)}
	client := expandInboundClientFromModel(m)
	if got, ok := client["limitHwid"].(int64); !ok || got != 5 {
		t.Fatalf("expected limitHwid 5 in payload, got %v", client["limitHwid"])
	}

	// Unset attribute must not send a key at all: the panel's create payload
	// decodes limitHwid with a custom unmarshaler, and 0 means "disabled".
	unset := expandInboundClientFromModel(&InboundClientResourceModel{})
	if _, ok := unset["limitHwid"]; ok {
		t.Fatal("unset limit_hwid must be omitted from the payload")
	}
}

func TestInboundClientExternalLinkExpandFlatten(t *testing.T) {
	enable := false
	rows := []ClientExternalLink{
		{Kind: "subscription", Value: "https://example.com/sub", Remark: "r", Enable: &enable, ExpiryTime: 42, NamePrefix: "p"},
		{Kind: "", Value: "vless://uuid@host:443#name"},
	}
	list := flattenClientExternalLinks(rows)
	if got := len(list.Elements()); got != 2 {
		t.Fatalf("expected 2 elements, got %d", got)
	}

	back := expandClientExternalLinks(list)
	if len(back) != 2 {
		t.Fatalf("expected 2 links after round-trip, got %d", len(back))
	}
	if back[0].Kind != "subscription" || back[0].Value != "https://example.com/sub" ||
		back[0].Remark != "r" || back[0].ExpiryTime != 42 || back[0].NamePrefix != "p" ||
		back[0].Enable == nil || *back[0].Enable {
		t.Fatalf("subscription link lost fields: %+v", back[0])
	}
	// Panel defaults: empty kind means "link", absent enable means true.
	if back[1].Kind != "link" || back[1].Enable == nil || !*back[1].Enable {
		t.Fatalf("defaults not applied: %+v", back[1])
	}
}

func TestInboundClientRecordExtrasMerge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/clients/get/a@b" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(okResponse(map[string]any{
			"client": map[string]any{"email": "a@b", "limitHwid": 3},
			"externalLinks": []map[string]any{
				{"kind": "link", "value": "vless://uuid@host:443#name"},
			},
		}))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	newAPI := true
	client.newClientAPI = &newAPI
	r := &InboundClientResource{client: client}
	m := inboundClientToModel(1, "uuid", map[string]any{"email": "a@b"})

	if err := r.readClientRecordExtras(context.Background(), m, "a@b"); err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if m.LimitHwid.ValueInt64() != 3 {
		t.Fatalf("expected limit_hwid 3 after merge, got %d", m.LimitHwid.ValueInt64())
	}
	if len(m.ExternalLink.Elements()) != 1 {
		t.Fatalf("expected 1 external link after merge, got %d", len(m.ExternalLink.Elements()))
	}
}

// TestInboundClientRecordExtrasLegacyAPI pins the pre-v3.1.0 fallback: without
// the dedicated client API there is no record read, and the settings-only
// defaults stay untouched.
func TestInboundClientRecordExtrasLegacyAPI(t *testing.T) {
	client := newTestClient(t, "http://unused.invalid")
	setLegacyClientAPI(client)
	r := &InboundClientResource{client: client}
	m := inboundClientToModel(1, "uuid", map[string]any{"email": "a@b"})

	if err := r.readClientRecordExtras(context.Background(), m, "a@b"); err != nil {
		t.Fatalf("legacy merge must be a no-op, got %v", err)
	}
	if m.LimitHwid.ValueInt64() != 0 || len(m.ExternalLink.Elements()) != 0 {
		t.Fatalf("legacy defaults changed: %d / %d links", m.LimitHwid.ValueInt64(), len(m.ExternalLink.Elements()))
	}
}

func TestClientGetClientRecordValidation(t *testing.T) {
	client := newTestClient(t, "http://unused.invalid")
	if _, err := client.GetClientRecord(context.Background(), ""); err == nil {
		t.Fatal("empty email must be rejected")
	}
}

func TestClientGetClientRecordHTTPNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	rec, err := client.GetClientRecord(context.Background(), "a@b")
	if err != nil || rec != nil {
		t.Fatalf("404 must map to (nil, nil), got %v / %v", rec, err)
	}
}

func TestClientGetClientRecordError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":false,"msg":"database locked","obj":null}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if _, err := client.GetClientRecord(context.Background(), "a@b"); err == nil || !strings.Contains(err.Error(), "database locked") {
		t.Fatalf("expected panel error to propagate, got %v", err)
	}
}

func TestClientSetClientExternalLinksValidation(t *testing.T) {
	client := newTestClient(t, "http://unused.invalid")
	if err := client.SetClientExternalLinks(context.Background(), "", nil); err == nil {
		t.Fatal("empty email must be rejected")
	}
}

// TestClientAPITokenProbeErrorBranches walks the probe's failure modes so a
// broken token never passes configuration silently.
func TestClientAPITokenProbeErrorBranches(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		want    string
	}{
		{"http 500", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) }, "status 500"},
		{"envelope failure", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"success":false,"msg":"token scope too narrow"}`))
		}, "token scope too narrow"},
		{"envelope failure without message", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"success":false}`)) }, "API token request failed"},
		{"non-JSON body", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("not json")) }, "invalid character"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()
			client := newTestClientWithToken(t, srv.URL, "tok")
			err := client.Login(context.Background())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q in error, got %v", tc.want, err)
			}
		})
	}
}

func TestClientAPITokenProbeNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	endpoint := srv.URL
	srv.Close() // connection refused on every request

	client := newTestClientWithToken(t, endpoint, "tok")
	if err := client.Login(context.Background()); err == nil {
		t.Fatal("unreachable panel must fail the probe")
	}
}

func TestInboundClientRecordExtrasRecordGone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":false,"msg":"record not found","obj":null}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	newAPI := true
	client.newClientAPI = &newAPI
	r := &InboundClientResource{client: client}
	m := inboundClientToModel(1, "uuid", map[string]any{"email": "a@b"})

	if err := r.readClientRecordExtras(context.Background(), m, "a@b"); err != nil {
		t.Fatalf("missing record must keep defaults, got %v", err)
	}
	if m.LimitHwid.ValueInt64() != 0 || len(m.ExternalLink.Elements()) != 0 {
		t.Fatal("defaults changed for a missing record")
	}
}

// TestExpandClientExternalLinksSkipsNonObjects covers the defensive branch for
// malformed list elements; the framework only produces objects here.
func TestExpandClientExternalLinksSkipsNonObjects(t *testing.T) {
	list := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("not-an-object")})
	if got := expandClientExternalLinks(list); len(got) != 0 {
		t.Fatalf("non-object elements must be skipped, got %+v", got)
	}
}

func TestInboundClientRecordExtrasError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":false,"msg":"database locked","obj":null}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	newAPI := true
	client.newClientAPI = &newAPI
	r := &InboundClientResource{client: client}
	m := inboundClientToModel(1, "uuid", map[string]any{"email": "a@b"})

	if err := r.readClientRecordExtras(context.Background(), m, "a@b"); err == nil {
		t.Fatal("record read errors must propagate, not silently drop to defaults")
	}
}
