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

2. **现有开发库保留数据平滑对齐**
   由于基线合并，现有开发库若已存在表结构，直接启动会因缺少 `001_baseline.sql` 版本记录再次执行 `CREATE TABLE` 而冲突报错。若需保留现有业务数据，请执行基线对齐：
   ```bash
   # 方式一：一键对齐命令（重置迁移记录为 001_baseline.sql 并执行 Seed）
   ./scripts/init_db.sh align
   # 或 cd admin-server && go run . align

   # 方式二：手动 SQL 对齐
   mysql -u root -p ai_shop -e "DELETE FROM schema_migrations; INSERT INTO schema_migrations (version) VALUES ('001_baseline.sql');"
   # 然后注入/刷新 Seed：
   ./scripts/init_db.sh seed
   ```

3. **仅执行种子数据注入**
   ```bash
   ./scripts/init_db.sh seed
   # 或 cd admin-server && go run . seed
   ```



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
permission. The initial `super_admin` user-role assignment is intentionally not
guessed by migration; assign the intended user through `casbin_rule` once when
bootstrapping an existing database, then restart the service so Casbin reloads
the policy:

```sql
INSERT IGNORE INTO casbin_rule (ptype, v0, v1)
VALUES ('g', 'user:<system_user.id>', 'super_admin');
```

Generate Swagger from Go annotations:

```bash
./scripts/generate-swagger.sh
```
