GO ?= $(shell command -v go)

APPLICATION := monctl
BIN_DIR     := bin
BINARY      := $(BIN_DIR)/$(APPLICATION)

GOLANGCI_LINT_VERSION := v2.15.0
GOLANGCI_LINT         := $(BIN_DIR)/golangci-lint-$(GOLANGCI_LINT_VERSION)
GOLANGCI_LINT_ARCHIVE := golangci-lint-$(GOLANGCI_LINT_VERSION:v%=%)-$(shell $(GO) env GOHOSTOS)-$(shell $(GO) env GOHOSTARCH)
GOLANGCI_LINT_TARBALL := $(BIN_DIR)/$(GOLANGCI_LINT_ARCHIVE).tar.gz
GOLANGCI_LINT_RELEASE := https://github.com/golangci/golangci-lint/releases/download/$(GOLANGCI_LINT_VERSION)
GOLANGCI_LINT_SUMS    := $(BIN_DIR)/golangci-lint-$(GOLANGCI_LINT_VERSION:v%=%)-checksums.txt

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null)
VERSION := $(if $(strip $(VERSION)),$(VERSION),dev)
export MONCTL_VERSION := $(VERSION)

GOOS         := $(shell $(GO) env GOOS)
GOARCH       := $(shell $(GO) env GOARCH)
export GOARCH
DIST_DIR     := dist
PLAIN        := $(VERSION:v%=%)
DEB_VERSION  := $(if $(filter 0% 1% 2% 3% 4% 5% 6% 7% 8% 9%,$(PLAIN)),$(PLAIN),0.0.0+$(PLAIN))
STAGE        := $(DIST_DIR)/$(APPLICATION)_$(PLAIN)_$(GOOS)_$(GOARCH)
DEB_ROOT     := $(DIST_DIR)/deb_$(GOARCH)
PACKAGING_FILES := $(if $(filter darwin,$(GOOS)),$(wildcard packaging/macos/*.plist),packaging/linux/60-monctl.rules packaging/linux/monctl-tray.desktop)

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo ""
	@echo "\033[1mUsage:\033[0m"
	@echo "  make <target>"
	@echo ""
	@echo "\033[1mDevelopment:\033[0m"
	@grep -E '^(dep|lint|test|clean):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "\033[1mBuild:\033[0m"
	@grep -E '^(build|dist|deb|package):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
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

$(GOLANGCI_LINT):
	@mkdir -p $(BIN_DIR)
	@curl -sSfL -o $(GOLANGCI_LINT_TARBALL) $(GOLANGCI_LINT_RELEASE)/$(GOLANGCI_LINT_ARCHIVE).tar.gz
	@curl -sSfL -o $(GOLANGCI_LINT_SUMS) $(GOLANGCI_LINT_RELEASE)/$(notdir $(GOLANGCI_LINT_SUMS))
	@grep " $(GOLANGCI_LINT_ARCHIVE).tar.gz$$" $(GOLANGCI_LINT_SUMS) | sed 's|  |  $(BIN_DIR)/|' | \
		{ if command -v sha256sum >/dev/null 2>&1; then sha256sum -c -; else shasum -a 256 -c -; fi; } >/dev/null || \
		{ rm -f $(GOLANGCI_LINT_TARBALL) $(GOLANGCI_LINT_SUMS); echo "checksum mismatch for $(GOLANGCI_LINT_ARCHIVE)"; exit 1; }
	@tar -xzf $(GOLANGCI_LINT_TARBALL) -C $(BIN_DIR) --strip-components=1 $(GOLANGCI_LINT_ARCHIVE)/golangci-lint
	@mv $(BIN_DIR)/golangci-lint $@
	@rm -f $(GOLANGCI_LINT_TARBALL) $(GOLANGCI_LINT_SUMS)

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run linters, downloading the pinned linter on first use
	@$(GOLANGCI_LINT) run ./...

.PHONY: test
test: ## Run tests with the race detector and coverage
	@$(GO) test -race -cover ./...

.PHONY: clean
clean: ## Remove built files and the downloaded linter
	@rm -rf $(BIN_DIR) $(DIST_DIR)

# =============================================================================
# Build
# =============================================================================

.PHONY: build
build: ## Build monctl for this machine
	@mkdir -p $(BIN_DIR)
	@$(GO) build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=$$MONCTL_VERSION" -o $(BINARY) ./cmd/$(APPLICATION)
	@echo "Build complete! Binary: $(BINARY)"

$(STAGE)/licenses:
	@mkdir -p $@
	@for module in $$($(GO) list -deps -f '{{with .Module}}{{if not .Main}}{{.Path}}{{end}}{{end}}' ./cmd/$(APPLICATION) | sort -u); do \
		dir=$$($(GO) list -m -f '{{.Dir}}' $$module); \
		install -m 644 $$dir/LICENSE $@/$$(echo $$module | tr '/' '_').txt || exit 1; \
	done

.PHONY: dist
dist: $(STAGE)/licenses ## Build the release archive for this machine
	@$(GO) build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=$$MONCTL_VERSION" -o $(STAGE)/$(APPLICATION) ./cmd/$(APPLICATION)
	@cp LICENSE README.md $(STAGE)/
	@mkdir -p $(STAGE)/packaging
	@install -m 644 $(PACKAGING_FILES) $(STAGE)/packaging/
	@tar -czf $(STAGE).tar.gz -C $(DIST_DIR) $(notdir $(STAGE))
	@echo "Archive: $(STAGE).tar.gz"

.PHONY: deb
deb: dist ## Build the Debian package from the release archive
	@mkdir -p $(DEB_ROOT)/DEBIAN $(DEB_ROOT)/usr/bin $(DEB_ROOT)/usr/lib/udev/rules.d $(DEB_ROOT)/usr/share/doc/$(APPLICATION) $(DEB_ROOT)/usr/share/$(APPLICATION)
	@cp $(STAGE)/$(APPLICATION) $(DEB_ROOT)/usr/bin/
	@cp packaging/linux/60-monctl.rules $(DEB_ROOT)/usr/lib/udev/rules.d/
	@cp packaging/linux/monctl-tray.desktop $(DEB_ROOT)/usr/share/$(APPLICATION)/
	@cp README.md $(DEB_ROOT)/usr/share/doc/$(APPLICATION)/
	@cp LICENSE $(DEB_ROOT)/usr/share/doc/$(APPLICATION)/copyright
	@cp -R $(STAGE)/licenses $(DEB_ROOT)/usr/share/doc/$(APPLICATION)/
	@cp packaging/linux/postinst $(DEB_ROOT)/DEBIAN/postinst
	@chmod -R u=rwX,go=rX $(DEB_ROOT)
	@chmod 755 $(DEB_ROOT)/DEBIAN/postinst $(DEB_ROOT)/usr/bin/$(APPLICATION)
	@sed -e 's/@VERSION@/$(DEB_VERSION)/' -e 's/@ARCH@/$(GOARCH)/' packaging/linux/control > $(DEB_ROOT)/DEBIAN/control
	@dpkg-deb --root-owner-group --build $(DEB_ROOT) $(DIST_DIR)/$(APPLICATION)_$(DEB_VERSION)_$(GOARCH).deb >/dev/null
	@echo "Package: $(DIST_DIR)/$(APPLICATION)_$(DEB_VERSION)_$(GOARCH).deb"

.PHONY: package
package: $(if $(filter linux,$(GOOS)),deb,dist) ## Build everything a release ships for this machine

# =============================================================================
# CI
# =============================================================================

.PHONY: ci
ci: dep lint test build package ## Run every check the ci workflow runs
