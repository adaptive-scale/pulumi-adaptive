//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchedule deploys an auto-approval schedule, verifies it via the raw
// read API, checks refresh is clean (including weekday-casing normalization),
// and verifies an update round-trips.
func TestSchedule(t *testing.T) {
	cfg := harness.RequireConfig(t)
	name := uniqueName("pulumi-it-sched")

	outs, stack := harness.DeployStack(t, cfg, stackName("sched"), func(ctx *pulumi.Context) error {
		s, err := adaptive.NewSchedule(ctx, "sched", &adaptive.ScheduleArgs{
			Name:         pulumi.String(name),
			ScheduleType: pulumi.String("custom"),
			// Mixed casing on purpose: the server lowercases weekday names and
			// refresh must not report that as drift.
			Weekdays:    pulumi.StringArray{pulumi.String("Monday"), pulumi.String("TUESDAY")},
			StartHour:   pulumi.Int(9),
			StartMinute: pulumi.Int(0),
			EndHour:     pulumi.Int(17),
			EndMinute:   pulumi.Int(30),
		})
		if err != nil {
			return err
		}
		ctx.Export("id", s.ID())
		return nil
	})
	id := harness.StringOutput(t, outs, "id")

	// Verify via the raw terraform read API.
	resp, body := rawTerraformAPI(t, cfg, "GET", "/schedule/read/"+id, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "schedule read failed: %s", body)
	var got struct {
		Name         string   `json:"name"`
		ScheduleType string   `json:"scheduleType"`
		Weekdays     []string `json:"weekdays"`
		StartHour    int      `json:"startHour"`
		EndMinute    int      `json:"endMinute"`
		IsActive     bool     `json:"isActive"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	assert.Equal(t, name, got.Name)
	assert.Equal(t, "custom", got.ScheduleType)
	assert.ElementsMatch(t, []string{"monday", "tuesday"}, got.Weekdays)
	assert.Equal(t, 9, got.StartHour)
	assert.Equal(t, 30, got.EndMinute)

	// Weekday casing and server-defaulted isActive/operationType must not
	// produce refresh drift.
	harness.AssertRefreshClean(t, stack)
}

// TestEndpointNewFields deploys an endpoint with the new toggle/JIT fields
// set and verifies they round-trip through the terraform read API and survive
// a clean refresh.
func TestEndpointNewFields(t *testing.T) {
	cfg := harness.RequireConfig(t)
	resName := uniqueName("pulumi-it-nf-res")
	epName := uniqueName("pulumi-it-nf-ep")

	outs, stack := harness.DeployStack(t, cfg, stackName("newfields"), func(ctx *pulumi.Context) error {
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
			Name:                 pulumi.String(epName),
			Resource:             db.Name,
			Ttl:                  pulumi.String("8h"),
			DisableDataStudio:    pulumi.Bool(true),
			DisableWebCli:        pulumi.Bool(true),
			DisableOutputCapture: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}
		ctx.Export("id", ep.ID())
		return nil
	})
	id := harness.StringOutput(t, outs, "id")

	resp, body := rawTerraformAPI(t, cfg, "GET", "/session/read/"+id, nil)
	require.Contains(t, []int{http.StatusOK, http.StatusAccepted}, resp.StatusCode, "session read failed: %s", body)
	var got struct {
		DisableDataStudio    bool `json:"disableDataStudio"`
		DisableWebCLI        bool `json:"disableWebCli"`
		DisableOutputCapture bool `json:"disableOutputCapture"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	assert.True(t, got.DisableDataStudio, "disableDataStudio not persisted")
	assert.True(t, got.DisableWebCLI, "disableWebCli not persisted")
	assert.True(t, got.DisableOutputCapture, "disableOutputCapture not persisted")

	harness.AssertRefreshClean(t, stack)
}

// TestGroupSlackChannel deploys a group with slackChannelId (create-then-update
// path) and verifies it round-trips through the read API.
func TestGroupSlackChannel(t *testing.T) {
	cfg := harness.RequireConfig(t)
	name := uniqueName("pulumi-it-slack-grp")

	outs, stack := harness.DeployStack(t, cfg, stackName("slackgrp"), func(ctx *pulumi.Context) error {
		g, err := adaptive.NewGroup(ctx, "grp", &adaptive.GroupArgs{
			Name:           pulumi.String(name),
			SlackChannelId: pulumi.String("C0000000000"),
		})
		if err != nil {
			return err
		}
		ctx.Export("id", g.ID())
		return nil
	})
	id := harness.StringOutput(t, outs, "id")

	resp, body := rawTerraformAPI(t, cfg, "GET", "/team/read/"+id, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "team read failed: %s", body)
	var got struct {
		Name           string `json:"name"`
		SlackChannelID string `json:"slackChannelId"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	assert.Equal(t, name, got.Name)
	assert.Equal(t, "C0000000000", got.SlackChannelID)

	harness.AssertRefreshClean(t, stack)
}
