#!/bin/bash
set -Eeuo pipefail

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$( cd "${SCRIPT_DIR}/.." && pwd )"
SERVER_DIR="${PROJECT_DIR}/admin-server"
ENV_FILE="${SERVER_DIR}/.env"

if ! command -v go >/dev/null 2>&1; then
  echo "需要 go 命令" >&2
  exit 1
fi

if [[ ! -d "${SERVER_DIR}" ]]; then
  echo "未找到后端目录: ${SERVER_DIR}" >&2
  exit 1
fi

if [[ -f "${ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1091
  . "${ENV_FILE}"
  set +a
fi

ACTION="${1:-init}"

case "${ACTION}" in
  init|seed|align|create-admin)
    ;;
  *)
    echo "未知命令: ${ACTION}" >&2
    echo "用法: $0 [init|seed|align|create-admin]" >&2
    echo "  init         - 执行全部迁移与种子数据（适用于全新/空数据库初始化）" >&2
    echo "  seed         - 仅执行种子数据（幂等注入默认角色、分组、基础配置）" >&2
    echo "  align        - 验证现有开发库结构，并将 schema_migrations 对齐至基线后执行种子数据" >&2
    echo "  create-admin - 创建首个后台超级管理员（需环境变量 ADMIN_USERNAME 和 ADMIN_PASSWORD）" >&2
    exit 1
    ;;
esac

echo "=========================================="
echo "执行数据库命令: ${ACTION}"
echo "环境文件: ${ENV_FILE}"
echo "=========================================="

(cd "${SERVER_DIR}" && go run . "${ACTION}")

echo "=========================================="
echo "数据库命令 ${ACTION} 执行完成！"
echo "=========================================="
