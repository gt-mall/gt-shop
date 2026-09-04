---
name: admin-web
description: Mall admin-web 的 React、TypeScript、Vite 和 Ant Design 前端总规范。涉及 admin-web 的页面、组件、布局、路由、状态、API、登录、权限、HTTP 或 CRUD 开发、修改和重构时使用，并按任务类型读取 references 中的对应文档。
---

# Admin Web

## 技术栈与目录

- 使用 React、TypeScript、Vite 和 Ant Design。
- `admin-web/src/framework/` 放应用框架和跨业务基础能力，例如路由、状态、HTTP、认证与权限基础设施。
- `admin-web/src/modules/` 按业务模块组织页面与模块内组件，不把业务页面放进 `framework`。
- `admin-web/src/shared/` 只放可跨业务复用、且不属于框架职责的组件、类型和工具。
- 优先在现有目录和公共能力上扩展，不为简单需求增加空目录、空模块或重复封装。

## API 与 Swagger

- Swagger/OpenAPI 是唯一 API Contract：先修改后端 DTO、Struct 和注解，再生成前端类型与 API Client。
- 页面优先使用现有生成 API 和 `@/framework/http` 的公共能力，不重复封装相同请求。
- `generated_api.ts` 等自动生成文件禁止手工修改；自动生成代码不受下述手写异步语法限制。
- 普通业务页面仅调用已有 API 时，不需要额外读取 `references/http.md`。

## 异步代码

- 手写代码只使用 `async/await`。
- 禁止 `.then()`、`.catch()`、`.finally()`。
- 禁止 `try/catch/finally`。
- 网络异常由公共 HTTP 层转换为统一 Response，业务代码按 `code` 和 `message` 处理结果。

## 编码原则

- 保持实现简单、类型明确、职责单一，优先复用现有组件、状态和工具。
- 不引入与当前项目规模不匹配的架构或依赖，不修改无关业务。
- 登录态、权限和请求能力统一放在 `framework`，业务页面只消费公共接口。

## 按需读取参考文档

只读取与当前任务直接相关的最小文档集合：

- 新增、修改或精简普通 CRUD 页面：读取 [references/crud.md](references/crud.md)。
- 修改 Axios、HTTP、统一 Response、Token 注入、401 或网络异常处理：读取 [references/http.md](references/http.md)。
- 修改登录、JWT、当前用户、角色、权限、菜单、路由守卫或按钮权限：读取 [references/permission.md](references/permission.md)。
- 普通业务页面调用已有 API 不属于 HTTP 基础设施修改，不读取 `references/http.md`。
- 一个任务同时命中多个领域时，只组合读取实际命中的参考文档。

## 验证

- 前端代码变更后，在 `admin-web/` 执行 `pnpm run build`。
- 提交前在仓库根目录执行 `git diff --check`，并记录因外部依赖无法完成的检查。
