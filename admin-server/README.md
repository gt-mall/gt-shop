# admin_core

Go + Gin backend for the Mall admin project.

## Run

```bash
cp .env.example .env
set -a
. ./.env
set +a
go run .
```

`DATABASE_DSN` is optional for the public demo endpoints. Protected admin APIs
require MySQL, `AUTH_JWT_SECRET`, and Casbin. When the DSN is set,
the process verifies MySQL and applies embedded migrations in `data/migrations/` when
`AUTO_MIGRATE=true`.

## 数据库与 Migration 规范

- **V1 数据库基线**：`data/migrations/001_baseline.sql`（包含全部26张核心业务表完整结构、主键、索引、约束与中文注释）。
- **默认安装数据**：`data/seeds/001_default_data.sql`（包含系统内置角色、默认文件分组、基础系统配置，具备幂等性）。
- **首次正式发布后冻结**：首次正式发布前允许按规范合并历史迁移与重置基线；首次正式发布后，已发布的 Migration 永久冻结，严禁再对其进行修改，后续数据库变化只能通过新增 Migration（如 `002_xxx.sql`）递增管理。
- **注释与字段标准**：所有业务表必须有中文表注释；所有业务字段原则上必须有中文注释；状态/类型字段标明枚举值；金额字段明确标注单位（元）；JSON字段说明数据结构用途；快照字段明确标注“快照”。
- **Seed 与 Migration 分离**：Migration 仅负责结构，Seed 负责初始化数据，测试/Demo 数据严禁进入正式 Seed。

### 数据库安装与初始化命令

提供显式 CLI 初始化命令与脚本，支持全新空库安装与现有开发库对齐：

1. **全新空库初始化（推荐）**
   适用于新环境或不需要保留旧开发数据的干净安装（一次性执行结构迁移与种子数据注入）：
   ```bash
   # 项目根目录一键脚本
   ./scripts/init_db.sh
   # 或在 admin-server 目录中运行
   cd admin-server && go run . init
   ```

2. **现有开发库基线安全对齐（align）**
   `align` **不会**修改任何业务 Schema。它仅用于已经实际具有 V1 Baseline 结构的旧开发库：
   - 验证现有数据库是否完全符合 V1 Baseline 关键结构要求（26张核心表、关键字段类型、UNIQUE 索引及 CHECK 约束）；
   - 验证通过后，在事务中重置 `schema_migrations` 为 `001_baseline.sql`；
   - 执行幂等 Seed（注入系统内置角色与默认配置）。

   如果数据库结构与 Baseline 不一致，`align` 会**拒绝执行**并输出明确的字段或约束差异提示，避免破坏现有库。
   ```bash
   # 项目根目录一键对齐脚本
   ./scripts/init_db.sh align
   # 或在 admin-server 目录中运行
   cd admin-server && go run . align
   ```

3. **仅执行种子数据注入（seed）**
   ```bash
   ./scripts/init_db.sh seed
   # 或 cd admin-server && go run . seed
   ```

4. **创建首个后台超级管理员（create-admin）**
   全新安装或初始化后数据库中没有用户，使用 `create-admin` 创建首个超级管理员并自动赋予 `super_admin` 角色：
   ```bash
   ADMIN_USERNAME=admin \
   ADMIN_PASSWORD='YourStrongPassword' \
   ADMIN_NICKNAME='超级管理员' \
   go run . create-admin
   # 或通过根目录脚本入口：
   ADMIN_USERNAME=admin ADMIN_PASSWORD='YourStrongPassword' ./scripts/init_db.sh create-admin
   ```
   *注意：`ADMIN_PASSWORD` 必须显式提供，禁止使用弱密码或硬编码默认密码。若用户名已存在将报错拒绝，不会覆盖现有密码。*

Endpoints:

- `GET /health`
- `GET /api/demo`
- `GET /api/plugins/hello`
- `POST /api/auth/login`
- `GET /api/auth/me`
- `POST /api/system/users/create`
- `POST /api/system/users/list`
- `POST /api/system/users/detail` (`{"id": 1}`)
- `POST /api/system/users/update` (`{"id": 1, "nickname": "New name"}`)
- `POST /api/system/users/delete` (`{"id": 1}`)
- `POST /api/system/users/roles`
- `POST /api/system/users/roles/save`
- `POST /api/system/roles/create|list|detail|update|delete`
- `POST /api/system/roles/permissions`
- `POST /api/system/roles/permissions/save`
- `GET /api/system/permissions/list`
- `GET /swagger/index.html`

All `/api/system/*` endpoints require a JWT in `Authorization: Bearer <token>`
or the legacy-compatible `token` header, followed by their declared Casbin
permission. The initial `super_admin` user is created using `go run . create-admin`,
which automatically sets up the Casbin role assignment.

Generate Swagger from Go annotations:

```bash
./scripts/generate-swagger.sh
```
