# HTTP 与统一 Response

## 适用范围

仅在修改 Axios、HTTP 请求封装、统一 Response、Token 注入、401、超时或网络异常处理时读取。普通业务页面只是调用已有 API 时不需要读取本文档。

## API Contract

- Swagger/OpenAPI 是唯一 API Contract；先更新后端 DTO、Struct 和注解，再重新生成前端 API Client 与类型。
- 页面和公共层优先使用生成客户端，不手写重复的 URL、请求类型或响应类型。
- `generated_api.ts` 等生成文件禁止手工修改，重新生成产生的代码不受手写异步语法限制。
- 统一 Response 结构为 `code`、`message`、`total`、`data`；HTTP 成功不等于业务成功，业务成功以 `code === 1` 为准。

## 公共 HTTP 层职责

- 从 `framework/store` 读取 JWT，并在公共请求层统一注入认证头。
- 在公共层把网络断开、超时、非预期 HTTP 状态和无法解析的响应转换为统一 Response，使业务调用始终获得可判断的结果。
- 401 统一清理 Zustand 中的 JWT、当前用户、角色和权限，并进入登录流程。
- 403 表示当前登录态仍有效但权限不足，不应清空登录状态。
- 错误转换、提示去重和认证失效处理集中维护，页面不重复实现。
- 不记录或提交真实密码、Token、Secret、完整认证头或敏感响应内容。

## 异步实现

- 所有手写请求层与调用代码遵循主 Skill：只使用 `async/await`，不得出现 `.then()`、`.catch()`、`.finally()` 或 `try/catch/finally`。
- Axios 拦截器需要错误回调时，使用拦截器 API 提供的回调参数完成统一转换，不在调用处追加 Promise 链。
- 公共请求函数应返回统一 Response；业务页面通过普通条件分支处理 `code` 和 `message`。
- 加载状态应通过明确的控制流程恢复，不依赖 `.finally()` 或 `finally`。

## 调用边界

- 页面只负责准备参数、等待结果、判断统一 Response 和更新界面状态。
- Token 获取、请求头拼装、401 跳转、网络异常文案与底层 Axios 错误不得散落到业务模块。
- 新增公共能力前先检查 `admin-web/src/framework/http/` 与生成客户端是否已有等价实现。
