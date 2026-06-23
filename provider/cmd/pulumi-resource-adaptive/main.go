package main

import (
	"context"
	"fmt"
	"os"

	p "github.com/pulumi/pulumi-go-provider"

	adaptive "github.com/adaptive-scale/pulumi-adaptive/provider"
)

func main() {
	prov, err := adaptive.Provider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to construct provider: %v\n", err)
		os.Exit(1)
	}
	if err := p.RunProvider(context.Background(), "adaptive", adaptive.Version, prov); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
