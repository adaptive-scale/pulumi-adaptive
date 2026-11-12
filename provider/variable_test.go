package adaptive

import (
	"context"
	"testing"

	p "github.com/pulumi/pulumi-go-provider"
)

func TestVariableSchemaMarksSecretAndReplacement(t *testing.T) {
	provider, err := Provider()
	if err != nil {
		t.Fatal(err)
	}
	spec, err := p.GetSchema(context.Background(), "adaptive", Version, provider)
	if err != nil {
		t.Fatal(err)
	}
	resource, ok := spec.Resources["adaptive:index:Variable"]
	if !ok {
		t.Fatal("Variable resource is missing from the provider schema")
	}
	if !resource.InputProperties["value"].Secret {
		t.Error("variable value must be marked secret")
	}
	if !resource.InputProperties["name"].ReplaceOnChanges {
		t.Error("variable name must replace on change")
	}
}

func TestApplyVariableReadPreservesEquivalentLabelOrder(t *testing.T) {
	prior := VariableArgs{Name: "host", Value: "old", Labels: []string{"production", "database"}}
	got := applyVariableRead(prior, &VariableReadResponse{
		Name: "host", Value: "new", Labels: []string{"database", "production"},
	})
	if got.Value != "new" {
		t.Errorf("value = %q, want new", got.Value)
	}
	if got.Labels[0] != "production" || got.Labels[1] != "database" {
		t.Errorf("equivalent labels lost input ordering: %v", got.Labels)
	}
}
