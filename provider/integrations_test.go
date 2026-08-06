package adaptive

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func strp(s string) *string { return &s }
func boolp(b bool) *bool    { return &b }

// toMap marshals a config struct to YAML and back into a map, so tests can
// assert on the exact YAML key names the Adaptive API receives.
func toMap(t *testing.T, cfg any) map[string]any {
	t.Helper()
	b, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

func TestBuildIntegrationConfig_Postgres(t *testing.T) {
	cfg, eff, err := buildIntegrationConfig(ResourceArgs{
		Type: "postgres", Name: "db",
		Host: strp("h"), Port: strp("5432"), Username: strp("admin"),
		Password: strp("secret"), DatabaseName: strp("app"), SSLMode: strp("require"),
		TLSCertFile: strp("CERT"), TLSKeyFile: strp("KEY"), TLSRootCert: strp("ROOT"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if eff != "postgres" {
		t.Errorf("effective type = %q, want postgres", eff)
	}
	m := toMap(t, cfg)
	for k, want := range map[string]any{
		"name": "db", "hostname": "h", "port": "5432", "sslMode": "require",
		"databaseName": "app", "crtText": "CERT", "keyText": "KEY", "rootCert": "ROOT",
	} {
		if m[k] != want {
			t.Errorf("postgres yaml[%q] = %v, want %v", k, m[k], want)
		}
	}
	// A password resource must marshal exactly as it did before the RDS IAM
	// fields existed.
	for _, k := range []string{"useRdsIam", "useIrsa", "awsRegion", "awsRoleArn", "awsServiceAccount"} {
		if _, ok := m[k]; ok {
			t.Errorf("password-mode postgres should not emit %q", k)
		}
	}
}

func TestBuildIntegrationConfig_PostgresRDSIAM(t *testing.T) {
	base := func() ResourceArgs {
		return ResourceArgs{
			Type: "postgres", Name: "db",
			Host: strp("mydb.abc123.us-east-1.rds.amazonaws.com"), Port: strp("5432"),
			Username: strp("iam_user"), SSLMode: strp("require"),
			UseRDSIAM: boolp(true),
		}
	}

	tests := []struct {
		name   string
		mutate func(*ResourceArgs)
		want   map[string]any
	}{
		{
			name:   "service account implies irsa",
			mutate: func(a *ResourceArgs) { a.AWSServiceAccount = strp("adaptive-rds-access") },
			want: map[string]any{
				"useRdsIam": true, "useIrsa": true,
				"awsServiceAccount": "adaptive-rds-access",
			},
		},
		{
			name: "static keys imply no irsa",
			mutate: func(a *ResourceArgs) {
				a.AWSAccessKeyID = strp("AKID")
				a.AWSSecretAccessKey = strp("secret")
			},
			want: map[string]any{
				"useRdsIam": true, "useIrsa": false,
				"awsAccessKeyId": "AKID", "awsSecretAccessKey": "secret",
			},
		},
		{
			name: "explicit irsa with a role arn",
			mutate: func(a *ResourceArgs) {
				a.UseIRSA = boolp(true)
				a.AWSRoleARN = strp("arn:aws:iam::123456789012:role/AdaptiveRDSAccess")
				a.AWSRegion = strp("us-east-1")
			},
			want: map[string]any{
				"useRdsIam": true, "useIrsa": true,
				"awsRoleArn": "arn:aws:iam::123456789012:role/AdaptiveRDSAccess",
				"awsRegion":  "us-east-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := base()
			tt.mutate(&args)
			cfg, _, err := buildIntegrationConfig(args)
			if err != nil {
				t.Fatal(err)
			}
			m := toMap(t, cfg)
			for k, want := range tt.want {
				if m[k] != want {
					t.Errorf("yaml[%q] = %v, want %v", k, m[k], want)
				}
			}
		})
	}
}

func TestBuildIntegrationConfig_RDSIAMRejectedOnOtherTypes(t *testing.T) {
	_, _, err := buildIntegrationConfig(ResourceArgs{
		Type: "mysql", Name: "db", Host: strp("h"), Username: strp("root"),
		UseRDSIAM: boolp(true),
	})
	if err == nil {
		t.Fatal("expected an error when the RDS IAM fields are set on a non-postgres resource")
	}
}

func TestBuildIntegrationConfig_SSH(t *testing.T) {
	// No key -> password auth.
	noKey, _, _ := buildIntegrationConfig(ResourceArgs{Type: "ssh", Name: "s", Username: strp("u"), Host: strp("h")})
	if m := toMap(t, noKey); m["usePassword"] != true {
		t.Errorf("ssh without key: usePassword = %v, want true", m["usePassword"])
	}
	// Key present -> key auth, and password mirrors the key (matches TF behavior).
	withKey, _, _ := buildIntegrationConfig(ResourceArgs{Type: "ssh", Name: "s", Key: strp("PRIVKEY")})
	m := toMap(t, withKey)
	if m["usePassword"] != false {
		t.Errorf("ssh with key: usePassword = %v, want false", m["usePassword"])
	}
	if m["sshKey"] != "PRIVKEY" || m["password"] != "PRIVKEY" {
		t.Errorf("ssh with key: sshKey=%v password=%v, want both PRIVKEY", m["sshKey"], m["password"])
	}
}

func TestBuildIntegrationConfig_ServicesEffectiveType(t *testing.T) {
	cfg, eff, err := buildIntegrationConfig(ResourceArgs{Type: "services", Name: "svc", URLs: strp("a,b,c")})
	if err != nil {
		t.Fatal(err)
	}
	if eff != "servicelist" {
		t.Errorf("services effective type = %q, want servicelist", eff)
	}
	m := toMap(t, cfg)
	if m["version"] != "1" || m["urls"] != "a,b,c" {
		t.Errorf("services yaml = %v", m)
	}
}

func TestBuildIntegrationConfig_MySQLForcesSSLRequire(t *testing.T) {
	cfg, _, _ := buildIntegrationConfig(ResourceArgs{Type: "mysql", Name: "m"})
	if m := toMap(t, cfg); m["sslMode"] != "require" {
		t.Errorf("mysql sslMode = %v, want require", m["sslMode"])
	}
}

func TestBuildIntegrationConfig_GCPTrimsAndVersions(t *testing.T) {
	cfg, _, _ := buildIntegrationConfig(ResourceArgs{Type: "gcp", Name: "g", ProjectID: strp("proj"), KeyFile: strp("  KEYDATA\n")})
	m := toMap(t, cfg)
	if m["version"] != "1" {
		t.Errorf("gcp version = %v, want 1", m["version"])
	}
	if m["key_file"] != "KEYDATA" { // trimmed
		t.Errorf("gcp key_file = %q, want trimmed KEYDATA", m["key_file"])
	}
}

func TestBuildIntegrationConfig_KubernetesTrims(t *testing.T) {
	cfg, _, _ := buildIntegrationConfig(ResourceArgs{
		Type: "kubernetes", Name: "k",
		ApiServer: strp("https://api"), ClusterToken: strp(" tok\n"), ClusterCert: strp(" cert "),
	})
	m := toMap(t, cfg)
	if m["apiserver"] != "https://api" || m["token"] != "tok" || m["cacrt"] != "cert" {
		t.Errorf("kubernetes yaml = %v", m)
	}
}

func TestBuildIntegrationConfig_AWSSecretsManagerShared(t *testing.T) {
	// postgres/sqlserver/snowflake AWS secrets manager share the arn/region/secret_id shape.
	cfg, eff, _ := buildIntegrationConfig(ResourceArgs{
		Type: "postgres_aws_secrets_manager", Name: "p",
		Arn: strp("arn:x"), Region: strp("us-east-1"), SecretID: strp("sec"),
	})
	if eff != "postgres_aws_secrets_manager" {
		t.Errorf("effective type = %q", eff)
	}
	m := toMap(t, cfg)
	if m["arn"] != "arn:x" || m["region"] != "us-east-1" || m["secret_id"] != "sec" {
		t.Errorf("secrets manager yaml = %v", m)
	}
}

func TestBuildIntegrationConfig_UseTenantBool(t *testing.T) {
	cfg, _, _ := buildIntegrationConfig(ResourceArgs{Type: "azureactivedirectory", Name: "ad", UseTenant: boolp(true)})
	if m := toMap(t, cfg); m["useTenant"] != true {
		t.Errorf("azuread useTenant = %v, want true", m["useTenant"])
	}
}

func TestBuildIntegrationConfig_InvalidType(t *testing.T) {
	if _, _, err := buildIntegrationConfig(ResourceArgs{Type: "not-a-real-type", Name: "x"}); err == nil {
		t.Fatal("expected error for invalid integration type, got nil")
	}
}

func TestAllValidTypesBuild(t *testing.T) {
	// Every declared valid type must produce a config without panicking or erroring.
	for typ := range validIntegrationTypes {
		_, _, err := buildIntegrationConfig(ResourceArgs{Type: typ, Name: "n"})
		if err != nil {
			t.Errorf("type %q failed to build: %v", typ, err)
		}
	}
}
