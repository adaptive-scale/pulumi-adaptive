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

// TestEndpoint deploys a Resource + Endpoint and verifies the endpoint via the
// Client App API. The endpoint record is created whether or not Adaptive can
// actually reach the target, so this test does not require a reachable database.
func TestEndpoint(t *testing.T) {
	cfg := harness.RequireConfig(t)
	resName := uniqueName("pulumi-it-res")
	epName := uniqueName("pulumi-it-ep")

	outs := harness.Deploy(t, cfg, stackName("ep"), func(ctx *pulumi.Context) error {
		db, err := adaptive.NewResource(ctx, "db", &adaptive.ResourceArgs{
			Name:     pulumi.String(resName),
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
		ep, err := adaptive.NewEndpoint(ctx, "ep", &adaptive.EndpointArgs{
			Name:     pulumi.String(epName),
			Resource: db.Name,
			Ttl:      pulumi.String("8h"),
			// No users specified: the endpoint defaults to the creator (the token
			// owner), who always exists — avoids "user not found" for a fake email.
		})
		if err != nil {
			return err
		}
		ctx.Export("id", ep.ID())
		return nil
	})

	id := harness.StringOutput(t, outs, "id")

	vc := client.New(cfg.URL, cfg.ClientID, cfg.ClientSecret)
	got := retryFind(t, "endpoint "+id,
		vc.ListEndpoints,
		func(e client.Endpoint) bool { return e.ID == id })

	assert.Equal(t, epName, got.Name)
	assert.Equal(t, "postgres", got.IntegrationType)
	// We intentionally do NOT assert Status == "created". In Adaptive an endpoint
	// is a live proxy backed by a container: "created" means that container
	// started up and is ready to accept connections, while "failed" means it
	// could not start (e.g. bad credentials / unreachable target) — see
	// SessionStatus* in inventorize model/session.go. The record itself is listed
	// regardless of container state, so a dummy target yields a real endpoint
	// record in "failed" state. The provider's job is to create that record with
	// the right metadata (asserted above); reaching "created" would require a
	// real reachable database. Status is logged for visibility only.
	t.Logf("endpoint %s status: %q", id, got.Status)
}
