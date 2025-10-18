# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o server cmd/server/main.go

# Final stage
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone database
COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /usr/local/go/lib/time/zoneinfo.zip

# Copy binary
COPY --from=builder /build/server /server

# Expose port
EXPOSE 8080

# Run
ENTRYPOINT ["/server"]
