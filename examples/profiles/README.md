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

- `default.toml`: minimal profile; uses only the automatic default agent window.
- `dev.toml`: fuller profile with optional scripts and one additional split-pane window.
- `../tmux/dev-layout.tmux`: sample layout snippet used by `dev.toml`.

## Field guide

### `[agent]`

- `command` (required): executable to run for your agent, for example `codex`.
- `args` (optional): static arguments passed on every launch.

### `[repo]`

- `path` (optional): absolute base repository path. If empty, yaama falls back to current directory.
  The resolved path should be a git repository.
- `default_branch` (optional): branch used when none is provided (default `main`).

### `[tmux]`

- `startup_window` (optional): window name selected after bootstrap. Use `agent` for the automatic default agent window.
- `layout_file` (optional): tmux layout snippet path. Relative paths resolve from `~/.config/yaama/`.

### `[scripts]`

- `before_start` (optional): commands/scripts run before tmux bootstrap.
- `after_start` (optional): commands/scripts run after windows/panes are created.
- `cleanup` (optional): commands/scripts run during cleanup.

Commands here can be plain shell commands (`"echo ready"`) or script paths.
Relative script paths resolve from `~/.config/yaama/`.

### `[[tmux.windows]]` and `[[tmux.windows.panes]]`

- `name` (required per window): tmux window name for additional windows created after the default agent window.
- `focus` (optional): focused window at startup (`true` or `false`). If none are focused, `startup_window` is used.
- `split` (optional per pane): `horizontal` or `vertical`.
- `size` (optional per pane): split size token like `30%`.
- `cwd` (optional per pane): pane working directory; `"."` means resolved working directory.
- `run` (optional per pane): command sent after pane creation.

## Common edits to make first

1. Set `[repo].path` to your local git repository path.
2. Confirm `[agent].command` exists in your `PATH`.
3. Remove or replace sample `run`/script commands you do not want.
