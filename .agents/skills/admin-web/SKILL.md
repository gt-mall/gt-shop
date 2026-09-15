---
name: admin-web
description: Mall admin-web 的 React、TypeScript、Vite 和 Ant Design 前端总规范。涉及页面、UI 美化、组件、路由、状态、API、认证权限或 CRUD 开发与重构时使用，按任务读取对应 references。
---

# Admin Web

## 技术栈与目录

- React、TypeScript、Vite、Ant Design；样式使用 Tailwind、Less/CSS 和 Theme Token，分工见 UI 参考文档。
- `admin-web/src/framework/`：路由、状态、HTTP、认证与权限等跨业务基础能力。
- `admin-web/src/modules/`：按业务模块组织页面与模块内组件。
- `admin-web/src/shared/`：可跨业务复用、且不属于框架职责的组件、类型和工具。

## 核心开发原则

- 保持实现简单、类型明确、职责单一；优先复用现有能力，不增加空目录、重复封装或无关依赖，不修改无关业务。
- Swagger/OpenAPI 是唯一 API Contract：变更接口先维护后端 DTO、Struct 和注解，再生成前端类型与 API Client；新业务核对后端接口、Swagger 与前端调用。
- 使用生成 API 和 `@/framework/http` 公共能力；禁止手工修改 `generated_api.ts` 等自动生成文件。
- 手写异步代码只使用 `async/await`，禁止 `.then()`、`.catch()`、`.finally()` 和 `try/catch/finally`；自动生成代码不受此语法限制。
- 网络异常由公共 HTTP 层转换为统一 Response，业务代码按 `code`、`message` 处理结果；登录态和权限统一消费 `framework` 公共接口。
- 前端代码变更后在 `admin-web/` 执行 `pnpm run build`；文档变更检查引用与规则一致性。提交前执行 `git diff --check`，记录未能完成的检查及原因。

## 按需读取参考文档

只读取任务实际命中的文档；同时涉及行为和界面时组合读取：

- API 调用、查询、分页、新增、编辑、删除或 CRUD 状态管理：读取 [crud.md](references/crud.md)，负责业务行为。
- 创建或调整后台页面、布局、列表页、详情页、编辑页，或进行美化、UI 优化、视觉改造：必须先读取 [admin-ui.md](references/admin-ui.md)，负责结构、视觉与交互一致性。
- 修改 Axios、HTTP、统一 Response、Token 注入、401 或网络异常处理：读取 [http.md](references/http.md)；仅调用已有 API 不需读取。
- 修改登录、JWT、当前用户、角色、权限、菜单、路由守卫或按钮权限：读取 [permission.md](references/permission.md)。
