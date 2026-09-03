# Mall

一个适合持续迭代的商城管理端基础项目：Go + Gin、React + TypeScript + Vite、MySQL，以及从后端 Swagger 生成的前端 API Client。

## 目录

- `admin-server/`：Gin 后端，按大模块到小模块组织业务代码，使用 GORM + MySQL。
- `admin-web/`：React 管理端，开发服务器将 API 代理到 Gin。
- `.agents/skills/`：项目级后续开发规则。
- `docs/`：架构、API Contract 和插件说明。

## 快速启动

后端不设置 `DATABASE_DSN` 也能启动 Demo；配置 MySQL 后会按环境变量执行迁移。

```bash
./scripts/install.sh
if [ ! -f admin-server/.env ]; then cp admin-server/.env.example admin-server/.env; fi
./scripts/start.sh
```

`start.sh` 会同时启动后端和前端。也可以分别启动：

```bash
./scripts/start-backend.sh
./scripts/start-frontend.sh
```

本机打开 <http://localhost:5173>。局域网其他设备使用运行本项目电脑的局域网 IP 访问，例如 `http://192.168.1.100:5173`。首页会真实请求 `/health`，Demo 和 Hello Plugin 页面分别请求 `/api/demo` 与 `/api/plugins/hello`。

## 生成文档和 Client

```bash
./scripts/generate-api.sh
```

`pnpm run build` 会先自动生成 Client，因此前端生成目录不会依赖手工维护。

## 验证

```bash
./scripts/test.sh
```

## 编译

```bash
./scripts/build.sh
```

也可以单独编译：

```bash
./scripts/build-backend.sh
./scripts/build-frontend.sh
```

## 可用地址

- Gin：`http://localhost:8080`
- 健康检查：`GET /health`
- 后端 Demo：`GET /api/demo`
- Hello Plugin：`GET /api/plugins/hello`
- 系统用户：`/api/system/users`
- Swagger UI：`http://localhost:8080/swagger/index.html`
