//go:build integration

package harness

import (
	"bytes"
	"context"
	"os/exec"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optimport"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optrefresh"
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
	outs, _ := DeployStack(t, cfg, stackName, program)
	return outs
}

// DeployStack is Deploy but also returns the stack, for tests that go on to
// refresh, import into, or re-up the same stack.
func DeployStack(t *testing.T, cfg Config, stackName string, program pulumi.RunFunc) (auto.OutputMap, *auto.Stack) {
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
	return res.Outputs, &stack
}

// Refresh runs `pulumi refresh` on the stack and returns the per-op resource
// change counts (e.g. {"same": 2, "delete": 1}).
func Refresh(t *testing.T, stack *auto.Stack) map[string]int {
	t.Helper()
	res, err := stack.Refresh(context.Background(), optrefresh.ProgressStreams(&logWriter{t}))
	if err != nil {
		t.Fatalf("pulumi refresh: %v", err)
	}
	if res.Summary.ResourceChanges == nil {
		return map[string]int{}
	}
	return *res.Summary.ResourceChanges
}

// AssertRefreshClean refreshes the stack and fails if any resource changed:
// every operation reported must be "same". This is the perpetual-diff
// regression check.
func AssertRefreshClean(t *testing.T, stack *auto.Stack) {
	t.Helper()
	changes := Refresh(t, stack)
	for op, n := range changes {
		if op != "same" && n > 0 {
			t.Errorf("refresh not clean: %d %q operations (all changes: %v)", n, op, changes)
		}
	}
}

// ImportResource imports an existing Adaptive object into the stack by ID and
// returns the generated program text. typ is the Pulumi type token, e.g.
// "adaptive:index:Endpoint".
func ImportResource(t *testing.T, stack *auto.Stack, typ, name, id string) string {
	t.Helper()
	code, err := ImportResourceErr(t, stack, typ, name, id)
	if err != nil {
		t.Fatalf("pulumi import %s %s %s: %v", typ, name, id, err)
	}
	return code
}

// ImportResourceErr is ImportResource without the fatal: it hands the error
// back so tests can assert that an import is *rejected*.
func ImportResourceErr(t *testing.T, stack *auto.Stack, typ, name, id string) (string, error) {
	t.Helper()
	res, err := stack.ImportResources(context.Background(),
		optimport.Resources([]*optimport.ImportResource{{Type: typ, Name: name, ID: id}}),
		optimport.ProgressStreams(&logWriter{t}),
	)
	if err != nil {
		return "", err
	}
	return res.GeneratedCode, nil
}

// Preview returns the per-op change counts a `pulumi preview` would apply.
func Preview(t *testing.T, stack *auto.Stack) map[string]int {
	t.Helper()
	res, err := stack.Preview(context.Background(), optpreview.ProgressStreams(&logWriter{t}))
	if err != nil {
		t.Fatalf("pulumi preview: %v", err)
	}
	out := make(map[string]int, len(res.ChangeSummary))
	for op, n := range res.ChangeSummary {
		out[string(op)] = n
	}
	return out
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
