# 09 Cleanup Lifecycle and Prune - Review

## Planned Scope
- ordered cleanup pipeline.
- archive vs prune semantics.
- resilient error handling across tmux/work-dir/scripts.

## Actual Implementation
- Reworked cleanup transitions in `internal/tui/update.go` into a deterministic
  staged pipeline used by both archive (`d`) and prune (`D`) actions:
  1) kill tmux session,
  2) prune working directory via adapter when configured/applicable,
  3) run profile cleanup hooks,
  4) persist resulting cleanup state (`archived` or `pruned`).
- Added explicit stage-aware failure handling:
  - tmux kill failures stop cleanup and persist actionable `last_error`,
  - work-dir prune failures stop before final prune transition and persist
    actionable `last_error`,
  - cleanup script failures do not roll back cleanup outcome; they persist
    `last_error` while keeping the final cleanup state transition visible.
- Replaced hard row deletion in prune flow with cleanup-state transition to
  `pruned`, preserving auditable lifecycle state in storage.
- Updated in-memory fallback persistence to mirror DB behavior for cleanup-state
  and error metadata updates.
- Added tmux session kill support in `internal/tmux/tmux.go` with idempotent
  behavior for already-missing sessions.
- Exposed tmux shell-hook execution as `tmux.RunShellHook` for reuse in profile
  cleanup hook execution.
- Updated operator-facing copy in `internal/tui/view.go` to clarify archive vs
  hard-prune cleanup semantics and expected outcomes.
- Added regression tests in `internal/tui/update_test.go` and
  `internal/tmux/tmux_test.go` covering:
  - archive/prune happy paths with staged cleanup prerequisites,
  - stop-on-failure semantics for tmux/work-dir stages,
  - script-failure persistence without losing cleanup outcome,
  - idempotent retry behavior after transient cleanup failures,
  - tmux missing-session output handling.

## Plan vs Actual Notes
- Work-dir prune remains adapter-driven and optional by design; when no adapter
  is configured, prune continues without adapter-side directory deletion, which
  aligns with the spec's optional adapter model.
- Hard prune now maps to `cleanup_state = pruned` (auditable lifecycle) instead
  of direct row deletion; this is closer to the cleanup lifecycle contract in
  `.plans/001_SPEC_ITERATION_UX.md`.

## Validation Evidence
- `go test ./internal/tui ./internal/tmux` passed.
- `make test` passed.
