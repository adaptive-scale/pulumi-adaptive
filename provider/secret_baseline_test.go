package adaptive

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestResourceWriteDigestBaseline(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		for _, tc := range []struct {
			name   string
			body   string
			status int
			want   map[string]string
			retain bool
		}{
			{name: "confirmed empty configuration", body: `{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":{"databasePassword":""}}`, want: map[string]string{}},
			{name: "complete fingerprints", body: `{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","redactedKeys":["databasePassword"],"redactedDigests":{"databasePassword":"after"}}`, want: map[string]string{"databasePassword": "after"}},
			{name: "older server with held secret", body: `{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":{"hostname":"db"},"redactedKeys":["databasePassword"]}`, retain: true},
			{name: "partial fingerprint metadata", body: `{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","redactedKeys":["databasePassword","clientcert"],"redactedDigests":{"clientcert":"cert"}}`, retain: true},
			{name: "missing configuration and metadata", body: `{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake"}`, retain: true},
			{name: "absent prior secret is not an acknowledged clear", body: `{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":{"hostname":"db"}}`, want: map[string]string{}, retain: true},
			{name: "failed capture", status: http.StatusInternalServerError, retain: true},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				var written atomic.Bool
				srv := newReadServer(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodPost {
						written.Store(true)
						_, _ = w.Write([]byte(`{"id":"snowflake-1"}`))
						return
					}
					if !written.Load() {
						_, _ = w.Write([]byte(`{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":{},"redactedKeys":["databasePassword"],"redactedDigests":{"databasePassword":"guard-only-live"}}`))
						return
					}
					if tc.status != 0 {
						w.WriteHeader(tc.status)
					}
					_, _ = w.Write([]byte(tc.body))
				})
				inputs := property.NewMap(map[string]property.Value{"name": property.New("snowflake-test"), "type": property.New("snowflake"), "password": property.New("").WithSecret(true), "clientcert": property.New("certificate").WithSecret(true)})
				var output property.Map
				want := tc.want
				if operation == "create" {
					created, err := srv.Create(p.CreateRequest{Urn: urnFor("adaptive:index:Resource"), Properties: inputs})
					require.NoError(t, err)
					require.Equal(t, "snowflake-1", created.ID)
					output = created.Properties
				} else {
					previous := map[string]property.Value{"name": property.New("snowflake-test"), "type": property.New("snowflake"), "appliedDigests": property.New(map[string]property.Value{"databasePassword": property.New("before")})}
					updated, err := srv.Update(p.UpdateRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), State: property.NewMap(previous), Inputs: inputs})
					require.NoError(t, err)
					output = updated.Properties
					if tc.retain {
						want = map[string]string{"databasePassword": "before"}
					}
				}
				value, ok := output.GetOk("appliedDigests")
				if want == nil {
					require.False(t, ok)
					return
				}
				require.True(t, ok, "known-empty baselines must encode as a map, not disappear")
				require.True(t, value.IsMap())
				got := map[string]string{}
				for key, digest := range value.AsMap().All {
					got[key] = digest.AsString()
				}
				require.Equal(t, want, got)
			})
		}
	}
}

func TestResourceUpdateRequiresUsableGuardObservation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"forbidden", 403, ""},
		{"server error", 500, ""},
		{"not found", 404, ""},
		{"empty envelope", 200, `{}`},
		{"null envelope", 200, `null`},
		{"missing identity", 200, `{"configuration":{}}`},
		{"missing configuration", 200, `{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","redactedKeys":["databasePassword"]}`},
		{"null configuration", 200, `{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":null}`},
		{"unusable configuration", 200, `{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":"invalid"}`},
		{"missing held-key evidence", 200, `{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":{}}`},
		{"unusable held-key value", 200, `{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":{"databasePassword":{}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var writes atomic.Int64
			srv := newReadServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					writes.Add(1)
					_, _ = w.Write([]byte(`{"id":"snowflake-1"}`))
					return
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			inputs := property.NewMap(map[string]property.Value{"name": property.New("snowflake-test"), "type": property.New("snowflake")})
			state := property.NewMap(map[string]property.Value{"name": property.New("snowflake-test"), "type": property.New("snowflake"), "password": property.New("prior-held-marker").WithSecret(true), "appliedDigests": property.New(map[string]property.Value{})})
			_, err := srv.Update(p.UpdateRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), State: state, Inputs: inputs})
			require.Zero(t, writes.Load(), "unverified omission must never reach POST")
			require.ErrorContains(t, err, "cannot verify server-held values")
			require.NotContains(t, err.Error(), "prior-held-marker")
		})
	}
}

func TestResourceReadPrunedBaselineDetectsReaddition(t *testing.T) {
	var restored atomic.Bool
	srv := newReadServer(t, func(w http.ResponseWriter, _ *http.Request) {
		if restored.Load() {
			_, _ = w.Write([]byte(`{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":{},"redactedKeys":["databasePassword"],"redactedDigests":{"databasePassword":"new-remote"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":{"databasePassword":""}}`))
	})
	inputs := property.NewMap(map[string]property.Value{"name": property.New("snowflake-test"), "type": property.New("snowflake")})
	state := property.NewMap(map[string]property.Value{"name": property.New("snowflake-test"), "type": property.New("snowflake"), "appliedDigests": property.New(map[string]property.Value{"databasePassword": property.New("previous")})})
	cleared, err := srv.Read(p.ReadRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), Inputs: inputs, Properties: state})
	require.NoError(t, err)
	baseline, ok := cleared.Properties.GetOk("appliedDigests")
	require.True(t, ok)
	require.True(t, baseline.IsMap())
	require.Zero(t, baseline.AsMap().Len())
	restored.Store(true)
	for cycle := 1; cycle <= 3; cycle++ {
		read, err := srv.Read(p.ReadRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), Inputs: cleared.Inputs, Properties: cleared.Properties})
		require.NoError(t, err)
		diff, err := srv.Diff(p.DiffRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), State: read.Properties, OldInputs: read.Inputs, Inputs: inputs})
		require.NoError(t, err)
		require.True(t, diff.HasChanges, "cycle %d", cycle)
		cleared = read
	}
}

func TestResourceReadEmptySecretBaseline(t *testing.T) {
	for _, tc := range []struct {
		name      string
		password  *string
		baseline  map[string]string
		wantDrift bool
	}{
		{"legacy explicit clear", strp(""), nil, true},
		{"known empty baseline", strp(""), map[string]string{}, true},
		{"previously seeded explicit clear", strp(""), map[string]string{"databasePassword": "remote-one"}, true},
		{"legacy managed secret seeds", strp("managed"), nil, false},
		{"legacy omitted secret seeds", nil, nil, false},
		{"known empty baseline omitted secret", nil, map[string]string{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := newReadServer(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":{},"redactedKeys":["databasePassword"],"redactedDigests":{"databasePassword":"remote-one"}}`))
			})
			args := map[string]property.Value{"name": property.New("snowflake-test"), "type": property.New("snowflake")}
			if tc.password != nil {
				args["password"] = property.New(*tc.password).WithSecret(true)
			}
			inputs := property.NewMap(args)
			stateArgs := make(map[string]property.Value)
			for k, v := range args {
				stateArgs[k] = v
			}
			if tc.baseline != nil {
				baseline := make(map[string]property.Value)
				for k, v := range tc.baseline {
					baseline[k] = property.New(v)
				}
				stateArgs["appliedDigests"] = property.New(baseline)
			}
			oldInputs, state := inputs, property.NewMap(stateArgs)
			for cycle := 1; cycle <= 3; cycle++ {
				read, err := srv.Read(p.ReadRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), Inputs: oldInputs, Properties: state})
				require.NoError(t, err)
				diff, err := srv.Diff(p.DiffRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), State: read.Properties, OldInputs: read.Inputs, Inputs: inputs})
				require.NoError(t, err)
				require.Equal(t, tc.wantDrift, diff.HasChanges, "cycle %d", cycle)
				if tc.wantDrift {
					baseline, ok := read.Properties.GetOk("appliedDigests")
					require.True(t, ok, "initialized baseline must survive encoding")
					require.True(t, baseline.IsMap())
					_, adopted := baseline.AsMap().GetOk("databasePassword")
					require.False(t, adopted, "must not adopt the conflicting remote digest")
				}
				oldInputs, state = read.Inputs, read.Properties
			}
		})
	}
}

func TestResourceUpdateGuardsHeldKeysWithoutFingerprints(t *testing.T) {
	var writes atomic.Int64
	srv := newReadServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			writes.Add(1)
			_, _ = w.Write([]byte(`{"id":"snowflake-1"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":{"hostname":"db"},"redactedKeys":["databasePassword"]}`))
	})
	inputs := property.NewMap(map[string]property.Value{"name": property.New("snowflake-test"), "type": property.New("snowflake")})
	_, err := srv.Update(p.UpdateRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), State: inputs, Inputs: inputs})
	require.ErrorContains(t, err, "server-held values for password")
	require.Zero(t, writes.Load())
	clear := property.NewMap(map[string]property.Value{"name": property.New("snowflake-test"), "type": property.New("snowflake"), "password": property.New("").WithSecret(true)})
	_, err = srv.Update(p.UpdateRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), State: inputs, Inputs: clear})
	require.NoError(t, err, "redactedKeys without digests must support explicit clear")
	require.EqualValues(t, 1, writes.Load())
}

func TestResourceUpdateNewHeldSecret(t *testing.T) {
	for _, clear := range []bool{false, true} {
		t.Run(map[bool]string{false: "omission rejected", true: "explicit clear converges"}[clear], func(t *testing.T) {
			var held atomic.Bool
			held.Store(true)
			var writes atomic.Int64
			srv := newReadServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					writes.Add(1)
					var request UpdateResourceRequest
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					var cfg map[string]any
					if err := yaml.Unmarshal([]byte(request.Configuration), &cfg); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					value, ok := cfg["databasePassword"]
					if !ok || value != "" {
						t.Error("write must contain an actual empty password, never a fingerprint marker")
					}
					held.Store(false)
					_, _ = w.Write([]byte(`{"id":"snowflake-1"}`))
					return
				}
				body := map[string]any{"id": "snowflake-1", "name": "snowflake-test", "integrationType": "snowflake", "configuration": map[string]any{"databasePassword": ""}}
				if held.Load() {
					body["configuration"] = map[string]any{}
					body["redactedKeys"] = []string{"databasePassword"}
					body["redactedDigests"] = map[string]string{"databasePassword": "remote-one"}
				}
				_ = json.NewEncoder(w).Encode(body)
			})
			original := property.NewMap(map[string]property.Value{"name": property.New("snowflake-test"), "type": property.New("snowflake"), "password": property.New("").WithSecret(true)})
			read, err := srv.Read(p.ReadRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), Inputs: original, Properties: original})
			require.NoError(t, err)
			args := map[string]property.Value{"name": property.New("snowflake-test"), "type": property.New("snowflake")}
			if clear {
				args["password"] = property.New("").WithSecret(true)
			}
			inputs := property.NewMap(args)
			updated, err := srv.Update(p.UpdateRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), State: read.Properties, OldInputs: read.Inputs, Inputs: inputs})
			if !clear {
				require.Error(t, err)
				require.True(t, strings.Contains(err.Error(), "server-held values for password"))
				require.Zero(t, writes.Load(), "unsafe omission must not reach the server")
				return
			}
			require.NoError(t, err)
			require.EqualValues(t, 1, writes.Load())
			after, err := srv.Read(p.ReadRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), Inputs: inputs, Properties: updated.Properties})
			require.NoError(t, err)
			diff, err := srv.Diff(p.DiffRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), State: after.Properties, OldInputs: after.Inputs, Inputs: inputs})
			require.NoError(t, err)
			require.False(t, diff.HasChanges, "explicit clear must converge")
		})
	}
}
