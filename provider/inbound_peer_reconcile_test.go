package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProtocolReconcilesPeersViaClientAPI(t *testing.T) {
	for _, tc := range []struct {
		protocol string
		want     bool
	}{
		{"wireguard", true},
		{"amneziawg", true},
		{"tuic", true},
		{"vless", false},
		{"vmess", false},
		{"trojan", false},
		{"shadowsocks", false},
		{"mtproto", false},
		{"", false},
	} {
		if got := protocolReconcilesPeersViaClientAPI(tc.protocol); got != tc.want {
			t.Errorf("protocolReconcilesPeersViaClientAPI(%q) = %v, want %v", tc.protocol, got, tc.want)
		}
	}
}

func TestSettingsClientsList(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings string
		want     int
	}{
		{"empty", "", 0},
		{"empty object", "{}", 0},
		{"no clients key", `{"server":{}}`, 0},
		{"invalid json", `{"clients":`, 0},
		{"clients not an array", `{"clients":{}}`, 0},
		{"two clients", `{"clients":[{"email":"a"},{"email":"b"}]}`, 2},
	} {
		if got := len(settingsClientsList(tc.settings)); got != tc.want {
			t.Errorf("%s: len(settingsClientsList(%q)) = %d, want %d", tc.name, tc.settings, got, tc.want)
		}
	}
}

func TestDiffInboundOwnedPeers(t *testing.T) {
	peer := func(email string, kv ...any) map[string]any {
		m := map[string]any{"email": email}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
		return m
	}

	for _, tc := range []struct {
		name       string
		plan       []any
		observed   []any
		wantAdd    []string // emails
		wantUpdate []string
		wantRemove []string
	}{
		{
			name: "both empty",
		},
		{
			name:     "identical with number type drift",
			plan:     []any{peer("a", "keepAlive", 25)},
			observed: []any{peer("a", "keepAlive", float64(25))},
		},
		{
			name:     "panel-only extra keys never trigger an update",
			plan:     []any{peer("a", "publicKey", "pk")},
			observed: []any{peer("a", "publicKey", "pk", "subId", "xyz", "created_at", float64(111), "enable", true)},
		},
		{
			name:     "updated_at alone is volatile",
			plan:     []any{peer("a", "publicKey", "pk", "updated_at", int64(1000))},
			observed: []any{peer("a", "publicKey", "pk", "updated_at", float64(2000))},
		},
		{
			name:     "blank plan subId carries no constraint",
			plan:     []any{peer("a", "publicKey", "pk", "subId", "")},
			observed: []any{peer("a", "publicKey", "pk", "subId", "panel-generated")},
		},
		{
			name:       "non-blank plan subId still compares",
			plan:       []any{peer("a", "publicKey", "pk", "subId", "mine")},
			observed:   []any{peer("a", "publicKey", "pk", "subId", "panel-generated")},
			wantUpdate: []string{"a"},
		},
		{
			name:     "new peer is added",
			plan:     []any{peer("a"), peer("b")},
			observed: []any{peer("a")},
			wantAdd:  []string{"b"},
		},
		{
			name:       "removed peer is deleted",
			plan:       []any{peer("a")},
			observed:   []any{peer("a"), peer("c")},
			wantRemove: []string{"c"},
		},
		{
			name:       "removal to zero",
			observed:   []any{peer("a"), peer("c")},
			wantRemove: []string{"a", "c"},
		},
		{
			name:       "changed field is updated",
			plan:       []any{peer("a", "comment", "new")},
			observed:   []any{peer("a", "comment", "old")},
			wantUpdate: []string{"a"},
		},
		{
			name:       "email change is remove plus add",
			plan:       []any{peer("b", "publicKey", "pk")},
			observed:   []any{peer("a", "publicKey", "pk")},
			wantAdd:    []string{"b"},
			wantRemove: []string{"a"},
		},
		{
			name:     "plan peer without email is skipped",
			plan:     []any{map[string]any{"publicKey": "pk"}},
			observed: []any{},
		},
		{
			name:     "non-map entries are skipped on both sides",
			plan:     []any{"junk", peer("a", "publicKey", "pk")},
			observed: []any{float64(42), peer("a", "publicKey", "pk")},
		},
		{
			name:     "observed peer without email is left alone",
			plan:     []any{},
			observed: []any{map[string]any{"publicKey": "pk"}},
		},
		{
			name:     "allowedIPs order does not matter",
			plan:     []any{peer("a", "allowedIPs", []any{"10.0.0.2/32", "10.0.0.3/32"})},
			observed: []any{peer("a", "allowedIPs", []any{"10.0.0.3/32", "10.0.0.2/32"})},
		},
		{
			name:     "duplicate observed emails take the first entry",
			plan:     []any{peer("a", "comment", "one")},
			observed: []any{peer("a", "comment", "one"), peer("a", "comment", "two")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diff := diffInboundOwnedPeers(tc.plan, tc.observed)
			gotAdd := make([]string, 0, len(diff.add))
			for _, m := range diff.add {
				gotAdd = append(gotAdd, m["email"].(string))
			}
			gotUpdate := make([]string, 0, len(diff.update))
			for _, m := range diff.update {
				gotUpdate = append(gotUpdate, m["email"].(string))
			}
			if !equalStringSlices(gotAdd, tc.wantAdd) {
				t.Errorf("add = %v, want %v", gotAdd, tc.wantAdd)
			}
			if !equalStringSlices(gotUpdate, tc.wantUpdate) {
				t.Errorf("update = %v, want %v", gotUpdate, tc.wantUpdate)
			}
			if !equalStringSlices(diff.remove, tc.wantRemove) {
				t.Errorf("remove = %v, want %v", diff.remove, tc.wantRemove)
			}
		})
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// peerReconcileServer is a stateful fake panel for the reconciliation tests.
// It keeps one inbound's settings.clients array and answers the inbound GET
// plus the shared client endpoints, mutating the array the way 3x-ui does:
// add appends, update replaces the entry matched by email, del removes it.
type peerReconcileServer struct {
	srv *httptest.Server

	protocol string
	clients  []map[string]any

	addCalls    atomic.Int32
	updateCalls atomic.Int32
	delCalls    atomic.Int32
	delNotFound atomic.Int32 // return a not-found failure for deleted emails
}

func newPeerReconcileServer(t *testing.T, protocol string, initial []map[string]any) *peerReconcileServer {
	t.Helper()
	s := &peerReconcileServer{protocol: protocol, clients: initial}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *peerReconcileServer) settingsJSON() string {
	out := map[string]any{"clients": s.clients}
	b, _ := json.Marshal(out)
	return string(b)
}

func (s *peerReconcileServer) inbound() map[string]any {
	return map[string]any{
		"id":             7,
		"remark":         "r",
		"enable":         true,
		"port":           25070,
		"protocol":       s.protocol,
		"settings":       s.settingsJSON(),
		"sniffing":       "{}",
		"streamSettings": "{}",
	}
}

func (s *peerReconcileServer) handle(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/panel/api/clients/list":
		// useNewClientAPI probe: the v3.1.0+ surface exists.
		_, _ = w.Write(okResponse([]any{}))
	case r.URL.Path == "/panel/api/inbounds/get/7":
		_, _ = w.Write(okResponse(s.inbound()))
	case r.URL.Path == "/panel/api/clients/add" && r.Method == http.MethodPost:
		var payload struct {
			Client map[string]any `json:"client"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		s.clients = append(s.clients, payload.Client)
		s.addCalls.Add(1)
		_, _ = w.Write(okResponse(nil))
	case strings.HasPrefix(r.URL.Path, "/panel/api/clients/update/") && r.Method == http.MethodPost:
		email := strings.TrimPrefix(r.URL.Path, "/panel/api/clients/update/")
		var client map[string]any
		_ = json.NewDecoder(r.Body).Decode(&client)
		for i, c := range s.clients {
			if c["email"] == email {
				s.clients[i] = client
				s.updateCalls.Add(1)
				_, _ = w.Write(okResponse(nil))
				return
			}
		}
		_, _ = w.Write(failResponse(fmt.Sprintf("client %q not found in any inbound or client record", email)))
	case strings.HasPrefix(r.URL.Path, "/panel/api/clients/del/") && r.Method == http.MethodPost:
		email := strings.TrimPrefix(r.URL.Path, "/panel/api/clients/del/")
		for i, c := range s.clients {
			if c["email"] == email {
				s.clients = append(s.clients[:i], s.clients[i+1:]...)
				s.delCalls.Add(1)
				_, _ = w.Write(okResponse(nil))
				return
			}
		}
		s.delNotFound.Add(1)
		_, _ = w.Write(failResponse(fmt.Sprintf("client %q not found in any inbound or client record", email)))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestReconcileInboundOwnedPeers(t *testing.T) {
	planSettings := `{"clients":[` +
		`{"email":"a@t.com","publicKey":"pkA","comment":"new","allowedIPs":["10.0.0.2/32"]},` +
		`{"email":"b@t.com","publicKey":"pkB","allowedIPs":["10.0.0.3/32"]}` +
		`]}`

	t.Run("add update and remove", func(t *testing.T) {
		srv := newPeerReconcileServer(t, "wireguard", []map[string]any{
			{"email": "a@t.com", "publicKey": "pkA", "comment": "old", "allowedIPs": []any{"10.0.0.2/32"}},
			{"email": "c@t.com", "publicKey": "pkC", "allowedIPs": []any{"10.0.0.4/32"}},
		})
		r := &InboundResource{client: newTestClient(t, srv.srv.URL)}
		got, err := r.reconcileInboundOwnedPeers(context.Background(), 7, "wireguard", planSettings, &Inbound{ID: 7, Settings: srv.settingsJSON()})
		if err != nil {
			t.Fatalf("reconcile: %v", err)
		}
		if got := srv.addCalls.Load(); got != 1 {
			t.Errorf("expected 1 add call, got %d", got)
		}
		if got := srv.updateCalls.Load(); got != 1 {
			t.Errorf("expected 1 update call, got %d", got)
		}
		if got := srv.delCalls.Load(); got != 1 {
			t.Errorf("expected 1 del call, got %d", got)
		}
		clients := settingsClientsList(got.Settings)
		if len(clients) != 2 {
			t.Fatalf("expected 2 peers after reconcile, got %d", len(clients))
		}
		if diff := diffInboundOwnedPeers(settingsClientsList(planSettings), clients); !diff.empty() {
			t.Fatalf("expected converged state, still have diff: %+v", diff)
		}
	})

	t.Run("no diff means no calls and the same inbound back", func(t *testing.T) {
		srv := newPeerReconcileServer(t, "wireguard", []map[string]any{
			{"email": "a@t.com", "publicKey": "pkA", "comment": "new", "allowedIPs": []any{"10.0.0.2/32"}, "subId": "panel"},
			{"email": "b@t.com", "publicKey": "pkB", "allowedIPs": []any{"10.0.0.3/32"}},
		})
		r := &InboundResource{client: newTestClient(t, srv.srv.URL)}
		observed := &Inbound{ID: 7, Settings: srv.settingsJSON()}
		got, err := r.reconcileInboundOwnedPeers(context.Background(), 7, "wireguard", planSettings, observed)
		if err != nil {
			t.Fatalf("reconcile: %v", err)
		}
		if got != observed {
			t.Errorf("expected the argument back when there is nothing to do")
		}
		if n := srv.addCalls.Load() + srv.updateCalls.Load() + srv.delCalls.Load(); n != 0 {
			t.Errorf("expected no client endpoint calls, got %d", n)
		}
	})

	t.Run("a stale first read does not fire spurious endpoint calls", func(t *testing.T) {
		// ≤ v3.8.5: the wholesale update already persisted the plan's peers,
		// but the first read after it can still serve the pre-update snapshot
		// (#157). The grace reads must let the committed write become visible
		// instead of pushing a spurious add the panel would reject with
		// "Duplicate email".
		var getCalls, addCalls, updateCalls, delCalls atomic.Int32
		stale := map[string]any{
			"id": 7, "remark": "r", "enable": true, "port": 25070, "protocol": "wireguard",
			"settings": `{"clients":[{"email":"a@t.com","publicKey":"pkA","comment":"old","allowedIPs":["10.0.0.2/32"]}]}`,
			"sniffing": "{}", "streamSettings": "{}",
		}
		fresh := map[string]any{
			"id": 7, "remark": "r", "enable": true, "port": 25070, "protocol": "wireguard",
			"settings": `{"clients":[{"email":"a@t.com","publicKey":"pkA","comment":"new","allowedIPs":["10.0.0.2/32"]},{"email":"b@t.com","publicKey":"pkB","allowedIPs":["10.0.0.3/32"]}]}`,
			"sniffing": "{}", "streamSettings": "{}",
		}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch {
			case req.URL.Path == "/panel/api/clients/list":
				_, _ = w.Write(okResponse([]any{}))
			case req.URL.Path == "/panel/api/inbounds/get/7":
				if getCalls.Add(1) == 1 {
					_, _ = w.Write(okResponse(stale))
				} else {
					_, _ = w.Write(okResponse(fresh))
				}
			case req.URL.Path == "/panel/api/clients/add":
				addCalls.Add(1)
				_, _ = w.Write(okResponse(nil))
			case strings.HasPrefix(req.URL.Path, "/panel/api/clients/update/"):
				updateCalls.Add(1)
				_, _ = w.Write(okResponse(nil))
			case strings.HasPrefix(req.URL.Path, "/panel/api/clients/del/"):
				delCalls.Add(1)
				_, _ = w.Write(okResponse(nil))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		r := &InboundResource{client: newTestClient(t, srv.URL)}
		got, err := r.reconcileInboundOwnedPeers(context.Background(), 7, "wireguard", planSettings, &Inbound{ID: 7, Settings: stale["settings"].(string)})
		if err != nil {
			t.Fatalf("reconcile: %v", err)
		}
		if n := addCalls.Load() + updateCalls.Load() + delCalls.Load(); n != 0 {
			t.Errorf("expected no client endpoint calls once the fresh read arrives, got %d", n)
		}
		if clients := settingsClientsList(got.Settings); len(clients) != 2 {
			t.Errorf("expected the fresh 2-peer read back, got %v", clients)
		}
	})

	t.Run("removal to zero", func(t *testing.T) {
		srv := newPeerReconcileServer(t, "amneziawg", []map[string]any{
			{"email": "a@t.com", "publicKey": "pkA"},
		})
		r := &InboundResource{client: newTestClient(t, srv.srv.URL)}
		got, err := r.reconcileInboundOwnedPeers(context.Background(), 7, "amneziawg", `{"server":{}}`, &Inbound{ID: 7, Settings: srv.settingsJSON()})
		if err != nil {
			t.Fatalf("reconcile: %v", err)
		}
		if got := srv.delCalls.Load(); got != 1 {
			t.Errorf("expected 1 del call, got %d", got)
		}
		if clients := settingsClientsList(got.Settings); len(clients) != 0 {
			t.Errorf("expected no peers left, got %v", clients)
		}
	})

	t.Run("protocols outside the gate pass through", func(t *testing.T) {
		srv := newPeerReconcileServer(t, "vless", nil)
		r := &InboundResource{client: newTestClient(t, srv.srv.URL)}
		observed := &Inbound{ID: 7, Settings: "{}"}
		got, err := r.reconcileInboundOwnedPeers(context.Background(), 7, "vless", planSettings, observed)
		if err != nil {
			t.Fatalf("reconcile: %v", err)
		}
		if got != observed {
			t.Errorf("expected the argument back for a non-reconciled protocol")
		}
		if n := srv.addCalls.Load() + srv.updateCalls.Load() + srv.delCalls.Load(); n != 0 {
			t.Errorf("expected no client endpoint calls, got %d", n)
		}
	})

	t.Run("endpoint failure surfaces", func(t *testing.T) {
		broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/panel/api/clients/list":
				_, _ = w.Write(okResponse([]any{}))
			case "/panel/api/inbounds/get/7":
				// Never gains the plan's peers, so the reconciliation fires.
				_, _ = w.Write(okResponse(map[string]any{
					"id": 7, "remark": "r", "enable": true, "port": 25070,
					"protocol": "wireguard", "settings": `{"clients":[{"email":"a@t.com","publicKey":"pkA","comment":"old"}]}`,
					"sniffing": "{}", "streamSettings": "{}",
				}))
			case "/panel/api/clients/add":
				_, _ = w.Write(failResponse("Duplicate email: a@t.com"))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer broken.Close()

		r := &InboundResource{client: newTestClient(t, broken.URL)}
		_, err := r.reconcileInboundOwnedPeers(context.Background(), 7, "wireguard", planSettings, &Inbound{ID: 7, Settings: "{}"})
		if err == nil || !strings.Contains(err.Error(), "adding peer") {
			t.Fatalf("expected an adding-peer error, got %v", err)
		}
	})

	t.Run("nil observed passes through", func(t *testing.T) {
		r := &InboundResource{}
		got, err := r.reconcileInboundOwnedPeers(context.Background(), 7, "wireguard", planSettings, nil)
		if err != nil || got != nil {
			t.Fatalf("expected (nil, nil), got (%v, %v)", got, err)
		}
	})

	// failReconcileServer serves the probe and a get that never converges,
	// and fails the endpoint named by failPrefix with failMsg.
	failReconcileServer := func(t *testing.T, storedSettings string, failPrefix, failMsg string) *httptest.Server {
		t.Helper()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch {
			case req.URL.Path == "/panel/api/clients/list":
				_, _ = w.Write(okResponse([]any{}))
			case strings.HasPrefix(req.URL.Path, failPrefix):
				_, _ = w.Write(failResponse(failMsg))
			case req.URL.Path == "/panel/api/inbounds/get/7":
				_, _ = w.Write(okResponse(map[string]any{
					"id": 7, "remark": "r", "enable": true, "port": 25070,
					"protocol": "wireguard", "settings": storedSettings,
					"sniffing": "{}", "streamSettings": "{}",
				}))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		t.Cleanup(srv.Close)
		return srv
	}

	t.Run("del failure surfaces", func(t *testing.T) {
		stored := `{"clients":[{"email":"a@t.com","publicKey":"pkA"}]}`
		srv := failReconcileServer(t, stored, "/panel/api/clients/del/", "cannot delete")
		r := &InboundResource{client: newTestClient(t, srv.URL)}
		_, err := r.reconcileInboundOwnedPeers(context.Background(), 7, "wireguard", `{"clients":[]}`, &Inbound{ID: 7, Settings: stored})
		if err == nil || !strings.Contains(err.Error(), `removing peer "a@t.com"`) {
			t.Fatalf("expected a removing-peer error, got %v", err)
		}
	})

	t.Run("update failure surfaces", func(t *testing.T) {
		stored := `{"clients":[{"email":"a@t.com","publicKey":"pkA","comment":"old"}]}`
		srv := failReconcileServer(t, stored, "/panel/api/clients/update/", "cannot update")
		r := &InboundResource{client: newTestClient(t, srv.URL)}
		plan := `{"clients":[{"email":"a@t.com","publicKey":"pkA","comment":"new"}]}`
		_, err := r.reconcileInboundOwnedPeers(context.Background(), 7, "wireguard", plan, &Inbound{ID: 7, Settings: stored})
		if err == nil || !strings.Contains(err.Error(), `updating peer "a@t.com"`) {
			t.Fatalf("expected an updating-peer error, got %v", err)
		}
	})

	t.Run("read failure exhausts the poll", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/panel/api/clients/list":
				_, _ = w.Write(okResponse([]any{}))
			default:
				http.Error(w, "boom", http.StatusInternalServerError)
			}
		}))
		defer srv.Close()

		r := &InboundResource{client: newTestClient(t, srv.URL)}
		_, err := r.reconcileInboundOwnedPeers(context.Background(), 7, "wireguard", planSettings, &Inbound{ID: 7, Settings: "{}"})
		if err == nil || !strings.Contains(err.Error(), "reconciled peers are not visible on the panel") {
			t.Fatalf("expected a poll-exhaustion error, got %v", err)
		}
	})
}

func awgPeerModel(email, comment string, updatedAt int64) InboundAmneziawgClientModel {
	return InboundAmneziawgClientModel{
		Email:        types.StringValue(email),
		PublicKey:    types.StringValue("pk-" + email),
		Comment:      types.StringValue(comment),
		AllowedIPs:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.9.1.2/32")}),
		CreatedAt:    types.Int64Value(500),
		UpdatedAt:    types.Int64Value(updatedAt),
		Enable:       types.BoolValue(true),
		TrafficReset: types.StringValue("never"),
	}
}

// tuicPeerModel builds a TUIC peer the way flattenTuicClientsToModel produces
// one: every optional field is a concrete zero value, never null.
func tuicPeerModel(email, comment, subID string, createdAt int64) InboundTuicClientModel {
	return InboundTuicClientModel{
		Email:     types.StringValue(email),
		ID:        types.StringValue("aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"),
		Password:  types.StringValue("pass-" + email),
		Enable:    types.BoolValue(true),
		Comment:   types.StringValue(comment),
		SubID:     types.StringValue(subID),
		CreatedAt: types.Int64Value(createdAt),
		UpdatedAt: types.Int64Value(1000),
		// Everything else is the flatten zero value.
		LimitIP:         types.Int64Value(0),
		TotalGB:         types.Int64Value(0),
		ExpiryTime:      types.Int64Value(0),
		TgID:            types.Int64Value(0),
		Reset:           types.Int64Value(0),
		ResetDay:        types.Int64Value(0),
		ResetMax:        types.Int64Value(0),
		TrafficReset:    types.StringValue(""),
		TrafficResetDay: types.Int64Value(0),
	}
}

func TestEditedOwnedPeerMarks(t *testing.T) {
	awg := func(peers ...InboundAmneziawgClientModel) *InboundResourceModel {
		return &InboundResourceModel{AmneziawgSettings: &InboundAmneziawgSettingsModel{Clients: peers}}
	}
	wgPeer := func(email, comment string) InboundWireguardClientModel {
		return InboundWireguardClientModel{
			Email:   types.StringValue(email),
			Comment: types.StringValue(comment),
		}
	}
	wg := func(peers ...InboundWireguardClientModel) *InboundResourceModel {
		return &InboundResourceModel{WireguardSettings: &InboundWireguardSettingsModel{Clients: peers}}
	}
	tuic := func(peers ...InboundTuicClientModel) *InboundResourceModel {
		return &InboundResourceModel{TuicSettings: &InboundTuicSettingsModel{Clients: peers}}
	}

	t.Run("non-owned protocol yields nothing", func(t *testing.T) {
		if got := editedOwnedPeerMarks("vless", awg(awgPeerModel("a", "x", 1)), awg(awgPeerModel("a", "y", 1))); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("unchanged peers yield nothing", func(t *testing.T) {
		if got := editedOwnedPeerMarks("amneziawg", awg(awgPeerModel("a", "x", 1000)), awg(awgPeerModel("a", "x", 1000))); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("new peer needs no marks", func(t *testing.T) {
		got := editedOwnedPeerMarks("amneziawg", awg(), awg(awgPeerModel("a", "x", 0)))
		if got != nil {
			t.Fatalf("expected nil for a peer only in plan, got %v", got)
		}
	})

	t.Run("edited amneziawg peer marks updated_at and null materialised fields", func(t *testing.T) {
		state := awg(awgPeerModel("a", "old", 1000), awgPeerModel("b", "same", 1000))
		planPeerA := awgPeerModel("a", "new", 1000) // comment changed, rest carried
		plan := awg(planPeerA, awgPeerModel("b", "same", 1000))
		got := editedOwnedPeerMarks("amneziawg", state, plan)
		if len(got) != 1 {
			t.Fatalf("expected marks for exactly one peer, got %v", got)
		}
		attrs := got[0]
		// private_key, sub_id and traffic_reset_day are null/zero in the
		// fixture; created_at/allowed_ips/traffic_reset are set.
		want := []string{"updated_at", "private_key", "sub_id", "traffic_reset_day"}
		if !equalStringSlices(attrs, want) {
			t.Fatalf("peer 0 marks = %v, want %v", attrs, want)
		}
	})

	t.Run("reorder alone is not an edit", func(t *testing.T) {
		state := awg(awgPeerModel("a", "x", 1000), awgPeerModel("b", "y", 1000))
		plan := awg(awgPeerModel("b", "y", 1000), awgPeerModel("a", "x", 1000))
		if got := editedOwnedPeerMarks("amneziawg", state, plan); got != nil {
			t.Fatalf("expected nil for a pure reorder, got %v", got)
		}
	})

	t.Run("edited wireguard peer marks null credentials but has no updated_at", func(t *testing.T) {
		state := wg(wgPeer("a", "old"))
		plan := wg(wgPeer("a", "new"))
		got := editedOwnedPeerMarks("wireguard", state, plan)
		if len(got) != 1 {
			t.Fatalf("expected marks for exactly one peer, got %v", got)
		}
		want := []string{"private_key", "public_key", "allowed_ips", "sub_id"}
		if !equalStringSlices(got[0], want) {
			t.Fatalf("peer 0 marks = %v, want %v", got[0], want)
		}
	})

	t.Run("edited tuic peer marks updated_at and the empty panel-rewritten fields", func(t *testing.T) {
		state := tuic(tuicPeerModel("a", "old", "", 500))
		plan := tuic(tuicPeerModel("a", "new", "", 500))
		got := editedOwnedPeerMarks("tuic", state, plan)
		if len(got) != 1 {
			t.Fatalf("expected marks for exactly one peer, got %v", got)
		}
		want := []string{"updated_at", "sub_id", "traffic_reset", "traffic_reset_day"}
		if !equalStringSlices(got[0], want) {
			t.Fatalf("peer 0 marks = %v, want %v", got[0], want)
		}
	})

	t.Run("edited tuic peer with a stored sub_id marks only the normalised fields", func(t *testing.T) {
		state := tuic(tuicPeerModel("a", "old", "sub-123", 500))
		plan := tuic(tuicPeerModel("a", "new", "sub-123", 500))
		got := editedOwnedPeerMarks("tuic", state, plan)
		if len(got) != 1 {
			t.Fatalf("expected marks for exactly one peer, got %v", got)
		}
		want := []string{"updated_at", "traffic_reset", "traffic_reset_day"}
		if !equalStringSlices(got[0], want) {
			t.Fatalf("peer 0 marks = %v, want %v", got[0], want)
		}
	})

	t.Run("edited tuic peer without created_at marks it too", func(t *testing.T) {
		state := tuic(tuicPeerModel("a", "old", "sub-123", 0))
		plan := tuic(tuicPeerModel("a", "new", "sub-123", 0))
		got := editedOwnedPeerMarks("tuic", state, plan)
		want := []string{"updated_at", "created_at", "traffic_reset", "traffic_reset_day"}
		if !equalStringSlices(got[0], want) {
			t.Fatalf("peer 0 marks = %v, want %v", got[0], want)
		}
	})

	t.Run("unchanged tuic peer yields nothing", func(t *testing.T) {
		if got := editedOwnedPeerMarks("tuic", tuic(tuicPeerModel("a", "x", "", 500)), tuic(tuicPeerModel("a", "x", "", 500))); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("plan peer without an email is skipped", func(t *testing.T) {
		state := wg(wgPeer("a", "old"))
		plan := wg(InboundWireguardClientModel{}, wgPeer("a", "new"))
		got := editedOwnedPeerMarks("wireguard", state, plan)
		if len(got) != 1 {
			t.Fatalf("expected marks only for the emailable peer, got %v", got)
		}
		if _, ok := got[0]; ok {
			t.Fatalf("the email-less peer at index 0 must be skipped, got %v", got)
		}
	})
}

// TestPeerHelperGuards covers the defensive guard branches of the peer
// helpers: nil/unknown inputs and can't-produce-a-value expander paths.
func TestPeerHelperGuards(t *testing.T) {
	t.Run("normalizePeerMap returns the input when it cannot be marshalled", func(t *testing.T) {
		in := map[string]any{"ch": make(chan int)}
		got := normalizePeerMap(in)
		if _, ok := got["ch"].(chan int); !ok {
			t.Fatalf("expected the input back unchanged, got %v", got)
		}
	})

	t.Run("ownedPeerCount", func(t *testing.T) {
		if got := ownedPeerCount("vless", &InboundResourceModel{}); got != 0 {
			t.Errorf("unknown protocol: got %d", got)
		}
		if got := ownedPeerCount("wireguard", nil); got != 0 {
			t.Errorf("nil model: got %d", got)
		}
	})

	t.Run("expandOwnedPeerAt guards", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			protocol string
			m        *InboundResourceModel
			i        int
		}{
			{"unknown protocol", "vless", &InboundResourceModel{}, 0},
			{"nil model amneziawg", "amneziawg", nil, 0},
			{"nil model wireguard", "wireguard", nil, 0},
			{"nil model tuic", "tuic", nil, 0},
			{"settings absent amneziawg", "amneziawg", &InboundResourceModel{}, 0},
			{"settings absent wireguard", "wireguard", &InboundResourceModel{}, 0},
			{"settings absent tuic", "tuic", &InboundResourceModel{}, 0},
			{"index out of range", "wireguard", &InboundResourceModel{WireguardSettings: &InboundWireguardSettingsModel{}}, 4},
			// An all-null client expands to an empty entry the batch expander
			// drops, exercising the length-mismatch fallback.
			{"empty expansion", "amneziawg", &InboundResourceModel{AmneziawgSettings: &InboundAmneziawgSettingsModel{
				Clients: []InboundAmneziawgClientModel{{}},
			}}, 0},
		} {
			if got := expandOwnedPeerAt(tc.protocol, tc.m, tc.i); len(got) != 0 {
				t.Errorf("%s: expected empty map, got %v", tc.name, got)
			}
		}
	})

	t.Run("ownedPeerAttrIsEmpty unknown attribute", func(t *testing.T) {
		m := &InboundResourceModel{WireguardSettings: &InboundWireguardSettingsModel{
			Clients: []InboundWireguardClientModel{{Email: types.StringValue("a")}},
		}}
		if ownedPeerAttrIsEmpty("wireguard", m, 0, "no_such_attr") {
			t.Error("unknown attribute must report not-empty")
		}
		if ownedPeerAttrIsEmpty("vless", m, 0, "sub_id") {
			t.Error("unknown protocol must report not-empty")
		}
	})

	t.Run("newOwnedPeerIndexes guards", func(t *testing.T) {
		m := &InboundResourceModel{}
		if got := newOwnedPeerIndexes("vless", m, m); got != nil {
			t.Errorf("unknown protocol: got %v", got)
		}
		if got := newOwnedPeerIndexes("wireguard", nil, m); got != nil {
			t.Errorf("nil state: got %v", got)
		}
		if got := newOwnedPeerIndexes("wireguard", m, nil); got != nil {
			t.Errorf("nil plan: got %v", got)
		}
	})

	t.Run("materializedPeerAttrs unknown protocol", func(t *testing.T) {
		if got := materializedPeerAttrs("vless"); got != nil {
			t.Errorf("got %v", got)
		}
	})
}

func TestInboundResourceModifyPlanMarksEditedPeerVolatileAttrs(t *testing.T) {
	r := &InboundResource{}
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	mkPlan := func(m *InboundResourceModel) tfsdk.Plan {
		p := tfsdk.Plan{Schema: schemaResp.Schema}
		if diags := p.Set(ctx, m); diags.HasError() {
			t.Fatalf("building plan fixture: %v", diags)
		}
		return p
	}

	stateModel := &InboundResourceModel{
		ID:       types.StringValue("7"),
		Remark:   types.StringValue("r"),
		Enable:   types.BoolValue(true),
		Port:     types.Int64Value(25070),
		Protocol: types.StringValue("amneziawg"),
		AmneziawgSettings: &InboundAmneziawgSettingsModel{
			Clients: []InboundAmneziawgClientModel{
				awgPeerModel("a", "old", 1000),
				awgPeerModel("b", "same", 1000),
			},
		},
	}
	planModel := &InboundResourceModel{
		ID:       types.StringValue("7"),
		Remark:   types.StringValue("r"),
		Enable:   types.BoolValue(true),
		Port:     types.Int64Value(25070),
		Protocol: types.StringValue("amneziawg"),
		AmneziawgSettings: &InboundAmneziawgSettingsModel{
			Clients: []InboundAmneziawgClientModel{
				awgPeerModel("a", "new", 1000), // comment edited
				awgPeerModel("b", "same", 1000),
			},
		},
	}

	plan := mkPlan(planModel)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, stateModel); diags.HasError() {
		t.Fatalf("building state fixture: %v", diags)
	}

	resp := &resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}

	get := func(idx int, attrName string) attr.Value {
		var v attr.Value
		diags := resp.Plan.GetAttribute(ctx,
			path.Root("amneziawg_settings").AtName("clients").AtListIndex(idx).AtName(attrName), &v)
		if diags.HasError() {
			t.Fatalf("reading %s of peer %d: %v", attrName, idx, diags)
		}
		return v
	}

	if v := get(0, "updated_at"); !v.IsUnknown() {
		t.Errorf("expected updated_at of the edited peer to be unknown, got %v", v)
	}
	if v := get(0, "private_key"); !v.IsUnknown() {
		t.Errorf("expected private_key (null in plan) of the edited peer to be unknown, got %v", v)
	}
	if v := get(0, "allowed_ips"); v.IsUnknown() {
		t.Errorf("allowed_ips is configured; it must stay promised, got unknown")
	}
	if v := get(0, "created_at"); v.IsUnknown() {
		t.Errorf("created_at is carried in state; it must stay promised, got unknown")
	}
	if v := get(1, "updated_at"); v.IsUnknown() {
		t.Errorf("the untouched peer must keep its planned updated_at, got unknown")
	}
}

// inboundResourceFixture builds plan/state fixtures for the inbound resource
// from Go models.
func inboundResourceFixture(t *testing.T, r *InboundResource, m *InboundResourceModel) tfsdk.Plan {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	p := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := p.Set(ctx, m); diags.HasError() {
		t.Fatalf("building fixture: %v", diags)
	}
	return p
}

func newInboundResourceUpdateResponse(t *testing.T, r *InboundResource) resource.UpdateResponse {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	return resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		},
	}
}

func newInboundResourceCreateResponse(t *testing.T, r *InboundResource) resource.CreateResponse {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	return resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		},
	}
}

// v390InboundServer simulates a 3x-ui v3.9.0 panel for the inbound resource:
// the update endpoint persists scalars but restores the stored clients array
// and the stored enable flag (keepStoredClients), while the shared client
// endpoints mutate the client array.
type v390InboundServer struct {
	srv *httptest.Server

	protocol string
	settings map[string]any
	remark   string
	enable   bool
	// dropEmptyClientsKey simulates panels whose settings lose the clients
	// key entirely once the last peer is deleted.
	dropEmptyClientsKey bool

	updateCalls    atomic.Int32
	setEnableCalls atomic.Int32
	addCalls       atomic.Int32
	clientUpdCalls atomic.Int32
	delCalls       atomic.Int32
}

func newV390InboundServer(t *testing.T, protocol string, settings map[string]any) *v390InboundServer {
	t.Helper()
	s := &v390InboundServer{protocol: protocol, settings: settings, remark: "before", enable: true}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *v390InboundServer) settingsString() string {
	b, _ := json.Marshal(s.settings)
	return string(b)
}

func (s *v390InboundServer) inbound() map[string]any {
	return map[string]any{
		"id":             7,
		"remark":         s.remark,
		"enable":         s.enable,
		"port":           25070,
		"protocol":       s.protocol,
		"settings":       s.settingsString(),
		"sniffing":       "{}",
		"streamSettings": "{}",
	}
}

func (s *v390InboundServer) handle(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/panel/api/clients/list":
		_, _ = w.Write(okResponse([]any{}))
	case r.URL.Path == "/panel/api/inbounds/get/7":
		_, _ = w.Write(okResponse(s.inbound()))
	case r.URL.Path == "/panel/api/inbounds/update/7" && r.Method == http.MethodPost:
		_ = r.ParseForm()
		// keepStoredClients: settings and enable from the form are ignored.
		s.remark = r.Form.Get("remark")
		s.updateCalls.Add(1)
		_, _ = w.Write(okResponse(s.inbound()))
	case r.URL.Path == "/panel/api/inbounds/setEnable/7" && r.Method == http.MethodPost:
		_ = r.ParseForm()
		s.enable = r.Form.Get("enable") == "true"
		s.setEnableCalls.Add(1)
		_, _ = w.Write(okResponse(nil))
	case r.URL.Path == "/panel/api/clients/add" && r.Method == http.MethodPost:
		var payload struct {
			Client map[string]any `json:"client"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		clients, _ := s.settings["clients"].([]any)
		s.settings["clients"] = append(clients, payload.Client)
		s.addCalls.Add(1)
		_, _ = w.Write(okResponse(nil))
	case strings.HasPrefix(r.URL.Path, "/panel/api/clients/update/") && r.Method == http.MethodPost:
		email := strings.TrimPrefix(r.URL.Path, "/panel/api/clients/update/")
		var client map[string]any
		_ = json.NewDecoder(r.Body).Decode(&client)
		clients, _ := s.settings["clients"].([]any)
		for i, c := range clients {
			if m, ok := c.(map[string]any); ok && m["email"] == email {
				clients[i] = client
				s.settings["clients"] = clients
				s.clientUpdCalls.Add(1)
				_, _ = w.Write(okResponse(nil))
				return
			}
		}
		_, _ = w.Write(failResponse(fmt.Sprintf("client %q not found in any inbound or client record", email)))
	case strings.HasPrefix(r.URL.Path, "/panel/api/clients/del/") && r.Method == http.MethodPost:
		email := strings.TrimPrefix(r.URL.Path, "/panel/api/clients/del/")
		clients, _ := s.settings["clients"].([]any)
		for i, c := range clients {
			if m, ok := c.(map[string]any); ok && m["email"] == email {
				remaining := append(clients[:i], clients[i+1:]...)
				if len(remaining) == 0 && s.dropEmptyClientsKey {
					delete(s.settings, "clients")
				} else {
					s.settings["clients"] = remaining
				}
				s.delCalls.Add(1)
				_, _ = w.Write(okResponse(nil))
				return
			}
		}
		// The row is already gone (reconcile deleted it): the panel reports
		// not-found, which the orphan cleanup tolerates.
		_, _ = w.Write(failResponse(fmt.Sprintf("client %q not found in any inbound or client record", email)))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func wgModel(id, remark string, peers ...InboundWireguardClientModel) *InboundResourceModel {
	return &InboundResourceModel{
		ID:       types.StringValue(id),
		Remark:   types.StringValue(remark),
		Enable:   types.BoolValue(true),
		Port:     types.Int64Value(25070),
		Protocol: types.StringValue("wireguard"),
		WireguardSettings: &InboundWireguardSettingsModel{
			MTU:     types.ListNull(types.Int64Type),
			Gateway: types.ListNull(types.StringType),
			DNS:     types.ListNull(types.StringType),
			Clients: peers,
		},
	}
}

func wgPeerModel(email string, kv ...string) InboundWireguardClientModel {
	m := InboundWireguardClientModel{
		Email:      types.StringValue(email),
		AllowedIPs: types.ListNull(types.StringType),
	}
	for i := 0; i+1 < len(kv); i += 2 {
		switch kv[i] {
		case "public_key":
			m.PublicKey = types.StringValue(kv[i+1])
		case "comment":
			m.Comment = types.StringValue(kv[i+1])
		case "allowed_ips":
			m.AllowedIPs = types.ListValueMust(types.StringType, []attr.Value{types.StringValue(kv[i+1])})
		}
	}
	return m
}

// TestInboundResourceUpdateReconcilesPeersOnV390 is the end-to-end check of
// the v3.9.0 fix: the update endpoint silently restores the stored clients,
// so the peer delta — one edit, one add, one removal — must land through the
// client endpoints, and the recorded state must match the plan. It is also
// the regression test for the partial-removal deadlock: the plan still has
// clients, so the update holds inboundClientMu across the reconciliation
// while releasePeerEmailsRemovedByUpdate takes the same mutex afterwards.
func TestInboundResourceUpdateReconcilesPeersOnV390(t *testing.T) {
	srv := newV390InboundServer(t, "wireguard", map[string]any{
		"clients": []any{
			map[string]any{"email": "a@t.com", "publicKey": "pkA", "comment": "old", "allowedIPs": []any{"10.0.0.2/32"}},
			map[string]any{"email": "c@t.com", "publicKey": "pkC", "allowedIPs": []any{"10.0.0.4/32"}},
		},
	})

	r := &InboundResource{client: newTestClient(t, srv.srv.URL)}
	stateModel := wgModel("7", "before",
		wgPeerModel("a@t.com", "public_key", "pkA", "comment", "old", "allowed_ips", "10.0.0.2/32"),
		wgPeerModel("c@t.com", "public_key", "pkC", "allowed_ips", "10.0.0.4/32"),
	)
	planModel := wgModel("7", "after",
		wgPeerModel("a@t.com", "public_key", "pkA", "comment", "new", "allowed_ips", "10.0.0.2/32"),
		wgPeerModel("b@t.com", "public_key", "pkB", "allowed_ips", "10.0.0.3/32"),
	)

	plan := inboundResourceFixture(t, r, planModel)
	state := tfsdk.State(inboundResourceFixture(t, r, stateModel))
	resp := newInboundResourceUpdateResponse(t, r)
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on Update: %v", resp.Diagnostics)
	}

	if got := srv.addCalls.Load(); got != 1 {
		t.Errorf("expected 1 client add (peer b), got %d", got)
	}
	if got := srv.clientUpdCalls.Load(); got != 1 {
		t.Errorf("expected 1 client update (peer a), got %d", got)
	}
	if got := srv.delCalls.Load(); got != 1 {
		t.Errorf("expected 1 client del (peer c), got %d", got)
	}
	if got := srv.setEnableCalls.Load(); got != 0 {
		t.Errorf("enable matched; expected no setEnable call, got %d", got)
	}
	if got := srv.remark; got != "after" {
		t.Errorf("expected scalar update to persist (remark=after), got %q", got)
	}

	var got InboundResourceModel
	resp.State.Get(context.Background(), &got)
	if len(got.WireguardSettings.Clients) != 2 {
		t.Fatalf("expected 2 peers in state, got %d", len(got.WireguardSettings.Clients))
	}
	if got.WireguardSettings.Clients[0].Comment.ValueString() != "new" {
		t.Errorf("expected peer a comment=new in state, got %q", got.WireguardSettings.Clients[0].Comment.ValueString())
	}
	if got.WireguardSettings.Clients[1].Email.ValueString() != "b@t.com" {
		t.Errorf("expected peer b in state, got %q", got.WireguardSettings.Clients[1].Email.ValueString())
	}
}

// TestInboundResourceUpdateRemovesLastPeerOnV390 covers removal-to-zero: the
// plan carries no clients block at all, the stored peer must be deleted
// through the client endpoint, and the orphan cleanup must not double-delete
// it (the panel's not-found is tolerated).
func TestInboundResourceUpdateRemovesLastPeerOnV390(t *testing.T) {
	srv := newV390InboundServer(t, "amneziawg", map[string]any{
		"server":  map[string]any{"publicKey": "server-pk"},
		"clients": []any{map[string]any{"email": "a@t.com", "publicKey": "pkA"}},
	})

	r := &InboundResource{client: newTestClient(t, srv.srv.URL)}
	stateModel := &InboundResourceModel{
		ID:       types.StringValue("7"),
		Remark:   types.StringValue("before"),
		Enable:   types.BoolValue(true),
		Port:     types.Int64Value(25070),
		Protocol: types.StringValue("amneziawg"),
		AmneziawgSettings: &InboundAmneziawgSettingsModel{
			Server: &InboundAmneziawgServerModel{PublicKey: types.StringValue("server-pk")},
			Clients: []InboundAmneziawgClientModel{{
				Email:      types.StringValue("a@t.com"),
				PublicKey:  types.StringValue("pkA"),
				AllowedIPs: types.ListNull(types.StringType),
			}},
		},
	}
	planModel := &InboundResourceModel{
		ID:       types.StringValue("7"),
		Remark:   types.StringValue("after"),
		Enable:   types.BoolValue(true),
		Port:     types.Int64Value(25070),
		Protocol: types.StringValue("amneziawg"),
		AmneziawgSettings: &InboundAmneziawgSettingsModel{
			Server: &InboundAmneziawgServerModel{PublicKey: types.StringValue("server-pk")},
		},
	}

	plan := inboundResourceFixture(t, r, planModel)
	state := tfsdk.State(inboundResourceFixture(t, r, stateModel))
	resp := newInboundResourceUpdateResponse(t, r)
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on Update: %v", resp.Diagnostics)
	}
	if resp.Diagnostics.WarningsCount() > 0 {
		t.Fatalf("expected no warnings (the not-found re-delete is tolerated), got %v", resp.Diagnostics)
	}
	if got := srv.delCalls.Load(); got != 1 {
		t.Errorf("expected exactly 1 client del, got %d", got)
	}

	var got InboundResourceModel
	resp.State.Get(context.Background(), &got)
	if got.AmneziawgSettings != nil && len(got.AmneziawgSettings.Clients) != 0 {
		t.Errorf("expected no peers in state, got %d", len(got.AmneziawgSettings.Clients))
	}
}

// TestInboundResourceCreatePeerSafetyNet covers the create side: AddInbound
// persists the posted peers on every panel, so the reconciliation must stay
// silent; if the panel ever drops a peer on create, the reconciliation
// re-adds it through the client endpoint.
// TestAlignBlocksWithPlanRestoresEmptyPeerBlocks pins the removal-to-zero
// round-trip: an empty wireguard_settings/tuic_settings block comes back from
// the panel as settings "{}", which flattens to nil — the declared block must
// be restored or the apply fails with "was present, but now absent".
func TestAlignBlocksWithPlanRestoresEmptyPeerBlocks(t *testing.T) {
	t.Run("wireguard block restored when plan declares it", func(t *testing.T) {
		state := &InboundResourceModel{}
		plan := &InboundResourceModel{WireguardSettings: &InboundWireguardSettingsModel{}}
		alignBlocksWithPlan(state, plan)
		if state.WireguardSettings == nil {
			t.Fatal("expected the wireguard block restored")
		}
		if !state.WireguardSettings.MTU.IsNull() {
			t.Errorf("MTU must be null, got %v", state.WireguardSettings.MTU)
		}
		if got := state.WireguardSettings.MTU.Type(context.Background()); got != (types.ListType{ElemType: types.Int64Type}) {
			t.Errorf("MTU must be a typed null list, got %v", got)
		}
		if len(state.WireguardSettings.Clients) != 0 {
			t.Errorf("expected no clients, got %d", len(state.WireguardSettings.Clients))
		}
	})

	t.Run("wireguard block nilled when plan omits it", func(t *testing.T) {
		state := &InboundResourceModel{WireguardSettings: &InboundWireguardSettingsModel{}}
		plan := &InboundResourceModel{}
		alignBlocksWithPlan(state, plan)
		if state.WireguardSettings != nil {
			t.Fatal("expected the wireguard block removed")
		}
	})

	t.Run("tuic block restored when plan declares it", func(t *testing.T) {
		state := &InboundResourceModel{}
		plan := &InboundResourceModel{TuicSettings: &InboundTuicSettingsModel{}}
		alignBlocksWithPlan(state, plan)
		if state.TuicSettings == nil {
			t.Fatal("expected the tuic block restored")
		}
	})

	t.Run("populated block left alone", func(t *testing.T) {
		existing := &InboundWireguardSettingsModel{
			Clients: []InboundWireguardClientModel{{Email: types.StringValue("a@t.com")}},
		}
		state := &InboundResourceModel{WireguardSettings: existing}
		plan := &InboundResourceModel{WireguardSettings: &InboundWireguardSettingsModel{}}
		alignBlocksWithPlan(state, plan)
		if state.WireguardSettings != existing {
			t.Fatal("a populated block must not be replaced")
		}
	})
}

// TestInboundResourceUpdateRemovesLastWireguardPeerKeepsBlock is the WireGuard
// removal-to-zero case: after the last peer is deleted the panel's settings
// lose the clients key entirely, so the flattened block is nil — the declared
// wireguard_settings block must survive in state.
func TestInboundResourceUpdateRemovesLastWireguardPeerKeepsBlock(t *testing.T) {
	srv := newV390InboundServer(t, "wireguard", map[string]any{
		"clients": []any{map[string]any{"email": "a@t.com", "publicKey": "pkA"}},
	})
	// Simulate the panel dropping the clients key once the last peer is gone.
	srv.dropEmptyClientsKey = true

	r := &InboundResource{client: newTestClient(t, srv.srv.URL)}
	stateModel := wgModel("7", "before", wgPeerModel("a@t.com", "public_key", "pkA"))
	planModel := &InboundResourceModel{
		ID:       types.StringValue("7"),
		Remark:   types.StringValue("after"),
		Enable:   types.BoolValue(true),
		Port:     types.Int64Value(25070),
		Protocol: types.StringValue("wireguard"),
		WireguardSettings: &InboundWireguardSettingsModel{
			MTU:     types.ListNull(types.Int64Type),
			Gateway: types.ListNull(types.StringType),
			DNS:     types.ListNull(types.StringType),
		},
	}

	plan := inboundResourceFixture(t, r, planModel)
	state := tfsdk.State(inboundResourceFixture(t, r, stateModel))
	resp := newInboundResourceUpdateResponse(t, r)
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on Update: %v", resp.Diagnostics)
	}
	if got := srv.delCalls.Load(); got != 1 {
		t.Errorf("expected exactly 1 client del, got %d", got)
	}

	var got InboundResourceModel
	resp.State.Get(context.Background(), &got)
	if got.WireguardSettings == nil {
		t.Fatal("expected the wireguard_settings block to survive removal-to-zero")
	}
	if len(got.WireguardSettings.Clients) != 0 {
		t.Errorf("expected no peers in state, got %d", len(got.WireguardSettings.Clients))
	}
}

func TestInboundResourceCreatePeerSafetyNet(t *testing.T) {
	newCreateServer := func(t *testing.T, dropClients bool) (*httptest.Server, *atomic.Int32) {
		t.Helper()
		var settings map[string]any
		var addCalls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/panel/api/clients/list":
				_, _ = w.Write(okResponse([]any{}))
			case r.URL.Path == "/panel/api/inbounds/add" && r.Method == http.MethodPost:
				_ = r.ParseForm()
				posted := r.Form.Get("settings")
				settings = map[string]any{}
				_ = json.Unmarshal([]byte(posted), &settings)
				if dropClients {
					delete(settings, "clients")
				}
				b, _ := json.Marshal(settings)
				_, _ = w.Write(okResponse(map[string]any{
					"id": 7, "remark": "r", "enable": true, "port": 25070,
					"protocol": "wireguard", "settings": string(b),
					"sniffing": "{}", "streamSettings": "{}",
				}))
			case r.URL.Path == "/panel/api/inbounds/get/7":
				b, _ := json.Marshal(settings)
				_, _ = w.Write(okResponse(map[string]any{
					"id": 7, "remark": "r", "enable": true, "port": 25070,
					"protocol": "wireguard", "settings": string(b),
					"sniffing": "{}", "streamSettings": "{}",
				}))
			case r.URL.Path == "/panel/api/clients/add" && r.Method == http.MethodPost:
				var payload struct {
					Client map[string]any `json:"client"`
				}
				_ = json.NewDecoder(r.Body).Decode(&payload)
				clients, _ := settings["clients"].([]any)
				settings["clients"] = append(clients, payload.Client)
				addCalls.Add(1)
				_, _ = w.Write(okResponse(nil))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		t.Cleanup(srv.Close)
		return srv, &addCalls
	}

	planModel := wgModel("", "r",
		wgPeerModel("a@t.com", "public_key", "pkA", "allowed_ips", "10.0.0.2/32"),
	)

	t.Run("silent when the panel stored the peers", func(t *testing.T) {
		srv, addCalls := newCreateServer(t, false)
		r := &InboundResource{client: newTestClient(t, srv.URL)}
		plan := inboundResourceFixture(t, r, planModel)
		resp := newInboundResourceCreateResponse(t, r)
		r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected error on Create: %v", resp.Diagnostics)
		}
		if got := addCalls.Load(); got != 0 {
			t.Errorf("expected no client-endpoint calls when create persisted the peers, got %d", got)
		}
		var got InboundResourceModel
		resp.State.Get(context.Background(), &got)
		if len(got.WireguardSettings.Clients) != 1 {
			t.Fatalf("expected 1 peer in state, got %d", len(got.WireguardSettings.Clients))
		}
	})

	t.Run("re-adds a peer the panel dropped", func(t *testing.T) {
		srv, addCalls := newCreateServer(t, true)
		r := &InboundResource{client: newTestClient(t, srv.URL)}
		plan := inboundResourceFixture(t, r, planModel)
		resp := newInboundResourceCreateResponse(t, r)
		r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected error on Create: %v", resp.Diagnostics)
		}
		if got := addCalls.Load(); got != 1 {
			t.Errorf("expected 1 compensating add for the dropped peer, got %d", got)
		}
		var got InboundResourceModel
		resp.State.Get(context.Background(), &got)
		if len(got.WireguardSettings.Clients) != 1 {
			t.Fatalf("expected 1 peer in state, got %d", len(got.WireguardSettings.Clients))
		}
	})
}

// TestInboundResourceCreateReconcileAndEnableErrors covers the Create-time
// error branches of the two post-write reconciliations: a peer the panel
// dropped that cannot be re-added, and an enable flag the panel dropped whose
// setEnable push fails.
func TestInboundResourceCreateReconcileAndEnableErrors(t *testing.T) {
	planModel := wgModel("", "r",
		wgPeerModel("a@t.com", "public_key", "pkA", "allowed_ips", "10.0.0.2/32"),
	)

	newServer := func(t *testing.T, dropClients bool, storedEnable bool, setEnableErr bool) *httptest.Server {
		t.Helper()
		var settings map[string]any
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/panel/api/clients/list":
				_, _ = w.Write(okResponse([]any{}))
			case r.URL.Path == "/panel/api/inbounds/add" && r.Method == http.MethodPost:
				_ = r.ParseForm()
				settings = map[string]any{}
				_ = json.Unmarshal([]byte(r.Form.Get("settings")), &settings)
				if dropClients {
					delete(settings, "clients")
				}
				b, _ := json.Marshal(settings)
				_, _ = w.Write(okResponse(map[string]any{
					"id": 7, "remark": "r", "enable": storedEnable, "port": 25070,
					"protocol": "wireguard", "settings": string(b),
					"sniffing": "{}", "streamSettings": "{}",
				}))
			case r.URL.Path == "/panel/api/inbounds/get/7":
				b, _ := json.Marshal(settings)
				_, _ = w.Write(okResponse(map[string]any{
					"id": 7, "remark": "r", "enable": storedEnable, "port": 25070,
					"protocol": "wireguard", "settings": string(b),
					"sniffing": "{}", "streamSettings": "{}",
				}))
			case r.URL.Path == "/panel/api/clients/add" && r.Method == http.MethodPost:
				_, _ = w.Write(failResponse("Duplicate email: a@t.com"))
			case r.URL.Path == "/panel/api/inbounds/setEnable/7" && r.Method == http.MethodPost:
				if setEnableErr {
					_, _ = w.Write(failResponse("cannot set enable"))
					return
				}
				_, _ = w.Write(okResponse(nil))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
	}

	t.Run("peer reconcile failure is a create error", func(t *testing.T) {
		srv := newServer(t, true, true, false)
		defer srv.Close()
		r := &InboundResource{client: newTestClient(t, srv.URL)}
		plan := inboundResourceFixture(t, r, planModel)
		resp := newInboundResourceCreateResponse(t, r)
		r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error when the dropped peer cannot be re-added")
		}
		if got := resp.Diagnostics.Errors()[0].Summary(); got != "Failed to reconcile inbound peers" {
			t.Fatalf("unexpected diagnostic summary: %q", got)
		}
	})

	t.Run("enable push failure is a create error", func(t *testing.T) {
		srv := newServer(t, false, false, true)
		defer srv.Close()
		r := &InboundResource{client: newTestClient(t, srv.URL)}
		plan := inboundResourceFixture(t, r, planModel)
		resp := newInboundResourceCreateResponse(t, r)
		r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error when the dropped enable flag cannot be pushed")
		}
		if got := resp.Diagnostics.Errors()[0].Summary(); got != "Failed to reconcile inbound enable state" {
			t.Fatalf("unexpected diagnostic summary: %q", got)
		}
	})

	t.Run("enable push success updates the local copy", func(t *testing.T) {
		srv := newServer(t, false, false, false)
		defer srv.Close()
		r := &InboundResource{client: newTestClient(t, srv.URL)}
		plan := inboundResourceFixture(t, r, planModel)
		resp := newInboundResourceCreateResponse(t, r)
		r.Create(context.Background(), resource.CreateRequest{Plan: plan}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected error on Create: %v", resp.Diagnostics)
		}
		var got InboundResourceModel
		resp.State.Get(context.Background(), &got)
		if !got.Enable.ValueBool() {
			t.Error("expected the pushed enable=true in state")
		}
	})
}

// TestInboundResourceUpdateErrorSurfaces covers the failure branch of the
// locked update section: the update endpoint rejects the write, the error
// reaches the diagnostics, and — because the lock is released before the
// error handling — a follow-up update against a healthy panel is not blocked
// by a forgotten mutex.
func TestInboundResourceUpdateErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panel/api/inbounds/get/7":
			_, _ = w.Write(okResponse(map[string]any{
				"id": 7, "remark": "before", "enable": true, "port": 25070,
				"protocol": "wireguard", "settings": `{"clients":[{"email":"a@t.com","publicKey":"pkA"}]}`,
				"sniffing": "{}", "streamSettings": "{}",
			}))
		case "/panel/api/inbounds/update/7":
			_, _ = w.Write(failResponse("boom"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	r := &InboundResource{client: newTestClient(t, srv.URL)}
	stateModel := wgModel("7", "before", wgPeerModel("a@t.com", "public_key", "pkA"))
	planModel := wgModel("7", "after", wgPeerModel("a@t.com", "public_key", "pkA"))
	plan := inboundResourceFixture(t, r, planModel)
	state := tfsdk.State(inboundResourceFixture(t, r, stateModel))
	resp := newInboundResourceUpdateResponse(t, r)
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when the update endpoint rejects the write")
	}
	if got := resp.Diagnostics.Errors()[0].Summary(); got != "Failed to update inbound" {
		t.Fatalf("unexpected diagnostic summary: %q", got)
	}
}

// TestInboundResourceUpdatePollBranches covers the error and retry branches
// inside updateInboundAndReconcilePeers' read-after-write poll: a GET that
// keeps failing, a setEnable push that fails, an enable flip that needs one
// retry to become visible, and a peer reconciliation that fails.
func TestInboundResourceUpdatePollBranches(t *testing.T) {
	storedInbound := func(enable bool, clients string) map[string]any {
		return map[string]any{
			"id": 7, "remark": "after", "enable": enable, "port": 25070,
			"protocol": "wireguard", "settings": fmt.Sprintf(`{"clients":[%s]}`, clients),
			"sniffing": "{}", "streamSettings": "{}",
		}
	}
	peerA := `{"email":"a@t.com","publicKey":"pkA"}`

	runUpdate := func(t *testing.T, srv *httptest.Server, planEnable bool) resource.UpdateResponse {
		t.Helper()
		r := &InboundResource{client: newTestClient(t, srv.URL)}
		stateModel := wgModel("7", "before", wgPeerModel("a@t.com", "public_key", "pkA"))
		planModel := wgModel("7", "after", wgPeerModel("a@t.com", "public_key", "pkA"))
		planModel.Enable = types.BoolValue(planEnable)
		plan := inboundResourceFixture(t, r, planModel)
		state := tfsdk.State(inboundResourceFixture(t, r, stateModel))
		resp := newInboundResourceUpdateResponse(t, r)
		r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &resp)
		return resp
	}

	t.Run("get failure inside the poll exhausts the budget", func(t *testing.T) {
		var getCalls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/panel/api/inbounds/get/7":
				// The pre-update read succeeds; every poll read fails.
				if getCalls.Add(1) == 1 {
					_, _ = w.Write(okResponse(storedInbound(true, peerA)))
					return
				}
				http.Error(w, "boom", http.StatusInternalServerError)
			case "/panel/api/inbounds/update/7":
				_, _ = w.Write(okResponse(nil))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		resp := runUpdate(t, srv, true)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error when the poll reads keep failing")
		}
		if got := resp.Diagnostics.Errors()[0].Summary(); got != "Failed to update inbound" {
			t.Fatalf("unexpected diagnostic summary: %q", got)
		}
		if got := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(got, "reading updated inbound 7") {
			t.Fatalf("expected the read-after-write context, got %q", got)
		}
	})

	t.Run("setEnable failure inside the poll", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/panel/api/inbounds/get/7":
				_, _ = w.Write(okResponse(storedInbound(false, peerA)))
			case "/panel/api/inbounds/update/7":
				_, _ = w.Write(okResponse(nil))
			case "/panel/api/inbounds/setEnable/7":
				_, _ = w.Write(failResponse("cannot set enable"))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		resp := runUpdate(t, srv, true)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error when the setEnable push fails")
		}
	})

	t.Run("enable flip becomes visible after one retry", func(t *testing.T) {
		enable := false
		var setEnableCalls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/panel/api/inbounds/get/7":
				_, _ = w.Write(okResponse(storedInbound(enable, peerA)))
			case "/panel/api/inbounds/update/7":
				// keepStoredClients: the stored enable wins.
				_, _ = w.Write(okResponse(nil))
			case "/panel/api/inbounds/setEnable/7":
				_ = req.ParseForm()
				enable = req.Form.Get("enable") == "true"
				setEnableCalls.Add(1)
				_, _ = w.Write(okResponse(nil))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		resp := runUpdate(t, srv, true)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected error: %v", resp.Diagnostics)
		}
		if got := setEnableCalls.Load(); got != 1 {
			t.Errorf("expected exactly 1 setEnable push, got %d", got)
		}
		var got InboundResourceModel
		resp.State.Get(context.Background(), &got)
		if !got.Enable.ValueBool() {
			t.Error("expected enable=true in state after the flip")
		}
	})

	t.Run("peer reconcile failure inside the update", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/panel/api/clients/list":
				_, _ = w.Write(okResponse([]any{}))
			case "/panel/api/inbounds/get/7":
				// keepStoredClients: the new peer never appears on reads.
				_, _ = w.Write(okResponse(storedInbound(true, peerA)))
			case "/panel/api/inbounds/update/7":
				_, _ = w.Write(okResponse(nil))
			case "/panel/api/clients/add":
				_, _ = w.Write(failResponse("Duplicate email: b@t.com"))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer srv.Close()

		r := &InboundResource{client: newTestClient(t, srv.URL)}
		stateModel := wgModel("7", "before", wgPeerModel("a@t.com", "public_key", "pkA"))
		planModel := wgModel("7", "after",
			wgPeerModel("a@t.com", "public_key", "pkA"),
			wgPeerModel("b@t.com", "public_key", "pkB"),
		)
		plan := inboundResourceFixture(t, r, planModel)
		state := tfsdk.State(inboundResourceFixture(t, r, stateModel))
		resp := newInboundResourceUpdateResponse(t, r)
		r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error when the peer reconciliation fails")
		}
		if got := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(got, "adding peer") {
			t.Fatalf("expected the adding-peer context, got %q", got)
		}
	})

	t.Run("malformed stored settings fail preserveInboundSettings", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/panel/api/inbounds/get/7" {
				_, _ = w.Write(okResponse(map[string]any{
					"id": 7, "remark": "before", "enable": true, "port": 25070,
					"protocol": "wireguard", "settings": `{"clients":`,
					"sniffing": "{}", "streamSettings": "{}",
				}))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		resp := runUpdate(t, srv, true)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error when the stored settings cannot be parsed")
		}
		if got := resp.Diagnostics.Errors()[0].Summary(); got != "Failed to preserve inbound settings" {
			t.Fatalf("unexpected diagnostic summary: %q", got)
		}
	})
}

// TestExpandInboundFromModelNodeID covers the optional node assignment in the
// model-to-API conversion.
func TestExpandInboundFromModelNodeID(t *testing.T) {
	m := &InboundResourceModel{
		Remark:   types.StringValue("r"),
		Enable:   types.BoolValue(true),
		Port:     types.Int64Value(25070),
		Protocol: types.StringValue("vless"),
		NodeID:   types.Int64Value(3),
	}
	inbound := expandInboundFromModel(m)
	if inbound.NodeID == nil || *inbound.NodeID != 3 {
		t.Fatalf("expected NodeID=3, got %v", inbound.NodeID)
	}
}

// TestInboundResourceModifyPlanEarlyReturns covers the guard branches: no
// state (create), no plan (destroy), and a no-op plan must all leave the plan
// untouched.
func TestInboundResourceModifyPlanEarlyReturns(t *testing.T) {
	r := &InboundResource{}
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	nullRaw := tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil)

	model := &InboundResourceModel{
		ID:       types.StringValue("7"),
		Remark:   types.StringValue("r"),
		Enable:   types.BoolValue(true),
		Port:     types.Int64Value(25070),
		Protocol: types.StringValue("amneziawg"),
	}
	plan := inboundResourceFixture(t, r, model)

	t.Run("create with no state", func(t *testing.T) {
		resp := &resource.ModifyPlanResponse{Plan: plan}
		r.ModifyPlan(ctx, resource.ModifyPlanRequest{
			Plan:  plan,
			State: tfsdk.State{Schema: schemaResp.Schema, Raw: nullRaw},
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected error: %v", resp.Diagnostics)
		}
		if !resp.Plan.Raw.Equal(plan.Raw) {
			t.Fatal("plan must be untouched on create")
		}
	})

	t.Run("destroy with null plan", func(t *testing.T) {
		resp := &resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: nullRaw}}
		r.ModifyPlan(ctx, resource.ModifyPlanRequest{
			Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: nullRaw},
			State: tfsdk.State(plan),
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected error: %v", resp.Diagnostics)
		}
	})

	t.Run("no-op plan", func(t *testing.T) {
		resp := &resource.ModifyPlanResponse{Plan: plan}
		r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: tfsdk.State(plan)}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected error: %v", resp.Diagnostics)
		}
		if !resp.Plan.Raw.Equal(plan.Raw) {
			t.Fatal("plan must be untouched when it equals state")
		}
	})

	t.Run("non-reconciled protocol leaves the plan untouched", func(t *testing.T) {
		vlessState := &InboundResourceModel{
			ID: types.StringValue("7"), Remark: types.StringValue("r"),
			Enable: types.BoolValue(true), Port: types.Int64Value(25070), Protocol: types.StringValue("vless"),
		}
		vlessPlan := &InboundResourceModel{
			ID: types.StringValue("7"), Remark: types.StringValue("r2"),
			Enable: types.BoolValue(true), Port: types.Int64Value(25070), Protocol: types.StringValue("vless"),
		}
		p := inboundResourceFixture(t, r, vlessPlan)
		st := tfsdk.State(inboundResourceFixture(t, r, vlessState))
		resp := &resource.ModifyPlanResponse{Plan: p}
		r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: p, State: st}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected error: %v", resp.Diagnostics)
		}
		var remark types.String
		resp.Diagnostics.Append(resp.Plan.GetAttribute(ctx, path.Root("remark"), &remark)...)
		if remark.ValueString() != "r2" {
			t.Fatalf("vless remark must stay promised, got %v", remark)
		}
	})
}

func TestInboundResourceModifyPlanWireguardAndAllowedIPsMarks(t *testing.T) {
	r := &InboundResource{}
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	get := func(p *tfsdk.Plan, block string, idx int, attrName string) attr.Value {
		var v attr.Value
		diags := p.GetAttribute(ctx, path.Root(block).AtName("clients").AtListIndex(idx).AtName(attrName), &v)
		if diags.HasError() {
			t.Fatalf("reading %s of peer %d: %v", attrName, idx, diags)
		}
		return v
	}
	run := func(state, plan *InboundResourceModel) tfsdk.Plan {
		p := inboundResourceFixture(t, r, plan)
		st := tfsdk.State(inboundResourceFixture(t, r, state))
		resp := &resource.ModifyPlanResponse{Plan: p}
		r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: p, State: st}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected error: %v", resp.Diagnostics)
		}
		return resp.Plan
	}

	t.Run("wireguard edited peer marks empty sub_id", func(t *testing.T) {
		peer := func(comment string) InboundWireguardClientModel {
			return InboundWireguardClientModel{
				Email:      types.StringValue("a@t.com"),
				PublicKey:  types.StringValue("pkA"),
				Comment:    types.StringValue(comment),
				AllowedIPs: types.ListNull(types.StringType),
			}
		}
		wgSettings := func(comment string) *InboundWireguardSettingsModel {
			return &InboundWireguardSettingsModel{
				MTU:     types.ListNull(types.Int64Type),
				Gateway: types.ListNull(types.StringType),
				DNS:     types.ListNull(types.StringType),
				Clients: []InboundWireguardClientModel{peer(comment)},
			}
		}
		state := &InboundResourceModel{
			ID: types.StringValue("7"), Remark: types.StringValue("r"),
			Enable: types.BoolValue(true), Port: types.Int64Value(25070), Protocol: types.StringValue("wireguard"),
			WireguardSettings: wgSettings("old"),
		}
		plan := &InboundResourceModel{
			ID: types.StringValue("7"), Remark: types.StringValue("r"),
			Enable: types.BoolValue(true), Port: types.Int64Value(25070), Protocol: types.StringValue("wireguard"),
			WireguardSettings: wgSettings("new"),
		}
		out := run(state, plan)
		if v := get(&out, "wireguard_settings", 0, "sub_id"); !v.IsUnknown() {
			t.Errorf("expected sub_id of the edited wireguard peer to be unknown, got %v", v)
		}
		if v := get(&out, "wireguard_settings", 0, "allowed_ips"); !v.IsUnknown() {
			t.Errorf("expected allowed_ips (null in plan) of the edited wireguard peer to be unknown, got %v", v)
		}
		if v := get(&out, "wireguard_settings", 0, "comment"); v.IsUnknown() {
			t.Errorf("comment is configured; it must stay promised, got unknown")
		}
	})

	t.Run("amneziawg edited peer with null allowed_ips plans a list unknown", func(t *testing.T) {
		peer := func(comment string) InboundAmneziawgClientModel {
			return InboundAmneziawgClientModel{
				Email:      types.StringValue("a@t.com"),
				PublicKey:  types.StringValue("pkA"),
				Comment:    types.StringValue(comment),
				AllowedIPs: types.ListNull(types.StringType),
				CreatedAt:  types.Int64Value(500),
				UpdatedAt:  types.Int64Value(1000),
			}
		}
		state := &InboundResourceModel{
			ID: types.StringValue("7"), Remark: types.StringValue("r"),
			Enable: types.BoolValue(true), Port: types.Int64Value(25070), Protocol: types.StringValue("amneziawg"),
			AmneziawgSettings: &InboundAmneziawgSettingsModel{Clients: []InboundAmneziawgClientModel{peer("old")}},
		}
		plan := &InboundResourceModel{
			ID: types.StringValue("7"), Remark: types.StringValue("r"),
			Enable: types.BoolValue(true), Port: types.Int64Value(25070), Protocol: types.StringValue("amneziawg"),
			AmneziawgSettings: &InboundAmneziawgSettingsModel{Clients: []InboundAmneziawgClientModel{peer("new")}},
		}
		out := run(state, plan)
		v := get(&out, "amneziawg_settings", 0, "allowed_ips")
		if !v.IsUnknown() {
			t.Fatalf("expected allowed_ips unknown, got %v", v)
		}
		if _, ok := v.(types.List); !ok {
			t.Errorf("allowed_ips unknown must be a list value, got %T", v)
		}
	})
}

func TestInboundResourceModifyPlanMarksEditedTuicPeerVolatileAttrs(t *testing.T) {
	r := &InboundResource{}
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	mkPlan := func(m *InboundResourceModel) tfsdk.Plan {
		p := tfsdk.Plan{Schema: schemaResp.Schema}
		if diags := p.Set(ctx, m); diags.HasError() {
			t.Fatalf("building plan fixture: %v", diags)
		}
		return p
	}

	stateModel := &InboundResourceModel{
		ID:       types.StringValue("7"),
		Remark:   types.StringValue("r"),
		Enable:   types.BoolValue(true),
		Port:     types.Int64Value(25070),
		Protocol: types.StringValue("tuic"),
		TuicSettings: &InboundTuicSettingsModel{
			Clients: []InboundTuicClientModel{
				tuicPeerModel("a", "old", "", 500),
				tuicPeerModel("b", "same", "sub-b", 500),
			},
		},
	}
	planModel := &InboundResourceModel{
		ID:       types.StringValue("7"),
		Remark:   types.StringValue("r"),
		Enable:   types.BoolValue(true),
		Port:     types.Int64Value(25070),
		Protocol: types.StringValue("tuic"),
		TuicSettings: &InboundTuicSettingsModel{
			Clients: []InboundTuicClientModel{
				tuicPeerModel("a", "new", "", 500), // comment edited
				tuicPeerModel("b", "same", "sub-b", 500),
			},
		},
	}

	plan := mkPlan(planModel)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, stateModel); diags.HasError() {
		t.Fatalf("building state fixture: %v", diags)
	}

	resp := &resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}

	get := func(idx int, attrName string) attr.Value {
		var v attr.Value
		diags := resp.Plan.GetAttribute(ctx,
			path.Root("tuic_settings").AtName("clients").AtListIndex(idx).AtName(attrName), &v)
		if diags.HasError() {
			t.Fatalf("reading %s of peer %d: %v", attrName, idx, diags)
		}
		return v
	}

	if v := get(0, "updated_at"); !v.IsUnknown() {
		t.Errorf("expected updated_at of the edited peer to be unknown, got %v", v)
	}
	if v := get(0, "sub_id"); !v.IsUnknown() {
		t.Errorf("expected sub_id (empty in plan) of the edited peer to be unknown, got %v", v)
	}
	if v := get(0, "created_at"); v.IsUnknown() {
		t.Errorf("created_at is carried in state; it must stay promised, got unknown")
	}
	if v := get(1, "updated_at"); v.IsUnknown() {
		t.Errorf("the untouched peer must keep its planned updated_at, got unknown")
	}
	if v := get(1, "sub_id"); v.IsUnknown() {
		t.Errorf("the untouched peer must keep its planned sub_id, got unknown")
	}
}

// TestInboundResourceModifyPlanMarksNewPeerAttrsUnknown covers peers the apply
// adds: a new list element has no prior state for UseStateForUnknown to copy,
// so its unset Optional+Computed attributes plan as null — while the panel
// materialises concrete values for them on every save. ModifyPlan must plan
// them as unknown instead, on every protocol and every panel version.
func TestInboundResourceModifyPlanMarksNewPeerAttrsUnknown(t *testing.T) {
	r := &InboundResource{}
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	mkPlan := func(m *InboundResourceModel) tfsdk.Plan {
		p := tfsdk.Plan{Schema: schemaResp.Schema}
		if diags := p.Set(ctx, m); diags.HasError() {
			t.Fatalf("building plan fixture: %v", diags)
		}
		return p
	}

	stateModel := &InboundResourceModel{
		ID:       types.StringValue("7"),
		Remark:   types.StringValue("r"),
		Enable:   types.BoolValue(true),
		Port:     types.Int64Value(25070),
		Protocol: types.StringValue("amneziawg"),
		AmneziawgSettings: &InboundAmneziawgSettingsModel{
			Clients: []InboundAmneziawgClientModel{awgPeerModel("a", "same", 1000)},
		},
	}
	newPeer := InboundAmneziawgClientModel{
		Email:      types.StringValue("b"),
		PublicKey:  types.StringValue("pk-b"),
		AllowedIPs: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("10.9.1.3/32")}),
		Enable:     types.BoolValue(true),
		// Everything else unset: plans as null without the fix.
	}
	planModel := &InboundResourceModel{
		ID:       types.StringValue("7"),
		Remark:   types.StringValue("r2"), // scalar change so ModifyPlan runs
		Enable:   types.BoolValue(true),
		Port:     types.Int64Value(25070),
		Protocol: types.StringValue("amneziawg"),
		AmneziawgSettings: &InboundAmneziawgSettingsModel{
			Clients: []InboundAmneziawgClientModel{awgPeerModel("a", "same", 1000), newPeer},
		},
	}

	plan := mkPlan(planModel)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, stateModel); diags.HasError() {
		t.Fatalf("building state fixture: %v", diags)
	}

	resp := &resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: plan, State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error: %v", resp.Diagnostics)
	}

	get := func(idx int, attrName string) attr.Value {
		var v attr.Value
		diags := resp.Plan.GetAttribute(ctx,
			path.Root("amneziawg_settings").AtName("clients").AtListIndex(idx).AtName(attrName), &v)
		if diags.HasError() {
			t.Fatalf("reading %s of peer %d: %v", attrName, idx, diags)
		}
		return v
	}

	// The new peer's unset computed attributes must plan as unknown...
	for _, attrName := range []string{"comment", "sub_id", "limit_ip", "total_gb", "reset_day", "traffic_reset", "created_at", "updated_at", "private_key", "tg_id"} {
		if v := get(1, attrName); !v.IsUnknown() {
			t.Errorf("new peer %s: expected unknown, got %v", attrName, v)
		}
	}
	// ...while its configured attributes keep their values...
	if v := get(1, "public_key"); v.IsUnknown() || v.(types.String).ValueString() != "pk-b" {
		t.Errorf("new peer public_key is configured; expected pk-b, got %v", v)
	}
	if v := get(1, "allowed_ips"); v.IsUnknown() {
		t.Errorf("new peer allowed_ips is configured; must stay promised, got unknown")
	}
	if v := get(1, "enable"); v.IsUnknown() {
		t.Errorf("new peer enable is configured; must stay promised, got unknown")
	}
	// ...and the existing peer is untouched.
	if v := get(0, "comment"); v.IsUnknown() {
		t.Errorf("existing peer comment must stay promised, got unknown")
	}
	if v := get(0, "updated_at"); v.IsUnknown() {
		t.Errorf("existing peer updated_at must stay promised, got unknown")
	}
}

// tuicV390Server simulates a 3x-ui v3.9.0 panel for a TUIC inbound, faithful
// to the two serialisation shapes the provider has to survive:
//
//   - the stored peers use the CREATE read-back shape (AddInbound re-marshals
//     every client as model.Client, so empty optional keys like subId:"" and
//     security:"" are present, and created_at/updated_at are stamped);
//   - the client update endpoint rewrites the entry to the sparse UPDATE
//     shape: the posted map, created_at preserved, updated_at bumped, and a
//     freshly generated subId when both posted and stored are blank
//     (client_inbound_apply.go:800-853).
type tuicV390Server struct {
	srv *httptest.Server

	settings map[string]any
	remark   string

	addCalls       atomic.Int32
	clientUpdCalls atomic.Int32
	delCalls       atomic.Int32
	setEnableCalls atomic.Int32
}

// tuicCreateSerialization renders a stored peer the way AddInbound's
// model.Client re-marshal does: every non-omitempty key present, stamped
// timestamps, blank subId.
func tuicCreateSerialization(email, comment string) map[string]any {
	return map[string]any{
		"id": "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee", "security": "",
		"password": "pass-" + email, "email": email,
		"limitIp": 0, "totalGB": 0, "expiryTime": 0, "enable": true,
		"tgId": 0, "subId": "", "comment": comment,
		"reset": 0, "resetDay": 0, "resetWeekday": 0, "resetMax": 0,
		"created_at": 1111, "updated_at": 2222,
	}
}

func newTuicV390Server(t *testing.T, stored []map[string]any) *tuicV390Server {
	t.Helper()
	s := &tuicV390Server{
		settings: map[string]any{"clients": []any{}},
		remark:   "before",
	}
	for _, c := range stored {
		s.settings["clients"] = append(s.settings["clients"].([]any), c)
	}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *tuicV390Server) settingsString() string {
	b, _ := json.Marshal(s.settings)
	return string(b)
}

func (s *tuicV390Server) handle(w http.ResponseWriter, r *http.Request) {
	clients, _ := s.settings["clients"].([]any)
	switch {
	case r.URL.Path == "/panel/api/clients/list":
		_, _ = w.Write(okResponse([]any{}))
	case r.URL.Path == "/panel/api/inbounds/get/7":
		_, _ = w.Write(okResponse(map[string]any{
			"id": 7, "remark": s.remark, "enable": true, "port": 25070,
			"protocol": "tuic", "settings": s.settingsString(),
			"sniffing": "{}", "streamSettings": "{}",
		}))
	case r.URL.Path == "/panel/api/inbounds/update/7" && r.Method == http.MethodPost:
		_ = r.ParseForm()
		// keepStoredClients: posted settings and enable are ignored.
		s.remark = r.Form.Get("remark")
		_, _ = w.Write(okResponse(nil))
	case r.URL.Path == "/panel/api/clients/add" && r.Method == http.MethodPost:
		var payload struct {
			Client map[string]any `json:"client"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if _, ok := payload.Client["created_at"]; !ok {
			payload.Client["created_at"] = 4444
		}
		payload.Client["updated_at"] = 4444
		if sub, _ := payload.Client["subId"].(string); sub == "" {
			payload.Client["subId"] = "gen-sub"
		}
		s.settings["clients"] = append(clients, payload.Client)
		s.addCalls.Add(1)
		_, _ = w.Write(okResponse(nil))
	case strings.HasPrefix(r.URL.Path, "/panel/api/clients/update/") && r.Method == http.MethodPost:
		email := strings.TrimPrefix(r.URL.Path, "/panel/api/clients/update/")
		var posted map[string]any
		_ = json.NewDecoder(r.Body).Decode(&posted)
		for i, c := range clients {
			m, ok := c.(map[string]any)
			if !ok || m["email"] != email {
				continue
			}
			// The endpoint preserves created_at, regenerates a blank subId
			// and bumps updated_at on a real change.
			if v, ok2 := m["created_at"]; ok2 {
				posted["created_at"] = v
			}
			if sub, _ := posted["subId"].(string); sub == "" {
				if stored, _ := m["subId"].(string); stored != "" {
					posted["subId"] = stored
				} else {
					posted["subId"] = "gen-sub"
				}
			}
			posted["updated_at"] = 3333
			clients[i] = posted
			s.settings["clients"] = clients
			s.clientUpdCalls.Add(1)
			_, _ = w.Write(okResponse(nil))
			return
		}
		_, _ = w.Write(failResponse(fmt.Sprintf("client %q not found in any inbound or client record", email)))
	case strings.HasPrefix(r.URL.Path, "/panel/api/clients/del/") && r.Method == http.MethodPost:
		email := strings.TrimPrefix(r.URL.Path, "/panel/api/clients/del/")
		for i, c := range clients {
			if m, ok := c.(map[string]any); ok && m["email"] == email {
				s.settings["clients"] = append(clients[:i], clients[i+1:]...)
				s.delCalls.Add(1)
				_, _ = w.Write(okResponse(nil))
				return
			}
		}
		_, _ = w.Write(failResponse(fmt.Sprintf("client %q not found in any inbound or client record", email)))
	case r.URL.Path == "/panel/api/inbounds/setEnable/7" && r.Method == http.MethodPost:
		s.setEnableCalls.Add(1)
		_, _ = w.Write(okResponse(nil))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// TestInboundResourceUpdateReconcilesTuicPeersOnV390 runs the full Update
// against the TUIC serialization quirk: the stored peers carry the create
// read-back shape (blank subId among the empty keys), and the endpoint update
// answers with the sparse shape plus a generated subId and a bumped
// updated_at. The reconciliation must still converge, and the recorded state
// must take the panel's rewritten values.
func TestInboundResourceUpdateReconcilesTuicPeersOnV390(t *testing.T) {
	srv := newTuicV390Server(t, []map[string]any{
		tuicCreateSerialization("a@t.com", "one"),
		tuicCreateSerialization("c@t.com", "cee"),
	})

	r := &InboundResource{client: newTestClient(t, srv.srv.URL)}
	peer := func(email, comment string) InboundTuicClientModel {
		return InboundTuicClientModel{
			Email: types.StringValue(email), ID: types.StringValue("aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"),
			Password: types.StringValue("pass-" + email), Enable: types.BoolValue(true),
			Comment: types.StringValue(comment), SubID: types.StringValue(""),
			CreatedAt: types.Int64Value(1111), UpdatedAt: types.Int64Value(2222),
		}
	}
	newPeer := InboundTuicClientModel{
		Email: types.StringValue("b@t.com"), ID: types.StringValue("bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"),
		Password: types.StringValue("pass-b"), Enable: types.BoolValue(true),
	}
	stateModel := &InboundResourceModel{
		ID: types.StringValue("7"), Remark: types.StringValue("before"),
		Enable: types.BoolValue(true), Port: types.Int64Value(25070), Protocol: types.StringValue("tuic"),
		TuicSettings: &InboundTuicSettingsModel{Clients: []InboundTuicClientModel{peer("a@t.com", "one"), peer("c@t.com", "cee")}},
	}
	planModel := &InboundResourceModel{
		ID: types.StringValue("7"), Remark: types.StringValue("after"),
		Enable: types.BoolValue(true), Port: types.Int64Value(25070), Protocol: types.StringValue("tuic"),
		TuicSettings: &InboundTuicSettingsModel{Clients: []InboundTuicClientModel{peer("a@t.com", "two"), newPeer}},
	}

	plan := inboundResourceFixture(t, r, planModel)
	state := tfsdk.State(inboundResourceFixture(t, r, stateModel))
	resp := newInboundResourceUpdateResponse(t, r)
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected error on Update: %v", resp.Diagnostics)
	}

	if got := srv.addCalls.Load(); got != 1 {
		t.Errorf("expected 1 client add (peer b), got %d", got)
	}
	if got := srv.clientUpdCalls.Load(); got != 1 {
		t.Errorf("expected 1 client update (peer a), got %d", got)
	}
	if got := srv.delCalls.Load(); got != 1 {
		t.Errorf("expected 1 client del (peer c), got %d", got)
	}
	if got := srv.setEnableCalls.Load(); got != 0 {
		t.Errorf("enable matched; expected no setEnable call, got %d", got)
	}

	var got InboundResourceModel
	resp.State.Get(context.Background(), &got)
	if got.TuicSettings == nil || len(got.TuicSettings.Clients) != 2 {
		t.Fatalf("expected 2 peers in state, got %+v", got.TuicSettings)
	}
	a := got.TuicSettings.Clients[0]
	if a.Comment.ValueString() != "two" {
		t.Errorf("expected peer a comment=two in state, got %q", a.Comment.ValueString())
	}
	if a.SubID.ValueString() != "gen-sub" {
		t.Errorf("expected the panel-generated sub_id in state, got %q", a.SubID.ValueString())
	}
	if a.UpdatedAt.ValueInt64() != 3333 {
		t.Errorf("expected the bumped updated_at in state, got %d", a.UpdatedAt.ValueInt64())
	}
	if a.CreatedAt.ValueInt64() != 1111 {
		t.Errorf("expected created_at preserved, got %d", a.CreatedAt.ValueInt64())
	}
	b := got.TuicSettings.Clients[1]
	if b.Email.ValueString() != "b@t.com" {
		t.Errorf("expected peer b in state, got %q", b.Email.ValueString())
	}
	if b.CreatedAt.ValueInt64() != 4444 {
		t.Errorf("expected the add-time created_at stamp on peer b, got %d", b.CreatedAt.ValueInt64())
	}
}
