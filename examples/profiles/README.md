# Profile Examples

These files are starter templates for `yaama` profiles.

## Install

From the repository root:

```bash
mkdir -p ~/.config/yaama/profiles
cp examples/profiles/default.toml ~/.config/yaama/profiles/default.toml
cp examples/profiles/dev.toml     ~/.config/yaama/profiles/dev.toml
cp examples/profiles/kiro.toml    ~/.config/yaama/profiles/kiro.toml
```

Then edit the copied files and update paths/commands for your machine.

Verify a profile without launching a session:

```bash
yaama profile check default
```

## Which file to use

- `default.toml`: minimal claude-code profile using the `solo` preset.
- `dev.toml`: codex profile using the `agent+tests` preset plus a
  `before_start` hook.
- `kiro.toml`: kiro-cli profile using longhand `[[tmux.windows]]` as the
  escape hatch when no preset matches.

## Field guide

### `[agent]`

- `harness` (required): one of the registered harness ids. Run
  `yaama profile check <name>` against a profile with an unknown id to
  see the current list. Today: `claude-code`, `codex`, `copilot`,
  `kiro`, `kiro-cli`, `vscode-copilot`.
- `command` / `args` (optional): override the harness launch defaults.
- `[agent.env]` (optional): harness-specific env vars merged onto the
  agent process.

### `[repo]`

- `path` (optional): absolute base repository path. If empty, yaama
  falls back to the current directory. Must be a git repository.
- `default_branch` (optional): branch used when none is provided
  (default `main`).

### `[tmux]`

- `preset` (optional): one of `solo`, `agent+logs`, `agent+tests`,
  `agent+git+tests`. Mutually exclusive with `[[tmux.windows]]`.
- `startup_window` (optional): window name selected after bootstrap.
  Use `agent` for the automatic default agent window.

### `[scripts]`

- `before_start`, `after_start`, `cleanup` (optional): shell commands
  or script paths. Relative script paths resolve from `~/.config/yaama/`.
- Agent command launch happens after both `before_start` and
  `after_start` complete.
- `before_start` and `after_start` also run during dead-session
  recovery (`r`). The agent command is **not** relaunched on recovery —
  write `after_start` scripts so they are idempotent.
- Every shell inside the tmux session receives `YAAMA_TMUX_SESSION` and
  `YAAMA_WORKING_DIR`.

### `[[tmux.windows]]` and `[[tmux.windows.panes]]`

- `name` (required per window): tmux window name for additional windows
  created after the default agent window.
- `focus` (optional): focused window at startup (`true`/`false`).
- `layout` (optional): tmux layout name applied with `select-layout`
  after panes are created (e.g. `even-vertical`).
- `split` (optional per pane): `horizontal` or `vertical`.
- `size` (optional per pane): split size token like `30%`.
- `cwd` (optional per pane): pane working directory; `"."` means the
  resolved working directory.
- `run` (optional per pane): command sent after pane creation.

## Common edits to make first

1. Set `[repo].path` to your local git repository path.
2. Confirm `[agent].harness` matches the harness you have installed.
3. Remove or replace sample `run`/script commands you do not want.
