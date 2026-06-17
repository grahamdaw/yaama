# 22 - Bare-Mode Sessions (no worktree, no branch)

## Goal
Let an operator start a yaama session that is *only* a tmux session
running the configured agent inside a chosen directory — no branch, no
worktree, no `git worktree add` / `remove`. Today every session is
hard-bound to a worktree + branch (work item 13), and the create form
unconditionally calls `gitworktree.Ensure`
(`internal/tui/form.go:339`). Bare mode covers three real cases that
the worktree flow makes awkward:

1. Ad-hoc exploration sessions where a dedicated branch + worktree is
   pure overhead.
2. Sessions inside directories that are not git repositories at all.
3. Main-branch / current-checkout work where the operator wants the
   agent to act in-place rather than in `.yaama-worktrees/<slug>`.

The current worktree-bound flow covers ~99% of session creates and
must stay frictionless. Bare mode is a deliberate, discoverable opt-in
*inside the existing create form* — not a separate command, not a
separate hotkey.

## Branch Name
`feat/22-bare-mode-sessions`

## Scope

- **Form (Option B, refined).**
  - Prepend a **Mode** field to the create wizard. Values:
    `worktree` (default) | `bare`.
  - When `n` opens the form, `active` starts at the Profile field —
    Mode is rendered but skipped on initial focus so the common
    keystroke count is unchanged. Up / Shift-Tab from Profile lands
    on Mode; left/right and `h`/`j`/`k`/`l` cycle the toggle, exactly
    like Profile selection does today
    (`cycleCreateProfile` at `internal/tui/form.go:109-124` is the
    template — add a sibling `cycleCreateMode`).
- **Bare-mode field set.**
  - In bare mode the **Branch** field is replaced by an editable
    **Working Dir** field, defaulting to `os.Getwd()`.
  - Branch field is hidden and not validated. Working Dir is required;
    validate that it exists and is a directory.
- **Persist path.** `persistCreateForm` branches on Mode:
  - `worktree` — unchanged.
  - `bare` — skip `gitworktree.Ensure`; resolve the chosen directory
    to an absolute path; call
    `profile.ResolveRuntimeValues(..., task, "")` (or a bare-friendly
    variant if it rejects empty branch); bootstrap tmux at that
    directory; persist `Branch=NULL`, `WorkingDir=<abs dir>`,
    `Mode="bare"`.
- **Schema migration.** Add `mode` column to `agents` so cleanup logic
  can distinguish worktree vs bare:
  ```sql
  -- internal/db/schema/002_agent_mode.sql
  -- +goose Up
  ALTER TABLE agents ADD COLUMN mode TEXT NOT NULL DEFAULT 'worktree'
      CHECK (mode IN ('worktree', 'bare'));
  CREATE INDEX IF NOT EXISTS idx_agents_mode ON agents(mode);
  -- +goose Down
  DROP INDEX IF EXISTS idx_agents_mode;
  ALTER TABLE agents DROP COLUMN mode;
  ```
  Existing rows backfill to `'worktree'` via the DEFAULT, preserving
  prune behavior for legacy sessions. Update
  `internal/db/queries/agents.sql` to surface `mode` in `CreateAgent`,
  `ListActiveAgents`, `ListAgentsByStatus`, `GetAgent`,
  `GetAgentByTmuxSession`, and `UpdateAgent`; regenerate
  `internal/db/generated/` via sqlc.
- **Cleanup path.** In the cleanup handler
  (`internal/tui/update.go:464-479`), gate `removeCleanupWorktree`
  and `pruneRequiresForce` on `target.Mode == "worktree"`. Bare prune
  still kills tmux + runs profile cleanup hooks + marks
  `cleanup_state=pruned`, but never touches the working directory.
  Update the prune-confirm body (`internal/tui/view.go:303`) to
  reflect this for bare targets.
- **Profile integration.** No profile schema change.
  `ResolveRuntimeValues` is called with an empty branch in bare mode
  — verify it tolerates that during phase 1; if not, add a
  bare-friendly variant rather than overloading the worktree path.
- **View.** Update the wizard renderer at
  `internal/tui/view.go:333-371`:
  - Render Mode line as stage 0 (visible from the start so the
    affordance is discoverable, but not focused on open).
  - In bare mode, label stage 3 "Working Dir" instead of "Branch"
    and adjust the help footer.
  - `Inferred name` / `Inferred tmux session` lines stay.
- **Edit form.** Allow editing `mode` on existing rows
  (`formPurposeEdit`) for consistency. Toggling mode on an existing
  session is operator-beware and does **not** retroactively create
  or remove a worktree; the persist path surfaces a warning toast
  on save so the operator knows the on-disk state did not change.

## Non-goals
- Auto-detection of "this directory is not a git repo, force bare
  mode." Operator selects the mode explicitly.
- Worktree-to-bare or bare-to-worktree migration of existing sessions.
- A separate CLI subcommand (`yaama bare`). Bare mode lives inside
  the existing TUI create flow.
- Profile-level default mode. The form remembers nothing across
  sessions; default is always `worktree`.
- Validating that the chosen bare directory is or isn't inside a git
  repo. It can be either.

## Design

### Form behavior summary

| State | Fields shown | `active` on open | Validation |
|---|---|---|---|
| `mode=worktree` (default) | Mode · Profile · Task · Branch | 1 (Profile) | Branch required + `gitworktree.ValidateBranch` |
| `mode=bare` | Mode · Profile · Task · Working Dir | 1 (Profile) | Working Dir required + `os.Stat` is dir |

### What moves where

- `internal/db/schema/002_agent_mode.sql` — new migration (above).
- `internal/db/queries/agents.sql` — surface `mode` everywhere agent
  rows are read or written. Regenerate `internal/db/generated/`.
- `internal/tui/model.go:122-136` — no struct change required; reuse
  the existing `formField` / `formState` shape.
- `internal/tui/form.go:47-78` — `newFormState`: prepend Mode; set
  `active = 1`; build fields dynamically so Branch ↔ Working Dir
  swap based on the current Mode value.
- `internal/tui/form.go:109-124` — add `cycleCreateMode` modeled on
  `cycleCreateProfile`.
- `internal/tui/form.go:173-235` — `validateForm`: in bare mode skip
  branch validation, require `working_dir`, check directory exists.
- `internal/tui/form.go:307-407` — `persistCreateForm`: branch on
  Mode as described; write `Mode` param; `CleanupState: "active"`.
- `internal/tui/update.go:291-391` — `handleFormMode`: extend
  left/right and `h`/`j`/`k`/`l` cycling when the active key is
  `mode`. Update the backspace guard at line 356 to compare against
  the profile/mode key, not literal index 0.
- `internal/tui/update.go:445-479, 515-545` — gate worktree removal
  on `target.Mode == "worktree"`.
- `internal/tui/view.go:301-371` — render Mode line, swap Branch ↔
  Working Dir label, update confirm-body wording.
- `internal/gitworktree/gitworktree.go` — **no change**.
- `internal/tmux/bootstrap.go` — **no change**; `BootstrapSession`
  already takes `WorkingDir` and runs
  `tmux new-session -d -s <name> -c <dir>`.

### Failure modes (load + run time)

| Condition | Behavior |
|---|---|
| Mode = bare, Working Dir empty | Form validation error "required" on the Working Dir row. |
| Mode = bare, Working Dir does not exist | Form validation error "directory not found: <path>". |
| Mode = bare, `ResolveRuntimeValues` rejects empty branch | Fall back to a bare-friendly resolver variant; preserve all other profile semantics. |
| Mode = worktree on a non-git dir | Today's existing error path, unchanged. |
| Editing an existing row to change `mode` | Allowed; UX warning toast on save: "mode changed; existing worktree (if any) is not removed and tmux session not recreated." |

### Tests

`internal/tui/update_test.go` + `internal/tui/form_test.go`:

- Form opens with `active==1`; Up lands on Mode.
- Left/right and `h`/`l` cycle Mode between `worktree` and `bare`.
- Bare mode hides Branch row, shows Working Dir, accepts arbitrary
  path, rejects missing path.
- `persistCreateForm` in bare mode does *not* invoke
  `ensureWorktreeFn`, writes `Mode="bare"` and `Branch=NULL`, and
  bootstraps tmux at the chosen dir.
- Cleanup of a `Mode="bare"` agent does *not* invoke
  `removeWorktreeFn`.
- Cleanup of a `Mode="worktree"` agent still calls `removeWorktreeFn`
  (regression).
- Schema migration: existing rows backfill to `mode='worktree'`.

## Phases

### Phase 1: Schema + queries + model
1. Add `internal/db/schema/002_agent_mode.sql` and regenerate sqlc.
2. Propagate `Mode` through `internal/tui/model.go` (no struct change
   beyond passing the new generated field) and any agent
   construction sites (recovery, fallback in-memory rows in
   `persistCreateForm`).
3. Tests that just write/read agents pass on the new column.

### Phase 2: Form + view wiring
1. Prepend the Mode field in `newFormState`; set `active = 1`; render
   Mode in the wizard view.
2. Add `cycleCreateMode` and extend `handleFormMode` to cycle Mode
   on left/right/`h`/`j`/`k`/`l` when the active key is `mode`.
3. Make the fourth slot dynamic: Branch when `mode=worktree`,
   Working Dir when `mode=bare`. Default Working Dir to
   `os.Getwd()`.
4. Update validation per the table above.
5. Confirm `n`-flow keystroke count is unchanged for worktree (the
   dominant path) via the existing `update_test.go` keystroke
   harness.

### Phase 3: Persist + cleanup branching
1. Wire the bare branch of `persistCreateForm`: skip
   `ensureWorktreeFn`, resolve absolute dir, bootstrap tmux, write
   `Mode="bare"` + `Branch=NULL`.
2. Gate `removeCleanupWorktree`/`pruneRequiresForce` on
   `target.Mode == "worktree"`. Update the confirm-body copy at
   `internal/tui/view.go:303` for bare targets.
3. Add the cleanup parity + bare-create integration tests above.

### Phase 4: Docs
1. README "Sessions" section gains a "Bare sessions" subsection
   describing the toggle, the directory default, and what cleanup
   does (and doesn't do).
2. AGENTS.md operator-runbook pointer to the new subsection.
3. `.plans/work/22-bare-mode-sessions-review.md` per the INDEX
   convention, in the same commit that ticks the box.

## Done Criteria

- `n` from the board still requires three keystrokes (Profile, Task,
  Branch) for a worktree session — measured by an `update_test.go`
  keystroke harness.
- `n` then Up + Right + Enter through the new flow produces a bare
  session in the current directory with no entry under
  `.yaama-worktrees/`, no new branch, and `tmux attach -t <name>`
  lands in cwd.
- Hard-prune of a bare session kills tmux and marks
  `cleanup_state=pruned` without invoking `git worktree remove`.
- Hard-prune of a pre-existing (legacy) row that has
  `mode='worktree'` (via DEFAULT backfill) behaves exactly as before.
- `grep -rn "gitworktree.Ensure" internal/tui/form.go` shows the
  call is reachable only on the worktree branch of
  `persistCreateForm`.
- `examples/profiles/` and the existing harness/preset surface are
  unchanged — this work item touches no profile TOML schema.

## Dependencies / Sequencing

- Depends on work item 13 (worktree-bound sessions) being the current
  baseline; it is, per `.plans/INDEX.md`.
- Independent of #21 (profile TOML); no profile schema change here.
- No follow-up work items spawned.
