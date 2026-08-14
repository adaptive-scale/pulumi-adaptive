package adaptive

import (
	"context"
	"strings"
	"testing"

	"github.com/blang/semver"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

// TestProviderConfigDiffNeverReplaces pins the upgrade/rotation story: a
// changed serviceToken or workspaceUrl must diff as an update to the provider,
// never as a replacement — a replacement cascades into delete/recreate of
// every resource in the stack (the engine consults DiffConfig when deciding
// whether a provider change forces resource replacement).
func TestProviderConfigDiffNeverReplaces(t *testing.T) {
	prov, err := Provider()
	if err != nil {
		t.Fatal(err)
	}
	srv, err := integration.NewServer(context.Background(), "adaptive", semver.MustParse("0.2.0"),
		integration.WithProvider(prov))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.DiffConfig(p.DiffRequest{
		Urn: "urn:pulumi:dev::proj::pulumi:providers:adaptive::default",
		OldInputs: property.NewMap(map[string]property.Value{
			"serviceToken": property.New("old-rotated-token"),
			"workspaceUrl": property.New("https://app.adaptive.live"),
		}),
		Inputs: property.NewMap(map[string]property.Value{
			"serviceToken": property.New("new-rotated-token"),
			"workspaceUrl": property.New("https://other.example.com"),
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasChanges {
		t.Fatal("expected the config change to be reported")
	}
	if resp.DeleteBeforeReplace {
		t.Error("provider config diff must not set DeleteBeforeReplace")
	}
	for key, d := range resp.DetailedDiff {
		if strings.Contains(string(d.Kind), "replace") {
			t.Errorf("property %q diffed as %q; provider config changes must never require replacement", key, d.Kind)
		}
	}
}
