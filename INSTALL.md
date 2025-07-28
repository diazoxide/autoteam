# Auto-Team Installation Guide

Multiple installation methods available for macOS and Linux systems.

## Quick Install (Recommended)

### Using Installation Script

```bash
# Install latest version
curl -fsSL https://raw.githubusercontent.com/diazoxide/auto-team/main/scripts/install.sh | bash

# Install specific version
curl -fsSL https://raw.githubusercontent.com/diazoxide/auto-team/main/scripts/install.sh | bash -s -- -v 1.0.0

# Install to custom directory
curl -fsSL https://raw.githubusercontent.com/diazoxide/auto-team/main/scripts/install.sh | bash -s -- -d ~/.local/bin
```

## Manual Installation

### Download Pre-built Binaries

1. **Download the appropriate binary for your platform:**

   **macOS (Intel):**

   ```bash
   curl -LO https://github.com/diazoxide/auto-team/releases/latest/download/autoteam-latest-darwin-amd64.tar.gz
   ```

   **macOS (Apple Silicon):**

   ```bash
   curl -LO https://github.com/diazoxide/auto-team/releases/latest/download/autoteam-latest-darwin-arm64.tar.gz
   ```

   **Linux (x86_64):**

   ```bash
   curl -LO https://github.com/diazoxide/auto-team/releases/latest/download/autoteam-latest-linux-amd64.tar.gz
   ```

   **Linux (ARM64):**

   ```bash
   curl -LO https://github.com/diazoxide/auto-team/releases/latest/download/autoteam-latest-linux-arm64.tar.gz
   ```

   **Linux (32-bit):**

   ```bash
   curl -LO https://github.com/diazoxide/auto-team/releases/latest/download/autoteam-latest-linux-386.tar.gz
   ```

   **Linux (ARM):**

   ```bash
   curl -LO https://github.com/diazoxide/auto-team/releases/latest/download/autoteam-latest-linux-arm.tar.gz
   ```

2. **Extract and install:**
   ```bash
   # Extract the archive
   tar -xzf autoteam-*.tar.gz
   
   # Move to installation directory
   sudo cp autoteam-*/autoteam /usr/local/bin/
   
   # Make executable
   sudo chmod +x /usr/local/bin/autoteam
   
   # Verify installation
   autoteam --version
   ```

## Build from Source

### Prerequisites

- Go 1.19 or later
- Git
- Make

### Build Steps

```bash
# Clone the repository
git clone https://github.com/diazoxide/auto-team.git
cd auto-team

# Build for current platform
make build

# Build for all platforms
make build-all

# Install to system
make install

# Create distribution packages
make package
```

### Development Build

```bash
# Quick development build
make dev

# Run tests
make test

# Format and lint
make check
```

## Package Managers

### Homebrew (macOS)

```bash
# Add tap (if available)
brew tap diazoxide/auto-team

# Install
brew install autoteam
```

### Snap (Linux)

```bash
# Install from Snap Store
sudo snap install autoteam
```

### APT (Debian/Ubuntu)

```bash
# Add repository
curl -fsSL https://diazoxide.github.io/auto-team/gpg.key | sudo apt-key add -
echo "deb https://diazoxide.github.io/auto-team/apt stable main" | sudo tee /etc/apt/sources.list.d/autoteam.list

# Install
sudo apt update
sudo apt install autoteam
```

## Installation Verification

After installation, verify that `autoteam` is working correctly:

```bash
# Check version
autoteam --version

# View help
autoteam --help

# Initialize a sample configuration
autoteam init

# Generate Docker Compose files
autoteam generate
```

## Supported Platforms

| OS | Architecture | Status |
|---|---|---|
| macOS | Intel (amd64) | ✅ Supported |
| macOS | Apple Silicon (arm64) | ✅ Supported |
| Linux | x86_64 (amd64) | ✅ Supported |
| Linux | ARM64 | ✅ Supported |
| Linux | 32-bit (386) | ✅ Supported |
| Linux | ARM | ✅ Supported |

## Dependencies

### Runtime Dependencies

- Docker (for running agents)
- Docker Compose (for orchestration)
- Git (for repository operations)

### Agent Dependencies

- GitHub CLI (`gh`) - Auto-installed in containers
- Claude Code - Auto-installed in containers
- Node.js - Provided by Docker image

## Post-Installation Setup

1. **Configure GitHub Tokens:**
   ```bash
   export DEVELOPER_GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
   export REVIEWER_GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
   ```

2. **Initialize Configuration:**
   ```bash
   autoteam init
   ```

3. **Edit Configuration:**
   ```bash
   # Edit autoteam.yaml to match your repository and requirements
   nano autoteam.yaml
   ```

4. **Deploy Your Team:**
   ```bash
   autoteam up
   ```

## Uninstallation

### Using Uninstall Script

```bash
curl -fsSL https://raw.githubusercontent.com/diazoxide/auto-team/main/scripts/uninstall.sh | bash
```

### Manual Uninstall

```bash
# Remove binary
sudo rm /usr/local/bin/autoteam

# Remove configuration files
rm -rf ~/.autoteam ~/.config/autoteam
```

## Troubleshooting

### Common Issues

1. **Permission Denied:**
   ```bash
   # Fix permissions
   sudo chmod +x /usr/local/bin/autoteam
   ```

2. **Command Not Found:**
   ```bash
   # Add to PATH
   echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
   source ~/.bashrc
   ```

3. **Old Version:**
   ```bash
   # Check which autoteam is being used
   which autoteam
   
   # Remove old versions
   sudo rm $(which autoteam)
   ```

### Get Help

- **Issues:** [GitHub Issues](https://github.com/diazoxide/auto-team/issues)
- **Documentation:** [README.md](README.md)
- **Examples:** [examples/](examples/)

## System Requirements

### Minimum Requirements
- **OS:** macOS 10.15+ or Linux (kernel 3.10+)
- **Memory:** 512MB RAM
- **Disk:** 100MB free space
- **Network:** Internet connection for GitHub API

### Recommended Requirements
- **Memory:** 2GB RAM
- **Disk:** 1GB free space
- **Docker:** 4GB RAM allocated to Docker

## Security Considerations

1. **GitHub Tokens:** Use tokens with minimal required permissions
2. **Container Security:** Regularly update Docker images
3. **Network Security:** Consider firewall rules for container networking
4. **Audit Logs:** Monitor GitHub API usage and agent activity

## Updates

### Automatic Updates
Auto-team includes an update checker that notifies you of new versions.

### Manual Updates
```bash
# Using installation script
curl -fsSL https://raw.githubusercontent.com/diazoxide/auto-team/main/scripts/install.sh | bash -s -- -f

# Using package managers
brew upgrade autoteam  # Homebrew
sudo apt update && sudo apt upgrade autoteam  # APT
```

## License

This software is licensed under the MIT License. See [LICENSE](LICENSE) for details.