# 05 Agent CRUD, Forms, and Guardrails - Review

## Planned Scope
- New/edit modal forms.
- Validation and dirty-form handling.
- Soft archive and explicit destructive confirms.

## Actual Implementation
- Added a reusable keyboard-driven form framework with shared field metadata and
  state for create/edit flows.
- Implemented two create paths:
  - generic create (`n`), and
  - profile-driven create (`p`) with stricter required fields.
- Implemented edit flow (`e`) that preloads selected agent values and updates
  the existing row in place.
- Added form validation for:
  - required fields per form path,
  - allowed status set,
  - tmux session uniqueness (DB-backed when available, in-memory fallback),
  - profile reference checks against `~/.config/yaam/profiles/<name>.toml`.
- Added dirty-state protection on `Esc`, with discard confirmation before
  leaving form mode.
- Added destructive-action confirm states:
  - `d` archives selected agent (`cleanup_state = archived`),
  - `D` triggers hard prune flow with explicit force step when working
    directory is non-empty.
- Wired archive/prune actions to DB queries when available and kept in-memory
  fallback behavior for tests.
- Added/update regression tests in `internal/tui/update_test.go` for:
  - create and edit save flows,
  - validation blocking invalid submissions,
  - archive and forced-prune destructive guardrails.
- Updated board help/footer/confirm copy and added a rendered form overlay with
  inline field errors.

## Plan vs Actual Notes
- Hard prune is intentionally separated behind `D` and, when required, an
  additional force key (`f`) before final `Enter`, satisfying explicit-force
  guardrails for non-empty working directories.
- Profile reference validation is implemented as file existence checks; richer
  profile-driven runtime resolution remains in scope for work item `08`.

## Validation Evidence
- `go test ./internal/tui` passed.
- `go test ./...` passed.
