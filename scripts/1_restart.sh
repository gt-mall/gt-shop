#!/bin/bash
set -Eeuo pipefail

# 获取脚本所在目录的绝对路径
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$( cd "${SCRIPT_DIR}/.." && pwd )"
SERVER_DIR="${PROJECT_DIR}/admin-server"
RELEASE_DIR="${PROJECT_DIR}/release"
RUN_DIR="${RELEASE_DIR}/run"
BACKEND_BINARY="${RUN_DIR}/mall_server"
ENV_FILE="${SERVER_DIR}/.env"
LOG_FILE="${RUN_DIR}/mall_server.log"

for command_name in go ps readlink tail; do
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "需要 ${command_name} 命令" >&2
    exit 1
  fi
done

if [[ ! -d "${SERVER_DIR}" ]]; then
  echo "未找到后端目录: ${SERVER_DIR}" >&2
  exit 1
fi

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "未找到环境配置文件: ${ENV_FILE}" >&2
  exit 1
fi

mkdir -p "${RUN_DIR}"

declare -a BACKEND_PIDS=()
if [[ -f "${RUN_DIR}/mall_server.pid" ]]; then
  saved_pid=$(cat "${RUN_DIR}/mall_server.pid" 2>/dev/null || true)
  if [[ -n "${saved_pid}" ]] && kill -0 "${saved_pid}" 2>/dev/null; then
    BACKEND_PIDS+=("${saved_pid}")
  fi
fi

while read -r pid command_line; do
  [[ -n "${pid}" && "${pid}" != "$$" && "${pid}" != "${PPID:-}" ]] || continue

  case "${command_line}" in
    *"${BACKEND_BINARY}"*|*"${RELEASE_DIR}/admin-server"*|*mall-admin-core*|*mall_server*|*"go run ."*)
      ;;
    *)
      continue
      ;;
  esac

  process_dir=""
  if [[ -e "/proc/${pid}/cwd" ]]; then
    process_dir=$(readlink -f "/proc/${pid}/cwd" 2>/dev/null || true)
  elif command -v lsof >/dev/null 2>&1; then
    process_dir=$(lsof -a -p "${pid}" -d cwd -Fn 2>/dev/null | sed -n 's/^n//p')
  fi

  is_backend=0
  case "${command_line}" in
    *"${BACKEND_BINARY}"*|*"${RELEASE_DIR}/admin-server"*|*mall-admin-core*)
      if [[ -z "${process_dir}" || "${process_dir}" == "${SERVER_DIR}" || "${process_dir}" == "${RUN_DIR}" || "${command_line}" == *"${PROJECT_DIR}"* ]]; then
        is_backend=1
      fi
      ;;
    *mall_server*)
      if [[ -z "${process_dir}" || "${process_dir}" == "${RUN_DIR}" || "${command_line}" == *"${RUN_DIR}"* ]]; then
        is_backend=1
      fi
      ;;
    *"go run ."*)
      if [[ -z "${process_dir}" || "${process_dir}" == "${SERVER_DIR}" ]]; then
        is_backend=1
      fi
      ;;
  esac

  if [[ "${is_backend}" -eq 1 ]]; then
    already_added=0
    for existing_pid in "${BACKEND_PIDS[@]}"; do
      if [[ "${existing_pid}" == "${pid}" ]]; then
        already_added=1
        break
      fi
    done
    if [[ "${already_added}" -eq 0 ]]; then
      BACKEND_PIDS+=("${pid}")
    fi
  fi
done < <(ps -eo pid=,args=)

# 停止服务
echo "Checking running backend..."
if [[ "${#BACKEND_PIDS[@]}" -eq 0 ]]; then
  echo "No running backend found."
else
  for pid in "${BACKEND_PIDS[@]}"; do
    echo "Stopping backend process ${pid}..."
    kill "${pid}" 2>/dev/null || true
  done

  for ((attempt = 0; attempt < 40; attempt++)); do
    running=0
    for pid in "${BACKEND_PIDS[@]}"; do
      if kill -0 "${pid}" 2>/dev/null; then
        running=1
        break
      fi
    done
    if [[ "${running}" -eq 0 ]]; then
      break
    fi
    sleep 0.25
  done

  for pid in "${BACKEND_PIDS[@]}"; do
    if kill -0 "${pid}" 2>/dev/null; then
      echo "Force stopping backend process ${pid}..."
      kill -KILL "${pid}" 2>/dev/null || true
    fi
  done
fi

# 编译后端
echo "Building backend..."
(cd "${SERVER_DIR}" && go build -trimpath -o "${BACKEND_BINARY}" .)
cp "${ENV_FILE}" "${RUN_DIR}/.env"
echo "Backend binary: ${BACKEND_BINARY}"
echo "Environment file: ${RUN_DIR}/.env"

echo "Starting backend..."
(
  cd "${SERVER_DIR}"
  set -a
  # shellcheck disable=SC1091
  . "${RUN_DIR}/.env"
  set +a
  if command -v setsid >/dev/null 2>&1; then
    setsid "${BACKEND_BINARY}" >>"${LOG_FILE}" 2>&1 < /dev/null &
  else
    nohup "${BACKEND_BINARY}" >>"${LOG_FILE}" 2>&1 < /dev/null &
  fi
  echo $! > "${RUN_DIR}/mall_server.pid"
)
BACKEND_PID=$(cat "${RUN_DIR}/mall_server.pid")

started=0
for ((attempt = 0; attempt < 40; attempt++)); do
  if kill -0 "${BACKEND_PID}" 2>/dev/null; then
    started=1
    break
  fi
  sleep 0.25
done

# 再次确认进程未在启动瞬间崩溃（如端口已被占用、数据库连不上等）
sleep 0.5
if ! kill -0 "${BACKEND_PID}" 2>/dev/null; then
  started=0
fi

if [[ "${started}" -eq 0 ]]; then
  echo "Restart failed. Recent log:" >&2
  tail -n 50 "${LOG_FILE}" >&2 || true
  exit 1
fi

FOLLOW_LOG=0
for arg in "$@"; do
  if [[ "${arg}" == "l" || "${arg}" == "-l" || "${arg}" == "--log" ]]; then
    FOLLOW_LOG=1
    break
  fi
done

echo "Restart completed."
echo "Backend PID: ${BACKEND_PID}"

if [[ "${FOLLOW_LOG}" -eq 1 ]]; then
  echo "Following log: ${LOG_FILE}"
  exec tail -n 100 -F "${LOG_FILE}"
fi

exit 0
