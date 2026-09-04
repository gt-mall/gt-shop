# Async Code Style

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

业务代码中不要自行捕获请求异常。

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

请求异常、网络错误、超时、401、403、500 等统一由 HTTP 请求层处理，业务页面不重复处理异常。

项目规则：

- 只使用 `async/await`
- 禁止 `.then()`
- 禁止 `.catch()`
- 禁止 `.finally()`
- 禁止 `try/catch/finally`
- HTTP 异常统一由公共请求层处理
- 业务失败通过统一 Response 的 `code/message` 判断
- 使用提前 `return`，避免多层 `if`
- 保持业务代码线性、扁平、简洁
- 修改旧代码时，发现上述禁止写法应一并重构
