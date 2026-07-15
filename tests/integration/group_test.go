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

// TestGroup deploys a Resource + Endpoint + Group and verifies via the Client
// App API both that the group exists (teams list) and that the endpoint is
// actually associated with it (team endpoint list).
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
		// creator is a member implicitly. The group references the real endpoint,
		// whose association we verify below.
		grp, err := adaptive.NewGroup(ctx, "grp", &adaptive.GroupArgs{
			Name:      pulumi.String(grpName),
			Endpoints: pulumi.StringArray{ep.Name},
		})
		if err != nil {
			return err
		}
		ctx.Export("id", grp.ID())
		ctx.Export("endpointId", ep.ID())
		return nil
	})

	id := harness.StringOutput(t, outs, "id")
	epID := harness.StringOutput(t, outs, "endpointId")

	vc := client.New(cfg.URL, cfg.ClientID, cfg.ClientSecret)
	got := retryFind(t, "group "+id,
		vc.ListTeams,
		func(tm client.Team) bool { return tm.ID == id })
	assert.Equal(t, grpName, got.Name)

	// Verify the endpoint is actually associated with the group. team_endpoints
	// stores the endpoint's session id (EndpointID), which equals the Pulumi
	// endpoint's ID. Match by id so a spurious empty-id row (see the make+append
	// in the backend's TerraformCreateTeam) doesn't affect the result.
	te := retryFind(t, "endpoint "+epID+" in group "+id,
		func() ([]client.TeamEndpoint, error) { return vc.ListTeamEndpoints(id) },
		func(te client.TeamEndpoint) bool { return te.EndpointID == epID })
	assert.Equal(t, epID, te.EndpointID)
}
