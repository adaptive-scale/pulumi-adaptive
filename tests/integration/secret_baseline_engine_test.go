//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// This exercises the real Pulumi engine and local provider binary, but uses a
// fake API with synthetic fingerprints. It requires no Adaptive credentials and
// does not establish compatibility with a running Adaptive backend.
func TestEmptySecretBaselineEngine(t *testing.T) {
	pluginDir, err := filepath.Abs("../../provider/bin")
	require.NoError(t, err)
	t.Setenv("PATH", pluginDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var mu sync.Mutex
	exists := false
	writes := 0
	guardStatus := 0 // Enabled only after refresh, during stack.Up.
	var configuration map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && (r.URL.Path == "/api/v1/terraform/resource/create" || r.URL.Path == "/api/v1/terraform/resource/update/engine-resource"):
			var request struct {
				Config string `json:"config"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			var cfg map[string]any
			if err := yaml.Unmarshal([]byte(request.Config), &cfg); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			if value, ok := cfg["databasePassword"]; !ok || value != "" {
				t.Error("the engine must send explicit clear intent, never a fingerprint")
				w.WriteHeader(400)
				return
			}
			configuration, exists = cfg, true
			writes++
			_, _ = w.Write([]byte(`{"id":"engine-resource"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/terraform/resource/read/engine-resource":
			if guardStatus != 0 {
				w.WriteHeader(guardStatus)
				if guardStatus == http.StatusOK {
					_, _ = w.Write([]byte(`{"id":"engine-resource","name":"engine-snowflake","integrationType":"snowflake"}`))
				}
				return
			}
			if !exists {
				w.WriteHeader(404)
				return
			}
			cfg := make(map[string]any, len(configuration))
			for key, value := range configuration {
				cfg[key] = value
			}
			body := map[string]any{"id": "engine-resource", "name": "engine-snowflake", "integrationType": "snowflake", "configuration": cfg}
			if password, ok := cfg["databasePassword"].(string); ok && password != "" {
				delete(cfg, "databasePassword")
				body["redactedKeys"] = []string{"databasePassword"}
				body["redactedDigests"] = map[string]string{"databasePassword": "synthetic-remote-fingerprint"}
			}
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/terraform/resource/delete/engine-resource":
			exists = false
			_, _ = w.Write([]byte(`true`))
		default:
			t.Errorf("unexpected API route: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(api.Close)

	omitPassword := false
	cfg := harness.Config{URL: api.URL, ServiceToken: "synthetic-engine-test-token"}
	_, stack := harness.DeployStack(t, cfg, "empty-secret-baseline", func(ctx *pulumi.Context) error {
		args := &adaptive.ResourceArgs{Name: pulumi.String("engine-snowflake"), Type: pulumi.String("snowflake")}
		if !omitPassword {
			args.Password = pulumi.String("")
		}
		_, err := adaptive.NewResource(ctx, "db", args)
		return err
	})
	require.NotNil(t, digests(t, stack), "create must record a known-empty baseline")
	require.Empty(t, digests(t, stack))
	require.Zero(t, harness.Preview(t, stack)["update"])

	// Inject only this disposable stack's legacy empty-map checkpoint. This
	// exercises the SDK's migration encoder through persisted engine state,
	// not merely an in-process Read response.
	importJourneyCheckpoint(t, stack, func(resource map[string]any) {
		outputs := resource["outputs"].(map[string]any)
		delete(outputs, "appliedDigests")
		outputs["redactedDigests"] = map[string]any{}
	})
	for cycle := 1; cycle <= 2; cycle++ {
		harness.Refresh(t, stack)
		require.NotContains(t, harness.StateOutputs(t, stack, resourceTypeToken), "redactedDigests", "legacy empty property must disappear on cycle %d", cycle)
		require.NotNil(t, digests(t, stack))
		require.Empty(t, digests(t, stack))
		require.Zero(t, harness.Preview(t, stack)["update"])
	}

	mu.Lock()
	configuration["databasePassword"] = "synthetic-remote-password"
	mu.Unlock()
	for cycle := 1; cycle <= 3; cycle++ {
		harness.Refresh(t, stack)
		require.Equal(t, 1, harness.Preview(t, stack)["update"], "cycle %d", cycle)
		require.Empty(t, digests(t, stack), "refresh must not accept the conflicting fingerprint")
	}

	// A changed program omitting the password still must not silently clear it.
	omitPassword = true
	for _, status := range []int{http.StatusForbidden, http.StatusInternalServerError, http.StatusNotFound, http.StatusOK} {
		mu.Lock()
		guardStatus = status
		before := writes
		mu.Unlock()
		_, err = stack.Up(context.Background())
		mu.Lock()
		after := writes
		guardStatus = 0
		mu.Unlock()
		require.ErrorContains(t, err, "cannot verify server-held values", "guard status %d", status)
		require.Equal(t, before, after, "failed guard status %d must not mutate", status)
	}
	mu.Lock()
	before := writes
	mu.Unlock()
	_, err = stack.Up(context.Background())
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "server-held values for password"))
	mu.Lock()
	after := writes
	mu.Unlock()
	require.Equal(t, before, after, "rejected omission must not reach the mutation route")

	omitPassword = false
	harness.Up(t, stack)
	mu.Lock()
	password := configuration["databasePassword"]
	mu.Unlock()
	require.Equal(t, "", password)
	harness.Refresh(t, stack)
	require.Zero(t, harness.Preview(t, stack)["update"])
	harness.Destroy(t, stack)
	mu.Lock()
	remaining := exists
	mu.Unlock()
	require.False(t, remaining, "destroy must remove the exact fixture resource")
}
