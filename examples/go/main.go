// Example Pulumi program (Go) using the Adaptive provider. This mirrors the
// Terraform playground: a Postgres resource, an endpoint that references it,
// an authorization, and a group.
//
// Run with:
//
//	export ADAPTIVE_SVC_TOKEN=...        # or ADAPTIVE_URL for a custom workspace
//	pulumi stack init dev
//	pulumi up
package main

import (
	"github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// A connection to a Postgres database.
		db, err := adaptive.NewResource(ctx, "my-db", &adaptive.ResourceArgs{
			Name:     pulumi.String("playground-postgres"),
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

		// A time-limited access endpoint. Referencing db.Name makes Pulumi
		// create the resource before the endpoint (the dependency graph).
		endpoint, err := adaptive.NewEndpoint(ctx, "my-db-access", &adaptive.EndpointArgs{
			Name:     pulumi.String("playground-db-access"),
			Resource: db.Name,
			Ttl:      pulumi.String("8h"),
			Users:    pulumi.StringArray{pulumi.String("you@example.com")},
		})
		if err != nil {
			return err
		}

		// A read-only permission policy. For postgres, permissions must be
		// structured YAML (a bare "SELECT" is rejected server-side).
		_, err = adaptive.NewAuthorization(ctx, "read-only", &adaptive.AuthorizationArgs{
			Name:         pulumi.String("playground-readonly"),
			ResourceType: pulumi.String("postgres"),
			Description:  pulumi.String("Read-only access for the playground"),
			Permissions: pulumi.String(`allow:
  - database: production
    privileges:
      - SELECT
    objects:
      - ALL
`),
		})
		if err != nil {
			return err
		}

		// A group bundling users and the endpoint above.
		_, err = adaptive.NewGroup(ctx, "developers", &adaptive.GroupArgs{
			Name:      pulumi.String("playground-developers"),
			Members:   pulumi.StringArray{pulumi.String("you@example.com")},
			Endpoints: pulumi.StringArray{endpoint.Name},
		})
		if err != nil {
			return err
		}

		ctx.Export("endpointName", endpoint.Name)
		return nil
	})
}
