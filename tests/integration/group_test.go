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

// TestGroup deploys a Resource + Endpoint + Group and verifies the group via the
// Client App API teams list.
func TestGroup(t *testing.T) {
	cfg := harness.RequireConfig(t)
	resName := uniqueName("pulumi-it-res")
	epName := uniqueName("pulumi-it-ep")
	grpName := uniqueName("pulumi-it-grp")

	outs := harness.Deploy(t, cfg, stackName("grp"), func(ctx *pulumi.Context) error {
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
			// No users: defaults to the creator (the token owner), who always
			// exists — avoids "user not found" for a fake email.
		})
		if err != nil {
			return err
		}
		// Members omitted: a fake email would fail "user not found", and the
		// creator is a member implicitly. The group still references a real
		// endpoint, which is what we verify.
		grp, err := adaptive.NewGroup(ctx, "grp", &adaptive.GroupArgs{
			Name:      pulumi.String(grpName),
			Endpoints: pulumi.StringArray{ep.Name},
		})
		if err != nil {
			return err
		}
		ctx.Export("id", grp.ID())
		return nil
	})

	id := harness.StringOutput(t, outs, "id")

	vc := client.New(cfg.URL, cfg.ClientID, cfg.ClientSecret)
	got := retryFind(t, "group "+id,
		vc.ListTeams,
		func(tm client.Team) bool { return tm.ID == id })

	assert.Equal(t, grpName, got.Name)
}
