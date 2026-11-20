GO ?= $(shell command -v go)

APPLICATION := monctl
BIN_DIR     := bin
BINARY      := $(BIN_DIR)/$(APPLICATION)

GOLANGCI_LINT_VERSION := v2.15.0
GOLANGCI_LINT         := $(BIN_DIR)/golangci-lint

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null)
VERSION := $(if $(strip $(VERSION)),$(VERSION),dev)
export MONCTL_VERSION := $(VERSION)

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo ""
	@echo "\033[1mUsage:\033[0m"
	@echo "  make <target>"
	@echo ""
	@echo "\033[1mDevelopment:\033[0m"
	@grep -E '^(dep|tools|lint|test|clean):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "\033[1mBuild:\033[0m"
	@grep -E '^(build):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "\033[1mCI:\033[0m"
	@grep -E '^(ci):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "\033[1mVariables:\033[0m"
	@echo "  \033[36mGO\033[0m          $(GO)"
	@echo "  \033[36mVERSION\033[0m     $(VERSION)"
	@echo "  \033[36mLINTER\033[0m      golangci-lint $(GOLANGCI_LINT_VERSION)"
	@echo ""

# =============================================================================
# Development
# =============================================================================

.PHONY: dep
dep: ## Check that go.mod is tidy and dependencies are intact
	@$(GO) mod tidy -diff
	@$(GO) mod verify

.PHONY: tools
tools: ## Install the pinned linter into bin
	@$(GOLANGCI_LINT) version 2>/dev/null | grep -q "version $(GOLANGCI_LINT_VERSION:v%=%) " || \
		GOTOOLCHAIN=auto GOBIN=$(abspath $(BIN_DIR)) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: lint
lint: tools ## Run linters
	@$(GOLANGCI_LINT) run ./...

.PHONY: test
test: ## Run tests with the race detector and coverage
	@$(GO) test -race -cover ./...

.PHONY: clean
clean: ## Remove built files and installed tools
	@rm -rf $(BIN_DIR) dist

# =============================================================================
# Build
# =============================================================================

.PHONY: build
build: ## Build monctl for this machine
	@mkdir -p $(BIN_DIR)
	@$(GO) build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=$$MONCTL_VERSION" -o $(BINARY) ./cmd/$(APPLICATION)
	@echo "Build complete! Binary: $(BINARY)"

# =============================================================================
# CI
# =============================================================================

.PHONY: ci
ci: dep lint test build ## Run every check the ci workflow runs
