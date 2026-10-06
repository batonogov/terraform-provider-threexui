package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientAddInbound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/inbounds/add" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.FormValue("port") != "1234" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.FormValue("protocol") != "vmess" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.FormValue("settings") != "{}" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Write(okResponse(&Inbound{ID: 1, Port: 1234, Protocol: "vmess", Settings: "{}", Enable: true}))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	inbound := &Inbound{
		Port:     1234,
		Protocol: "vmess",
		Settings: "{}",
		Enable:   true,
	}

	created, err := client.AddInbound(context.Background(), inbound)
	if err != nil {
		t.Fatalf("AddInbound failed: %v", err)
	}
	if created.ID != 1 {
		t.Fatalf("unexpected ID: %d", created.ID)
	}
}

func TestClientUpdateInbound(t *testing.T) {
	var gotID int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/panel/api/inbounds/update/10" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		gotID, _ = strconv.Atoi(r.FormValue("id"))
		w.Write(okResponse(&Inbound{ID: 10, Port: 443, Protocol: "vless", Settings: "{}", Enable: true}))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	inbound := &Inbound{
		ID:       10,
		Port:     443,
		Protocol: "vless",
		Settings: "{}",
		Enable:   true,
	}

	updated, err := client.UpdateInbound(context.Background(), inbound)
	if err != nil {
		t.Fatalf("UpdateInbound failed: %v", err)
	}
	if updated.ID != 10 || gotID != 10 {
		t.Fatalf("unexpected IDs: updated=%d form=%d", updated.ID, gotID)
	}
}

func TestClientGetInbound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/panel/api/inbounds/get/7" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(okResponse(&Inbound{ID: 7, Port: 8080, Protocol: "vmess", Settings: "{}"}))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	inbound, err := client.GetInbound(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetInbound failed: %v", err)
	}
	if inbound.ID != 7 || inbound.Port != 8080 {
		t.Fatalf("unexpected inbound: %#v", inbound)
	}
}

func TestClientDeleteInbound(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/panel/api/inbounds/del/5" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		called = true
		w.Write(okResponse(map[string]any{"id": 5}))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.DeleteInbound(context.Background(), 5); err != nil {
		t.Fatalf("DeleteInbound failed: %v", err)
	}
	if !called {
		t.Fatalf("delete endpoint not called")
	}
}

// TestInboundResourceWaitForDeletion exercises the post-Delete poll added in
// #136. The helper checks the inbound list for the absence of id; it does NOT
// re-issue DELETE (DelInbound is not idempotent in 3x-ui).
func TestInboundResourceWaitForDeletion(t *testing.T) {
	t.Run("immediate-absent", func(t *testing.T) {
		var lists int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/panel/api/inbounds/list" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			atomic.AddInt32(&lists, 1)
			w.Write(okResponse([]Inbound{{ID: 7}}))
		}))
		defer srv.Close()

		res := &InboundResource{client: newTestClient(t, srv.URL)}
		if err := res.waitForInboundDeletion(context.Background(), 9); err != nil {
			t.Fatalf("waitForInboundDeletion: %v", err)
		}
		if got := atomic.LoadInt32(&lists); got != 1 {
			t.Fatalf("expected exactly 1 list call, got %d", got)
		}
	})

	t.Run("becomes-absent-after-poll", func(t *testing.T) {
		var lists int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/panel/api/inbounds/list" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			n := atomic.AddInt32(&lists, 1)
			if n <= 2 {
				w.Write(okResponse([]Inbound{{ID: 3}}))
				return
			}
			w.Write(okResponse([]Inbound{}))
		}))
		defer srv.Close()

		res := &InboundResource{client: newTestClient(t, srv.URL)}
		if err := res.waitForInboundDeletion(context.Background(), 3); err != nil {
			t.Fatalf("waitForInboundDeletion: %v", err)
		}
		if got := atomic.LoadInt32(&lists); got != 3 {
			t.Fatalf("expected exactly 3 list calls, got %d", got)
		}
	})

	t.Run("never-absent-returns-error", func(t *testing.T) {
		var lists int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/panel/api/inbounds/list" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			atomic.AddInt32(&lists, 1)
			w.Write(okResponse([]Inbound{{ID: 1}}))
		}))
		defer srv.Close()

		res := &InboundResource{client: newTestClient(t, srv.URL)}
		if err := res.waitForInboundDeletion(context.Background(), 1); err == nil {
			t.Fatalf("expected error after exhausting attempts")
		}
		// waitForInboundDeletion uses client.ReadAfterWriteConfig() for its
		// attempt budget. In tests this is the test-injected value (3).
		expectedAttempts, _ := res.client.ReadAfterWriteConfig()
		if got := atomic.LoadInt32(&lists); got != int32(expectedAttempts) {
			t.Fatalf("expected %d list calls, got %d", expectedAttempts, got)
		}
	})

	t.Run("transient-list-error-not-treated-as-success", func(t *testing.T) {
		// A transient 5xx must not be misread as confirmed deletion.
		var lists int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/panel/api/inbounds/list" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			atomic.AddInt32(&lists, 1)
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer srv.Close()

		res := &InboundResource{client: newTestClient(t, srv.URL)}
		err := res.waitForInboundDeletion(context.Background(), 5)
		if err == nil {
			t.Fatalf("expected error when list keeps failing")
		}
	})

	t.Run("ctx-canceled-during-backoff", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/panel/api/inbounds/list" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Write(okResponse([]Inbound{{ID: 11}}))
		}))
		defer srv.Close()

		client := newTestClient(t, srv.URL)
		client.rawBackoff = 500 * time.Millisecond
		client.rawAttempts = 20
		res := &InboundResource{client: client}
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		start := time.Now()
		err := res.waitForInboundDeletion(ctx, 11)
		elapsed := time.Since(start)
		if err == nil {
			t.Fatalf("expected ctx error")
		}
		if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context error, got %v", err)
		}
		// Full attempt budget is 20 x 500ms = 10s. Cancellation must
		// short-circuit the backoff well before that.
		if elapsed > 600*time.Millisecond {
			t.Fatalf("backoff did not respect ctx cancel; elapsed=%v", elapsed)
		}
	})
}

func TestInboundToForm(t *testing.T) {
	nodeID := 42
	in := &Inbound{
		ID:             1,
		Up:             2,
		Down:           3,
		Total:          4,
		Remark:         "r",
		Enable:         true,
		ExpiryTime:     5,
		TrafficReset:   "daily",
		Listen:         "0.0.0.0",
		Port:           1234,
		Protocol:       "vmess",
		Settings:       "{}",
		StreamSettings: "{}",
		Sniffing:       "{}",
		NodeID:         &nodeID,
	}

	form := inboundToForm(in)
	want := url.Values{
		"id":                   []string{"1"},
		"up":                   []string{"2"},
		"down":                 []string{"3"},
		"total":                []string{"4"},
		"remark":               []string{"r"},
		"enable":               []string{"true"},
		"expiryTime":           []string{"5"},
		"trafficReset":         []string{"daily"},
		"lastTrafficResetTime": []string{"0"},
		"listen":               []string{"0.0.0.0"},
		"port":                 []string{"1234"},
		"protocol":             []string{"vmess"},
		"settings":             []string{"{}"},
		"streamSettings":       []string{"{}"},
		"sniffing":             []string{"{}"},
		"nodeId":               []string{"42"},
		"disableFlow":          []string{"false"},
		"trafficResetDay":      []string{"0"},
		"excludeFromSub":       []string{"false"},
	}
	if form.Encode() != want.Encode() {
		t.Fatalf("unexpected form: %s", form.Encode())
	}
}

// TestInboundToForm_SubscriptionFields is a regression test for the write path
// of the v3.3.1 multi-node subscription fields. expandInboundFromModel
// populates these on the struct, but AddInbound/UpdateInbound serialize via
// inboundToForm (url.Values) — if a field is missing there it is silently
// dropped and the panel stores its default (subSortIndex=1, shareAddr="",
// shareAddrStrategy="node"). See PR #306 review (BLOCKER).
func TestInboundToForm_SubscriptionFields(t *testing.T) {
	in := &Inbound{
		ID:                1,
		Port:              1234,
		Protocol:          "trojan",
		Settings:          "{}",
		SubSortIndex:      2,
		ShareAddr:         "203.0.113.10",
		ShareAddrStrategy: "custom",
	}

	form := inboundToForm(in)
	if got := form.Get("subSortIndex"); got != "2" {
		t.Errorf("subSortIndex not serialized: got %q, want \"2\"", got)
	}
	if got := form.Get("shareAddr"); got != "203.0.113.10" {
		t.Errorf("shareAddr not serialized: got %q, want \"203.0.113.10\"", got)
	}
	if got := form.Get("shareAddrStrategy"); got != "custom" {
		t.Errorf("shareAddrStrategy not serialized: got %q, want \"custom\"", got)
	}
}

// TestInboundToForm_TrafficResetDayAndDisableFlow guards the same write path for
// the v3.6.0 `trafficResetDay` and v3.7.0 `disableFlow` fields. Both are sent
// unconditionally — the panel's normalizeTrafficResetDay clamps anything below 1
// up to 1, so an omitted key and an explicit 0 are indistinguishable to it, and
// the attribute's Between(1, 31) validator is what keeps a 0 out of the request.
func TestInboundToForm_TrafficResetDayAndDisableFlow(t *testing.T) {
	set := inboundToForm(&Inbound{
		ID:              1,
		Port:            1234,
		Protocol:        "vless",
		Settings:        "{}",
		TrafficReset:    "monthly",
		TrafficResetDay: 15,
		DisableFlow:     true,
	})
	if got := set.Get("trafficResetDay"); got != "15" {
		t.Errorf("trafficResetDay not serialized: got %q, want \"15\"", got)
	}
	if got := set.Get("disableFlow"); got != "true" {
		t.Errorf("disableFlow not serialized: got %q, want \"true\"", got)
	}

	unset := inboundToForm(&Inbound{ID: 1, Port: 1234, Protocol: "vless", Settings: "{}"})
	if got := unset.Get("trafficResetDay"); got != "0" {
		t.Errorf("trafficResetDay should always be sent: got %q, want \"0\"", got)
	}
	if got := unset.Get("disableFlow"); got != "false" {
		t.Errorf("disableFlow should always be sent: got %q, want \"false\"", got)
	}
}
