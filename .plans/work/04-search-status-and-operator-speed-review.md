# 04 Search, Status Picker, and Operator Speed - Review

## Planned Scope
- Live search/filter flow.
- Inline status picker with optional reverse cycle shortcut.
- Keyboard shortcuts tuned for low-keystroke operator loops.

## Actual Implementation
- Kept `/`-driven Search mode and validated live filtering on every keystroke.
- Search filtering applies across `name`, `task`, `branch`, and `tmux_session`.
- `Enter` in Search mode now exits to Normal while preserving the active filter.
- `Esc` in Search mode clears the filter and rebuilds the full board view.
- Replaced status-change behavior with explicit inline status picker:
  - `s` opens Status Picker mode for the selected agent.
  - `1..5` choose target status (`idle`, `running`, `blocked`, `review`, `done`).
  - `Enter` applies the selected status and returns to Normal mode.
  - `Esc` cancels picker mode.
- Added power-user shortcut `S` for immediate reverse status cycle.
- Added status transition toasts in notices and expanded footer/help hints.
- Status transitions now write through DB (`UpdateAgentStatusByID`) and
  immediately refresh active agents from DB (`ListActiveAgents`).
- Selection/focus is re-anchored to the updated agent after transition.

## Plan vs Actual Notes
- Inline status picker is implemented as a dedicated mode rendered as a focused
  status bar, preserving existing form/help/confirm overlays.
- Backward cycle (`S`) applies directly without opening picker, matching the
  requested optional power-user flow.

## Validation Evidence
- `go test ./internal/tui` passed.
- `go test ./...` passed.
