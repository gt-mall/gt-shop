# admin_web

React + TypeScript + Vite admin console.

The dev server proxies `/health`, `/api`, and `/swagger` to Gin on port 8080.
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
