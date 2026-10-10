FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 \
    GOOS="${TARGETOS:-linux}" \
    GOARCH="${TARGETARCH:-amd64}" \
    go build \
      -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/opencode2api ./cmd/opencode2api

FROM alpine:latest

# Runs as root by design. The container is a single-process appliance whose
# state dir is typically a bind mount the operator owns on the host, so dropping
# to a build-time uid only adds an id-mapping failure mode without protecting
# anything the operator cannot already reach. See docker-entrypoint.sh, which
# deliberately does not re-exec through su-exec.
RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /app /var/lib/opencode2api

COPY --from=builder /out/opencode2api /usr/local/bin/opencode2api
COPY --chmod=0755 docker-entrypoint.sh /usr/local/bin/docker-entrypoint
COPY config.example.json /app/config.example.json
COPY LICENSE /usr/share/licenses/opencode2api/LICENSE
LABEL org.opencontainers.image.licenses="GPL-3.0-or-later"

# LISTEN_ADDRESS is intentionally unset: config.json is authoritative. Export
# it to override the configured listen address.
ENV CONFIG_PATH=/var/lib/opencode2api/config.json \
    CONFIG_SEED_PATH= \
    LISTEN_ADDRESS= \
    STATE_DIR=/var/lib/opencode2api

WORKDIR /app

# The management console is served from this port under /admin.
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1

# Explicit so a base-image USER change cannot silently alter the contract.
USER root

ENTRYPOINT ["/usr/local/bin/docker-entrypoint"]
