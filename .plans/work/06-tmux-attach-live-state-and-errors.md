# 06 - tmux Attach, Live State, and Error Handling

## Goal
Make tmux interaction resilient and transparent, including dead-session detection and recovery prompts.

## Scope
- tmux session discovery and attach/switch behavior
- periodic refresh reconciliation against DB state
- tmux and runtime error UX

## Phase 1: tmux Integration Core
### Steps
1. Implement tmux wrappers for session listing and current-session resolution.
2. Implement attach/switch logic using `tea.ExecProcess` for proper terminal handoff.
3. Detect tmux binary availability at startup and disable attach actions when missing.
4. Return actionable errors when attach target is unavailable.

## Phase 2: Live Refresh + Reconciliation
### Steps
1. Add periodic refresh tick for DB rows and tmux session list.
2. Reconcile live sessions to mark dead cards visually without deleting rows.
3. Compute stale indicators from status + `updated_at` threshold logic.
4. Trigger immediate refresh after state-changing actions instead of waiting for tick.

## Phase 3: Error Surfaces and Toasts
### Steps
1. Implement success/warning/error toast system with severity-specific timing.
2. Add persistent banner behavior for DB unavailable/locked conditions.
3. Ensure migration/tmux/runtime errors include explicit next actions.
4. Add tests around no-sessions, tmux-not-found, and refresh failure paths.

## Definition of Done
- `Enter` reliably attaches/switches to live sessions and recovers UI on return.
- Dead/stale indicators are visible and accurate.
- Error behavior is clear, non-silent, and does not crash operator workflows.
