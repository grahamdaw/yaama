# 13 - Git Worktree Branch-Bound Sessions

## Goal
Make profile-driven session creation branch-first and git-only: every new session requires a branch, creates/attaches a native git worktree, and runs tmux/bootstrap/cleanup lifecycle inside that worktree.

## Scope
- require branch input in create flow (`profile -> task -> branch`)
- enforce git-repository validation for profile-backed create flow
- native `git worktree` wrapper for create/attach/remove lifecycle
- deterministic worktree path convention: `<repo_parent>/.yaama-worktrees/<session-or-task-slug>`
- lifecycle wiring so tmux panes and init/cleanup hooks execute in worktree directory

## Phase 1: Create Flow Contract Update
### Steps
1. Extend create flow to require non-empty branch name.
2. Validate branch input (trimmed, non-empty, safe token constraints).
3. Persist branch as required runtime metadata for profile-backed sessions.
4. Return actionable validation errors when branch is missing/invalid.

## Phase 2: Git Repo Resolution and Worktree Provisioning
### Steps
1. Resolve repo path from profile (`repo.path`) with current-directory fallback.
2. Fail create flow when resolved path is not inside a git repository.
3. Add internal git-worktree wrapper that:
   - refreshes local `main` baseline when `origin/main` is available,
   - creates/attaches requested branch worktree,
   - resolves and returns canonical worktree path at `<repo_parent>/.yaama-worktrees/<session-or-task-slug>`.
4. Set resolved worktree path as item `working_dir` before tmux bootstrap.

## Phase 3: Bootstrap and Cleanup Lifecycle Integration
### Steps
1. Ensure tmux session/windows/panes all initialize from worktree `working_dir`.
2. Ensure profile `before_start` and `after_start` hooks run with worktree `working_dir`.
3. Replace adapter-specific prune stage with native `git worktree remove` stage.
4. Ensure cleanup hooks run with consistent worktree context and persist stage-aware failures.

## Phase 4: Tests and Documentation
### Steps
1. Add unit tests for branch-required create validation.
2. Add tests for git-repo validation and worktree path convention.
3. Add tests for cleanup failure semantics when worktree removal fails.
4. Update user/docs/spec copy to state: profile-backed sessions require git repos and branch input; external worktree tools are not required.

## Definition of Done
- Create flow blocks profile-backed sessions without branch input.
- Create flow blocks non-git repository paths with actionable errors.
- New profile-backed sessions resolve `working_dir` to `<repo_parent>/.yaama-worktrees/<session-or-task-slug>`.
- tmux panes and profile hooks run in the resolved worktree directory.
- Cleanup removes the git worktree via native wrapper before final state transition.
- Tests cover happy path and stage-specific failure paths for create and cleanup.
