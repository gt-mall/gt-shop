#!/bin/bash
set -Eeuo pipefail

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$( cd "${SCRIPT_DIR}/.." && pwd )"
SERVER_DIR="${PROJECT_DIR}/admin-server"

if ! command -v go >/dev/null 2>&1; then
  echo "需要 go 命令" >&2
  exit 1
fi

echo "Generating Swagger..."
"${SERVER_DIR}/scripts/generate-swagger.sh"
echo "Swagger generation completed."
