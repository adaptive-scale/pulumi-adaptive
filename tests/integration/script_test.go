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

// TestScript deploys a Resource + Endpoint + Script and verifies the script
// appears in the Client App scripts list. The API does not return the script
// command/body, so verification is by name (see CLIENT_API_GAPS.md).
func TestScript(t *testing.T) {
	cfg := harness.RequireConfig(t)
	resName := uniqueName("pulumi-it-res")
	epName := uniqueName("pulumi-it-ep")
	scName := uniqueName("pulumi-it-script")

	harness.Deploy(t, cfg, stackName("script"), func(ctx *pulumi.Context) error {
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
		_, err = adaptive.NewScript(ctx, "sc", &adaptive.ScriptArgs{
			Name:     pulumi.String(scName),
			Command:  pulumi.String("echo hello from integration test"),
			Endpoint: ep.Name,
		})
		return err
	})

	vc := client.New(cfg.URL, cfg.ClientID, cfg.ClientSecret)
	got := retryFind(t, "script "+scName,
		vc.ListScripts,
		func(s client.Script) bool { return s.Name == scName })

	assert.Equal(t, scName, got.Name)
}
