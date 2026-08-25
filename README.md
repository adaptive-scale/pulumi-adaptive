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
| `adaptive.Resource` | A connection to an external service (database, cloud, Kubernetes, …). The `type` field selects the integration; every integration type the Adaptive platform registers (~82) is supported. |
| `adaptive.Endpoint` | A secure, time-limited access point to a resource (TTL, JIT approval and modes, users/groups, web-access toggles, output capture). |
| `adaptive.Authorization` | A permission policy for a resource type. |
| `adaptive.Group` | A bundle of users and endpoints, optionally linked to a Slack channel. |
| `adaptive.Script` | A command attached to an endpoint, with optional description and parameter docs. |
| `adaptive.Schedule` | An auto-approval (or auto-reject) schedule window applied to users, groups, and endpoints. |
| `adaptive.DataProtection` | A data-masking policy for a resource (per-database/table/column masking rules). |

Notification/webhook integrations (e.g. MS Teams workflows) are managed through
`adaptive.Resource` with the matching `type` (e.g. `msteams_workflow` + `webhookUrl`).

### Data protection (masking)

```typescript
const policy = new adaptive.DataProtection("mask-pii", {
    resource: db.name,
    masks: [{
        databaseName: "shop",
        tables: [{
            tableName: "users",
            schema: "public",
            maskedColumns: [
                { columnName: "email", maskingType: "email" },
                { columnName: "ssn", maskingType: "ssn" },
            ],
        }],
    }],
});

// Serve masked sessions by attaching the generated authorization:
const masked = new adaptive.Endpoint("masked-access", {
    name: "shop-masked",
    resource: db.name,
    authorization: policy.authorizationName,
});
```

Masking types: `name`, `email`, `ssn`, `phone`, `cc`, `aadhar`, `first4`,
`last4`, `redact`, `hash`, and `disable` (omits the column entirely). Caveats:
creating a policy for a resource that already has one (e.g. configured in the
UI) adopts and overwrites it; `pulumi destroy` turns masking **off** (the
platform's documented behavior — there is no hard delete), leaving the
`masked_<resource>` authorization and any already-provisioned masked views in
place; masked views are created lazily when a session starts.

All resources support `pulumi refresh` (drift detection, including out-of-band
deletion) and `pulumi import` (see [Importing existing objects](#importing-existing-objects)).

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

## Secrets

Credential-bearing fields (`password`, `secretAccessKey`, `clientSecret`, SSH
keys, API tokens, webhook URLs, connection-string `uri`s, script `command`s, …)
are marked secret in the provider schema. The engine therefore shows them as
`[secret]` in `pulumi preview`/`up` output and encrypts them in the stack
state — **even when the program passes a plaintext literal**, in every
language.

To keep secrets out of source code, put them in stack config:

```bash
pulumi config set --secret dbPassword 'S3cret!'
```

```python
cfg = pulumi.Config()
db = adaptive.Resource("db",
    name="prod-postgres", type="postgres",
    host="...", username="app",
    password=cfg.require_secret("dbPassword"))
```

**Secret drift detection.** The Adaptive API never returns secret values, but
it returns an opaque server-side fingerprint per secret field
(`appliedDigests` on resources, `commandDigest` on scripts). The provider
records the fingerprints as of its last write and compares them on
`pulumi refresh`, so a secret rotated out-of-band (e.g. in the UI) shows up in
the next preview on the argument it belongs to — as `[secret]`, never in
plaintext. The argument name is what carries the information.

What the preview then does depends on your program, because the platform stores
the configuration as a single blob and every update replaces it whole:

- the argument is in your program — the update re-applies your value.
- it is not — the update clears it, because your program is the source of truth
  for the whole resource. That has always been what an update does to an
  argument you do not set; it is now visible beforehand instead of happening
  silently on whatever unrelated `up` ran first.

A secret *added* out-of-band is reported the same way. Notes:

- Requires an Adaptive backend with digest support; against older servers,
  refresh keeps the prior value silently (previous behavior).
- Nothing is reported on the first refresh after an import or a provider
  upgrade: with no recorded fingerprints there is no baseline, so they are
  recorded and compared from the next refresh on.
- Fingerprints are HMACs under a server-held key (`ADAPTIVE_SECRET_DIGEST_KEY`)
  — they cannot be reversed or brute-forced, and rotating that key causes one
  self-healing drift cycle.
- `pulumi import` cannot recover secret values: set them in config, and the
  first `pulumi up` re-establishes them; drift detection is active from then on.
- A secret nested inside a structured field (the `adaptive_rdp` target list) is
  reported as a warning naming the field rather than as a preview diff — there
  is no single argument to attach it to.

## Importing existing objects

Every resource type imports by its Adaptive object ID:

```bash
pulumi import adaptive:index:Resource     my-db      <resource-id>
pulumi import adaptive:index:Endpoint     my-access  <endpoint-id>
pulumi import adaptive:index:Authorization my-auth   <authorization-id>
pulumi import adaptive:index:Group        my-group   <group-id>
pulumi import adaptive:index:Script       my-script  <script-id>
pulumi import adaptive:index:Schedule     my-window  <schedule-id>
pulumi import adaptive:index:DataProtection my-policy <resource-id>
```

(`DataProtection` imports by the protected **resource's** ID — a policy has no
separate identity of its own.)

Caveats — some values are write-only in the Adaptive API and cannot be
recovered on import:

- **Script `command`**: script bodies are never returned by the API. After
  importing a script, set `command` in your program; the first `pulumi up`
  rewrites it (refresh never touches it).
- **Resource secrets**: secret configuration values (passwords, keys, tokens)
  are stripped from API reads. The import warns with the exact list of
  redacted fields to fill in. On refresh, secrets already in state are kept.
- **Webhook URLs** (e.g. `msteams_workflow` resources): redacted like other secrets.
- **Schedules are upserted by name**: creating a schedule whose name already
  exists on the backend adopts the existing schedule instead of failing.

Refresh behavior: optional inputs you never set are not overwritten with
server-computed defaults (memory/cpu/cluster/idle timeout), so refresh stays
clean; out-of-band deletion drops the resource from state; against Adaptive
servers older than the accompanying backend change, refreshing a deleted
endpoint/authorization fails loudly instead of pruning it.

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
