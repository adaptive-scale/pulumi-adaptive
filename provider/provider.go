// Package adaptive implements a native Pulumi provider for Adaptive
// (https://adaptive.live), built with pulumi-go-provider. It mirrors the
// resource surface of the Adaptive Terraform provider: resources, endpoints,
// authorizations, groups, scripts, and schedules.
package adaptive

import (
	"context"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

// Version is the provider version, overridable at build time via -ldflags.
var Version = "0.2.0"

// Config holds provider-level configuration. Values fall back to the
// ADAPTIVE_SVC_TOKEN and ADAPTIVE_URL environment variables, matching the
// Terraform provider.
type Config struct {
	ServiceToken string `pulumi:"serviceToken,optional" provider:"secret"`
	WorkspaceURL string `pulumi:"workspaceUrl,optional"`
}

func (c *Config) Annotate(a infer.Annotator) {
	a.Describe(&c.ServiceToken, "Service account token for authenticating with the Adaptive service. "+
		"If not provided, the provider reads the token from the default adaptive-cli location (~/.adaptive/token).")
	a.SetDefault(&c.ServiceToken, "", "ADAPTIVE_SVC_TOKEN")
	a.Describe(&c.WorkspaceURL, "The Adaptive workspace URL. Defaults to https://app.adaptive.live.")
	a.SetDefault(&c.WorkspaceURL, "https://app.adaptive.live", "ADAPTIVE_URL")
}

// clientFromConfig builds an Adaptive API client from provider configuration.
func clientFromConfig(ctx context.Context) (*Client, error) {
	cfg := infer.GetConfig[Config](ctx)
	token, url, err := resolveToken(cfg.ServiceToken, cfg.WorkspaceURL)
	if err != nil {
		return nil, err
	}
	return NewClient(token, url), nil
}

// Provider builds the inferred Pulumi provider.
func Provider() (p.Provider, error) {
	prov, err := buildProvider()
	if err != nil {
		return prov, err
	}

	// Provider configuration is credentials plus the workspace URL. Changing
	// either must reconfigure the provider in place — never replace the
	// resources it manages. The infer layer reports every config change as a
	// replacement (pulumi-go-provider#409), which the engine cascades into a
	// full delete/recreate of the whole stack whenever the service token
	// rotates or the provider version recorded in state differs from the SDK.
	inner := prov.DiffConfig
	prov.DiffConfig = func(ctx context.Context, req p.DiffRequest) (p.DiffResponse, error) {
		resp, err := inner(ctx, req)
		if err != nil {
			return resp, err
		}
		resp.DeleteBeforeReplace = false
		for key, d := range resp.DetailedDiff {
			switch d.Kind {
			case p.AddReplace:
				d.Kind = p.Add
			case p.DeleteReplace:
				d.Kind = p.Delete
			case p.UpdateReplace:
				d.Kind = p.Update
			}
			resp.DetailedDiff[key] = d
		}
		return resp, nil
	}
	return prov, nil
}

func buildProvider() (p.Provider, error) {
	return infer.NewProviderBuilder().
		WithNamespace("adaptive-scale").
		WithDisplayName("Adaptive").
		WithDescription("A Pulumi provider for managing Adaptive (adaptive.live) infrastructure: "+
			"resources, endpoints, authorizations, groups, and scripts.").
		WithHomepage("https://adaptive.live").
		WithRepository("https://github.com/adaptive-scale/pulumi-adaptive").
		WithLicense("Apache-2.0").
		WithPublisher("Adaptive Scale").
		WithLogoURL("https://raw.githubusercontent.com/adaptive-scale/pulumi-adaptive/main/assets/logo.png").
		WithPluginDownloadURL("github://api.github.com/adaptive-scale/pulumi-adaptive").
		WithKeywords(
			"pulumi", "adaptive", "access-management", "security",
			"category/infrastructure", "kind/native",
		).
		// Replaces the builder's default language map, so each language must
		// restate the defaults it needs alongside our package-name overrides.
		WithLanguageMap(map[string]any{
			"nodejs": map[string]any{
				"packageName":          "@adaptive-scale/pulumi-adaptive",
				"respectSchemaVersion": true,
			},
			"go": map[string]any{
				"generateResourceContainerTypes": true,
				"respectSchemaVersion":           true,
			},
			"python": map[string]any{
				"packageName":          "pulumi_adaptive",
				"respectSchemaVersion": true,
				"pyproject": map[string]any{
					"enabled": true,
				},
			},
			"csharp": map[string]any{
				"rootNamespace":        "AdaptiveScale",
				"respectSchemaVersion": true,
				"packageReferences": map[string]string{
					"Pulumi": "3.*",
				},
			},
		}).
		WithGoImportPath("github.com/adaptive-scale/pulumi-adaptive/sdk/go/adaptive").
		// Resource structs live in the Go package "adaptive" (dir name "provider"),
		// which infer would otherwise expose as the "provider" module. Remap it to
		// the conventional "index" module so users write adaptive.NewResource(...).
		WithModuleMap(map[tokens.ModuleName]tokens.ModuleName{"provider": "index"}).
		WithConfig(infer.Config(Config{})).
		WithResources(
			infer.Resource(&Resource{}),
			infer.Resource(&Endpoint{}),
			infer.Resource(&Authorization{}),
			infer.Resource(&Group{}),
			infer.Resource(&Script{}),
			infer.Resource(&Schedule{}),
		).
		Build()
}

// ---------------------------------------------------------------------------
// shared helpers
// ---------------------------------------------------------------------------

// sv dereferences an optional string pointer, returning "" for nil.
func sv(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// bv dereferences an optional bool pointer, returning false for nil.
func bv(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}
