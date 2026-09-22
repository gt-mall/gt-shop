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
(cd "${SERVER_DIR}" && go build -trimpath -o "${RUN_DIR}/mall_server" .)
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
echo "Backend: ${RUN_DIR}/mall_server"
echo "Frontend: ${RUN_DIR}/static"
echo "Restart script: ${RUN_DIR}/restart.sh"
