# admin_web

React + TypeScript + Vite admin console.

The dev server proxies `/health`, `/api`, and `/swagger` to Gin on port 8100.
The generated client is rebuilt from `../admin_core/docs/swagger.json` before
each production build.

```bash
pnpm install
pnpm start
```

To regenerate only the API client:

```bash
pnpm run generate:api
```

## 用户与权限 Hooks

使用方式见 [用户与权限 Hooks](src/framework/store/README.md)。
运行示例位于 **首页 → Hooks 示例**（`/dashboard/demo`）。
