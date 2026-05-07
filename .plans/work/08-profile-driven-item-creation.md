# 08 - Profile-Driven Item Creation

## Goal
Create new work items from profile definitions in `~/.config/yaam/`, including repo/work-dir resolution and tmux bootstrap.

## Scope
- profile discovery and validation
- work directory resolution (adapter or manual fallback)
- tmux session/window/pane initialization and agent command launch

## Phase 1: Profile Config and Validation
### Steps
1. Implement profile loader for `~/.config/yaam/profiles/*.toml`.
2. Validate required profile sections and fields before create flow continues.
3. Resolve relative script/layout paths against `~/.config/yaam/`.
4. Return clear create-flow errors for missing/invalid profile references.

## Phase 2: Create Flow Runtime Resolution
### Steps
1. Build create form requiring `profile_name`, `ticket_id`, and `initial_prompt`, with optional `branch`.
2. Resolve base repository path from profile, then fallback to current working directory.
3. Resolve `working_dir` via adapter (worktrunk/worktree) when configured, else manual path strategy.
4. Persist resolved runtime values (`tmux_session`, `working_dir`, `branch`) as source of truth.

## Phase 3: tmux Bootstrap and Agent Start
### Steps
1. Create tmux session name from profile prefix + ticket/context.
2. Materialize configured windows/panes and startup focus behavior.
3. Execute before/after start hooks in correct order.
4. Start agent command with ticket/prompt arguments in the intended pane.

## Definition of Done
- Operators can create items from valid profiles end-to-end.
- Session bootstrapping reflects profile layout/scripts reliably.
- Created rows contain normalized runtime metadata for later attach/recovery/cleanup.
