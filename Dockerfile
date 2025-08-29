# Build stage
FROM golang:1.20-alpine AS build

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy all source files
COPY *.go ./

# Build with optimizations - use native architecture detection
RUN CGO_ENABLED=0 GOOS=linux \
    go build -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o goburn .

# Final stage - use alpine for better compatibility
FROM alpine:latest

# Install ca-certificates for HTTPS requests to k8s API
RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -s /bin/sh goburn

WORKDIR /app

# Copy the binary with correct permissions
COPY --from=build /app/goburn /app/goburn
RUN chmod +x /app/goburn

# Use non-root user
USER goburn

ENTRYPOINT ["/app/goburn"]
