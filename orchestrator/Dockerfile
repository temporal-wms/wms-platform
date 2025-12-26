FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy shared module and service go.mod first for better caching
COPY shared/ ./shared/
COPY orchestrator/go.mod orchestrator/go.sum* ./orchestrator/

WORKDIR /app/orchestrator

# Download dependencies
ENV GOWORK=off
RUN go mod download

# Copy source code
COPY orchestrator/ .

# Tidy dependencies and build
RUN go mod tidy

# Worker binary only (no API for orchestrator)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/worker ./cmd/worker

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/bin/worker /app/worker

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set ownership
RUN chown -R appuser:appuser /app

USER appuser

# No EXPOSE or HEALTHCHECK - Temporal manages worker lifecycle

CMD ["/app/worker"]
