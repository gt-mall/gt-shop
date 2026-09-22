#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_NAME="mall_server"
APP_PATH="${SCRIPT_DIR}/${APP_NAME}"
ENV_FILE="${SCRIPT_DIR}/.env"
LOG_FILE="${SCRIPT_DIR}/mall_server.log"

for command_name in pgrep nohup tail; do
  command -v "${command_name}" >/dev/null 2>&1 || {
    echo "缺少命令：${command_name}" >&2
    exit 1
  }
done

if [[ ! -x "${APP_PATH}" ]]; then
  echo "程序不存在或不可执行：${APP_PATH}" >&2
  exit 1
fi
if [[ ! -f "${ENV_FILE}" ]]; then
  echo "配置文件不存在：${ENV_FILE}" >&2
  exit 1
fi

mapfile -t running_pids < <(pgrep -x "${APP_NAME}" || true)
if [[ "${#running_pids[@]}" -gt 0 ]]; then
  echo "停止 ${APP_NAME}：${running_pids[*]}"
  kill "${running_pids[@]}"
  for _ in {1..20}; do
    mapfile -t running_pids < <(pgrep -x "${APP_NAME}" || true)
    [[ "${#running_pids[@]}" -eq 0 ]] && break
    sleep 0.5
  done
  if [[ "${#running_pids[@]}" -gt 0 ]]; then
    echo "强制停止 ${APP_NAME}：${running_pids[*]}"
    kill -KILL "${running_pids[@]}"
  fi
else
  echo "未发现运行中的 ${APP_NAME}，直接启动"
fi

set -a
# shellcheck disable=SC1091
. "${ENV_FILE}"
set +a

echo "启动 ${APP_PATH}"
cd "${SCRIPT_DIR}"
nohup "${APP_PATH}" >>"${LOG_FILE}" 2>&1 </dev/null &
started_pid=$!
sleep 2
if ! kill -0 "${started_pid}" 2>/dev/null; then
  echo "启动失败，最近日志：" >&2
  tail -n 50 "${LOG_FILE}" >&2 || true
  exit 1
fi

echo "服务已启动。按 Ctrl+C 退出日志查看，服务会继续运行。"
exec tail -n 100 -F "${LOG_FILE}"
