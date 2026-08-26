//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Secret drift can only be exercised here. The server withholds secret values
// and reports an opaque fingerprint instead, so the fingerprints only move
// against a real backend — a unit test has to supply them by hand, which proves
// the classification but not that the mechanism is wired up.
//
// These tests need no Client App: Pulumi does the writing and rawTerraformAPI
// does the out-of-band changing, so RequireProviderConfig is enough and they run
// against a local server with just a service token.

const resourceTypeToken = "adaptive:index:Resource"

// pgSpec is the configuration a drift test deploys. The program closes over it,
// so mutating a field and calling harness.Up produces a real update.
type pgSpec struct {
	Name     string
	Password string // "" means the program does not manage it
	RootCert string // "" means the program does not manage it
}

func (s *pgSpec) declare(ctx *pulumi.Context) error {
	args := &adaptive.ResourceArgs{
		Name:     pulumi.String(s.Name),
		Type:     pulumi.String("postgres"),
		Host:     pulumi.String("db.example.com"),
		Port:     pulumi.String("5432"),
		Username: pulumi.String("admin"),
		SslMode:  pulumi.String("require"),
	}
	if s.Password != "" {
		args.Password = pulumi.String(s.Password)
	}
	if s.RootCert != "" {
		args.TlsRootCert = pulumi.String(s.RootCert)
	}
	db, err := adaptive.NewResource(ctx, "db", args)
	if err != nil {
		return err
	}
	ctx.Export("id", db.ID())
	return nil
}

// rotateServerSide replaces the whole stored configuration, which is what the
// update route does — so every field has to be present, not just the one being
// changed.
func rotateServerSide(t *testing.T, cfg harness.Config, id string, fields map[string]string) {
	t.Helper()
	yaml := ""
	for k, v := range map[string]string{
		"username": "admin", "hostname": "db.example.com",
		"port": "5432", "sslMode": "require",
	} {
		yaml += fmt.Sprintf("%s: %q\n", k, v)
	}
	for k, v := range fields {
		yaml += fmt.Sprintf("%s: %q\n", k, v)
	}
	resp, body := rawTerraformAPI(t, cfg, "POST", "/resource/update/"+id, map[string]any{
		"integrationType": "postgres",
		"config":          yaml,
		"userTags":        []string{},
	})
	require.Equal(t, http.StatusOK, resp.StatusCode, "out-of-band update failed: %s", body)
}

// digests reads the fingerprints the provider recorded at its last write.
func digests(t *testing.T, stack *auto.Stack) map[string]string {
	t.Helper()
	return harness.StateStringMap(t, stack, resourceTypeToken, "appliedDigests")
}

// deployDrift stands up a postgres resource and returns its id, the stack, and
// the spec the program closes over.
func deployDrift(t *testing.T, cfg harness.Config, label string, spec *pgSpec) (string, *auto.Stack) {
	t.Helper()
	outs, stack := harness.DeployStack(t, cfg, stackName(label), spec.declare)
	return harness.StringOutput(t, outs, "id"), stack
}

// A secret the program sets, changed on the server, must show up in the plan and
// be re-applied by the next up — and the drift must be gone afterwards.
func TestSecretDriftManagedField(t *testing.T) {
	t.Parallel()
	cfg := harness.RequireProviderConfig(t)

	spec := &pgSpec{Name: uniqueName("pulumi-it-drift-managed"), Password: "original-password"}
	id, stack := deployDrift(t, cfg, "drift-managed", spec)

	before := digests(t, stack)
	require.NotEmpty(t, before["password"],
		"no fingerprint recorded at create; the rest of this test cannot mean anything")

	rotateServerSide(t, cfg, id, map[string]string{"password": "rotated-outside-pulumi"})

	harness.Refresh(t, stack)
	assert.Equal(t, before, digests(t, stack),
		"refresh must not move the recorded fingerprints — that baseline is what the next diff compares against")

	changes := harness.Preview(t, stack)
	assert.NotZero(t, changes["update"], "the rotated secret should show as an update, got %v", changes)

	harness.Up(t, stack)
	assert.NotEqual(t, before["password"], digests(t, stack)["password"],
		"update must re-record the fingerprint, or the drift is reported forever")

	// Convergence. A marker that survives its own up is a perpetual diff, which
	// is the failure mode this design is most exposed to.
	harness.AssertRefreshClean(t, stack)
	assert.Zero(t, harness.Preview(t, stack)["update"], "drift should be resolved after up")
}

// The reported bug: a secret the program does NOT set. This produced no diff at
// all before the marker was introduced, because the old signal was to blank a
// field that was never set.
func TestSecretDriftUnmanagedField(t *testing.T) {
	t.Parallel()
	cfg := harness.RequireProviderConfig(t)

	// No password in the program.
	spec := &pgSpec{Name: uniqueName("pulumi-it-drift-unmanaged")}
	id, stack := deployDrift(t, cfg, "drift-unmanaged", spec)

	rotateServerSide(t, cfg, id, map[string]string{"password": "set-outside-pulumi"})
	harness.Refresh(t, stack)

	changes := harness.Preview(t, stack)
	assert.NotZero(t, changes["update"],
		"a secret added outside the program must still reach the plan; silence here is the original bug (%v)", changes)

	// The program owns the whole configuration, so reconciling means clearing it.
	harness.Up(t, stack)
	harness.AssertRefreshClean(t, stack)
	assert.Zero(t, harness.Preview(t, stack)["update"], "should converge after one up")
}

// rootCert is withheld by the server like any password, but its argument was not
// declared secret — so drift on it degraded to a warning instead of a diff. This
// is the case that surfaced in a live run.
func TestSecretDriftRootCert(t *testing.T) {
	t.Parallel()
	cfg := harness.RequireProviderConfig(t)

	const cert = "-----BEGIN CERTIFICATE-----\noriginal\n-----END CERTIFICATE-----"
	spec := &pgSpec{
		Name:     uniqueName("pulumi-it-drift-cert"),
		Password: "original-password",
		RootCert: cert,
	}
	id, stack := deployDrift(t, cfg, "drift-cert", spec)

	before := digests(t, stack)
	require.NotEmpty(t, before["rootCert"],
		"the server should withhold and fingerprint rootCert; if it does not, this test proves nothing")

	rotateServerSide(t, cfg, id, map[string]string{
		"password": "original-password",
		"rootCert": "-----BEGIN CERTIFICATE-----\nrotated\n-----END CERTIFICATE-----",
	})

	harness.Refresh(t, stack)
	changes := harness.Preview(t, stack)
	assert.NotZero(t, changes["update"],
		"a drifted rootCert must show as a diff, not only a warning (%v)", changes)

	harness.Up(t, stack)
	harness.AssertRefreshClean(t, stack)
}

// The common case, and the one that matters most: nothing changed anywhere. A
// misfiring rule turns every preview into a diff for every resource holding a
// secret, which would be worse than the bug it fixes. Run twice, because a rule
// that churns state settles only on the second pass.
func TestSecretDriftQuietWhenNothingChanged(t *testing.T) {
	t.Parallel()
	cfg := harness.RequireProviderConfig(t)

	spec := &pgSpec{Name: uniqueName("pulumi-it-drift-quiet"), Password: "original-password"}
	_, stack := deployDrift(t, cfg, "drift-quiet", spec)

	first := digests(t, stack)
	for i := 1; i <= 2; i++ {
		harness.AssertRefreshClean(t, stack)
		assert.Empty(t, harness.Preview(t, stack)["update"], "pass %d: preview should be clean", i)
		assert.Equal(t, first, digests(t, stack), "pass %d: fingerprints should not churn", i)
	}
}
