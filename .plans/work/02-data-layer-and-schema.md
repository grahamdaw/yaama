# 02 - Data Layer and Schema

## Goal
Implement durable local persistence and typed query access, including schema changes needed for UX/state signals.

## Scope
- SQLite connection and migrations
- Core `agents` schema and indexes
- SQL query layer generation with `sqlc`

## Phase 1: Database Initialization
### Steps
1. Implement DB path resolution (config default + `--db`/env override).
2. Create startup migration runner to auto-initialize a missing DB.
3. Add fail-fast error messaging for migration failures with actionable output.

## Phase 2: Schema v1 + UX Additions
### Steps
1. Create initial `agents` table migration with status enum and unique `tmux_session`.
2. Add fields required by iteration spec: `working_dir`, `profile_name`, `ticket_id`, `initial_prompt`, `last_heartbeat_at`, `last_error`, and `cleanup_state`.
3. Add status and lookup indexes needed for board rendering and filtering.
4. Define lifecycle constraints for `cleanup_state` and status values.

## Phase 3: Query Contracts with sqlc
### Steps
1. Add SQL query files for list/create/update/delete/status transitions.
2. Add lookup queries by `tmux_session` for CLI status updates.
3. Generate typed query package with `sqlc` and wire into DB package.
4. Add tests for migrations and basic CRUD/query correctness.

## Definition of Done
- Fresh startup creates DB and applies migrations automatically.
- Typed queries cover all CRUD/status/filter operations needed by board and CLI.
- Schema supports stale/dead/cleanup behavior without ad-hoc runtime state hacks.
