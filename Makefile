PROVIDER  := pulumi-resource-adaptive
VERSION   := 0.1.0
GOBIN     := $(shell go env GOPATH)/bin

.PHONY: help build gen build_sdk install clean test

help:
	@echo "Targets:"
	@echo "  build      Build the provider plugin binary into provider/bin/"
	@echo "  gen        Regenerate schema.json and the Go SDK (sdk/go)"
	@echo "  build_sdk  Build the generated Go SDK"
	@echo "  install    Build the plugin onto your PATH ($(GOBIN)) for use by Pulumi"
	@echo "  test       Compile the provider and SDK (sanity check)"
	@echo "  clean      Remove build artifacts"

# Compile the provider plugin binary.
build:
	cd provider && go build -o bin/$(PROVIDER) ./cmd/pulumi-resource-adaptive

# Regenerate the Pulumi schema and Go SDK in-process (no Pulumi CLI required).
gen:
	cd provider && go run ./cmd/gen ../schema.json ../sdk/go

# Build the generated Go SDK as its own module.
build_sdk:
	cd sdk && go build ./...

# Install the plugin where Pulumi can discover it as an ambient plugin.
# Ensure $(GOBIN) is on your PATH.
install:
	cd provider && go build -o $(GOBIN)/$(PROVIDER) ./cmd/pulumi-resource-adaptive
	@echo "Installed $(PROVIDER) ($(VERSION)) to $(GOBIN)"

# Sanity check: everything compiles.
test:
	cd provider && go build ./...
	cd sdk && go build ./...

clean:
	rm -rf provider/bin
