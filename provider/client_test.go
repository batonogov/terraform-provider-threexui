package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

func okResponse(obj any) []byte {
	resp := apiResponse{Success: true}
	if obj != nil {
		b, _ := json.Marshal(obj)
		resp.Obj = b
	}
	b, _ := json.Marshal(resp)
	return b
}

func failResponse(msg string) []byte {
	b, _ := json.Marshal(apiResponse{Success: false, Msg: msg})
	return b
}

func newTestClient(t *testing.T, endpoint string) *Client {
	t.Helper()
	// Mirror the provider's default max_retries=1 so existing tests cover
	// the retry path with the same configuration users get out of the box.
	return newTestClientWithRetries(t, endpoint, 1)
}

func newTestClientWithRetries(t *testing.T, endpoint string, maxRetries int) *Client {
	t.Helper()
	c, err := NewClient(ClientConfig{
		Endpoint:               endpoint,
		BasePath:               "/",
		Username:               "admin",
		Password:               "admin",
		InsecureSkipVerify:     true,
		Timeout:                2 * time.Second,
		MaxRetries:             maxRetries,
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

func TestLoginSuccess(t *testing.T) {
	var loginCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&loginCalls, 1)
		r.ParseForm()
		if r.FormValue("username") != "admin" || r.FormValue("password") != "admin" {
			w.Write(failResponse("bad creds"))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
		w.Write(okResponse(nil))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.Login(context.Background()); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if atomic.LoadInt32(&loginCalls) != 1 {
		t.Fatalf("expected 1 login call, got %d", loginCalls)
	}
}

func TestLoginWithCSRFToken(t *testing.T) {
	var csrfCalls int32
	var loginCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/csrf-token":
			atomic.AddInt32(&csrfCalls, 1)
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "prelogin"})
			w.Write(okResponse("csrf-token"))
		case "/login":
			atomic.AddInt32(&loginCalls, 1)
			if r.Header.Get(csrfHeaderName) != "csrf-token" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if _, err := r.Cookie("3x-ui"); err != nil {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			r.ParseForm()
			if r.FormValue("_csrf") != "csrf-token" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.Write(okResponse(nil))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.Login(context.Background()); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if got := atomic.LoadInt32(&csrfCalls); got != 1 {
		t.Fatalf("expected 1 csrf call, got %d", got)
	}
	if got := atomic.LoadInt32(&loginCalls); got != 1 {
		t.Fatalf("expected 1 login call, got %d", got)
	}
}

func TestPostRefreshesCSRFTokenOn403(t *testing.T) {
	var csrfCalls int32
	var apiCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panel/csrf-token":
			atomic.AddInt32(&csrfCalls, 1)
			w.Write(okResponse("fresh-token"))
		case "/panel/api/inbounds/onlines":
			n := atomic.AddInt32(&apiCalls, 1)
			if n == 1 {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if r.Header.Get(csrfHeaderName) != "fresh-token" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.Write(okResponse([]string{"client@example.com"}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	clients, err := client.GetOnlineClients(context.Background())
	if err != nil {
		t.Fatalf("GetOnlineClients failed: %v", err)
	}
	if len(clients) != 1 || clients[0] != "client@example.com" {
		t.Fatalf("unexpected clients: %#v", clients)
	}
	if got := atomic.LoadInt32(&csrfCalls); got != 1 {
		t.Fatalf("expected 1 csrf refresh call, got %d", got)
	}
	if got := atomic.LoadInt32(&apiCalls); got != 2 {
		t.Fatalf("expected 2 api calls, got %d", got)
	}
}

func TestAutoReloginOn404(t *testing.T) {
	var loginCalls int32
	var apiCalls int32
	var authed atomic.Bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			atomic.AddInt32(&loginCalls, 1)
			authed.Store(true)
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
		case "/panel/api/server/status":
			atomic.AddInt32(&apiCalls, 1)
			if !authed.Load() {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Write(okResponse(map[string]any{"ok": true}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)

	var out map[string]any
	if err := client.doJSON(context.Background(), http.MethodGet, "panel/api/server/status", nil, &out); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if out["ok"] != true {
		t.Fatalf("unexpected response: %#v", out)
	}
	if atomic.LoadInt32(&loginCalls) != 1 {
		t.Fatalf("expected 1 login call, got %d", loginCalls)
	}
	if atomic.LoadInt32(&apiCalls) != 2 {
		t.Fatalf("expected 2 api calls (fail+retry), got %d", apiCalls)
	}
}

func TestAutoReloginOn401(t *testing.T) {
	var loginCalls int32
	var apiCalls int32
	var authed atomic.Bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			atomic.AddInt32(&loginCalls, 1)
			authed.Store(true)
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
		case "/panel/api/server/status":
			atomic.AddInt32(&apiCalls, 1)
			if !authed.Load() {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Write(okResponse(map[string]any{"ok": true}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)

	var out map[string]any
	if err := client.doJSON(context.Background(), http.MethodGet, "panel/api/server/status", nil, &out); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if out["ok"] != true {
		t.Fatalf("unexpected response: %#v", out)
	}
	if atomic.LoadInt32(&loginCalls) != 1 {
		t.Fatalf("expected 1 login call, got %d", loginCalls)
	}
	if atomic.LoadInt32(&apiCalls) != 2 {
		t.Fatalf("expected 2 api calls (fail+retry), got %d", apiCalls)
	}
}

func TestLoginFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(failResponse("wrongUsernameOrPassword"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.Login(context.Background()); err == nil {
		t.Fatalf("expected login error")
	}
}

func TestLoginWithBootstrapCredentialsUsesBootstrapFirstWithoutCSRF(t *testing.T) {
	var loginCalls int32
	var attempts []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		atomic.AddInt32(&loginCalls, 1)
		r.ParseForm()
		attempts = append(attempts, r.FormValue("username")+":"+r.FormValue("password"))
		if r.FormValue("username") == "admin" && r.FormValue("password") == "admin" {
			w.Write(okResponse(nil))
			return
		}

		w.Write(failResponse("unexpected credentials"))
	}))
	defer srv.Close()

	client, err := NewClient(ClientConfig{
		Endpoint: srv.URL,
		BasePath: "/",
		Username: "desired-admin",
		Password: "desired-pass",
		Timeout:  time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	used, err := client.LoginWithBootstrapCredentials(context.Background(), "admin", "admin")
	if err != nil {
		t.Fatalf("LoginWithBootstrapCredentials failed: %v", err)
	}
	if !used {
		t.Fatal("expected bootstrap credentials to be used")
	}
	if got := atomic.LoadInt32(&loginCalls); got != 1 {
		t.Fatalf("expected only the bootstrap login attempt, got %d", got)
	}
	if len(attempts) != 1 || attempts[0] != "admin:admin" {
		t.Fatalf("expected bootstrap credentials first without probing desired password, got %v", attempts)
	}
	if client.username != "admin" || client.password != "admin" {
		t.Fatalf("expected active client credentials to switch to bootstrap credentials")
	}
}

func TestLoginWithBootstrapCredentialsPrefersPrimaryWithCSRF(t *testing.T) {
	var loginCalls int32
	var attempts []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/csrf-token":
			w.Write(okResponse("csrf-token"))
		case "/login":
			atomic.AddInt32(&loginCalls, 1)
			r.ParseForm()
			attempts = append(attempts, r.FormValue("username")+":"+r.FormValue("password"))
			if r.Header.Get(csrfHeaderName) != "csrf-token" || r.FormValue("_csrf") != "csrf-token" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if r.FormValue("username") == "desired-admin" && r.FormValue("password") == "desired-pass" {
				w.Write(okResponse(nil))
				return
			}
			w.Write(failResponse("unexpected credentials"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client, err := NewClient(ClientConfig{
		Endpoint: srv.URL,
		BasePath: "/",
		Username: "desired-admin",
		Password: "desired-pass",
		Timeout:  time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	used, err := client.LoginWithBootstrapCredentials(context.Background(), "admin", "admin")
	if err != nil {
		t.Fatalf("LoginWithBootstrapCredentials failed: %v", err)
	}
	if used {
		t.Fatal("bootstrap credentials should not be used when primary credentials work")
	}
	if got := atomic.LoadInt32(&loginCalls); got != 1 {
		t.Fatalf("expected only the primary login attempt, got %d", got)
	}
	if len(attempts) != 1 || attempts[0] != "desired-admin:desired-pass" {
		t.Fatalf("expected primary credentials first with CSRF support, got %v", attempts)
	}
	if client.username != "desired-admin" || client.password != "desired-pass" {
		t.Fatalf("expected active client credentials to remain primary credentials")
	}
}

func TestLoginWithBootstrapCredentialsDoesNotFallbackOnHTTPError(t *testing.T) {
	var loginCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&loginCalls, 1)
		http.Error(w, "temporary failure", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	used, err := client.LoginWithBootstrapCredentials(context.Background(), "admin", "admin")
	if err == nil {
		t.Fatal("expected HTTP 500 login error")
	}
	if used {
		t.Fatal("bootstrap credentials must not be used for HTTP errors")
	}
	if got := atomic.LoadInt32(&loginCalls); got != 1 {
		t.Fatalf("expected exactly 1 login attempt, got %d", got)
	}
}

func TestLoginWithBootstrapCredentialsRestoresPrimaryWhenBootstrapFails(t *testing.T) {
	var loginCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&loginCalls, 1)
		w.Write(failResponse("wrongUsernameOrPassword"))
	}))
	defer srv.Close()

	client, err := NewClient(ClientConfig{
		Endpoint: srv.URL,
		BasePath: "/",
		Username: "desired-admin",
		Password: "desired-pass",
		Timeout:  time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}

	used, err := client.LoginWithBootstrapCredentials(context.Background(), "admin", "admin")
	if err == nil {
		t.Fatal("expected bootstrap login error")
	}
	if used {
		t.Fatal("bootstrap should not be reported as used when it failed")
	}
	if got := atomic.LoadInt32(&loginCalls); got != 2 {
		t.Fatalf("expected bootstrap+primary login attempts, got %d", got)
	}
	if client.username != "desired-admin" || client.password != "desired-pass" {
		t.Fatalf("expected primary credentials to be restored after failed bootstrap login")
	}
}

func TestUpdateInboundMaxRetriesZeroDisablesRetry(t *testing.T) {
	// Operators must be able to opt out of retry entirely (max_retries=0)
	// so a transient 5xx surfaces immediately without any silent retry.
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/inbounds/update/7" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&calls, 1)
		http.Error(w, "transient", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClientWithRetries(t, srv.URL, 0)
	if _, err := client.UpdateInbound(context.Background(), &Inbound{ID: 7}); err == nil {
		t.Fatalf("expected 5xx to surface when max_retries=0")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly 1 call (no retry when max_retries=0), got %d", got)
	}
}

func TestUpdateInboundEmitsRetryWarnLog(t *testing.T) {
	// The retry path's promise of observability is part of the contract:
	// every retry must emit a Warn entry with operation, attempt,
	// max_attempts, status_code, and backoff. Verify it actually happens.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/inbounds/update/7" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		http.Error(w, "transient", http.StatusInternalServerError)
	}))
	defer srv.Close()

	var output bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &output)

	client := newTestClient(t, srv.URL)
	if _, err := client.UpdateInbound(ctx, &Inbound{ID: 7}); err == nil {
		t.Fatalf("expected persistent 5xx to surface")
	}

	entries, err := tflogtest.MultilineJSONDecode(&output)
	if err != nil {
		t.Fatalf("MultilineJSONDecode: %v", err)
	}
	var warn map[string]any
	for _, e := range entries {
		if e["@level"] == "warn" && e["@message"] == "retrying transient 5xx" {
			warn = e
			break
		}
	}
	if warn == nil {
		t.Fatalf("expected a warn entry for the retry; got entries: %v", entries)
	}
	for _, key := range []string{"operation", "attempt", "max_attempts", "status_code", "backoff"} {
		if _, ok := warn[key]; !ok {
			t.Errorf("retry warn entry missing %q field: %v", key, warn)
		}
	}
	if op, _ := warn["operation"].(string); op != "POST panel/api/inbounds/update/7" {
		t.Errorf("unexpected operation field: %v", warn["operation"])
	}
	if code, _ := warn["status_code"].(float64); code != 500 {
		t.Errorf("unexpected status_code field: %v", warn["status_code"])
	}
}

func TestUpdateInboundMaxRetriesTwo(t *testing.T) {
	// Two retries: 500, 500, 200 → succeed on third attempt, exactly 3 calls.
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/inbounds/update/7" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		n := atomic.AddInt32(&calls, 1)
		if n <= 2 {
			http.Error(w, "transient", http.StatusInternalServerError)
			return
		}
		w.Write(okResponse(&Inbound{ID: 7}))
	}))
	defer srv.Close()

	client := newTestClientWithRetries(t, srv.URL, 2)
	if _, err := client.UpdateInbound(context.Background(), &Inbound{ID: 7}); err != nil {
		t.Fatalf("UpdateInbound after 500-500-200 failed: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 calls (initial + 2 retries), got %d", got)
	}
}

func TestUpdateInboundRetriesTransient5xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/inbounds/update/7" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			http.Error(w, "transient", http.StatusInternalServerError)
			return
		}
		w.Write(okResponse(&Inbound{ID: 7}))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if _, err := client.UpdateInbound(context.Background(), &Inbound{ID: 7}); err != nil {
		t.Fatalf("UpdateInbound after 500-then-200 failed: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 calls (1 retry on 5xx), got %d", got)
	}
}

func TestUpdateInboundSurfacesPersistent5xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "still broken", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.UpdateInbound(context.Background(), &Inbound{ID: 9})
	if err == nil {
		t.Fatalf("expected error after persistent 5xx")
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 calls (initial + 1 retry), got %d", got)
	}
}

func TestUpdateInbound4xxIsNotRetried(t *testing.T) {
	// Non-5xx errors must surface immediately — only transient server-side
	// contention is retryable.
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if _, err := client.UpdateInbound(context.Background(), &Inbound{ID: 1}); err == nil {
		t.Fatalf("expected 4xx to surface")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly 1 call, got %d", got)
	}
}

func TestGetInboundDoesNotRetryOn5xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/inbounds/get/7" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&calls, 1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if _, err := client.GetInbound(context.Background(), 7); err == nil {
		t.Fatalf("expected 5xx GET error to surface")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly 1 call (no retry on GET), got %d", got)
	}
}

func TestAddInboundDoesNotRetryOn5xx(t *testing.T) {
	// Retrying a non-idempotent create would risk creating a duplicate
	// inbound on the panel.
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/inbounds/add" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&calls, 1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if _, err := client.AddInbound(context.Background(), &Inbound{Port: 1, Protocol: "vmess", Settings: "{}"}); err == nil {
		t.Fatalf("expected 5xx error to surface")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly 1 call (no retry on Add), got %d", got)
	}
}

func TestSetInboundEnableSendsFormPayload(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/inbounds/setEnable/7" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&calls, 1)
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("expected form content type, got %q", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		if got := r.Form.Get("enable"); got != "false" {
			t.Errorf("expected enable=false form value, got %q", got)
		}
		// The panel answers the standard envelope with a nil obj.
		w.Write(okResponse(nil))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.SetInboundEnable(context.Background(), 7, false); err != nil {
		t.Fatalf("SetInboundEnable: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly 1 call, got %d", got)
	}
}

func TestSetInboundEnableRequiresID(t *testing.T) {
	client := newTestClient(t, "http://127.0.0.1:0")
	if err := client.SetInboundEnable(context.Background(), 0, true); err == nil {
		t.Fatalf("expected an error for id=0")
	}
}

func TestSetInboundEnableSurfacesFailureEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(failResponse("port already in use"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.SetInboundEnable(context.Background(), 7, true); err == nil {
		t.Fatalf("expected the failure envelope to surface as an error")
	}
}

func TestSetInboundEnableRetriesTransient5xx(t *testing.T) {
	// setEnable is idempotent server-side (SetInboundEnable early-returns
	// when the flag already matches), so the standard transient-5xx retry
	// is safe here.
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/inbounds/setEnable/7" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			http.Error(w, "transient", http.StatusInternalServerError)
			return
		}
		w.Write(okResponse(nil))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.SetInboundEnable(context.Background(), 7, true); err != nil {
		t.Fatalf("SetInboundEnable after 500-200 failed: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 calls (initial + 1 retry), got %d", got)
	}
}

func TestUpdateUserDoesNotRetryOn5xx(t *testing.T) {
	// Retrying credentials change with stale old creds (the second call
	// would still send oldUsername/oldPassword from the first attempt) is
	// unsafe and would diverge provider state from the panel.
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Probe from useSettingsAPI: return 404 → legacy paths.
		if r.URL.Path == "/panel/api/setting/all" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/panel/setting/updateUser" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&calls, 1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.UpdateUser(context.Background(), "admin", "admin", "newuser", "newpass"); err == nil {
		t.Fatalf("expected 5xx to surface")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly 1 call (no retry on UpdateUser), got %d", got)
	}
}

// TestUpdateUserSendsTwoFactorCode verifies that UpdateUser includes the
// twoFactorCode field in its payload when the provider has a 2FA code
// configured. 3x-ui v3.4.2 requires this on /setting/updateUser whenever 2FA
// is enabled; older panels ignore the extra key.
func TestUpdateUserSendsTwoFactorCode(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/panel/api/setting/all" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/panel/setting/updateUser" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		got = body
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	client.twoFactor = "123456"
	if err := client.UpdateUser(context.Background(), "admin", "admin", "newuser", "newpass"); err != nil {
		t.Fatalf("UpdateUser error: %v", err)
	}
	if got["twoFactorCode"] != "123456" {
		t.Fatalf("expected twoFactorCode=123456 in payload, got: %v", got["twoFactorCode"])
	}
	if got["newPassword"] != "newpass" {
		t.Fatalf("expected newPassword=newpass, got: %v", got["newPassword"])
	}
}

// TestUpdateUserOmitsTwoFactorCodeWhenUnset verifies the payload omits
// twoFactorCode entirely when no 2FA code is configured (backward-compatible
// for panels/users without 2FA).
func TestUpdateUserOmitsTwoFactorCodeWhenUnset(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/panel/api/setting/all" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/panel/setting/updateUser" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		got = body
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	// twoFactor is intentionally left empty.
	if err := client.UpdateUser(context.Background(), "admin", "admin", "newuser", "newpass"); err != nil {
		t.Fatalf("UpdateUser error: %v", err)
	}
	if _, has := got["twoFactorCode"]; has {
		t.Fatalf("expected twoFactorCode absent when unset, got: %v", got["twoFactorCode"])
	}
}

// TestUpdateSettingsSendsTwoFactorCodeNoMutation verifies UpdateSettings adds
// twoFactorCode to the on-the-wire payload (needed on v3.4.2 to disable 2FA via
// /setting/update) WITHOUT mutating the caller's settings map.
func TestUpdateSettingsSendsTwoFactorCodeNoMutation(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/panel/api/setting/all" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/panel/setting/update" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		got = body
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	client.twoFactor = "654321"
	settings := map[string]any{"webPort": 2053}
	if err := client.UpdateSettings(context.Background(), settings); err != nil {
		t.Fatalf("UpdateSettings error: %v", err)
	}
	// On-wire payload carries the code.
	if got["twoFactorCode"] != "654321" {
		t.Fatalf("expected twoFactorCode=654321 in payload, got: %v", got["twoFactorCode"])
	}
	if got["webPort"] != float64(2053) {
		t.Fatalf("expected webPort=2053 preserved in payload, got: %v", got["webPort"])
	}
	// Caller's map must NOT be mutated (it may feed plan/state).
	if _, polluted := settings["twoFactorCode"]; polluted {
		t.Fatalf("caller settings map was mutated with twoFactorCode: %v", settings)
	}
}

func TestUpdateInboundRetryRespectsCtxCancel(t *testing.T) {
	// Context cancellation during the backoff must short-circuit the retry,
	// not block the caller.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "transient", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClientWithRetries(t, srv.URL, 1)
	client.retryBackoff = 50 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := client.UpdateInbound(ctx, &Inbound{ID: 1}); err == nil {
		t.Fatalf("expected ctx error")
	}
	if elapsed := time.Since(start); elapsed > 40*time.Millisecond {
		t.Fatalf("backoff did not respect ctx cancel; elapsed=%v", elapsed)
	}
}

func TestUpdateInbound5xxThenReloginThenSuccess(t *testing.T) {
	// First call: 401 → re-login → still 500 (transient) → backoff → retry → 200.
	// Verifies the 5xx-retry composes with the existing 401-relogin flow.
	var n int32
	var loginCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			atomic.AddInt32(&loginCalls, 1)
			http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
			w.Write(okResponse(nil))
		case "/panel/api/inbounds/update/7":
			c := atomic.AddInt32(&n, 1)
			switch c {
			case 1:
				w.WriteHeader(http.StatusUnauthorized)
			case 2:
				http.Error(w, "transient", http.StatusInternalServerError)
			default:
				w.Write(okResponse(&Inbound{ID: 7}))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if _, err := client.UpdateInbound(context.Background(), &Inbound{ID: 7}); err != nil {
		t.Fatalf("UpdateInbound: %v", err)
	}
	if got := atomic.LoadInt32(&n); got != 3 {
		t.Fatalf("expected 3 update calls (401, 500, 200), got %d", got)
	}
	if got := atomic.LoadInt32(&loginCalls); got != 1 {
		t.Fatalf("expected 1 login call, got %d", got)
	}
}

func TestUpdateInboundClientRetriesTransient5xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/inbounds/updateClient/abc" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			http.Error(w, "transient", http.StatusInternalServerError)
			return
		}
		w.Write(okResponse(nil))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.UpdateInboundClient(context.Background(), 7, "abc", "", map[string]any{"id": "abc"}); err != nil {
		t.Fatalf("UpdateInboundClient after 500-then-200 failed: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 calls (1 retry on 5xx), got %d", got)
	}
}

func TestUpdateSettingsRetriesTransient5xx(t *testing.T) {
	// Covers doJSONRetryable: same retry shape as the form path, but
	// exercises the JSON encoding/content-type branch.
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Probe from useSettingsAPI: return 404 → legacy paths.
		if r.URL.Path == "/panel/api/setting/all" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/panel/setting/update" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			http.Error(w, "transient", http.StatusInternalServerError)
			return
		}
		w.Write(okResponse(nil))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.UpdateSettings(context.Background(), map[string]any{"webPort": 2053}); err != nil {
		t.Fatalf("UpdateSettings after 500-then-200 failed: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 calls (1 retry on 5xx), got %d", got)
	}
}

func TestDeleteInboundReturnsSuccessIfRowAbsentAfter5xx(t *testing.T) {
	// 3x-ui's DelInbound is multi-step: a panic after the SQLite row was
	// already removed surfaces as 5xx, but the row is gone. We verify with
	// GetInbounds — if the row is absent, the delete succeeded.
	var deleteCalls, listCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panel/api/inbounds/del/7":
			atomic.AddInt32(&deleteCalls, 1)
			http.Error(w, "boom", http.StatusInternalServerError)
		case "/panel/api/inbounds/list":
			atomic.AddInt32(&listCalls, 1)
			_, _ = w.Write(okResponse([]Inbound{{ID: 99}}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.DeleteInbound(context.Background(), 7); err != nil {
		t.Fatalf("expected success after verifying row absent, got: %v", err)
	}
	if got := atomic.LoadInt32(&deleteCalls); got != 1 {
		t.Fatalf("expected exactly 1 DELETE call, got %d", got)
	}
	if got := atomic.LoadInt32(&listCalls); got != 1 {
		t.Fatalf("expected exactly 1 LIST call (verification), got %d", got)
	}
}

func TestDeleteInboundRetriesOnce5xxIfRowStillPresent(t *testing.T) {
	// If the row is still present after a 5xx, retry the DELETE once. A
	// retry that succeeds returns nil; a retry that 5xxes again surfaces
	// the error.
	var deleteCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panel/api/inbounds/del/7":
			n := atomic.AddInt32(&deleteCalls, 1)
			if n == 1 {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write(okResponse(nil))
		case "/panel/api/inbounds/list":
			_, _ = w.Write(okResponse([]Inbound{{ID: 7}}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.DeleteInbound(context.Background(), 7); err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if got := atomic.LoadInt32(&deleteCalls); got != 2 {
		t.Fatalf("expected 2 DELETE calls (1 original + 1 retry), got %d", got)
	}
}

func TestDeleteInboundProceedsToRetryWhenVerifyFails(t *testing.T) {
	// If the verify call (GetInbounds) itself fails, we cannot conclude
	// "row gone", so we fall through to retrying the DELETE. The DELETE
	// retry succeeds here, so the operation succeeds.
	var deleteCalls, listCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panel/api/inbounds/del/7":
			n := atomic.AddInt32(&deleteCalls, 1)
			if n == 1 {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write(okResponse(nil))
		case "/panel/api/inbounds/list":
			atomic.AddInt32(&listCalls, 1)
			http.Error(w, "list down", http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.DeleteInbound(context.Background(), 7); err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if got := atomic.LoadInt32(&deleteCalls); got != 2 {
		t.Fatalf("expected 2 DELETE calls (original + retry), got %d", got)
	}
	if got := atomic.LoadInt32(&listCalls); got != 1 {
		t.Fatalf("expected exactly 1 LIST call (verify attempted once), got %d", got)
	}
}

func TestDeleteInboundReverifiesAfterSecond5xx(t *testing.T) {
	// Both DELETEs return 5xx, but on the second 5xx the row is now
	// gone (panic-after-commit, twice). The second verify catches this
	// and we treat the whole operation as success rather than surfacing
	// a false failure.
	var deleteCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panel/api/inbounds/del/7":
			atomic.AddInt32(&deleteCalls, 1)
			http.Error(w, "boom", http.StatusInternalServerError)
		case "/panel/api/inbounds/list":
			// First verify: row still present. Second verify: gone.
			n := atomic.LoadInt32(&deleteCalls)
			if n == 1 {
				_, _ = w.Write(okResponse([]Inbound{{ID: 7}}))
			} else {
				_, _ = w.Write(okResponse([]Inbound{}))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.DeleteInbound(context.Background(), 7); err != nil {
		t.Fatalf("expected success after second-5xx re-verify, got: %v", err)
	}
	if got := atomic.LoadInt32(&deleteCalls); got != 2 {
		t.Fatalf("expected 2 DELETE calls, got %d", got)
	}
}

func TestDeleteInboundSurfacesSecond5xxIfRowStillPresent(t *testing.T) {
	// Both DELETEs return 5xx and the row is still present after each
	// verify. The error from the retry surfaces.
	var deleteCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panel/api/inbounds/del/7":
			atomic.AddInt32(&deleteCalls, 1)
			http.Error(w, "boom", http.StatusInternalServerError)
		case "/panel/api/inbounds/list":
			_, _ = w.Write(okResponse([]Inbound{{ID: 7}}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	err := client.DeleteInbound(context.Background(), 7)
	if err == nil {
		t.Fatalf("expected 5xx to surface when row remains")
	}
	if got := atomic.LoadInt32(&deleteCalls); got != 2 {
		t.Fatalf("expected 2 DELETE calls, got %d", got)
	}
}

func TestDeleteInboundDoesNotRetryOn4xx(t *testing.T) {
	// 4xx (auth, validation) is not transient — surface immediately.
	var deleteCalls, listCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panel/api/inbounds/del/7":
			atomic.AddInt32(&deleteCalls, 1)
			http.Error(w, "bad request", http.StatusBadRequest)
		case "/panel/api/inbounds/list":
			atomic.AddInt32(&listCalls, 1)
			_, _ = w.Write(okResponse([]Inbound{}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.DeleteInbound(context.Background(), 7); err == nil {
		t.Fatalf("expected 4xx to surface")
	}
	if got := atomic.LoadInt32(&deleteCalls); got != 1 {
		t.Fatalf("expected exactly 1 DELETE call (no retry on 4xx), got %d", got)
	}
	if got := atomic.LoadInt32(&listCalls); got != 0 {
		t.Fatalf("expected no LIST call (no verification on 4xx), got %d", got)
	}
}

func TestAddInboundClientDoesNotRetryOn5xx(t *testing.T) {
	// AddClient is non-idempotent — a retry could create a duplicate
	// client on the panel.
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/inbounds/addClient" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&calls, 1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	if err := client.AddInboundClient(context.Background(), 7, map[string]any{"id": "abc"}); err == nil {
		t.Fatalf("expected 5xx to surface")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly 1 call (no retry on AddClient), got %d", got)
	}
}

func TestIsUpstreamRateLimitError(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("something else"), false},
		{errors.New("request failed: status 200, msg: Get Version (GitHub API error: API rate limit exceeded for 1.2.3.4)"), true},
		{errors.New("request failed: status 200, msg: Get Version (GitHub API error: Rate Limit exceeded)"), true},
		{errors.New("request failed: status 200, msg: connection refused"), false},
	} {
		if got := isUpstreamRateLimitError(tc.err); got != tc.want {
			t.Errorf("isUpstreamRateLimitError(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestGetXrayVersionsRetriesOnRateLimit(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/server/getXrayVersion" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.Write(failResponse("Get Version (GitHub API error: API rate limit exceeded)"))
			return
		}
		w.Write(okResponse([]string{"v26.2.6", "v26.2.5"}))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	versions, err := client.GetXrayVersions(context.Background())
	if err != nil {
		t.Fatalf("GetXrayVersions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 calls (1 rate-limit + 1 success), got %d", got)
	}
}

func TestGetXrayVersionsRetriesExhausted(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/server/getXrayVersion" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&calls, 1)
		w.Write(failResponse("Get Version (GitHub API error: API rate limit exceeded)"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.GetXrayVersions(context.Background())
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	// 1 initial + versionRetryAttempts retries
	if got := atomic.LoadInt32(&calls); got != int32(1+client.versionRetryAttempts) {
		t.Fatalf("expected %d calls, got %d", 1+client.versionRetryAttempts, got)
	}
}

func TestGetXrayVersionsNoRetryNonRateLimit(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/server/getXrayVersion" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&calls, 1)
		w.Write(failResponse("connection refused"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	_, err := client.GetXrayVersions(context.Background())
	if err == nil {
		t.Fatal("expected non-rate-limit error to surface")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected exactly 1 call (no retry), got %d", got)
	}
}

func TestGetXrayVersionsRateLimitRetryRespectsCtxCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/server/getXrayVersion" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(failResponse("Get Version (GitHub API error: API rate limit exceeded)"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	client.versionRetryBaseBackoff = 500 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := client.GetXrayVersions(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	// Context should cancel during the first or second backoff,
	// so elapsed must be well under the full retry window.
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("backoff did not respect ctx cancel; elapsed=%v", elapsed)
	}
}

func TestResolvePathWithBasePath(t *testing.T) {
	client := newTestClient(t, "http://example.com")
	client.basePath = "/xui/"

	endpoint, err := client.resolvePath("panel/api/inbounds/list")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if parsed.Path != "/xui/panel/api/inbounds/list" {
		t.Fatalf("unexpected path: %s", parsed.Path)
	}
}

func TestWithReadAfterWriteRetry_RetriesTransient5xx(t *testing.T) {
	client := newTestClient(t, "http://localhost")
	var calls int

	err := client.WithReadAfterWriteRetry(context.Background(), "test-op", func() (bool, error) {
		calls++
		if calls < 3 {
			return false, &HTTPStatusError{StatusCode: 500, Body: "internal error"}
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestWithReadAfterWriteRetry_AbortsOnNonTransientError(t *testing.T) {
	client := newTestClient(t, "http://localhost")
	var calls int

	err := client.WithReadAfterWriteRetry(context.Background(), "test-op", func() (bool, error) {
		calls++
		return false, &HTTPStatusError{StatusCode: 401, Body: "unauthorized"}
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestWithReadAfterWriteRetry_RetriesNotFoundThenFound(t *testing.T) {
	client := newTestClient(t, "http://localhost")
	var calls int

	err := client.WithReadAfterWriteRetry(context.Background(), "test-op", func() (bool, error) {
		calls++
		if calls < 2 {
			return false, nil // row not visible yet
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestWithReadAfterWriteRetry_ExhaustsBudget(t *testing.T) {
	client := newTestClient(t, "http://localhost")
	var calls int

	err := client.WithReadAfterWriteRetry(context.Background(), "test-op", func() (bool, error) {
		calls++
		return false, nil // always not found
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != client.rawAttempts {
		t.Fatalf("expected %d calls, got %d", client.rawAttempts, calls)
	}
}

func TestWithReadAfterWriteRetry_Transient5xxExhaustsBudget(t *testing.T) {
	client := newTestClient(t, "http://localhost")
	var calls int

	err := client.WithReadAfterWriteRetry(context.Background(), "test-op", func() (bool, error) {
		calls++
		return false, &HTTPStatusError{StatusCode: 502, Body: "bad gateway"}
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != client.rawAttempts {
		t.Fatalf("expected %d calls, got %d", client.rawAttempts, calls)
	}
}

func TestWithReadAfterWriteRetry_Transient5xxRespectsPreCancelledContext(t *testing.T) {
	client := newTestClient(t, "http://localhost")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var calls int
	err := client.WithReadAfterWriteRetry(ctx, "test-op", func() (bool, error) {
		calls++
		return false, &HTTPStatusError{StatusCode: 500, Body: "internal error"}
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestWithReadAfterWriteRetry_Transient5xxRespectsCtxCancelDuringBackoff(t *testing.T) {
	client := newTestClient(t, "http://localhost")
	client.rawBackoff = 500 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())

	var calls int
	errCh := make(chan error, 1)
	go func() {
		errCh <- client.WithReadAfterWriteRetry(ctx, "test-op", func() (bool, error) {
			calls++
			return false, &HTTPStatusError{StatusCode: 500, Body: "internal error"}
		})
	}()

	// Cancel during the first backoff (500ms).
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for retry to return")
	}
}

func TestWithReadAfterWriteRetry_NotFoundRespectsCancelledContext(t *testing.T) {
	client := newTestClient(t, "http://localhost")
	client.rawBackoff = 500 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())

	var calls int
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := client.WithReadAfterWriteRetry(ctx, "test-op", func() (bool, error) {
		calls++
		return false, nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWithReadAfterWriteRetry_Transient5xxOnLastAttemptReturnsErr(t *testing.T) {
	client := newTestClient(t, "http://localhost")
	var calls int

	err := client.WithReadAfterWriteRetry(context.Background(), "test-op", func() (bool, error) {
		calls++
		if calls < client.rawAttempts {
			return false, nil
		}
		return false, &HTTPStatusError{StatusCode: 500, Body: "late 5xx"}
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != client.rawAttempts {
		t.Fatalf("expected %d calls, got %d", client.rawAttempts, calls)
	}
}

// -------------------------------------------------------------------------
// Settings API fallback: v3.3.0+ 404 → legacy /panel/setting/*
// -------------------------------------------------------------------------

func TestGetSettingsFallsBackOn404(t *testing.T) {
	var legacyCalls int32
	newAPIGot404 := int32(0)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panel/api/setting/all":
			if atomic.CompareAndSwapInt32(&newAPIGot404, 0, 1) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Write(okResponse(map[string]any{"webPort": 2053}))
		case "/panel/setting/all":
			atomic.AddInt32(&legacyCalls, 1)
			w.Write(okResponse(map[string]any{"webPort": 2053}))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	// Force v3.3.0+ detection.
	client.settingsAPIMu.Lock()
	v := true
	client.settingsUnderAPI = &v
	client.settingsAPIMu.Unlock()

	result, err := client.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if result["webPort"] == nil {
		t.Fatal("expected webPort in result")
	}
	if got := atomic.LoadInt32(&legacyCalls); got != 1 {
		t.Fatalf("expected 1 legacy call, got %d", got)
	}
}

func TestUpdateSettingsFallsBackOn404(t *testing.T) {
	var newAPICalls, legacyCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panel/api/setting/update":
			atomic.AddInt32(&newAPICalls, 1)
			w.WriteHeader(http.StatusNotFound)
		case "/panel/setting/update":
			atomic.AddInt32(&legacyCalls, 1)
			w.Write(okResponse(nil))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	client.settingsAPIMu.Lock()
	v := true
	client.settingsUnderAPI = &v
	client.settingsAPIMu.Unlock()

	if err := client.UpdateSettings(context.Background(), map[string]any{"webPort": 2053}); err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}
	if got := atomic.LoadInt32(&newAPICalls); got != 1 {
		t.Fatalf("expected 1 new API call, got %d", got)
	}
	if got := atomic.LoadInt32(&legacyCalls); got != 1 {
		t.Fatalf("expected 1 legacy call, got %d", got)
	}
}

func TestSendRestartFallsBackOn404(t *testing.T) {
	var newAPICalls, legacyCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panel/api/setting/restartPanel":
			atomic.AddInt32(&newAPICalls, 1)
			w.WriteHeader(http.StatusNotFound)
		case "/panel/setting/restartPanel":
			atomic.AddInt32(&legacyCalls, 1)
			w.Write(okResponse(nil))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	client.settingsAPIMu.Lock()
	v := true
	client.settingsUnderAPI = &v
	client.settingsAPIMu.Unlock()

	if err := client.SendRestart(context.Background()); err != nil {
		t.Fatalf("SendRestart failed: %v", err)
	}
	if got := atomic.LoadInt32(&newAPICalls); got != 1 {
		t.Fatalf("expected 1 new API call, got %d", got)
	}
	if got := atomic.LoadInt32(&legacyCalls); got != 1 {
		t.Fatalf("expected 1 legacy call, got %d", got)
	}
}

func TestFallbackDoesNotTriggerOnNon404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	client.settingsAPIMu.Lock()
	v := true
	client.settingsUnderAPI = &v
	client.settingsAPIMu.Unlock()

	err := client.UpdateSettings(context.Background(), map[string]any{"webPort": 2053})
	if err == nil {
		t.Fatal("expected error from 500 response")
	}
}

func TestFallbackDoesNotTriggerWhenAlreadyLegacy(t *testing.T) {
	var legacyCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panel/setting/update":
			atomic.AddInt32(&legacyCalls, 1)
			w.Write(okResponse(nil))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	client.settingsAPIMu.Lock()
	v := false
	client.settingsUnderAPI = &v
	client.settingsAPIMu.Unlock()

	if err := client.UpdateSettings(context.Background(), map[string]any{"webPort": 2053}); err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}
	if got := atomic.LoadInt32(&legacyCalls); got != 1 {
		t.Fatalf("expected 1 legacy call, got %d", got)
	}
}
