PROVIDER  := pulumi-resource-adaptive
VERSION   ?= 0.3.0
GOBIN     := $(shell go env GOPATH)/bin
LDFLAGS   := -X github.com/adaptive-scale/pulumi-adaptive/provider.Version=$(VERSION)

.PHONY: help build gen gen_dotnet build_go build_nodejs build_python build_dotnet \
	build_sdks install clean test test-integration print-version

help:
	@echo "Targets:"
	@echo "  build         Build the provider plugin binary into provider/bin/"
	@echo "  gen           Regenerate schema.json and the Go/Node.js/Python SDKs (VERSION=x.y.z)"
	@echo "  gen_dotnet    Regenerate the .NET SDK from schema.json (requires the Pulumi CLI)"
	@echo "  build_go      Compile the generated Go SDK"
	@echo "  build_nodejs  Compile the Node.js SDK into sdk/nodejs/bin"
	@echo "  build_python  Build the Python sdist/wheel into sdk/python/bin/dist"
	@echo "  build_dotnet  Build the .NET SDK and NuGet package (requires dotnet)"
	@echo "  build_sdks    Build every language SDK"
	@echo "  install       Build the plugin onto your PATH ($(GOBIN)) for use by Pulumi"
	@echo "  test          Run provider unit tests and compile the Go SDK"
	@echo "  test-integration  Deploy real resources and verify via the Client API (needs creds + install)"
	@echo "  print-version Print the build version, for scripts and install instructions"
	@echo "  clean         Remove build artifacts"

# The single source of the version, so install instructions and scripts never
# hardcode a tag that can drift from what is actually built.
print-version:
	@echo $(VERSION)

# Compile the provider plugin binary.
build:
	cd provider && go build -ldflags "$(LDFLAGS)" -o bin/$(PROVIDER) ./cmd/pulumi-resource-adaptive

# Regenerate the Pulumi schema and the Go, Node.js, and Python SDKs in-process
# (no Pulumi CLI required). Pass VERSION=x.y.z to stamp a release version.
gen:
	cd provider && go run -ldflags "$(LDFLAGS)" ./cmd/gen ../schema.json ../sdk

# Regenerate the .NET SDK. Its codegen lives outside pulumi/pkg, so this one
# goes through the Pulumi CLI.
gen_dotnet:
	pulumi package gen-sdk ./schema.json --language dotnet -o ./sdk

# Build the generated Go SDK as its own module.
build_go:
	cd sdk && go build ./...

# Compile the Node.js SDK; the publishable package lands in sdk/nodejs/bin.
build_nodejs:
	cd sdk/nodejs && npm install --no-audit --no-fund && npm run build && \
		cp package.json README.md bin/

# Build the Python sdist and wheel; artifacts land in sdk/python/bin/dist.
build_python:
	cd sdk/python && rm -rf ./bin ./venv && \
		cp -R . ../python.bin && mv ../python.bin ./bin && \
		python3 -m venv venv && \
		./venv/bin/python -m pip install --quiet build && \
		cd ./bin && ../venv/bin/python -m build .

# Build the .NET SDK; GeneratePackageOnBuild drops the .nupkg in bin/Debug.
build_dotnet:
	cd sdk/dotnet && dotnet build

build_sdks: build_go build_nodejs build_python build_dotnet

# Install the plugin where Pulumi can discover it as an ambient plugin.
# Ensure $(GOBIN) is on your PATH.
install:
	cd provider && go build -ldflags "$(LDFLAGS)" -o $(GOBIN)/$(PROVIDER) ./cmd/pulumi-resource-adaptive
	@echo "Installed $(PROVIDER) ($(VERSION)) to $(GOBIN)"

# Run unit tests, and make sure the Go SDK still compiles.
test:
	cd provider && go test ./...
	cd sdk && go build ./...

# Run the integration tests: deploy real resources via the SDK and verify them
# through the Client App API. Requires ADAPTIVE_CLIENT_ID/SECRET (e.g. in
# tests/.env.local) and a valid Adaptive token; installs the plugin first.
# 45m rather than 30m: endpoint creation polls the server until the session is
# up (~5 min each) and the lifecycle matrix creates several.
test-integration: install
	cd tests/integration && go test -tags=integration ./... -v -timeout 45m

clean:
	rm -rf provider/bin dist sdk/nodejs/bin sdk/nodejs/node_modules \
		sdk/python/bin sdk/python/venv sdk/dotnet/bin sdk/dotnet/obj sdk/*.tar.gz
