# Pulumi Provider for Adaptive

A native [Pulumi](https://www.pulumi.com/) provider for [Adaptive](https://adaptive.live),
the access-management platform. It lets you manage Adaptive infrastructure —
resource connections, access endpoints, authorizations, groups, and scripts —
as code, the same surface offered by the
[Adaptive Terraform provider](https://registry.terraform.io/providers/adaptive-scale/adaptive),
but for Pulumi.

It is built with [`pulumi-go-provider`](https://github.com/pulumi/pulumi-go-provider)
and talks directly to the Adaptive REST API — there is no dependency on
Terraform or the Terraform bridge.

## Resources

| Pulumi resource | Description |
|---|---|
| `adaptive.Resource` | A connection to an external service (database, cloud, Kubernetes, …). The `type` field selects the integration; ~50 are supported. |
| `adaptive.Endpoint` | A secure, time-limited access point to a resource (TTL, JIT approval, users/groups). |
| `adaptive.Authorization` | A permission policy for a resource type. |
| `adaptive.Group` | A bundle of users and endpoints. |
| `adaptive.Script` | A command attached to an endpoint. |
| `adaptive.MSTeamsWorkflow` | An MS Teams workflow webhook integration. |

## Configuration

| Setting | Env var | Default |
|---|---|---|
| `serviceToken` | `ADAPTIVE_SVC_TOKEN` | falls back to `~/.adaptive/token` |
| `workspaceUrl` | `ADAPTIVE_URL` | `https://app.adaptive.live` |

## Repository layout

```
provider/            The provider plugin (its own Go module)
  *.go                client, resources, integration config builders
  cmd/
    pulumi-resource-adaptive/   the plugin binary
    gen/                        in-process schema + SDK generator (no Pulumi CLI needed)
sdk/go/adaptive/     The generated Go SDK (its own Go module)
schema.json          The generated Pulumi schema
examples/go/         An example Pulumi (Go) program
Makefile             build / gen / install helpers
```

## Building

```bash
make build      # compile the provider plugin into provider/bin/
make gen        # regenerate schema.json and the Go SDK
make build_sdk  # compile the generated Go SDK
make test       # sanity-check that everything compiles
```

The schema and SDK are generated **in-process** by `provider/cmd/gen` using
`pulumi/pkg` codegen, so generation does not require the Pulumi CLI.

## Using it

The provider plugin must be discoverable by Pulumi. Install it onto your `PATH`:

```bash
make install    # builds pulumi-resource-adaptive into $(go env GOPATH)/bin
```

Then, from a Pulumi program (see `examples/go`):

```bash
export ADAPTIVE_SVC_TOKEN="your-service-token"
export ADAPTIVE_URL="https://app.adaptive.live"   # or your workspace
pulumi stack init dev
pulumi up
```

```go
db, _ := adaptive.NewResource(ctx, "my-db", &adaptive.ResourceArgs{
    Name:     pulumi.String("playground-postgres"),
    Type:     pulumi.String("postgres"),
    Host:     pulumi.String("db.example.com"),
    Port:     pulumi.String("5432"),
    Username: pulumi.String("admin"),
    Password: pulumi.String("secret"),
    SslMode:  pulumi.String("require"),
})
```

> **Authorization permissions:** for SQL resource types (postgres, mysql,
> sqlserver, …) and kubernetes, `permissions` must be structured YAML — a bare
> value like `"SELECT"` is rejected by the API. See `examples/go/main.go`.

## Status

Early/experimental. The provider compiles, emits a valid schema, and the
generated Go SDK builds and is consumable (see `examples/go`). SDKs for other
languages (TypeScript, Python, .NET, Java) can be generated from `schema.json`.
