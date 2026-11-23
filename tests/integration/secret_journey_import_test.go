//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optimport"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// These journeys use only owned disposable stacks and synthetic integration
// configuration. They do not connect to PostgreSQL or Snowflake databases.

type importJourneyResult struct {
	Name         string   `json:"name"`
	Outcome      string   `json:"outcome"`
	ID           string   `json:"id,omitempty"`
	Stack        string   `json:"stack,omitempty"`
	Checks       []string `json:"checks"`
	Cleanup404   bool     `json:"cleanup404"`
	StackRemoved bool     `json:"stackRemoved"`
}

type importJourneySpec struct {
	kind, name, cluster string
	password            *string
	tags                []string
	enabled             bool
}

func (s *importJourneySpec) program(ctx *pulumi.Context) error {
	if !s.enabled {
		return nil
	}
	args := &adaptive.ResourceArgs{Name: pulumi.String(s.name), Type: pulumi.String(s.kind), Username: pulumi.String("journey_admin"), DefaultCluster: pulumi.String(s.cluster)}
	if len(s.tags) > 0 {
		args.Tags = pulumi.ToStringArray(s.tags)
	}
	if s.kind == "postgres" {
		args.Host = pulumi.String("journey.invalid")
		args.Port = pulumi.String("5432")
		args.SslMode = pulumi.String("require")
	} else {
		args.Hostname = pulumi.String("journey.invalid")
		args.DatabaseName = pulumi.String("JOURNEY_DB")
		args.Warehouse = pulumi.String("JOURNEY_WH")
		args.Schema = pulumi.String("PUBLIC")
		args.Role = pulumi.String("JOURNEY_ROLE")
	}
	if s.password != nil {
		args.Password = pulumi.String(*s.password)
	}
	_, err := adaptive.NewResource(ctx, "db", args)
	return err
}

func (s *importJourneySpec) wire(password string) map[string]any {
	if s.kind == "postgres" {
		return map[string]any{"name": s.name, "hostname": "journey.invalid", "port": "5432", "username": "journey_admin", "password": password, "sslMode": "require"}
	}
	return map[string]any{"name": s.name, "databaseAccount": "journey.invalid", "databaseUsername": "journey_admin", "databasePassword": password, "databaseName": "JOURNEY_DB", "warehouse": "JOURNEY_WH", "schema": "PUBLIC", "role": "JOURNEY_ROLE", "clientcert": ""}
}
func (s *importJourneySpec) key() string {
	if s.kind == "snowflake" {
		return "databasePassword"
	}
	return "password"
}

// All raw writes replace the complete stored integration config, never a
// secret-only fragment. Readback checks below are separate from HTTP success.
func importJourneyWrite(t *testing.T, cfg harness.Config, s *importJourneySpec, id, password string) string {
	t.Helper()
	config, err := yaml.Marshal(s.wire(password))
	require.NoError(t, err)
	path := "/resource/create"
	if id != "" {
		path = "/resource/update/" + id
	}
	resp, body := rawTerraformAPI(t, cfg, "POST", path, map[string]any{"name": s.name, "integrationType": s.kind, "config": string(config), "userTags": s.tags, "defaultCluster": s.cluster})
	require.Equal(t, http.StatusOK, resp.StatusCode, "full-config mutation status")
	if id != "" {
		return id
	}
	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(body, &created))
	require.NotEmpty(t, created.ID)
	return created.ID
}

func importJourneyCheckpoint(t *testing.T, stack *auto.Stack, edit func(map[string]any)) {
	t.Helper()
	dep, err := stack.Export(context.Background())
	require.NoError(t, err)
	var state map[string]any
	require.NoError(t, json.Unmarshal(dep.Deployment, &state))
	found := false
	var expectedInputs, expectedOutputs any
	for _, raw := range state["resources"].([]any) {
		r := raw.(map[string]any)
		if r["type"] == resourceTypeToken {
			edit(r)
			expectedInputs, expectedOutputs = r["inputs"], r["outputs"]
			found = true
		}
	}
	require.True(t, found, "only our disposable resource checkpoint may be edited")
	dep.Deployment, err = json.Marshal(state)
	require.NoError(t, err)
	require.NoError(t, stack.Import(context.Background(), dep))
	// Read back the edited checkpoint without exposing encrypted secret envelopes.
	after, err := stack.Export(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, after.Deployment)
	var readback map[string]any
	require.NoError(t, json.Unmarshal(after.Deployment, &readback))
	verified := false
	for _, raw := range readback["resources"].([]any) {
		resource := raw.(map[string]any)
		if resource["type"] == resourceTypeToken {
			want, err := json.Marshal([]any{expectedInputs, expectedOutputs})
			require.NoError(t, err)
			got, err := json.Marshal([]any{resource["inputs"], resource["outputs"]})
			require.NoError(t, err)
			require.True(t, string(want) == string(got), "edited owned checkpoint must read back exactly (secret values withheld)")
			verified = true
		}
	}
	require.True(t, verified)
}

func TestSecretJourneyImport(t *testing.T) {
	cfg := harness.RequireProviderConfig(t)
	require.Equal(t, "http://localhost:8080", cfg.URL, "these live journeys are authorized only on the local backend")
	pluginDir, err := filepath.Abs("../../provider/bin")
	require.NoError(t, err)
	t.Setenv("PATH", pluginDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	var results []*importJourneyResult
	evidencePath := os.Getenv("ADAPTIVE_JOURNEY_EVIDENCE")
	if evidencePath == "" {
		evidencePath = filepath.Join(t.TempDir(), "secret-journey-import-results.json")
	}
	t.Cleanup(func() {
		data, err := json.MarshalIndent(results, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(evidencePath, data, 0600))
	})
	for _, kind := range []string{"postgres", "snowflake"} {
		for _, scenario := range []string{"import-zero", "import-held", "legacy-managed-no-baseline", "legacy-empty-no-baseline", "legacy-empty-buggy-seeded", "legacy-v1-migration", "legacy-empty-v1-migration"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				r := &importJourneyResult{Name: t.Name(), Outcome: "STARTED", Checks: []string{}}
				results = append(results, r)
				t.Cleanup(func() {
					if t.Failed() {
						r.Outcome = "FAIL"
					} else {
						r.Outcome = "PASS"
					}
				})
				original := "synthetic-import-original"
				empty := ""
				spec := &importJourneySpec{kind: kind, name: uniqueName("pr24-import-" + kind), enabled: false, tags: []string{}}
				r.Stack = stackName("pr24-import")
				var stack *auto.Stack
				backendDir := t.TempDir() // Registered before owned-stack cleanup (LIFO).
				// Register exact-identity fallback cleanup immediately after receiving ID,
				// before stack creation/import; never enumerate or delete unowned objects.
				initial := original
				if scenario == "import-zero" || strings.Contains(scenario, "legacy-empty") {
					initial = empty
				}
				r.ID = importJourneyWrite(t, cfg, spec, "", initial)
				t.Cleanup(func() {
					if stack != nil {
						_, err := stack.Destroy(context.Background())
						if err != nil {
							t.Errorf("owned stack destroy failed (details withheld)")
						}
					}
					resp, _ := rawTerraformAPI(t, cfg, "GET", "/resource/read/"+r.ID, nil)
					if resp.StatusCode != http.StatusNotFound {
						// Raw creation ownership was established before any import attempt.
						rawTerraformAPI(t, cfg, "POST", "/resource/delete/"+r.ID, nil)
						resp, _ = rawTerraformAPI(t, cfg, "GET", "/resource/read/"+r.ID, nil)
					}
					r.Cleanup404 = resp.StatusCode == http.StatusNotFound
					require.True(t, r.Cleanup404, "exact owned identity must return 404")
					if stack != nil {
						require.NoError(t, stack.Workspace().RemoveStack(context.Background(), r.Stack))
						stacks, err := stack.Workspace().ListStacks(context.Background())
						require.NoError(t, err)
						r.StackRemoved = true
						for _, summary := range stacks {
							if summary.Name == r.Stack {
								r.StackRemoved = false
							}
						}
						require.True(t, r.StackRemoved, "owned disposable stack must be removed")
					}
				})
				remote := terraformRead(t, cfg, "resource", r.ID)
				require.Equal(t, spec.name, remote["name"])
				project := auto.Project(workspace.Project{Name: "adaptive-integration", Runtime: workspace.NewProjectRuntimeInfo("go", nil)})
				owned, err := auto.UpsertStackInlineSource(context.Background(), r.Stack, "adaptive-integration", spec.program, project, auto.EnvVars(map[string]string{"PULUMI_BACKEND_URL": "file://" + backendDir, "PULUMI_CONFIG_PASSPHRASE": "adaptive-integration-test", "ADAPTIVE_URL": cfg.URL, "ADAPTIVE_SVC_TOKEN": cfg.ServiceToken, "PATH": os.Getenv("PATH")}))
				require.NoError(t, err)
				stack = &owned
				_, err = stack.Up(context.Background())
				require.NoError(t, err)
				_, err = stack.ImportResources(context.Background(), optimport.Resources([]*optimport.ImportResource{{Type: resourceTypeToken, Name: "db", ID: r.ID}}), optimport.Protect(false), optimport.GenerateCode(true), optimport.ShowSecrets(false))
				require.NoError(t, err)
				r.Checks = append(r.Checks, "actual-engine-import")
				state := harness.StateOutputs(t, stack, resourceTypeToken)
				spec.cluster, _ = state["defaultCluster"].(string)
				if initial != "" {
					require.Nil(t, state["password"], "import must not recover a held plaintext password")
				}
				r.Checks = append(r.Checks, "no-held-plaintext-recovered")
				spec.enabled = true
				// Import seeds observed fingerprints without inventing a credential diff.
				baseline := digests(t, stack)
				require.NotNil(t, baseline)
				if initial != "" {
					require.NotEmpty(t, baseline[spec.key()])
				} else {
					require.Empty(t, baseline)
				}
				preview, previewErr := stack.Preview(context.Background())
				require.NoError(t, previewErr)
				for _, line := range strings.Split(preview.StdOut, "\n") {
					if strings.Contains(line, "[diff:") {
						t.Log(line)
					}
				}
				require.Zero(t, preview.ChangeSummary["update"], "quiet import baseline")
				r.Checks = append(r.Checks, "quiet-import-baseline")
				if strings.HasPrefix(scenario, "import-") {
					if initial != "" {
						before := serverDigests(t, cfg, r.ID)
						spec.tags = []string{"metadata-update"}
						_, err = stack.Up(context.Background())
						require.Error(t, err)
						require.True(t, strings.Contains(err.Error(), "server-held values for password"), "omission rejection diagnostic")
						require.Equal(t, before, serverDigests(t, cfg, r.ID), "rejected update must retain held fingerprint")
						got := terraformRead(t, cfg, "resource", r.ID)
						require.Empty(t, got["userTags"], "rejected metadata write must not reach remote")
						r.Checks = append(r.Checks, "omitted-held-metadata-update-rejected-and-read-back")
						spec.password = &original
						harness.Up(t, stack)
						require.Equal(t, before, serverDigests(t, cfg, r.ID))
						require.Contains(t, terraformRead(t, cfg, "resource", r.ID)["userTags"], "metadata-update")
						r.Checks = append(r.Checks, "explicit-resupply-succeeds")
						spec.password = &empty
						harness.Up(t, stack)
						require.Empty(t, serverDigests(t, cfg, r.ID))
						r.Checks = append(r.Checks, "explicit-clear-succeeds")
					}
					harness.Refresh(t, stack)
					require.Zero(t, harness.Preview(t, stack)["update"])
					return
				}
				// Establish a real managed checkpoint before simulating its old shape.
				if initial != "" {
					spec.password = &original
				} else {
					spec.password = &empty
				}
				harness.Up(t, stack)
				saved := digests(t, stack)
				rotated := "synthetic-import-rotated"
				importJourneyWrite(t, cfg, spec, r.ID, rotated)
				current := serverDigests(t, cfg, r.ID)
				require.NotEmpty(t, current[spec.key()])
				require.NotEqual(t, saved, current)
				importJourneyCheckpoint(t, stack, func(resource map[string]any) {
					outputs := resource["outputs"].(map[string]any)
					delete(outputs, "appliedDigests")
					if scenario == "legacy-empty-buggy-seeded" {
						outputs["appliedDigests"] = current
					}
					if strings.HasSuffix(scenario, "v1-migration") {
						outputs["redactedDigests"] = saved
					}
				})
				for cycle := 0; cycle < 2; cycle++ {
					harness.Refresh(t, stack)
					observed := digests(t, stack)
					if scenario == "legacy-managed-no-baseline" {
						require.Equal(t, current, observed, "legacy unknown managed baseline quietly seeds observed fingerprint")
						require.Zero(t, harness.Preview(t, stack)["update"])
					} else {
						require.Equal(t, 1, harness.Preview(t, stack)["update"], "legacy explicit-empty or migrated old baseline must preserve drift")
						if strings.HasSuffix(scenario, "v1-migration") {
							require.Equal(t, saved, observed)
							require.NotContains(t, harness.StateOutputs(t, stack, resourceTypeToken), "redactedDigests")
						} else {
							require.NotNil(t, observed)
							require.Empty(t, observed)
						}
					}
				}
				r.Checks = append(r.Checks, "export-edit-import-checkpoint", "two-persisted-refresh-preview-cycles")
				if scenario != "legacy-managed-no-baseline" {
					harness.Up(t, stack)
					if strings.Contains(scenario, "legacy-empty") {
						require.Empty(t, serverDigests(t, cfg, r.ID))
					} else {
						require.Equal(t, saved, serverDigests(t, cfg, r.ID))
					}
					harness.Refresh(t, stack)
					require.Zero(t, harness.Preview(t, stack)["update"])
					r.Checks = append(r.Checks, "explicit-reconciliation-and-clean-preview")
				}
			})
		}
	}
}
