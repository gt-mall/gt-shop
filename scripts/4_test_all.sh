#!/bin/bash
set -Eeuo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$( cd "${SCRIPT_DIR}/.." && pwd )"
SERVER_DIR="${PROJECT_DIR}/admin-server"
WEB_DIR="${PROJECT_DIR}/admin-web"
MINIAPP_DIR="${PROJECT_DIR}/miniapp"

if ! command -v go >/dev/null 2>&1 || ! command -v pnpm >/dev/null 2>&1; then
  echo "需要 go 和 pnpm 命令" >&2
  exit 1
fi

echo "运行后端测试..."
(cd "${SERVER_DIR}" && go test ./...)

echo "运行管理端测试与构建..."
(cd "${WEB_DIR}" && pnpm run test && pnpm run build)

echo "运行小程序测试、类型检查与双端构建..."
(cd "${MINIAPP_DIR}" && pnpm run test && pnpm run type-check && pnpm run build:h5 && pnpm run build:mp-weixin)

echo "验证完成"
