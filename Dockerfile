# Build stage - Pin patch version for reproducibility
FROM golang:1.24.8-alpine3.21 AS builder

# Install ca-certificates and tzdata in builder
RUN apk add --no-cache \
    ca-certificates \
    tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build server binary with security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a \
    -installsuffix cgo \
    -ldflags '-w -s -extldflags "-static"' \
    -trimpath \
    -o server \
    cmd/server/main.go

# Build tiny healthcheck binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a \
    -installsuffix cgo \
    -ldflags '-w -s -extldflags "-static"' \
    -trimpath \
    -o healthcheck \
    cmd/healthcheck/main.go

# Final stage - Use distroless or minimal alpine for non-root user
FROM alpine:3.21.0

# Install ca-certificates and tzdata
RUN apk add --no-cache \
    ca-certificates \
    tzdata && \
    # Create non-root user and group
    addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup && \
    # Create directory for the application
    mkdir -p /app && \
    chown -R appuser:appgroup /app

# Copy CA certificates and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binaries from builder
COPY --from=builder --chown=appuser:appgroup /build/server /app/server
COPY --from=builder --chown=appuser:appgroup /build/healthcheck /app/healthcheck

# Switch to non-root user
USER appuser:appgroup

# Set working directory
WORKDIR /app

# Expose port (non-privileged port)
EXPOSE 8080

# Add healthcheck using dedicated probe binary
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/healthcheck"]

# Drop all capabilities and run with read-only root filesystem
# These will be enforced at runtime via security context in k8s/docker run

# Run the server (no args - server starts in HTTP mode by default)
ENTRYPOINT ["/app/server"]
