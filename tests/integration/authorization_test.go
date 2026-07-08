//go:build integration

package integration

import (
	"testing"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/client"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
)

// TestAuthorization deploys an Authorization and verifies its metadata via the
// Client App API. (The permission policy body is not returned by the API, so we
// verify name/resourceType only — see CLIENT_API_GAPS.md.)
func TestAuthorization(t *testing.T) {
	cfg := harness.RequireConfig(t)
	name := uniqueName("pulumi-it-authz")

	outs := harness.Deploy(t, cfg, stackName("authz"), func(ctx *pulumi.Context) error {
		a, err := adaptive.NewAuthorization(ctx, "authz", &adaptive.AuthorizationArgs{
			Name:         pulumi.String(name),
			ResourceType: pulumi.String("postgres"),
			Description:  pulumi.String("integration-test read-only policy"),
			Permissions: pulumi.String(`allow:
  - database: production
    privileges:
      - SELECT
    objects:
      - ALL
`),
		})
		if err != nil {
			return err
		}
		ctx.Export("id", a.ID())
		return nil
	})

	id := harness.StringOutput(t, outs, "id")

	vc := client.New(cfg.URL, cfg.ClientID, cfg.ClientSecret)
	got := retryFind(t, "authorization "+id,
		vc.ListAuthorizations,
		func(a client.Authorization) bool { return a.ID == id })

	assert.Equal(t, name, got.Name)
	assert.Equal(t, "postgres", got.ResourceType)
}
