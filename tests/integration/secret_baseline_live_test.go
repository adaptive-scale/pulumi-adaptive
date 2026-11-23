//go:build integration

package integration

import (
	"testing"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/require"
)

// Opt-in live-backend counterpart of TestEmptySecretBaselineEngine. Creates and
// deletes one disposable integration; requires an authorized service token.
func TestSecretDriftExplicitEmptyBaseline(t *testing.T) {
	cfg := harness.RequireProviderConfig(t)
	name := uniqueName("pulumi-it-empty-secret")
	outputs, stack := harness.DeployStack(t, cfg, stackName("empty-secret"), func(ctx *pulumi.Context) error {
		db, err := adaptive.NewResource(ctx, "db", &adaptive.ResourceArgs{
			Name: pulumi.String(name), Type: pulumi.String("postgres"),
			Host: pulumi.String("db.example.com"), Port: pulumi.String("5432"),
			Username: pulumi.String("admin"), SslMode: pulumi.String("require"),
			Password: pulumi.String(""),
		})
		if err != nil {
			return err
		}
		ctx.Export("id", db.ID())
		return nil
	})
	id := harness.StringOutput(t, outputs, "id")
	require.NotNil(t, digests(t, stack), "create must capture known absence, not an unknown baseline")
	require.Empty(t, digests(t, stack))
	rotateServerSide(t, cfg, id, name, map[string]string{"password": "synthetic-out-of-band-password"})
	for cycle := 1; cycle <= 3; cycle++ {
		harness.Refresh(t, stack)
		require.Equal(t, 1, harness.Preview(t, stack)["update"], "cycle %d", cycle)
		require.Empty(t, digests(t, stack), "refresh must not adopt the out-of-band credential")
	}
	harness.Up(t, stack)
	require.Empty(t, serverDigests(t, cfg, id))
	remote := terraformRead(t, cfg, "resource", id)
	configuration, ok := remote["configuration"].(map[string]any)
	require.True(t, ok)
	password, present := configuration["password"]
	require.True(t, present, "the backend must acknowledge the explicit clear")
	require.Equal(t, "", password)
	harness.Refresh(t, stack)
	require.Zero(t, harness.Preview(t, stack)["update"])
	harness.Destroy(t, stack)
	assertTerraformGone(t, cfg, "resource", id)
}
