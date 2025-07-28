# Auto-Team Dockerfile
# Multi-stage build for minimal production image

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Production stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    git \
    docker-cli \
    docker-compose \
    curl \
    bash

# Create non-root user
RUN addgroup -g 1001 -S autoteam && \
    adduser -u 1001 -S autoteam -G autoteam

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/build/autoteam /usr/local/bin/autoteam

# Copy templates and examples
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/examples ./examples

# Create data directories
RUN mkdir -p /app/agents /app/shared && \
    chown -R autoteam:autoteam /app

# Switch to non-root user
USER autoteam

# Set default command
CMD ["autoteam", "--help"]