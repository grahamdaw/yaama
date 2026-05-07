# 03 Board Layout, Navigation, and Modes - Review

## Planned Scope
- Render status columns (`idle`, `running`, `blocked`, `review`, `done`) with per-column counts.
- Add detail panel bound to current selection, including expanded metadata fields.
- Implement header stats: total, running, blocked, dead.
- Implement navigation semantics (row preservation, clamping, empty-column header focus).
- Implement mode/state handling for `Normal`, `Search`, `Form`, `Confirm`, and `Help`.
- Add `Esc` hierarchy and regression tests for focus/mode transitions.

## Actual Implementation
- Replaced placeholder board data with DB-backed agent loading via `ListActiveAgents`.
- Added board column construction and live per-column counts in column titles.
- Implemented header stats for total/running/blocked/dead plus current mode.
- Added detail panel bound to the current focused selection with expanded metadata:
  - status/session/task/branch/working dir/profile/ticket/activity/heartbeat/error/cleanup/updated.
- Implemented deterministic keyboard navigation:
  - horizontal movement preserves row index where possible,
  - selection clamps when target column is shorter,
  - empty columns keep header-row focus (`-1`) without invalid cursor behavior.
- Added explicit focus styling:
  - active column border highlighting,
  - selected card highlighting.
- Implemented mode-specific update handlers with isolated key handling for:
  - `Normal`, `Search`, `Form`, `Confirm`, and `Help`.
- Implemented `Esc` behavior hierarchy:
  - closes help/confirm,
  - clears and exits search,
  - opens discard confirm when form is dirty.
- Added help and confirm overlays and updated footer key hints.
- Added regression tests in `internal/tui/update_test.go` covering:
  - row preservation + clamping,
  - empty-column header selection,
  - dirty form discard confirmation flow,
  - escape behavior across help/confirm/search modes.

## Plan vs Actual Notes
- `dead` header stat is currently derived from agents with non-empty `last_error` in the visible set.
- Delete confirmation UI flow is present, but destructive action wiring is deferred to upcoming CRUD work item scope.

## Validation Evidence
- `go test ./internal/tui` passed.
- `go test ./...` passed.
