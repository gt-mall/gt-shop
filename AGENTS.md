# Mall 项目协作规则

- Swagger/OpenAPI 是唯一 API Contract：先维护 Go DTO、Struct 和注解，再生成前端类型与 API Client。
- 新业务需要同时检查后端接口、Swagger 文档和前端调用。
- 金额以元为单位使用 `DECIMAL` 存储；计算时精确转换为整数分或使用十进制类型，禁止使用 `float32` / `float64` 进行金额转换和计算。
- ID 优先使用 `bigint` / `int64`。
- 禁止提交真实密码、Token、Secret 或 `.env` 文件。
- 修改后必须执行相关测试或构建验证，并记录未能执行的外部依赖检查。
- 优先复用已有代码和公共能力，不重复造相同工具。
- 涉及 `admin-web` 的开发、修改或重构时，读取 `.agents/skills/admin-web/SKILL.md`，其他前端参考文档按该 Skill 的规则按需读取。
- 涉及 `admin-server` 的开发、修改或重构时，读取 `.agents/skills/admin-server/SKILL.md`，其他后端参考文档按该 Skill 的规则按需读取。
