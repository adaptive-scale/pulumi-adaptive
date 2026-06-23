// Package adaptive implements a native Pulumi provider for Adaptive
// (https://adaptive.live), built with pulumi-go-provider. It mirrors the
// resource surface of the Adaptive Terraform provider: resources, endpoints,
// authorizations, groups, scripts, and the MS Teams workflow integration.
package adaptive

import (
	"context"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Version is the provider version, overridable at build time via -ldflags.
var Version = "0.1.0"

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
	return infer.NewProviderBuilder().
		WithNamespace("adaptive-scale").
		WithDisplayName("Adaptive").
		WithDescription("A Pulumi provider for managing Adaptive (adaptive.live) infrastructure: "+
			"resources, endpoints, authorizations, groups, and scripts.").
		WithHomepage("https://adaptive.live").
		WithRepository("https://github.com/adaptive-scale/pulumi-adaptive").
		WithLicense("Apache-2.0").
		WithConfig(infer.Config(Config{})).
		WithResources(
			infer.Resource(&Resource{}),
			infer.Resource(&Endpoint{}),
			infer.Resource(&Authorization{}),
			infer.Resource(&Group{}),
			infer.Resource(&Script{}),
			infer.Resource(&MSTeamsWorkflow{}),
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
