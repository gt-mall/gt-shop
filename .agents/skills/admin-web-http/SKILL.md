---
name: admin-web-http
description: Mall admin-web 的 Axios、HTTP 请求封装、统一 Response、Token、401、网络异常和 Swagger API Client 规范。涉及请求层或接口调用时使用。
---

# Admin Web HTTP

先遵循 [admin-web 总规范](../admin-web/SKILL.md)。

## API 契约

- Swagger/OpenAPI 是唯一接口契约；先更新后端 DTO 和注解，再重新生成 Client。
- 统一使用 `@/framework/http` 导出的 API。
- 禁止手工修改 `src/framework/http/generated_api.ts`。
- 统一 Response 字段为 `code`、`message`、`total`、`data`。

## 异步代码

前端 TypeScript / JavaScript 异步代码统一使用 `async/await`。

禁止以下写法：

```ts
.then()
.catch()
.finally()

try {
} catch {
} finally {
}
```

业务代码中不要自行捕获请求异常，也不能收到 rejected Promise。公共 HTTP 层必须把网络错误、超时和服务端异常转换为统一 Response。

统一写成：

```ts
async function loadData() {
  const result = await api.getUsers(params);

  if (result.code !== 1) {
    message.error(result.message);
    return;
  }

  setUsers(result.data);
}
```

有 loading 状态时：

```ts
async function loadData() {
  setLoading(true);

  const result = await api.getUsers(params);

  setLoading(false);

  if (result.code !== 1) {
    message.error(result.message);
    return;
  }

  setUsers(result.data);
}
```

## HTTP 层职责

- 使用 Axios 请求/响应拦截器或等价的公共边界统一处理请求异常。
- 需要处理 Promise rejection 时，使用 Axios 拦截器的错误回调，不在业务代码中增加 `.catch()` 或 `try/catch`。
- 401 或 Token 失效时清理 Zustand 登录状态，由登录守卫显示登录页。
- 403 保留登录状态并返回明确的无权限消息。
- 网络错误、超时和 5xx 转换为统一 Response；业务调用 `await api...` 后仍能按 `code/message` 判断。
- 请求头中的 JWT 必须从 `framework/store` 读取，不重复维护 Token 状态。
- 不记录或输出真实 JWT、密码、Secret。

## 页面调用

业务页面使用提前返回：

```ts
const result = await api.api.listSystemUsers(params);
if (result.code !== 1) {
  message.error(result.message || '请求失败');
  return;
}

setUsers(result.data ?? []);
```

有 loading 状态时，无论成功或失败都要在统一 Response 返回后恢复状态。

## 检查

- 搜索手写代码中的 `.then()`、`.catch()`、`.finally()` 和 `try/catch/finally`。
- 排除 `generated_api.ts` 等生成文件。
- 执行 `pnpm run build` 和 `git diff --check`。
