package adaptive

import (
	"context"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
)

func TestBuildIntegrationConfig_GoogleDirectoryCredentials(t *testing.T) {
	cfg, wire, err := buildIntegrationConfig(fillArgs(t, "google"))
	if err != nil {
		t.Fatal(err)
	}
	if wire != "google" {
		t.Fatalf("wire type = %q, want google", wire)
	}
	m := toMap(t, cfg)
	for key, want := range map[string]string{
		"domain":         "Domain-val",
		"clientID":       "ClientID-val",
		"clientSecret":   "ClientSecret-val",
		"serviceAccount": "ServiceAccount-val",
		"adminEmail":     "AdminEmail-val",
	} {
		if m[key] != want {
			t.Errorf("google yaml[%q] = %v, want %q", key, m[key], want)
		}
	}
}

func TestApplyIntegrationConfig_GoogleDirectoryCredentials(t *testing.T) {
	args := ResourceArgs{Type: "google", Name: "sso"}
	applyIntegrationConfig(&args, "google", map[string]any{
		"serviceAccount": `{"type":"service_account","private_key":"test-key"}`,
		"adminEmail":     "admin@example.com",
	})
	if sv(args.ServiceAccount) != `{"type":"service_account","private_key":"test-key"}` {
		t.Fatal("serviceAccount was not recovered during import")
	}
	if sv(args.AdminEmail) != "admin@example.com" {
		t.Fatal("adminEmail was not recovered during import")
	}

	// A redacted key is absent, not empty: refresh must preserve its prior value.
	applyIntegrationConfig(&args, "google", map[string]any{"adminEmail": "new-admin@example.com"})
	if sv(args.ServiceAccount) != `{"type":"service_account","private_key":"test-key"}` {
		t.Fatal("refresh discarded the redacted serviceAccount")
	}
	if sv(args.AdminEmail) != "new-admin@example.com" {
		t.Fatal("refresh missed adminEmail drift")
	}

	applyIntegrationConfig(&args, "google", map[string]any{"serviceAccount": "", "adminEmail": ""})
	if args.ServiceAccount != nil || args.AdminEmail != nil {
		t.Fatal("explicit empty values must clear the prior directory settings")
	}
}

func TestGoogleDirectoryCredentialSchema(t *testing.T) {
	prov, err := Provider()
	if err != nil {
		t.Fatal(err)
	}
	spec, err := p.GetSchema(context.Background(), "adaptive", Version, prov)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := spec.Resources["adaptive:index:Resource"]
	if !ok {
		t.Fatal("Resource is missing from the provider schema")
	}
	for _, name := range []string{"serviceAccount", "adminEmail"} {
		prop, ok := r.InputProperties[name]
		if !ok || prop.Type != "string" {
			t.Fatalf("%s must be a string input", name)
		}
		if prop.Secret != (name == "serviceAccount") {
			t.Errorf("%s secret = %v, want %v", name, prop.Secret, name == "serviceAccount")
		}
		if r.Properties[name].Secret != prop.Secret {
			t.Errorf("%s output secrecy differs from input", name)
		}
		for _, required := range r.RequiredInputs {
			if required == name {
				t.Errorf("%s must remain optional for OAuth-only configurations", name)
			}
		}
	}
}
