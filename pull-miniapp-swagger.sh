#!/bin/bash
set -Eeuo pipefail

# 获取项目根目录绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="${SCRIPT_DIR}"

# 默认拉取地址，支持通过命令行参数或环境变量覆盖
DEFAULT_URL="http://192.168.1.60:8100/swagger/miniapp/swagger.json"
SWAGGER_URL="${1:-${MINIAPP_SWAGGER_URL:-$DEFAULT_URL}}"

# 本地保存目标路径
TARGET_ADMIN_SERVER="${PROJECT_DIR}/admin-server/docs-miniapp/swagger.json"
TARGET_DOCS_MINIAPP="${PROJECT_DIR}/docs/miniapp/swagger.json"

echo "==> 准备从小程序接口文档地址拉取最新 Swagger 配置..."
echo "    URL: ${SWAGGER_URL}"

# 依赖检查
if ! command -v curl >/dev/null 2>&1; then
  echo "错误: 未找到 curl 命令，请先安装 curl。" >&2
  exit 1
fi

TMP_FILE="$(mktemp "${TMPDIR:-/tmp}/swagger-miniapp.XXXXXX.json")"
trap 'rm -f "${TMP_FILE}"' EXIT

# 拉取文档
HTTP_CODE=$(curl -sL -w "%{http_code}" -o "${TMP_FILE}" "${SWAGGER_URL}")

if [[ "${HTTP_CODE}" != "200" ]]; then
  echo "错误: 拉取失败，HTTP 状态码: ${HTTP_CODE} (URL: ${SWAGGER_URL})" >&2
  exit 1
fi

if [[ ! -s "${TMP_FILE}" ]]; then
  echo "错误: 下载的文件为空。" >&2
  exit 1
fi

# 校验 JSON 格式合法性
IS_VALID_JSON=0
if command -v jq >/dev/null 2>&1; then
  if jq empty "${TMP_FILE}" >/dev/null 2>&1; then
    IS_VALID_JSON=1
    # 格式化 JSON
    FORMATTED_TMP="$(mktemp "${TMPDIR:-/tmp}/swagger-miniapp-fmt.XXXXXX.json")"
    jq . "${TMP_FILE}" > "${FORMATTED_TMP}"
    mv "${FORMATTED_TMP}" "${TMP_FILE}"
  fi
elif command -v python3 >/dev/null 2>&1; then
  if python3 -m json.tool "${TMP_FILE}" >/dev/null 2>&1; then
    IS_VALID_JSON=1
  fi
elif command -v node >/dev/null 2>&1; then
  if node -e 'JSON.parse(require("fs").readFileSync(process.argv[1]))' "${TMP_FILE}" >/dev/null 2>&1; then
    IS_VALID_JSON=1
  fi
fi

if [[ "${IS_VALID_JSON}" -ne 1 ]]; then
  echo "错误: 下载的内容不是合法的 JSON 格式，请检查服务端响应内容。" >&2
  head -n 20 "${TMP_FILE}" >&2
  exit 1
fi

# 确保目标目录存在并写入目标文件
mkdir -p "$(dirname "${TARGET_ADMIN_SERVER}")"
cp "${TMP_FILE}" "${TARGET_ADMIN_SERVER}"
echo "    已更新: ${TARGET_ADMIN_SERVER}"

mkdir -p "$(dirname "${TARGET_DOCS_MINIAPP}")"
cp "${TMP_FILE}" "${TARGET_DOCS_MINIAPP}"
echo "    已更新: ${TARGET_DOCS_MINIAPP}"

FILE_SIZE=$(wc -c < "${TARGET_ADMIN_SERVER}" | tr -d ' ')
echo "==> 小程序 Swagger 文档更新成功！(大小: ${FILE_SIZE} 字节)"
