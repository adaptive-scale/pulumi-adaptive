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

// TestResource deploys a postgres Resource via the Pulumi SDK and verifies it is
// visible through the Client App API with the expected metadata.
func TestResource(t *testing.T) {
	cfg := harness.RequireConfig(t)
	name := uniqueName("pulumi-it-res")

	outs := harness.Deploy(t, cfg, stackName("res"), func(ctx *pulumi.Context) error {
		db, err := adaptive.NewResource(ctx, "db", &adaptive.ResourceArgs{
			Name:     pulumi.String(name),
			Type:     pulumi.String("postgres"),
			Host:     pulumi.String("db.example.com"),
			Port:     pulumi.String("5432"),
			Username: pulumi.String("admin"),
			Password: pulumi.String("not-a-real-password"),
			SslMode:  pulumi.String("require"),
		})
		if err != nil {
			return err
		}
		ctx.Export("id", db.ID())
		return nil
	})

	id := harness.StringOutput(t, outs, "id")

	vc := client.New(cfg.URL, cfg.ClientID, cfg.ClientSecret)
	got := retryFind(t, "resource "+id,
		vc.ListResources,
		func(r client.Resource) bool { return r.ID == id })

	assert.Equal(t, name, got.Name)
	assert.Equal(t, "postgres", got.Type)
}
