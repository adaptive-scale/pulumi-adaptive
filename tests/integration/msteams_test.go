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

// TestMSTeamsWorkflow deploys an MS Teams workflow integration via the generic
// Resource type and verifies it appears in the Client App resources list with
// the msteams_workflow type. (The webhook URL is not returned by the API — see
// CLIENT_API_GAPS.md.)
func TestMSTeamsWorkflow(t *testing.T) {
	cfg := harness.RequireConfig(t)
	name := uniqueName("pulumi-it-msteams")

	outs := harness.Deploy(t, cfg, stackName("msteams"), func(ctx *pulumi.Context) error {
		wf, err := adaptive.NewResource(ctx, "wf", &adaptive.ResourceArgs{
			Name:       pulumi.String(name),
			Type:       pulumi.String("msteams_workflow"),
			WebhookUrl: pulumi.String("https://example.com/webhook"),
		})
		if err != nil {
			return err
		}
		ctx.Export("id", wf.ID())
		return nil
	})

	id := harness.StringOutput(t, outs, "id")

	vc := client.New(cfg.URL, cfg.ClientID, cfg.ClientSecret)
	got := retryFind(t, "msteams workflow "+id,
		vc.ListResources,
		func(r client.Resource) bool { return r.ID == id })

	assert.Equal(t, name, got.Name)
	assert.Equal(t, "msteams_workflow", got.Type)
}
