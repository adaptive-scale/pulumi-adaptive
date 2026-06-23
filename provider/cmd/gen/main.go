// Command gen emits the provider's Pulumi schema and generates the Go SDK
// in-process, without requiring the Pulumi CLI.
//
// Usage (run from the provider/ directory):
//
//	go run ./cmd/gen <schema-out-file> <go-sdk-out-dir>
//
// e.g. go run ./cmd/gen ../schema.json ../sdk/go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	p "github.com/pulumi/pulumi-go-provider"
	gogen "github.com/pulumi/pulumi/pkg/v3/codegen/go"
	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/provider"
)

func main() {
	schemaOut := "../schema.json"
	sdkOut := "../sdk/go"
	if len(os.Args) > 1 {
		schemaOut = os.Args[1]
	}
	if len(os.Args) > 2 {
		sdkOut = os.Args[2]
	}

	if err := run(schemaOut, sdkOut); err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		os.Exit(1)
	}
}

func run(schemaOut, sdkOut string) error {
	ctx := context.Background()

	prov, err := adaptive.Provider()
	if err != nil {
		return fmt.Errorf("constructing provider: %w", err)
	}

	// 1. Emit the schema.
	spec, err := p.GetSchema(ctx, "adaptive", adaptive.Version, prov)
	if err != nil {
		return fmt.Errorf("getting schema: %w", err)
	}
	specJSON, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling schema: %w", err)
	}
	if err := os.WriteFile(schemaOut, append(specJSON, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", schemaOut, err)
	}
	fmt.Printf("wrote schema -> %s (%d resources)\n", schemaOut, len(spec.Resources))

	// 2. Bind the schema into a Package.
	pkg, err := schema.ImportSpec(spec, nil, schema.ValidationOptions{})
	if err != nil {
		return fmt.Errorf("binding schema: %w", err)
	}

	// 3. Generate the Go SDK.
	files, err := gogen.GeneratePackage("pulumi-adaptive-gen", pkg, nil)
	if err != nil {
		return fmt.Errorf("generating go sdk: %w", err)
	}

	// 4. Write the generated files.
	if err := os.RemoveAll(sdkOut); err != nil {
		return fmt.Errorf("cleaning %s: %w", sdkOut, err)
	}
	for path, contents := range files {
		full := filepath.Join(sdkOut, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", full, err)
		}
		if err := os.WriteFile(full, contents, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", full, err)
		}
	}
	fmt.Printf("wrote Go SDK -> %s (%d files)\n", sdkOut, len(files))
	return nil
}
