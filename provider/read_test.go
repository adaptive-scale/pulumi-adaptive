package adaptive

import (
	"reflect"
	"strings"
	"testing"
)

// pulumiPropNames lists the pulumi property names declared by ResourceArgs.
func pulumiPropNames() []string {
	var out []string
	for prop := range pulumiPropSecrecy() {
		out = append(out, prop)
	}
	return out
}

// pulumiPropSecrecy maps each ResourceArgs pulumi property to whether it carries
// the `provider:"secret"` tag.
func pulumiPropSecrecy() map[string]bool {
	t := reflect.TypeOf(ResourceArgs{})
	out := make(map[string]bool, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("pulumi")
		if tag == "" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		out[name] = f.Tag.Get("provider") == "secret"
	}
	return out
}

// The overloaded keys are the ones worth pinning: writing one into the wrong
// property would not fail loudly, it would just show drift forever.
func TestResourceConfigPropOverloadedKeys(t *testing.T) {
	cases := []struct {
		integrationType string
		key             string
		wantProp        string
		wantOK          bool
	}{
		{"postgres", "hostname", "host", true},
		{"mysql", "hostname", "host", true},
		{"paloalto_ngfw", "hostname", "hostname", true},
		{"syslog", "hostname", "hostname", true},
		{"snowflake", "databaseAccount", "hostname", true},
		{"snowflake", "databaseUsername", "username", true},
		{"splunk", "url", "url", true},
		{"coralogix", "url", "uri", true},
		{"elasticsearch", "url", "uri", true},
		{"paloalto_ngfw", "login_url", "loginUrl", true},
		{"cisco_ngfw", "login_url", "uri", true},
		{"azurecosmosnosql", "endpoint", "uri", true},
		{"aws", "aws_region_name", "regionName", true},
		{"awssecretsmanager", "aws_region_name", "awsRegionName", true},
		// Keys the provider never wrote have no inverse and must be ignored.
		{"postgres", "version", "", false},
		{"postgres", "totally_unknown", "", false},
	}

	for _, tc := range cases {
		gotProp, gotOK := resourceConfigProp(tc.integrationType, tc.key)
		if gotOK != tc.wantOK || gotProp != tc.wantProp {
			t.Errorf("resourceConfigProp(%q, %q) = (%q, %v), want (%q, %v)",
				tc.integrationType, tc.key, gotProp, gotOK, tc.wantProp, tc.wantOK)
		}
	}
}

// Every key in the inverse tables must name a property ResourceArgs actually
// has, or a refresh would silently drop it.
func TestResourceConfigPropsExistOnResourceArgs(t *testing.T) {
	known := map[string]bool{}
	for _, tag := range pulumiPropNames() {
		known[tag] = true
	}

	for key, prop := range resourceConfigKeyToProp {
		if !known[prop] {
			t.Errorf("key %q maps to %q, which is not a property of ResourceArgs", key, prop)
		}
	}
	for integrationType, overrides := range resourceConfigKeyPropOverrides {
		for key, prop := range overrides {
			if !known[prop] {
				t.Errorf("%s override %q maps to %q, which is not a property of ResourceArgs",
					integrationType, key, prop)
			}
		}
	}
}

// setPulumiProp has to convert the shapes JSON decoding produces into the
// pointer and slice types ResourceArgs declares.
func TestSetPulumiPropConvertsDecodedValues(t *testing.T) {
	var args ResourceArgs

	// A port is a string property but decodes as a JSON number.
	if err := setPulumiProp(&args, "port", float64(5432)); err != nil {
		t.Fatalf("port: %v", err)
	}
	if args.Port == nil || *args.Port != "5432" {
		t.Errorf("port = %v, want \"5432\" without a decimal suffix", args.Port)
	}

	if err := setPulumiProp(&args, "host", "db.internal"); err != nil {
		t.Fatalf("host: %v", err)
	}
	if args.Host == nil || *args.Host != "db.internal" {
		t.Errorf("host = %v", args.Host)
	}

	if err := setPulumiProp(&args, "useRdsIam", true); err != nil {
		t.Fatalf("useRdsIam: %v", err)
	}
	if args.UseRdsIam == nil || !*args.UseRdsIam {
		t.Errorf("useRdsIam = %v", args.UseRdsIam)
	}

	// serverlist stores hosts newline-joined; a list property must come back as
	// a list either way.
	if err := setPulumiProp(&args, "hosts", "a.example.com\nb.example.com\n"); err != nil {
		t.Fatalf("hosts from string: %v", err)
	}
	if len(args.Hosts) != 2 || args.Hosts[1] != "b.example.com" {
		t.Errorf("hosts = %#v", args.Hosts)
	}

	if err := setPulumiProp(&args, "hosts", []interface{}{"c.example.com"}); err != nil {
		t.Fatalf("hosts from sequence: %v", err)
	}
	if len(args.Hosts) != 1 || args.Hosts[0] != "c.example.com" {
		t.Errorf("hosts = %#v", args.Hosts)
	}

	// Type is a plain string, not a pointer.
	if err := setPulumiProp(&args, "type", "postgres"); err != nil {
		t.Fatalf("type: %v", err)
	}
	if args.Type != "postgres" {
		t.Errorf("type = %q", args.Type)
	}
}

// An unknown property is a table bug, not a reason to fail an entire refresh.
func TestSetPulumiPropIgnoresUnknownProperty(t *testing.T) {
	var args ResourceArgs
	if err := setPulumiProp(&args, "noSuchProperty", "x"); err != nil {
		t.Fatalf("expected unknown properties to be ignored, got %v", err)
	}
}

// optString keeps an absent value absent rather than turning it into "".
func TestOptStringEmptyIsNil(t *testing.T) {
	if optString("") != nil {
		t.Error("empty string should map to nil so the property stays unset")
	}
	if got := optString("x"); got == nil || *got != "x" {
		t.Errorf("optString(\"x\") = %v", got)
	}
}

// Credentials must be marked secret so they are encrypted in state rather than
// printed in previews. Before this the provider marked nothing secret at all.
func TestCredentialPropertiesAreSecret(t *testing.T) {
	secretProps := map[string]bool{}
	for prop, isSecret := range pulumiPropSecrecy() {
		secretProps[prop] = isSecret
	}

	for _, prop := range []string{
		"password", "clientSecret", "apiClientSecret", "secretAccessKey", "accessKeyId",
		"clusterToken", "clusterCert", "key", "keyFile", "privateKey", "apiToken",
		"ddApiKey", "sharedSecret", "databasePassword", "awsSecretAccessKey",
		"tlsKeyFile", "tlsCertFile", "webhookUrl",
	} {
		if !secretProps[prop] {
			t.Errorf("%s is a credential but is not marked secret", prop)
		}
	}

	// Topology and identity fields must NOT be secret, or previews become
	// unreadable for no security gain.
	for _, prop := range []string{"host", "hostname", "port", "username", "databaseName", "region"} {
		if secretProps[prop] {
			t.Errorf("%s is not a credential but is marked secret", prop)
		}
	}
}
