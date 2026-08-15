# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend-builder
WORKDIR /src/fronted

COPY fronted/package.json fronted/package-lock.json ./
RUN npm ci

COPY fronted/ ./
COPY VERSION /src/VERSION
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS go-builder
WORKDIR /src/backend

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION
ARG UPDATE_PUBLIC_KEY_BASE64=""

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
COPY VERSION ./VERSION
COPY --from=frontend-builder /src/backend/internal/webui/dist ./internal/webui/dist

RUN version="$(tr -d '[:space:]' < VERSION)" \
    && test -n "$version" \
    && test "$VERSION" = "$version" \
    && CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
    go build -trimpath -buildvcs=false \
    -ldflags="-s -w -buildid= -X github.com/submerge/submerge/backend/version.Value=${version} -X github.com/submerge/submerge/backend/internal/updater.PublicKeyBase64=${UPDATE_PUBLIC_KEY_BASE64}" \
    -o /out/submerge .

FROM alpine:3.22

ARG VERSION
ARG VCS_REF=unknown
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="SubMerge" \
      org.opencontainers.image.description="Self-hosted Clash subscription manager" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.source="https://github.com/syueya/subMerge"

RUN apk add --no-cache ca-certificates su-exec tzdata \
    && addgroup -g 10001 -S submerge \
    && adduser -u 10001 -S -D -H -G submerge submerge \
    && mkdir -p /app/data /app/log /app/defaults/geo /app/runtime \
    && chown -R submerge:submerge /app

WORKDIR /app
COPY --from=go-builder --chmod=0755 /out/submerge /app/submerge
COPY --chmod=0755 docker/entrypoint.sh /usr/local/bin/submerge-entrypoint

ENV APP_ENV=production \
    HTTP_ADDR=:8080 \
    TZ=Asia/Shanghai \
    SUBMERGE_UPDATE_DIR=/app/runtime \
    SUBMERGE_RESTART_MODE=exit

VOLUME ["/app/data", "/app/log", "/app/defaults/geo", "/app/runtime"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -q -T 4 -O /dev/null http://127.0.0.1:8080/api/health || exit 1

ENTRYPOINT ["/usr/local/bin/submerge-entrypoint"]
