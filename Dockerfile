# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /build/cicd-runner \
    ./cmd/cicd-runner

# Final stage
FROM alpine:latest

RUN apk add --no-cache git

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/cicd-runner /app/cicd-runner

# Copy skills
COPY --from=builder /build/skills /app/skills

# Create cache directory
RUN mkdir -p /app/cache

ENV PATH="/app:${PATH}"
ENV CICD_SKILLS_PATH=/app/skills
ENV CICD_CACHE_PATH=/app/cache

ENTRYPOINT ["/app/cicd-runner"]
CMD ["--help"]
