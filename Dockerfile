FROM golang:1.24.2

RUN apt update
RUN apt install -y  \
    curl  \
    gh  \
    git \
    gnupg \
    ca-certificates \
    build-essential

RUN curl -fsSL https://deb.nodesource.com/setup_current.x | bash - && \
    apt-get install -y nodejs


RUN npm install -g @anthropic-ai/claude-code@1.0.83 && \
    npm install -g @qwen-code/qwen-code@latest && \
    npm install -g @google/gemini-cli

# Download the correct prebuilt github-mcp-server for the current CPU architecture
RUN set -eux; \
    mkdir -p /opt/autoteam/custom/mcp/github/bin; \
    cd /opt/autoteam/custom/mcp/github/bin; \
    arch="$(uname -m)"; \
    case "$arch" in \
      aarch64|arm64) asset_arch=arm64 ;; \
      x86_64|amd64)  asset_arch=x86_64 ;; \
      i386|i686)     asset_arch=i386 ;; \
      *) echo "Unsupported architecture: $arch" >&2; exit 1 ;; \
    esac; \
    version="v0.10.0"; \
    file="github-mcp-server_Linux_${asset_arch}.tar.gz"; \
    url="https://github.com/github/github-mcp-server/releases/download/${version}/${file}"; \
    echo "Downloading ${url}"; \
    curl -fsSL -o "${file}" "${url}"; \
    tar -xzf "${file}"; \
    rm -f "${file}"

