# Agent Board — Spec Iteration (v0.2)

This iteration keeps the architecture in `000_INITIAL_SPEC.md` and sharpens product behavior, UX decisions, and operator workflows so implementation can proceed with fewer ambiguities.

---

## 1. Product Intent

Agent Board is an **operator console** for AI agents in tmux:

- **Observe** what every agent is doing right now.
- **Intervene** quickly when an agent is blocked or stale.
- **Coordinate** handoffs from running -> review -> done.

Primary user: a single developer/operator running multiple tmux agent sessions locally.

---

## 2. UX Principles

- **Fast keyboard-first loop**: all core actions in <= 2 keypresses.
- **Safe by default**: destructive actions require explicit confirmation.
- **Reality over optimism**: clearly show mismatch between DB intent and tmux reality.
- **Low cognitive load**: always-visible context (selection, status counts, stale/dead markers).

---

## 3. Scope Refinement for v1

### In scope (must-have)

- Kanban view by status.
- Attach/switch into selected tmux session.
- Recreate dead tmux session in the agent's existing working directory.
- Create/edit/delete agent records.
- Create new items from agent profiles (command + tmux layout/scripts).
- Status update from board and from `board status`.
- Dead session detection + visual badge.
- Search/filter by name, branch, task, session.

### Out of scope (explicit)

- Supporting non-git repository profiles.
- External worktree managers (`worktrunk`, `wt`) for v1.
- Multi-user sync / collaboration.
- Full activity timeline (single `last_activity` only).
- Multi-agent sharing a single tmux/working-dir binding.

---

## 4. Revised Information Architecture

### Card fields (compact view)

- `name` (primary label)
- `task` (single line, optional)
- badges: `dead`, `stale`, `dirty` (future), `review-blocked` (future)

### Detail panel fields (full view)

- Name
- Status
- tmux session
- agent profile
- ticket ID
- branch
- working directory
- task (full text)
- initial prompt (preview/truncated)
- last activity + "updated X ago"

### Header stats

- Total agents
- Running count
- Blocked count
- Dead sessions count

---

## 5. Interaction Model

## 5.1 Global Modes

- `Normal`: navigation and quick actions.
- `Search`: filter input active.
- `Form`: new/edit modal.
- `Confirm`: destructive confirm dialog.
- `Help`: keymap overlay.

Escape behavior:

- `Esc` closes `Help`, `Confirm`, and `Form` (if no dirty changes).
- `Esc` in dirty form opens discard confirmation.

## 5.2 Navigation semantics

- Horizontal movement switches columns and preserves row index if possible.
- If target column has fewer rows, clamp to last row.
- Empty column selection lands on column header (still focusable).

## 5.3 Status transitions

`s` opens a small inline status picker rather than blind cycling (fewer accidental transitions):

- keys: `1..5` map to statuses in configured order
- `Enter` applies
- `Esc` cancels

Shift+`s` can remain "quick cycle backward" for power users, but picker is default.

## 5.4 Search/filter behavior

- `/` enters Search mode.
- Filter applies live as user types.
- Match fields: `name`, `task`, `branch`, `tmux_session`.
- `Enter` keeps filter active and returns to Normal mode.
- `Esc` clears filter and exits Search mode.

---

## 6. Critical User Flows

## 6.1 First run

1. User runs `board`.
2. If DB file missing, create DB + run migrations automatically.
3. If no agents exist, show empty state with hints:
  - `n` create first agent
  - `board status running --task "..."`
4. Footer displays "DB initialized at " once.

## 6.2 Attach to agent

1. Select card.
2. Press `Enter`.
3. If session alive -> attach/switch.
4. If dead -> show non-blocking error toast: "Session not found. Press `e` to update session or `d` to remove."

## 6.3 Handle blocked agent

1. Filter by `blocked` or navigate to Blocked column.
2. Open detail panel, review task/last activity.
3. Press `e` to update task with blocker context.
4. Press `s` and move to `running` when resolved.

## 6.4 Review handoff

1. Agent marks itself `review` via CLI.
2. Operator sees Review column count bump.
3. Operator attaches to inspect work and decides:
  - set `done` if accepted
  - set back to `running` if changes requested

---

## 7. Error and Recovery UX

- **tmux unavailable**: banner "tmux binary not found in PATH"; attach action disabled.
- **DB locked/unavailable**: show persistent error bar with retry every tick.
- **Migration failure on startup**: fail fast with actionable message + migration path.
- **Invalid status from CLI**: return non-zero with accepted values listed.

Toast style:

- Success: green, auto-dismiss 2s.
- Warning: yellow, auto-dismiss 4s.
- Error: red, requires keypress or 6s timeout.

---

## 8. Data Contract Updates

Recommended additions:

- `last_heartbeat_at DATETIME NULL`
- `last_error TEXT NULL`
- `cleanup_state TEXT NOT NULL DEFAULT 'active' CHECK (cleanup_state IN ('active', 'archived', 'pruned'))`
- `working_dir TEXT`
- `profile_name TEXT`
- `ticket_id TEXT`
- `initial_prompt TEXT`

Rationale:

- Better stale/dead heuristics without full event logging.
- Lets CLI status updates include "I am alive now" cheaply.
- Supports safe lifecycle cleanup without hard-delete as first step.

Stale heuristic:

- `stale` when `now - updated_at > 15m` and status is `running`.
- `dead` when tmux session missing (independent of stale).

Execution binding invariant (v1):

- One `Agent` maps to exactly one tmux session and one optional `working_dir`.
- If `branch` is set, `working_dir` is expected and used for recovery actions.
- `tmux_session` remains unique across all active rows.
- Core logic owns git worktree lifecycle for profile-backed sessions and persists the resolved `working_dir`.

---

## 8.1 tmux/working-dir Recovery and Cleanup (v1 required)

### Recovery

- When selected agent is `dead`, `Enter` should offer:
  - `r` Recreate tmux in existing working directory
  - `e` Edit session/working-dir mapping
  - `d` Delete/prune agent record
- Recreate action:
  1. Validate `working_dir` exists.
  2. If path exists, run tmux new detached session bound to that path.
  3. Update heartbeat/activity and attach/switch into recreated session.
  4. If path missing, show actionable error and stop.

### Reattach behavior

- If `working_dir` already exists for an agent, board must reuse it.
- Board must not create a second directory binding for the same agent in v1.
- Session recreation should preserve branch/working-dir metadata unless user edits it.
- Recovery reuses the persisted git-worktree-backed `working_dir` mapping.

### Cleanup policy

- Soft cleanup first: mark `cleanup_state = 'archived'` (hide by default, recoverable).
- Hard cleanup (`pruned`) requires explicit confirm and should remove stale bindings.
- If directory cleanup is requested, confirm separately from DB row cleanup.
- Never delete a non-empty `working_dir` without explicit force confirmation.

---

## 8.2 Item Lifecycle Contract (create, attach, cleanup)

### Create new item (required inputs)

- `profile_name`: selects an agent profile from config.
- `branch` (required): feature branch to create/check out in the session worktree.
- `ticket_id`: free-text external identifier.
- `initial_prompt`: initial instructions for the agent.

### Create new item (runtime behavior)

1. Load profile from `~/.config/yaama/`.
2. Resolve base repository directory:
  - use profile repository path if configured
  - otherwise use current directory where command is run
  - resolved path must be a git repository (`.git` context required)
3. Prepare branch worktree with native `git worktree` wrapper:
  - derive/create branch from the latest local `main` baseline (fast-forward from `origin/main` when available)
  - materialize worktree at `<repo_parent>/.yaama-worktrees/<session-or-task-slug>`
  - set item `working_dir` to this worktree path
4. Create tmux session using profile tmux setup (panes/windows/init scripts), with all windows/panes rooted at resolved `working_dir`.
5. Start agent command from profile with `initial_prompt` context.
6. Persist item row with profile, ticket ID, branch, and resolved `working_dir`.

### Attach to item

- Attach action always targets stored `tmux_session`.
- tmux session working directory (all panes/windows) must be `working_dir`.

### Cleanup item

Cleanup is an explicit action with ordered steps:

1. Kill tmux session for the item.
2. Remove branch worktree via native `git worktree remove` wrapper.
3. Run profile cleanup script (if configured).
4. Mark item as `pruned` (or archive first, based on chosen policy).

Failure handling:

- If session kill fails, show error and stop.
- If worktree remove fails, do not run final delete; keep row with error state.
- If cleanup script fails, record error but keep cleanup outcome visible.

---

## 9. CLI UX Iteration

Keep `board status <status>` and extend flags:

- `--task "<text>"` update task summary
- `--activity "<text>"` update last_activity
- `--branch "<name>"` optional branch refresh

Behavior:

- Resolves current session via tmux.
- Updates matching agent by `tmux_session`.
- If no matching agent:
  - default: error with recommendation to create agent
  - optional future flag `--register-if-missing`

---

## 10. Acceptance Criteria (v1)

- App starts on fresh machine with no DB/manual migration steps.
- Operator can create, edit, filter, attach, and update status without mouse.
- Dead sessions are visually obvious and do not crash interactions.
- Dead sessions can be recreated in the existing `working_dir` from the board.
- Existing `working_dir` bindings are reused instead of creating duplicates.
- New items can be created from profiles in `~/.config/yaama/`.
- Creating an item requires branch input and resolves a git worktree at `<repo_parent>/.yaama-worktrees/<session-or-task-slug>`.
- Creating an item correctly resolves repo/branch/working-dir and boots tmux layout in that worktree path.
- Cleanup kills tmux, removes git worktree branch directory, and runs cleanup script.
- `board status` works reliably from inside tmux and fails clearly outside it.
- Empty and error states provide explicit next actions.

---

## 11. Implementation Slices (updated)

1. **Boot path + empty state**: startup init, migration run, empty-state UX.
2. **Board navigation + detail panel**: stable keyboard semantics.
3. **Form + confirm UX**: create/edit/delete with guardrails.
4. **Attach + dead-state handling**: robust tmux integration.
5. **Recovery flow**: recreate tmux in existing `working_dir` with guardrails.
6. **Profile-driven create flow**: profile load, git repo validation, branch-required git worktree create/attach, tmux bootstrap.
7. **Cleanup flow**: session kill, git worktree remove, cleanup script execution with robust errors.
8. **Search + status picker**: operator speed features.
9. **CLI parity**: `board status` plus task/activity flags.
10. **Stale/dead indicators + polish**: badges, toasts, help overlays.

---

## 12. Working Directory Contract (v1)

- Canonical agent context fields are `tmux_session`, `working_dir`, and `branch`.
- Agent Board core uses only filesystem and tmux primitives.
- All profile-backed create flows are git-repository backed; non-git paths are rejected.
- Board provides a native `git worktree` wrapper (no `worktrunk`/`wt` dependency in v1) that:
  - updates the base `main` baseline (when remote-tracking ref is available),
  - creates/attaches branch worktrees,
  - resolves `working_dir` deterministically to `<repo_parent>/.yaama-worktrees/<session-or-task-slug>`,
  - removes worktrees during cleanup.
- Manual `working_dir` entry is not part of create flow for profile-backed sessions in this mode.

---

## 12.1 Profile and Script Configuration

Configuration root:

- `~/.config/yaama/`

Directory-based layout (v1):

- `~/.config/yaama/profiles/*.toml` (one profile per file)
- `~/.config/yaama/scripts/init/` (optional init scripts)
- `~/.config/yaama/scripts/cleanup/` (optional cleanup scripts)
- `~/.config/yaama/tmux/` (optional reusable tmux layout snippets)

Expected profile capabilities:

- agent command and args template
- repository path (optional)
- tmux layout/setup definition (windows, panes, init scripts)
- cleanup script hooks

Profile resolution rules:

- Profile name is derived from filename (`dev.toml` -> `dev`).
- `profile_name` on an item must reference an existing profile file.
- Missing profile should fail create flow with actionable error.
- Script paths in profiles can be relative to `~/.config/yaama/` or absolute.

Operational rule:

- Profile definitions drive session bootstrapping and cleanup behavior.
- Board stores resolved runtime values (`tmux_session`, `working_dir`, `branch`) as source of truth.

---

## 12.2 Profile TOML Schema (v1 draft)

This is a pragmatic schema for `~/.config/yaama/profiles/*.toml`.

Required sections:

- `[agent]`
- `[repo]`
- `[tmux]`

Optional sections:

- `[scripts]`
- `[[tmux.windows]]` and nested `[[tmux.windows.panes]]`

Field definitions:

- `[agent]`
  - `command` (string, required): executable command
  - `args` (array of strings, optional): static command args
- `[repo]`
  - `path` (string, optional): base repo path; fallback to current directory if unset. Resolved path must be a git repo.
  - `default_branch` (string, optional, default `main`)
- `[tmux]`
  - `session_prefix` (string, optional, default `yaam`)
  - `layout_file` (string, optional): reusable tmux layout snippet under `~/.config/yaama/tmux/`
  - `startup_window` (string, optional, default `agent`)
- `[scripts]`
  - `before_start` (array of strings, optional): run in `working_dir` before tmux bootstrap
  - `after_start` (array of strings, optional): run after session/panes are created
  - `cleanup` (array of strings, optional): run during cleanup after session kill and git worktree removal stage
- `[[tmux.windows]]`
  - `name` (string, required)
  - `focus` (bool, optional, default false)
- `[[tmux.windows.panes]]`
  - `split` (string, optional): `horizontal` or `vertical`
  - `size` (string, optional): tmux split size token (e.g. `50%`)
  - `cwd` (string, optional): pane cwd, relative values resolve from `working_dir`
  - `run` (string, optional): command sent to pane after creation

Operational defaults:

- If no windows are declared, create a single `agent` window with one pane in `working_dir`.
- The first pane of the focused window is where agent command starts unless overridden in future.
- Relative script and layout paths resolve against `~/.config/yaama/`.
- Create flow requires explicit branch input and resolves `working_dir` to `<repo_parent>/.yaama-worktrees/<session-or-task-slug>`.

Example profile: `~/.config/yaama/profiles/dev.toml`

```toml
[agent]
command = "codex"
args = ["--model", "gpt-5-codex"]

[repo]
path = "/Users/grahamdaw/repos/grahamdaw/yaama"
default_branch = "main"

[tmux]
session_prefix = "yaam"
startup_window = "agent"
layout_file = "tmux/default-layout.tmux"

[scripts]
before_start = [
  "scripts/init/check-tools.sh",
  "scripts/init/setup-env.sh"
]
after_start = [
  "scripts/init/post-attach-hint.sh"
]
cleanup = [
  "scripts/cleanup/remove-temp-files.sh",
  "scripts/cleanup/notify-cleanup.sh"
]

[[tmux.windows]]
name = "agent"
focus = true

[[tmux.windows.panes]]
cwd = "."
run = "echo 'Agent pane ready'"

[[tmux.windows.panes]]
split = "vertical"
size = "30%"
cwd = "."
run = "git status -sb"

[[tmux.windows]]
name = "ops"

[[tmux.windows.panes]]
cwd = "."
run = "watch -n 2 'git branch --show-current && git status -sb'"
```

Create flow example (with this profile):

1. User selects profile `dev`.
2. User enters `ticket_id = "CREW-301"`.
3. User enters `branch = "feat/crew-301"`.
4. Board resolves/creates git worktree directory at `<repo_parent>/.yaama-worktrees/crew-301-dev`.
5. Board creates tmux session named like `yaam-CREW-301`.
6. Board creates windows/panes, runs init hooks, and starts:
  - `codex --model gpt-5-codex`

---

## 13. Decisions Needed From You

1. Should `board status` be allowed to auto-register unknown sessions in v1?
2. Do you want inline status picker (`s`) as default over cycle behavior?
3. Should "stale" threshold be 15m or configurable in `config.toml` from day one?
4. For cleanup defaults, should archived agents be hidden by default or shown with a filter toggle?

