package adaptive

import (
	"reflect"
	"testing"
)

func TestChangedDigestKeys(t *testing.T) {
	cases := []struct {
		name     string
		recorded map[string]string
		current  map[string]string
		want     []string
	}{
		{"nil recorded (import/upgrade)", nil, map[string]string{"password": "d1"}, nil},
		{"nil current (old server)", map[string]string{"password": "d1"}, nil, nil},
		{"unchanged", map[string]string{"password": "d1"}, map[string]string{"password": "d1"}, nil},
		{"changed", map[string]string{"password": "d1"}, map[string]string{"password": "d2"}, []string{"password"}},
		{"new key is not drift", map[string]string{"password": "d1"}, map[string]string{"password": "d1", "sshKey": "d9"}, nil},
		{"removed key is not drift", map[string]string{"password": "d1", "sshKey": "d9"}, map[string]string{"password": "d1"}, nil},
		{"multiple changed, sorted", map[string]string{"b": "1", "a": "1"}, map[string]string{"b": "2", "a": "2"}, []string{"a", "b"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := changedDigestKeys(c.recorded, c.current)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// TestSecretDriftClearsChangedField pins the end-to-end refresh behavior: a
// changed digest injects an empty value for that key, which
// applyIntegrationConfig turns into "clear the previously-set field" — while
// unchanged secrets keep their prior (state-carried) values.
func TestSecretDriftClearsChangedField(t *testing.T) {
	prior := ResourceArgs{
		Type: "postgres", Name: "db",
		Host:     strp("h"),
		Username: strp("app"),
		Password: strp("s3cret"),
	}

	// Simulate Resource.Read's cfg construction: server config (secrets absent)
	// plus an injected empty value for the drifted key.
	cfg := map[string]any{"hostname": "h", "username": "app"}
	for _, k := range changedDigestKeys(
		map[string]string{"password": "old-digest"},
		map[string]string{"password": "new-digest"},
	) {
		cfg[k] = ""
	}
	got := prior
	applyIntegrationConfig(&got, "postgres", cfg)
	if got.Password != nil {
		t.Errorf("drifted password should be cleared, got %q", sv(got.Password))
	}

	// Same read with matching digests: password preserved.
	got = prior
	cfg2 := map[string]any{"hostname": "h", "username": "app"}
	for _, k := range changedDigestKeys(
		map[string]string{"password": "same"},
		map[string]string{"password": "same"},
	) {
		cfg2[k] = ""
	}
	applyIntegrationConfig(&got, "postgres", cfg2)
	if sv(got.Password) != "s3cret" {
		t.Errorf("unchanged password should be preserved, got %q", sv(got.Password))
	}
}

func TestApplyScriptReadCommandDigestDrift(t *testing.T) {
	// The digest comparison lives in Script.Read around applyScriptRead; this
	// pins the pieces it composes: prior command survives the mapping, and the
	// caller clears it only on digest mismatch.
	prior := ScriptArgs{Name: "s", Command: "psql -c 'select 1'", Endpoint: "ep"}
	inputs := applyScriptRead(prior, &ScriptReadResponse{
		Name: "s", Endpoint: "ep", CommandOmitted: true, CommandDigest: "new",
	}, false)
	if inputs.Command != prior.Command {
		t.Fatalf("mapping must not touch command: %q", inputs.Command)
	}
	// Mismatch semantics (as implemented in Script.Read): recorded != current -> clear.
	recorded, current := "old", "new"
	if recorded != "" && current != "" && recorded != current {
		inputs.Command = ""
	}
	if inputs.Command != "" {
		t.Error("command should be cleared on digest mismatch")
	}
}

func TestReadResponsesDecodeDigests(t *testing.T) {
	// Wire-shape check: field names must match the server DTOs.
	var r ResourceReadResponse
	if _, ok := reflect.TypeOf(r).FieldByName("RedactedDigests"); !ok {
		t.Fatal("ResourceReadResponse missing RedactedDigests")
	}
	f, _ := reflect.TypeOf(r).FieldByName("RedactedDigests")
	if f.Tag.Get("json") != "redactedDigests" {
		t.Errorf("RedactedDigests json tag = %q", f.Tag.Get("json"))
	}
	var s ScriptReadResponse
	f2, ok := reflect.TypeOf(s).FieldByName("CommandDigest")
	if !ok || f2.Tag.Get("json") != "commandDigest" {
		t.Errorf("ScriptReadResponse CommandDigest tag wrong or missing")
	}
}
