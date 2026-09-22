# 编译与上传

脚本默认使用 SSH 别名 `14`，部署到 `/www/wwwroot/gt-shop/run`。执行 `bash scripts/5_deploy.sh`，也可直接用环境变量覆盖：

```bash
SSH_TARGET=mall-prod DEPLOY_DIR=/opt/mall bash scripts/5_deploy.sh
# 指定用户名、IP 和 SSH 端口
SSH_TARGET=ssh://root@192.0.2.10:2222 DEPLOY_DIR=/opt/mall bash scripts/5_deploy.sh
```

SSH 别名读取本机 `~/.ssh/config`。需要密码、私钥口令或首次确认主机指纹时，脚本会等待终端输入；请在交互终端运行。认证失败则停止。

执行顺序：连接服务器识别 Linux amd64/arm64 → 编译 admin-web → 复制到 `admin-server/static/dist` → 交叉编译 Go → 将 `mall_server` 和 `static/` 压缩 → 上传压缩包 → 在部署目录解压并替换。前端也嵌入程序；独立的 `static/` 可供 Nginx 使用。上传不自动重启运行中的服务。

本地依赖 Go、pnpm、SSH，前端依赖需先在 `admin-web` 执行 `pnpm install --frozen-lockfile`。服务器需要 Linux、SSH、POSIX shell 和常用的 `uname`、`mktemp` 等命令。

仅本地编译后台：

```bash
bash scripts/2_build_all.sh --admin-only
```

产物为 `release/run/mall_server`，同时保留独立前端文件 `release/run/static/`（包含 `index.html` 和 `assets/`）。Go 程序使用内嵌页面；独立文件可供 Nginx 等静态服务器使用。原 `2_build_all.sh` 无参数执行时仍额外构建小程序。

前端构建自动为 JS、CSS、HTML 等生成更小的 `.gz` 副本，保留原文件。Go 根据 `Accept-Encoding` 返回 gzip 内容，并设置 `Content-Encoding` 和 `Vary`。如果由 Nginx 直接托管 `static/`，需在对应静态站点配置中启用 `gzip_static on;`。

服务器首次运行需自行配置环境变量（参考 `admin-server/.env.example`）。Go 本身不自动读取 `.env`；若使用服务器上的 `.env`，可这样启动：

```bash
cd /opt/mall
set -a
. ./.env
set +a
./mall_server
```

长期运行建议交给已有的 systemd 等服务管理器，并将工作目录设为部署目录。上传后重启对应服务，访问 `http://服务器地址:APP_PORT/`（默认 8100）。页面刷新支持前端路由，`/api`、`/media` 等仍由后端处理。

服务器部署目录提供 `restart.sh`。它按进程名查找并停止 `mall_server`，读取同目录 `.env` 后使用 `nohup` 启动，最后通过 `tail -F` 查看实时日志。按 `Ctrl+C` 仅退出日志查看，不影响后台服务；脚本不创建 PID 文件。

脚本不上传本地 `.env`、数据库或 uploads，不执行数据库初始化。服务配置和上传文件保留在服务器。修改前端后需重新编译 Go；仅 `go run .` 且未构建前端时，后端照常提供 API，首页返回 404。
