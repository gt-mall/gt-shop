# API Contract

后端 Go DTO、Struct 和 Swagger 注解是唯一契约来源：

```text
Go DTO / Struct / Swagger annotations
        -> admin-server/docs
        -> admin-web/src/framework/http/generated_api.ts
```

后台与小程序 Swagger 分开维护：

- 后台：`admin-server/docs/`
- 小程序：`admin-server/docs-miniapp/`

在仓库根目录生成 Swagger 和后台 Client：

```bash
./scripts/3_generate_api.sh
./sw.sh
```

禁止手工修改生成的 `generated_api.ts`。小程序继续通过 `miniapp/src/api/` 提供薄封装，字段和响应结构以小程序 Swagger 为准。

所有响应使用统一结构：

```json
{
  "code": 1,
  "message": "ok",
  "total": 0,
  "data": null
}
```

- `code === 1`：成功。
- `code === 0`：业务失败。
- `code === -1`：登录失效或需要重新登录。

新增或修改业务时，按“后端 DTO/注解 → Swagger → 生成 Client → 前端调用”的顺序完成，不能根据页面临时猜测或重复定义后端模型。
