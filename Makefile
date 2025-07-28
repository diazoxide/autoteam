# Auto-Team Makefile
# Cross-platform build system for macOS and Linux

# Version and metadata
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | awk '{print $$3}')

# Build configuration
BINARY_NAME := autoteam
ENTRYPOINT_BINARY_NAME := autoteam-entrypoint
MAIN_PATH := ./cmd/autoteam
ENTRYPOINT_MAIN_PATH := ./cmd/entrypoint
BUILD_DIR := build
DIST_DIR := dist

# Go build flags
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"
GO_BUILD := go build $(LDFLAGS)

# Platform and architecture combinations
PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	linux/386 \
	linux/arm

# Linux platforms for entrypoint (Docker focus)
LINUX_PLATFORMS := \
	linux/amd64 \
	linux/arm64

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
PURPLE := \033[0;35m
CYAN := \033[0;36m
NC := \033[0m # No Color

.PHONY: all build clean test install dev help
.PHONY: build-all build-darwin build-linux build-entrypoint build-entrypoint-all
.PHONY: package package-all release
.PHONY: install-darwin install-linux

# Default target
all: clean test build

# Help target
help: ## Show this help
	@echo "$(CYAN)Auto-Team Build System$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(YELLOW)Build Information:$(NC)"
	@echo "  Version:     $(VERSION)"
	@echo "  Build Time:  $(BUILD_TIME)"
	@echo "  Git Commit:  $(GIT_COMMIT)"
	@echo "  Go Version:  $(GO_VERSION)"

# Development build (current platform)
build: ## Build binary for current platform
	@echo "$(BLUE)Building $(BINARY_NAME) for current platform...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✓ Built: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

# Build entrypoint binary (current platform)
build-entrypoint: ## Build entrypoint binary for current platform
	@echo "$(BLUE)Building $(ENTRYPOINT_BINARY_NAME) for current platform...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD) -o $(BUILD_DIR)/$(ENTRYPOINT_BINARY_NAME) $(ENTRYPOINT_MAIN_PATH)
	@echo "$(GREEN)✓ Built: $(BUILD_DIR)/$(ENTRYPOINT_BINARY_NAME)$(NC)"

# Build entrypoint for Linux platforms (Docker focus)
build-entrypoint-all: clean-build $(LINUX_PLATFORMS:=/entrypoint) ## Build entrypoint binaries for Linux platforms
	@echo "$(GREEN)✓ All entrypoint builds completed in $(BUILD_DIR)/$(NC)"

# Build for all platforms
build-all: clean-build $(PLATFORMS) ## Build binaries for all supported platforms
	@echo "$(GREEN)✓ All builds completed in $(BUILD_DIR)/$(NC)"

# Build for macOS platforms
build-darwin: clean-build darwin/amd64 darwin/arm64 ## Build binaries for macOS (Intel + Apple Silicon)
	@echo "$(GREEN)✓ macOS builds completed$(NC)"

# Build for Linux platforms  
build-linux: clean-build linux/amd64 linux/arm64 linux/386 linux/arm ## Build binaries for Linux (all architectures)
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

# Individual entrypoint platform targets
$(LINUX_PLATFORMS:=/entrypoint):
	$(eval GOOS := $(word 1,$(subst /, ,$(subst /entrypoint,,$@))))
	$(eval GOARCH := $(word 2,$(subst /, ,$(subst /entrypoint,,$@))))
	$(eval BINARY := $(BUILD_DIR)/$(ENTRYPOINT_BINARY_NAME)-$(GOOS)-$(GOARCH))
	@echo "$(PURPLE)Building entrypoint for $(GOOS)/$(GOARCH)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO_BUILD) -o $(BINARY) $(ENTRYPOINT_MAIN_PATH)
	@echo "$(GREEN)  ✓ $(BINARY)$(NC)"

# Development target with hot reload
dev: ## Build and install for development
	@echo "$(BLUE)Building development version...$(NC)"
	$(GO_BUILD) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)✓ Development build ready: ./$(BINARY_NAME)$(NC)"

# Test targets
test: ## Run all tests
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
		cp -r templates "$$ARCHIVE/"; \
		tar -czf "$$ARCHIVE.tar.gz" -C $(DIST_DIR) "$$(basename $$ARCHIVE)"; \
		rm -rf "$$ARCHIVE"; \
		echo "$(GREEN)  ✓ $$ARCHIVE.tar.gz$(NC)"; \
	done
	@echo "$(GREEN)✓ All packages created in $(DIST_DIR)/$(NC)"

# Installation targets
install: build ## Install binary to system (current platform)
	@echo "$(BLUE)Installing $(BINARY_NAME)...$(NC)"
	@if [ "$$(uname)" = "Darwin" ]; then \
		$(MAKE) install-darwin; \
	elif [ "$$(uname)" = "Linux" ]; then \
		$(MAKE) install-linux; \
	else \
		echo "$(RED)✗ Unsupported platform: $$(uname)$(NC)"; \
		exit 1; \
	fi

install-darwin: ## Install on macOS
	@if [ "$$(uname -m)" = "arm64" ]; then \
		BINARY="$(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"; \
	else \
		BINARY="$(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64"; \
	fi; \
	if [ ! -f "$$BINARY" ]; then \
		echo "$(RED)✗ Binary not found: $$BINARY$(NC)"; \
		echo "$(YELLOW)Run 'make build-darwin' first$(NC)"; \
		exit 1; \
	fi; \
	sudo cp "$$BINARY" /usr/local/bin/$(BINARY_NAME); \
	sudo chmod +x /usr/local/bin/$(BINARY_NAME); \
	echo "$(GREEN)✓ Installed to /usr/local/bin/$(BINARY_NAME)$(NC)"

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
		echo "$(RED)✗ Binary not found: $$BINARY$(NC)"; \
		echo "$(YELLOW)Run 'make build-linux' first$(NC)"; \
		exit 1; \
	fi; \
	sudo cp "$$BINARY" /usr/local/bin/$(BINARY_NAME); \
	sudo chmod +x /usr/local/bin/$(BINARY_NAME); \
	echo "$(GREEN)✓ Installed to /usr/local/bin/$(BINARY_NAME)$(NC)"

# Uninstall target
uninstall: ## Uninstall binary from system
	@echo "$(BLUE)Uninstalling $(BINARY_NAME)...$(NC)"
	@if [ -f "/usr/local/bin/$(BINARY_NAME)" ]; then \
		sudo rm /usr/local/bin/$(BINARY_NAME); \
		echo "$(GREEN)✓ Uninstalled from /usr/local/bin/$(BINARY_NAME)$(NC)"; \
	else \
		echo "$(YELLOW)! $(BINARY_NAME) not found in /usr/local/bin/$(NC)"; \
	fi

# Release target
release: clean test build-all package ## Create a complete release (test + build + package)
	@echo "$(GREEN)✓ Release $(VERSION) ready in $(DIST_DIR)/$(NC)"
	@echo "$(CYAN)Release artifacts:$(NC)"
	@ls -la $(DIST_DIR)/

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
	@echo "  Build Dir:    $(BUILD_DIR)"
	@echo "  Dist Dir:     $(DIST_DIR)"
	@echo ""
	@echo "$(CYAN)Supported Platforms:$(NC)"
	@for platform in $(PLATFORMS); do \
		echo "  - $$platform"; \
	done