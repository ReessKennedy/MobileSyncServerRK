# Sync Overview

This server provides a minimal, local-first sync loop with no authentication. The API is append-only on the server via a changes log, with a cursor-based pull.

## High-level flow

1. Client writes locally and emits change events.
2. Client `POST /sync/push` with a batch of events.
3. Server applies events (idempotent by `event_id`) and appends each event to the `changes` log.
4. Client `GET /sync/pull?cursor=...&limit=...` to fetch server-side changes after its last cursor.
5. Client applies those changes locally and advances its cursor to `next_cursor`.

## Push details

- Endpoint: `POST /sync/push`
- Idempotency: `event_id` is stored in `seen_events`. Duplicate events are ignored.
- Persistence:
  - For `entity_type = note`, the server upserts into the `notes` table for `op = upsert`, or soft-deletes (`deleted_at`) for `op = delete`.
  - For other entity types, the server only records the change in the `changes` table.
- All events are appended to `changes` with an auto-incrementing `id` used as the cursor.

## Pull details

- Endpoint: `GET /sync/pull?cursor=0&limit=500`
- Cursor semantics: `cursor` is the last seen `changes.id`. The server returns rows with `id > cursor`.
- `next_cursor` is the highest `changes.id` returned. If no changes are returned, it stays the same as the input cursor.
- Limit: default 500, max 1000. Out-of-range values are normalized to 500.

## Data shape

- `PushRequest` contains:
  - `client_id`
  - `events[]` with `event_id`, `entity_type`, `entity_id`, `op`, and `payload`.
- `PullResponse` contains:
  - `next_cursor`
  - `changes[]` with `id`, `entity_type`, `entity_id`, `op`, and `payload`.

## Notes payload

For `entity_type = note`, the payload should include fields from the `notes` table (see `migrations/001_init.sql`). Typical fields include `id`, `type`, `text`, `is_completed`, `created_at`, `updated_at`, and `deleted_at`.

## Conflict behavior

This PoC does not implement conflict resolution beyond last-write-wins via upsert semantics in MySQL. If multiple clients write the same note, the most recent push to the server will overwrite prior state.

## Auth and security

There is no authentication or authorization. Any client can push or pull.
