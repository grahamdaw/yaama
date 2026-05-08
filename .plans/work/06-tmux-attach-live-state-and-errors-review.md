# 06 tmux Attach, Live State, and Error Handling - Review

## Planned Scope
- tmux session discovery and attach/switch behavior.
- Periodic DB + tmux reconciliation and live-state indicators.
- Explicit, non-silent runtime error UX and tests for failure paths.

## Actual Implementation
- Expanded `internal/tmux` wrappers with:
  - session listing (`list-sessions -F "#{session_name}"`),
  - current session resolution (`display-message -p "#S"`),
  - attach/switch command selection (attach outside tmux, switch inside tmux),
  - actionable tmux-unavailable error surface.
- Added startup tmux capability detection and surfaced degraded behavior through
  notices/banner copy.
- Added board-level refresh loop:
  - periodic refresh tick (`5s`) in `Init`,
  - immediate refresh command path for post-action reconciliation,
  - DB + tmux session reconciliation in update pipeline.
- Added runtime reconciliation metadata to the TUI model (`liveSessions`,
  stale threshold, banner/toast state) and helper predicates for:
  - `dead` (tmux session missing),
  - `stale` (`running` + old `updated_at` timestamp).
- Wired `Enter` in Normal mode to attach/switch into selected live tmux session
  using `tea.ExecProcess`, with clear user-facing errors for:
  - no selection,
  - tmux unavailable,
  - missing target session,
  - attach command failures.
- Replaced plain notices with severity-based toast rendering and expiration:
  - success (`2s`),
  - warning (`4s`),
  - error (`6s`).
- Added persistent runtime banner behavior for DB lock/unavailable and tmux
  availability issues.
- Updated card/detail rendering and stats:
  - card badges include `[dead]` / `[stale]`,
  - detail panel includes explicit runtime state,
  - header dead count now reflects missing live tmux sessions.
- Added regression tests in `internal/tui/update_test.go` for:
  - `Enter` with missing sessions,
  - `Enter` when tmux is unavailable,
  - refresh failure with persistent lock banner,
  - live-session attach command creation.

## Plan vs Actual Notes
- Immediate DB refresh after state changes was already present from prior work
  item logic; this update focused on periodic reconciliation + attach/runtime
  reliability and surfaced runtime state in UI.
- Error toast timeout currently follows the bounded timeout path (`6s`) and does
  not add an extra explicit-dismiss key beyond the existing global key loop.

## Validation Evidence
- `go test ./internal/tui` passed.
- `go test ./...` passed.
