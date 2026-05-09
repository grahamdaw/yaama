# 08 - Profile-Driven Item Creation

## Goal
Create new work items from profile definitions in `~/.config/yaama/`, including repo/work-dir resolution and tmux bootstrap.

## Scope
- profile discovery and validation
- work directory resolution (adapter or manual fallback)
- tmux session/window/pane initialization and agent command launch

## Phase 1: Profile Config and Validation
### Steps
1. Implement profile loader for `~/.config/yaama/profiles/*.toml`.
2. Validate required profile sections and fields before create flow continues.
3. Resolve relative script/layout paths against `~/.config/yaama/`.
4. Return clear create-flow errors for missing/invalid profile references.

## Phase 2: Create Flow Runtime Resolution
### Steps
1. Extend existing 2-step wizard (`profile -> task`) without adding operator-heavy fields.
2. From wizard input, resolve runtime values (`working_dir`, `branch`, startup command args) from profile defaults and optional adapters.
3. Resolve base repository path from profile, then fallback to current working directory.
4. Persist resolved runtime values (`tmux_session`, `working_dir`, `branch`) as source of truth while keeping inferred `name`/`tmux_session` stable.

## Phase 3: tmux Bootstrap and Agent Start
### Steps
1. Reuse inferred tmux session name from the create wizard (`<lowercase-task-id>-<profile>`).
2. Materialize configured windows/panes and startup focus behavior.
3. Execute before/after start hooks in correct order.
4. Start agent command with task/profile-derived arguments in the intended pane.

## Definition of Done
- Operators can create items from valid profiles end-to-end.
- Session bootstrapping reflects profile layout/scripts reliably.
- Created rows contain normalized runtime metadata for later attach/recovery/cleanup.
