---
name: admin-web
description: Mall admin-web 的 React、TypeScript、Vite 和 Ant Design 前端总规范。涉及页面、组件、布局、路由、状态、API、登录或权限的开发、修改、重构时使用；具体规则按需读取对应子 Skill。
---

# Admin Web

## 技术与目录

前端使用 React + TypeScript + Vite + Ant Design。

目录职责：

```text
src/framework/   框架基础设施
src/modules/     业务模块
src/shared/      跨模块公共能力
```

业务统一按：

```text
modules / 大功能 / 小功能
```

组织，例如：

```text
modules/system/user
modules/system/role
modules/product/category
```

优先修改已有结构，不创建不必要的抽象层。

## API

- Swagger/OpenAPI 是唯一接口契约。
- 使用 `@/framework/http` 已生成的 API。
- 禁止手工修改 `generated_api.ts`。
- 生成代码不受项目手写代码风格限制。

## 异步代码

- 手写业务代码只使用 `async/await`，不使用 `.then()`、`.catch()`、`.finally()` 或 `try/catch/finally`。
- 网络异常必须由公共 HTTP 层转换成统一 Response，不能让请求异常直接进入业务页面。
- 业务失败通过统一 Response 的 `code/message` 判断。
- 生成代码不受手写代码风格限制。
- 优先提前 `return`，保持代码扁平。

## 编码原则

- 优先使用现有组件、工具和 API
- 不重复封装已有能力
- 简单页面不要提前拆大量组件
- 不引入无关依赖
- 不修改无关代码
- 保持 TypeScript 类型明确
- 菜单、路由、目录、权限命名尽量保持一致

## 按需读取子 Skill

仅当任务匹配时读取对应 Skill，不要默认加载所有子 Skill。

### 普通 CRUD 页面

涉及列表、搜索、分页、新增、编辑、删除时，读取：

```text
../antd-crud-page/SKILL.md
```

### HTTP

涉及 Axios、请求封装、Response、Token、401、网络异常、API Client 时，读取：

```text
../admin-web-http/SKILL.md
```

### 登录与权限

涉及登录、Token、角色、权限、菜单过滤、路由守卫、按钮权限时，读取：

```text
../admin-web-permission/SKILL.md
```

没有匹配到子 Skill 时，仅遵循本 Skill 和现有代码实现。

## 验证

修改完成后至少执行：

```bash
pnpm run build
```

并检查：

```bash
git diff --check
```

只修复与本次任务有关的问题。
