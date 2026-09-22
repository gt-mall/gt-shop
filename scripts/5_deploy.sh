#!/usr/bin/env bash
set -Eeuo pipefail

# 仅需配置这两项，也可通过同名环境变量覆盖。
# SSH 别名：14；指定 IP/端口：ssh://root@192.0.2.10:2222
SSH_TARGET="${SSH_TARGET:-14}"
DEPLOY_DIR="${DEPLOY_DIR:-/www/wwwroot/gt-shop/run}"

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
for command_name in ssh go pnpm mktemp tar; do
  command -v "${command_name}" >/dev/null || { echo "缺少命令：${command_name}" >&2; exit 1; }
done
if [[ "$SSH_TARGET" == -* || "$DEPLOY_DIR" != /* || "$DEPLOY_DIR" == / ]]; then
  echo "SSH 目标不能以 - 开头；部署目录必须是非根目录的绝对路径。" >&2
  exit 1
fi

# 复用连接，上传阶段无需重复输入密码。首次连接保留 SSH 的交互提示。
SSH_TEMP_DIR="$(mktemp -d)"
SSH_OPTIONS=(-o BatchMode=no -o ControlMaster=auto -o ControlPersist=600 -o "ControlPath=${SSH_TEMP_DIR}/socket")
cleanup() {
  ssh "${SSH_OPTIONS[@]}" -O exit "$SSH_TARGET" >/dev/null 2>&1 || true
  if [[ -n "${DEPLOY_ARCHIVE:-}" ]]; then
    rm -f "$DEPLOY_ARCHIVE"
  fi
  rm -rf "$SSH_TEMP_DIR"
}
trap cleanup EXIT

echo "连接 ${SSH_TARGET}；如提示密码，请在终端输入..."
REMOTE_PLATFORM="$(ssh "${SSH_OPTIONS[@]}" "$SSH_TARGET" 'uname -s; uname -m')"
case "$REMOTE_PLATFORM" in
  $'Linux\nx86_64') TARGET_ARCH=amd64 ;;
  $'Linux\naarch64'|$'Linux\narm64') TARGET_ARCH=arm64 ;;
  *) echo "暂不支持服务器平台：${REMOTE_PLATFORM}" >&2; exit 1 ;;
esac

GOOS=linux GOARCH="$TARGET_ARCH" CGO_ENABLED=0 bash "${PROJECT_DIR}/scripts/2_build_all.sh" --admin-only

DEPLOY_ARCHIVE="$(mktemp "${TMPDIR:-/tmp}/mall-deploy.XXXXXX.tar.gz")"
tar -C "${PROJECT_DIR}/release/run" -czf "$DEPLOY_ARCHIVE" mall_server static restart.sh

# 用 POSIX shell 单引号转义远端路径，支持空格，避免命令注入。
remote_quote() { printf "'%s'" "${1//\'/\'\\\'\'}"; }
REMOTE_DIR="$(remote_quote "$DEPLOY_DIR")"
echo "上传并解压到 ${SSH_TARGET}:${DEPLOY_DIR} ..."
# 压缩包完整上传并成功解压后，才替换服务器上的程序和静态目录。
ssh "${SSH_OPTIONS[@]}" "$SSH_TARGET" "set -eu
mkdir -p -- ${REMOTE_DIR}
cd -- ${REMOTE_DIR}
upload_archive=\$(mktemp .mall-deploy.XXXXXX.tar.gz)
staging_dir=\$(mktemp -d .mall-deploy.XXXXXX)
trap 'rm -rf -- \"\$upload_archive\" \"\$staging_dir\"' EXIT HUP INT TERM
cat > \"\$upload_archive\"
tar -xzf \"\$upload_archive\" -C \"\$staging_dir\"
test -x \"\$staging_dir/mall_server\"
test -x \"\$staging_dir/restart.sh\"
test -f \"\$staging_dir/static/index.html\"
chmod 755 \"\$staging_dir/mall_server\"
chmod 755 \"\$staging_dir/restart.sh\"
rm -rf -- static.previous
if [ -e static ]; then mv -- static static.previous; fi
mv -- \"\$staging_dir/static\" static
mv -f -- \"\$staging_dir/mall_server\" mall_server
mv -f -- \"\$staging_dir/restart.sh\" restart.sh
rm -rf -- static.previous
" < "$DEPLOY_ARCHIVE"

echo "编译上传完成：${DEPLOY_DIR}/mall_server、${DEPLOY_DIR}/static 和 ${DEPLOY_DIR}/restart.sh。"
echo "请使用服务器现有服务管理方式启动或重启；运行配置通过服务器环境变量提供。"
