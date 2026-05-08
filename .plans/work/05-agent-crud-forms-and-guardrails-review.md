# 05 Agent CRUD, Forms, and Guardrails - Review

## Planned Scope
- New/edit modal forms.
- Validation and dirty-form handling.
- Soft archive and explicit destructive confirms.

## Actual Implementation
- Added a reusable keyboard-driven form framework with shared field metadata and
  state for create/edit flows.
- Simplified create flow to a two-step wizard on `n`:
  - step 1 selects a profile from discovered `~/.config/yaam/profiles/*.toml`
    (or `default` when none exist),
  - step 2 captures task text.
- Create flow now infers both `name` and `tmux_session` as
  `<lowercase-task-id>-<profile>` and persists those inferred values.
- Implemented edit flow (`e`) that preloads selected agent values and updates
  the existing row in place.
- Added form validation for:
  - required wizard fields (`profile_name`, `task`) for create,
  - inferred tmux session uniqueness (DB-backed when available, in-memory
    fallback),
  - allowed status set + session uniqueness + profile checks for edit flows.
- Added dirty-state protection on `Esc`, with discard confirmation before
  leaving form mode.
- Added destructive-action confirm states:
  - `d` archives selected agent (`cleanup_state = archived`),
  - `D` triggers hard prune flow with explicit force step when working
    directory is non-empty.
- Wired archive/prune actions to DB queries when available and kept in-memory
  fallback behavior for tests.
- Added/update regression tests in `internal/tui/update_test.go` for:
  - wizard create + inferred naming/session behavior,
  - validation blocking duplicate inferred session submissions,
  - edit save flows,
  - archive and forced-prune destructive guardrails.
- Updated board help/footer/confirm copy and added a rendered form overlay with
  inline field errors.

## Plan vs Actual Notes
- Hard prune is intentionally separated behind `D` and, when required, an
  additional force key (`f`) before final `Enter`, satisfying explicit-force
  guardrails for non-empty working directories.
- Original broader create-form field set was intentionally reduced to a
  low-friction wizard (`profile -> task`) to improve operator speed; remaining
  runtime/profile expansion is deferred to work item `08`.
- Profile reference validation is implemented as file existence checks; richer
  profile-driven runtime resolution remains in scope for work item `08`.

## Validation Evidence
- `go test ./internal/tui` passed.
- `go test ./...` passed.
