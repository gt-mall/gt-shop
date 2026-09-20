# Mall Admin Web

React + TypeScript + Vite 管理端。开发服务器默认运行在 `http://localhost:6100`，并把 `/health`、`/api`、`/media` 和 `/swagger` 代理到 `VITE_BACKEND_URL`，默认目标为 `http://localhost:8100`。

## 开发

```bash
pnpm install
pnpm start
```

## 测试与构建

```bash
pnpm run test
pnpm run build
```

后台 API Client 位于 `src/framework/http/generated_api.ts`。接口发生变化时在仓库根目录执行：

```bash
./scripts/3_generate_api.sh
./sw.sh
```

生成文件禁止手工修改。

用户、角色和权限 Hooks 的使用方式见 [framework/store/README.md](src/framework/store/README.md)。
