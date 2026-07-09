# Pulumi Provider for Adaptive

[![CI](https://github.com/adaptive-scale/pulumi-adaptive/actions/workflows/ci.yml/badge.svg)](https://github.com/adaptive-scale/pulumi-adaptive/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

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

## Installation

| Language | Package |
|---|---|
| TypeScript/JavaScript | `npm install @adaptive-scale/pulumi-adaptive` |
| Python | `pip install pulumi-adaptive` |
| Go | `go get github.com/adaptive-scale/pulumi-adaptive/sdk` |
| .NET | `dotnet add package AdaptiveScale.Adaptive` |

The plugin binary is downloaded automatically from this repo's GitHub releases
(the schema's `pluginDownloadURL` points at
`github://api.github.com/adaptive-scale/pulumi-adaptive`).

## Configuration

Generate a service account token in the Adaptive console, then either:

```bash
pulumi config set adaptive:serviceToken --secret <your-service-token>
```

or set the environment variables:

| Setting | Env var | Default |
|---|---|---|
| `serviceToken` | `ADAPTIVE_SVC_TOKEN` | falls back to `~/.adaptive/token` |
| `workspaceUrl` | `ADAPTIVE_URL` | `https://app.adaptive.live` |

## Usage

```typescript
import * as adaptive from "@adaptive-scale/pulumi-adaptive";

// A connection to a Postgres database.
const db = new adaptive.Resource("my-db", {
    name: "playground-postgres",
    type: "postgres",
    host: "db.example.com",
    port: "5432",
    username: "admin",
    password: "not-a-real-password",
    sslMode: "require",
});

// A time-limited access endpoint to it.
const endpoint = new adaptive.Endpoint("my-db-access", {
    name: "playground-db-access",
    resource: db.name,
    ttl: "8h",
    users: ["you@example.com"],
});
```

The same program — plus an authorization and a group — lives in each language
under [`examples/`](examples): [`go`](examples/go), [`nodejs`](examples/nodejs),
[`python`](examples/python). Run one with:

```bash
export ADAPTIVE_SVC_TOKEN="your-service-token"
export ADAPTIVE_URL="https://app.adaptive.live"   # or your workspace
pulumi stack init dev
pulumi up
```

> **Authorization permissions:** for SQL resource types (postgres, mysql,
> sqlserver, …) and kubernetes, `permissions` must be structured YAML — a bare
> value like `"SELECT"` is rejected by the API. See `examples/go/main.go`.

## Development

### Repository layout

```
provider/            The provider plugin (its own Go module)
  *.go                client, resources, integration config builders
  cmd/
    pulumi-resource-adaptive/   the plugin binary
    gen/                        in-process schema + SDK generator (no Pulumi CLI needed)
sdk/                 The generated SDKs (go/ is its own Go module; nodejs/, python/, dotnet/)
schema.json          The generated Pulumi schema
docs/                Pulumi Registry docs (_index.md, installation-configuration.md)
examples/            Example Pulumi programs (go/, nodejs/, python/)
Makefile             build / gen / install helpers
.goreleaser.yml      Plugin binary release configuration
```

### Building

```bash
make build       # compile the provider plugin into provider/bin/
make gen         # regenerate schema.json and the Go/Node.js/Python SDKs
make gen_dotnet  # regenerate the .NET SDK (requires the Pulumi CLI)
make build_sdks  # build every language SDK
make test        # sanity-check that everything compiles
```

The schema and the Go/Node.js/Python SDKs are generated **in-process** by
`provider/cmd/gen` using `pulumi/pkg` codegen, so generation does not require
the Pulumi CLI. The .NET SDK goes through `pulumi package gen-sdk` because its
codegen lives outside `pulumi/pkg`.

To develop against an unreleased provider, install the plugin onto your `PATH`
where Pulumi discovers it as an ambient plugin:

```bash
make install    # builds pulumi-resource-adaptive into $(go env GOPATH)/bin
```

### Releasing

Push a `vX.Y.Z` tag. The release workflow then:

1. builds the plugin binaries for every OS/arch with GoReleaser and attaches
   them to a GitHub release (`pulumi-resource-adaptive-vX.Y.Z-<os>-<arch>.tar.gz`),
2. regenerates the SDKs at the release version and publishes them to npm,
   PyPI, and NuGet via `pulumi/pulumi-package-publisher`,
3. tags `sdk/vX.Y.Z` so the Go SDK resolves.

It needs the `NPM_TOKEN`, `PYPI_API_TOKEN`, and `NUGET_PUBLISH_KEY` repo secrets.

## Status

Early/experimental. The provider compiles, emits a valid schema, and the
generated SDKs (Go, TypeScript, Python, .NET) build; the Go SDK is exercised
end-to-end by the integration tests (see `examples/go` and `tests/integration`).
