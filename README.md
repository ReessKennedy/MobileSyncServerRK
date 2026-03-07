# MobileSyncServerRK (PoC)

Minimal Go server for local-first sync (no auth) using MySQL.

## Setup

1. Create a MySQL database and user using values from `.env.example`.
2. Copy `.env.example` to `.env` and update credentials.
3. Run migrations (one-time, optional because auto-migrate runs on startup):

```bash
go run ./cmd/server --migrate
```

4. Start server (auto-migrate runs unless disabled):

```bash
go run ./cmd/server
```

Disable auto-migrate:

```bash
go run ./cmd/server --no-migrate
```

## Endpoints

- `POST /sync/push`
- `GET /sync/pull?cursor=0&limit=500`

This is a proof of concept and accepts any requests (no auth).

## Docs

- OpenAPI spec: `openapi.yaml`
- Sync overview: `docs/sync-overview.md`
