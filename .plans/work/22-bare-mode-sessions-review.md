# 22 - Bare-Mode Sessions · Review

Comparison of the shipped implementation against
`.plans/work/22-bare-mode-sessions.md`.

## Implementation summary

- **Schema.** `internal/db/schema/002_agent_mode.sql` adds
  `agents.mode TEXT NOT NULL DEFAULT 'worktree' CHECK (mode IN ('worktree','bare'))`
  and `idx_agents_mode`. Existing rows backfill to `'worktree'` via the
  DEFAULT.
- **Queries.** `internal/db/queries/agents.sql` surfaces `mode` on
  `ListAgents`, `ListActiveAgents`, `ListAgentsByStatus`, `GetAgentByID`,
  `GetAgentByTmuxSession`, `CreateAgent`, and `UpdateAgent`. To avoid sqlc
  generating divergent row types, the column is appended after
  `updated_at` in every SELECT/RETURNING so the projection matches the
  generated `Agent` struct field order. sqlc was regenerated; no
  hand-written changes to `internal/db/generated/`.
- **Form.** `newFormState` for the create wizard prepends a Mode field
  (`worktree` | `bare`, defaulting to `worktree`) and sets `active = 1`
  so initial focus lands on Profile and the common worktree keystroke
  count is unchanged. The Edit form gains Mode as a fourth field.
- **Cycling.** `cycleCreateMode(delta int)` toggles the value and
  `swapWizardModeField` rebuilds the fourth wizard slot between
  Branch (required) and Working Dir (required, defaulted to
  `os.Getwd()`). Left/Right and `h`/`j`/`k`/`l` cycle when Mode is
  focused.
- **Backspace guard.** Now keyed off the active field's `key`
  (`mode` or `profile_name`) instead of literal index `0`, so the
  guard survives field reordering.
- **Validation.** `validateForm` skips branch validation in bare
  mode, requires `working_dir`, and runs `os.Stat` to verify the
  directory exists and is a directory.
- **Persist.** `persistCreateForm` branches on Mode:
  - `worktree` — unchanged.
  - `bare` — skip `gitworktree.Ensure`; resolve the chosen path to
    an absolute path; call the new
    `profile.ResolveRuntimeValuesBare(cfg, absDir)`; bootstrap tmux
    at that directory; persist `Mode="bare"`, `Branch=NULL`.
- **Cleanup.** `pruneRequiresForce` and `removeCleanupWorktree` gate
  on `agentModeOrDefault(target.Mode) == "worktree"`. The hard-prune
  confirm body switches to a bare-aware copy when the target is a
  bare session.
- **Edit form.** Allows editing Mode; on save, surfaces a warning
  toast when the value changed so the operator knows the on-disk
  state was not reconciled.
- **Profile package.** Added
  `profile.ResolveRuntimeValuesBare(cfg Config, workingDir string)`
  to avoid overloading the worktree resolver with an empty branch.
- **Tests.** Added five tests in `internal/tui/update_test.go`
  plus updated `view_test.go` field indices:
  - `TestCreateWizardOpensWithProfileFocusedAndUpLandsOnMode`
  - `TestCreateWizardCyclesModeAndSwapsBranchForWorkingDir`
  - `TestBareCreateSkipsWorktreeAndPersistsMode`
  - `TestBareValidationRejectsMissingWorkingDir`
  - `TestBareCleanupSkipsWorktreeRemove`
  - `TestWorktreeCleanupStillCallsRemoveWorktree`
- **Docs.** README gains a "Bare sessions" subsection under "Profiles".
  AGENTS.md notes the `mode` column and gates.

## Deviations from the plan

- **Migration `DROP COLUMN`.** The `--+goose Down` step keeps the
  spec's text (`ALTER TABLE agents DROP COLUMN mode`); SQLite 3.35+
  supports `DROP COLUMN`. No deviation from the plan, but worth
  flagging if a sufficiently old SQLite is in the deployment path.
- **Column ordering in queries.** The plan placed `mode` after
  `cleanup_state` and before `created_at`. To keep the generated row
  type identical to `generated.Agent` (which appends `Mode` at the
  end), the shipped queries select `mode` after `updated_at`.
  Behavior is identical; only the on-the-wire projection order
  differs.
- **Confirm-state plumbing.** The plan suggested keying confirm-body
  copy directly off the target. The shipped change adds an explicit
  `bareSession bool` to `confirmState` so the view does not need to
  re-lookup the agent. Cosmetic difference only.
- **Edit-form warning copy.** The plan specified a toast on mode
  change; the shipped copy is
  `"Mode changed to <mode>; existing worktree (if any) is not
  removed and tmux session is not recreated."` — same intent.

## Done-criteria verification

- `n` then Enter, Enter, Enter still produces a worktree session
  (Mode rendered but not focused on open; active starts at Profile).
- `n` then Up + Right + Enter through the new flow produces a bare
  session in the current directory (verified in
  `TestBareCreateSkipsWorktreeAndPersistsMode`).
- Hard-prune of a bare session never calls `removeWorktreeFn`
  (`TestBareCleanupSkipsWorktreeRemove`).
- Hard-prune of a `mode='worktree'` row still calls
  `removeWorktreeFn` (`TestWorktreeCleanupStillCallsRemoveWorktree`).
- `gitworktree.Ensure` in `internal/tui/form.go` is reachable only on
  the worktree branch of `persistCreateForm` (the bare branch returns
  before that call).
- `examples/profiles/` and the harness/preset surface are unchanged.

## Follow-ups (none required for this work item)

- Auto-detection of "this directory is not a git repository, force
  bare mode" is out of scope per the plan and remains so.
- Bare↔worktree migration of existing rows is intentionally
  out of scope; the edit-form warning toast is the only operator
  affordance.
