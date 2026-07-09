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
Accounts), then configure the provider:

```bash
pulumi config set adaptive:serviceToken --secret <your-service-token>
pulumi config set adaptive:workspaceUrl https://app.adaptive.live
```

The following configuration points are available:

| Setting | Env var | Description |
|---|---|---|
| `adaptive:serviceToken` | `ADAPTIVE_SVC_TOKEN` | Service account token. If unset, the provider falls back to the adaptive-cli token at `~/.adaptive/token`. Treat as a secret. |
| `adaptive:workspaceUrl` | `ADAPTIVE_URL` | The Adaptive workspace URL. Defaults to `https://app.adaptive.live`. |

Environment variables work as an alternative to `pulumi config`:

```bash
export ADAPTIVE_SVC_TOKEN=<your-service-token>
export ADAPTIVE_URL=https://app.adaptive.live
```
