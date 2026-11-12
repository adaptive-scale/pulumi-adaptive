package adaptive

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// These journeys use the real inferred provider and an exclusively loopback API.
// All values and opaque fingerprints are synthetic. No external API is contacted.
// Limitations are asserted as observations, not represented as desired guarantees.
type secretJourneyAPI struct {
	mu          sync.Mutex
	body        string
	status      int
	afterWrite  string
	afterStatus int
	writes      []map[string]any
}

func (a *secretJourneyAPI) serve(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	a.mu.Lock()
	defer a.mu.Unlock()
	if r.Method == http.MethodPost {
		var req UpdateResourceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		var cfg map[string]any
		if err := yaml.Unmarshal([]byte(req.Configuration), &cfg); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		a.writes = append(a.writes, cfg)
		a.body, a.status = a.afterWrite, a.afterStatus
		_, _ = w.Write([]byte(`{"id":"journey-1"}`))
		return
	}
	if r.Method != http.MethodGet {
		t.Errorf("unexpected method %s", r.Method)
		w.WriteHeader(405)
		return
	}
	if a.status != 0 {
		w.WriteHeader(a.status)
	}
	_, _ = w.Write([]byte(a.body))
}

func (a *secretJourneyAPI) observe(body string, status int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.body, a.status = body, status
}
func (a *secretJourneyAPI) count() int { a.mu.Lock(); defer a.mu.Unlock(); return len(a.writes) }
func (a *secretJourneyAPI) last(t *testing.T) map[string]any {
	t.Helper()
	a.mu.Lock()
	defer a.mu.Unlock()
	require.NotEmpty(t, a.writes)
	return a.writes[len(a.writes)-1]
}
func (a *secretJourneyAPI) server(t *testing.T) integration.Server {
	return newReadServer(t, func(w http.ResponseWriter, r *http.Request) { a.serve(t, w, r) })
}

func journeyBody(digests map[string]string, keys []string, cfg map[string]any) string {
	body := map[string]any{"id": "journey-1", "name": "journey", "integrationType": "snowflake"}
	if cfg != nil {
		body["configuration"] = cfg
	}
	if digests != nil {
		body["redactedDigests"] = digests
	}
	if keys != nil {
		body["redactedKeys"] = keys
	}
	b, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return string(b)
}
func journeyHeld(digest string) string {
	return journeyBody(map[string]string{"databasePassword": digest}, []string{"databasePassword"}, map[string]any{})
}
func journeyClear() string { return journeyBody(nil, nil, map[string]any{"databasePassword": ""}) }
func journeyArgs(password *string) property.Map {
	args := map[string]property.Value{"name": property.New("journey"), "type": property.New("snowflake")}
	if password != nil {
		args["password"] = property.New(*password).WithSecret(true)
	}
	return property.NewMap(args)
}
func journeyState(inputs property.Map, field string, baseline map[string]string) property.Map {
	state := map[string]property.Value{}
	for k, v := range inputs.All {
		state[k] = v
	}
	if baseline != nil {
		m := map[string]property.Value{}
		for k, v := range baseline {
			m[k] = property.New(v)
		}
		state[field] = property.New(m)
	}
	return property.NewMap(state)
}
func journeyBaseline(t *testing.T, state property.Map, want map[string]string) {
	t.Helper()
	value, ok := state.GetOk("appliedDigests")
	if want == nil {
		require.False(t, ok, "unknown baseline must remain absent")
		return
	}
	require.True(t, ok, "known-empty baseline must survive provider encoding")
	require.True(t, value.IsMap())
	got := map[string]string{}
	for k, v := range value.AsMap().All {
		got[k] = v.AsString()
	}
	require.Equal(t, want, got)
	_, legacy := state.GetOk("redactedDigests")
	assert.False(t, legacy, "migrated state must not carry the old property forward")
}
func journeyRead(t *testing.T, srv integration.Server, inputs, state property.Map) p.ReadResponse {
	t.Helper()
	out, err := srv.Read(p.ReadRequest{ID: "journey-1", Urn: urnFor("adaptive:index:Resource"), Inputs: inputs, Properties: state})
	require.NoError(t, err)
	_, stateOnly := out.Inputs.GetOk("appliedDigests")
	require.False(t, stateOnly, "baseline is output-only")
	return out
}
func journeyDiff(t *testing.T, srv integration.Server, read p.ReadResponse, desired property.Map, changed bool) {
	t.Helper()
	out, err := srv.Diff(p.DiffRequest{ID: "journey-1", Urn: urnFor("adaptive:index:Resource"), State: read.Properties, OldInputs: read.Inputs, Inputs: desired})
	require.NoError(t, err)
	require.Equal(t, changed, out.HasChanges)
}
func journeyUpdate(t *testing.T, srv integration.Server, read p.ReadResponse, desired property.Map) p.UpdateResponse {
	t.Helper()
	out, err := srv.Update(p.UpdateRequest{ID: "journey-1", Urn: urnFor("adaptive:index:Resource"), State: read.Properties, OldInputs: read.Inputs, Inputs: desired})
	require.NoError(t, err)
	return out
}

func TestSecretJourneyMissingCapture(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"failed_read", "", 500},
		{"missing_configuration_and_metadata", journeyBody(nil, nil, nil), 0},
		{"old_server_withheld_without_fingerprint", journeyBody(nil, []string{"databasePassword"}, map[string]any{}), 0},
		{"partial_fingerprints", journeyBody(map[string]string{"clientcert": "cert-A"}, []string{"databasePassword", "clientcert"}, map[string]any{}), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &secretJourneyAPI{afterWrite: tc.body, afterStatus: tc.status}
			srv := api.server(t)
			desired := journeyArgs(strp("synthetic-program"))
			created, err := srv.Create(p.CreateRequest{Urn: urnFor("adaptive:index:Resource"), Properties: desired})
			require.NoError(t, err)
			require.Equal(t, "journey-1", created.ID)
			require.Equal(t, 1, api.count())
			journeyBaseline(t, created.Properties, nil)
			api.observe(journeyHeld("remote-B"), 0)
			read := journeyRead(t, srv, desired, created.Properties)
			journeyBaseline(t, read.Properties, map[string]string{"databasePassword": "remote-B"})
			journeyDiff(t, srv, read, desired, false)
			t.Log("LIMITATION: unknown post-create baseline seeds first complete observation; historical changes before that observation cannot be proven")
			api.observe(journeyHeld("remote-C"), 0)
			for i := 0; i < 3; i++ {
				read = journeyRead(t, srv, read.Inputs, read.Properties)
				journeyBaseline(t, read.Properties, map[string]string{"databasePassword": "remote-B"})
				journeyDiff(t, srv, read, desired, true)
			}
			before := api.count()
			_, err = srv.Update(p.UpdateRequest{ID: "journey-1", Urn: urnFor("adaptive:index:Resource"), State: read.Properties, Inputs: journeyArgs(nil)})
			require.ErrorContains(t, err, "server-held values for password")
			require.Equal(t, before, api.count())
		})
	}
	t.Run("update_failed_capture_retains_prior_not_guard", func(t *testing.T) {
		api := &secretJourneyAPI{body: journeyHeld("guard-B"), afterStatus: 500}
		srv := api.server(t)
		desired := journeyArgs(strp("synthetic-program"))
		state := journeyState(desired, "appliedDigests", map[string]string{"databasePassword": "write-A"})
		updated := journeyUpdate(t, srv, p.ReadResponse{Properties: state, Inputs: desired}, desired)
		journeyBaseline(t, updated.Properties, map[string]string{"databasePassword": "write-A"})
		require.Equal(t, "synthetic-program", api.last(t)["databasePassword"])
		api.observe(journeyHeld("write-C"), 0)
		read := journeyRead(t, srv, desired, updated.Properties)
		journeyBaseline(t, read.Properties, map[string]string{"databasePassword": "write-A"})
		journeyDiff(t, srv, read, desired, true)
		t.Log("LIMITATION: retaining prior evidence after a failed post-update capture may conservatively report the provider's own write as drift")
	})
}

func TestSecretJourneyBaselineOrigin(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		baseline    map[string]string
		importing   bool
		changed     bool
	}{
		{"legacy_absent", "appliedDigests", nil, false, false},
		{"current_known_empty", "appliedDigests", map[string]string{}, false, true},
		{"v1_absent", "redactedDigests", nil, false, false},
		{"v1_known_empty", "redactedDigests", map[string]string{}, false, true},
		{"v1_recorded", "redactedDigests", map[string]string{"databasePassword": "write-A"}, false, true},
		{"import_absent", "appliedDigests", nil, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &secretJourneyAPI{body: journeyHeld("remote-B"), afterWrite: journeyClear()}
			srv := api.server(t)
			desired := journeyArgs(nil)
			inputs := desired
			state := journeyState(desired, tc.field, tc.baseline)
			if tc.importing {
				inputs = property.Map{}
				state = property.Map{}
			}
			read := journeyRead(t, srv, inputs, state)
			want := tc.baseline
			if want == nil {
				want = map[string]string{"databasePassword": "remote-B"}
			}
			journeyBaseline(t, read.Properties, want)
			journeyDiff(t, srv, read, desired, tc.changed)
			// Checkpoint round-trips must preserve absent vs initialized-empty semantics.
			for i := 0; i < 2; i++ {
				read = journeyRead(t, srv, read.Inputs, read.Properties)
				journeyBaseline(t, read.Properties, want)
				journeyDiff(t, srv, read, desired, tc.changed)
			}
			api.observe(journeyHeld("remote-C"), 0)
			read = journeyRead(t, srv, read.Inputs, read.Properties)
			journeyDiff(t, srv, read, desired, true)
			journeyBaseline(t, read.Properties, want)
			_, err := srv.Update(p.UpdateRequest{ID: "journey-1", Urn: urnFor("adaptive:index:Resource"), State: read.Properties, Inputs: desired})
			require.ErrorContains(t, err, "server-held values for password")
			require.Zero(t, api.count())
			clear := journeyArgs(strp(""))
			updated := journeyUpdate(t, srv, read, clear)
			require.Equal(t, "", api.last(t)["databasePassword"])
			journeyBaseline(t, updated.Properties, map[string]string{})
			read = journeyRead(t, srv, clear, updated.Properties)
			journeyDiff(t, srv, read, clear, false)
		})
	}
}

func TestSecretJourneyNullV1Baseline(t *testing.T) {
	api := &secretJourneyAPI{body: journeyHeld("remote-B")}
	srv := api.server(t)
	desired := journeyArgs(strp("synthetic-program"))
	values := map[string]property.Value{}
	for key, value := range desired.All {
		values[key] = value
	}
	values["redactedDigests"] = property.Value{}
	state := property.NewMap(values)
	inputs := desired
	for cycle := 0; cycle < 3; cycle++ {
		read := journeyRead(t, srv, inputs, state)
		journeyBaseline(t, read.Properties, map[string]string{"databasePassword": "remote-B"})
		journeyDiff(t, srv, read, desired, false)
		inputs, state = read.Inputs, read.Properties
	}
}

func TestSecretJourneyOldServerMetadataRecovery(t *testing.T) {
	for _, known := range []bool{false, true} {
		name := "unknown_baseline"
		if known {
			name = "recorded_baseline"
		}
		t.Run(name, func(t *testing.T) {
			desired := journeyArgs(strp("synthetic-program"))
			var baseline map[string]string
			if known {
				baseline = map[string]string{"databasePassword": "write-A"}
			}
			api := &secretJourneyAPI{body: journeyBody(nil, []string{"databasePassword"}, map[string]any{}), afterWrite: journeyClear()}
			srv := api.server(t)
			read := journeyRead(t, srv, desired, journeyState(desired, "appliedDigests", baseline))
			journeyBaseline(t, read.Properties, baseline)
			journeyDiff(t, srv, read, desired, false)
			_, err := srv.Update(p.UpdateRequest{ID: "journey-1", Urn: urnFor("adaptive:index:Resource"), State: read.Properties, Inputs: journeyArgs(nil)})
			require.ErrorContains(t, err, "server-held values for password")
			require.Zero(t, api.count())
			// Losing even redactedKeys must fail closed when state knew a held value.
			api.observe(journeyBody(nil, nil, map[string]any{}), 0)
			_, err = srv.Update(p.UpdateRequest{ID: "journey-1", Urn: urnFor("adaptive:index:Resource"), State: read.Properties, Inputs: desired})
			require.ErrorContains(t, err, "cannot verify server-held values")
			require.Zero(t, api.count())
			api.observe(journeyHeld("remote-B"), 0)
			read = journeyRead(t, srv, read.Inputs, read.Properties)
			journeyDiff(t, srv, read, desired, known)
			want := baseline
			if !known {
				want = map[string]string{"databasePassword": "remote-B"}
			}
			journeyBaseline(t, read.Properties, want)
			api.observe(journeyBody(nil, []string{"databasePassword"}, map[string]any{}), 0)
			clear := journeyArgs(strp(""))
			updated := journeyUpdate(t, srv, read, clear)
			journeyBaseline(t, updated.Properties, map[string]string{})
			require.Equal(t, "", api.last(t)["databasePassword"])
			read = journeyRead(t, srv, clear, updated.Properties)
			journeyDiff(t, srv, read, clear, false)
			t.Log("LIMITATION: no fingerprints means no digest comparison during metadata loss; unknown baselines seed on recovery, recorded baselines retain historical evidence")
		})
	}
}

func TestSecretJourneyAlreadySeededExplicitClear(t *testing.T) {
	desired := journeyArgs(strp(""))
	api := &secretJourneyAPI{body: journeyHeld("wrongly-seeded-A"), afterWrite: journeyClear()}
	srv := api.server(t)
	read := p.ReadResponse{Inputs: desired, Properties: journeyState(desired, "appliedDigests", map[string]string{"databasePassword": "wrongly-seeded-A"})}
	for i := 0; i < 3; i++ {
		read = journeyRead(t, srv, read.Inputs, read.Properties)
		journeyBaseline(t, read.Properties, map[string]string{})
		journeyDiff(t, srv, read, desired, true)
		marker := read.Inputs.Get("password")
		require.True(t, marker.Secret())
		require.Equal(t, "wrongly-seeded-A", marker.AsString())
	}
	updated := journeyUpdate(t, srv, read, desired)
	require.Equal(t, "", api.last(t)["databasePassword"])
	journeyBaseline(t, updated.Properties, map[string]string{})
	for i := 0; i < 3; i++ {
		read = journeyRead(t, srv, desired, updated.Properties)
		journeyDiff(t, srv, read, desired, false)
	}
	api.observe(journeyHeld("readded-B"), 0)
	read = journeyRead(t, srv, read.Inputs, read.Properties)
	journeyDiff(t, srv, read, desired, true)
	journeyBaseline(t, read.Properties, map[string]string{})
	require.Equal(t, 1, api.count(), "reads and diffs must not write")
}

func TestSecretJourneyUnattributableWarnings(t *testing.T) {
	for _, tc := range []struct{ name, key, warning string }{
		{"unmapped", "unknownCredential", "maps to no single argument"},
		{"nested", "targets[0].password", "maps to no single argument"},
		{"mapped_nonsecret", "databaseAccount", "not declared secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			old := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
			t.Cleanup(func() { slog.SetDefault(old) })
			desired := journeyArgs(nil)
			api := &secretJourneyAPI{body: journeyBody(map[string]string{tc.key: "opaque-new-marker"}, []string{tc.key}, map[string]any{})}
			srv := api.server(t)
			read := p.ReadResponse{Inputs: desired, Properties: journeyState(desired, "appliedDigests", map[string]string{tc.key: "opaque-old-marker"})}
			for i := 0; i < 2; i++ {
				read = journeyRead(t, srv, read.Inputs, read.Properties)
				journeyDiff(t, srv, read, desired, false)
				journeyBaseline(t, read.Properties, map[string]string{tc.key: "opaque-old-marker"})
			}
			require.Contains(t, logs.String(), tc.key)
			require.Contains(t, logs.String(), tc.warning)
			require.Equal(t, 2, strings.Count(logs.String(), tc.warning))
			require.NotContains(t, logs.String(), "opaque-new-marker")
			require.NotContains(t, logs.String(), "opaque-old-marker")
			require.Zero(t, api.count())
			t.Log("LIMITATION: warning repeats, but unrepresentable or nonsecret key drift cannot become a flat secret argument diff")
		})
	}
}

func TestSecretJourneyFingerprintLimitations(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantChange bool
	}{
		{"same_fingerprint_changed_secret", journeyHeld("write-A"), false},
		{"rotated_fingerprint_unchanged_secret", journeyHeld("rotated-B"), true},
		{"missing_current_digest_and_no_clear_ack", journeyBody(nil, nil, map[string]any{}), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			desired := journeyArgs(strp("synthetic-program"))
			api := &secretJourneyAPI{body: tc.body}
			srv := api.server(t)
			read := journeyRead(t, srv, desired, journeyState(desired, "appliedDigests", map[string]string{"databasePassword": "write-A"}))
			journeyBaseline(t, read.Properties, map[string]string{"databasePassword": "write-A"})
			journeyDiff(t, srv, read, desired, tc.wantChange)
			require.Zero(t, api.count())
			t.Log("LIMITATION: opaque fingerprints supply no plaintext equality, historical completeness, or cross-scope comparability proof; this is a controlled DTO observation, not a backend collision/rotation experiment")
		})
	}
	t.Run("unknown_held_secret_without_any_metadata", func(t *testing.T) {
		desired := journeyArgs(nil)
		api := &secretJourneyAPI{body: journeyBody(nil, nil, map[string]any{}), afterWrite: journeyClear()}
		srv := api.server(t)
		state := journeyState(desired, "appliedDigests", map[string]string{})
		journeyUpdate(t, srv, p.ReadResponse{Inputs: desired, Properties: state}, desired)
		require.Equal(t, 1, api.count())
		require.Equal(t, "", api.last(t)["databasePassword"])
		t.Log("LIMITATION: an unknown held secret entirely omitted from an otherwise valid DTO is indistinguishable from absence; without a completeness token the guard permits this update")
	})
}
