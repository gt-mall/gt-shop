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

`DATABASE_DSN` is optional for the demo. When it is set, the process verifies
MySQL and applies files in `migrations/` when `AUTO_MIGRATE=true`.

Endpoints:

- `GET /health`
- `GET /api/demo`
- `GET /api/plugins/hello`
- `POST /api/system/users/create`
- `POST /api/system/users/list`
- `POST /api/system/users/detail` (`{"id": 1}`)
- `POST /api/system/users/update` (`{"id": 1, "nickname": "New name"}`)
- `POST /api/system/users/delete` (`{"id": 1}`)
- `GET /swagger/index.html`

Generate Swagger from Go annotations:

```bash
./scripts/generate-swagger.sh
```
