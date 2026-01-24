# Dockerfile for cicd-ai-toolkit/runner
# Multi-stage build for minimal image size (~50MB target)

# ============================================
# Stage 1: Builder
# ============================================
FROM golang:1.22.5-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    make

# Set working directory
WORKDIR /src

# Copy go mod files (for better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG BUILD_DATE=
ARG GIT_COMMIT=

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w \
        -X github.com/cicd-ai-toolkit/cicd-runner/pkg/version.Version=${VERSION} \
        -X github.com/cicd-ai-toolkit/cicd-runner/pkg/version.BuildDate=${BUILD_DATE} \
        -X github.com/cicd-ai-toolkit/cicd-runner/pkg/version.GitCommit=${GIT_COMMIT}" \
    -trimpath \
    -o /tmp/cicd-runner \
    ./cmd/cicd-runner

# ============================================
# Stage 2: Distroless Runtime (Production)
# ============================================
FROM gcr.io/distroless/static:nonroot AS production

# Copy CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /tmp/cicd-runner /bin/cicd-runner

# Copy skills (optional - for built-in skills)
COPY skills/ /opt/cicd-ai/skills/

# Set non-root user (distroless:nonroot uses UID 65532)
USER 65532:65532

# Set entrypoint
ENTRYPOINT ["/bin/cicd-runner"]

# Default command
CMD ["--help"]
