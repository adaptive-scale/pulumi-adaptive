package adaptive

import (
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// Recheck the unchanged program on every cycle. Refreshed inputs are observed
// state, not a substitute for the desired configuration in the program.
func TestLogForwarderUnchangedProgramConverges(t *testing.T) {
	for _, tc := range []struct{ name, program, persisted string }{
		{"reversed", `["logger-z","logger-a"]`, `["logger-a","logger-z"]`},
		{"whitespace", ` [ "logger-z", "logger-a" ] `, `["logger-a","logger-z"]`},
		{"singleton", `["logger-a"]`, "logger-a"},
		{"deduplicated", `["logger-a"," logger-a ",""]`, "logger-a"},
		{"empty-array", `[]`, ""},
		{"null", `null`, ""},
		{"empty-string", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, f := newWorkspaceServer(t)
			program := property.NewMap(map[string]property.Value{"defaultLogForwarder": property.New(tc.program)})
			checked, err := srv.Check(p.CheckRequest{Urn: urnFor(wsTok), Inputs: program})
			if err != nil || len(checked.Failures) != 0 {
				t.Fatalf("initial check: %v %v", err, checked.Failures)
			}
			id, state := create(t, srv, wsTok, checked.Inputs)
			f.mu.Lock()
			f.settings["defaultLogForwarder"] = tc.persisted
			f.mu.Unlock()
			observedInputs := checked.Inputs
			for i := 0; i < 3; i++ {
				refreshed, err := srv.Read(p.ReadRequest{ID: id, Urn: urnFor(wsTok), Inputs: observedInputs, Properties: state})
				if err != nil {
					t.Fatal(err)
				}
				desired, err := srv.Check(p.CheckRequest{Urn: urnFor(wsTok), Inputs: program})
				if err != nil || len(desired.Failures) != 0 {
					t.Fatalf("cycle %d check: %v %v", i, err, desired.Failures)
				}
				diff, err := srv.Diff(p.DiffRequest{ID: id, Urn: urnFor(wsTok), State: refreshed.Properties, Inputs: desired.Inputs, OldInputs: refreshed.Inputs})
				if err != nil || diff.HasChanges {
					t.Fatalf("cycle %d unchanged program differs: %v %+v", i, err, diff)
				}
				observedInputs, state = refreshed.Inputs, refreshed.Properties
			}
		})
	}
}

func TestSsoSyncCheckPreservesUnknownGroups(t *testing.T) {
	for _, groups := range []property.Value{
		property.New(property.Computed),
		property.New(property.NewArray([]property.Value{property.New(property.Computed)})),
	} {
		srv, _ := newWorkspaceServer(t)
		inputs := property.NewMap(map[string]property.Value{
			"providerId": property.New("provider-unknown"),
			"groups":     groups,
		})
		checked, err := srv.Check(p.CheckRequest{Urn: urnFor(ssoTok), Inputs: inputs})
		if err != nil || len(checked.Failures) != 0 {
			t.Fatalf("unknown groups check: %v %v", err, checked.Failures)
		}
		got := checked.Inputs.Get("groups")
		if groups.IsComputed() {
			if !got.IsComputed() {
				t.Fatalf("unknown collection became known: %v", got)
			}
		} else if !got.IsArray() || got.AsArray().Len() != 1 || !got.AsArray().Get(0).IsComputed() {
			t.Fatalf("unknown group element was lost: %v", got)
		}
	}
}
