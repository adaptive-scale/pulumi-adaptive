package adaptive

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"github.com/blang/semver"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/integration"
)

// The provider must declare no configuration. Credentials live in the
// environment; provider config is persisted on the provider resource in the
// stack, which is how the service token used to end up in state (and, being
// secret, made the stack unreadable without its passphrase). Re-adding config
// should be a deliberate act, so this fails if one appears.
func TestProviderDeclaresNoConfig(t *testing.T) {
	prov, err := Provider()
	if err != nil {
		t.Fatal(err)
	}
	srv, err := integration.NewServer(context.Background(), "adaptive",
		semver.MustParse(Version), integration.WithProvider(prov))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := srv.GetSchema(p.GetSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}

	var schema struct {
		Config struct {
			Variables map[string]any `json:"variables"`
		} `json:"config"`
		Provider struct {
			InputProperties map[string]any `json:"inputProperties"`
		} `json:"provider"`
	}
	if err := json.Unmarshal([]byte(resp.Schema), &schema); err != nil {
		t.Fatal(err)
	}
	if len(schema.Config.Variables) > 0 {
		t.Errorf("provider declares config %v; credentials belong in the environment, "+
			"not in stack state", keysOf(schema.Config.Variables))
	}
	if len(schema.Provider.InputProperties) > 0 {
		t.Errorf("provider declares input properties %v; these become an explicit "+
			"Provider type in every SDK and land in state", keysOf(schema.Provider.InputProperties))
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
