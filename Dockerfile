FROM golang:1.20-alpine as build

WORKDIR /go/src/app

# Copy go mod files first for better caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY main.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static" -w -s' -tags timetzdata -o goburn

FROM alpine:latest

# Install ca-certificates for HTTPS requests to k8s API
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary
COPY --from=build /go/src/app/goburn .

# Use non-root user for security
RUN adduser -D -s /bin/sh goburn
USER goburn

ENTRYPOINT ["./goburn"]
