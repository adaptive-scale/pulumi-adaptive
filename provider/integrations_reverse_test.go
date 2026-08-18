package adaptive

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

// fillArgs sets every optional field of a ResourceArgs to a distinctive
// non-zero value: *string -> "<FieldName>-val", *bool -> true,
// []string -> {"h1", "h2"}. Type and Name are set by the caller.
func fillArgs(t *testing.T, typ string) ResourceArgs {
	t.Helper()
	a := ResourceArgs{Type: typ, Name: "rt-" + typ}
	v := reflect.ValueOf(&a).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)
		if f.Name == "Type" || f.Name == "Name" || f.Name == "Tags" || f.Name == "DefaultCluster" {
			continue
		}
		switch f.Type.Kind() {
		case reflect.Ptr:
			switch f.Type.Elem().Kind() {
			case reflect.String:
				s := f.Name + "-val"
				v.Field(i).Set(reflect.ValueOf(&s))
			case reflect.Bool:
				b := true
				v.Field(i).Set(reflect.ValueOf(&b))
			default:
				t.Fatalf("fillArgs: unhandled pointer kind %s for field %s", f.Type.Elem().Kind(), f.Name)
			}
		case reflect.Slice:
			if f.Type.Elem().Kind() == reflect.String {
				v.Field(i).Set(reflect.ValueOf([]string{"h1", "h2"}))
			} else {
				t.Fatalf("fillArgs: unhandled slice kind for field %s", f.Name)
			}
		default:
			t.Fatalf("fillArgs: unhandled kind %s for field %s", f.Type.Kind(), f.Name)
		}
	}
	return a
}

// configToMap marshals a config struct to YAML and back into the string-keyed
// map shape the server's read endpoint returns.
func configToMap(t *testing.T, cfg any) map[string]any {
	t.Helper()
	b, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[any]any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		ks, ok := k.(string)
		if !ok {
			t.Fatalf("non-string yaml key %v", k)
		}
		out[ks] = v
	}
	return out
}

// TestIntegrationConfigRoundTrip is the completeness guard: for every
// supported integration type, build a config from fully-populated args, feed
// it through applyIntegrationConfig onto fresh args, rebuild, and require the
// two configs to be identical. A type added to buildIntegrationConfig but
// forgotten in applyIntegrationConfig fails here.
func TestIntegrationConfigRoundTrip(t *testing.T) {
	for typ := range validIntegrationTypes {
		t.Run(typ, func(t *testing.T) {
			args := fillArgs(t, typ)
			cfg1, wire, err := buildIntegrationConfig(args)
			if err != nil {
				t.Fatalf("forward build: %v", err)
			}
			m := configToMap(t, cfg1)
			// The server strips the name key from read responses.
			delete(m, "name")

			back := ResourceArgs{Type: typ, Name: args.Name}
			applyIntegrationConfig(&back, wire, m)

			cfg2, _, err := buildIntegrationConfig(back)
			if err != nil {
				t.Fatalf("rebuild: %v", err)
			}
			m1 := configToMap(t, cfg1)
			m2 := configToMap(t, cfg2)
			if !reflect.DeepEqual(m1, m2) {
				t.Errorf("round-trip mismatch:\n forward: %v\n rebuilt: %v", m1, m2)
			}

			// Behavioral completeness: the reverse mapping must have populated
			// something beyond the envelope (securetunnels has no config
			// fields beyond name, so it is exempt).
			if typ != "securetunnels" && reflect.DeepEqual(back, (ResourceArgs{Type: typ, Name: args.Name})) {
				t.Errorf("applyIntegrationConfig set no fields for %q", typ)
			}
		})
	}
}

// TestApplyIntegrationConfigRedaction pins the refresh semantics around
// server-side secret stripping: absent keys never clear prior values, present
// keys update them, and empty values clear only previously-set fields.
func TestApplyIntegrationConfigRedaction(t *testing.T) {
	prior := ResourceArgs{
		Type: "postgres", Name: "db",
		Host:       strp("h"),
		Port:       strp("5432"),
		Username:   strp("app"),
		Password:   strp("s3cret"),
		TLSKeyFile: strp("KEY"),
		SSLMode:    strp("require"),
	}
	// Server response: secrets (password, keyText) stripped, hostname drifted.
	applyIntegrationConfig(&prior, "postgres", map[string]any{
		"hostname":     "newhost",
		"port":         "5432",
		"username":     "app",
		"databaseName": "",
		"sslMode":      "require",
	})
	if sv(prior.Password) != "s3cret" || sv(prior.TLSKeyFile) != "KEY" {
		t.Errorf("redacted keys clobbered secrets: password=%q key=%q", sv(prior.Password), sv(prior.TLSKeyFile))
	}
	if sv(prior.Host) != "newhost" {
		t.Errorf("hostname drift missed: %q", sv(prior.Host))
	}

	// Empty values: clear a set field, never populate an unset one.
	a := ResourceArgs{Type: "postgres", SSLMode: strp("require")}
	applyIntegrationConfig(&a, "postgres", map[string]any{"sslMode": "", "port": ""})
	if a.SSLMode != nil {
		t.Errorf("empty server value should clear a set field, got %q", sv(a.SSLMode))
	}
	if a.Port != nil {
		t.Errorf("empty server value populated an unset field: %q", sv(a.Port))
	}
}

func TestRefreshIntegrationConfigClearsMissingKubernetesFields(t *testing.T) {
	prior := ResourceArgs{
		Type:         "kubernetes",
		Name:         "eks-prod",
		ApiServer:    strp("https://old.example.com"),
		Namespace:    strp("removed-remotely"),
		Tolerations:  strp("old-toleration"),
		Annotations:  strp("old-annotation"),
		NodeSelector: strp("old-selector"),
		NodeAffinity: strp("old-affinity"),
		ClusterToken: strp("write-only-token"),
		ClusterCert:  strp("write-only-cert"),
	}
	live := map[string]any{"apiserver": "https://new.example.com"}

	cfg, err := refreshIntegrationConfig(
		prior,
		live,
		[]string{"token", "cacrt"},
		map[string]string{"token": "digest-1", "cacrt": "digest-2"},
	)
	if err != nil {
		t.Fatal(err)
	}
	applyIntegrationConfig(&prior, "kubernetes", cfg)

	if sv(prior.ApiServer) != "https://new.example.com" {
		t.Errorf("apiServer drift was not refreshed: %q", sv(prior.ApiServer))
	}
	if prior.Namespace != nil || prior.Tolerations != nil || prior.Annotations != nil ||
		prior.NodeSelector != nil || prior.NodeAffinity != nil {
		t.Errorf("removed fields survived refresh: %+v", prior)
	}
	if sv(prior.ClusterToken) != "write-only-token" || sv(prior.ClusterCert) != "write-only-cert" {
		t.Errorf("redacted credentials were not preserved: token=%q cert=%q",
			sv(prior.ClusterToken), sv(prior.ClusterCert))
	}
}

func TestRefreshIntegrationConfigClearsMissingBooleanField(t *testing.T) {
	prior := ResourceArgs{
		Type:     "fortinet_ngfw",
		Name:     "firewall",
		UseProxy: boolp(true),
	}
	cfg, err := refreshIntegrationConfig(prior, map[string]any{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	applyIntegrationConfig(&prior, "fortinet_ngfw", cfg)
	if prior.UseProxy == nil || *prior.UseProxy {
		t.Errorf("removed useProxy should refresh to false, got %v", prior.UseProxy)
	}
}

// TestWireTypeMapping pins the two wire-name rewrites and their inverses.
func TestWireTypeMapping(t *testing.T) {
	cases := map[string]string{"services": "servicelist", "zerotier": "zero_tier", "postgres": "postgres"}
	for prog, wire := range cases {
		if got := wireType(prog); got != wire {
			t.Errorf("wireType(%q) = %q, want %q", prog, got, wire)
		}
		if got := providerType(wire); got != prog {
			t.Errorf("providerType(%q) = %q, want %q", wire, got, prog)
		}
	}
	// Legacy stored spelling still resolves to the program type.
	if got := providerType("zerotier"); got != "zerotier" {
		t.Errorf("providerType(zerotier legacy) = %q", got)
	}
}
