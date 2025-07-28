#!/bin/bash

# Auto-Team Universal Installation Script
# Supports macOS and Linux with automatic platform detection

set -e

# Configuration
REPO="diazoxide/auto-team"
DEFAULT_BINARY="autoteam"
BINARY_NAME=""
INSTALL_DIR="/usr/local/bin"
TEMP_DIR=$(mktemp -d)
VERSION=${VERSION:-latest}

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Cleanup function
cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

log_header() {
    echo -e "${CYAN}$1${NC}"
}

# Platform detection
detect_platform() {
    local os arch
    
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)
    
    case "$os" in
        darwin)
            OS="darwin"
            ;;
        linux)
            OS="linux"
            ;;
        *)
            log_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac
    
    case "$arch" in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        i386|i686)
            ARCH="386"
            ;;
        arm*)
            ARCH="arm"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
    
    PLATFORM="${OS}/${ARCH}"
    log_info "Detected platform: $PLATFORM"
}

# Check if binary exists and get version
check_existing() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local current_version
        current_version=$($BINARY_NAME --version 2>/dev/null | head -1 || echo "unknown")
        log_warning "$BINARY_NAME is already installed: $current_version"
        
        if [ "$FORCE_INSTALL" != "true" ]; then
            # Check if we're running in a pipe (non-interactive)
            if [ ! -t 0 ]; then
                log_info "Non-interactive mode detected. Use -f/--force flag to reinstall."
                log_info "Or run: curl -fsSL https://raw.githubusercontent.com/diazoxide/auto-team/main/scripts/install.sh | bash -s -- --force"
                exit 0
            fi
            
            echo -n "Do you want to reinstall? [y/N]: "
            read -r response </dev/tty
            case "$response" in
                [yY][eE][sS]|[yY])
                    log_info "Proceeding with reinstallation..."
                    ;;
                *)
                    log_info "Installation cancelled."
                    exit 0
                    ;;
            esac
        fi
    fi
}

# Check dependencies
check_dependencies() {
    local missing_deps=()
    
    # Check for required commands
    for cmd in curl tar sudo; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing_deps+=("$cmd")
        fi
    done
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_info "Please install them and run this script again."
        exit 1
    fi
}

# Download binary
download_binary() {
    local download_url binary_name
    
    if [ "$VERSION" = "latest" ]; then
        # For entrypoint binary, try direct download first, then build from source
        if [ "$BINARY_NAME" = "autoteam-entrypoint" ]; then
            binary_name="${BINARY_NAME}-${OS}-${ARCH}"
            download_url="https://github.com/${REPO}/releases/download/${VERSION}/${binary_name}"
            
            log_info "Downloading $binary_name..."
            if curl -fsSL "$download_url" -o "$TEMP_DIR/$BINARY_NAME" 2>/dev/null; then
                chmod +x "$TEMP_DIR/$BINARY_NAME"
                return
            fi
        fi
        
        # Fall back to building from source
        log_info "Building from source..."
        build_from_source
        return
    fi
    
    # For versioned releases, use packaged downloads
    if [ "$BINARY_NAME" = "autoteam-entrypoint" ]; then
        # Direct binary download for entrypoint
        binary_name="${BINARY_NAME}-${OS}-${ARCH}"
        download_url="https://github.com/${REPO}/releases/download/v${VERSION}/${binary_name}"
        
        log_info "Downloading $binary_name..."
        if ! curl -fsSL "$download_url" -o "$TEMP_DIR/$BINARY_NAME"; then
            log_error "Failed to download binary from $download_url"
            log_info "Falling back to building from source..."
            build_from_source
            return
        fi
        chmod +x "$TEMP_DIR/$BINARY_NAME"
    else
        # Packaged download for main binary
        binary_name="${BINARY_NAME}-${VERSION}-${OS}-${ARCH}.tar.gz"
        download_url="https://github.com/${REPO}/releases/download/v${VERSION}/${binary_name}"
        
        log_info "Downloading $binary_name..."
        
        if ! curl -fsSL "$download_url" -o "$TEMP_DIR/$binary_name"; then
            log_error "Failed to download binary from $download_url"
            log_info "Falling back to building from source..."
            build_from_source
            return
        fi
        
        log_info "Extracting binary..."
        tar -xzf "$TEMP_DIR/$binary_name" -C "$TEMP_DIR"
        
        local extracted_dir="$TEMP_DIR/${BINARY_NAME}-${VERSION}-${OS}-${ARCH}"
        if [ -f "$extracted_dir/$BINARY_NAME" ]; then
            cp "$extracted_dir/$BINARY_NAME" "$TEMP_DIR/$BINARY_NAME"
        else
            log_error "Binary not found in extracted archive"
            exit 1
        fi
    fi
}

# Build from source
build_from_source() {
    log_info "Building $BINARY_NAME from source..."
    
    # Check for Go
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is required to build from source"
        log_info "Please install Go from https://golang.org/dl/"
        exit 1
    fi
    
    # Check for git
    if ! command -v git >/dev/null 2>&1; then
        log_error "Git is required to build from source"
        exit 1
    fi
    
    local repo_dir="$TEMP_DIR/auto-team"
    
    log_info "Cloning repository..."
    git clone https://github.com/diazoxide/auto-team.git "$repo_dir" >/dev/null 2>&1 || {
        log_error "Failed to clone repository"
        log_info "You can build manually by running: make build"
        exit 1
    }
    
    cd "$repo_dir"
    
    log_info "Building binary..."
    if [ "$BINARY_NAME" = "autoteam-entrypoint" ]; then
        if ! make build-entrypoint >/dev/null 2>&1; then
            log_error "Failed to build entrypoint binary"
            exit 1
        fi
    else
        if ! make build >/dev/null 2>&1; then
            log_error "Failed to build binary"
            exit 1
        fi
    fi
    
    cp "build/$BINARY_NAME" "$TEMP_DIR/$BINARY_NAME"
}

# Install binary
install_binary() {
    local install_path
    
    if [ -n "$TARGET_PATH" ]; then
        install_path="$TARGET_PATH"
    else
        install_path="$INSTALL_DIR/$BINARY_NAME"
    fi
    
    log_info "Installing $BINARY_NAME to $install_path..."
    
    # Create directory if it doesn't exist
    local install_dir=$(dirname "$install_path")
    if [ ! -d "$install_dir" ]; then
        if [ ! -w "$(dirname "$install_dir")" ]; then
            log_info "Administrator privileges required to create directory"
            sudo mkdir -p "$install_dir"
        else
            mkdir -p "$install_dir"
        fi
    fi
    
    # Install the binary
    if [ ! -w "$install_dir" ]; then
        log_info "Administrator privileges required for installation"
        sudo cp "$TEMP_DIR/$BINARY_NAME" "$install_path"
        sudo chmod +x "$install_path"
    else
        cp "$TEMP_DIR/$BINARY_NAME" "$install_path"
        chmod +x "$install_path"
    fi
    
    log_success "$BINARY_NAME installed successfully to $install_path!"
}

# Verify installation
verify_installation() {
    local install_path
    
    if [ -n "$TARGET_PATH" ]; then
        install_path="$TARGET_PATH"
    else
        install_path="$INSTALL_DIR/$BINARY_NAME"
    fi
    
    # Check if the binary exists at the install path
    if [ -f "$install_path" ] && [ -x "$install_path" ]; then
        local version
        version=$("$install_path" --version 2>/dev/null | head -1 || echo "unknown")
        log_success "Verification successful: $version"
        
        # Only suggest running the binary if it's in PATH
        if [ -z "$TARGET_PATH" ] && command -v "$BINARY_NAME" >/dev/null 2>&1; then
            log_info "Try running: $BINARY_NAME --help"
        else
            log_info "Binary installed to: $install_path"
        fi
    else
        log_error "Installation verification failed"
        log_info "Binary not found at: $install_path"
        exit 1
    fi
}

# Usage information
usage() {
    echo "Auto-Team Installation Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -b, --binary NAME       Binary to install (autoteam|autoteam-entrypoint, default: autoteam)"
    echo "  -v, --version VERSION   Install specific version (default: latest)"
    echo "  -f, --force            Force installation even if already installed"
    echo "  -d, --dir DIRECTORY    Install directory (default: /usr/local/bin)"
    echo "  -t, --target PATH      Target installation path (overrides -d)"
    echo "  -h, --help             Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  VERSION                 Version to install"
    echo "  FORCE_INSTALL           Force installation (true/false)"
    echo "  INSTALL_DIR             Installation directory"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Install autoteam (latest)"
    echo "  $0 --binary autoteam-entrypoint      # Install entrypoint binary"
    echo "  $0 -v 1.0.0                         # Install specific version"
    echo "  $0 -f                                # Force reinstall"
    echo "  $0 -d ~/.local/bin                   # Install to custom directory"
    echo "  $0 -t /tmp/autoteam-entrypoint       # Install to specific path"
}

# Parse command line arguments
parse_args() {
    # Set default binary
    BINARY_NAME="$DEFAULT_BINARY"
    TARGET_PATH=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -b|--binary)
                case "$2" in
                    autoteam|autoteam-entrypoint)
                        BINARY_NAME="$2"
                        ;;
                    *)
                        log_error "Invalid binary name: $2. Use 'autoteam' or 'autoteam-entrypoint'"
                        exit 1
                        ;;
                esac
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -f|--force)
                FORCE_INSTALL="true"
                shift
                ;;
            -d|--dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            -t|--target)
                TARGET_PATH="$2"
                shift 2
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Main installation process
main() {
    log_header "Auto-Team Installation Script"
    log_header "=============================="
    
    parse_args "$@"
    detect_platform
    check_dependencies
    check_existing
    download_binary
    install_binary
    verify_installation
    
    echo ""
    log_success "Installation completed successfully!"
    log_info "Run '$BINARY_NAME --help' to get started."
}

# Run main function
main "$@"