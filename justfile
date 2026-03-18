# mac — declarative macOS configuration tool

binary  := "mac"
cmd     := "./cmd/mac"
install := "/usr/local/bin"

version := `git describe --tags --always --dirty 2>/dev/null || echo "dev"`
ldflags := "-s -w -X main.version=" + version

# Default: show available recipes
[private]
default:
    @just --list

# Build the binary for the current platform
build:
    go build -ldflags "{{ldflags}}" -o {{binary}} {{cmd}}

# Build and install to /usr/local/bin
install: build
    sudo cp {{binary}} {{install}}/{{binary}}
    sudo chmod +x {{install}}/{{binary}}
    @echo "Installed to {{install}}/{{binary}}"

# Cross-compile named release artifacts for darwin/arm64 and darwin/amd64
build-all:
    GOOS=darwin GOARCH=arm64 go build -ldflags "{{ldflags}}" -o {{binary}}-darwin-arm64 {{cmd}}
    GOOS=darwin GOARCH=amd64 go build -ldflags "{{ldflags}}" -o {{binary}}-darwin-amd64 {{cmd}}
    @echo "Built: {{binary}}-darwin-arm64  {{binary}}-darwin-amd64"

# Build a universal (fat) macOS binary combining amd64 + arm64
universal:
    GOOS=darwin GOARCH=amd64 go build -ldflags "{{ldflags}}" -o {{binary}}-amd64 {{cmd}}
    GOOS=darwin GOARCH=arm64 go build -ldflags "{{ldflags}}" -o {{binary}}-arm64 {{cmd}}
    lipo -create -output {{binary}} {{binary}}-amd64 {{binary}}-arm64
    rm {{binary}}-amd64 {{binary}}-arm64
    @echo "Built universal binary: {{binary}}"

# Remove all build artifacts
clean:
    rm -f {{binary}} {{binary}}-amd64 {{binary}}-arm64 {{binary}}-darwin-arm64 {{binary}}-darwin-amd64

# Format all Go source files
fmt:
    go fmt ./...

# Run go vet
lint:
    go vet ./...

# Run tests
test:
    go test ./...

# fmt + lint + test (pre-commit gate)
check: fmt lint test
