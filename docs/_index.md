---
title: Adaptive
meta_desc: Provides an overview of the Adaptive Provider for Pulumi.
layout: package
---

The Adaptive provider for Pulumi lets you manage [Adaptive](https://adaptive.live)
infrastructure — resource connections, access endpoints, authorizations, groups,
and scripts — as code, in any Pulumi language.

Adaptive is an access-management platform: it brokers secure, audited,
time-limited access to databases, cloud accounts, Kubernetes clusters, servers,
and other infrastructure. This provider manages that surface declaratively:

- `adaptive.Resource` — a connection to an external service (database, cloud
  account, Kubernetes cluster, …). The `type` field selects the integration;
  around 50 are supported.
- `adaptive.Endpoint` — a secure, time-limited access point to a resource
  (TTL, just-in-time approval, users/groups).
- `adaptive.Authorization` — a permission policy for a resource type.
- `adaptive.Group` — a bundle of users and endpoints.
- `adaptive.Script` — a command attached to an endpoint.
- `adaptive.MSTeamsWorkflow` — an MS Teams workflow webhook integration.

## Example

{{< chooser language "typescript,python,go,csharp" >}}

{{% choosable language typescript %}}

```typescript
import * as adaptive from "@adaptive-scale/pulumi-adaptive";

const db = new adaptive.Resource("my-db", {
    name: "staging-postgres",
    type: "postgres",
    host: "db.example.internal",
    port: "5432",
    username: "adaptive",
    password: "example-password",
    sslMode: "require",
});

const endpoint = new adaptive.Endpoint("my-db-access", {
    name: "staging-db-access",
    resource: db.name,
    ttl: "8h",
    users: ["dev@example.com"],
});
```

{{% /choosable %}}

{{% choosable language python %}}

```python
import pulumi_adaptive as adaptive

db = adaptive.Resource("my-db",
    name="staging-postgres",
    type="postgres",
    host="db.example.internal",
    port="5432",
    username="adaptive",
    password="example-password",
    ssl_mode="require")

endpoint = adaptive.Endpoint("my-db-access",
    name="staging-db-access",
    resource=db.name,
    ttl="8h",
    users=["dev@example.com"])
```

{{% /choosable %}}

{{% choosable language go %}}

```go
package main

import (
	"github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		db, err := adaptive.NewResource(ctx, "my-db", &adaptive.ResourceArgs{
			Name:     pulumi.String("staging-postgres"),
			Type:     pulumi.String("postgres"),
			Host:     pulumi.String("db.example.internal"),
			Port:     pulumi.String("5432"),
			Username: pulumi.String("adaptive"),
			Password: pulumi.String("example-password"),
			SslMode:  pulumi.String("require"),
		})
		if err != nil {
			return err
		}

		_, err = adaptive.NewEndpoint(ctx, "my-db-access", &adaptive.EndpointArgs{
			Name:     pulumi.String("staging-db-access"),
			Resource: db.Name,
			Ttl:      pulumi.String("8h"),
			Users:    pulumi.StringArray{pulumi.String("dev@example.com")},
		})
		return err
	})
}
```

{{% /choosable %}}

{{% choosable language csharp %}}

```csharp
using Pulumi;
using Adaptive = AdaptiveScale.Adaptive;

return Deployment.RunAsync(() =>
{
    var db = new Adaptive.Resource("my-db", new Adaptive.ResourceArgs
    {
        Name = "staging-postgres",
        Type = "postgres",
        Host = "db.example.internal",
        Port = "5432",
        Username = "adaptive",
        Password = "example-password",
        SslMode = "require",
    });

    var endpoint = new Adaptive.Endpoint("my-db-access", new Adaptive.EndpointArgs
    {
        Name = "staging-db-access",
        Resource = db.Name,
        Ttl = "8h",
        Users = new[] { "dev@example.com" },
    });
});
```

{{% /choosable %}}

{{< /chooser >}}

## Postgres with AWS RDS IAM authentication

Set `useRdsIam` to authenticate with short-lived AWS RDS IAM auth tokens instead
of a stored password, so no `password` is needed. The database user must have
been granted `rds_iam`, the RDS instance must have IAM authentication enabled,
and `sslMode` cannot be `disable`.

`useIrsa` selects the identity that mints the tokens. With it on, tokens are
minted through `awsServiceAccount` — a Kubernetes ServiceAccount in the session
cluster annotated with `eks.amazonaws.com/role-arn` — or, when that is empty,
through the platform's own IRSA role or instance profile. With it off,
`awsAccessKeyId` / `awsSecretAccessKey` are used instead. Leaving `useIrsa`
unset infers it from whether static keys were given.

`awsRegion` is only needed when the region cannot be derived from an
`*.rds.amazonaws.com` hostname, and `awsRoleArn` is an optional role to assume.

```typescript
const rdsDb = new adaptive.Resource("rds-db", {
    name: "production-rds",
    type: "postgres",
    host: "mydb.abc123.us-east-1.rds.amazonaws.com",
    port: "5432",
    username: "iam_user",
    sslMode: "require",
    useRdsIam: true,
    useIrsa: true,
    awsServiceAccount: "adaptive-rds-access",
});
```
