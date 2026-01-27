# Multi-stage Dockerfile for cicd-ai-toolkit
# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev} -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /bin/cicd-runner \
    ./cmd/cicd-runner

# Stage 2: Runtime
FROM alpine:3.19

# Install runtime dependencies
# - git: for git operations
# - bash: for skill scripts
# - curl: for health checks
RUN apk add --no-cache \
    git \
    bash \
    ca-certificates \
    curl

# Create non-root user
RUN addgroup -g 1000 cicd && \
    adduser -D -u 1000 -G cicd cicd

# Set working directory
WORKDIR /workspace

# Copy binary from builder
COPY --from=builder /bin/cicd-runner /usr/local/bin/cicd-runner

# Copy skills directory
COPY --from=builder /src/skills /usr/local/share/skills

# Copy default config
COPY --from=builder /src/configs/.cicd-ai-toolkit.yaml /etc/cicd-ai-toolkit/config.yaml

# Create cache directory
RUN mkdir -p /workspace/.cicd-ai-cache && \
    chown -R cicd:cicd /workspace

# Switch to non-root user
USER cicd

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD cicd-runner --health || exit 1

# Set entrypoint
ENTRYPOINT ["cicd-runner"]
CMD ["--help"]

# Metadata
LABEL org.opencontainers.image.title="cicd-ai-toolkit" \
      org.opencontainers.image.description="AI-powered CI/CD toolkit based on Claude Code" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.source="https://github.com/cicd-ai-toolkit/cicd-runner"
