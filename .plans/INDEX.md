# Work Item Index

Use this checklist to track completion status of work items under
`.plans/work/`.

## Work Item Completion Checklist

- [x] `000-index` (`.plans/work/000-index.md`)
- [x] `00-repository-init` (`.plans/work/00-repository-init.md`)
- [x] `01-bootstrap-and-foundation` (`.plans/work/01-bootstrap-and-foundation.md`)
- [x] `02-data-layer-and-schema` (`.plans/work/02-data-layer-and-schema.md`)
- [x] `03-board-layout-navigation-and-modes` (`.plans/work/03-board-layout-navigation-and-modes.md`)
- [x] `04-search-status-and-operator-speed` (`.plans/work/04-search-status-and-operator-speed.md`)
- [x] `05-agent-crud-forms-and-guardrails` (`.plans/work/05-agent-crud-forms-and-guardrails.md`)
- [x] `06-tmux-attach-live-state-and-errors` (`.plans/work/06-tmux-attach-live-state-and-errors.md`)
- [x] `07-dead-session-recovery-flow` (`.plans/work/07-dead-session-recovery-flow.md`)
- [x] `08-profile-driven-item-creation` (`.plans/work/08-profile-driven-item-creation.md`)
- [x] `09-cleanup-lifecycle-and-prune` (`.plans/work/09-cleanup-lifecycle-and-prune.md`)
- [x] `10-cli-status-parity` (`.plans/work/10-cli-status-parity.md`)
- [x] `11-polish-acceptance-and-release` (`.plans/work/11-polish-acceptance-and-release.md`)
- [x] `12-profile-config-root-and-end-user-examples` (`.plans/work/12-profile-config-root-and-end-user-examples.md`)
- [x] `13-git-worktree-branch-bound-sessions` (`.plans/work/13-git-worktree-branch-bound-sessions.md`)
- [x] `14-fix-create-form-branch-name-regression` (`.plans/work/14-fix-create-form-branch-name-regression.md`)
- [x] `15-tmux-bootstrap-and-recovery-parity` (`.plans/work/15-tmux-bootstrap-and-recovery-parity.md`)
- [ ] `16-rename-board-binary-to-yaama` (`.plans/work/16-rename-board-binary-to-yaama.md`)
- [ ] `17-agent-hook-cli` (`.plans/work/17-agent-hook-cli.md`)
- [ ] `18-tmux-bootstrap-system-tests` (`.plans/work/18-tmux-bootstrap-system-tests.md`)
- [x] `19-action-logger` (`.plans/work/19-action-logger.md`)

## Notes

- Mark an item complete only after its done criteria have been validated.
- Add a new checklist entry here whenever a new file is added to `.plans/work/`.

## When marking a work items is complete
1. Add a new .md file postfixed with -rewview (e.g. for 00-repository-init.md add 00-repository-init-review.md).
2. Compare the current changes that have been made (including user udpates) against what was planned.
3. Summerise any differences or deviations from the plan.
4. Make sure the file is added to the commit that also updates this work tracking  file.


## Plan vs Actual (This Update)

- Actual tracking scope now follows work items (not spec files), per operator
  request.
- Work item `03-board-layout-navigation-and-modes` completed with DB-backed
  board columns, deterministic navigation, mode escape hierarchy, and
  regression tests for focus/mode transitions.
- Work item `04-search-status-and-operator-speed` completed with live
  name/task/branch/session filtering, inline status picker (`s` + `1..5` +
  `Enter`), reverse cycle shortcut (`S`), status update toasts, and immediate
  post-transition board refresh.
- Work item `05-agent-crud-forms-and-guardrails` completed with keyboard-first
  create/edit forms, including a low-friction create wizard (`profile ->
  task`) with inferred name/session values, validation + dirty-form
  protection, soft archive + guarded hard prune flows, and regression tests
  covering CRUD + destructive guardrails.
- Work item `06-tmux-attach-live-state-and-errors` completed with `Enter`
  attach/switch handoff via `tea.ExecProcess`, startup/runtime tmux
  availability handling, periodic DB/tmux reconciliation ticks, visible
  `dead`/`stale` runtime indicators, severity-based toast UX, persistent DB/tmux
  runtime banners, and regression coverage for missing sessions,
  tmux-unavailable attach, and refresh failure behavior.
- Work item `07-dead-session-recovery-flow` completed with `r`-based dead-session
  recreation from the board, working-directory validation before tmux creation,
  persisted `last_error` on recreate failures, immediate attach after successful
  recreate, and regression tests for missing/invalid paths and recovery success.
  Detached recreate/bootstrap session creation now also enforces
  `destroy-unattached=off` on target sessions to prevent auto-destruction
  after client detach in tmux environments that default to
  `destroy-unattached=on`.
- Work item `08-profile-driven-item-creation` completed with TOML-backed profile
  loading/validation from `~/.config/yaama/profiles`, relative script/layout path
  resolution against `~/.config/yaama`, runtime value derivation (`working_dir`,
  `branch`, startup command args) with repo-path fallback to current directory,
  persisted create-time runtime metadata, tmux bootstrap orchestration
  (before/after hooks, windows/panes, startup focus, agent command launch),
  explicit session-targeted layout sourcing + hook session context isolation, and
  regression tests covering runtime persistence and profile load failures.
- Work item `09-cleanup-lifecycle-and-prune` completed with deterministic
  cleanup ordering (tmux kill -> optional work-dir prune adapter -> profile
  cleanup hooks -> final cleanup state transition), explicit archive/prune UX
  semantics, persisted stage-aware `last_error` failure context, idempotent retry
  behavior for recoverable cleanup failures, and regression tests covering
  tmux/work-dir/script partial-failure paths.
- Work item `10-cli-status-parity` completed with a dedicated
  `board status <status>` command contract (`--task`, `--activity`, optional
  `--branch`), strict accepted-status validation + non-zero invalid-input exits,
  tmux-session-derived agent resolution (`$TMUX` + `display-message -p '#S'`),
  session-keyed status/metadata/heartbeat writes that preserve unset fields, and
  actionable outside-tmux / missing-agent failure guidance with focused tests.
- Work item `11-polish-acceptance-and-release` completed with runtime badge copy
  consistency (`[DEAD]`/`[STALE]`), actionable footer/help/empty-state coverage,
  first-run bootstrap acceptance testing for fresh DB initialization, release
  readiness checks via cross-platform `make release-check` in CI, and expanded
  README operator runbook + troubleshooting + post-v1 scope freeze notes.
- Work item `12-profile-config-root-and-end-user-examples` completed with
  profile-root standardization to `~/.config/yaama` only (no legacy `yaam`
  compatibility path), updated plan/spec/docs references, and end-user setup
  assets under `examples/profiles/` (`default.toml`, `dev.toml`, and usage
  guide) linked from `README.md`.
- Work item `15-tmux-bootstrap-and-recovery-parity` completed with a unified
  `tmux.BootstrapSession` pipeline shared between create and dead-session
  recovery, session-scoped `YAAMA_TMUX_SESSION` / `YAAMA_WORKING_DIR` env
  injection (plus pane-0 export shim), `AgentCommand=nil` recovery so the
  agent process is not relaunched, profile-missing fallback with operator
  warning toast, parse-error abort with persisted `last_error`, and unit
  coverage for env ordering, profile-applied recovery, missing-profile
  fallback, and parse-error abort.
- Work item `19-action-logger` completed with a new `internal/logging`
  package (slog text handler, XDG state path resolution, 5 MiB
  rotate-on-open), wiring through `startup.State` and `cmd/board`,
  per-step `tmux.BootstrapSession` instrumentation, `profile.LoadWithLogger`,
  TUI recovery/refresh log lines, an `L` key + help-overlay surface for
  the resolved log path, and README/AGENTS.md documentation. Tests cover
  level/path/rotation in the logging package and the `L`-key toast paths
  in the TUI.
- Work item `13-git-worktree-branch-bound-sessions` completed with a required
  branch create-step (`profile -> task -> branch`), branch safety validation,
  git-repository enforcement for profile-backed sessions, native
  `git worktree` create/remove lifecycle wiring (including deterministic
  `<repo_parent>/.yaama-worktrees/<session-slug>` paths), cleanup-stage
  conversion from adapter prune to worktree remove semantics, and expanded
  unit coverage for branch validation, git-path failures, and worktree wrapper
  behavior.
