# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy dependency definition
COPY go.mod ./

# Copy source code
COPY main.go ./
COPY pkg/ ./pkg/

# Build static binary for Linux
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server main.go

# Production stage - scratch minimal image
FROM scratch

WORKDIR /

# Copy CA certificates for HTTPS outbound connections
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /app/server /server

# Expose default port
EXPOSE 8080

# Cloud Run injects PORT env variable automatically
ENV PORT=8080

CMD ["/server"]
