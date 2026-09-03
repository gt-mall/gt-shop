# Architecture

本项目使用一个 Git 仓库，后端和管理端分别位于 `admin_core/` 与 `admin_web/`，不引入复杂 Monorepo 工具。

## Backend

请求流向固定为：`routers -> controller -> service -> dao -> PostgreSQL`。

- `controller`：绑定请求参数、做输入校验、调用 service、通过 `pkg/response` 返回。
- `service`：业务编排和领域规则。
- `dao`：只负责数据库访问和查询。
- `models`：数据库模型与 API DTO。
- `bootstrap`：配置、数据库连接、迁移。
- `pkg`：跨业务的公共基础能力。

没有 Repository、CQRS、DDD 或微服务层。未来增加商品、订单等业务时，沿用同一分层。

## Frontend

`core/http` 负责生成 Client 与响应码判断，`pages` 负责页面，`plugins` 负责编译期插件入口，`router` 负责页面路由。Vite 开发服务器把 `/health`、`/api` 和 `/swagger` 转发给 Gin。

## Database

所有连接参数来自环境变量。`DATABASE_URL` 未配置时，Demo 以无数据库模式启动；配置后启动会验证连接并按文件名顺序执行 `migrations/`。
