# syntax=docker.io/docker/dockerfile:1.9

FROM golang:1.23-bookworm as builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY *.go .
ARG APP_VERSION='0.0.0-dev'
ARG APP_REVISION='0000000000000000000000000000000000000000'
RUN CGO_ENABLED=0 go build -ldflags="-s -X main.version=${APP_VERSION} -X main.revision=${APP_REVISION}"

# NB we use the bookworm-slim (instead of scratch) image so we can enter the container to execute bash etc.
FROM debian:bookworm-slim
RUN <<EOF
#!/bin/bash
set -euxo pipefail
apt-get update
apt-get install -y --no-install-recommends \
    wget \
    openssl \
    iproute2 \
    bind9-dnsutils \
    ca-certificates
rm -rf /var/lib/apt/lists/*
EOF
WORKDIR /
# see https://docs.aws.amazon.com/documentdb/latest/developerguide/connect_programmatically.html#connect_programmatically-tls_enabled
ADD --chmod=444 https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem .
COPY --from=builder /app/aws-docdb-example .
EXPOSE 8000
# NB 65534:65534 is the uid:gid of the nobody:nogroup user:group.
# NB we use a numeric uid:gid to easy the use in kubernetes securityContext.
#    k8s will only be able to infer the runAsUser and runAsGroup values when
#    the USER intruction has a numeric uid:gid. otherwise it will fail with:
#       kubelet Error: container has runAsNonRoot and image has non-numeric
#       user (nobody), cannot verify user is non-root
USER 65534:65534
ENTRYPOINT ["/aws-docdb-example"]
ARG APP_SOURCE_URL
ARG APP_REVISION
LABEL org.opencontainers.image.source="$APP_SOURCE_URL"
LABEL org.opencontainers.image.revision="$APP_REVISION"
LABEL org.opencontainers.image.description="AWS DocumentDB example application using Go"
LABEL org.opencontainers.image.licenses="MIT"
