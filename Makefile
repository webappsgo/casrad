# CASRAD Makefile - Simplified
PROJECT := casrad
VERSION ?= 1.0.0
BUILD_TIME := $(shell date +%FT%T%z)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build settings for static binary
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -s -w -extldflags '-static'"
BUILD_FLAGS := -trimpath -tags "sqlite_omit_load_extension,osusergo,netgo"
CGO_ENABLED := 1

# Platforms
PLATFORMS := \
	linux-amd64 \
	linux-arm64 \
	darwin-amd64 \
	darwin-arm64 \
	windows-amd64 \
	freebsd-amd64

.PHONY: build release docker test clean help

## help: Show this help message
help:
	@echo "CASRAD Build System"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  build    - Build all platform binaries + host binary"
	@echo "  release  - Create release artifacts for GitHub"
	@echo "  docker   - Build and push Docker image to ghcr.io"
	@echo "  test     - Run all tests"
	@echo "  clean    - Remove build artifacts"

## build: Build all platform binaries + host binary
build: clean
	@echo "Building CASRAD static binaries..."
	@mkdir -p dist
	@# Build for all platforms
	@$(foreach platform,$(PLATFORMS), \
		$(call build_platform,$(platform)); \
	)
	@# Build host binary
	@echo "Building host binary: $(PROJECT)"
	@CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_FLAGS) $(LDFLAGS) -o dist/$(PROJECT) cmd/casrad/main.go
	@echo ""
	@echo "✓ Build complete!"
	@echo ""
	@ls -lh dist/

define build_platform
	$(eval OS := $(word 1,$(subst -, ,$(1))))
	$(eval ARCH := $(word 2,$(subst -, ,$(1))))
	$(eval EXT := $(if $(filter windows,$(OS)),.exe,))
	@echo "  → $(PROJECT)-$(1)$(EXT)"
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=$(OS) GOARCH=$(ARCH) \
		go build $(BUILD_FLAGS) $(LDFLAGS) \
		-o dist/$(PROJECT)-$(1)$(EXT) \
		cmd/casrad/main.go
endef

## release: Create release artifacts for GitHub
release: build
	@echo "Creating release artifacts..."
	@mkdir -p dist/release
	@# Create archives for each platform
	@cd dist && for file in $(PROJECT)-*; do \
		if [ -f "$$file" ]; then \
			echo "  → $$file"; \
			if echo "$$file" | grep -q "windows"; then \
				zip -q "release/$$file.zip" "$$file"; \
			else \
				tar czf "release/$$file.tar.gz" "$$file"; \
			fi; \
		fi; \
	done
	@# Create checksums
	@cd dist/release && sha256sum * > checksums.txt
	@echo ""
	@echo "✓ Release artifacts created in dist/release/"
	@ls -lh dist/release/

## docker: Build and push Docker image to ghcr.io
docker:
	@echo "Building Docker image..."
	@docker build -t ghcr.io/casapps/$(PROJECT):$(VERSION) -t ghcr.io/casapps/$(PROJECT):latest .
	@echo ""
	@echo "Pushing to ghcr.io..."
	@docker push ghcr.io/casapps/$(PROJECT):$(VERSION)
	@docker push ghcr.io/casapps/$(PROJECT):latest
	@echo ""
	@echo "✓ Docker image pushed to ghcr.io/casapps/$(PROJECT)"

## test: Run all tests
test:
	@echo "Running tests..."
	@go test -v -race -cover ./...
	@echo ""
	@echo "✓ Tests complete"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf dist
	@go clean
	@echo "✓ Clean complete"