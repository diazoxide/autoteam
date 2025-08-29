# AutoTeam Makefile
# Cross-platform build system for macOS and Linux

# Version and metadata
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | awk '{print $$3}')

# Build configuration
BINARY_NAME := autoteam
WORKER_BINARY_NAME := autoteam-worker
MAIN_PATH := ./cmd/autoteam
WORKER_MAIN_PATH := ./cmd/worker
BUILD_DIR := build
DIST_DIR := dist

# Build mode (dev or prod)
BUILD_MODE ?= prod

# Go build flags - different for dev vs prod
ifeq ($(BUILD_MODE),dev)
	LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"
	GO_BUILD := go build -race $(LDFLAGS)
	BUILD_SUFFIX := -dev
else
	LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"
	GO_BUILD := go build $(LDFLAGS)
	BUILD_SUFFIX :=
endif

# Source files for dependency tracking
GO_SOURCES := $(shell find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*")
MAIN_SOURCES := $(shell find $(MAIN_PATH) -name "*.go")
WORKER_SOURCES := $(shell find $(WORKER_MAIN_PATH) -name "*.go")

# Platform and architecture combinations
PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	linux/386 \
	linux/arm

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
PURPLE := \033[0;35m
CYAN := \033[0;36m
NC := \033[0m # No Color

.PHONY: all build clean test install dev help
.PHONY: build-all build-darwin build-linux build-worker build-worker-all
.PHONY: package package-all release checksums verify
.PHONY: install-darwin install-linux dev-mode prod-mode

# Default target
all: clean test build

# Help target
help: ## Show this help
	@echo "$(CYAN)AutoTeam Build System$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(YELLOW)Build Information:$(NC)"
	@echo "  Version:     $(VERSION)"
	@echo "  Build Time:  $(BUILD_TIME)"
	@echo "  Git Commit:  $(GIT_COMMIT)"
	@echo "  Go Version:  $(GO_VERSION)"

# Development build (current platform) - with dependency tracking
$(BUILD_DIR)/$(BINARY_NAME): $(GO_SOURCES) $(MAIN_SOURCES) | $(BUILD_DIR) codegen
	@echo "$(BLUE)Building $(BINARY_NAME) for current platform...$(NC)"
	$(GO_BUILD) -o $@ $(MAIN_PATH)
	@echo "$(GREEN)✓ Built: $@$(NC)"

# Build worker binary (current platform) - with dependency tracking
$(BUILD_DIR)/$(WORKER_BINARY_NAME): $(GO_SOURCES) $(WORKER_SOURCES) | $(BUILD_DIR)
	@echo "$(BLUE)Building $(WORKER_BINARY_NAME) for current platform...$(NC)"
	$(GO_BUILD) -o $@ $(WORKER_MAIN_PATH)
	@echo "$(GREEN)✓ Built: $@$(NC)"

# Convenience targets
build: $(BUILD_DIR)/$(BINARY_NAME) ## Build binary for current platform
build-worker: $(BUILD_DIR)/$(WORKER_BINARY_NAME) ## Build worker binary for current platform

# Ensure build directory exists
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# Build worker binaries for all platforms
build-worker-all: $(PLATFORMS:=/worker) ## Build worker binaries for all platforms
	@echo "$(GREEN)✓ All worker builds completed in $(BUILD_DIR)/$(NC)"

# Build for all platforms (main + worker binaries) - with parallel execution
build-all: clean-build ## Build main and worker binaries for all supported platforms
	@echo "$(BLUE)Building all platforms in parallel...$(NC)"
	@$(MAKE) -j$(shell nproc 2>/dev/null || echo 4) $(PLATFORMS) $(PLATFORMS:=/worker)
	@echo "$(GREEN)✓ All builds completed in $(BUILD_DIR)/$(NC)"

# Build for macOS platforms - with parallel execution
build-darwin: clean-build ## Build binaries for macOS (Intel + Apple Silicon)
	@echo "$(BLUE)Building macOS platforms in parallel...$(NC)"
	@$(MAKE) -j$(shell nproc 2>/dev/null || echo 2) darwin/amd64 darwin/arm64
	@echo "$(GREEN)✓ macOS builds completed$(NC)"

# Build for Linux platforms (main + worker binaries) - with parallel execution
build-linux: clean-build ## Build main and worker binaries for Linux (all architectures)
	@echo "$(BLUE)Building Linux platforms in parallel...$(NC)"
	@$(MAKE) -j$(shell nproc 2>/dev/null || echo 4) linux/amd64 linux/arm64 linux/386 linux/arm linux/amd64/worker linux/arm64/worker
	@echo "$(GREEN)✓ Linux builds completed$(NC)"

# Individual platform targets
$(PLATFORMS):
	$(eval GOOS := $(word 1,$(subst /, ,$@)))
	$(eval GOARCH := $(word 2,$(subst /, ,$@)))
	$(eval BINARY := $(BUILD_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH))
	$(eval BINARY_EXT := $(if $(filter windows,$(GOOS)),.exe,))
	@echo "$(PURPLE)Building for $(GOOS)/$(GOARCH)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO_BUILD) -o $(BINARY)$(BINARY_EXT) $(MAIN_PATH)
	@echo "$(GREEN)  ✓ $(BINARY)$(BINARY_EXT)$(NC)"

# Individual worker platform targets
$(PLATFORMS:=/worker):
	$(eval GOOS := $(word 1,$(subst /, ,$(subst /worker,,$@))))
	$(eval GOARCH := $(word 2,$(subst /, ,$(subst /worker,,$@))))
	$(eval BINARY := $(BUILD_DIR)/$(WORKER_BINARY_NAME)-$(GOOS)-$(GOARCH))
	@echo "$(PURPLE)Building worker for $(GOOS)/$(GOARCH)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO_BUILD) -o $(BINARY) $(WORKER_MAIN_PATH)
	@echo "$(GREEN)  ✓ $(BINARY)$(NC)"

# Development mode builds
dev-mode: ## Switch to development build mode (race detection, no optimization)
	@echo "$(BLUE)Switching to development build mode...$(NC)"
	@$(MAKE) BUILD_MODE=dev build
	@echo "$(GREEN)✓ Development build ready with race detection$(NC)"

# Production mode builds
prod-mode: ## Switch to production build mode (optimized, stripped)
	@echo "$(BLUE)Switching to production build mode...$(NC)"
	@$(MAKE) BUILD_MODE=prod build
	@echo "$(GREEN)✓ Production build ready$(NC)"

# Development target with hot reload
dev: dev-mode ## Build and install for development

# Test targets
test: codegen ## Run all tests
	@echo "$(BLUE)Running tests...$(NC)"
	go test -v ./...
	@echo "$(GREEN)✓ All tests passed$(NC)"

test-coverage: ## Run tests with coverage report
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report generated: coverage.html$(NC)"

test-race: ## Run tests with race detection
	@echo "$(BLUE)Running tests with race detection...$(NC)"
	go test -race ./...
	@echo "$(GREEN)✓ Race condition tests passed$(NC)"

# Packaging targets
package: build-all ## Create distribution packages for all platforms
	@echo "$(BLUE)Creating distribution packages...$(NC)"
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		BINARY="$(BUILD_DIR)/$(BINARY_NAME)-$$GOOS-$$GOARCH"; \
		ARCHIVE="$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-$$GOOS-$$GOARCH"; \
		echo "$(PURPLE)Packaging $$GOOS/$$GOARCH...$(NC)"; \
		if [ "$$GOOS" = "windows" ]; then \
			BINARY="$$BINARY.exe"; \
		fi; \
		mkdir -p "$$ARCHIVE"; \
		cp "$$BINARY" "$$ARCHIVE/$(BINARY_NAME)$$([[ $$GOOS == windows ]] && echo .exe)"; \
		cp README.md "$$ARCHIVE/"; \
		cp -r examples "$$ARCHIVE/"; \
		tar -czf "$$ARCHIVE.tar.gz" -C $(DIST_DIR) "$$(basename $$ARCHIVE)"; \
		rm -rf "$$ARCHIVE"; \
		echo "$(GREEN)  ✓ $$ARCHIVE.tar.gz$(NC)"; \
	done
	@echo "$(GREEN)✓ All packages created in $(DIST_DIR)/$(NC)"

# Installation targets
install: build build-worker-all ## Install binaries to system (current platform + all worker platforms)
	@echo "$(BLUE)Installing $(BINARY_NAME) and worker binaries...$(NC)"
	@if [ "$$(uname)" = "Darwin" ]; then \
		$(MAKE) install-darwin; \
	elif [ "$$(uname)" = "Linux" ]; then \
		$(MAKE) install-linux; \
	else \
		echo "$(RED)✗ Unsupported platform: $$(uname)$(NC)"; \
		exit 1; \
	fi
	@$(MAKE) install-workers

install-darwin: ## Install on macOS
	@if [ "$$(uname -m)" = "arm64" ]; then \
		BINARY="$(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"; \
	else \
		BINARY="$(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64"; \
	fi; \
	if [ ! -f "$$BINARY" ]; then \
		echo "$(YELLOW)⚠ Cross-platform binary not found: $$BINARY$(NC)"; \
		echo "$(BLUE)Using current platform binary: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"; \
		BINARY="$(BUILD_DIR)/$(BINARY_NAME)"; \
		if [ ! -f "$$BINARY" ]; then \
			echo "$(RED)✗ Binary not found: $$BINARY$(NC)"; \
			echo "$(YELLOW)Run 'make build' first$(NC)"; \
			exit 1; \
		fi; \
	fi; \
	sudo cp "$$BINARY" /usr/local/bin/$(BINARY_NAME); \
	sudo chmod +x /usr/local/bin/$(BINARY_NAME); \
	echo "$(GREEN)✓ Installed $(BINARY_NAME) to /usr/local/bin/$(BINARY_NAME)$(NC)"; \
	\
	if [ -f "$(BUILD_DIR)/$(WORKER_BINARY_NAME)" ]; then \
		sudo cp "$(BUILD_DIR)/$(WORKER_BINARY_NAME)" /usr/local/bin/$(WORKER_BINARY_NAME); \
		sudo chmod +x /usr/local/bin/$(WORKER_BINARY_NAME); \
		echo "$(GREEN)✓ Installed $(WORKER_BINARY_NAME) to /usr/local/bin/$(WORKER_BINARY_NAME)$(NC)"; \
	else \
		CURRENT_ARCH=$$(uname -m | sed 's/x86_64/amd64/'); \
		PLATFORM_BINARY="$(BUILD_DIR)/$(WORKER_BINARY_NAME)-darwin-$$CURRENT_ARCH"; \
		if [ -f "$$PLATFORM_BINARY" ]; then \
			sudo cp "$$PLATFORM_BINARY" /usr/local/bin/$(WORKER_BINARY_NAME); \
			sudo chmod +x /usr/local/bin/$(WORKER_BINARY_NAME); \
			echo "$(GREEN)✓ Installed $(WORKER_BINARY_NAME) to /usr/local/bin/$(WORKER_BINARY_NAME)$(NC)"; \
		else \
			echo "$(YELLOW)⚠ No worker binary found, run 'make build-worker' or 'make build-worker-all' first$(NC)"; \
		fi; \
	fi

install-linux: ## Install on Linux
	@ARCH=$$(uname -m); \
	case $$ARCH in \
		x86_64) GOARCH="amd64" ;; \
		aarch64|arm64) GOARCH="arm64" ;; \
		i386|i686) GOARCH="386" ;; \
		arm*) GOARCH="arm" ;; \
		*) echo "$(RED)✗ Unsupported architecture: $$ARCH$(NC)"; exit 1 ;; \
	esac; \
	BINARY="$(BUILD_DIR)/$(BINARY_NAME)-linux-$$GOARCH"; \
	if [ ! -f "$$BINARY" ]; then \
		echo "$(YELLOW)⚠ Cross-platform binary not found: $$BINARY$(NC)"; \
		echo "$(BLUE)Using current platform binary: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"; \
		BINARY="$(BUILD_DIR)/$(BINARY_NAME)"; \
		if [ ! -f "$$BINARY" ]; then \
			echo "$(RED)✗ Binary not found: $$BINARY$(NC)"; \
			echo "$(YELLOW)Run 'make build' first$(NC)"; \
			exit 1; \
		fi; \
	fi; \
	sudo cp "$$BINARY" /usr/local/bin/$(BINARY_NAME); \
	sudo chmod +x /usr/local/bin/$(BINARY_NAME); \
	echo "$(GREEN)✓ Installed $(BINARY_NAME) to /usr/local/bin/$(BINARY_NAME)$(NC)"; \
	\
	ENTRYPOINT_BINARY="$(BUILD_DIR)/$(WORKER_BINARY_NAME)-linux-$$GOARCH"; \
	if [ -f "$$ENTRYPOINT_BINARY" ]; then \
		sudo cp "$$ENTRYPOINT_BINARY" /usr/local/bin/$(WORKER_BINARY_NAME); \
		sudo chmod +x /usr/local/bin/$(WORKER_BINARY_NAME); \
		echo "$(GREEN)✓ Installed $(WORKER_BINARY_NAME) to /usr/local/bin/$(WORKER_BINARY_NAME)$(NC)"; \
	elif [ -f "$(BUILD_DIR)/$(WORKER_BINARY_NAME)" ]; then \
		sudo cp "$(BUILD_DIR)/$(WORKER_BINARY_NAME)" /usr/local/bin/$(WORKER_BINARY_NAME); \
		sudo chmod +x /usr/local/bin/$(WORKER_BINARY_NAME); \
		echo "$(GREEN)✓ Installed $(WORKER_BINARY_NAME) to /usr/local/bin/$(WORKER_BINARY_NAME)$(NC)"; \
	else \
		echo "$(YELLOW)⚠ No worker binary found, run 'make build-worker' or 'make build-worker-all' first$(NC)"; \
	fi

install-workers: ## Install worker binaries for all platforms to /opt/autoteam/bin
	@echo "$(BLUE)Installing worker binaries for all platforms...$(NC)"
	@sudo mkdir -p /opt/autoteam/bin
	@echo "$(BLUE)Installing entrypoint.sh script...$(NC)"
	@sudo cp scripts/entrypoint.sh /opt/autoteam/bin/entrypoint.sh
	@sudo chmod +x /opt/autoteam/bin/entrypoint.sh
	@echo "$(GREEN)✓ Installed entrypoint.sh to /opt/autoteam/bin/entrypoint.sh$(NC)"
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		BINARY="$(BUILD_DIR)/$(WORKER_BINARY_NAME)-$$GOOS-$$GOARCH"; \
		TARGET="/opt/autoteam/bin/$(WORKER_BINARY_NAME)-$$GOOS-$$GOARCH"; \
		if [ -f "$$BINARY" ]; then \
			sudo cp "$$BINARY" "$$TARGET"; \
			sudo chmod +x "$$TARGET"; \
			echo "$(GREEN)✓ Installed $$TARGET$(NC)"; \
		else \
			echo "$(YELLOW)⚠ Binary not found: $$BINARY$(NC)"; \
		fi; \
	done
	@echo "$(GREEN)✓ All available entrypoint binaries installed to /opt/autoteam/bin$(NC)"

# Uninstall target
uninstall: ## Uninstall binaries from system
	@echo "$(BLUE)Uninstalling $(BINARY_NAME) and $(WORKER_BINARY_NAME)...$(NC)"
	@if [ -f "/usr/local/bin/$(BINARY_NAME)" ]; then \
		sudo rm /usr/local/bin/$(BINARY_NAME); \
		echo "$(GREEN)✓ Uninstalled $(BINARY_NAME) from /usr/local/bin/$(NC)"; \
	else \
		echo "$(YELLOW)! $(BINARY_NAME) not found in /usr/local/bin/$(NC)"; \
	fi
	@if [ -f "/usr/local/bin/$(WORKER_BINARY_NAME)" ]; then \
		sudo rm /usr/local/bin/$(WORKER_BINARY_NAME); \
		echo "$(GREEN)✓ Uninstalled $(WORKER_BINARY_NAME) from /usr/local/bin/$(NC)"; \
	else \
		echo "$(YELLOW)! $(WORKER_BINARY_NAME) not found in /usr/local/bin/$(NC)"; \
	fi
	@if [ -d "/opt/autoteam/bin" ]; then \
		sudo rm -rf /opt/autoteam/bin; \
		echo "$(GREEN)✓ Uninstalled entrypoint binaries from /opt/autoteam/bin$(NC)"; \
	else \
		echo "$(YELLOW)! Binary directory not found in /opt/autoteam/bin$(NC)"; \
	fi

# Generate checksums for build artifacts
checksums: build-all ## Generate SHA256 checksums for all build artifacts
	@echo "$(BLUE)Generating checksums...$(NC)"
	@cd $(BUILD_DIR) && find . -name "$(BINARY_NAME)*" -o -name "$(WORKER_BINARY_NAME)*" | xargs shasum -a 256 > checksums.txt
	@echo "$(GREEN)✓ Checksums generated in $(BUILD_DIR)/checksums.txt$(NC)"

# Verify build artifacts
verify: ## Verify build artifact checksums
	@echo "$(BLUE)Verifying checksums...$(NC)"
	@if [ -f "$(BUILD_DIR)/checksums.txt" ]; then \
		cd $(BUILD_DIR) && shasum -a 256 -c checksums.txt; \
		echo "$(GREEN)✓ All checksums verified$(NC)"; \
	else \
		echo "$(YELLOW)⚠ No checksums file found, run 'make checksums' first$(NC)"; \
	fi

# Release target
release: clean test build-all checksums package ## Create a complete release (test + build + package + checksums)
	@echo "$(GREEN)✓ Release $(VERSION) ready in $(DIST_DIR)/$(NC)"
	@echo "$(CYAN)Release artifacts:$(NC)"
	@ls -la $(DIST_DIR)/
	@echo "$(CYAN)Build checksums:$(NC)"
	@cat $(BUILD_DIR)/checksums.txt

# Clean targets
clean: clean-build clean-dist ## Clean all generated files
	@echo "$(GREEN)✓ Cleaned all generated files$(NC)"

clean-build: ## Clean build directory
	@echo "$(BLUE)Cleaning build directory...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)

clean-dist: ## Clean distribution directory
	@echo "$(BLUE)Cleaning distribution directory...$(NC)"
	@rm -rf $(DIST_DIR)

clean-test: ## Clean test artifacts
	@echo "$(BLUE)Cleaning test artifacts...$(NC)"
	@rm -f coverage.out coverage.html

# Go module management
mod-tidy: ## Tidy and verify go modules
	@echo "$(BLUE)Tidying Go modules...$(NC)"
	go mod tidy
	go mod verify
	@echo "$(GREEN)✓ Go modules are tidy$(NC)"

mod-update: ## Update Go modules
	@echo "$(BLUE)Updating Go modules...$(NC)"
	go get -u ./...
	go mod tidy
	@echo "$(GREEN)✓ Go modules updated$(NC)"

# Linting and formatting
fmt: ## Format Go code
	@echo "$(BLUE)Formatting Go code...$(NC)"
	go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	go vet ./...
	@echo "$(GREEN)✓ go vet passed$(NC)"

# Code generation
codegen: ## Generate API code from OpenAPI spec
	@echo "$(BLUE)Generating Worker API code from OpenAPI specification...$(NC)"
	@cd api/worker && go generate .
	@echo "$(BLUE)Copying OpenAPI spec for server embedding...$(NC)"
	@cd internal/server && go generate .
	@echo "$(GREEN)✓ Worker API code generated$(NC)"

# Development workflow  
check: fmt vet test ## Run all checks (format, vet, test)
	@echo "$(GREEN)✓ All checks passed$(NC)"

# Show build information
info: ## Show build information
	@echo "$(CYAN)Build Information:$(NC)"
	@echo "  Binary Name:  $(BINARY_NAME)"
	@echo "  Version:      $(VERSION)"
	@echo "  Build Time:   $(BUILD_TIME)"
	@echo "  Git Commit:   $(GIT_COMMIT)"
	@echo "  Go Version:   $(GO_VERSION)"
	@echo "  Build Mode:   $(BUILD_MODE)$(BUILD_SUFFIX)"
	@echo "  Build Dir:    $(BUILD_DIR)"
	@echo "  Dist Dir:     $(DIST_DIR)"
	@echo ""
	@echo "$(CYAN)Supported Platforms:$(NC)"
	@for platform in $(PLATFORMS); do \
		echo "  - $$platform"; \
	done
