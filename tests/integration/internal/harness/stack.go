//go:build integration

package harness

import (
	"bytes"
	"context"
	"os/exec"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const projectName = "adaptive-integration"

// Deploy stands up an ephemeral, local-file-backend Pulumi stack from the given
// inline program, runs `up`, and registers `destroy` + stack removal on
// t.Cleanup (so resources are torn down even if assertions fail). It returns the
// stack outputs. The program should export whatever ids the test needs to verify.
//
// Requires the `pulumi-resource-adaptive` plugin on PATH (run `make install`).
func Deploy(t *testing.T, cfg Config, stackName string, program pulumi.RunFunc) auto.OutputMap {
	t.Helper()

	if _, err := exec.LookPath("pulumi-resource-adaptive"); err != nil {
		t.Fatalf("pulumi-resource-adaptive not found on PATH; run `make install` first: %v", err)
	}

	ctx := context.Background()
	backendDir := t.TempDir()
	env := map[string]string{
		"PULUMI_BACKEND_URL":       "file://" + backendDir,
		"PULUMI_CONFIG_PASSPHRASE": "adaptive-integration-test",
		"ADAPTIVE_URL":             cfg.URL,
		"ADAPTIVE_SVC_TOKEN":       cfg.ServiceToken,
	}

	proj := auto.Project(workspace.Project{
		Name:    projectName,
		Runtime: workspace.NewProjectRuntimeInfo("go", nil),
	})

	stack, err := auto.UpsertStackInlineSource(ctx, stackName, projectName, program, proj, auto.EnvVars(env))
	if err != nil {
		t.Fatalf("create stack %q: %v", stackName, err)
	}

	t.Cleanup(func() {
		if _, err := stack.Destroy(context.Background(), optdestroy.ProgressStreams(&logWriter{t})); err != nil {
			t.Logf("WARNING: destroy of stack %q failed; manual cleanup may be needed: %v", stackName, err)
			return
		}
		_ = stack.Workspace().RemoveStack(context.Background(), stackName)
	})

	res, err := stack.Up(ctx, optup.ProgressStreams(&logWriter{t}))
	if err != nil {
		t.Fatalf("pulumi up for stack %q: %v", stackName, err)
	}
	return res.Outputs
}

// StringOutput extracts a required string stack output.
func StringOutput(t *testing.T, outs auto.OutputMap, key string) string {
	t.Helper()
	v, ok := outs[key]
	if !ok {
		t.Fatalf("missing stack output %q", key)
	}
	s, ok := v.Value.(string)
	if !ok {
		t.Fatalf("stack output %q is not a string: %T", key, v.Value)
	}
	return s
}

// logWriter routes Pulumi engine progress to the test log.
type logWriter struct{ t *testing.T }

func (w *logWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s", bytes.TrimRight(p, "\n"))
	return len(p), nil
}
