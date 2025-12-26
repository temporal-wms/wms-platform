FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy shared module and service go.mod first for better caching
COPY shared/ ./shared/
COPY services/packing-service/go.mod services/packing-service/go.sum* ./services/packing-service/

WORKDIR /app/services/packing-service

# Download dependencies
ENV GOWORK=off
RUN go mod download

# Copy source code
COPY services/packing-service/ .

# Tidy dependencies and build
RUN go mod tidy

# Build both API and Worker binaries
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/worker ./cmd/worker

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binaries from builder
COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/bin/worker /app/worker

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set ownership
RUN chown -R appuser:appuser /app

USER appuser

# Expose port (for API)
EXPOSE 8006

# Health check (for API mode)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8006/health || exit 1

# Default to API server (can override with /app/worker)
CMD ["/app/api"]
