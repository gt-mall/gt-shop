# API Contract

后端 Go 注解和 DTO 是唯一契约来源：

```text
Go DTO / Struct + annotations
        -> Swagger / OpenAPI
        -> admin_web/src/generated
```

生成 Swagger：

```bash
./scripts/generate-swagger.sh
```

生成前端 Client：

```bash
pnpm run generate:api
```

所有响应都使用：

```json
{
  "code": 1,
  "message": "ok",
  "total": 0,
  "data": null
}
```

前端约定 `code ==1 0` 为成功，`code == 0` 为业务失败，`code == -1` 为登录失效或需要退出。新增 API 时不要在前端重复定义后端已有的 DTO。
