# 仅打包运行镜像。请先在本机或 CI 编好产物：
#   cd frontend && npm ci && npm run build
#   cd backend && \
#     CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ../bin/linux/amd64/submerge . && \
#     CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ../bin/linux/arm64/submerge .
#
# 需要的产物（构建上下文为 docker-submerge/build）：
#   bin/linux/amd64/submerge
#   bin/linux/arm64/submerge
#   frontend/dist/submerge/browser/

FROM alpine:3.21

ARG TARGETPLATFORM
ARG APP_VERSION=0.1.0
LABEL org.opencontainers.image.title="submerge" \
      org.opencontainers.image.description="Clash subscription merge panel" \
      org.opencontainers.image.version="${APP_VERSION}"

# 仅业务 ENV；路径不设 → WORKDIR=/app 下默认 ./data ./log ./frontend/...
ENV TZ=Asia/Shanghai \
    APP_ENV=production \
    HTTP_ADDR=:8080 \
    LOG_OUTPUT=both

WORKDIR /app

COPY bin/${TARGETPLATFORM}/submerge /app/submerge
COPY frontend/dist/submerge/browser /app/frontend/dist/submerge/browser

RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 10001 submerge \
 && mkdir -p /app/data /app/log \
 && chmod +x /app/submerge \
 && chown -R submerge:submerge /app

USER submerge

EXPOSE 8080

# 持久化：库 + 日志（compose 再挂命名卷）
VOLUME ["/app/data", "/app/log"]

ENTRYPOINT ["/app/submerge"]
