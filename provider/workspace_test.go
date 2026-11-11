package adaptive

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"context"
	"github.com/blang/semver"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// fakeWorkspace is an httptest stand-in for the workspace settings and sso-sync
// terraform routes.
type fakeWorkspace struct {
	mu       sync.Mutex
	settings map[string]any
	sso      map[string]*SsoSyncReadResponse // by provider id
	deleted  []string
}

func newFakeWorkspace() *fakeWorkspace {
	return &fakeWorkspace{
		// Server reports every field, including ones the program never sets.
		settings: map[string]any{
			"defaultCluster": "main", "mfaEnabled": false, "timezone": "UTC",
			"maxJitRequestDuration": "8h", "defaultLoginEnabled": true,
		},
		sso: map[string]*SsoSyncReadResponse{},
	}
}

func (f *fakeWorkspace) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	path := r.URL.Path
	write := func(v any) { _ = json.NewEncoder(w).Encode(v) }
	switch {
	case strings.HasSuffix(path, "/terraform/workspace/settings/read"):
		write(f.settings)
	case strings.HasSuffix(path, "/terraform/workspace/settings/update"):
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		for k, v := range body {
			f.settings[k] = v
		}
		write(f.settings)
	case strings.Contains(path, "/terraform/sso-sync/read/"):
		id := path[strings.LastIndex(path, "/")+1:]
		s, ok := f.sso[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		write(s)
	case strings.Contains(path, "/terraform/sso-sync/update/"):
		id := path[strings.LastIndex(path, "/")+1:]
		var req SsoSyncUpdateRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		s := f.sso[id]
		if s == nil {
			s = &SsoSyncReadResponse{ProviderID: id, ProviderType: "okta", LoginEnabled: true}
			f.sso[id] = s
		}
		if req.LoginEnabled != nil {
			s.LoginEnabled = *req.LoginEnabled
		}
		if req.AutoSyncEnabled != nil {
			s.AutoSyncEnabled = *req.AutoSyncEnabled
		}
		if req.RoleSyncEnabled != nil {
			s.RoleSyncEnabled = *req.RoleSyncEnabled
		}
		s.Groups = nil
		// The IdP returns groups in its own (reverse) order.
		for i := len(req.Groups) - 1; i >= 0; i-- {
			s.Groups = append(s.Groups, SsoSyncGroup{ID: req.Groups[i], Name: "name-" + req.Groups[i]})
		}
		write(s)
	case strings.Contains(path, "/terraform/sso-sync/delete/"):
		id := path[strings.LastIndex(path, "/")+1:]
		f.deleted = append(f.deleted, id)
		delete(f.sso, id)
		write("ok")
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func newWorkspaceServer(t *testing.T) (integration.Server, *fakeWorkspace) {
	t.Helper()
	f := newFakeWorkspace()
	api := httptest.NewServer(http.HandlerFunc(f.handler))
	t.Cleanup(api.Close)
	t.Setenv("ADAPTIVE_SVC_TOKEN", "test-token")
	t.Setenv("ADAPTIVE_URL", api.URL)
	prov, err := Provider()
	if err != nil {
		t.Fatal(err)
	}
	srv, err := integration.NewServer(context.Background(), "adaptive",
		semver.MustParse(Version), integration.WithProvider(prov))
	if err != nil {
		t.Fatal(err)
	}
	return srv, f
}

const (
	wsTok  = "adaptive:index:WorkspaceSettings"
	ssoTok = "adaptive:index:SsoSync"
)

func strList(vs ...string) property.Value {
	arr := make([]property.Value, len(vs))
	for i, v := range vs {
		arr[i] = property.New(v)
	}
	return property.New(property.NewArray(arr))
}

// cycle runs Read then Diff against the same inputs, as refresh then preview do,
// and returns the new state plus whether the diff reported changes.
func cycle(t *testing.T, srv integration.Server, tok, id string, inputs, state property.Map) (property.Map, property.Map, bool) {
	t.Helper()
	rr, err := srv.Read(p.ReadRequest{ID: id, Urn: urnFor(tok), Inputs: inputs, Properties: state})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if rr.ID == "" {
		t.Fatal("read dropped the resource")
	}
	d, err := srv.Diff(p.DiffRequest{ID: id, Urn: urnFor(tok), State: rr.Properties, Inputs: inputs, OldInputs: rr.Inputs})
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	return rr.Inputs, rr.Properties, d.HasChanges
}

func create(t *testing.T, srv integration.Server, tok string, inputs property.Map) (string, property.Map) {
	t.Helper()
	cr, err := srv.Create(p.CreateRequest{Urn: urnFor(tok), Properties: inputs})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return cr.ID, cr.Properties
}

func TestWorkspaceSettingsLifecycleAndConvergence(t *testing.T) {
	srv, f := newWorkspaceServer(t)
	inputs := property.NewMap(map[string]property.Value{
		"mfaEnabled": property.New(true), "timezone": property.New("Europe/Stockholm"),
	})
	id, state := create(t, srv, wsTok, inputs)
	if id != workspaceSettingsID {
		t.Fatalf("id = %q", id)
	}
	if f.settings["mfaEnabled"] != true || f.settings["timezone"] != "Europe/Stockholm" {
		t.Fatalf("create did not apply: %v", f.settings)
	}
	// Unset fields are never sent.
	if f.settings["defaultCluster"] != "main" {
		t.Errorf("unset field was touched: %v", f.settings)
	}
	// 3 consecutive refresh -> preview cycles stay clean even though the server
	// reports many fields the program never set.
	in := inputs
	for i := 0; i < 3; i++ {
		var changed bool
		in, state, changed = cycle(t, srv, wsTok, id, in, state)
		if changed {
			t.Fatalf("cycle %d: unexpected diff", i)
		}
	}
	for _, k := range []string{"defaultCluster", "maxJitRequestDuration", "defaultLoginEnabled"} {
		if _, ok := in.GetOk(k); ok {
			t.Errorf("unset field %s adopted into inputs", k)
		}
	}
}

func TestWorkspaceSettingsUIDrift(t *testing.T) {
	srv, f := newWorkspaceServer(t)
	inputs := property.NewMap(map[string]property.Value{"mfaEnabled": property.New(true)})
	id, state := create(t, srv, wsTok, inputs)

	f.mu.Lock()
	f.settings["mfaEnabled"] = false // changed in the UI
	f.mu.Unlock()

	_, _, changed := cycle(t, srv, wsTok, id, inputs, state)
	if !changed {
		t.Fatal("UI drift on a managed field must produce a diff")
	}
	// Update restores it.
	if _, err := srv.Update(p.UpdateRequest{ID: id, Urn: urnFor(wsTok), State: state, Inputs: inputs}); err != nil {
		t.Fatal(err)
	}
	if f.settings["mfaEnabled"] != true {
		t.Fatal("update did not restore the setting")
	}
}

func TestWorkspaceSettingsExplicitEmptySent(t *testing.T) {
	srv, f := newWorkspaceServer(t)
	inputs := property.NewMap(map[string]property.Value{"defaultCluster": property.New("")})
	create(t, srv, wsTok, inputs)
	if v, ok := f.settings["defaultCluster"]; !ok || v != "" {
		t.Fatalf("explicit empty string must be sent, got %v", f.settings["defaultCluster"])
	}
}

func TestWorkspaceSettingsEmptyProgramRejected(t *testing.T) {
	srv, _ := newWorkspaceServer(t)
	resp, err := srv.Check(p.CheckRequest{Urn: urnFor(wsTok), Inputs: property.NewMap(nil)})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Failures) == 0 {
		t.Fatal("empty WorkspaceSettings should fail check")
	}
}

func TestWorkspaceSettingsDeleteIsNoop(t *testing.T) {
	srv, f := newWorkspaceServer(t)
	inputs := property.NewMap(map[string]property.Value{"mfaEnabled": property.New(true)})
	id, state := create(t, srv, wsTok, inputs)
	if err := srv.Delete(p.DeleteRequest{ID: id, Urn: urnFor(wsTok), Properties: state}); err != nil {
		t.Fatal(err)
	}
	if f.settings["mfaEnabled"] != true {
		t.Fatal("delete must not reset settings")
	}
}

func TestSsoSyncLifecycle(t *testing.T) {
	srv, f := newWorkspaceServer(t)
	inputs := property.NewMap(map[string]property.Value{
		"providerId": property.New("okta-1"), "groups": strList("g3", "g1", "g2"),
		"autoSyncEnabled": property.New(true),
	})
	chk, err := srv.Check(p.CheckRequest{Urn: urnFor(ssoTok), Inputs: inputs})
	if err != nil || len(chk.Failures) > 0 {
		t.Fatalf("check: %v %v", err, chk.Failures)
	}
	inputs = chk.Inputs
	id, state := create(t, srv, ssoTok, inputs)
	if id != "okta-1" {
		t.Fatalf("id = %q", id)
	}
	if pt := state.Get("providerType"); pt.AsString() != "okta" {
		t.Errorf("providerType = %v", pt)
	}
	if !f.sso["okta-1"].AutoSyncEnabled {
		t.Error("autoSyncEnabled not applied")
	}
	// Server returns groups reversed; 3 cycles must stay clean.
	in := inputs
	for i := 0; i < 3; i++ {
		var changed bool
		in, state, changed = cycle(t, srv, ssoTok, id, in, state)
		if changed {
			t.Fatalf("cycle %d: unexpected diff", i)
		}
	}
	// loginEnabled was never set: not adopted.
	if _, ok := in.GetOk("loginEnabled"); ok {
		t.Error("unset loginEnabled adopted")
	}
	// Delete calls the endpoint.
	if err := srv.Delete(p.DeleteRequest{ID: id, Urn: urnFor(ssoTok), Properties: state}); err != nil {
		t.Fatal(err)
	}
	if len(f.deleted) != 1 || f.deleted[0] != "okta-1" {
		t.Fatalf("delete endpoint not called: %v", f.deleted)
	}
}

func TestSsoSyncGroupOrderInsensitive(t *testing.T) {
	srv, _ := newWorkspaceServer(t)
	mk := func(gs ...string) property.Map {
		return property.NewMap(map[string]property.Value{
			"providerId": property.New("okta-1"), "groups": strList(gs...),
		})
	}
	a, _ := srv.Check(p.CheckRequest{Urn: urnFor(ssoTok), Inputs: mk("b", "a", "c", "a")})
	id, state := create(t, srv, ssoTok, a.Inputs)
	b, _ := srv.Check(p.CheckRequest{Urn: urnFor(ssoTok), Inputs: mk("c", "b", "a")})
	d, err := srv.Diff(p.DiffRequest{ID: id, Urn: urnFor(ssoTok), State: state, Inputs: b.Inputs, OldInputs: a.Inputs})
	if err != nil {
		t.Fatal(err)
	}
	if d.HasChanges {
		t.Fatalf("reordered groups produced a diff: %+v", d.DetailedDiff)
	}
}

func TestSsoSyncUIDriftAndReplace(t *testing.T) {
	srv, f := newWorkspaceServer(t)
	inputs := property.NewMap(map[string]property.Value{
		"providerId": property.New("okta-1"), "groups": strList("a", "b"),
	})
	id, state := create(t, srv, ssoTok, inputs)

	f.mu.Lock()
	f.sso["okta-1"].Groups = append(f.sso["okta-1"].Groups, SsoSyncGroup{ID: "z", Name: "z"})
	f.mu.Unlock()
	_, _, changed := cycle(t, srv, ssoTok, id, inputs, state)
	if !changed {
		t.Fatal("group added in the UI must produce a diff")
	}

	// providerId change replaces.
	other := property.NewMap(map[string]property.Value{
		"providerId": property.New("okta-2"), "groups": strList("a", "b"),
	})
	d, err := srv.Diff(p.DiffRequest{ID: id, Urn: urnFor(ssoTok), State: state, Inputs: other, OldInputs: inputs})
	if err != nil {
		t.Fatal(err)
	}
	if pd, ok := d.DetailedDiff["providerId"]; !ok || !strings.Contains(string(pd.Kind), "replace") {
		t.Fatalf("providerId change must replace: %+v", d.DetailedDiff)
	}
}

func TestSsoSyncRead404DropsFromState(t *testing.T) {
	srv, f := newWorkspaceServer(t)
	inputs := property.NewMap(map[string]property.Value{
		"providerId": property.New("okta-1"), "groups": strList("a"),
	})
	id, state := create(t, srv, ssoTok, inputs)
	f.mu.Lock()
	delete(f.sso, "okta-1")
	f.mu.Unlock()
	rr, err := srv.Read(p.ReadRequest{ID: id, Urn: urnFor(ssoTok), Inputs: inputs, Properties: state})
	if err != nil {
		t.Fatal(err)
	}
	if rr.ID != "" {
		t.Fatalf("404 must drop from state, got id %q", rr.ID)
	}
	// Import of an unknown id fails.
	if _, err := srv.Read(p.ReadRequest{ID: "nope", Urn: urnFor(ssoTok)}); err == nil {
		t.Fatal("import of unknown provider must fail")
	}
}

func TestNormalizeGroups(t *testing.T) {
	got := normalizeGroups([]string{"b", "a", "b"})
	if !sort.StringsAreSorted(got) || len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if normalizeGroups(nil) != nil {
		t.Fatal("nil must stay nil")
	}
}
