---
title: Adaptive Installation & Configuration
meta_desc: Information on how to install the Adaptive provider for Pulumi.
layout: package
---

## Installation

The Adaptive provider is available as a package in all Pulumi languages:

- JavaScript/TypeScript: [`@adaptive-scale/pulumi-adaptive`](https://www.npmjs.com/package/@adaptive-scale/pulumi-adaptive)
- Python: [`pulumi_adaptive`](https://pypi.org/project/pulumi-adaptive/)
- Go: [`github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive`](https://pkg.go.dev/github.com/adaptive-scale/pulumi-adaptive/sdk)
- .NET: [`AdaptiveScale.Adaptive`](https://www.nuget.org/packages/AdaptiveScale.Adaptive)

### Node.js (JavaScript/TypeScript)

```bash
npm install @adaptive-scale/pulumi-adaptive
```

### Python

```bash
pip install pulumi-adaptive
```

### Go

```bash
go get github.com/adaptive-scale/pulumi-adaptive/sdk
```

### .NET

```bash
dotnet add package AdaptiveScale.Adaptive
```

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
