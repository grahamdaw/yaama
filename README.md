# yaama

Terminal-first kanban board for operating AI coding agents running inside tmux.
The app is a single Go binary with a Bubble Tea UI and SQLite persistence.

## Prerequisites

- Go `1.23.4` (pinned in `go.mod`)
- Optional: `mise` to install the pinned toolchain from `.mise.toml`

## Quick Start

```bash
git clone <repo-url>
cd yaama
mise install # optional, if you use mise
make build
make run
```

For a guided first-run flow, use `examples/WALKTHROUGH.md`.

## Profiles (End-User Setup)

`yaama` reads profiles from `~/.config/yaama/profiles/*.toml`.
For step-by-step setup and first-run guidance, see `examples/WALKTHROUGH.md`.

To get started quickly, copy the examples from this repository:

```bash
mkdir -p ~/.config/yaama/profiles
cp examples/profiles/default.toml ~/.config/yaama/profiles/default.toml
cp examples/profiles/dev.toml     ~/.config/yaama/profiles/dev.toml
cp examples/profiles/kiro.toml    ~/.config/yaama/profiles/kiro.toml
```

Then edit at least `repo.path` in each file so it points to your local git repository path.

Example files in this repo:

- `examples/profiles/default.toml`: minimal `claude-code` profile with the `solo` preset
- `examples/profiles/dev.toml`: `codex` profile using the `agent+tests` preset plus a `before_start` hook
- `examples/profiles/kiro.toml`: `kiro-cli` profile using `[[tmux.windows]]` longhand as the escape hatch

After creating profiles, start the board and press `n` to create an item from a selected profile.
Profile-backed create requires:
- a repository path that resolves to a git repository, and
- an explicit branch name (`profile -> task -> branch` wizard).

`yaama` manages native `git worktree` lifecycle directly; no external worktree manager is required. Window `0` is always created first as the default agent window and the agent command always starts there; `[[tmux.windows]]` entries (or the preset's expansion) come after.

### Bare sessions

Worktree-bound sessions are the default and cover ~99% of cases. For ad-hoc exploration, sessions inside non-git directories, or main-branch in-place work, the create wizard exposes a **Mode** toggle at the top of the form:

- `n` opens the wizard with focus on Profile, so the common keystroke count is unchanged.
- Up (or Shift-Tab) lands on Mode; left/right or `h`/`l` toggles between `worktree` (default) and `bare`.
- In `bare` mode the Branch stage is replaced by an editable **Working Dir** field, defaulted to the current directory. The directory must exist.
- Persist: no `git worktree add` is run, no branch is created. The session is launched with `tmux new-session -d -c <chosen dir>` and persisted with `mode='bare'`, `branch=NULL`.
- Cleanup: hard prune (`D`) of a bare session still kills tmux and runs `[scripts].cleanup` hooks, but never invokes `git worktree remove` and never touches the chosen directory. Legacy rows (created before this column existed) default to `mode='worktree'` and behave as before.

Bare mode is operator-selected and discoverable inside the existing TUI — there is no separate command, no auto-detection, and no profile-level default.

### Harnesses

Each profile must name a harness via `[agent].harness`. The harness contributes launch defaults (command, args, env) when the operator leaves those fields empty.

| Harness id | Hook integration | Notes |
|---|---|---|
| `claude-code` | yes | Full Claude Code hook contract (`yaama hook claude-code`) |
| `codex` | not yet | Launch defaults only |
| `kiro-cli` | not yet | Launch defaults only |
| `kiro` | not yet | Launch defaults only |
| `copilot` | not yet | Launch defaults only |
| `vscode-copilot` | not yet | Launch defaults only |

A harness that has no hook parser yet exits cleanly with a `harness has no hook integration yet` message when `yaama hook <id>` is invoked.

### Presets

`[tmux].preset = "<name>"` expands into a canned window set. Setting both `preset` and `[[tmux.windows]]` fails load.

| Preset | Expansion |
|---|---|
| `solo` | Agent window only |
| `agent+logs` | Adds a `logs` window tailing the action log path |
| `agent+tests` | Adds a `tests` window in the repo root |
| `agent+git+tests` | Adds an `ops` window with `git status -sb` and `make test` on a 30% vertical split |

### Verify a profile

Run `yaama profile check <name>` to resolve the profile against the current working directory and print the plan (working dir, branch, agent command + env, git worktree commands, tmux commands, hooks) without executing anything. Validation errors are the same ones the launch flow would surface.

```bash
$ yaama profile check default
Profile: default
Harness: claude-code

Resolved:
  1. working_dir = /Users/me/code/project
  2. branch      = main
  3. agent       = claude
...
```

## Developer Commands

```bash
make build    # build bin/yaama
make run      # run the board TUI
make test     # run go test ./...
make vet      # run go vet ./...
make lint     # run golangci-lint
make tools    # install goose/sqlc/golangci-lint
make generate # run sqlc generate
make migrate  # run local sqlite migrations
make release-check # cross-build checks for macOS/Linux artifacts
```

### Tmux system tests

`internal/tmux/bootstrap_system_test.go` exercises `BootstrapSession`
against a real `tmux` server (session env injection, layout
application, create-vs-recovery agent-command parity). The file is
gated by the `system` build tag, so the default `go test ./...` (and
`make test`) **do not** include it.

Run locally with `tmux` installed:

```bash
go test -tags=system -count=1 ./internal/tmux/...
```

In CI, the `system-tests` GitHub Actions workflow installs `tmux` on
`ubuntu-latest` and runs the same command on every PR and push to
`main`.

## Repository Layout

- `cmd/yaama/`: CLI entrypoint and inline bootstrap (config -> logging -> DB init -> tmux probe -> TUI)
- `internal/tui/`: Bubble Tea model, update loop, and rendering
- `internal/profile/`: profile TOML loading and runtime config (DB path) resolution
- `internal/db/`: DB bootstrap, migration files, and SQL queries
- `.plans/`: product specs and phased work items

## Work Tracking

Implementation order and completion state are tracked in `.plans/INDEX.md`.
Work-item scope and done criteria live in `.plans/work/`.

## Operator Runbook

- Start the board with `make run` (or `./bin/yaama` after `make build`).
- Keyboard-only core flow:
  - `n` create agent (profile -> task -> branch wizard)
  - `e` edit selected agent
  - `/` filter by name/task/branch/session
  - `s` open status picker (`1..5` then `Enter`, or `S` reverse cycle)
  - `Enter` attach to selected live tmux session
  - `r` recover selected dead session in existing `working_dir` (recreates the
    full tmux layout — windows, panes, profile `after_start` setup script,
    and `YAAMA_TMUX_SESSION`/`YAAMA_WORKING_DIR` env vars — but does **not**
    relaunch the agent process; restart it manually inside the agent window)
  - `d` archive cleanup, `D` hard prune cleanup
- From inside an agent tmux session, update without opening TUI:
  - `yaama status running --task "..." --activity "..."`
  - `yaama hook claude-code` (reads a Claude Code hook payload from stdin and
    updates status/activity/last_error for the agent bound to the current
    tmux session). Wire into `~/.claude/settings.json` hooks, for example:

    ```json
    {
      "hooks": {
        "PreToolUse":  [{"hooks": [{"type": "command", "command": "yaama hook claude-code"}]}],
        "PostToolUse": [{"hooks": [{"type": "command", "command": "yaama hook claude-code"}]}],
        "Notification":[{"hooks": [{"type": "command", "command": "yaama hook claude-code"}]}],
        "Stop":        [{"hooks": [{"type": "command", "command": "yaama hook claude-code"}]}]
      }
    }
    ```

    Additional agents can be added by registering a new parser under
    `internal/agenthook/` and invoking `yaama hook <agent-name>`.

## Troubleshooting

- **`tmux unavailable in PATH`**: install tmux or update `PATH`; attach/recover actions are disabled until available.
- **`No agent found for current tmux session`**: create/edit a board item so `tmux_session` matches your current session.
- **Dead session shown as `[DEAD]`**: select item and press `r`; if working dir is invalid, press `e` to fix mapping first. Recovery re-applies the profile layout (extra windows, after_start scripts) without relaunching the agent — start the agent yourself in the named agent window if the original profile was an agent profile. If the profile file no longer exists, recovery falls back to a minimal session and surfaces a warning toast.
- **DB lock/unavailable banners**: keep board open; it retries on refresh ticks. Validate DB path/permissions if it persists.

### Logs

yaama writes a single rolling log file for action paths (startup, tmux
bootstrap, recovery, cleanup, profile load). The TUI keeps stdout/stderr
clean — the log file is the place to look for diagnostic detail.

- **Path resolution** (first match wins):
  1. `$YAAMA_LOG_FILE` (absolute path),
  2. `$XDG_STATE_HOME/yaama/yaama.log`,
  3. `$HOME/.local/state/yaama/yaama.log`.
  Press `L` on the board to toast the resolved path. The `?` help overlay
  shows it too.
- **Level**: set `YAAMA_LOG_LEVEL=debug|info|warn|error`. Default is
  `info`. Unknown values fall back to `info` and emit a one-line warning.
- **Rotation**: at startup, if the log file exceeds 5 MiB it is renamed to
  `<path>.1` (overwriting any prior backup). Operators wanting more
  history should point `$YAAMA_LOG_FILE` at a directory they manage.
- **Tail it live**: `tail -f ~/.local/state/yaama/yaama.log` (or
  `tail -f "$YAAMA_LOG_FILE"` when you've set it).

## v1 Scope Freeze

v1 is frozen to reliable operator workflows already captured in `.plans/work/`.
Post-v1 candidates:

- auto-register unknown tmux sessions from `yaama status`
- richer activity timeline / event history
- improved native git-worktree lifecycle ergonomics
