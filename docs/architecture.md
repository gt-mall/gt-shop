# Architecture

本项目使用一个 Git 仓库，后端和管理端分别位于 `admin-server/` 与 `admin-web/`，不引入复杂 Monorepo 工具。

## Backend

请求流向固定为：`routers -> 大模块/小模块 API -> 小模块 service -> MySQL`。

- `internal/model`：数据库结构体集中位置；后续 GORM Gen 生成的模型也放在这里。
- `internal/query`：仅在引入 GORM Gen 查询代码后创建，目前没有该目录。
- `internal/system`：系统大模块；当前仅包含实际使用的 `demo`、`health` 和运行时数据库状态协调。
- `internal/auth`：认证边界；当前仅保留已有的 JWT 中间件占位。
- `internal/plugin`：已有的编译期插件模块，具体插件位于自己的小模块目录。
- 小模块中的 `api.go`、`service.go`、`types.go` 分别承担接口、业务和 API 类型职责；所有 API 路由集中放在最外层 `routers/`，按大类拆分文件。
- `routers/router.go` 负责总装配，`routers/system.go`、`routers/plugin.go` 等文件负责对应大类的路由注册。
- `bootstrap`：配置、数据库连接、迁移。
- `pkg`：跨业务的公共基础能力。

当前没有 `admin`、`role`、`permission`、商品 API 或 SKU API 实现，因此不创建对应的空目录。没有 Repository、DAO、CQRS、DDD 或微服务层；后续增加业务时，沿用大模块到小模块的结构。

## Frontend

`core/http` 负责生成 Client 与响应码判断，`pages` 负责页面，`plugins` 负责编译期插件入口，`router` 负责页面路由。Vite 开发服务器把 `/health`、`/api` 和 `/swagger` 转发给 Gin。

## Database

所有连接参数来自环境变量。`DATABASE_DSN` 未配置时，Demo 以无数据库模式启动；配置后启动会验证 MySQL 连接并按文件名顺序执行 `migrations/`。
