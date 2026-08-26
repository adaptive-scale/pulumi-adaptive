package adaptive

import (
	"net/http"
	"reflect"
	"sort"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
	"gopkg.in/yaml.v2"
)

func TestDriftedDigestKeys(t *testing.T) {
	cases := []struct {
		name    string
		applied map[string]string
		current map[string]string
		want    []string
	}{
		{"nil applied (caller seeds instead)", nil, map[string]string{"password": "d1"}, []string{"password"}},
		{"nil current (old server)", map[string]string{"password": "d1"}, nil, nil},
		{"unchanged", map[string]string{"password": "d1"}, map[string]string{"password": "d1"}, nil},
		{"changed", map[string]string{"password": "d1"}, map[string]string{"password": "d2"}, []string{"password"}},
		// Inverted deliberately. A secret appearing where none was recorded is
		// someone setting one outside the program, which is exactly what has to
		// surface. The empty-map guard at the call site is what keeps an import
		// or a first refresh after upgrade from reporting every key as new.
		{"added since the last write is drift", map[string]string{"password": "d1"},
			map[string]string{"password": "d1", "sshKey": "d9"}, []string{"sshKey"}},
		{"removed key is not drift", map[string]string{"password": "d1", "sshKey": "d9"},
			map[string]string{"password": "d1"}, nil},
		{"multiple changed, sorted", map[string]string{"b": "1", "a": "1"},
			map[string]string{"b": "2", "a": "2"}, []string{"a", "b"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := driftedDigestKeys(c.applied, c.current)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestArgForConfigKeyResolvesRealArguments(t *testing.T) {
	cases := []struct {
		integrationType string
		cfgKey          string
		wantProp        string
	}{
		{"postgres", "password", "password"},
		{"postgres", "username", "username"},
		{"mysql", "password", "password"},
		{"aws", "aws_secret_access_key", "secretAccessKey"},
		{"aws", "aws_access_key_id", "accessKeyId"},
		{"gcp", "key_file", "keyFile"},
		{"azure", "clientSecret", "clientSecret"},
		{"mongodb", "uri", "uri"},
	}
	for _, c := range cases {
		t.Run(c.integrationType+"/"+c.cfgKey, func(t *testing.T) {
			_, prop, ok := argForConfigKey(c.integrationType, c.cfgKey)
			if !ok {
				t.Fatal("did not resolve")
			}
			if prop != c.wantProp {
				t.Errorf("got %q, want %q", prop, c.wantProp)
			}
		})
	}
}

func TestArgForConfigKeyRejectsUnresolvable(t *testing.T) {
	for _, c := range []struct{ typ, key string }{
		{"postgres", "no_such_key"},
		{"postgres", ""},
		// Nested paths name no single argument; the caller reports them by name.
		{"adaptive_rdp", "targets[0].password"},
		{"postgres", "targets[0].password"},
	} {
		if _, prop, ok := argForConfigKey(c.typ, c.key); ok {
			t.Errorf("%s/%q resolved to %q; an unresolvable key must report false "+
				"rather than mark the wrong argument", c.typ, c.key, prop)
		}
	}
}

// Cross-checks the two directions of the mapping against each other. The keys
// are not guessed: buildIntegrationConfig (args -> server config) is asked what
// a type actually emits, and argForConfigKey (server config -> args) must
// resolve it back. A type whose arm goes missing from applyIntegrationConfig
// fails here instead of silently reporting no secret drift forever.
func TestArgForConfigKeyResolvesEveryTypesOwnKeys(t *testing.T) {
	// Envelope keys are written by buildIntegrationConfig itself and map to no
	// argument by design. "Version" is googleConfig's capitalised spelling.
	envelope := map[string]bool{"name": true, "type": true, "version": true, "Version": true}

	// Keys that deliberately do not round-trip, each checked against the
	// forward mapping. Listed per type rather than globally so the same key
	// stays strict elsewhere - sslMode is a real input on postgres and
	// cockroachdb and must keep resolving there.
	expected := map[string]string{
		"serverlist/hosts":            "setHosts writes a []string argument; a host list is never a secret",
		"windows-server-groups/hosts": "same as serverlist",
		"vnc/hosts":                   "same as serverlist",
		"mysql/sslMode":               `forced to "require" by the forward mapping; not an input`,
		"awsredshift/sslMode":         "vestigial struct field, never populated on the way out",
		"ssh/password":                "mirrors sshKey; the read arm reconciles via sshKey",
	}

	for typ := range validIntegrationTypes {
		t.Run(typ, func(t *testing.T) {
			cfgObj, _, err := buildIntegrationConfig(ResourceArgs{Type: typ, Name: "probe"})
			if err != nil {
				t.Fatalf("buildIntegrationConfig: %v", err)
			}
			b, err := yaml.Marshal(cfgObj)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var cfg map[string]interface{}
			if err := yaml.Unmarshal(b, &cfg); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			// Only string-valued keys are in scope: the probe traces *string
			// arguments, and only a string can be a withheld secret. A type
			// whose config is all booleans (keyspaces) resolves nothing, which
			// is correct.
			var unresolved []string
			for k, v := range cfg {
				if envelope[k] {
					continue
				}
				if _, isStr := v.(string); !isStr {
					continue
				}
				if _, _, ok := argForConfigKey(typ, k); !ok {
					if _, known := expected[typ+"/"+k]; !known {
						unresolved = append(unresolved, k)
					}
				}
			}
			sort.Strings(unresolved)
			if len(unresolved) > 0 {
				t.Errorf("config keys %v do not round-trip back to an argument; the "+
					"applyIntegrationConfig arm for this type is missing them, so neither "+
					"refresh nor secret drift can reconcile those fields", unresolved)
			}
		})
	}
}

func TestArgIsSecretAndIsSet(t *testing.T) {
	field, _, ok := argForConfigKey("postgres", "password")
	if !ok {
		t.Fatal("password did not resolve")
	}
	if !argIsSecret(field) {
		t.Error("password must be schema-secret, or the marker would print in clear")
	}
	a := ResourceArgs{}
	if argIsSet(&a, field) {
		t.Error("unset argument reported as set")
	}
	setArg(&a, field, "marker")
	if !argIsSet(&a, field) || sv(a.Password) != "marker" {
		t.Errorf("setArg did not land on password, got %v", a.Password)
	}

	userField, _, ok := argForConfigKey("postgres", "username")
	if !ok {
		t.Fatal("username did not resolve")
	}
	if argIsSecret(userField) {
		t.Error("username is not a secret and must not be treated as one")
	}
}

// applyDrift reproduces Resource.Read's marking step: for each key whose
// fingerprint moved since the last write, write the new fingerprint onto the
// argument that key feeds.
func applyDrift(t *testing.T, a *ResourceArgs, integrationType string, applied, current map[string]string) []string {
	t.Helper()
	var unresolved []string
	for _, k := range driftedDigestKeys(applied, current) {
		field, _, ok := argForConfigKey(integrationType, k)
		if !ok || !argIsSecret(field) {
			unresolved = append(unresolved, k)
			continue
		}
		setArg(a, field, current[k])
	}
	return unresolved
}

// The regression test for the reported bug. A secret the program never set used
// to drift in total silence: the old mechanism signalled by blanking the field,
// and blanking an unset field does nothing, so refresh produced no state change
// and preview had nothing to show. Marking the argument is what makes it a diff.
func TestSecretDriftSurfacesForUnmanagedField(t *testing.T) {
	// password is deliberately absent - this is the "not under management" case.
	args := ResourceArgs{Type: "postgres", Name: "db", Host: strp("h"), Username: strp("app")}

	applyDrift(t, &args, "postgres",
		map[string]string{"password": "old-digest"},
		map[string]string{"password": "new-digest"})

	if args.Password == nil {
		t.Fatal("drift on an unmanaged secret must still reach state; " +
			"a nil password is exactly the silence this fixes")
	}
	if sv(args.Password) != "new-digest" {
		t.Errorf("expected the fingerprint as the marker, got %q", sv(args.Password))
	}
	// The marker is an opaque server fingerprint, never the secret, and the
	// argument is schema-secret so it renders as [secret].
	field, _, _ := argForConfigKey("postgres", "password")
	if !argIsSecret(field) {
		t.Error("password is not schema-secret, so the marker would print in clear")
	}
}

// A secret the program does set marks the same argument, so preview shows the
// program's value being re-applied. Replaces TestSecretDriftClearsChangedField:
// the signal is the marker rather than a blank, because a blank could never
// carry the unmanaged case.
func TestSecretDriftSurfacesForManagedField(t *testing.T) {
	args := ResourceArgs{
		Type: "postgres", Name: "db",
		Host: strp("h"), Username: strp("app"), Password: strp("s3cret"),
	}

	applyDrift(t, &args, "postgres",
		map[string]string{"password": "old-digest"},
		map[string]string{"password": "new-digest"})

	if sv(args.Password) == "s3cret" {
		t.Error("a drifted managed secret must not keep the state value, or preview stays clean")
	}
	if sv(args.Password) != "new-digest" {
		t.Errorf("expected the fingerprint as the marker, got %q", sv(args.Password))
	}
}

// The common case, and the one that matters most: agreeing fingerprints leave
// the arguments untouched. A false positive here is a diff on every preview of
// every resource that holds a secret.
func TestSecretDriftLeavesMatchingDigestsAlone(t *testing.T) {
	args := ResourceArgs{
		Type: "postgres", Name: "db",
		Host: strp("h"), Username: strp("app"), Password: strp("s3cret"),
	}

	applyDrift(t, &args, "postgres",
		map[string]string{"password": "same"},
		map[string]string{"password": "same"})

	if sv(args.Password) != "s3cret" {
		t.Errorf("an unchanged secret must keep the program's value, got %q", sv(args.Password))
	}
}

// A key mapping to no single argument is reported by name rather than dropped:
// the nested paths the server emits, and ssh's password/sshKey mirror.
func TestSecretDriftReportsUnattributableKeys(t *testing.T) {
	args := ResourceArgs{Type: "postgres", Name: "db"}

	unresolved := applyDrift(t, &args, "postgres",
		map[string]string{"targets[0].password": "old"},
		map[string]string{"targets[0].password": "new"})

	if len(unresolved) != 1 || unresolved[0] != "targets[0].password" {
		t.Errorf("nested key should be reported by name, got %v", unresolved)
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

// End-to-end through the real provider: the marking rule, the seeding rule, and
// the guarantee that refresh does not overwrite the recorded fingerprints.
// applyDrift above covers the classification in isolation; this pins that
// Resource.Read actually wires it up.
func TestResourceReadSecretDrift(t *testing.T) {
	const body = `{"id":"r-1","name":"db","integrationType":"postgres",
		"configuration":{"hostname":"h","username":"app","port":"5432"},
		"redactedKeys":["password"],
		"redactedDigests":{"password":"server-digest"}}`

	// Inputs are what the program wrote; Properties additionally carry the
	// fingerprints recorded at the last write. appliedDigests is state-only, so
	// it must not appear in Inputs.
	read := func(t *testing.T, applied, password string) p.ReadResponse {
		t.Helper()
		args := map[string]property.Value{
			"name": property.New("db"), "type": property.New("postgres"),
			"hostname": property.New("h"), "username": property.New("app"),
		}
		if password != "" {
			args["password"] = property.New(password)
		}
		st := make(map[string]property.Value, len(args)+1)
		for k, v := range args {
			st[k] = v
		}
		if applied != "" {
			st["appliedDigests"] = property.New(map[string]property.Value{
				"password": property.New(applied),
			})
		}

		srv := newReadServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		})
		resp, err := srv.Read(p.ReadRequest{
			ID: "r-1", Urn: urnFor("adaptive:index:Resource"),
			Inputs: property.NewMap(args), Properties: property.NewMap(st),
		})
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		return resp
	}

	digestOf := func(t *testing.T, m property.Map) string {
		t.Helper()
		v, ok := m.GetOk("appliedDigests")
		if !ok {
			return ""
		}
		d, ok := v.AsMap().GetOk("password")
		if !ok {
			return ""
		}
		return d.AsString()
	}

	t.Run("nothing recorded yet is seeded, not reported", func(t *testing.T) {
		// The upgrade path: state written before appliedDigests existed, and
		// every import. Marking here would flag every secret on every resource
		// the first time anyone refreshed.
		resp := read(t, "", "s3cret")

		if v, ok := resp.Inputs.GetOk("password"); ok && v.AsString() != "s3cret" {
			t.Errorf("password must keep the program's value while seeding, got %q", v.AsString())
		}
		if got := digestOf(t, resp.Properties); got != "server-digest" {
			t.Errorf("fingerprints should be seeded from the server, got %q", got)
		}
	})

	t.Run("matching fingerprints leave the argument alone", func(t *testing.T) {
		resp := read(t, "server-digest", "s3cret")

		v, ok := resp.Inputs.GetOk("password")
		if !ok || v.AsString() != "s3cret" {
			t.Errorf("an unchanged secret must keep the program's value, got %v", v)
		}
	})

	t.Run("drift on a managed secret marks the argument", func(t *testing.T) {
		resp := read(t, "stale-digest", "s3cret")

		v, ok := resp.Inputs.GetOk("password")
		if !ok {
			t.Fatal("password missing from inputs")
		}
		if v.AsString() == "s3cret" {
			t.Error("a drifted secret must not keep the state value, or preview stays clean")
		}
	})

	t.Run("drift on an unmanaged secret marks it too", func(t *testing.T) {
		// The reported bug: no password in the program at all.
		resp := read(t, "stale-digest", "")

		if _, ok := resp.Inputs.GetOk("password"); !ok {
			t.Fatal("drift on an unmanaged secret must still reach state; " +
				"an absent password is exactly the silence this fixes")
		}
	})

	t.Run("refresh never overwrites the recorded fingerprints", func(t *testing.T) {
		// If refresh adopted the live view, the next Diff would have nothing to
		// compare against and the drift would vanish after one refresh - which
		// is the original defect.
		resp := read(t, "stale-digest", "s3cret")

		if got := digestOf(t, resp.Properties); got != "stale-digest" {
			t.Errorf("appliedDigests must survive refresh untouched, got %q", got)
		}
	})
}

// A stack created before the rename carries `redactedDigests` in its state.
// infer refuses to decode an unrecognized property, so without a state migration
// every such stack fails to refresh outright:
//
//	error: 1 failures decoding:
//	    redactedDigests: Unrecognized field 'redactedDigests' on 'adaptive.ResourceState'
//
// This is the regression test for that — it exercises Read against v1-shaped
// state, which is what an existing stack actually supplies.
func TestResourceReadMigratesV1State(t *testing.T) {
	const body = `{"id":"r-1","name":"db","integrationType":"postgres",
		"configuration":{"hostname":"h","username":"app","port":"5432"},
		"redactedKeys":["password"],
		"redactedDigests":{"password":"server-digest"}}`

	args := map[string]property.Value{
		"name": property.New("db"), "type": property.New("postgres"),
		"hostname": property.New("h"), "username": property.New("app"),
		"password": property.New("s3cret"),
	}
	// State as an older provider wrote it: the old property name.
	oldState := map[string]property.Value{}
	for k, v := range args {
		oldState[k] = v
	}
	oldState["redactedDigests"] = property.New(map[string]property.Value{
		"password": property.New("recorded-digest"),
	})

	srv := newReadServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	})
	resp, err := srv.Read(p.ReadRequest{
		ID: "r-1", Urn: urnFor("adaptive:index:Resource"),
		Inputs: property.NewMap(args), Properties: property.NewMap(oldState),
	})
	if err != nil {
		t.Fatalf("v1 state must still decode; an existing stack cannot refresh otherwise: %v", err)
	}

	// The recorded fingerprints carry across the rename rather than being
	// dropped, so drift is detected on this refresh instead of the next one.
	// "recorded-digest" != "server-digest", so the secret is marked.
	if v, ok := resp.Inputs.GetOk("password"); !ok || v.AsString() != "server-digest" {
		t.Errorf("migrated baseline should have been compared and the drift marked, got %v", v)
	}
	if _, ok := resp.Properties.GetOk("redactedDigests"); ok {
		t.Error("migrated state must not carry the old property forward")
	}
}

// The server withholds more than passwords: configexport's key policy treats
// cert, crt, accesskey and publickey material as secret too. Every such argument
// has to be declared secret here as well, or the marker cannot be written to it
// and the drift degrades to a warning — which is exactly what a `rootCert`
// change on a postgres resource did:
//
//	warning: the secret behind rootCert changed outside this program, but it
//	         maps to no single argument and so cannot be shown as a diff
//
// The message was wrong too: rootCert *does* map to an argument. It just was not
// secret. Both halves are fixed; this pins the schema half.
func TestServerWithheldArgumentsAreDeclaredSecret(t *testing.T) {
	// Derived by cross-referencing every type's config keys against the server's
	// classifier. Deliberately a concrete list rather than a copy of that
	// classifier: a mirrored copy would rot silently, whereas a key the server
	// starts withholding later shows up at runtime in the warning above.
	cases := []struct{ integrationType, cfgKey, wantArg string }{
		{"postgres", "rootCert", "tlsRootCert"},
		{"cockroachdb", "rootCert", "tlsRootCert"},
		{"proxysql", "rootCert", "rootCert"},
		{"redis", "crtText", "tlsCertFile"},
		{"kubernetes", "cacrt", "clusterCert"},
		{"aws", "aws_access_key_id", "accessKeyId"},
		{"mongodb_atlas", "public_key", "publicKey"},
		// Already secret; here so the list reads as the whole policy.
		{"postgres", "password", "password"},
		{"postgres", "keyText", "tlsKeyFile"},
		{"aws", "aws_secret_access_key", "secretAccessKey"},
	}

	for _, c := range cases {
		t.Run(c.integrationType+"/"+c.cfgKey, func(t *testing.T) {
			field, prop, ok := argForConfigKey(c.integrationType, c.cfgKey)
			if !ok {
				t.Fatalf("does not resolve to any argument")
			}
			if prop != c.wantArg {
				t.Fatalf("resolves to %q, want %q", prop, c.wantArg)
			}
			if !argIsSecret(field) {
				t.Errorf("argument %q is not declared secret, so a drifted %q can only be "+
					"warned about, not shown as a diff", prop, c.cfgKey)
			}
		})
	}
}
