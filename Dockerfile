# Stage 1: Build Frontend
FROM node:20-alpine AS web-builder
WORKDIR /app/admin-web
RUN npm install -g pnpm --registry=https://registry.npmmirror.com
COPY admin-web/package.json admin-web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --registry=https://registry.npmmirror.com
COPY admin-web/ ./
COPY VERSION /app/VERSION
RUN pnpm run build

# Stage 2: Build Backend
FROM golang:1.24-alpine AS server-builder
WORKDIR /app

ARG VERSION=""
ARG BUILD_TIME=""
ARG GIT_COMMIT=""

COPY admin-server/go.mod admin-server/go.sum ./admin-server/
RUN cd admin-server && go env -w GOPROXY=https://goproxy.cn,direct && go mod download

COPY admin-server/ ./admin-server/
COPY VERSION ./VERSION
COPY --from=web-builder /app/admin-web/dist ./admin-server/static/dist

RUN set -e; \
    if [ -z "$VERSION" ]; then \
      if [ -f "VERSION" ]; then \
        VERSION=$(cat VERSION | tr -d '\r\n'); \
      else \
        echo "VERSION file not found" && exit 1; \
      fi; \
    fi; \
    if [ -z "$BUILD_TIME" ]; then \
      BUILD_TIME=$(date +"%Y%m%d.%H%M"); \
    fi; \
    if [ -z "$GIT_COMMIT" ]; then \
      GIT_COMMIT="unknown"; \
    fi; \
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
