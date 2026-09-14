#!/bin/bash
set -Eeuo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
SERVER_DIR="$( cd "${SCRIPT_DIR}/.." && pwd )"
ENV_FILE="${SERVER_DIR}/.env"

if ! command -v go >/dev/null 2>&1; then
  echo "需要 go 命令" >&2
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
  init|seed|align)
    ;;
  *)
    echo "未知命令: ${ACTION}" >&2
    echo "用法: $0 [init|seed|align]" >&2
    exit 1
    ;;
esac

(cd "${SERVER_DIR}" && go run . "${ACTION}")
