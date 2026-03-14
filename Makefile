.PHONY: build install build-all clean test fmt lint

BINARY    := macsetup
CMD       := ./cmd/macsetup
INSTALL   := /usr/local/bin

# Detect arch for cross-compilation
GOARCH    ?= $(shell go env GOARCH)
GOOS      ?= darwin

# Version from git tag or fallback
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -s -w -X main.version=$(VERSION)

## build: Build the binary for the current platform
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

## install: Build and install to /usr/local/bin
install: build
	sudo cp $(BINARY) $(INSTALL)/$(BINARY)
	sudo chmod +x $(INSTALL)/$(BINARY)
	@echo "Installed to $(INSTALL)/$(BINARY)"

## build-all: Build named release artifacts for darwin/arm64 and darwin/amd64
build-all:
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-darwin-arm64 $(CMD)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-darwin-amd64 $(CMD)
	@echo "Built: $(BINARY)-darwin-arm64  $(BINARY)-darwin-amd64"

## universal: Build a universal (fat) binary for macOS (amd64 + arm64)
universal:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-amd64 $(CMD)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BINARY)-arm64 $(CMD)
	lipo -create -output $(BINARY) $(BINARY)-amd64 $(BINARY)-arm64
	rm $(BINARY)-amd64 $(BINARY)-arm64
	@echo "Built universal binary: $(BINARY)"

## clean: Remove build artifacts
clean:
	rm -f $(BINARY) $(BINARY)-amd64 $(BINARY)-arm64 $(BINARY)-darwin-arm64 $(BINARY)-darwin-amd64

## fmt: Format all Go source files
fmt:
	go fmt ./...

## lint: Run go vet
lint:
	go vet ./...

## test: Run tests
test:
	go test ./...

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
