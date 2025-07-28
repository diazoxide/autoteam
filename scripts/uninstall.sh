#!/bin/bash

# Auto-Team Uninstallation Script
# Removes autoteam binary and configuration files

set -e

# Configuration
BINARY_NAME="autoteam"
ENTRYPOINT_BINARY_NAME="autoteam-entrypoint"
INSTALL_LOCATIONS=(
    "/usr/local/bin/$BINARY_NAME"
    "/usr/bin/$BINARY_NAME"
    "$HOME/.local/bin/$BINARY_NAME"
    "$HOME/bin/$BINARY_NAME"
    "/usr/local/bin/$ENTRYPOINT_BINARY_NAME"
    "/usr/bin/$ENTRYPOINT_BINARY_NAME"
    "$HOME/.local/bin/$ENTRYPOINT_BINARY_NAME"
    "$HOME/bin/$ENTRYPOINT_BINARY_NAME"
)

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Check if binaries are installed
check_installation() {
    local found=false
    
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local version
        version=$($BINARY_NAME --version 2>/dev/null | head -1 || echo "unknown")
        log_info "Found $BINARY_NAME: $version"
        found=true
    fi
    
    if command -v "$ENTRYPOINT_BINARY_NAME" >/dev/null 2>&1; then
        local version
        version=$($ENTRYPOINT_BINARY_NAME --version 2>/dev/null | head -1 || echo "unknown")
        log_info "Found $ENTRYPOINT_BINARY_NAME: $version"
        found=true
    fi
    
    if [ "$found" = false ]; then
        log_warning "Neither $BINARY_NAME nor $ENTRYPOINT_BINARY_NAME is installed or in PATH"
        return 1
    fi
    return 0
}

# Find installed binary locations
find_installations() {
    local found_locations=()
    
    for location in "${INSTALL_LOCATIONS[@]}"; do
        if [ -f "$location" ]; then
            found_locations+=("$location")
        fi
    done
    
    echo "${found_locations[@]}"
}

# Remove binary files
remove_binaries() {
    local locations=("$@")
    local removed_count=0
    
    for location in "${locations[@]}"; do
        log_info "Removing $location..."
        
        if [ -w "$(dirname "$location")" ]; then
            rm -f "$location"
        else
            log_info "Administrator privileges required to remove $location"
            sudo rm -f "$location"
        fi
        
        if [ ! -f "$location" ]; then
            log_success "Removed $location"
            ((removed_count++))
        else
            log_error "Failed to remove $location"
        fi
    done
    
    return $removed_count
}

# Clean up auto-team directories and files
cleanup_files() {
    local cleanup_paths=(
        "$HOME/.autoteam"
        "$HOME/.config/autoteam"
        "/tmp/autoteam-*"
    )
    
    log_info "Cleaning up configuration files..."
    
    for path in "${cleanup_paths[@]}"; do
        if [[ "$path" == *"*"* ]]; then
            # Handle glob patterns
            for file in $path; do
                if [ -e "$file" ]; then
                    log_info "Removing $file..."
                    rm -rf "$file"
                fi
            done
        else
            if [ -e "$path" ]; then
                log_info "Removing $path..."
                rm -rf "$path"
            fi
        fi
    done
}

# Main uninstallation process
main() {
    echo -e "${BLUE}Auto-Team Uninstallation Script${NC}"
    echo -e "${BLUE}===============================${NC}"
    echo ""
    
    # Check if installed
    if ! check_installation; then
        # Still look for binary files
        local found_locations
        mapfile -t found_locations < <(find_installations)
        
        if [ ${#found_locations[@]} -eq 0 ]; then
            log_warning "No $BINARY_NAME installations found"
            echo ""
            echo "If you believe $BINARY_NAME is installed elsewhere,"
            echo "please remove it manually."
            exit 0
        fi
        
        log_info "Found binary files even though $BINARY_NAME is not in PATH"
    fi
    
    # Find all installations
    local found_locations
    mapfile -t found_locations < <(find_installations)
    
    if [ ${#found_locations[@]} -eq 0 ]; then
        log_warning "No binary files found in standard locations"
        log_info "If $BINARY_NAME is installed elsewhere, please remove it manually"
        exit 0
    fi
    
    echo "Found installations:"
    for location in "${found_locations[@]}"; do
        echo "  - $location"
    done
    echo ""
    
    # Confirmation
    echo -n "Are you sure you want to uninstall $BINARY_NAME? [y/N]: "
    read -r response
    case "$response" in
        [yY][eE][sS]|[yY])
            log_info "Proceeding with uninstallation..."
            ;;
        *)
            log_info "Uninstallation cancelled."
            exit 0
            ;;
    esac
    
    echo ""
    
    # Remove binaries
    if remove_binaries "${found_locations[@]}"; then
        log_success "Binary files removed successfully"
    fi
    
    # Clean up configuration files
    cleanup_files
    
    # Verify removal
    echo ""
    if ! command -v "$BINARY_NAME" >/dev/null 2>&1; then
        log_success "Uninstallation completed successfully!"
        log_info "$BINARY_NAME has been removed from your system"
    else
        log_warning "Warning: $BINARY_NAME is still available in PATH"
        log_info "You may need to restart your shell or check for additional installations"
    fi
    
    echo ""
    log_info "Thank you for using Auto-Team!"
}

# Show usage
usage() {
    echo "Auto-Team Uninstallation Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help              Show this help message"
    echo ""
    echo "This script will:"
    echo "  - Remove autoteam binary from standard installation directories"  
    echo "  - Clean up configuration files and temporary data"
    echo "  - Verify complete removal"
}

# Parse command line arguments
case "${1:-}" in
    -h|--help)
        usage
        exit 0
        ;;
    "")
        # No arguments, proceed with uninstallation
        ;;
    *)
        log_error "Unknown option: $1"
        usage
        exit 1
        ;;
esac

# Run main function
main