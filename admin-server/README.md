# admin_core

Go + Gin backend for the Mall admin project.

## Run

```bash
cp .env.example .env
set -a
. ./.env
set +a
go run .
```

`DATABASE_URL` is optional for the demo. When it is set, the process verifies
PostgreSQL and applies files in `migrations/` when `AUTO_MIGRATE=true`.

Endpoints:

- `GET /health`
- `GET /api/demo`
- `GET /api/plugins/hello`
- `GET /swagger/index.html`

Generate Swagger from Go annotations:

```bash
./scripts/generate-swagger.sh
```
