FROM golang:1.24.5-alpine AS builder

WORKDIR /app

# Set Go environment variables for optimal caching
ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/go/pkg/mod
ENV CGO_ENABLED=0

COPY go.mod go.sum ./

# Use cache mount for Go modules to persist downloads across builds
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

COPY . .

# Use cache mounts for both Go modules and build cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags '-extldflags "-static" -w -s' -o /app/cloudflared-mapper ./cmd

####################

FROM scratch AS rootfs

COPY --from=builder /app/cloudflared-mapper /cloudflared-mapper
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=cloudflare/cloudflared:latest /usr/local/bin/cloudflared /usr/local/bin/cloudflared

USER 65535:65535
