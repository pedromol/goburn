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

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
    go build -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o goburn .

# Final stage - use distroless for minimal size and security
FROM gcr.io/distroless/static-debian11:nonroot

# Copy ca-certificates and timezone data
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=build /app/goburn /goburn

# Use non-root user (already set in distroless:nonroot)
ENTRYPOINT ["/goburn"]
