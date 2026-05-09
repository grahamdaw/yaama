# 13 Git Worktree Branch-Bound Sessions - Review

## Planned Scope
- Require explicit branch entry in profile-backed create flow.
- Enforce git repository validation before provisioning new profile-backed sessions.
- Replace adapter-specific lifecycle with native `git worktree` create/remove operations.
- Use deterministic worktree paths at `<repo_parent>/.yaama-worktrees/<session-or-task-slug>`.
- Ensure tmux bootstrap/hooks run from resolved worktree directories.
- Expand tests/docs to reflect branch-first, git-only workflow.

## Actual Implementation
- Updated create wizard and help/runbook copy to a 3-step flow (`profile -> task -> branch`).
- Added create-time branch validation (required + git-safe constraints) via
  `internal/gitworktree.ValidateBranch`.
- Added native worktree lifecycle wrapper in `internal/gitworktree`:
  - `Ensure` validates git repo context, resolves deterministic worktree paths, and provisions/attaches branch worktrees.
  - `Remove` performs native `git worktree remove --force` cleanup using the resolved working directory.
- Wired `internal/tui` create flow to:
  - resolve runtime values with explicit branch input,
  - provision git worktrees before persistence/bootstrap,
  - persist `branch` and resolved worktree `working_dir`.
- Wired cleanup prune stage to native worktree removal semantics (replacing adapter prune wiring and messaging).
- Expanded tests across `internal/tui`, `internal/profile`, and `internal/gitworktree` for:
  - branch-required and branch-safety validation,
  - git-repo validation failure handling in create flow,
  - deterministic worktree path behavior and remove flow (via command-stubbed unit tests),
  - updated runtime resolver contract requiring branch input.

## Plan vs Actual Notes
- Scope matched plan intent for branch-first, git-only profile-backed sessions.
- `origin/main` baseline refresh is implemented by syncing local `main` ref from
  `refs/remotes/origin/main` when available; no explicit network fetch step was
  added in this work item.
- Worktree wrapper tests are command-stub based to keep coverage deterministic in
  constrained/sandboxed environments.

## Validation Evidence
- `go test ./internal/gitworktree ./internal/profile ./internal/tui` passed.
- `make test` passed.
