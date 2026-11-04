package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ---------------------------------------------------------------------------
// Fake panel: the sub-balancer controller and service of 3x-ui v3.7.0-v3.11.0
// (internal/web/controller/sub_balancer.go, internal/web/service/sub_balancer.go).
// ---------------------------------------------------------------------------

type fakeSubBalancerRow struct {
	ID            int
	Remark        string
	Strategy      string
	InboundIDs    []int
	MemberWeights map[int]float64
	SortOrder     int
	Enabled       bool
}

type fakeSubBalancerRequest struct {
	Method      string
	Path        string
	ContentType string
	Form        url.Values
}

type fakeSubBalancerPanel struct {
	mu   sync.Mutex
	rows map[int]*fakeSubBalancerRow
	next int
	// noAPI models v3.3.0-v3.6.0, which have no /panel/api/sub-balancers routes.
	noAPI bool
	// noWeights models v3.7.x: the controller does not read memberWeights.
	noWeights  bool
	failDelete bool
	requests   []fakeSubBalancerRequest
}

func newFakeSubBalancerPanel(t *testing.T, panel *fakeSubBalancerPanel) (*fakeSubBalancerPanel, *Client) {
	t.Helper()
	panel.rows = map[int]*fakeSubBalancerRow{}
	srv := httptest.NewServer(panel)
	t.Cleanup(srv.Close)
	return panel, newTestClient(t, srv.URL)
}

func (p *fakeSubBalancerPanel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/login" {
		http.SetCookie(w, &http.Cookie{Name: "3x-ui", Value: "sess"})
		_, _ = w.Write(okResponse(nil))
		return
	}
	rest, ok := strings.CutPrefix(r.URL.Path, "/panel/api/sub-balancers")
	if !ok || p.noAPI {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.requests = append(p.requests, fakeSubBalancerRequest{
		Method: r.Method, Path: r.URL.Path, ContentType: r.Header.Get("Content-Type"), Form: r.PostForm,
	})

	var obj any
	var err error
	switch {
	case rest == "" && r.Method == http.MethodGet:
		obj = p.list()
	case rest == "" && r.Method == http.MethodPost:
		obj, err = p.create(r.PostForm)
	case strings.HasSuffix(rest, "/del") && r.Method == http.MethodPost:
		obj, err = "", p.del(strings.TrimSuffix(strings.TrimPrefix(rest, "/"), "/del"))
	case r.Method == http.MethodPost:
		obj, err = p.update(strings.TrimPrefix(rest, "/"), r.PostForm)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		_, _ = w.Write(failResponse("Failed to save subscription balancer (" + err.Error() + ")"))
		return
	}
	_, _ = w.Write(okResponse(obj))
}

func (p *fakeSubBalancerPanel) parse(form url.Values) (*fakeSubBalancerRow, *bool, error) {
	row := &fakeSubBalancerRow{Remark: form.Get("remark"), Strategy: form.Get("strategy")}
	if raw := form.Get("sortOrder"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return nil, nil, err
		}
		row.SortOrder = v
	}
	var enabled *bool
	if values, ok := form["enabled"]; ok {
		v, err := strconv.ParseBool(values[0])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid enabled %q: %w", values[0], err)
		}
		enabled = &v
	}
	for _, raw := range form["inboundIds"] {
		id, err := strconv.Atoi(raw)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid inbound id %q: %w", raw, err)
		}
		row.InboundIDs = append(row.InboundIDs, id)
	}
	if values, ok := form["memberWeights"]; ok && !p.noWeights && strings.TrimSpace(values[0]) != "" {
		weights := map[int]float64{}
		if err := json.Unmarshal([]byte(values[0]), &weights); err != nil {
			return nil, nil, fmt.Errorf("invalid memberWeights %q: %w", values[0], err)
		}
		row.MemberWeights = weights
	}
	return row, enabled, nil
}

func (p *fakeSubBalancerPanel) validate(row *fakeSubBalancerRow) error {
	row.Remark = strings.TrimSpace(row.Remark)
	if row.Remark == "" {
		return fmt.Errorf("balancer remark is required")
	}
	if len(row.Remark) > 256 {
		return fmt.Errorf("balancer remark too long (max 256)")
	}
	if row.Strategy == "" {
		row.Strategy = "random"
	}
	if !slices.Contains([]string{"leastLoad", "leastPing", "random", "roundRobin"}, row.Strategy) {
		return fmt.Errorf("invalid balancer strategy: %s", row.Strategy)
	}
	if len(row.InboundIDs) == 0 {
		return fmt.Errorf("balancer must select at least one inbound")
	}
	if len(row.MemberWeights) > 0 {
		if row.Strategy != "leastLoad" {
			return fmt.Errorf("balancer weights only apply to the leastLoad strategy")
		}
		cleaned := map[int]float64{}
		for id, weight := range row.MemberWeights {
			if !slices.Contains(row.InboundIDs, id) {
				continue
			}
			if weight <= 0 || weight > math.MaxFloat32 || weight < math.SmallestNonzeroFloat32 {
				return fmt.Errorf("balancer member weights must be a positive float32 value")
			}
			cleaned[id] = weight
		}
		row.MemberWeights = cleaned
	}
	if len(row.MemberWeights) == 0 {
		row.MemberWeights = nil
	}
	if row.SortOrder < 1 {
		row.SortOrder = 1
	}
	return nil
}

func (p *fakeSubBalancerPanel) wire(row *fakeSubBalancerRow) map[string]any {
	out := map[string]any{
		"id": row.ID, "remark": row.Remark, "strategy": row.Strategy, "inboundIds": row.InboundIDs,
		"sortOrder": row.SortOrder, "enabled": row.Enabled,
		"createdAt": int64(1710000000000), "updatedAt": int64(1710000000000),
	}
	// memberWeights is json:"memberWeights,omitempty" and absent from v3.7.x.
	if len(row.MemberWeights) > 0 && !p.noWeights {
		out["memberWeights"] = row.MemberWeights
	}
	return out
}

func (p *fakeSubBalancerPanel) list() []map[string]any {
	rows := slices.Collect(maps.Values(p.rows))
	slices.SortFunc(rows, func(a, b *fakeSubBalancerRow) int {
		if a.SortOrder != b.SortOrder {
			return a.SortOrder - b.SortOrder
		}
		return a.ID - b.ID
	})
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, p.wire(row))
	}
	return out
}

func (p *fakeSubBalancerPanel) create(form url.Values) (any, error) {
	row, enabled, err := p.parse(form)
	if err != nil {
		return nil, err
	}
	row.Enabled = enabled == nil || *enabled
	if err := p.validate(row); err != nil {
		return nil, err
	}
	p.next++
	row.ID = p.next
	p.rows[row.ID] = row
	return p.wire(row), nil
}

func (p *fakeSubBalancerPanel) update(rawID string, form url.Values) (any, error) {
	id, err := strconv.Atoi(rawID)
	if err != nil || id < 1 {
		return nil, fmt.Errorf("invalid id %q", rawID)
	}
	row, enabled, err := p.parse(form)
	if err != nil {
		return nil, err
	}
	if err := p.validate(row); err != nil {
		return nil, err
	}
	current, ok := p.rows[id]
	if !ok {
		return nil, fmt.Errorf("record not found")
	}
	current.Remark, current.Strategy, current.InboundIDs = row.Remark, row.Strategy, row.InboundIDs
	current.MemberWeights, current.SortOrder = row.MemberWeights, row.SortOrder
	if enabled != nil {
		current.Enabled = *enabled
	}
	return p.wire(current), nil
}

func (p *fakeSubBalancerPanel) del(rawID string) error {
	if p.failDelete {
		return fmt.Errorf("database is locked")
	}
	id, _ := strconv.Atoi(rawID)
	if _, ok := p.rows[id]; !ok {
		return fmt.Errorf("sub balancer not found")
	}
	delete(p.rows, id)
	return nil
}

// deleteInbound mirrors InboundService.DelInbound: the id leaves inboundIds of
// every balancer while memberWeights stay untouched (internal/web/service/inbound.go).
func (p *fakeSubBalancerPanel) deleteInbound(inboundID int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, row := range p.rows {
		row.InboundIDs = slices.DeleteFunc(row.InboundIDs, func(id int) bool { return id == inboundID })
	}
}

func (p *fakeSubBalancerPanel) row(t *testing.T, id int) *fakeSubBalancerRow {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	row, ok := p.rows[id]
	if !ok {
		t.Fatalf("panel has no balancer %d", id)
	}
	return row
}

func (p *fakeSubBalancerPanel) lastRequest(t *testing.T) fakeSubBalancerRequest {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.requests) == 0 {
		t.Fatal("panel received no sub-balancer request")
	}
	return p.requests[len(p.requests)-1]
}

// ---------------------------------------------------------------------------
// Terraform value helpers
// ---------------------------------------------------------------------------

func subBalancerObjectType() tftypes.Object {
	return subBalancerResourceSchema().Type().TerraformType(context.Background()).(tftypes.Object)
}

func tfSubBalancerValue(attrs map[string]tftypes.Value) tftypes.Value {
	objType := subBalancerObjectType()
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		vals[name] = tftypes.NewValue(typ, nil)
	}
	maps.Copy(vals, attrs)
	return tftypes.NewValue(objType, vals)
}

func tfInt64Set(ids ...int64) tftypes.Value {
	elems := make([]tftypes.Value, 0, len(ids))
	for _, id := range ids {
		elems = append(elems, tftypes.NewValue(tftypes.Number, id))
	}
	return tftypes.NewValue(tftypes.Set{ElementType: tftypes.Number}, elems)
}

func tfWeights(weights map[string]any) tftypes.Value {
	elems := make(map[string]tftypes.Value, len(weights))
	for key, weight := range weights {
		elems[key] = tftypes.NewValue(tftypes.Number, weight)
	}
	return tftypes.NewValue(tftypes.Map{ElementType: tftypes.Number}, elems)
}

func tfUnknown(typ tftypes.Type) tftypes.Value {
	return tftypes.NewValue(typ, tftypes.UnknownValue)
}

// oneThird is what HCL computes for 1/3: a 512-bit number with far more digits
// than a float64 holds.
func oneThird() *big.Float {
	x := new(big.Float).SetPrec(512).SetInt64(1)
	return x.Quo(x, new(big.Float).SetPrec(512).SetInt64(3))
}

// subBalancerPlanned returns the planned values of a typical balancer, defaults
// included, overridden by attrs.
func subBalancerPlanned(attrs map[string]tftypes.Value) map[string]tftypes.Value {
	planned := map[string]tftypes.Value{
		"id":          tfUnknown(tftypes.String),
		"remark":      tftypes.NewValue(tftypes.String, "auto"),
		"strategy":    tftypes.NewValue(tftypes.String, "random"),
		"inbound_ids": tfInt64Set(3, 1),
		"sort_order":  tftypes.NewValue(tftypes.Number, 1),
		"enabled":     tftypes.NewValue(tftypes.Bool, true),
	}
	maps.Copy(planned, attrs)
	return planned
}

func subBalancerPlan(attrs map[string]tftypes.Value) tfsdk.Plan {
	return tfsdk.Plan{Schema: subBalancerResourceSchema(), Raw: tfSubBalancerValue(attrs)}
}

func subBalancerState(attrs map[string]tftypes.Value) tfsdk.State {
	return tfsdk.State{Schema: subBalancerResourceSchema(), Raw: tfSubBalancerValue(attrs)}
}

func nullSubBalancerState() tfsdk.State {
	return tfsdk.State{Schema: subBalancerResourceSchema(), Raw: tftypes.NewValue(subBalancerObjectType(), nil)}
}

func subBalancerModelFromState(t *testing.T, state tfsdk.State) SubBalancerResourceModel {
	t.Helper()
	var m SubBalancerResourceModel
	if diags := state.Get(context.Background(), &m); diags.HasError() {
		t.Fatalf("decoding state: %v", diags)
	}
	return m
}

func setInt64s(t *testing.T, set types.Set) []int64 {
	t.Helper()
	var ids []int64
	if diags := set.ElementsAs(context.Background(), &ids, false); diags.HasError() {
		t.Fatalf("decoding set: %v", diags)
	}
	slices.Sort(ids)
	return ids
}

func diagsContain(diags diag.Diagnostics, substr string) bool {
	for _, d := range diags {
		if strings.Contains(d.Summary()+" "+d.Detail(), substr) {
			return true
		}
	}
	return false
}

func (p *fakeSubBalancerPanel) createRow(t *testing.T, row fakeSubBalancerRow) int {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.next++
	row.ID = p.next
	p.rows[row.ID] = &row
	return row.ID
}

func createSubBalancer(t *testing.T, client *Client, planned map[string]tftypes.Value) resource.CreateResponse {
	t.Helper()
	r := &SubBalancerResource{client: client}
	plan := subBalancerPlan(planned)
	resp := resource.CreateResponse{State: nullSubBalancerState()}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan, Config: tfsdk.Config(plan)}, &resp)
	return resp
}

func updateSubBalancer(t *testing.T, client *Client, prior, planned map[string]tftypes.Value) resource.UpdateResponse {
	t.Helper()
	r := &SubBalancerResource{client: client}
	plan := subBalancerPlan(planned)
	resp := resource.UpdateResponse{State: subBalancerState(planned)}
	r.Update(context.Background(), resource.UpdateRequest{Plan: plan, Config: tfsdk.Config(plan), State: subBalancerState(prior)}, &resp)
	return resp
}

func readSubBalancer(t *testing.T, client *Client, prior map[string]tftypes.Value) resource.ReadResponse {
	t.Helper()
	r := &SubBalancerResource{client: client}
	resp := resource.ReadResponse{State: subBalancerState(prior)}
	r.Read(context.Background(), resource.ReadRequest{State: subBalancerState(prior)}, &resp)
	return resp
}

// ---------------------------------------------------------------------------
// Client wire format
// ---------------------------------------------------------------------------

// The controller reads inboundIds with PostFormArray and enabled with
// GetPostForm: a JSON body, a comma-joined list or a missing enabled key would
// all be accepted by the panel and silently stored wrong.
func TestSubBalancerToForm(t *testing.T) {
	form, err := subBalancerToForm(&SubBalancer{
		Remark: "auto", Strategy: "leastLoad", InboundIds: []int64{1, 3}, SortOrder: 2, Enabled: false,
		MemberWeights: map[int64]float64{3: 0.5},
	})
	if err != nil {
		t.Fatalf("subBalancerToForm: %v", err)
	}
	want := url.Values{
		"remark": {"auto"}, "strategy": {"leastLoad"}, "sortOrder": {"2"}, "enabled": {"false"},
		"inboundIds": {"1", "3"}, "memberWeights": {`{"3":0.5}`},
	}
	if form.Encode() != want.Encode() {
		t.Fatalf("form = %s, want %s", form.Encode(), want.Encode())
	}

	// Without weights the key must be absent, not "{}" or "null": absence is what clears them.
	form, err = subBalancerToForm(&SubBalancer{Remark: "auto", Strategy: "random", InboundIds: []int64{1}, SortOrder: 1, Enabled: true})
	if err != nil {
		t.Fatalf("subBalancerToForm: %v", err)
	}
	if _, ok := form["memberWeights"]; ok {
		t.Fatalf("memberWeights must be omitted without weights, form = %s", form.Encode())
	}
	if form.Get("enabled") != "true" {
		t.Fatalf("enabled must always be sent, form = %s", form.Encode())
	}
}

// A weight JSON cannot encode must fail the call instead of posting a body
// whose memberWeights the panel would read as "clear the weights".
func TestSubBalancerToFormRejectsUnencodableWeights(t *testing.T) {
	if _, err := subBalancerToForm(&SubBalancer{InboundIds: []int64{1}, MemberWeights: map[int64]float64{1: math.Inf(1)}}); err == nil ||
		!strings.Contains(err.Error(), "encoding member weights") {
		t.Fatalf("want an encoding error, got %v", err)
	}
	if _, err := subBalancerToForm(nil); err == nil || err.Error() != "sub balancer is nil" {
		t.Fatalf("want a nil-balancer error, got %v", err)
	}
	client := newTestClient(t, "http://127.0.0.1:0")
	if _, err := client.CreateSubBalancer(context.Background(), nil); err == nil || err.Error() != "sub balancer is nil" {
		t.Fatalf("CreateSubBalancer(nil) = %v", err)
	}
	if _, err := client.UpdateSubBalancer(context.Background(), 1, nil); err == nil || err.Error() != "sub balancer is nil" {
		t.Fatalf("UpdateSubBalancer(nil) = %v", err)
	}
}

func TestSubBalancerClientRoundTrip(t *testing.T) {
	panel, client := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{})
	ctx := context.Background()

	created, err := client.CreateSubBalancer(ctx, &SubBalancer{
		Remark: "auto", Strategy: "leastLoad", InboundIds: []int64{1, 3}, SortOrder: 1, Enabled: true,
		MemberWeights: map[int64]float64{3: 2.5},
	})
	if err != nil {
		t.Fatalf("CreateSubBalancer: %v", err)
	}
	if req := panel.lastRequest(t); req.ContentType != "application/x-www-form-urlencoded" || req.Path != "/panel/api/sub-balancers" {
		t.Fatalf("create sent %s %s (%s), want a form POST to /panel/api/sub-balancers", req.Method, req.Path, req.ContentType)
	}
	if created.Id != 1 || created.MemberWeights[3] != 2.5 || !slices.Equal(created.InboundIds, []int64{1, 3}) {
		t.Fatalf("created = %+v", created)
	}

	got, err := client.GetSubBalancer(ctx, created.Id)
	if err != nil || got == nil || got.Remark != "auto" {
		t.Fatalf("GetSubBalancer = %+v, %v", got, err)
	}
	missing, err := client.GetSubBalancer(ctx, 42)
	if err != nil || missing != nil {
		t.Fatalf("GetSubBalancer of a missing id = %+v, %v; want nil, nil", missing, err)
	}

	updated, err := client.UpdateSubBalancer(ctx, created.Id, &SubBalancer{
		Remark: "renamed", Strategy: "random", InboundIds: []int64{1}, SortOrder: 4, Enabled: false,
	})
	if err != nil {
		t.Fatalf("UpdateSubBalancer: %v", err)
	}
	if req := panel.lastRequest(t); req.Path != "/panel/api/sub-balancers/1" {
		t.Fatalf("update posted to %s", req.Path)
	}
	if updated.Remark != "renamed" || updated.Enabled || updated.SortOrder != 4 || len(updated.MemberWeights) != 0 {
		t.Fatalf("updated = %+v", updated)
	}

	if err := client.DeleteSubBalancer(ctx, created.Id); err != nil {
		t.Fatalf("DeleteSubBalancer: %v", err)
	}
	if req := panel.lastRequest(t); req.Method != http.MethodPost || req.Path != "/panel/api/sub-balancers/1/del" {
		t.Fatalf("delete sent %s %s", req.Method, req.Path)
	}
	err = client.DeleteSubBalancer(ctx, created.Id)
	if !isSubBalancerNotFound(err) {
		t.Fatalf("second delete = %v, want the panel's not-found error", err)
	}
}

// Panels before v3.7.0 answer 404 on every route; the error must name the
// version instead of a bare status code.
func TestSubBalancerClientPreV370(t *testing.T) {
	_, client := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{noAPI: true})
	ctx := context.Background()
	b := &SubBalancer{Remark: "auto", Strategy: "random", InboundIds: []int64{1}, SortOrder: 1, Enabled: true}

	_, listErr := client.ListSubBalancers(ctx)
	_, createErr := client.CreateSubBalancer(ctx, b)
	_, updateErr := client.UpdateSubBalancer(ctx, 1, b)
	deleteErr := client.DeleteSubBalancer(ctx, 1)
	for name, err := range map[string]error{"list": listErr, "create": createErr, "update": updateErr, "delete": deleteErr} {
		if err == nil || !strings.Contains(err.Error(), "requires 3x-ui v3.7.0 or later") || !isHTTPNotFound(err) {
			t.Errorf("%s: want a v3.7.0 requirement wrapping the 404, got %v", name, err)
		}
	}
	if _, err := client.GetSubBalancer(ctx, 1); err == nil {
		t.Error("GetSubBalancer must propagate the list error, not report the balancer as gone")
	}
}

// Without an id the state could not address the row again; without the
// expected id the response is not the row that was written.
func TestSubBalancerClientRejectsUnexpectedRows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			_, _ = w.Write(okResponse(nil))
		case "/panel/api/sub-balancers":
			_, _ = w.Write(okResponse(map[string]any{"remark": "auto"}))
		default:
			_, _ = w.Write(okResponse(map[string]any{"id": 7, "remark": "auto"}))
		}
	}))
	defer srv.Close()
	client := newTestClient(t, srv.URL)
	b := &SubBalancer{Remark: "auto", Strategy: "random", InboundIds: []int64{1}, SortOrder: 1, Enabled: true}

	if _, err := client.CreateSubBalancer(context.Background(), b); err == nil || !strings.Contains(err.Error(), "no balancer id") {
		t.Errorf("create without id: %v", err)
	}
	if _, err := client.UpdateSubBalancer(context.Background(), 3, b); err == nil || !strings.Contains(err.Error(), "returned balancer 7, expected 3") {
		t.Errorf("update with another id: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Resource CRUD against the fake panel
// ---------------------------------------------------------------------------

func TestSubBalancerResourceCreate(t *testing.T) {
	panel, client := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{})
	resp := createSubBalancer(t, client, subBalancerPlanned(map[string]tftypes.Value{
		"strategy":       tftypes.NewValue(tftypes.String, "leastLoad"),
		"member_weights": tfWeights(map[string]any{"3": 2.5, "1": oneThird()}),
		"sort_order":     tftypes.NewValue(tftypes.Number, 5),
		"enabled":        tftypes.NewValue(tftypes.Bool, false),
	}))
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}

	state := subBalancerModelFromState(t, resp.State)
	if state.ID.ValueString() != "1" || state.Remark.ValueString() != "auto" || state.Strategy.ValueString() != "leastLoad" ||
		state.SortOrder.ValueInt64() != 5 || state.Enabled.ValueBool() {
		t.Fatalf("state = %+v", state)
	}
	if ids := setInt64s(t, state.InboundIDs); !slices.Equal(ids, []int64{1, 3}) {
		t.Fatalf("inbound_ids = %v", ids)
	}
	weights := state.MemberWeights.Elements()
	if !weights["1"].Equal(types.Float64Value(1.0/3)) || !weights["3"].Equal(types.Float64Value(2.5)) || len(weights) != 2 {
		t.Fatalf("member_weights = %v", state.MemberWeights)
	}

	row := panel.row(t, 1)
	if row.Enabled || row.SortOrder != 5 || row.MemberWeights[3] != 2.5 || row.MemberWeights[1] != 1.0/3 {
		t.Fatalf("panel row = %+v", row)
	}
	if got := panel.lastRequest(t).Form["inboundIds"]; !slices.Equal(got, []string{"1", "3"}) {
		t.Fatalf("inboundIds form values = %v", got)
	}
}

// v3.7.x saves the balancer and drops memberWeights. The create must fail and
// must not leave a balancer behind that Terraform does not track.
func TestSubBalancerResourceCreateWeightsUnsupported(t *testing.T) {
	panel, client := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{noWeights: true})
	resp := createSubBalancer(t, client, subBalancerPlanned(map[string]tftypes.Value{
		"strategy":       tftypes.NewValue(tftypes.String, "leastLoad"),
		"member_weights": tfWeights(map[string]any{"3": 2.0}),
	}))
	if !diagsContain(resp.Diagnostics, "requires 3x-ui v3.8.0") || !diagsContain(resp.Diagnostics, "nothing was created") {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatal("a rolled-back create must not write state")
	}
	if len(panel.list()) != 0 {
		t.Fatalf("panel still has %d balancers after the rollback", len(panel.list()))
	}
}

// If the rollback fails the row exists; state must track it so Terraform can
// still destroy it.
func TestSubBalancerResourceCreateRollbackFailure(t *testing.T) {
	_, client := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{noWeights: true, failDelete: true})
	resp := createSubBalancer(t, client, subBalancerPlanned(map[string]tftypes.Value{
		"strategy":       tftypes.NewValue(tftypes.String, "leastLoad"),
		"member_weights": tfWeights(map[string]any{"3": 2.0}),
	}))
	if !diagsContain(resp.Diagnostics, "stays in state as tainted") || !diagsContain(resp.Diagnostics, "database is locked") {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
	state := subBalancerModelFromState(t, resp.State)
	if state.ID.ValueString() != "1" || !state.MemberWeights.IsNull() {
		t.Fatalf("state must hold the stored row without weights, got %+v", state)
	}
}

func TestSubBalancerResourceCreateErrors(t *testing.T) {
	_, legacy := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{noAPI: true})
	if resp := createSubBalancer(t, legacy, subBalancerPlanned(nil)); !diagsContain(resp.Diagnostics, "requires 3x-ui v3.7.0") {
		t.Errorf("pre-v3.7.0 create: %v", resp.Diagnostics)
	}

	// A key the conversion cannot encode must stop the create, not be dropped from the request.
	panel, client := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{})
	resp := createSubBalancer(t, client, subBalancerPlanned(map[string]tftypes.Value{
		"strategy":       tftypes.NewValue(tftypes.String, "leastLoad"),
		"member_weights": tfWeights(map[string]any{"09": 2.0}),
	}))
	if !diagsContain(resp.Diagnostics, `"09" is not an inbound id`) {
		t.Errorf("non-canonical key: %v", resp.Diagnostics)
	}
	if len(panel.requests) != 0 {
		t.Errorf("an unencodable plan reached the panel: %+v", panel.requests)
	}
}

func TestSubBalancerResourceRead(t *testing.T) {
	panel, client := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{})
	id := panel.createRow(t, fakeSubBalancerRow{
		Remark: "auto", Strategy: "leastLoad", InboundIDs: []int{3, 1, 3}, SortOrder: 2, Enabled: true,
		MemberWeights: map[int]float64{3: 1.5},
	})

	// Import: only the id is known.
	resp := readSubBalancer(t, client, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, strconv.Itoa(id))})
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	state := subBalancerModelFromState(t, resp.State)
	if state.Remark.ValueString() != "auto" || state.Strategy.ValueString() != "leastLoad" || state.SortOrder.ValueInt64() != 2 || !state.Enabled.ValueBool() {
		t.Fatalf("state = %+v", state)
	}
	if ids := setInt64s(t, state.InboundIDs); !slices.Equal(ids, []int64{1, 3}) {
		t.Fatalf("inbound_ids = %v, want the deduplicated stored ids", ids)
	}
	if w, _ := state.MemberWeights.Elements()["3"].(types.Float64); w.ValueFloat64() != 1.5 {
		t.Fatalf("member_weights = %v", state.MemberWeights)
	}

	// Deleting inbound 3 on the panel strips its id but not its weight; both
	// must show so the next plan is not empty.
	panel.deleteInbound(3)
	resp = readSubBalancer(t, client, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, strconv.Itoa(id))})
	state = subBalancerModelFromState(t, resp.State)
	if ids := setInt64s(t, state.InboundIDs); !slices.Equal(ids, []int64{1}) {
		t.Fatalf("inbound_ids after inbound delete = %v", ids)
	}
	if _, ok := state.MemberWeights.Elements()["3"]; !ok {
		t.Fatalf("member_weights after inbound delete = %v, want the stale key kept", state.MemberWeights)
	}

	// The row survives its last inbound; it must stay in state, with no members.
	panel.deleteInbound(1)
	resp = readSubBalancer(t, client, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, strconv.Itoa(id))})
	if resp.State.Raw.IsNull() {
		t.Fatal("a balancer whose inbounds were all deleted must stay in state")
	}
	state = subBalancerModelFromState(t, resp.State)
	if state.InboundIDs.IsNull() || len(state.InboundIDs.Elements()) != 0 {
		t.Fatalf("inbound_ids = %v, want an empty set like the panel's []", state.InboundIDs)
	}
}

func TestSubBalancerResourceReadMissingAndErrors(t *testing.T) {
	_, client := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{})
	resp := readSubBalancer(t, client, subBalancerPlanned(map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, "9")}))
	if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
		t.Fatalf("a balancer missing from the list must be removed from state: %v", resp.Diagnostics)
	}

	// A 404 from an old panel is not "deleted": dropping the state would plan a create.
	_, legacy := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{noAPI: true})
	resp = readSubBalancer(t, legacy, subBalancerPlanned(map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, "9")}))
	if !diagsContain(resp.Diagnostics, "requires 3x-ui v3.7.0") || resp.State.Raw.IsNull() {
		t.Fatalf("pre-v3.7.0 read must fail and keep state: %v", resp.Diagnostics)
	}
}

func TestSubBalancerResourceUpdate(t *testing.T) {
	panel, client := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{})
	id := panel.createRow(t, fakeSubBalancerRow{
		Remark: "auto", Strategy: "leastLoad", InboundIDs: []int{1, 3}, SortOrder: 1, Enabled: true,
		MemberWeights: map[int]float64{3: 2},
	})
	prior := subBalancerPlanned(map[string]tftypes.Value{
		"id":             tftypes.NewValue(tftypes.String, strconv.Itoa(id)),
		"strategy":       tftypes.NewValue(tftypes.String, "leastLoad"),
		"member_weights": tfWeights(map[string]any{"3": 2.0}),
	})

	// The panel clears weights the update omits, and keeps the stored enabled
	// flag unless the update sends one.
	planned := subBalancerPlanned(map[string]tftypes.Value{
		"id":       tftypes.NewValue(tftypes.String, strconv.Itoa(id)),
		"remark":   tftypes.NewValue(tftypes.String, "renamed"),
		"strategy": tftypes.NewValue(tftypes.String, "roundRobin"),
		"enabled":  tftypes.NewValue(tftypes.Bool, false),
	})
	resp := updateSubBalancer(t, client, prior, planned)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	row := panel.row(t, id)
	if row.Remark != "renamed" || row.Strategy != "roundRobin" || row.MemberWeights != nil || row.Enabled {
		t.Fatalf("panel row = %+v", row)
	}
	state := subBalancerModelFromState(t, resp.State)
	if !state.MemberWeights.IsNull() || state.Enabled.ValueBool() || state.Remark.ValueString() != "renamed" {
		t.Fatalf("state = %+v", state)
	}
}

// On v3.7.x the update applies everything but the weights. State must record
// what the panel stored, or the next plan would hide the missing weights.
func TestSubBalancerResourceUpdateWeightsUnsupported(t *testing.T) {
	panel, client := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{noWeights: true})
	id := panel.createRow(t, fakeSubBalancerRow{Remark: "auto", Strategy: "random", InboundIDs: []int{1, 3}, SortOrder: 1, Enabled: true})
	prior := subBalancerPlanned(map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, strconv.Itoa(id))})
	planned := subBalancerPlanned(map[string]tftypes.Value{
		"id":             tftypes.NewValue(tftypes.String, strconv.Itoa(id)),
		"strategy":       tftypes.NewValue(tftypes.String, "leastLoad"),
		"member_weights": tfWeights(map[string]any{"3": 2.0}),
	})
	resp := updateSubBalancer(t, client, prior, planned)
	if !diagsContain(resp.Diagnostics, "requires 3x-ui v3.8.0") || !diagsContain(resp.Diagnostics, "other changes were applied") {
		t.Fatalf("diagnostics = %v", resp.Diagnostics)
	}
	state := subBalancerModelFromState(t, resp.State)
	if state.Strategy.ValueString() != "leastLoad" || !state.MemberWeights.IsNull() {
		t.Fatalf("state must match the panel (leastLoad, no weights), got %+v", state)
	}
}

func TestSubBalancerResourceUpdateErrors(t *testing.T) {
	_, client := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{})
	prior := subBalancerPlanned(map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, "4")})

	resp := updateSubBalancer(t, client, prior, prior)
	if !diagsContain(resp.Diagnostics, "Failed to update sub balancer") || !diagsContain(resp.Diagnostics, "record not found") {
		t.Errorf("update of a missing row: %v", resp.Diagnostics)
	}

	planned := subBalancerPlanned(map[string]tftypes.Value{
		"id":             tftypes.NewValue(tftypes.String, "4"),
		"strategy":       tftypes.NewValue(tftypes.String, "leastLoad"),
		"member_weights": tfWeights(map[string]any{"+3": 2.0}),
	})
	resp = updateSubBalancer(t, client, prior, planned)
	if !diagsContain(resp.Diagnostics, `"+3" is not an inbound id`) {
		t.Errorf("non-canonical key: %v", resp.Diagnostics)
	}

	bad := subBalancerPlanned(map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, "x")})
	resp = updateSubBalancer(t, client, bad, bad)
	if !diagsContain(resp.Diagnostics, "Invalid sub balancer id") {
		t.Errorf("invalid id: %v", resp.Diagnostics)
	}
}

func TestSubBalancerResourceDelete(t *testing.T) {
	panel, client := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{})
	id := panel.createRow(t, fakeSubBalancerRow{Remark: "auto", Strategy: "random", InboundIDs: []int{1}, SortOrder: 1, Enabled: true})
	r := &SubBalancerResource{client: client}
	state := subBalancerState(subBalancerPlanned(map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, strconv.Itoa(id))}))

	var resp resource.DeleteResponse
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() || len(panel.list()) != 0 {
		t.Fatalf("Delete: %v, rows left: %d", resp.Diagnostics, len(panel.list()))
	}

	// Already deleted on the panel: destroy must still succeed.
	resp = resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete of a missing row: %v", resp.Diagnostics)
	}

	panel.failDelete = true
	resp = resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
	if !diagsContain(resp.Diagnostics, "database is locked") {
		t.Fatalf("a failed delete must surface: %v", resp.Diagnostics)
	}

	bad := subBalancerState(subBalancerPlanned(map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, "0")}))
	resp = resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: bad}, &resp)
	if !diagsContain(resp.Diagnostics, "Invalid sub balancer id") {
		t.Fatalf("invalid id: %v", resp.Diagnostics)
	}
}

func TestSubBalancerResourceInvalidStateID(t *testing.T) {
	_, client := newFakeSubBalancerPanel(t, &fakeSubBalancerPanel{})
	resp := readSubBalancer(t, client, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, "-1")})
	if !diagsContain(resp.Diagnostics, "Invalid sub balancer id") {
		t.Fatalf("Read with id -1: %v", resp.Diagnostics)
	}
}

// Requests whose values do not decode into the model must return the
// framework's diagnostics instead of acting on a zero model.
func TestSubBalancerResourceUndecodableRequests(t *testing.T) {
	r := &SubBalancerResource{}
	ctx := context.Background()
	wrong := tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.Number}},
		map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.Number, 1)})
	schema := subBalancerResourceSchema()

	var createResp resource.CreateResponse
	r.Create(ctx, resource.CreateRequest{Plan: tfsdk.Plan{Schema: schema, Raw: wrong}}, &createResp)
	var readResp resource.ReadResponse
	r.Read(ctx, resource.ReadRequest{State: tfsdk.State{Schema: schema, Raw: wrong}}, &readResp)
	var updateResp resource.UpdateResponse
	r.Update(ctx, resource.UpdateRequest{Plan: tfsdk.Plan{Schema: schema, Raw: wrong}, State: tfsdk.State{Schema: schema, Raw: wrong}}, &updateResp)
	var deleteResp resource.DeleteResponse
	r.Delete(ctx, resource.DeleteRequest{State: tfsdk.State{Schema: schema, Raw: wrong}}, &deleteResp)
	for name, diags := range map[string]diag.Diagnostics{
		"create": createResp.Diagnostics, "read": readResp.Diagnostics, "update": updateResp.Diagnostics, "delete": deleteResp.Diagnostics,
	} {
		if !diags.HasError() {
			t.Errorf("%s accepted an undecodable request", name)
		}
	}
}

func TestSubBalancerResourceWiring(t *testing.T) {
	r := NewSubBalancerResource().(*SubBalancerResource)
	ctx := context.Background()

	var meta resource.MetadataResponse
	r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "threexui"}, &meta)
	if meta.TypeName != "threexui_sub_balancer" {
		t.Fatalf("TypeName = %q", meta.TypeName)
	}

	var cfg resource.ConfigureResponse
	r.Configure(ctx, resource.ConfigureRequest{}, &cfg)
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: "not a client"}, &cfg)
	if !cfg.Diagnostics.HasError() {
		t.Fatal("Configure must reject foreign provider data")
	}
	client := newTestClient(t, "http://127.0.0.1:0")
	cfg = resource.ConfigureResponse{}
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &cfg)
	if cfg.Diagnostics.HasError() || r.client != client {
		t.Fatalf("Configure: %v", cfg.Diagnostics)
	}

	for _, tc := range []struct {
		id      string
		wantErr bool
	}{{"12", false}, {"0", true}, {"abc", true}, {"", true}} {
		resp := resource.ImportStateResponse{State: nullSubBalancerState()}
		r.ImportState(ctx, resource.ImportStateRequest{ID: tc.id}, &resp)
		if resp.Diagnostics.HasError() != tc.wantErr {
			t.Errorf("ImportState(%q): %v", tc.id, resp.Diagnostics)
		}
		if !tc.wantErr {
			var id types.String
			resp.State.GetAttribute(ctx, path.Root("id"), &id)
			if id.ValueString() != tc.id {
				t.Errorf("ImportState(%q) set id %q", tc.id, id.ValueString())
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Plan-time validation, through the framework as Terraform runs it
// ---------------------------------------------------------------------------

func validateSubBalancerConfig(t *testing.T, attrs map[string]tftypes.Value) []*tfprotov6.Diagnostic {
	t.Helper()
	server, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatalf("provider server: %v", err)
	}
	config, err := tfprotov6.NewDynamicValue(subBalancerObjectType(), tfSubBalancerValue(attrs))
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	resp, err := server.ValidateResourceConfig(context.Background(), &tfprotov6.ValidateResourceConfigRequest{
		TypeName: "threexui_sub_balancer",
		Config:   &config,
	})
	if err != nil {
		t.Fatalf("ValidateResourceConfig: %v", err)
	}
	return resp.Diagnostics
}

// Each case is a value the panel rejects or silently rewrites; all of them
// must fail at plan time, before anything is written.
func TestSubBalancerConfigValidation(t *testing.T) {
	base := func(attrs map[string]tftypes.Value) map[string]tftypes.Value {
		config := map[string]tftypes.Value{
			"remark":      tftypes.NewValue(tftypes.String, "auto"),
			"inbound_ids": tfInt64Set(1, 3),
		}
		maps.Copy(config, attrs)
		return config
	}
	leastLoad := tftypes.NewValue(tftypes.String, "leastLoad")
	cases := []struct {
		name    string
		config  map[string]tftypes.Value
		wantErr string
	}{
		{name: "minimal", config: base(nil)},
		{name: "weights with leastLoad", config: base(map[string]tftypes.Value{
			"strategy": leastLoad, "member_weights": tfWeights(map[string]any{"3": 0.3, "1": oneThird()}),
		})},
		{name: "empty remark", config: base(map[string]tftypes.Value{"remark": tftypes.NewValue(tftypes.String, "  ")}), wantErr: "must not be empty"},
		{name: "untrimmed remark", config: base(map[string]tftypes.Value{"remark": tftypes.NewValue(tftypes.String, " auto")}), wantErr: "whitespace"},
		{name: "remark over 256 bytes", config: base(map[string]tftypes.Value{
			"remark": tftypes.NewValue(tftypes.String, strings.Repeat("я", 129)),
		}), wantErr: "at most 256 bytes, got 258"},
		{name: "unknown strategy", config: base(map[string]tftypes.Value{"strategy": tftypes.NewValue(tftypes.String, "fastest")}), wantErr: "value must be one of"},
		{name: "no inbounds", config: base(map[string]tftypes.Value{"inbound_ids": tfInt64Set()}), wantErr: "at least 1"},
		{name: "inbound id 0", config: base(map[string]tftypes.Value{"inbound_ids": tfInt64Set(0)}), wantErr: "at least 1"},
		{name: "sort_order 0", config: base(map[string]tftypes.Value{"sort_order": tftypes.NewValue(tftypes.Number, 0)}), wantErr: "at least 1"},
		{name: "weights need leastLoad (default random)", config: base(map[string]tftypes.Value{
			"member_weights": tfWeights(map[string]any{"3": 2.0}),
		}), wantErr: "strategy \"random\""},
		{name: "weights need leastLoad", config: base(map[string]tftypes.Value{
			"strategy": tftypes.NewValue(tftypes.String, "leastPing"), "member_weights": tfWeights(map[string]any{"3": 2.0}),
		}), wantErr: "strategy \"leastPing\""},
		{name: "weight key outside inbound_ids", config: base(map[string]tftypes.Value{
			"strategy": leastLoad, "member_weights": tfWeights(map[string]any{"5": 2.0}),
		}), wantErr: "Inbound 5 is not a member"},
		{name: "non-canonical weight key", config: base(map[string]tftypes.Value{
			"strategy": leastLoad, "member_weights": tfWeights(map[string]any{"03": 2.0}),
		}), wantErr: "without sign or leading zeros"},
		{name: "weight key beyond int64", config: base(map[string]tftypes.Value{
			"strategy": leastLoad, "member_weights": tfWeights(map[string]any{"99999999999999999999": 2.0}),
		}), wantErr: "Inbound 99999999999999999999 is not a member"},
		{name: "zero weight", config: base(map[string]tftypes.Value{
			"strategy": leastLoad, "member_weights": tfWeights(map[string]any{"3": 0.0}),
		}), wantErr: "must be between"},
		{name: "weight above float32", config: base(map[string]tftypes.Value{
			"strategy": leastLoad, "member_weights": tfWeights(map[string]any{"3": 1e39}),
		}), wantErr: "must be between"},
		{name: "empty weights map", config: base(map[string]tftypes.Value{
			"strategy": leastLoad, "member_weights": tfWeights(map[string]any{}),
		}), wantErr: "at least 1"},
		{name: "unknown inbound ids defer the key check", config: base(map[string]tftypes.Value{
			"strategy": leastLoad, "inbound_ids": tfUnknown(tftypes.Set{ElementType: tftypes.Number}),
			"member_weights": tfWeights(map[string]any{"5": 2.0}),
		})},
		{name: "an unknown inbound id may match the key", config: base(map[string]tftypes.Value{
			"strategy": leastLoad, "inbound_ids": tftypes.NewValue(tftypes.Set{ElementType: tftypes.Number}, []tftypes.Value{
				tftypes.NewValue(tftypes.Number, 1), tfUnknown(tftypes.Number),
			}),
			"member_weights": tfWeights(map[string]any{"5": 2.0}),
		})},
		{name: "unknown strategy defers the strategy check", config: base(map[string]tftypes.Value{
			"strategy": tfUnknown(tftypes.String), "member_weights": tfWeights(map[string]any{"3": 2.0}),
		})},
		{name: "unknown weights are checked at apply", config: base(map[string]tftypes.Value{
			"member_weights": tfUnknown(tftypes.Map{ElementType: tftypes.Number}),
		})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diags := validateSubBalancerConfig(t, tc.config)
			var errs []string
			for _, d := range diags {
				if d.Severity == tfprotov6.DiagnosticSeverityError {
					errs = append(errs, d.Summary+": "+d.Detail)
				}
			}
			switch {
			case tc.wantErr == "" && len(errs) > 0:
				t.Fatalf("unexpected errors: %v", errs)
			case tc.wantErr != "" && !slices.ContainsFunc(errs, func(e string) bool { return strings.Contains(e, tc.wantErr) }):
				t.Fatalf("want an error containing %q, got %v", tc.wantErr, errs)
			}
		})
	}
}

func TestSubBalancerValidatorDescriptions(t *testing.T) {
	ctx := context.Background()
	for _, d := range []interface {
		Description(context.Context) string
		MarkdownDescription(context.Context) string
	}{subBalancerRemarkValidator{}, subBalancerWeightsValidator{}} {
		if d.Description(ctx) == "" || d.MarkdownDescription(ctx) != d.Description(ctx) {
			t.Errorf("%T must describe itself", d)
		}
	}
	// A remark interpolated from another resource is checked at apply instead.
	var resp validator.StringResponse
	subBalancerRemarkValidator{}.ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringUnknown()}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unknown remark: %v", resp.Diagnostics)
	}
	// A config that does not decode into the model must report, not panic.
	var cvResp resource.ValidateConfigResponse
	subBalancerWeightsValidator{}.ValidateResource(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{
		Schema: subBalancerResourceSchema(),
		Raw:    tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.Number}}, map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.Number, 1)}),
	}}, &cvResp)
	if !cvResp.Diagnostics.HasError() {
		t.Fatal("an undecodable config must produce diagnostics")
	}
}

// ---------------------------------------------------------------------------
// Conversion
// ---------------------------------------------------------------------------

func subBalancerModel(t *testing.T, attrs map[string]tftypes.Value) *SubBalancerResourceModel {
	t.Helper()
	m := subBalancerModelFromState(t, subBalancerState(subBalancerPlanned(attrs)))
	return &m
}

func TestSubBalancerFromModel(t *testing.T) {
	ctx := context.Background()
	b, diags := subBalancerFromModel(ctx, subBalancerModel(t, map[string]tftypes.Value{
		"strategy":       tftypes.NewValue(tftypes.String, "leastLoad"),
		"inbound_ids":    tfInt64Set(9, 1, 3),
		"member_weights": tfWeights(map[string]any{"9": oneThird()}),
	}))
	if diags.HasError() {
		t.Fatalf("subBalancerFromModel: %v", diags)
	}
	if !slices.Equal(b.InboundIds, []int64{1, 3, 9}) || b.MemberWeights[9] != 1.0/3 {
		t.Fatalf("balancer = %+v", b)
	}

	// The key pattern alone admits it; dropping it silently would post fewer weights than planned.
	_, diags = subBalancerFromModel(ctx, subBalancerModel(t, map[string]tftypes.Value{
		"strategy": tftypes.NewValue(tftypes.String, "leastLoad"), "inbound_ids": tfInt64Set(9),
		"member_weights": tfWeights(map[string]any{"99999999999999999999": 2.0}),
	}))
	if !diagsContain(diags, `"99999999999999999999" is not an inbound id`) {
		t.Errorf("key beyond int64: %v", diags)
	}
}

func TestSubBalancerHelpers(t *testing.T) {
	if !subBalancerWeightsDropped(&SubBalancer{MemberWeights: map[int64]float64{1: 2}}, &SubBalancer{}) {
		t.Error("weights sent but not stored must be detected")
	}
	if subBalancerWeightsDropped(&SubBalancer{}, &SubBalancer{}) {
		t.Error("no weights sent is not a drop")
	}
	if isSubBalancerNotFound(nil) || !isSubBalancerNotFound(fmt.Errorf("request failed: status 200, msg: Failed to delete subscription balancer (sub balancer not found)")) {
		t.Error("isSubBalancerNotFound must match the panel message only")
	}
	if subBalancerAPIError(nil) != nil {
		t.Error("subBalancerAPIError(nil) must stay nil")
	}
}
