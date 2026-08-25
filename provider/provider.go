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
var Version = "0.3.0"

// The provider declares no configuration. Credentials come from the environment
// (ADAPTIVE_SVC_TOKEN, ADAPTIVE_URL) or from ~/.adaptive/token — see
// resolveToken.
//
// They used to be provider config, which Pulumi persists on the provider
// resource in the stack. That put the service token in state, and because it was
// marked secret the stack could not be read or exported without the passphrase
// or KMS key. It also meant rotating the token diffed the provider resource,
// which the infer layer reported as a replacement (pulumi-go-provider#409) and
// the engine cascaded into recreating every resource in the stack — this file
// used to carry a DiffConfig wrapper whose only job was to downgrade those
// replacement diff kinds. Declaring no config removes the cause rather than the
// symptom.
//
// Two consequences worth knowing. One process targets one workspace: there is no
// per-stack override, because there is no config to put it in. And no Provider
// type is generated in the SDKs, so an explicit provider instance is not
// something a program can construct.

// clientFromConfig builds an Adaptive API client from the ambient environment.
func clientFromConfig(ctx context.Context) (*Client, error) {
	token, url, err := resolveToken()
	if err != nil {
		return nil, err
	}
	return NewClient(token, url), nil
}

// Provider builds the inferred Pulumi provider.
func Provider() (p.Provider, error) {
	return buildProvider()
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
		WithResources(
			infer.Resource(&Resource{}),
			infer.Resource(&Endpoint{}),
			infer.Resource(&Authorization{}),
			infer.Resource(&Group{}),
			infer.Resource(&Script{}),
			infer.Resource(&Schedule{}),
			infer.Resource(&DataProtection{}),
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
