package adaptive

import (
	"net/http"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// snowflakeWithRemotePassword answers a resource read as the server does once
// someone has set a password in the UI: the value is withheld, only a
// fingerprint is returned.
func snowflakeWithRemotePassword(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte(`{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake",` +
		`"configuration":{"databaseAccount":"example-account","databaseUsername":"example-user"},` +
		`"redactedKeys":["databasePassword"],"redactedDigests":{"databasePassword":"ui-set-digest"}}`))
}

func snowflakeInputs(password *string) property.Map {
	m := map[string]property.Value{
		"name":     property.New("snowflake-test"),
		"type":     property.New("snowflake"),
		"hostname": property.New("example-account"),
		"username": property.New("example-user"),
	}
	if password != nil {
		m["password"] = property.New(*password).WithSecret(true)
	}
	return property.NewMap(m)
}

// The program set password="" so the last write left nothing to fingerprint,
// and appliedDigests is empty. A password set out-of-band afterwards must show
// as drift on every refresh until up re-applies "", not be adopted as baseline.
func TestExplicitEmptySecretSetOutOfBandIsDrift(t *testing.T) {
	srv := newReadServer(t, snowflakeWithRemotePassword)
	inputs := snowflakeInputs(strp(""))
	oldInputs, state := inputs, inputs
	for cycle := 0; cycle < 3; cycle++ {
		read, err := srv.Read(p.ReadRequest{
			ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"),
			Inputs: oldInputs, Properties: state,
		})
		if err != nil {
			t.Fatal(err)
		}
		if v, ok := read.Inputs.GetOk("password"); !ok || v.AsString() != "ui-set-digest" {
			t.Fatalf("cycle %d: password must carry the drift marker, got %v", cycle, v)
		}
		if d, ok := read.Properties.GetOk("appliedDigests"); ok && d.IsMap() {
			if _, has := d.AsMap().GetOk("databasePassword"); has {
				t.Fatalf("cycle %d: the out-of-band fingerprint must not become the baseline", cycle)
			}
		}
		diff, err := srv.Diff(p.DiffRequest{
			ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"),
			State: read.Properties, OldInputs: read.Inputs, Inputs: inputs,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := diff.DetailedDiff["password"]; !diff.HasChanges || !ok {
			t.Fatalf("cycle %d: expected a password diff, got %v", cycle, diff.DetailedDiff)
		}
		oldInputs, state = read.Inputs, read.Properties
	}
}

// Seeding must still apply where there is genuinely no baseline.
func TestEmptyBaselineStillSeedsWhenNotExplicitEmpty(t *testing.T) {
	cases := []struct {
		name     string
		password *string
	}{
		{"password omitted", nil},
		{"password set in program", strp("program-password")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newReadServer(t, snowflakeWithRemotePassword)
			inputs := snowflakeInputs(tc.password)
			read, err := srv.Read(p.ReadRequest{
				ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"),
				Inputs: inputs, Properties: inputs,
			})
			if err != nil {
				t.Fatal(err)
			}
			if v, ok := read.Inputs.GetOk("password"); ok && v.AsString() == "ui-set-digest" {
				t.Fatal("no drift marker expected without an explicit empty input")
			}
			d, ok := read.Properties.GetOk("appliedDigests")
			if !ok || !d.IsMap() {
				t.Fatal("appliedDigests must be seeded")
			}
			if v, has := d.AsMap().GetOk("databasePassword"); !has || v.AsString() != "ui-set-digest" {
				t.Fatal("appliedDigests must be seeded from the server")
			}
		})
	}
}

func TestImportDoesNotReportExplicitEmptyDrift(t *testing.T) {
	srv := newReadServer(t, snowflakeWithRemotePassword)
	read, err := srv.Read(p.ReadRequest{ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource")})
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := read.Inputs.GetOk("password"); ok && v.AsString() == "ui-set-digest" {
		t.Fatal("import must not mark drift")
	}
}

// Once up writes "" again the server holds no secret, and refresh converges.
func TestExplicitEmptySecretConvergesAfterRestore(t *testing.T) {
	srv := newReadServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake",` +
			`"configuration":{"databaseAccount":"example-account","databaseUsername":"example-user","databasePassword":""}}`))
	})
	inputs := snowflakeInputs(strp(""))
	read, err := srv.Read(p.ReadRequest{
		ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), Inputs: inputs, Properties: inputs,
	})
	if err != nil {
		t.Fatal(err)
	}
	diff, err := srv.Diff(p.DiffRequest{
		ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"),
		State: read.Properties, OldInputs: read.Inputs, Inputs: inputs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if diff.HasChanges {
		t.Fatalf("expected convergence, got %v", diff.DetailedDiff)
	}
}
