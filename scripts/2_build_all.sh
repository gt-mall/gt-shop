#!/bin/bash
set -e

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$( cd "${SCRIPT_DIR}/.." && pwd )"
SERVER_DIR="${PROJECT_DIR}/admin-server"
WEB_DIR="${PROJECT_DIR}/admin-web"
RELEASE_DIR="${PROJECT_DIR}/release"
RUN_DIR="${RELEASE_DIR}/run"

echo "Building backend..."
mkdir -p "${RUN_DIR}"
(cd "${SERVER_DIR}" && go build -trimpath -o "${RUN_DIR}/mall_server" .)
if [[ -f "${SERVER_DIR}/.env" ]]; then
  cp "${SERVER_DIR}/.env" "${RUN_DIR}/.env"
fi

echo "Building frontend..."
(cd "${WEB_DIR}" && pnpm run build)

echo "Build completed."
echo "Backend: ${RUN_DIR}/mall_server"
