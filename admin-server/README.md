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

`DATABASE_DSN` is optional for the public demo endpoints. Protected admin APIs
require MySQL, `AUTH_JWT_SECRET`, and the Casbin migration. When the DSN is set,
the process verifies MySQL and applies embedded migrations in `data/migrations/` when
`AUTO_MIGRATE=true`.

Endpoints:

- `GET /health`
- `GET /api/demo`
- `GET /api/plugins/hello`
- `POST /api/auth/login`
- `GET /api/auth/me`
- `POST /api/system/users/create`
- `POST /api/system/users/list`
- `POST /api/system/users/detail` (`{"id": 1}`)
- `POST /api/system/users/update` (`{"id": 1, "nickname": "New name"}`)
- `POST /api/system/users/delete` (`{"id": 1}`)
- `POST /api/system/users/roles`
- `POST /api/system/users/roles/save`
- `POST /api/system/roles/create|list|detail|update|delete`
- `POST /api/system/roles/permissions`
- `POST /api/system/roles/permissions/save`
- `GET /api/system/permissions/list`
- `GET /swagger/index.html`

All `/api/system/*` endpoints require a JWT in `Authorization: Bearer <token>`
or the legacy-compatible `token` header, followed by their declared Casbin
permission. The initial `super_admin` user-role assignment is intentionally not
guessed by migration; assign the intended user through `casbin_rule` once when
bootstrapping an existing database, then restart the service so Casbin reloads
the policy:

```sql
INSERT IGNORE INTO casbin_rule (ptype, v0, v1)
VALUES ('g', 'user:<system_user.id>', 'super_admin');
```

Generate Swagger from Go annotations:

```bash
./scripts/generate-swagger.sh
```
