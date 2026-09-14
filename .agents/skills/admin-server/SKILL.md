---
name: admin-server
description: Mall admin-server 的 Go 后端总规范。涉及 admin-server 的架构、模块划分、API 与 Swagger 设计、数据库访问、金额计算或测试与重构时使用，并按任务类型读取 references 中的对应文档。
---

# Admin Server

## 技术栈与目录

- 使用 Go、Gin、GORM (GORM Gen)、MySQL。
- `admin-server/internal/` 按业务领域组织“大模块/小模块”（例如 `internal/product/sku`、`internal/system/role`），不按技术层级平铺。
- `admin-server/internal/model/` 与 `admin-server/internal/query/` 放置 GORM Gen 数据库反向生成代码，禁止手工修改。
- `admin-server/pkg/` 放置跨模块复用的公共工具库（如 decimal 工具、HTML 净化、响应封装等）。
- `admin-server/routers/` 负责路由注册与中间件挂载。
- 单元测试与集成测试统一放在 `admin-server/internal/test/<大模块>/`，不分散在业务模块内部。

## API 与 Swagger

- Swagger/OpenAPI 是唯一 API Contract：新增或修改业务时，先维护 Go DTO、Struct、路由以及 Swagger 注解，再运行生成工具更新 Swagger 文档，最后生成前端类型与 Client。
- 统一响应格式通过 `pkg/response` 输出；业务错误通过统一状态码与消息返回。
- 主键与关联 ID 优先使用 `int64` / `bigint`。
- 禁止提交真实密码、Token、Secret 或 `.env` 文件。

## 金额核心规则

- 数据库存储：以元为单位使用 `DECIMAL`（商品价格默认 `DECIMAL(12,2)`），字段名称为 `price`、`amount` 等；只有单位确为分时才使用 `_cent` 后缀。
- 计算与转换：在 Go 中保留十进制精度，使用十进制类型或转为整数分（`int64`）计算；严禁使用 `float32` / `float64` 进行金额转换和计算。
- API 交互：接口默认以元为单位，使用十进制字符串保留精度并在 Swagger 中清晰声明。

## 编码原则

- 保持实现简单、职责单一，小模块内部默认只包含 `api.go`、`service.go`、`types.go`。
- 优先复用已有代码和公共能力，不重复造相同工具。
- 不引入与当前项目规模不匹配的复杂架构，避免小模块间循环调用，多模块业务由大模块协调。

## 数据库与 Migration 规范

- V1 数据库基线为 `admin-server/data/migrations/001_baseline.sql`，默认安装数据为 `admin-server/data/seeds/001_default_data.sql`。
- 首次正式发布后冻结：首次正式发布前允许按规范合并历史迁移与重置基线；首次正式发布后，已发布的 Migration 严禁修改，后续数据库变化只能通过新增 Migration（如 `002_xxx.sql`）递增管理。
- 初始化与开发库对齐命令：全新空库使用 `./scripts/init_db.sh`（或 `cd admin-server && go run . init`）一键执行迁移和 Seed；已有业务数据的开发库使用 `./scripts/init_db.sh align`（或 `cd admin-server && go run . align`）对齐基线版本号并补充 Seed，防止重新执行 CREATE TABLE 报错。`align` 仅限正式发布前、migrations 目录只有 `001_baseline.sql` 时使用；一旦出现 `002_xxx.sql` 等后续迁移，`align` 会直接拒绝执行，后续结构变更必须通过新增 Migration 递增管理。
- 每个业务表必须有清晰中文表注释（`COMMENT='...'`）。
- 每个业务字段原则上必须有中文注释，避免模糊词汇，明确业务含义。
- 状态与类型字段 COMMENT 必须标明对应代码常量的枚举值与含义。
- 金额字段必须标明单位（如 `单位：元`），禁止使用浮点类型。
- JSON 字段 COMMENT 必须说明内部结构及数据用途。
- 历史交易快照字段必须在 COMMENT 中明确标注“快照”语义。
- Seed 与 Migration 严格分离，系统结构由 Migration 管理，初始数据由 Seed 管理。
- 测试数据、Demo 演示数据严禁进入正式 Seed。


## 按需读取参考文档

只读取与当前任务直接相关的最小文档集合：

- 涉及 Go 后端新增大/小模块、模块代码归属、结构拆分，或涉及金额字段设计、单位转换与金额精确计算：读取 [references/module-structure.md](references/module-structure.md)。
- 涉及 Go 后端测试编写、测试目录归类、SQLite/MySQL 测试环境或断言规范：读取 [references/test.md](references/test.md)。

## 验证

- 后端代码变更后，在 `admin-server/` 或仓库根目录执行 Go 编译与测试：
  ```bash
  go test -v ./admin-server/internal/test/...
  ```
- 提交前在仓库根目录执行 `git diff --check`，并记录因外部依赖无法完成的检查。
