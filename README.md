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

## Profiles (End-User Setup)

`yaama` reads profiles from `~/.config/yaama/profiles/*.toml`.

To get started quickly, copy the examples from this repository:

```bash
mkdir -p ~/.config/yaama/profiles
mkdir -p ~/.config/yaama/tmux
cp examples/profiles/default.toml ~/.config/yaama/profiles/default.toml
cp examples/profiles/dev.toml ~/.config/yaama/profiles/dev.toml
cp examples/tmux/dev-layout.tmux ~/.config/yaama/tmux/dev-layout.tmux
```

Then edit at least `repo.path` in each file so it points to your local git repository path.

Example files in this repo:

- `examples/profiles/default.toml`: minimal single-window profile
- `examples/profiles/dev.toml`: richer profile with scripts and multi-pane tmux layout
- `examples/tmux/dev-layout.tmux`: sample layout file referenced by `dev.toml`

After creating profiles, start the board and press `n` to create an item from a selected profile.
Profile-backed create now requires:
- a repository path that resolves to a git repository, and
- an explicit branch name (`profile -> task -> branch` wizard).
`yaama` manages native `git worktree` lifecycle directly; no external worktree manager is required.

## Developer Commands

```bash
make build    # build bin/board
make run      # run the board TUI
make test     # run go test ./...
make vet      # run go vet ./...
make lint     # run golangci-lint
make tools    # install goose/sqlc/golangci-lint
make generate # run sqlc generate
make migrate  # run local sqlite migrations
make release-check # cross-build checks for macOS/Linux artifacts
```

## Repository Layout

- `cmd/board/`: CLI entrypoint
- `internal/tui/`: Bubble Tea model, update loop, and rendering
- `internal/config/`: runtime config loading
- `internal/startup/`: startup bootstrap flow (config -> DB init -> first render state)
- `internal/db/`: DB bootstrap, migration files, and SQL queries
- `.plans/`: product specs and phased work items

## Work Tracking

Implementation order and completion state are tracked in `.plans/INDEX.md`.
Work-item scope and done criteria live in `.plans/work/`.

## Operator Runbook

- Start the board with `make run` (or `./bin/board` after `make build`).
- Keyboard-only core flow:
  - `n` create agent (profile -> task -> branch wizard)
  - `e` edit selected agent
  - `/` filter by name/task/branch/session
  - `s` open status picker (`1..5` then `Enter`, or `S` reverse cycle)
  - `Enter` attach to selected live tmux session
  - `r` recover selected dead session in existing `working_dir`
  - `d` archive cleanup, `D` hard prune cleanup
- From inside an agent tmux session, update without opening TUI:
  - `board status running --task "..." --activity "..."`

## Troubleshooting

- **`tmux unavailable in PATH`**: install tmux or update `PATH`; attach/recover actions are disabled until available.
- **`No agent found for current tmux session`**: create/edit a board item so `tmux_session` matches your current session.
- **Dead session shown as `[DEAD]`**: select item and press `r`; if working dir is invalid, press `e` to fix mapping first.
- **DB lock/unavailable banners**: keep board open; it retries on refresh ticks. Validate DB path/permissions if it persists.

## v1 Scope Freeze

v1 is frozen to reliable operator workflows already captured in `.plans/work/`.
Post-v1 candidates:

- auto-register unknown tmux sessions from `board status`
- richer activity timeline / event history
- improved native git-worktree lifecycle ergonomics
