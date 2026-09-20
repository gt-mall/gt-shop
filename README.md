# Mall

一个持续迭代的商城项目，包含 Go/Gin/MySQL 后端、React 管理端和 UniApp 商城用户端。

## 目录

- `admin-server/`：后台与小程序 API，包含 Migration、Seed、Swagger 和业务测试。
- `admin-web/`：React + TypeScript + Vite 管理端。
- `miniapp/`：UniApp + Vue 3 用户端，支持 H5 和微信小程序。
- `scripts/`：构建、测试、API 生成、数据库初始化和后端重启脚本。
- `.agents/skills/`：项目级开发规范。

## 首次准备

```bash
cd admin-server && go mod download
cd ../admin-web && pnpm install
cd ../miniapp && pnpm install
cd ..

cp admin-server/.env.example admin-server/.env
```

按实际环境填写 `admin-server/.env`。禁止提交真实密码、Token、Secret 或 `.env` 文件。

## 开发启动

后端使用 `admin-server/.env` 编译并重启：

```bash
./scripts/1_restart.sh
```

管理端开发服务器：

```bash
cd admin-web
pnpm start
```

小程序端开发服务器：

```bash
cd miniapp
pnpm dev:h5
# 或 pnpm dev:mp-weixin
```

默认地址：后端 `http://localhost:8100`，管理端 `http://localhost:6100`。

## API Contract

Go DTO、Struct 和 Swagger 注解是唯一 API Contract。更新后台 API 后依次执行：

```bash
./scripts/3_generate_api.sh
./sw.sh
```

生成文件 `admin-web/src/framework/http/generated_api.ts` 禁止手工修改。小程序 Swagger 位于 `admin-server/docs-miniapp/`。

## 验证与构建

```bash
./scripts/4_test_all.sh
./scripts/2_build_all.sh
git diff --check
```

需要真实 MySQL 并发或 Migration 验证时，必须配置指向可丢弃测试库的 `TEST_DATABASE_DSN`、`ORDER_TEST_MYSQL_DSN` 或对应专项测试变量，禁止使用业务库。

## 数据库初始化

```bash
./scripts/init_db.sh init
./scripts/init_db.sh seed
./scripts/init_db.sh create-admin
```

项目已经进入增量 Migration 阶段，存在 `002_product_dict.sql`，不要再执行 `align`。
