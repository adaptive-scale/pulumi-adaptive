//go:build integration

package integration

import (
	"testing"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
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
	cfg := harness.RequireProviderConfig(t)

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

	assert.Equal(t, name, terraformRead(t, cfg, "resource", id)["name"])

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
	assertTerraformGone(t, cfg, "resource", id)
}

func TestAuthorizationLifecycle(t *testing.T) {
	t.Parallel()
	cfg := harness.RequireProviderConfig(t)

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

	assert.Equal(t, desc, terraformRead(t, cfg, "authorization", id)["description"])

	desc = "integration-test read-only policy (updated)"
	harness.Up(t, stack)
	assert.Equal(t, desc, terraformRead(t, cfg, "authorization", id)["description"],
		"the updated description should be readable back")

	harness.AssertRefreshClean(t, stack)
	harness.Destroy(t, stack)
	assertTerraformGone(t, cfg, "authorization", id)
}

func TestGroupLifecycle(t *testing.T) {
	t.Parallel()
	cfg := harness.RequireProviderConfig(t)

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

	assert.Equal(t, grpName, terraformRead(t, cfg, "team", id)["name"])

	// The slack channel only lands via a follow-up update, so setting it here
	// exercises the create-then-update path the provider relies on.
	slack = "C0INTEGRATION"
	harness.Up(t, stack)
	assert.Equal(t, slack, terraformRead(t, cfg, "team", id)["slackChannelId"],
		"the slack channel should reach the server on update")

	harness.AssertRefreshClean(t, stack)
	harness.Destroy(t, stack)
	assertTerraformGone(t, cfg, "team", id)
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

// Endpoint has the most optional arguments of any type — TTL, JIT mode and
// approvers, the web-access toggles — and several are reconciled specially on
// read because the server supplies its own defaults. That makes its update path
// the likeliest place for a reconciliation bug, and it had no coverage.
func TestEndpointLifecycle(t *testing.T) {
	t.Parallel()
	cfg := harness.RequireProviderConfig(t)

	resName := uniqueName("pulumi-it-lc-ep-res")
	epName := uniqueName("pulumi-it-lc-ep")
	spec := struct {
		TTL              string
		ScriptOnlyAccess bool
	}{TTL: "8h"}

	outs, stack := harness.DeployStack(t, cfg, stackName("lc-ep"), func(ctx *pulumi.Context) error {
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
		args := &adaptive.EndpointArgs{
			Name:     pulumi.String(epName),
			Resource: db.Name,
			Ttl:      pulumi.String(spec.TTL),
		}
		if spec.ScriptOnlyAccess {
			args.ScriptOnlyAccess = pulumi.Bool(true)
		}
		ep, err := adaptive.NewEndpoint(ctx, "ep", args)
		if err != nil {
			return err
		}
		ctx.Export("id", ep.ID())
		return nil
	})
	id := harness.StringOutput(t, outs, "id")

	assert.Equal(t, epName, terraformRead(t, cfg, "session", id)["name"])

	// A scalar change and a toggle going from unset to set: an absent optional
	// and a changed one take different paths through the read reconciliation.
	spec.TTL = "12h"
	spec.ScriptOnlyAccess = true
	harness.Up(t, stack)
	harness.AssertRefreshClean(t, stack)

	assert.Equal(t, "12h", terraformRead(t, cfg, "session", id)["ttl"],
		"the updated TTL should be readable back")

	harness.Destroy(t, stack)
	assertTerraformGone(t, cfg, "session", id)
}

// Script's body is write-only, so an update has to be taken on trust: the server
// never echoes it back. All the more reason to exercise the path.
func TestScriptLifecycle(t *testing.T) {
	t.Parallel()
	cfg := harness.RequireProviderConfig(t)

	resName := uniqueName("pulumi-it-lc-scr-res")
	epName := uniqueName("pulumi-it-lc-scr-ep")
	scrName := uniqueName("pulumi-it-lc-scr")
	spec := struct{ Command, Description string }{
		Command:     "psql -c 'select 1'",
		Description: "integration-test script",
	}

	outs, stack := harness.DeployStack(t, cfg, stackName("lc-scr"), func(ctx *pulumi.Context) error {
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
		})
		if err != nil {
			return err
		}
		scr, err := adaptive.NewScript(ctx, "scr", &adaptive.ScriptArgs{
			Name:        pulumi.String(scrName),
			Endpoint:    ep.Name,
			Command:     pulumi.String(spec.Command),
			Description: pulumi.String(spec.Description),
		})
		if err != nil {
			return err
		}
		ctx.Export("id", scr.ID())
		return nil
	})
	id := harness.StringOutput(t, outs, "id")

	assert.Equal(t, spec.Description, terraformRead(t, cfg, "script", id)["description"])

	// The description is readable, so it is what the update is verified through;
	// the body is not, which is what the drift test covers instead.
	spec.Command = "psql -c 'select 2'"
	spec.Description = "integration-test script (updated)"
	harness.Up(t, stack)
	assert.Equal(t, spec.Description, terraformRead(t, cfg, "script", id)["description"],
		"the updated description should be readable back")

	harness.AssertRefreshClean(t, stack)
	harness.Destroy(t, stack)
	assertTerraformGone(t, cfg, "script", id)
}
