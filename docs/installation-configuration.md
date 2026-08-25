---
title: Adaptive Installation & Configuration
meta_desc: Information on how to install the Adaptive provider for Pulumi.
layout: package
---

## Installation

The plugin binary needs no installation: it is resolved from this repository's
GitHub releases through the schema's `pluginDownloadURL`
(`github://api.github.com/adaptive-scale/pulumi-adaptive`).

The language SDKs are **not published to any package registry** — there is no npm,
PyPI, or NuGet package. Install them from the repository instead.

Where a command needs a version, take it from the build rather than typing one, so
it cannot drift from what is actually released:

```bash
VERSION=$(make -s print-version)
```

### Go

Go resolves the version itself, from the `sdk/vX.Y.Z` tag the release publishes:

```bash
go get github.com/adaptive-scale/pulumi-adaptive/sdk@latest
```

### Python

From the `sdk/python` subdirectory of a release tag:

```bash
pip install "pulumi_adaptive @ git+https://github.com/adaptive-scale/pulumi-adaptive.git@v$VERSION#subdirectory=sdk/python"
```

### Node.js (JavaScript/TypeScript)

No published package, so build the SDK and install it from the local path:

```bash
make build_nodejs
npm install /path/to/pulumi-adaptive/sdk/nodejs/bin
```

### .NET

Likewise — `make build_dotnet` writes a `.nupkg` under `sdk/dotnet/bin`, which can
be added as a local NuGet source.

### Developing against an unreleased provider

`make install` builds the plugin into `$(go env GOPATH)/bin` at the Makefile's
version. Pulumi prefers an ambient plugin on `PATH` over a downloaded one and
prints a warning naming the path it used — if that warning is absent, you are
running a downloaded build rather than yours.

## Configuration

The provider authenticates against your Adaptive workspace with a service
account token. Generate one in the Adaptive console (Settings → Service
Accounts), then put it in the environment:

```bash
export ADAPTIVE_SVC_TOKEN=<your-service-token>
export ADAPTIVE_URL=https://app.adaptive.live   # optional
```

| Env var | Description |
|---|---|
| `ADAPTIVE_SVC_TOKEN` | Service account token. If unset, the provider falls back to the adaptive-cli token at `~/.adaptive/token`. |
| `ADAPTIVE_URL` | The Adaptive workspace URL. Defaults to `https://app.adaptive.live`. |

`ADAPTIVE_SVC_TOKEN` accepts a raw token or the JSON that `adaptive login`
writes, in either the `{token,url}` or the `{deployments:{...}}` shape. In the
latter, the deployment marked `default` is used; a single deployment needs no
marker, and several without one are refused rather than picked arbitrarily — that
would send credentials to whichever environment happened to come up first.

### Why there is no `pulumi config`

Pulumi persists provider configuration on the provider resource inside the stack.
A `adaptive:serviceToken` config value therefore ended up in state, and because
it was marked secret the stack could not be read or exported without its
passphrase or KMS key. Rotating the token also diffed the provider resource,
which the engine can cascade into replacing every resource it manages.

The provider now declares no configuration at all, which removes the cause rather
than working around it. Two things follow:

- **One process, one workspace.** There is no per-stack workspace override,
  because there is no config to hold one. Set `ADAPTIVE_URL` per invocation.
- **No explicit provider instances.** The SDKs no longer generate a `Provider`
  type, so a program cannot construct one.

### Upgrading from a version with config

Nothing breaks and no resource is replaced — the provider no longer implements
`DiffConfig`, and the engine treats that as "do not replace". A leftover
`adaptive:serviceToken` is ignored rather than rejected, so clean it up
explicitly:

```bash
pulumi config rm adaptive:serviceToken
pulumi config rm adaptive:workspaceUrl
```

The token already recorded in an existing stack's state is not scrubbed by the
upgrade itself. If `pulumi stack export | grep serviceToken` still finds it and
that matters to you, remove the provider resource from state
(`pulumi state delete` on the `pulumi:providers:adaptive` URN) and let the next
`pulumi up` register a fresh one.
