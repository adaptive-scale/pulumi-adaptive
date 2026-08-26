//go:build integration

package integration

import (
	"testing"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/client"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The per-type tests elsewhere in this package create an object and verify it.
// These go the whole way round — create, update, refresh, import, destroy —
// because the create path is the one least likely to be broken, and a provider
// that cannot update or cannot be imported cleanly is still broken.
//
// Each declares its resources through a closure over a mutable spec, so
// harness.Up re-runs the same program against changed inputs and produces a
// genuine update rather than a second stack.
//
// Written per type rather than as one table: Endpoint needs a Resource, Script
// needs an Endpoint, DataProtection needs a Resource, so a shared matrix would
// contort more than it saves.

// TestResourceLifecycle is the fullest of these, and the template the others
// follow.
func TestResourceLifecycle(t *testing.T) {
	t.Parallel()
	cfg := harness.RequireConfig(t)

	name := uniqueName("pulumi-it-lc-res")
	spec := struct {
		Port string
		Tags pulumi.StringArray
	}{Port: "5432"}

	outs, stack := harness.DeployStack(t, cfg, stackName("lc-res"), func(ctx *pulumi.Context) error {
		db, err := adaptive.NewResource(ctx, "db", &adaptive.ResourceArgs{
			Name:     pulumi.String(name),
			Type:     pulumi.String("postgres"),
			Host:     pulumi.String("db.example.com"),
			Port:     pulumi.String(spec.Port),
			Username: pulumi.String("admin"),
			Password: pulumi.String("not-a-real-password"),
			SslMode:  pulumi.String("require"),
			Tags:     spec.Tags,
		})
		if err != nil {
			return err
		}
		ctx.Export("id", db.ID())
		return nil
	})
	id := harness.StringOutput(t, outs, "id")

	vc := client.New(cfg.URL, cfg.ClientID, cfg.ClientSecret)
	got := retryFind(t, "resource "+id, vc.ListResources,
		func(r client.Resource) bool { return r.ID == id })
	assert.Equal(t, name, got.Name)

	// Update. Tags go from absent to present as well as a scalar changing,
	// because an added collection and a changed scalar take different paths
	// through the diff.
	spec.Port = "5433"
	spec.Tags = pulumi.StringArray{pulumi.String("env=integration")}
	harness.Up(t, stack)
	harness.AssertRefreshClean(t, stack)

	// Import into a fresh stack and confirm the imported state needs no changes.
	// This is what catches state-shape regressions: a renamed or undecodable
	// property fails here and nowhere else.
	_, importStack := harness.DeployStack(t, cfg, stackName("lc-res-imp"),
		func(ctx *pulumi.Context) error { return nil })
	generated := harness.ImportResource(t, importStack, "adaptive:index:Resource", "imported", id)
	assert.Contains(t, generated, name, "generated code should carry the resource name")

	harness.Destroy(t, stack)
	assertResourceGone(t, vc, id)
}

func assertResourceGone(t *testing.T, vc *client.Client, id string) {
	t.Helper()
	all, err := vc.ListResources()
	require.NoError(t, err)
	for _, r := range all {
		if r.ID == id {
			t.Errorf("resource %s still present after destroy", id)
		}
	}
}

func TestAuthorizationLifecycle(t *testing.T) {
	t.Parallel()
	cfg := harness.RequireConfig(t)

	name := uniqueName("pulumi-it-lc-authz")
	desc := "integration-test read-only policy"

	outs, stack := harness.DeployStack(t, cfg, stackName("lc-authz"), func(ctx *pulumi.Context) error {
		a, err := adaptive.NewAuthorization(ctx, "authz", &adaptive.AuthorizationArgs{
			Name:         pulumi.String(name),
			ResourceType: pulumi.String("postgres"),
			Description:  pulumi.String(desc),
			Permissions:  pulumi.String("allow:\n  - database: production\n    privileges:\n      - SELECT\n    objects:\n      - ALL\n"),
		})
		if err != nil {
			return err
		}
		ctx.Export("id", a.ID())
		return nil
	})
	id := harness.StringOutput(t, outs, "id")

	vc := client.New(cfg.URL, cfg.ClientID, cfg.ClientSecret)
	got := retryFind(t, "authorization "+id, vc.ListAuthorizations,
		func(a client.Authorization) bool { return a.ID == id })
	assert.Equal(t, desc, got.Description)

	desc = "integration-test read-only policy (updated)"
	harness.Up(t, stack)
	updated := retryFind(t, "authorization "+id, vc.ListAuthorizations,
		func(a client.Authorization) bool { return a.ID == id && a.Description == desc })
	assert.Equal(t, desc, updated.Description)

	harness.AssertRefreshClean(t, stack)
	harness.Destroy(t, stack)
}

func TestGroupLifecycle(t *testing.T) {
	t.Parallel()
	cfg := harness.RequireConfig(t)

	grpName := uniqueName("pulumi-it-lc-grp")
	slack := ""

	outs, stack := harness.DeployStack(t, cfg, stackName("lc-grp"), func(ctx *pulumi.Context) error {
		args := &adaptive.GroupArgs{Name: pulumi.String(grpName)}
		if slack != "" {
			args.SlackChannelId = pulumi.String(slack)
		}
		g, err := adaptive.NewGroup(ctx, "grp", args)
		if err != nil {
			return err
		}
		ctx.Export("id", g.ID())
		return nil
	})
	id := harness.StringOutput(t, outs, "id")

	vc := client.New(cfg.URL, cfg.ClientID, cfg.ClientSecret)
	retryFind(t, "team "+id, vc.ListTeams, func(g client.Team) bool { return g.ID == id })

	// The slack channel only lands via a follow-up update, so setting it here
	// exercises the create-then-update path the provider relies on.
	slack = "C0INTEGRATION"
	harness.Up(t, stack)
	harness.AssertRefreshClean(t, stack)
	harness.Destroy(t, stack)
}

func TestScheduleLifecycle(t *testing.T) {
	t.Parallel()
	cfg := harness.RequireProviderConfig(t)

	name := uniqueName("pulumi-it-lc-sched")
	endHour := 17

	_, stack := harness.DeployStack(t, cfg, stackName("lc-sched"), func(ctx *pulumi.Context) error {
		s, err := adaptive.NewSchedule(ctx, "sched", &adaptive.ScheduleArgs{
			Name:         pulumi.String(name),
			ScheduleType: pulumi.String("custom"),
			Weekdays:     pulumi.StringArray{pulumi.String("Monday"), pulumi.String("TUESDAY")},
			StartHour:    pulumi.Int(9),
			StartMinute:  pulumi.Int(0),
			EndHour:      pulumi.Int(endHour),
			EndMinute:    pulumi.Int(30),
		})
		if err != nil {
			return err
		}
		ctx.Export("id", s.ID())
		return nil
	})

	endHour = 18
	harness.Up(t, stack)
	harness.AssertRefreshClean(t, stack)
	harness.Destroy(t, stack)
}
