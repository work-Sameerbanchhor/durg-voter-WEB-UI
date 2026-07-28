# Build stage
FROM golang:1.24-bookworm AS builder

WORKDIR /app/backend

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Copy dependency files
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source code
COPY backend/main.go ./
COPY backend/pkg/ ./pkg/

# Build optimized production binary with DuckDB CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /app/server main.go

# Production stage
FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /app/server /app/server
COPY backend/database/ /app/backend/database/
COPY frontend/ /app/frontend/

# Expose HTTP port
EXPOSE 7860

ENV PORT=7860
ENV ENVIRONMENT=production
ENV DB_PATH=/app/backend/database/durg_voters.duckdb

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:7860/api/v1/health || exit 1

CMD ["/app/server"]
