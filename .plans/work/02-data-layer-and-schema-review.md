# 02 - Data Layer and Schema Review

## Planned Scope
- SQLite connection and migrations
- Core `agents` schema and indexes
- `sqlc` query contracts for CRUD + status transitions + session lookup
- Migration and query correctness tests

## Actual Delivery
- Added DB path resolution precedence: `--db` flag override, then `YAAMA_DB`, then default path.
- Reworked `internal/db` initialization to:
  - create parent directories and DB file,
  - open SQLite connection,
  - run embedded goose migrations automatically,
  - expose typed sqlc queries from startup state.
- Expanded `agents` schema with lifecycle and UX fields:
  - `working_dir`, `profile_name`, `ticket_id`, `initial_prompt`,
    `last_heartbeat_at`, `last_error`, `cleanup_state`, and `last_activity`.
- Added status and cleanup lifecycle constraints with CHECK clauses.
- Added indexes for status/filtering and recency lookups.
- Expanded `agents.sql` query contracts for list/create/update/delete, status updates by id/session, and cleanup state transitions.
- Generated typed query package under `internal/db/generated`.
- Added DB tests for:
  - migration/bootstrap correctness,
  - CRUD and lifecycle transitions,
  - schema constraint enforcement.

## Plan vs Actual Notes
- Included `last_activity` in schema/query contracts even though it was not listed in this task's scoped field additions; this aligns with existing spec usage and upcoming CLI parity requirements.
- No other deviations from planned scope.
