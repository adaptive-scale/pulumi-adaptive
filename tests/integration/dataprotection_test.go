//go:build integration

package integration

import (
	"testing"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/adaptive-scale/pulumi-adaptive/tests/integration/internal/harness"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DataProtection had no integration coverage at all, despite being the resource
// with the least conventional lifecycle: destroy turns masking *off* rather than
// deleting anything, and leaves the generated masked_<resource> authorization
// behind. That is the platform's documented behaviour, so the teardown assertion
// has to check for "masking disabled", not "object gone".
func TestDataProtectionLifecycle(t *testing.T) {
	t.Parallel()
	cfg := harness.RequireProviderConfig(t)

	resName := uniqueName("pulumi-it-lc-dp-res")
	scoped := true

	outs, stack := harness.DeployStack(t, cfg, stackName("lc-dp"), func(ctx *pulumi.Context) error {
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
		policy, err := adaptive.NewDataProtection(ctx, "mask", &adaptive.DataProtectionArgs{
			Resource: db.Name,
			Scoped:   pulumi.Bool(scoped),
			Masks: adaptive.DataProtectionMaskArray{
				adaptive.DataProtectionMaskArgs{
					DatabaseName: pulumi.String("shop"),
					Tables: adaptive.DataProtectionTableArray{
						adaptive.DataProtectionTableArgs{
							TableName: pulumi.String("users"),
							Schema:    pulumi.String("public"),
							MaskedColumns: adaptive.DataProtectionColumnMaskArray{
								adaptive.DataProtectionColumnMaskArgs{
									ColumnName:  pulumi.String("email"),
									MaskingType: pulumi.String("email"),
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("policyId", policy.ID())
		ctx.Export("authorizationName", policy.AuthorizationName)
		return nil
	})

	policyID := harness.StringOutput(t, outs, "policyId")
	require.NotEmpty(t, policyID)

	// The generated authorization is what an Endpoint attaches to in order to
	// serve masked sessions, so an empty one means the policy is unusable even
	// though the resource "succeeded".
	authName := harness.StringOutput(t, outs, "authorizationName")
	assert.Contains(t, authName, resName,
		"the generated authorization should be named after the protected resource")

	harness.AssertRefreshClean(t, stack)

	// Toggle scoped: the one field that can be updated in place, and one whose
	// server-side default (true) makes an unset value indistinguishable from a
	// set one — so refresh must stay clean after flipping it.
	scoped = false
	harness.Up(t, stack)
	harness.AssertRefreshClean(t, stack)

	harness.Destroy(t, stack)
}
