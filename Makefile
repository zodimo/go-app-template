.PHONY: help build test vet config-init migrate version validate-version version-bump-patch version-bump-minor version-bump-major

.DEFAULT_GOAL := help

## Display available targets and descriptions
help:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# MODULE is derived from go.mod so it survives `gonew`; BIN_TRIO is the single
# `setup`-managed line so it survives re-identification.
MODULE := $(shell go list -m 2>/dev/null)
BIN_TRIO ?= cli
BINARY = $(BIN_TRIO)
CONFIG_BINARY = $(BIN_TRIO)-config
MIGRATION_BINARY = $(BIN_TRIO)-migration
BUILD_DIR=bin


# Version information from git
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
# Version stripped from the leading v
CLEAN_VERSION = $(patsubst v%,%,$(VERSION))
COMMIT_ID ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date +%FT%T%z)

# Version extraction and manipulation
CURRENT_VERSION := $(shell v=$$(git tag --list "v[0-9]*.[0-9]*.[0-9]*" --sort=-v:refname | head -n1 | sed 's/^v//' | tr -d '\n'); if [ -z "$$v" ]; then echo "0.0.0"; else echo "$$v"; fi)
MAJOR := $(shell echo $(CURRENT_VERSION) | cut -d. -f1)
MINOR := $(shell echo $(CURRENT_VERSION) | cut -d. -f2)
PATCH := $(shell echo $(CURRENT_VERSION) | cut -d. -f3)

NEW_PATCH := $(shell echo $$(($(PATCH) + 1)))
NEW_MINOR := $(shell echo $$(($(MINOR) + 1)))
NEW_MAJOR := $(shell echo $$(($(MAJOR) + 1)))
NEXT_PATCH_VERSION := $(MAJOR).$(MINOR).$(NEW_PATCH)
NEXT_MINOR_VERSION := $(MAJOR).$(NEW_MINOR).0
NEXT_MAJOR_VERSION := $(NEW_MAJOR).0.0
# Allow pre-release and build metadata in version validation
VERSION_VALID := $(shell echo $(CURRENT_VERSION) | grep -E '^[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9\.-]+)?(\+[A-Za-z0-9\.-]+)?$$' >/dev/null && echo "true" || echo "false")
# has uncommitted files
IS_DIRTY := $(shell git diff-index --quiet HEAD -- || echo "true")


# Build flags content
LDFLAGS_CONTENT = -s -w -X $(MODULE)/internal/version.Version=v$(CLEAN_VERSION) \
					-X $(MODULE)/internal/version.CommitID=$(COMMIT_ID) \
					-X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE)

# Build flags
LDFLAGS = -ldflags "$(LDFLAGS_CONTENT)"


build: ## Build all binaries and packages
	@echo "Building $(BINARY) v$(CLEAN_VERSION)..."
	@mkdir -p $(BUILD_DIR)/$(BINARY)-v$(CLEAN_VERSION)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/$(BINARY)
	@cp $(BUILD_DIR)/$(BINARY) $(BUILD_DIR)/$(BINARY)-v$(CLEAN_VERSION)/$(BINARY)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(CONFIG_BINARY) ./cmd/$(CONFIG_BINARY)
	@cp $(BUILD_DIR)/$(CONFIG_BINARY) $(BUILD_DIR)/$(BINARY)-v$(CLEAN_VERSION)/$(CONFIG_BINARY)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(MIGRATION_BINARY) ./cmd/$(MIGRATION_BINARY)
	@cp $(BUILD_DIR)/$(MIGRATION_BINARY) $(BUILD_DIR)/$(BINARY)-v$(CLEAN_VERSION)/$(MIGRATION_BINARY)


test: ## Run the test suite
	go test ./...


vet: ## Run go vet
	go vet ./...


config-init: ## Generate the config file on first run
	go run ./cmd/$(CONFIG_BINARY) init


migrate: ## Run database migrations
	go run ./cmd/$(MIGRATION_BINARY) migrate

version: ## Display current version
	@echo "Current version: $(CURRENT_VERSION)"

validate-version: ## Validate current version is semver compliant
ifeq ($(VERSION_VALID),false)
	@echo "Error: Current version '$(CURRENT_VERSION)' is not a valid semver."
	@exit 1
endif

version-bump-patch: validate-version ## Bump patch version
	@if [ "$(IS_DIRTY)" = "true" ]; then \
		echo "Warning: You have uncommitted changes."; \
		read -p "Continue anyway? [y/N] " confirm; \
		if [ "$$confirm" != "y" ]; then exit 1; fi; \
	fi
	@echo "Bumping patch version: $(CURRENT_VERSION) -> $(NEXT_PATCH_VERSION)"
	@git tag -a v$(NEXT_PATCH_VERSION) -m "Bump patch version to $(NEXT_PATCH_VERSION)"
	@echo "Tagged with v$(NEXT_PATCH_VERSION)"

version-bump-minor: validate-version ## Bump minor version
	@if [ "$(IS_DIRTY)" = "true" ]; then \
		echo "Warning: You have uncommitted changes."; \
		read -p "Continue anyway? [y/N] " confirm; \
		if [ "$$confirm" != "y" ]; then exit 1; fi; \
	fi
	@echo "Bumping minor version: $(CURRENT_VERSION) -> $(NEXT_MINOR_VERSION)"
	@git tag -a v$(NEXT_MINOR_VERSION) -m "Bump minor version to $(NEXT_MINOR_VERSION)"
	@echo "Tagged with v$(NEXT_MINOR_VERSION)"

version-bump-major: validate-version ## Bump major version
	@if [ "$(IS_DIRTY)" = "true" ]; then \
		echo "Warning: You have uncommitted changes."; \
		read -p "Continue anyway? [y/N] " confirm; \
		if [ "$$confirm" != "y" ]; then exit 1; fi; \
	fi
	@echo "Bumping major version: $(CURRENT_VERSION) -> $(NEXT_MAJOR_VERSION)"
	@git tag -a v$(NEXT_MAJOR_VERSION) -m "Bump major version to $(NEXT_MAJOR_VERSION)"
	@echo "Tagged with v$(NEXT_MAJOR_VERSION)"
