# Stage 1: Build Frontend
FROM node:20-alpine AS web-builder
WORKDIR /app/admin-web
RUN npm install -g pnpm --registry=https://registry.npmmirror.com
COPY admin-web/package.json admin-web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --registry=https://registry.npmmirror.com
COPY admin-web/ ./
COPY VERSION /app/VERSION
ARG BUILD_TIME=""
ARG GIT_COMMIT=""
RUN set -eu; \
    build_time="${BUILD_TIME:-$(date +"%Y%m%d.%H%M")}"; \
    git_commit="${GIT_COMMIT:-unknown}"; \
    printf '%s' "$build_time" > /app/BUILD_TIME; \
    printf '%s' "$git_commit" > /app/GIT_COMMIT; \
    VITE_BUILD_TIME="$build_time" BUILD_TIME="$build_time" \
      VITE_GIT_COMMIT="$git_commit" GIT_COMMIT="$git_commit" pnpm run build

# Stage 2: Build Backend
FROM golang:1.24-alpine AS server-builder
WORKDIR /app

COPY admin-server/go.mod admin-server/go.sum ./admin-server/
RUN cd admin-server && go env -w GOPROXY=https://goproxy.cn,direct && go mod download

COPY admin-server/ ./admin-server/
COPY VERSION ./VERSION
COPY --from=web-builder /app/BUILD_TIME ./BUILD_TIME
COPY --from=web-builder /app/GIT_COMMIT ./GIT_COMMIT
COPY --from=web-builder /app/admin-web/dist ./admin-server/static/dist

RUN set -e; \
    VERSION=$(tr -d '\r\n' < VERSION); \
    BUILD_TIME=$(cat BUILD_TIME); \
    GIT_COMMIT=$(cat GIT_COMMIT); \
    cd admin-server && \
    CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X 'mall/admin_core/pkg/version.Version=${VERSION}' -X 'mall/admin_core/pkg/version.BuildTime=${BUILD_TIME}' -X 'mall/admin_core/pkg/version.GitCommit=${GIT_COMMIT}'" \
      -o /app/mall_server .

# Stage 3: Runtime
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=server-builder /app/mall_server /app/mall_server
COPY --from=web-builder /app/admin-web/dist /app/static
EXPOSE 8100
CMD ["/app/mall_server"]
