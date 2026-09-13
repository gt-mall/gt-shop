#!/bin/bash

# 若端口 6100 已被占用，先安全释放
OLD_PIDS=$(lsof -ti :6100 2>/dev/null || true)
if [[ -n "${OLD_PIDS}" ]]; then
  echo "Stopping existing process on port 6100 (PID: ${OLD_PIDS})..."
  kill -9 ${OLD_PIDS} 2>/dev/null || true
  sleep 0.5
fi

exec pnpm run start
