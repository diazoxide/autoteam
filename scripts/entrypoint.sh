#!/bin/sh

# AutoTeam Universal Container Entrypoint
# This script handles platform detection and entrypoint binary execution

set -e  # Exit on any error

echo "=== AutoTeam Agent Starting ==="
echo "Agent: ${AGENT_NAME:-unknown}"
echo "Repositories Include: ${REPOSITORIES_INCLUDE:-none}"
echo "Repositories Exclude: ${REPOSITORIES_EXCLUDE:-none}"
echo "Current working directory: $(pwd)"
echo "Available environment variables:"
printenv | grep -E "(GH_|GITHUB_|AGENT_|TEAM_|CHECK_|INSTALL_|ENTRYPOINT_|MAX_|DEBUG|REPOSITORIES_)" | sort || true
echo "================================="

# Detect container platform
CONTAINER_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
CONTAINER_ARCH=$(uname -m)

# Normalize architecture names
case "$CONTAINER_ARCH" in
  x86_64)
    CONTAINER_ARCH="amd64"
    ;;
  aarch64)
    CONTAINER_ARCH="arm64"
    ;;
  armv7l)
    CONTAINER_ARCH="arm"
    ;;
esac

PLATFORM="${CONTAINER_OS}-${CONTAINER_ARCH}"
ENTRYPOINT_BINARY="/opt/autoteam/entrypoints/autoteam-entrypoint-${PLATFORM}"

echo "📦 Detected platform: ${PLATFORM}"
echo "🔍 Looking for entrypoint binary: ${ENTRYPOINT_BINARY}"

# Check if system-installed entrypoint binary exists
if [ -f "$ENTRYPOINT_BINARY" ] && [ -x "$ENTRYPOINT_BINARY" ]; then
  echo "📥 Using system-installed entrypoint binary..."
  ls -la "$ENTRYPOINT_BINARY"
  cp "$ENTRYPOINT_BINARY" /tmp/autoteam-entrypoint
  chmod +x /tmp/autoteam-entrypoint
  echo "✅ System binary copied successfully"
else
  echo "❌ System entrypoint binary not found for platform ${PLATFORM}"
  echo "💡 Run 'autoteam --install-entrypoints' on the host to install entrypoint binaries"
  echo "📁 Expected location: ${ENTRYPOINT_BINARY}"
  exit 1
fi

# Verify binary is executable
if [ -x "/tmp/autoteam-entrypoint" ]; then
  echo "✅ Binary is executable"
  /tmp/autoteam-entrypoint --version || echo "Version check failed, but continuing..."
else
  echo "❌ Binary is not executable"
  ls -la /tmp/autoteam-entrypoint
  exit 1
fi

echo "=== Starting autoteam-entrypoint ==="
echo "Configuration will be read from environment variables"

# Execute the entrypoint
exec /tmp/autoteam-entrypoint
