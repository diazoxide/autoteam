#!/bin/sh

# AutoTeam Universal Container Entrypoint
# This script handles platform detection and worker binary execution

set -e  # Exit on any error

echo "=== AutoTeam Agent Starting ==="
echo "Agent: ${AGENT_NAME:-unknown}"
echo "Repositories Include: ${REPOSITORIES_INCLUDE:-none}"
echo "Repositories Exclude: ${REPOSITORIES_EXCLUDE:-none}"
echo "Current working directory: $(pwd)"
echo "Available environment variables:"
printenv | grep -E "(GH_|GITHUB_|AGENT_|TEAM_|CHECK_|INSTALL_|WORKER_|AUTOTEAM_|MAX_|DEBUG|REPOSITORIES_)" | sort || true
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
WORKER_BINARY="/opt/autoteam/bin/autoteam-worker-${PLATFORM}"

echo "📦 Detected platform: ${PLATFORM}"
echo "🔍 Looking for worker binary: ${WORKER_BINARY}"

# Check if system-installed worker binary exists
if [ -f "$WORKER_BINARY" ] && [ -x "$WORKER_BINARY" ]; then
  echo "📥 Using system-installed worker binary..."
  ls -la "$WORKER_BINARY"
  cp "$WORKER_BINARY" /tmp/autoteam-worker
  chmod +x /tmp/autoteam-worker
  echo "✅ System binary copied successfully"
else
  echo "❌ System worker binary not found for platform ${PLATFORM}"
  echo "💡 Run 'autoteam --install-workers' on the host to install worker binaries"
  echo "📁 Expected location: ${WORKER_BINARY}"
  exit 1
fi

# Verify binary is executable
if [ -x "/tmp/autoteam-worker" ]; then
  echo "✅ Binary is executable"
  /tmp/autoteam-worker --version || echo "Version check failed, but continuing..."
else
  echo "❌ Binary is not executable"
  ls -la /tmp/autoteam-worker
  exit 1
fi

echo "=== Starting autoteam-worker ==="
echo "Configuration will be read from environment variables"

# Execute the worker
exec /tmp/autoteam-worker
