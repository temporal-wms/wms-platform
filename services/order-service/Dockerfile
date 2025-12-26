FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy shared module and service go.mod first for better caching
COPY shared/ ./shared/
COPY services/order-service/go.mod services/order-service/go.sum* ./services/order-service/

WORKDIR /app/services/order-service

# Download dependencies
ENV GOWORK=off
RUN go mod download

# Copy source code
COPY services/order-service/ .

# Tidy dependencies and build
RUN go mod tidy

# application
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
RUN adduser -D -g '' appuser
USER appuser

# Default command (can be overridden)
EXPOSE 8001

CMD ["/app/api"]
