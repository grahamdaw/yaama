# 08 Profile-Driven Item Creation - Review

## Planned Scope
- profile discovery and validation.
- work directory resolution (adapter or manual fallback).
- tmux session/window/pane initialization and agent command launch.

## Actual Implementation
- Added `internal/profile` package for create-flow profile handling:
  - profile discovery for `~/.config/yaama/profiles/*.toml` (`ListAvailable`),
  - profile reference validation with clear user-facing errors (`ValidateReference`),
  - TOML profile loading with required section checks for `[agent]`, `[repo]`, and `[tmux]`,
  - schema validation for required fields and pane split values,
  - relative `layout_file` and script entry path resolution against `~/.config/yaama`,
  - default argument/branch behavior and a safe `default` profile fallback.
- Extended create-flow runtime resolution in `internal/tui/form.go`:
  - wizard remains a low-friction 2-step flow (`profile -> task`),
  - runtime values are resolved from selected profile + current working directory fallback,
  - resolved runtime metadata is persisted during create:
    - `tmux_session` (inferred stable ID),
    - `working_dir`,
    - `branch`.
- Added tmux bootstrap orchestration in `internal/tmux/bootstrap.go`:
  - before-start hooks run first,
  - tmux session creation and window/pane materialization from profile layout,
  - optional layout file sourcing targeted to the created session/window,
  - after-start hooks,
  - agent command launch in focused/startup window pane,
  - startup window selection after bootstrap.
- Integrated bootstrap into create-submit flow:
  - bootstrap uses inferred session name from wizard,
  - create returns actionable warning on bootstrap failure and stores `last_error`.
- Added regression coverage:
  - `internal/profile/profile_test.go` covers loader/path/default/runtime behavior,
  - `internal/tui/update_test.go` covers runtime persistence, bootstrap invocation, and profile-load failure handling.

## Plan vs Actual Notes
- Adapter-based working-directory creation is still represented as optional; this implementation completes the manual fallback path by resolving to profile repo path (or current directory when unset) and persisting resolved runtime metadata as source of truth.
- `layout_file` is sourced after session and pane creation to keep default layout creation reliable even when no external layout snippet is configured.
- bootstrap hardening update: shell hooks now run with explicit session/working-dir env (`YAAMA_TMUX_SESSION`, `YAAMA_WORKING_DIR`) and cleared inherited `TMUX` so setup/layout scripts do not accidentally apply to another active client session.

## Validation Evidence
- `go test ./internal/profile ./internal/tui` passed.
- `make test` passed.
