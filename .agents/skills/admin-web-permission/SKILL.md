---
name: admin-web-permission
description: Mall admin-web 的登录、JWT、当前用户、角色、权限、菜单过滤、路由守卫和按钮权限规范。涉及认证或前端权限控制时使用。
---

# Admin Web Permission

先遵循 [admin-web 总规范](../admin-web/SKILL.md)。涉及请求封装或异常处理时，同时读取 [HTTP 规范](../admin-web-http/SKILL.md)。

## 状态归属

- JWT、当前用户、角色和权限统一放在 `src/framework/store` 的 Zustand store 中。
- 权限判断统一由 store 提供，例如 `hasPermission(code)`。
- 不创建独立的 `framework/auth` 目录，不在页面中重复维护认证状态。
- HTTP 层只从 store 读取 JWT；401 时调用 store 的退出能力。

## 登录与刷新

- 登录使用 Swagger 生成的 API，不手写重复请求类型。
- 未勾选记住登录时，Token 使用后端默认有效期并保存在 `sessionStorage`。
- 勾选“记住登录 30 天”时，请求携带 `remember: true`，后端签发 30 天 JWT，前端保存到 `localStorage`。
- 同时保存后端返回的 `expires_at`，启动时先检查本地过期时间。
- 应用启动或 F5 刷新时，只要存在有效 Token，就调用 `/api/auth/me`，并用响应覆盖 store 中的用户、角色和权限。
- 退出、Token 过期或认证失败时，清除内存状态及两种浏览器存储。

## 权限控制

- 菜单、路由和按钮根据 store 中的权限 code 控制显示与访问。
- 权限 code 使用后端统一定义的三段式格式，例如 `system:user:list`。
- 前端权限只用于交互和路由体验；重要接口必须继续由后端中间件鉴权。
- `super_admin` 的全部权限由后端 `/api/auth/me` 返回，前端不硬编码超级管理员绕过逻辑。

## 修改检查

- 检查登录、刷新、退出、401 和本地过期清理流程。
- 检查菜单、路由和按钮权限是否使用同一个 store 方法。
- 执行 `pnpm run build` 和 `git diff --check`。
