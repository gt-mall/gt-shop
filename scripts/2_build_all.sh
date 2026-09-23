#!/bin/bash
set -Eeuo pipefail

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$( cd "${SCRIPT_DIR}/.." && pwd )"
SERVER_DIR="${PROJECT_DIR}/admin-server"
WEB_DIR="${PROJECT_DIR}/admin-web"
MINIAPP_DIR="${PROJECT_DIR}/miniapp"
RELEASE_DIR="${PROJECT_DIR}/release"
RUN_DIR="${RELEASE_DIR}/run"
VERSION_FILE="${PROJECT_DIR}/VERSION"

if [[ ! -f "${VERSION_FILE}" ]]; then
  echo "VERSION file not found: ${VERSION_FILE}" >&2
  exit 1
fi

VERSION="$(tr -d '\r\n' < "${VERSION_FILE}" | xargs)"
if [[ -z "${VERSION}" ]]; then
  echo "VERSION file is empty: ${VERSION_FILE}" >&2
  exit 1
fi

BUILD_TIME="$(date +"%Y%m%d.%H%M")"
GIT_COMMIT="$(git -C "${PROJECT_DIR}" rev-parse --short HEAD 2>/dev/null || echo "unknown")"
if [[ -z "${GIT_COMMIT}" ]]; then
  GIT_COMMIT="unknown"
fi

echo "Building version: GT-Shop v${VERSION} (${BUILD_TIME}-${GIT_COMMIT})"
LDFLAGS="-s -w -X 'mall/admin_core/pkg/version.Version=${VERSION}' -X 'mall/admin_core/pkg/version.BuildTime=${BUILD_TIME}' -X 'mall/admin_core/pkg/version.GitCommit=${GIT_COMMIT}'"

echo "Building frontend..."
(cd "${WEB_DIR}" && pnpm run build)
mkdir -p "${SERVER_DIR}/static/dist"
find "${SERVER_DIR}/static/dist" -mindepth 1 ! -name .gitkeep -delete
cp -R "${WEB_DIR}/dist/." "${SERVER_DIR}/static/dist/"
mkdir -p "${RUN_DIR}/static"
find "${RUN_DIR}/static" -mindepth 1 -delete
cp -R "${WEB_DIR}/dist/." "${RUN_DIR}/static/"

echo "Building backend (with embedded frontend)..."
mkdir -p "${RUN_DIR}"
(cd "${SERVER_DIR}" && go build -trimpath -ldflags "${LDFLAGS}" -o "${RUN_DIR}/mall_server" .)
cp "${SCRIPT_DIR}/server_restart.sh" "${RUN_DIR}/restart.sh"
chmod 755 "${RUN_DIR}/restart.sh"
if [[ "${1:-}" != "--admin-only" && -f "${SERVER_DIR}/.env" ]]; then
  cp "${SERVER_DIR}/.env" "${RUN_DIR}/.env"
fi

if [[ "${1:-}" != "--admin-only" ]]; then
  echo "Building miniapp H5 and WeChat targets..."
  (cd "${MINIAPP_DIR}" && pnpm run build:h5 && pnpm run build:mp-weixin)
fi

echo "Build completed."
echo "Version: GT-Shop v${VERSION} (${BUILD_TIME}-${GIT_COMMIT})"
echo "Backend: ${RUN_DIR}/mall_server"
echo "Frontend: ${RUN_DIR}/static"
echo "Restart script: ${RUN_DIR}/restart.sh"
