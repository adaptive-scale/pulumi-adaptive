//go:build integration

package integration

import (
	"os"
	"testing"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The provider declares no configuration: credentials come from the environment
// only, so they never reach stack state. Two things follow that are worth
// pinning, because both fail silently rather than loudly.
//
// Not parallel: it inspects process-wide state and, in the second case, the
// stack's config file.

// Nothing credential-shaped may appear in the exported state. If provider config
// ever comes back, the token lands in the stack again — encrypted, but present,
// which is what made stacks unreadable without their passphrase.
func TestCredentialsAbsentFromState(t *testing.T) {
	cfg := harness.RequireProviderConfig(t)

	name := uniqueName("pulumi-it-creds")
	_, stack := harness.DeployStack(t, cfg, stackName("creds"), func(ctx *pulumi.Context) error {
		_, err := adaptive.NewResource(ctx, "db", &adaptive.ResourceArgs{
			Name:     pulumi.String(name),
			Type:     pulumi.String("postgres"),
			Host:     pulumi.String("db.example.com"),
			Port:     pulumi.String("5432"),
			Username: pulumi.String("admin"),
			Password: pulumi.String("not-a-real-password"),
			SslMode:  pulumi.String("require"),
		})
		return err
	})

	dep, err := stack.Export(t.Context())
	require.NoError(t, err)
	state := string(dep.Deployment)

	for _, forbidden := range []string{"serviceToken", "workspaceUrl"} {
		assert.NotContains(t, state, forbidden,
			"%q is in the stack state; provider config is meant to be gone", forbidden)
	}
	// The token value itself must not appear anywhere, under any key.
	if tok := os.Getenv("ADAPTIVE_SVC_TOKEN"); tok != "" {
		assert.NotContains(t, state, tok, "the service token leaked into stack state")
	}
}

// A stale `pulumi config set adaptive:serviceToken` left over from an older
// provider must be ignored, not rejected. The provider no longer declares that
// key, and an upgrade that started failing on leftover config would strand every
// existing stack.
func TestStaleProviderConfigIsIgnored(t *testing.T) {
	cfg := harness.RequireProviderConfig(t)

	name := uniqueName("pulumi-it-staleconf")
	_, stack := harness.DeployStack(t, cfg, stackName("staleconf"), func(ctx *pulumi.Context) error {
		_, err := adaptive.NewResource(ctx, "db", &adaptive.ResourceArgs{
			Name:     pulumi.String(name),
			Type:     pulumi.String("postgres"),
			Host:     pulumi.String("db.example.com"),
			Port:     pulumi.String("5432"),
			Username: pulumi.String("admin"),
			Password: pulumi.String("not-a-real-password"),
			SslMode:  pulumi.String("require"),
		})
		return err
	})

	// Both a bogus token and a URL pointing nowhere: if either were still
	// honoured, the operations below would fail to reach the real backend.
	require.NoError(t, stack.SetConfig(t.Context(), "adaptive:serviceToken",
		auto.ConfigValue{Value: "stale-and-wrong", Secret: true}))
	require.NoError(t, stack.SetConfig(t.Context(), "adaptive:workspaceUrl",
		auto.ConfigValue{Value: "https://stale.invalid"}))

	harness.AssertRefreshClean(t, stack)
	assert.Zero(t, harness.Preview(t, stack)["update"],
		"leftover adaptive:* config must be ignored, not honoured or rejected")
}
