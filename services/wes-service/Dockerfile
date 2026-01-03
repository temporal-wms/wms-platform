FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./
COPY ../../shared ./shared

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the API server and worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /wes-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /wes-worker ./cmd/worker

# API image
FROM alpine:3.19 AS api

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /wes-api .

EXPOSE 8016

CMD ["./wes-api"]

# Worker image
FROM alpine:3.19 AS worker

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /wes-worker .

CMD ["./wes-worker"]
