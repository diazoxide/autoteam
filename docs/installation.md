# Installation Guide

## Quick Install (Recommended)

```bash
# Install latest version (macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/diazoxide/autoteam/main/scripts/install.sh | bash
```

This script will:
- Install the main `autoteam` binary to `/usr/local/bin/`
- Install worker binaries for all supported platforms to `/opt/autoteam/bin/`
- Install the universal `entrypoint.sh` script
- Build from source if pre-built binaries aren't available

## Manual Installation

### From Releases

Download the appropriate binary for your platform from the [releases page](https://github.com/diazoxide/autoteam/releases):

```bash
# Download and extract for your platform
wget https://github.com/diazoxide/autoteam/releases/latest/download/autoteam-<version>-<os>-<arch>.tar.gz
tar -xzf autoteam-<version>-<os>-<arch>.tar.gz
sudo mv autoteam /usr/local/bin/
```

### Build from Source

Requirements:
- Go 1.22 or later
- Git
- Make

```bash
# Clone repository
git clone https://github.com/diazoxide/autoteam.git
cd autoteam

# Build main binary
make build

# Build worker binaries for all platforms
make build-worker-all

# Install (requires sudo)
make install
```

## Verify Installation

```bash
autoteam --version
```

## Dependencies

### Required Dependencies

AutoTeam requires the following tools to be installed:

- **Docker** - Container runtime for agent isolation
- **Docker Compose** - Container orchestration (installed with Docker Desktop)
- **Git** - For repository cloning and version control
- **GitHub CLI (gh)** - For GitHub API interactions

### Installing Dependencies

**macOS (Homebrew):**
```bash
brew install docker docker-compose git gh
```

**Ubuntu/Debian:**
```bash
# Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# GitHub CLI
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
sudo apt update
sudo apt install gh git
```

**Fedora/RHEL:**
```bash
# Docker
sudo dnf install docker docker-compose git
sudo systemctl enable --now docker
sudo usermod -aG docker $USER

# GitHub CLI
sudo dnf install gh
```

## Platform-Specific Notes

### macOS
- Use Homebrew for dependency management
- Docker Desktop includes Docker Compose
- May need to allow binary execution in System Preferences > Security & Privacy

### Linux
- Add user to docker group: `sudo usermod -aG docker $USER`
- May need to start Docker service: `sudo systemctl enable --now docker`
- Log out and back in after adding to docker group

### Windows (WSL2)
- Install WSL2 with Ubuntu
- Follow Ubuntu installation instructions within WSL2
- Install Docker Desktop for Windows with WSL2 backend

## Troubleshooting

### Permission Errors
```bash
# Fix binary permissions
sudo chmod +x /usr/local/bin/autoteam

# Fix docker group membership
sudo usermod -aG docker $USER
# Log out and back in
```

### Docker Issues
```bash
# Check Docker status
docker --version
docker compose version

# Test Docker access
docker run hello-world
```

### Build Errors
```bash
# Check Go version
go version  # Should be 1.22+

# Clean and rebuild
make clean
make build
```

## Uninstallation

To remove AutoTeam completely:

```bash
# Use the uninstall script
curl -fsSL https://raw.githubusercontent.com/diazoxide/autoteam/main/scripts/uninstall.sh | bash

# Or manually remove
sudo rm -f /usr/local/bin/autoteam
sudo rm -rf /opt/autoteam
rm -rf ~/.autoteam
```

## Next Steps

After installation, proceed to [Configuration](configuration.md) to set up your first AI agent team.