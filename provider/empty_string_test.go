package adaptive

import (
	"net/http"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

func TestSnowflakeExplicitEmptyPasswordRefreshHasNoDiff(t *testing.T) {
	srv := newReadServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/terraform/resource/read/snowflake-1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":{"databaseAccount":"example-account","databaseUsername":"example-user","databasePassword":""}}`))
	})
	inputs := property.NewMap(map[string]property.Value{
		"name":     property.New("snowflake-test"),
		"type":     property.New("snowflake"),
		"hostname": property.New("example-account"),
		"username": property.New("example-user"),
		"password": property.New("").WithSecret(true),
	})
	state := inputs
	oldInputs := inputs
	for cycle := 0; cycle < 3; cycle++ {
		read, err := srv.Read(p.ReadRequest{
			ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"),
			Inputs: oldInputs, Properties: state,
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
			t.Fatalf("refresh cycle %d generated a diff for an unchanged empty password: %v", cycle, diff.DetailedDiff)
		}
		oldInputs, state = read.Inputs, read.Properties
	}
}

func TestSetStrEmptyReconciliation(t *testing.T) {
	cases := []struct {
		name  string
		prior *string
		cfg   map[string]any
		want  *string
	}{
		{"omitted stays omitted", nil, map[string]any{"value": ""}, nil},
		{"explicit empty stays empty", strp(""), map[string]any{"value": ""}, strp("")},
		{"remote clear removes prior value", strp("previous"), map[string]any{"value": ""}, nil},
		{"withheld value is preserved", strp("previous"), nil, strp("previous")},
		{"withheld empty is preserved", strp(""), nil, strp("")},
		{"remote value is adopted", nil, map[string]any{"value": "remote"}, strp("remote")},
		{"remote value replaces empty", strp(""), map[string]any{"value": "remote"}, strp("remote")},
		{"null preserves explicit empty", strp(""), map[string]any{"value": nil}, strp("")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.prior
			setStr(&got, tc.cfg, "value")
			if (got == nil) != (tc.want == nil) || sv(got) != sv(tc.want) {
				t.Fatalf("reconciliation = %v (%q), want %v (%q)", got != nil, sv(got), tc.want != nil, sv(tc.want))
			}
		})
	}
}

func TestApplyIntegrationConfigPreservesExplicitEmptyTLSSecrets(t *testing.T) {
	args := ResourceArgs{
		Type: "postgres", TLSRootCert: strp(""), TLSCertFile: strp(""), TLSKeyFile: strp(""),
	}
	applyIntegrationConfig(&args, "postgres", map[string]any{"rootCert": "", "crtText": "", "keyText": ""})
	for name, value := range map[string]*string{
		"tlsRootCert": args.TLSRootCert, "tlsCertFile": args.TLSCertFile, "tlsKeyFile": args.TLSKeyFile,
	} {
		if value == nil || *value != "" {
			t.Errorf("%s must retain the program's explicit empty value", name)
		}
	}
}

func TestSnowflakeEmptyPasswordDoesNotHideRemoteSecret(t *testing.T) {
	srv := newReadServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"snowflake-1","name":"snowflake-test","integrationType":"snowflake","configuration":{},"redactedKeys":["databasePassword"],"redactedDigests":{"databasePassword":"remote-password-digest","clientcert":"unchanged-cert-digest"}}`))
	})
	inputs := property.NewMap(map[string]property.Value{
		"name": property.New("snowflake-test"), "type": property.New("snowflake"),
		"password": property.New("").WithSecret(true),
	})
	state := property.NewMap(map[string]property.Value{
		"name": property.New("snowflake-test"), "type": property.New("snowflake"),
		"password":       property.New("").WithSecret(true),
		"appliedDigests": property.New(map[string]property.Value{"clientcert": property.New("unchanged-cert-digest")}),
	})
	read, err := srv.Read(p.ReadRequest{
		ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"), Inputs: inputs, Properties: state,
	})
	if err != nil {
		t.Fatal(err)
	}
	value, ok := read.Inputs.GetOk("password")
	if !ok || value.AsString() != "remote-password-digest" {
		t.Fatal("a remote secret added to an explicitly empty password must still surface as drift")
	}
	diff, err := srv.Diff(p.DiffRequest{
		ID: "snowflake-1", Urn: urnFor("adaptive:index:Resource"),
		State: read.Properties, OldInputs: read.Inputs, Inputs: inputs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !diff.HasChanges {
		t.Fatal("remote credential drift must not be suppressed")
	}
	if _, ok := diff.DetailedDiff["password"]; !ok {
		t.Fatalf("expected a password diff, got %v", diff.DetailedDiff)
	}
}
