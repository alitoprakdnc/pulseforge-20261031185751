# syntax=docker/dockerfile:1.4

############################
# Builder Stage
############################
FROM golang:1.22-alpine AS builder

# Install required packages for building
RUN apk add --no-cache git ca-certificates && update-ca-certificates

# Set working directory
WORKDIR /src

# Cache go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source tree
COPY . .

# Build the binary
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}
RUN go build -ldflags="-s -w" -o /go/bin/pulseforge ./main.go

############################
# Runtime Stage
############################
FROM alpine:3.19 AS runtime

# Install runtime certificates
RUN apk add --no-cache ca-certificates && update-ca-certificates

# Create a non‑root user and group
RUN addgroup -S appgroup && adduser -S -G appgroup appuser

# Set working directory
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /go/bin/pulseforge /usr/local/bin/pulseforge

# Expose the HTTP port used by the API
EXPOSE 8080

# Default environment variables (can be overridden at runtime)
ENV PORT=8080 \
    GIN_MODE=release

# Run as non‑root user
USER appuser

# Health‑check endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:${PORT}/api/health || exit 1

# Entrypoint
ENTRYPOINT ["pulseforge"]