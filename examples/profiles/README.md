# Profile Examples

These files are starter templates for `yaama` profiles.

## Install

From the repository root:

```bash
mkdir -p ~/.config/yaama/profiles
mkdir -p ~/.config/yaama/tmux
cp examples/profiles/default.toml ~/.config/yaama/profiles/default.toml
cp examples/profiles/dev.toml ~/.config/yaama/profiles/dev.toml
cp examples/tmux/dev-layout.tmux ~/.config/yaama/tmux/dev-layout.tmux
```

Then edit the copied files and update paths/commands for your machine.

## Which file to use

- `default.toml`: minimal profile — single agent window, no worktree, no scripts.
- `dev.toml`: fuller profile with an opt-in worktree, a setup/teardown script, and an additional `ops` window with a split pane.
- `../tmux/dev-layout.tmux`: sample layout snippet used by `dev.toml`.

## Field guide

A profile is a flat TOML file. All top-level keys are optional except `[[windows]]`.

### Top-level keys

- `repo` (optional): absolute base repository path. Falls back to the current directory when empty.
- `default_branch` (optional): branch used when none is provided. Defaults to `main`.
- `worktree` (optional, default `false`): when `true`, yaama creates a git worktree under `<repo>/.yaama-worktrees/<session-slug>` on session create and removes it on hard prune. When `false`, the session uses `repo` directly.
- `setup` (optional): single command or script path. Runs after worktree creation (if any) and before tmux session creation.
- `teardown` (optional): single command or script path. Runs during cleanup (archive/prune).
- `layout_file` (optional): tmux `source-file` snippet to apply after windows are created. Relative paths resolve from `~/.config/yaama/`.
- `startup_window` (optional): window name to focus on attach if no window has `focus = true`.

Relative script paths in `setup`/`teardown` resolve from `~/.config/yaama/`. Bare commands (e.g. `"echo ready"`) are run via `sh -lc`. Both `setup` and `teardown` execute with `YAAMA_TMUX_SESSION` and `YAAMA_WORKING_DIR` in the environment.

`setup` re-runs during dead-session recovery (`r`). Write it to be idempotent.

### `[[windows]]` and `[[windows.panes]]`

Windows are created in order. The first window becomes window 0 of the tmux session — no extra "agent" window is auto-created.

Window fields:
- `name` (required): tmux window name.
- `focus` (optional): when `true`, this window is focused on attach.

Pane fields:
- `split` (optional): `horizontal` or `vertical`. Only meaningful on panes after the first in a window.
- `size` (optional): split size token like `30%`.
- `cwd` (optional): pane working directory; `"."` means the resolved session working directory.
- `run` (optional): command sent to the pane after creation.
- `agent` (optional, at most one across the whole profile): marks the pane as the agent pane. On dead-session recovery (`r`), this pane's `run` is **not** re-executed — yaama assumes the agent process is long-lived and you don't want to relaunch it. Other panes' `run` commands are re-executed normally.

## Common edits to make first

1. Set `repo` to your local git repository path.
2. Update the `run` command in the agent pane to your agent CLI.
3. Decide whether you want `worktree = true`.
4. Add/remove additional `[[windows]]` to fit your workflow.
