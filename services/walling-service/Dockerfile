FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy shared module and service go.mod first for better caching
COPY shared/ ./shared/
COPY services/walling-service/go.mod services/walling-service/go.sum* ./services/walling-service/

WORKDIR /app/services/walling-service

# Download dependencies
ENV GOWORK=off
RUN go mod download

# Copy source code
COPY services/walling-service/ .

# Tidy dependencies and build
RUN go mod tidy

# API application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/api ./cmd/api

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/bin/api /app/api

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set ownership
RUN chown -R appuser:appuser /app

USER appuser

# Expose port
EXPOSE 8017

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8017/health || exit 1

CMD ["/app/api"]
