#!/bin/bash
set -Eeuo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$( cd "${SCRIPT_DIR}/.." && pwd )"
SERVER_DIR="${PROJECT_DIR}/admin-server"
WEB_DIR="${PROJECT_DIR}/admin-web"

if ! command -v go >/dev/null 2>&1 || ! command -v pnpm >/dev/null 2>&1; then
  echo "需要 go 和 pnpm 命令" >&2
  exit 1
fi

echo "运行后端测试..."
(cd "${SERVER_DIR}" && go test ./...)

echo "检查前端构建..."
(cd "${WEB_DIR}" && pnpm run build)

echo "验证完成"
