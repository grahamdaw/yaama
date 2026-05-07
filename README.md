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
