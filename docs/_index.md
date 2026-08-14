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
  every integration type the platform registers (~82) is supported.
- `adaptive.Endpoint` — a secure, time-limited access point to a resource
  (TTL, just-in-time approval, users/groups).
- `adaptive.Authorization` — a permission policy for a resource type.
- `adaptive.Group` — a bundle of users and endpoints.
- `adaptive.Script` — a command attached to an endpoint.
- `adaptive.Schedule` — an auto-approval schedule window for users, groups, and endpoints.
- `adaptive.DataProtection` — a data-masking policy for a resource.

Full documentation, including a field reference for every resource, is at
[documentation.adaptive.live/developer-guide/pulumi](https://documentation.adaptive.live/developer-guide/pulumi).

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
