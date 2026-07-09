// Command gen emits the provider's Pulumi schema and generates the Go,
// Node.js, and Python SDKs in-process, without requiring the Pulumi CLI.
// (The .NET SDK is generated separately via `pulumi package gen-sdk`, since
// its codegen lives outside pulumi/pkg — see `make gen_dotnet`.)
//
// Usage (run from the provider/ directory):
//
//	go run ./cmd/gen <schema-out-file> <sdk-out-dir>
//
// e.g. go run ./cmd/gen ../schema.json ../sdk
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	p "github.com/pulumi/pulumi-go-provider"
	gogen "github.com/pulumi/pulumi/pkg/v3/codegen/go"
	nodejsgen "github.com/pulumi/pulumi/pkg/v3/codegen/nodejs"
	pythongen "github.com/pulumi/pulumi/pkg/v3/codegen/python"
	"github.com/pulumi/pulumi/pkg/v3/codegen/schema"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/provider"
)

func main() {
	schemaOut := "../schema.json"
	sdkOut := "../sdk"
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

	// 3. Generate each language SDK.
	for _, lang := range []struct {
		name string
		dir  string
		gen  func(*schema.Package) (map[string][]byte, error)
	}{
		{"go", "go", func(pkg *schema.Package) (map[string][]byte, error) {
			return gogen.GeneratePackage("pulumi-adaptive-gen", pkg, nil)
		}},
		{"nodejs", "nodejs", func(pkg *schema.Package) (map[string][]byte, error) {
			return nodejsgen.GeneratePackage("pulumi-adaptive-gen", pkg, nil, nil, false, nil)
		}},
		{"python", "python", func(pkg *schema.Package) (map[string][]byte, error) {
			files, err := pythongen.GeneratePackage("pulumi-adaptive-gen", pkg, nil, nil)
			if err != nil {
				return nil, err
			}
			// The generated pyproject.toml declares readme = "README.md" at the
			// package root, but codegen only emits pulumi_adaptive/README.md.
			// Mirror it to the root so `python -m build` succeeds.
			if _, ok := files["README.md"]; !ok {
				if readme, ok := files["pulumi_adaptive/README.md"]; ok {
					files["README.md"] = readme
				}
			}
			return files, nil
		}},
	} {
		files, err := lang.gen(pkg)
		if err != nil {
			return fmt.Errorf("generating %s sdk: %w", lang.name, err)
		}
		out := filepath.Join(sdkOut, lang.dir)
		if err := writeFiles(out, files); err != nil {
			return fmt.Errorf("writing %s sdk: %w", lang.name, err)
		}
		fmt.Printf("wrote %s SDK -> %s (%d files)\n", lang.name, out, len(files))
	}
	return nil
}

// writeFiles replaces dir with the given file set.
func writeFiles(dir string, files map[string][]byte) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("cleaning %s: %w", dir, err)
	}
	for path, contents := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", full, err)
		}
		if err := os.WriteFile(full, contents, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", full, err)
		}
	}
	return nil
}
