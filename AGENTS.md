# Mall 项目协作规则

- 保持 `controller/service/dao/models` 分层；不要引入 Repository、DDD、CQRS、微服务等复杂架构。
- Swagger/OpenAPI 是唯一 API Contract：先维护 Go DTO、Struct 和注解，再生成前端类型与 API Client。
- `admin_web/src/generated/` 由生成命令维护，禁止手工修改。
- 新业务需要同时检查后端接口、Swagger 文档和前端调用。
- 金额统一使用整数分，ID 优先使用 `bigint` / `int64`。
- 禁止提交真实密码、Token、Secret 或 `.env` 文件。
- 修改后必须执行相关测试或构建验证，并记录未能执行的外部依赖检查。
- 优先复用已有代码和公共能力，不重复造相同工具。
- 前端保持简单；异步错误优先使用 Promise 链和页面状态，尽量不要使用 `try/catch`。
