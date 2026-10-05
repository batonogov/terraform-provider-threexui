package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// defaultRetryBackoff is the fixed delay between retry attempts on transient
// 5xx responses. It is intentionally short — a longer wait would mask real
// bugs, since this retry exists only to absorb sub-second contention spikes
// in 3x-ui's SQLite write path on older versions.
const defaultRetryBackoff = 500 * time.Millisecond

// readAfterWriteAttempts is the maximum number of times a post-write read
// will be retried while the just-written row is not yet visible. 3x-ui
// occasionally returns success from add/update endpoints before the SQLite
// commit becomes visible to a follow-up GET — see issue #157. The matrix
// acceptance test sustains write pressure for tens of seconds, and CI
// runners are slower than local hardware, so the budget is sized for the
// worst-case lag observed in CI (~10s — local lag is sub-second).
const readAfterWriteAttempts = 20

// readAfterWriteBackoff is the delay between read-after-write retry attempts.
// Same rationale as defaultRetryBackoff: long enough to absorb a typical
// SQLite contention spike, short enough not to mask real bugs.
const readAfterWriteBackoff = 500 * time.Millisecond

const csrfHeaderName = "X-CSRF-Token"

type ClientConfig struct {
	Endpoint           string
	BasePath           string
	Username           string
	Password           string
	TwoFactorCode      string
	InsecureSkipVerify bool
	Timeout            time.Duration
	// MaxRetries is the maximum number of *additional* attempts on transient
	// 5xx for idempotent write endpoints. 0 disables retry entirely; 1 (the
	// default applied by the provider) means up to one retry after a backoff.
	MaxRetries int
	// The fields below override production retry/backoff timings for tests.
	// Zero means use the production default.
	RetryBackoff           time.Duration // default: 500ms
	ReadAfterWriteAttempts int           // default: 20
	ReadAfterWriteBackoff  time.Duration // default: 500ms
	VersionRetryAttempts   int           // default: 4
	VersionRetryBackoff    time.Duration // default: 2s
}

type Client struct {
	baseURL                 *url.URL
	basePath                string
	username                string
	password                string
	twoFactor               string
	httpClient              *http.Client
	maxRetries              int
	retryBackoff            time.Duration
	rawAttempts             int
	rawBackoff              time.Duration
	versionRetryAttempts    int
	versionRetryBaseBackoff time.Duration
	authMu                  sync.Mutex
	csrfToken               string
	settingsSecretMu        sync.Mutex
	settingsSecrets         map[string]string
	newClientMu             sync.Mutex
	newClientAPI            *bool // nil=undetected, true=v3.1.0+ /panel/api/clients/*, false=old
	settingsAPIMu           sync.Mutex
	settingsUnderAPI        *bool // nil=undetected, true=v3.3.0+ /panel/api/setting/*, false=old /panel/setting/*
}

// SetBasePath updates the client's base path to match a new webBasePath.
func (c *Client) SetBasePath(p string) {
	c.basePath = normalizeBasePath(p)
}

// ReadAfterWriteConfig returns the read-after-write retry budget (attempts and
// backoff) so callers like waitForInboundDeletion can align their own polling
// loops with the same budget without hard-coding the constants.
func (c *Client) ReadAfterWriteConfig() (attempts int, backoff time.Duration) {
	return c.rawAttempts, c.rawBackoff
}

type apiResponse struct {
	Success bool            `json:"success"`
	Msg     string          `json:"msg"`
	Obj     json.RawMessage `json:"obj"`
}

type loginFailedError struct {
	message string
}

func (e *loginFailedError) Error() string {
	return e.message
}

func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}
	baseURL, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	if baseURL.Scheme == "" {
		return nil, errors.New("endpoint must include scheme (http or https)")
	}

	basePath := normalizeBasePath(cfg.BasePath)

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	// #nosec G402 -- InsecureSkipVerify is intentional: the provider manages
	// self-hosted panels that frequently use self-signed certificates. The
	// user explicitly opts in via the insecure_skip_verify provider attribute.
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}

	client := &http.Client{
		Jar:       jar,
		Timeout:   cfg.Timeout,
		Transport: transport,
	}

	maxRetries := cfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	rb := cfg.RetryBackoff
	if rb == 0 {
		rb = defaultRetryBackoff
	}
	ra := cfg.ReadAfterWriteAttempts
	if ra == 0 {
		ra = readAfterWriteAttempts
	}
	raBackoff := cfg.ReadAfterWriteBackoff
	if raBackoff == 0 {
		raBackoff = readAfterWriteBackoff
	}
	vAttempts := cfg.VersionRetryAttempts
	if vAttempts == 0 {
		vAttempts = versionRetryAttempts
	}
	vBackoff := cfg.VersionRetryBackoff
	if vBackoff == 0 {
		vBackoff = versionRetryBaseBackoff
	}

	return &Client{
		baseURL:                 baseURL,
		basePath:                basePath,
		username:                cfg.Username,
		password:                cfg.Password,
		twoFactor:               cfg.TwoFactorCode,
		httpClient:              client,
		maxRetries:              maxRetries,
		retryBackoff:            rb,
		rawAttempts:             ra,
		rawBackoff:              raBackoff,
		versionRetryAttempts:    vAttempts,
		versionRetryBaseBackoff: vBackoff,
	}, nil
}

func (c *Client) Login(ctx context.Context) error {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	return c.loginLocked(ctx)
}

type loginCredentialAttempt struct {
	username  string
	password  string
	bootstrap bool
	label     string
}

// LoginWithBootstrapCredentials tries primary and opt-in bootstrap credentials
// in the safest order for the detected panel generation. 3x-ui v2.9 logs the
// password from failed login attempts, so panels without anonymous CSRF support
// try bootstrap credentials first and avoid probing with the desired steady
// state password during fresh-panel bootstrap. 3x-ui v3 has anonymous CSRF
// bootstrap and redacted failed-login logs, so it keeps the steady-state
// primary-first behavior.
func (c *Client) LoginWithBootstrapCredentials(ctx context.Context, bootstrapUsername, bootstrapPassword string) (bool, error) {
	c.authMu.Lock()
	defer c.authMu.Unlock()

	primaryUsername := c.username
	primaryPassword := c.password

	csrfToken, err := c.fetchCSRFToken(ctx, "csrf-token", true)
	if err != nil {
		return false, err
	}
	c.csrfToken = csrfToken

	primary := loginCredentialAttempt{
		username: primaryUsername,
		password: primaryPassword,
		label:    "primary",
	}
	bootstrap := loginCredentialAttempt{
		username:  bootstrapUsername,
		password:  bootstrapPassword,
		bootstrap: true,
		label:     "bootstrap",
	}

	attempts := []loginCredentialAttempt{primary, bootstrap}
	if csrfToken == "" {
		attempts = []loginCredentialAttempt{bootstrap, primary}
	}

	usedBootstrap, err := c.loginWithCredentialOrderLocked(ctx, attempts)
	if err != nil {
		c.username = primaryUsername
		c.password = primaryPassword
		return false, err
	}

	return usedBootstrap, nil
}

func (c *Client) loginWithCredentialOrderLocked(ctx context.Context, attempts []loginCredentialAttempt) (bool, error) {
	var firstErr error
	var firstLabel string
	var lastErr error

	for _, attempt := range attempts {
		c.username = attempt.username
		c.password = attempt.password

		err := c.loginLocked(ctx)
		if err == nil {
			return attempt.bootstrap, nil
		}
		if firstErr == nil {
			firstErr = err
			firstLabel = attempt.label
		}
		lastErr = err

		var loginErr *loginFailedError
		if !errors.As(err, &loginErr) {
			return false, err
		}
	}

	if len(attempts) == 0 {
		return false, errors.New("no login credentials configured")
	}
	lastLabel := attempts[len(attempts)-1].label
	return false, fmt.Errorf("%s login failed; %s login also failed: %w", firstLabel, lastLabel, lastErr)
}

func (c *Client) loginLocked(ctx context.Context) error {
	csrfToken, err := c.fetchCSRFToken(ctx, "csrf-token", true)
	if err != nil {
		return err
	}
	c.csrfToken = csrfToken

	form := url.Values{}
	form.Set("username", c.username)
	form.Set("password", c.password)
	if c.twoFactor != "" {
		form.Set("twoFactorCode", c.twoFactor)
	}
	if csrfToken != "" {
		form.Set("_csrf", csrfToken)
	}

	endpoint, err := c.resolvePath("login")
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if csrfToken != "" {
		req.Header.Set(csrfHeaderName, csrfToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return httpStatusError(resp.StatusCode, body)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return err
	}
	if !apiResp.Success {
		if apiResp.Msg == "" {
			return &loginFailedError{message: "login failed"}
		}
		return &loginFailedError{message: apiResp.Msg}
	}
	return nil
}

func (c *Client) fetchCSRFToken(ctx context.Context, relPath string, optional bool) (string, error) {
	endpoint, err := c.resolvePath(relPath)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		if optional && (resp.StatusCode == http.StatusUnauthorized ||
			resp.StatusCode == http.StatusForbidden ||
			resp.StatusCode == http.StatusNotFound) {
			return "", nil
		}
		return "", httpStatusError(resp.StatusCode, body)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		if optional {
			return "", nil
		}
		return "", err
	}
	if !apiResp.Success {
		if optional {
			return "", nil
		}
		if apiResp.Msg == "" {
			return "", errors.New("csrf token request failed")
		}
		return "", errors.New(apiResp.Msg)
	}
	if apiResp.Obj == nil {
		return "", nil
	}

	var token string
	if err := json.Unmarshal(apiResp.Obj, &token); err != nil {
		if optional {
			return "", nil
		}
		return "", err
	}
	return token, nil
}

func (c *Client) refreshCSRFToken(ctx context.Context) (bool, error) {
	c.authMu.Lock()
	defer c.authMu.Unlock()

	for _, relPath := range []string{"panel/csrf-token", "csrf-token"} {
		token, err := c.fetchCSRFToken(ctx, relPath, true)
		if err != nil {
			return false, err
		}
		if token != "" {
			c.csrfToken = token
			return true, nil
		}
	}
	c.csrfToken = ""
	return false, nil
}

func (c *Client) doJSON(ctx context.Context, method, relPath string, body any, out any) error {
	endpoint, err := c.resolvePath(relPath)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
	}
	contentType := ""
	if body != nil {
		contentType = "application/json"
	}
	return c.doRequest(ctx, method, endpoint, contentType, buf.Bytes(), out)
}

func (c *Client) doForm(ctx context.Context, method, relPath string, form url.Values, out any) error { //nolint:unparam // method kept for API consistency with doJSON
	endpoint, err := c.resolvePath(relPath)
	if err != nil {
		return err
	}

	return c.doRequest(ctx, method, endpoint, "application/x-www-form-urlencoded", []byte(form.Encode()), out)
}

func (c *Client) AddInbound(ctx context.Context, inbound *Inbound) (*Inbound, error) {
	var out Inbound
	if err := c.doForm(ctx, http.MethodPost, "panel/api/inbounds/add", inboundToForm(inbound), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateInbound(ctx context.Context, inbound *Inbound) (*Inbound, error) {
	if inbound == nil {
		return nil, errors.New("inbound is nil")
	}
	if inbound.ID == 0 {
		return nil, errors.New("inbound id is required for update")
	}
	relPath := fmt.Sprintf("panel/api/inbounds/update/%d", inbound.ID)
	var out Inbound
	if err := c.doFormRetryable(ctx, http.MethodPost, relPath, inboundToForm(inbound), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetInboundEnable flips only the enable flag of an inbound via the
// dedicated POST /panel/api/inbounds/setEnable/:id endpoint. Since 3x-ui
// v3.9.0, UpdateInbound silently restores the stored enable flag (the posted
// value is ignored unless the request is a master→node sync —
// 3x-ui-3.9.0/internal/web/service/inbound.go:1851-1862), so an enable change
// must go through this endpoint. It exists on every supported panel (present
// in the v3.3.0 snapshot, controller/inbound.go:74), so no version gate is
// needed. The handler answers the standard envelope with a nil obj; when the
// flip requires it, the panel only arms its own deferred restart flag
// (XrayService.SetToNeedRestart) rather than restarting inline, so the
// resource's restart_xray handling is unaffected.
func (c *Client) SetInboundEnable(ctx context.Context, id int, enable bool) error {
	if id == 0 {
		return errors.New("inbound id is required for setEnable")
	}
	relPath := fmt.Sprintf("panel/api/inbounds/setEnable/%d", id)
	form := url.Values{}
	form.Set("enable", strconv.FormatBool(enable))
	return c.doFormRetryable(ctx, http.MethodPost, relPath, form, nil)
}

func (c *Client) DeleteInbound(ctx context.Context, id int) error {
	if id == 0 {
		return errors.New("inbound id is required for delete")
	}
	relPath := fmt.Sprintf("panel/api/inbounds/del/%d", id)
	op := "DELETE " + relPath

	// 3x-ui's DelInbound is multi-step (xray API call → traffic cleanup →
	// row delete). On a transient panic the handler returns 5xx after the
	// row has already been removed — a naive `withRetry` would then hit
	// the `GetInbound first, error on missing row` path inside DelInbound
	// and turn success into failure (issue #161). So on 5xx we verify
	// with GetInbounds before deciding to retry.
	err := c.doForm(ctx, http.MethodPost, relPath, url.Values{}, nil)
	if err == nil {
		return nil
	}
	code, transient := transient5xxStatus(err)
	if !transient {
		return err
	}

	// First verify-and-maybe-retry pass.
	if c.deleteVerifyAbsent(ctx, op, id, code) {
		return nil
	}

	// Row still present (or verify itself failed). Retry the DELETE
	// once. If it succeeds, we are done.
	retryErr := c.doForm(ctx, http.MethodPost, relPath, url.Values{}, nil)
	if retryErr == nil {
		return nil
	}

	// Retry also failed. If it was another 5xx, the same panic-after-
	// commit case may have just played out a second time — verify once
	// more before propagating the error.
	if retryCode, retryTransient := transient5xxStatus(retryErr); retryTransient {
		if c.deleteVerifyAbsent(ctx, op, id, retryCode) {
			return nil
		}
	}
	return retryErr
}

// deleteVerifyAbsent calls GetInbounds and reports whether the inbound is
// gone. Logs both the verify attempt and any verify failure so operators
// can distinguish "row gone" from "could not check". Returns false on a
// verify-call error — caller treats that as "still present" and proceeds
// to retry the DELETE.
func (c *Client) deleteVerifyAbsent(ctx context.Context, op string, id, statusCode int) bool {
	tflog.Warn(ctx, "verifying delete after transient 5xx", map[string]any{
		"operation":   op,
		"status_code": statusCode,
	})
	gone, verifyErr := c.inboundAbsent(ctx, id)
	if verifyErr != nil {
		tflog.Warn(ctx, "delete verification failed; will retry DELETE", map[string]any{
			"operation": op,
			"error":     verifyErr.Error(),
		})
		return false
	}
	return gone
}

// inboundAbsent reports whether the inbound with id is no longer present in
// the panel's list. A list-call error is propagated — callers must not treat
// "could not check" as "row gone".
func (c *Client) inboundAbsent(ctx context.Context, id int) (bool, error) {
	inbounds, err := c.GetInbounds(ctx)
	if err != nil {
		return false, err
	}
	for _, in := range inbounds {
		if in.ID == id {
			return false, nil
		}
	}
	return true, nil
}

func (c *Client) GetInbound(ctx context.Context, id int) (*Inbound, error) {
	if id == 0 {
		return nil, errors.New("inbound id is required for get")
	}
	relPath := fmt.Sprintf("panel/api/inbounds/get/%d", id)
	var out Inbound
	if err := c.doJSON(ctx, http.MethodGet, relPath, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetInbounds(ctx context.Context) ([]Inbound, error) {
	var out []Inbound
	if err := c.doJSON(ctx, http.MethodGet, "panel/api/inbounds/list", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetNodes lists the cluster node tree registered with the central panel
// (3x-ui multi-node surface, /panel/api/nodes). The response is a tree that
// includes transitive sub-nodes (read-only projections with Id == 0 and
// Transitive == true). Available since 3x-ui v3.0.2; there is no legacy
// fallback path, so no API-surface auto-detection is needed.
func (c *Client) GetNodes(ctx context.Context) ([]Node, error) {
	var out []Node
	if err := c.doJSON(ctx, http.MethodGet, "panel/api/nodes/list", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetNode fetches a single cluster node by its numeric id
// (GET /panel/api/nodes/:id). Used by the threexui_node resource for
// read-after-create and import. Returns ErrNotFound when the node does not
// exist (central panel responds 404 / error envelope).
func (c *Client) GetNode(ctx context.Context, id int) (*Node, error) {
	if id == 0 {
		return nil, errors.New("node id is required")
	}
	var out Node
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("panel/api/nodes/get/%d", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateNode registers a remote 3x-ui panel as a cluster node on the central
// panel (POST /panel/api/nodes/add). The central panel probes the node for
// reachability (ensureReachable) before persisting it, so the node's web API
// must be reachable from the central panel during apply. Returns the created
// node (the controller echoes back the bound model.Node).
func (c *Client) CreateNode(ctx context.Context, n *Node) (*Node, error) {
	if n == nil {
		return nil, errors.New("node is nil")
	}
	var out Node
	if err := c.doForm(ctx, http.MethodPost, "panel/api/nodes/add", nodeToForm(n), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateNode updates a cluster node's managed fields (POST
// /panel/api/nodes/update/:id). The 3x-ui controller restarts the Xray core
// itself when outbound_tag changes (controller/node.go:180-181) and runs
// ensureReachable, so the provider does NOT need to call RestartXrayService.
// The update handler returns only a status message (no object), so callers
// must re-read via GetNode to refresh observed state.
func (c *Client) UpdateNode(ctx context.Context, id int, n *Node) error {
	if n == nil {
		return errors.New("node is nil")
	}
	if id == 0 {
		return errors.New("node id is required for update")
	}
	form := nodeToForm(n)
	return c.doForm(ctx, http.MethodPost, fmt.Sprintf("panel/api/nodes/update/%d", id), form, nil)
}

// DeleteNode unregisters a cluster node from the central panel (POST
// /panel/api/nodes/del/:id — gin uses POST, not DELETE). 3x-ui refuses to
// delete a node that still owns inbounds (DB-002); the panel surfaces that as
// HTTP 200 + success:false with an "inbound(s) still attached" message, which
// doForm surfaces as a request-failed error. Callers should surface that to
// the operator so they detach/delete the inbounds first.
func (c *Client) DeleteNode(ctx context.Context, id int) error {
	if id == 0 {
		return errors.New("node id is required for delete")
	}
	return c.doForm(ctx, http.MethodPost, fmt.Sprintf("panel/api/nodes/del/%d", id), url.Values{}, nil)
}

// CreateHostGroup creates a host group on the central panel (POST
// /panel/api/hosts/add, JSON body). 3x-ui v3.5.0+ only. The server generates a
// groupId (random.NumLower(16)) when the request's GroupId is empty; the
// created host rows are echoed back. Callers should re-read via GetHostGroup
// to capture the server-generated groupId and canonical observed state.
func (c *Client) CreateHostGroup(ctx context.Context, hg *HostGroup) (*HostGroup, error) {
	if hg == nil {
		return nil, errors.New("host group is nil")
	}
	// /add returns the created host rows ([]*model.Host via jsonMsgObj), not a
	// single HostGroup. Extract the server-generated groupId from the first
	// row, then re-read the canonical HostGroup via /get/:groupId.
	var rows []struct {
		GroupId string `json:"groupId"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "panel/api/hosts/add", hg, &rows); err != nil {
		return nil, err
	}
	groupID := hg.GroupId
	if len(rows) > 0 && rows[0].GroupId != "" {
		groupID = rows[0].GroupId
	}
	if groupID == "" {
		return nil, errors.New("host group add returned no rows and no group id was provided")
	}
	return c.GetHostGroup(ctx, groupID)
}

// GetHostGroup fetches a host group by its groupId (GET
// /panel/api/hosts/get/:groupId). Returns (nil, nil) when the group does not
// exist (the panel signals this as HTTP 200 + success:false with a gorm
// "record not found" message, surfaced via isAPIRecordNotFound). 3x-ui v3.5.0+.
func (c *Client) GetHostGroup(ctx context.Context, groupID string) (*HostGroup, error) {
	if groupID == "" {
		return nil, errors.New("host group id is required")
	}
	var out HostGroup
	if err := c.doJSON(ctx, http.MethodGet, fmt.Sprintf("panel/api/hosts/get/%s", groupID), nil, &out); err != nil {
		// 3x-ui signals a missing group as HTTP 200 + success:false with a
		// "host group not found" message (the service uses .Find() + an explicit
		// common.NewError, not gorm.ErrRecordNotFound), so isAPIRecordNotFound
		// does not match. Treat it as "gone" so the resource removes from state.
		if isHostGroupNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

// isHostGroupNotFound detects the panel's "host not found" sentinel.
// Unlike nodes (gorm.ErrRecordNotFound → "record not found"), host groups use
// an explicit common.NewError, which the generic isAPIRecordNotFound does not
// match. The message changed across 3x-ui versions:
//   - v3.5.x: "host group not found"
//   - v3.6.0: "Failed to load host (host not found"
//
// Both variants are matched here.
func isHostGroupNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "host group not found") ||
		strings.Contains(msg, "host not found")
}

// UpdateHostGroup updates a host group (POST /panel/api/hosts/update/:groupId,
// JSON body). The handler does a delete-then-recreate under a transaction;
// callers must re-read via GetHostGroup to refresh observed state. 3x-ui v3.5.0+.
func (c *Client) UpdateHostGroup(ctx context.Context, groupID string, hg *HostGroup) error {
	if hg == nil {
		return errors.New("host group is nil")
	}
	if groupID == "" {
		return errors.New("host group id is required for update")
	}
	return c.doJSON(ctx, http.MethodPost, fmt.Sprintf("panel/api/hosts/update/%s", groupID), hg, nil)
}

// DeleteHostGroup deletes a host group (POST /panel/api/hosts/del/:groupId).
// gin uses POST, not DELETE. 3x-ui v3.5.0+.
func (c *Client) DeleteHostGroup(ctx context.Context, groupID string) error {
	if groupID == "" {
		return errors.New("host group id is required for delete")
	}
	return c.doJSON(ctx, http.MethodPost, fmt.Sprintf("panel/api/hosts/del/%s", groupID), nil, nil)
}

// nodeToForm builds the application/x-www-form-urlencoded body for node
// create/update, mirroring the form tags on model.Node in 3x-ui. inboundTags
// is serialized to a JSON string (gorm json serializer on the upstream side).
func nodeToForm(in *Node) url.Values {
	form := url.Values{}
	if in == nil {
		return form
	}
	form.Set("name", in.Name)
	form.Set("remark", in.Remark)
	if in.Scheme != "" {
		form.Set("scheme", in.Scheme)
	}
	form.Set("address", in.Address)
	form.Set("port", strconv.Itoa(in.Port))
	if in.BasePath != "" {
		form.Set("basePath", in.BasePath)
	}
	form.Set("apiToken", in.ApiToken)
	form.Set("enable", strconv.FormatBool(in.Enable))
	form.Set("allowPrivateAddress", strconv.FormatBool(in.AllowPrivateAddress))
	if in.TlsVerifyMode != "" {
		form.Set("tlsVerifyMode", in.TlsVerifyMode)
	}
	if in.PinnedCertSha256 != "" {
		form.Set("pinnedCertSha256", in.PinnedCertSha256)
	}
	if in.InboundSyncMode != "" {
		form.Set("inboundSyncMode", in.InboundSyncMode)
	}
	tags := in.InboundTags
	if tags == nil {
		tags = []string{}
	}
	if b, err := json.Marshal(tags); err == nil {
		form.Set("inboundTags", string(b))
	}
	if in.OutboundTag != "" {
		form.Set("outboundTag", in.OutboundTag)
	}
	return form
}

func (c *Client) AddInboundClient(ctx context.Context, inboundID int, client map[string]any) error {
	if inboundID == 0 {
		return errors.New("inbound id is required for add client")
	}
	if client == nil {
		return errors.New("client data is required")
	}

	if c.useNewClientAPI(ctx) {
		payload := map[string]any{
			"client":     client,
			"inboundIds": []int{inboundID},
		}
		err := c.doJSON(ctx, http.MethodPost, "panel/api/clients/add", payload, nil)
		if err == nil {
			return nil
		}
		if isHTTPNotFound(err) {
			c.markLegacyClientAPI()
		} else {
			return err
		}
	}

	// Old endpoint (v2.9.x, v3.0.x)
	payload := map[string]any{"clients": []map[string]any{client}}
	settings, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	form := url.Values{}
	form.Set("id", strconv.Itoa(inboundID))
	form.Set("settings", string(settings))
	return c.doForm(ctx, http.MethodPost, "panel/api/inbounds/addClient", form, nil)
}

func (c *Client) UpdateInboundClient(ctx context.Context, inboundID int, clientID string, currentEmail string, client map[string]any) error {
	if inboundID == 0 {
		return errors.New("inbound id is required for update client")
	}
	if clientID == "" {
		return errors.New("client id is required for update client")
	}
	if client == nil {
		return errors.New("client data is required")
	}

	if c.useNewClientAPI(ctx) {
		if currentEmail == "" {
			currentEmail, _ = client["email"].(string)
		}
		if currentEmail == "" {
			return errors.New("client email is required for v3.1.0+ update")
		}
		relPath := fmt.Sprintf("panel/api/clients/update/%s", url.PathEscape(currentEmail))
		err := c.doJSONRetryable(ctx, http.MethodPost, relPath, client, nil)
		if err == nil {
			return nil
		}
		if isHTTPNotFound(err) {
			c.markLegacyClientAPI()
		} else {
			return err
		}
	}

	// Old endpoint (v2.9.x, v3.0.x)
	payload := map[string]any{"clients": []map[string]any{client}}
	settings, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	form := url.Values{}
	form.Set("id", strconv.Itoa(inboundID))
	form.Set("settings", string(settings))
	relPath := fmt.Sprintf("panel/api/inbounds/updateClient/%s", clientID)
	return c.doFormRetryable(ctx, http.MethodPost, relPath, form, nil)
}

// isClientAlreadyGone reports whether a delete failed because the client was
// not there to begin with, which makes the delete a success.
//
// The message is not a fixed string: 3x-ui formats it as
// `client %q not found in any inbound or client record`
// (3x-ui-3.7.0/internal/web/service/client_crud.go:919), so the quoted email
// sits between "client" and "not found" — matching the literal phrase "client
// not found" never fired. Matching "not found" alone keeps this robust against
// the wording drifting again, and is specific enough: nothing else on this
// route reports a success-shaped failure.
func isClientAlreadyGone(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

func (c *Client) DeleteInboundClient(ctx context.Context, inboundID int, clientID string, email string) error {
	if inboundID == 0 {
		return errors.New("inbound id is required for delete client")
	}
	if clientID == "" {
		return errors.New("client id is required for delete client")
	}

	if c.useNewClientAPI(ctx) && email != "" {
		relPath := fmt.Sprintf("panel/api/clients/del/%s", url.PathEscape(email))
		err := c.doForm(ctx, http.MethodPost, relPath, url.Values{}, nil)
		if err == nil {
			return nil
		}
		if isHTTPNotFound(err) {
			c.markLegacyClientAPI()
		} else if isClientAlreadyGone(err) {
			return nil
		} else {
			return err
		}
	}

	// Old endpoint (v2.9.x, v3.0.x)
	relPath := fmt.Sprintf("panel/api/inbounds/%d/delClient/%s", inboundID, clientID)
	err := c.doForm(ctx, http.MethodPost, relPath, url.Values{}, nil)
	if err != nil && strings.Contains(err.Error(), "Client Not Found") {
		return nil
	}
	return err
}

func (c *Client) GetServerStatus(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doJSON(ctx, http.MethodGet, "panel/api/server/status", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

const (
	// versionRetryAttempts is the maximum number of retries for GetXrayVersions
	// when the upstream GitHub API rate-limits the 3x-ui panel's internal
	// cache fetch. Exponential backoff: 2s, 4s, 8s, 16s.
	versionRetryAttempts    = 4
	versionRetryBaseBackoff = 2 * time.Second
)

// isUpstreamRateLimitError reports whether err originated from 3x-ui's
// getXrayVersion handler failing due to an upstream GitHub API rate limit.
// The panel returns success:false with a message containing "rate limit".
func isUpstreamRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "rate limit")
}

func (c *Client) GetXrayVersions(ctx context.Context) ([]string, error) {
	var out []string
	var lastErr error
	backoff := c.versionRetryBaseBackoff
	for attempt := 0; attempt <= c.versionRetryAttempts; attempt++ {
		err := c.doJSON(ctx, http.MethodGet, "panel/api/server/getXrayVersion", nil, &out)
		if err == nil {
			return out, nil
		}
		lastErr = err
		if !isUpstreamRateLimitError(err) {
			return nil, err
		}
		if attempt == c.versionRetryAttempts {
			break
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		tflog.Warn(ctx, "retrying getXrayVersion after upstream rate limit", map[string]any{
			"attempt":      attempt + 1,
			"max_attempts": c.versionRetryAttempts,
			"backoff":      backoff.String(),
		})
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
		}
	}
	return nil, fmt.Errorf("getXrayVersion: upstream rate limit persisted after %d retries: %w", c.versionRetryAttempts, lastErr)
}

// ErrXrayVersionUnknown is returned when the 3x-ui API reports the Xray
// version as "Unknown", which typically means Xray is not running.
var ErrXrayVersionUnknown = errors.New("xray version is unknown (Xray may not be running)")

// GetCurrentXrayVersion returns the installed Xray version with "v" prefix.
// The server status API returns version without "v" (e.g. "26.2.6"),
// but installXray and getXrayVersion use "v"-prefixed tags (e.g. "v26.2.6").
// When Xray is not running, the API may return "Unknown" — this is treated as
// an error because the resource cannot determine the actual installed version.
func (c *Client) GetCurrentXrayVersion(ctx context.Context) (string, error) {
	status, err := c.GetServerStatus(ctx)
	if err != nil {
		return "", err
	}
	xray, ok := status["xray"].(map[string]any)
	if !ok {
		return "", errors.New("xray section not found in server status")
	}
	version, ok := xray["version"].(string)
	if !ok {
		return "", errors.New("xray version not found in server status")
	}
	if version == "Unknown" {
		return "", ErrXrayVersionUnknown
	}
	return normalizeXrayVersion(version), nil
}

// normalizeXrayVersion ensures the version string has a "v" prefix.
func normalizeXrayVersion(v string) string {
	if v != "" && v[0] != 'v' {
		return "v" + v
	}
	return v
}

func (c *Client) InstallXray(ctx context.Context, version string) error {
	return c.doJSON(ctx, http.MethodPost, "panel/api/server/installXray/"+version, nil, nil)
}

func (c *Client) GetXrayConfig(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doJSON(ctx, http.MethodGet, "panel/api/server/getConfigJson", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetOnlineClients(ctx context.Context) ([]string, error) {
	if c.useNewClientAPI(ctx) {
		var out []string
		err := c.doJSON(ctx, http.MethodPost, "panel/api/clients/onlines", nil, &out)
		if err == nil {
			if out == nil {
				out = []string{}
			}
			return out, nil
		}
		if isHTTPNotFound(err) {
			c.markLegacyClientAPI()
		} else {
			return nil, err
		}
	}

	// Old endpoint (v2.9.x, v3.0.x)
	var out []string
	if err := c.doJSON(ctx, http.MethodPost, "panel/api/inbounds/onlines", nil, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}

func (c *Client) GetClientTraffics(ctx context.Context, email string) (*ClientTraffic, error) {
	if email == "" {
		return nil, errors.New("email is required for get client traffics")
	}

	if c.useNewClientAPI(ctx) {
		relPath := fmt.Sprintf("panel/api/clients/traffic/%s", url.PathEscape(email))
		var out ClientTraffic
		err := c.doJSON(ctx, http.MethodGet, relPath, nil, &out)
		if err == nil {
			if out.Email == "" {
				return nil, fmt.Errorf("client with email %q not found", email)
			}
			return &out, nil
		}
		if isHTTPNotFound(err) {
			c.markLegacyClientAPI()
		} else {
			return nil, err
		}
	}

	// Old endpoint (v2.9.x, v3.0.x)
	relPath := fmt.Sprintf("panel/api/inbounds/getClientTraffics/%s", url.PathEscape(email))
	var out ClientTraffic
	if err := c.doJSON(ctx, http.MethodGet, relPath, nil, &out); err != nil {
		return nil, err
	}
	if out.Email == "" {
		return nil, fmt.Errorf("client with email %q not found", email)
	}
	return &out, nil
}

func (c *Client) GetNewX25519Cert(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.doJSON(ctx, http.MethodGet, "panel/api/server/getNewX25519Cert", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type VlessEncAuth struct {
	Label      string `json:"label"`
	Decryption string `json:"decryption"`
	Encryption string `json:"encryption"`
}

func (c *Client) GetNewVlessEnc(ctx context.Context) ([]VlessEncAuth, error) {
	var out struct {
		Auths []VlessEncAuth `json:"auths"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "panel/api/server/getNewVlessEnc", nil, &out); err != nil {
		return nil, err
	}
	return out.Auths, nil
}

func (c *Client) UpdateUser(ctx context.Context, oldUsername, oldPassword, newUsername, newPassword string) error {
	payload := map[string]any{
		"oldUsername": oldUsername,
		"oldPassword": oldPassword,
		"newUsername": newUsername,
		"newPassword": newPassword,
	}
	// 3x-ui v3.4.2 requires a fresh twoFactorCode on /setting/updateUser
	// whenever 2FA is enabled. Older panels ignore the extra field (gin's
	// JSON binder does not reject unknown keys), so sending it unconditionally
	// when a code is configured is backward-compatible.
	if c.twoFactor != "" {
		payload["twoFactorCode"] = c.twoFactor
	}
	err := c.withSettingsFallback(ctx, func() error {
		return c.doJSON(ctx, http.MethodPost, c.settingPath(ctx, "updateUser"), payload, nil)
	})
	if err != nil {
		return err
	}
	c.username = newUsername
	c.password = newPassword
	return nil
}

func (c *Client) GetSettings(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	err := c.withSettingsFallback(ctx, func() error {
		return c.doForm(ctx, http.MethodPost, c.settingPath(ctx, "all"), url.Values{}, &out)
	})
	return out, err
}

func (c *Client) UpdateSettings(ctx context.Context, settings map[string]any) error {
	if settings == nil {
		return errors.New("settings payload is required")
	}
	payload := settings
	// 3x-ui v3.4.2 requires a fresh twoFactorCode on /setting/update when
	// disabling 2FA (updateSettingForm embeds AllSetting + TwoFactorCode;
	// the controller only verifies the code on the disable-2FA path).
	// Older panels bind into AllSetting and ignore the extra key, so adding
	// it when a code is configured is backward-compatible. Copy to avoid
	// mutating the caller's map (which may feed plan/state).
	if c.twoFactor != "" {
		payload = make(map[string]any, len(settings)+1)
		for k, v := range settings {
			payload[k] = v
		}
		payload["twoFactorCode"] = c.twoFactor
	}
	return c.withSettingsFallback(ctx, func() error {
		return c.doJSONRetryable(ctx, http.MethodPost, c.settingPath(ctx, "update"), payload, nil)
	})
}

// SendRestart sends the restart request but does not wait for readiness.
func (c *Client) SendRestart(ctx context.Context) error {
	// The panel may close the connection mid-response, so ignore EOF.
	var firstErr error
	err := c.withSettingsFallback(ctx, func() error {
		e := c.doForm(ctx, http.MethodPost, c.settingPath(ctx, "restartPanel"), url.Values{}, nil)
		if e != nil && e.Error() == "EOF" {
			firstErr = e
			return nil
		}
		return e
	})
	if err != nil {
		return err
	}
	if firstErr != nil {
		return nil
	}
	return nil
}

// RestartXrayService restarts the Xray core via the 3x-ui API.
func (c *Client) RestartXrayService(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodPost, "panel/api/server/restartXrayService", nil, nil)
}

// WaitForReady polls the panel until it responds successfully or the context
// is cancelled. The panel needs a few seconds to come back after a restart.
func (c *Client) WaitForReady(ctx context.Context) error {
	const (
		interval = 2 * time.Second
		timeout  = 30 * time.Second
		// 3x-ui restarts its web server asynchronously after /setting/restartPanel:
		// the SIGHUP is delivered to a channel and the main loop does Stop()+Start()
		// in a goroutine (see main.go), so the old socket can briefly still answer
		// before being torn down. A single successful login is therefore racy — a
		// caller can observe "ready" on the dying socket and then hit a connection
		// reset a moment later. Require a few CONSECUTIVE successes so the restart
		// has actually settled.
		consecutiveRequired = 3
	)
	deadline := time.Now().Add(timeout)
	streak := 0
	for {
		if time.Now().After(deadline) {
			return errors.New("panel did not become ready after restart")
		}
		// Try to login — this also verifies the panel is reachable.
		if err := c.Login(ctx); err == nil {
			streak++
			if streak >= consecutiveRequired {
				return nil
			}
		} else {
			streak = 0
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

func (c *Client) GetXrayTemplate(ctx context.Context) (map[string]any, error) {
	var raw string
	err := c.withSettingsFallback(ctx, func() error {
		return c.doForm(ctx, http.MethodPost, c.xraySettingPath(ctx, ""), url.Values{}, &raw)
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	var payload struct {
		XraySetting any `json:"xraySetting"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	if payload.XraySetting == nil {
		return map[string]any{}, nil
	}
	settings, ok := payload.XraySetting.(map[string]any)
	if !ok {
		return nil, errors.New("xraySetting is not an object")
	}
	return settings, nil
}

func (c *Client) UpdateXrayTemplate(ctx context.Context, settings map[string]any) error {
	if settings == nil {
		return errors.New("xraySetting payload is required")
	}
	// Preserve outboundTestUrl so it is not reset by 3x-ui when only
	// the xray template is being updated.
	testURL, err := c.GetXrayOutboundTestURL(ctx)
	if err != nil {
		return fmt.Errorf("failed to read outboundTestUrl: %w", err)
	}

	payload, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	form := url.Values{}
	form.Set("xraySetting", string(payload))
	if testURL != "" {
		form.Set("outboundTestUrl", testURL)
	}
	return c.withSettingsFallback(ctx, func() error {
		return c.doFormRetryable(ctx, http.MethodPost, c.xraySettingPath(ctx, "update"), form, nil)
	})
}

func (c *Client) GetXrayOutboundTestURL(ctx context.Context) (string, error) {
	var raw string
	err := c.withSettingsFallback(ctx, func() error {
		return c.doForm(ctx, http.MethodPost, c.xraySettingPath(ctx, ""), url.Values{}, &raw)
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	var payload struct {
		OutboundTestURL string `json:"outboundTestUrl"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", err
	}
	return payload.OutboundTestURL, nil
}

func (c *Client) SetXrayOutboundTestURL(ctx context.Context, testURL string) error {
	// Read current xray template to preserve it.
	tmpl, err := c.GetXrayTemplate(ctx)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(tmpl)
	if err != nil {
		return err
	}
	form := url.Values{}
	form.Set("xraySetting", string(payload))
	form.Set("outboundTestUrl", testURL)
	return c.withSettingsFallback(ctx, func() error {
		return c.doFormRetryable(ctx, http.MethodPost, c.xraySettingPath(ctx, "update"), form, nil)
	})
}

func (c *Client) doRequest(ctx context.Context, method, endpoint, contentType string, body []byte, out any) error {
	resp, err := c.doRequestOnce(ctx, method, endpoint, contentType, body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusForbidden && requiresCSRF(method) {
		// #nosec G104 -- discarding body before retry; Close error is not actionable
		resp.Body.Close()
		refreshed, refreshErr := c.refreshCSRFToken(ctx)
		if refreshErr != nil {
			return refreshErr
		}
		if !refreshed {
			return &HTTPStatusError{StatusCode: http.StatusForbidden}
		}
		resp, err = c.doRequestOnce(ctx, method, endpoint, contentType, body)
		if err != nil {
			return err
		}
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound {
		// #nosec G104 -- discarding body before re-login; Close error is not actionable
		resp.Body.Close()
		if err := c.Login(ctx); err != nil {
			return err
		}
		resp, err = c.doRequestOnce(ctx, method, endpoint, contentType, body)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusForbidden && requiresCSRF(method) {
			// #nosec G104 -- discarding body before retry; Close error is not actionable
			resp.Body.Close()
			refreshed, refreshErr := c.refreshCSRFToken(ctx)
			if refreshErr != nil {
				return refreshErr
			}
			if !refreshed {
				return &HTTPStatusError{StatusCode: http.StatusForbidden}
			}
			resp, err = c.doRequestOnce(ctx, method, endpoint, contentType, body)
			if err != nil {
				return err
			}
		}
	}
	defer resp.Body.Close()

	return decodeAPIResponse(resp, out)
}

// HTTPStatusError carries the HTTP status code from an upstream response so
// callers can distinguish transient 5xx (retryable) from 4xx (do not retry).
type HTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *HTTPStatusError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("request failed: status %d", e.StatusCode)
	}
	return fmt.Sprintf("request failed: status %d, body: %s", e.StatusCode, e.Body)
}

func httpStatusError(statusCode int, body []byte) error {
	msg := strings.TrimSpace(string(body))
	if len(msg) > 1024 {
		msg = msg[:1024] + "...(truncated)"
	}
	return &HTTPStatusError{StatusCode: statusCode, Body: msg}
}

// transient5xxStatus reports whether err is an *HTTPStatusError with a 5xx
// code, returning the code alongside the boolean for callers that want to
// log it without re-running errors.As. 5xx on a write endpoint typically
// originates from gin's recovery middleware after a panic in the handler —
// the panel's controllers otherwise return 200 with success:false. A panic
// during a SQLite transaction is the canonical "transient" failure this
// retry targets.
func transient5xxStatus(err error) (int, bool) {
	var httpErr *HTTPStatusError
	if !errors.As(err, &httpErr) {
		return 0, false
	}
	if httpErr.StatusCode < 500 || httpErr.StatusCode >= 600 {
		return httpErr.StatusCode, false
	}
	return httpErr.StatusCode, true
}

// withRetry runs fn up to (1 + c.maxRetries) times, retrying only on a
// transient 5xx response from an idempotent endpoint. It is the single
// retry policy in the client; doFormRetryable / doJSONRetryable wrap it
// so callers do not have to construct logging fields themselves.
//
// Not safe for non-idempotent endpoints: AddInbound (would create a
// duplicate), AddInboundClient (duplicate), UpdateUser (the second call
// would run with stale credentials and could leave provider state and
// panel state out of sync). DeleteInbound has its own retry-with-verify
// path — see DeleteInbound — because a naive retry would turn a
// successful-but-5xx delete into a failure (DelInbound errors on a
// missing row).
//
// Retries are visible: every retry emits a tflog.Warn so operators can
// detect upstream flakiness instead of having it silently absorbed.
func (c *Client) withRetry(ctx context.Context, op string, fn func() error) error {
	var err error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}
		statusCode, retryable := transient5xxStatus(err)
		if !retryable || attempt == c.maxRetries {
			return err
		}
		// Honor cancellation before logging or sleeping so a cancelled
		// context does not produce a "retrying" entry that we never act on.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		tflog.Warn(ctx, "retrying transient 5xx", map[string]any{
			"operation":    op,
			"attempt":      attempt + 1,
			"max_attempts": c.maxRetries,
			"status_code":  statusCode,
			"backoff":      c.retryBackoff.String(),
		})
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.retryBackoff):
		}
	}
	return err
}

func (c *Client) doFormRetryable(ctx context.Context, method, relPath string, form url.Values, out any) error { //nolint:unparam // method kept for API symmetry with doJSONRetryable / doForm
	return c.withRetry(ctx, method+" "+relPath, func() error {
		return c.doForm(ctx, method, relPath, form, out)
	})
}

func (c *Client) doJSONRetryable(ctx context.Context, method, relPath string, body any, out any) error {
	return c.withRetry(ctx, method+" "+relPath, func() error {
		return c.doJSON(ctx, method, relPath, body, out)
	})
}

// WithReadAfterWriteRetry polls fn until the row is visible (found=true) or
// the budget is exhausted. It handles two transient conditions:
//
//   - found=false: the write succeeded but SQLite hasn't made the row visible yet.
//   - err is a transient 5xx: the panel is temporarily unavailable under load.
//
// Non-transient errors (4xx, auth failures, etc.) abort immediately.
func (c *Client) WithReadAfterWriteRetry(ctx context.Context, opName string, fn func() (bool, error)) error {
	for attempt := 0; attempt < c.rawAttempts; attempt++ {
		found, err := fn()
		if err != nil {
			if _, retryable := transient5xxStatus(err); !retryable || attempt == c.rawAttempts-1 {
				return err
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			tflog.Warn(ctx, "retrying read-after-write (transient error)", map[string]any{
				"operation":    opName,
				"attempt":      attempt + 1,
				"max_attempts": c.rawAttempts,
				"backoff":      c.rawBackoff.String(),
			})
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.rawBackoff):
			}
			continue
		}
		if found {
			return nil
		}
		if attempt == c.rawAttempts-1 {
			break
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		tflog.Warn(ctx, "retrying read-after-write", map[string]any{
			"operation":    opName,
			"attempt":      attempt + 1,
			"max_attempts": c.rawAttempts,
			"backoff":      c.rawBackoff.String(),
		})
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.rawBackoff):
		}
	}
	return fmt.Errorf("%s: row not visible after %d attempts (%s total)", opName, c.rawAttempts, c.rawBackoff*time.Duration(c.rawAttempts-1))
}

func (c *Client) doRequestOnce(ctx context.Context, method, endpoint, contentType string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if requiresCSRF(method) {
		c.authMu.Lock()
		token := c.csrfToken
		c.authMu.Unlock()
		if token != "" {
			req.Header.Set(csrfHeaderName, token)
		}
	}
	return c.httpClient.Do(req)
}

func requiresCSRF(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}

func decodeAPIResponse(resp *http.Response, out any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		if resp.StatusCode >= 400 {
			return &HTTPStatusError{StatusCode: resp.StatusCode}
		}
		return nil
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		if resp.StatusCode >= 400 {
			return httpStatusError(resp.StatusCode, body)
		}
		return err
	}
	if !apiResp.Success {
		if apiResp.Msg == "" {
			return fmt.Errorf("request failed: status %d", resp.StatusCode)
		}
		return fmt.Errorf("request failed: status %d, msg: %s", resp.StatusCode, apiResp.Msg)
	}

	if out == nil || apiResp.Obj == nil {
		return nil
	}

	return json.Unmarshal(apiResp.Obj, out)
}

func (c *Client) resolvePath(rel string) (string, error) {
	if rel == "" {
		return "", errors.New("empty path")
	}

	base := *c.baseURL
	basePath := strings.TrimSuffix(base.Path, "/")
	merged := basePath + c.basePath
	if !strings.HasSuffix(merged, "/") {
		merged += "/"
	}
	merged += strings.TrimPrefix(rel, "/")
	base.Path = merged
	return base.String(), nil
}

func normalizeBasePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

func inboundToForm(in *Inbound) url.Values {
	form := url.Values{}
	if in == nil {
		return form
	}
	form.Set("id", strconv.Itoa(in.ID))
	form.Set("up", strconv.FormatInt(in.Up, 10))
	form.Set("down", strconv.FormatInt(in.Down, 10))
	form.Set("total", strconv.FormatInt(in.Total, 10))
	form.Set("remark", in.Remark)
	form.Set("enable", strconv.FormatBool(in.Enable))
	form.Set("expiryTime", strconv.FormatInt(in.ExpiryTime, 10))
	form.Set("trafficReset", in.TrafficReset)
	form.Set("lastTrafficResetTime", strconv.FormatInt(in.LastTrafficResetTime, 10))
	form.Set("listen", in.Listen)
	form.Set("port", strconv.Itoa(in.Port))
	form.Set("protocol", in.Protocol)
	form.Set("settings", in.Settings)
	form.Set("streamSettings", in.StreamSettings)
	form.Set("sniffing", in.Sniffing)
	if in.NodeID != nil {
		form.Set("nodeId", strconv.Itoa(*in.NodeID))
	}
	if in.SubSortIndex > 0 {
		form.Set("subSortIndex", strconv.Itoa(in.SubSortIndex))
	}
	if in.ShareAddr != "" {
		form.Set("shareAddr", in.ShareAddr)
	}
	if in.ShareAddrStrategy != "" {
		form.Set("shareAddrStrategy", in.ShareAddrStrategy)
	}
	// trafficResetDay (v3.6.0+) and disableFlow (v3.7.0+) are unknown form keys
	// on older panels, where gin silently ignores them. Both are sent
	// unconditionally: gin binds an absent key to the struct zero value anyway,
	// and AddInbound/UpdateInbound then run normalizeTrafficResetDay, which
	// clamps anything below 1 up to 1 — so omitting the key and posting a 0 are
	// indistinguishable to the panel. The attribute's Between(1, 31) validator
	// is what keeps a non-round-trippable 0 out of the request.
	form.Set("trafficResetDay", strconv.Itoa(in.TrafficResetDay))
	form.Set("disableFlow", strconv.FormatBool(in.DisableFlow))
	// excludeFromSub (v3.9.0+) is an unknown form key on older panels, where
	// gin silently ignores it — same contract as trafficResetDay/disableFlow
	// above. Sent unconditionally so the value round-trips on v3.9.0+.
	form.Set("excludeFromSub", strconv.FormatBool(in.ExcludeFromSub))
	return form
}

// useNewClientAPI returns true if the v3.1.0+ /panel/api/clients/* surface
// should be tried. On the first call it probes the panel to detect the
// available API surface. Subsequent calls use the cached result.
// The probe goes through doRequest so it benefits from auto re-login
// on expired sessions (a raw HTTP GET would get 404 on unauthenticated
// v3.1.0+ and incorrectly mark the API as old).
func (c *Client) useNewClientAPI(ctx context.Context) bool {
	c.newClientMu.Lock()
	defer c.newClientMu.Unlock()
	if c.newClientAPI != nil {
		return *c.newClientAPI
	}

	var out json.RawMessage
	err := c.doJSON(ctx, http.MethodGet, "panel/api/clients/list", nil, &out)
	isNew := err == nil
	c.newClientAPI = &isNew
	if isNew {
		tflog.Info(ctx, "detected 3x-ui v3.1.0+ client API surface")
	} else {
		tflog.Info(ctx, "detected 3x-ui v2.9.x/v3.0.x client API surface")
	}
	return isNew
}

func (c *Client) markLegacyClientAPI() {
	v := false
	c.newClientMu.Lock()
	c.newClientAPI = &v
	c.newClientMu.Unlock()
}

// useSettingsAPI returns true if the v3.3.0+ /panel/api/setting/* surface
// should be used. On the first call it probes the panel to detect the
// available API surface. Subsequent calls use the cached result.
func (c *Client) useSettingsAPI(ctx context.Context) bool {
	c.settingsAPIMu.Lock()
	defer c.settingsAPIMu.Unlock()
	if c.settingsUnderAPI != nil {
		return *c.settingsUnderAPI
	}

	var out json.RawMessage
	err := c.doForm(ctx, http.MethodPost, "panel/api/setting/all", url.Values{}, &out)
	isNew := err == nil
	c.settingsUnderAPI = &isNew
	if isNew {
		tflog.Info(ctx, "detected 3x-ui v3.3.0+ settings API surface (/panel/api/setting/*)")
	} else {
		tflog.Info(ctx, "detected pre-v3.3.0 settings API surface (/panel/setting/*)")
	}
	return isNew
}

func (c *Client) markLegacySettingsAPI(ctx context.Context) {
	v := false
	c.settingsAPIMu.Lock()
	c.settingsUnderAPI = &v
	c.settingsAPIMu.Unlock()
	tflog.Warn(ctx, "settings API returned 404, falling back to pre-v3.3.0 API surface (/panel/setting/*)")
}

// withSettingsFallback executes fn. If the v3.3.0+ API was detected but the
// request returns 404, it marks the API as legacy and retries once.
func (c *Client) withSettingsFallback(ctx context.Context, fn func() error) error {
	err := fn()
	if err == nil || !isHTTPNotFound(err) || !c.useSettingsAPI(ctx) {
		return err
	}
	c.markLegacySettingsAPI(ctx)
	return fn()
}

// settingPath returns the correct API path for a settings endpoint based on
// the detected 3x-ui version.
func (c *Client) settingPath(ctx context.Context, suffix string) string {
	if c.useSettingsAPI(ctx) {
		return "panel/api/setting/" + suffix
	}
	return "panel/setting/" + suffix
}

// xraySettingPath returns the correct API path for an xray settings endpoint
// based on the detected 3x-ui version.
func (c *Client) xraySettingPath(ctx context.Context, suffix string) string {
	prefix := "panel/xray"
	if c.useSettingsAPI(ctx) {
		prefix = "panel/api/xray"
	}
	if suffix == "" {
		return prefix
	}
	return prefix + "/" + suffix
}

// isHTTPNotFound reports whether err originated from an HTTP 404 response.
func isHTTPNotFound(err error) bool {
	var httpErr *HTTPStatusError
	if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
		return true
	}
	return false
}

// isAPIRecordNotFound reports whether the panel reported a missing record.
// 3x-ui handlers wrap gorm errors into an HTTP 200 success:false envelope
// (util.go jsonMsgObj), so a missing node surfaces as a request-failed error
// whose message contains the gorm "record not found" text — not as HTTP 404.
func isAPIRecordNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "record not found") || strings.Contains(msg, "no rows")
}
