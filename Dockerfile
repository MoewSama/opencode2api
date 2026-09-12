FROM alpine:latest

ARG VERSION=dev

RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates

WORKDIR /app

COPY bin/opencode2api /app/opencode2api
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint
COPY config.example.json /app/config.example.json

RUN chmod +x /app/opencode2api /usr/local/bin/docker-entrypoint \
    && mkdir -p /var/lib/opencode2api

ENV CONFIG_PATH=/var/lib/opencode2api/config.json \
    LISTEN_ADDRESS=0.0.0.0:8080 \
    VERSION=${VERSION}

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/docker-entrypoint"]
