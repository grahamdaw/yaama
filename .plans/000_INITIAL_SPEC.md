# Agent Board — Technical Spec

A terminal kanban board for managing AI coding agents running in tmux sessions. Single Go binary, SQLite-backed, runs anywhere.

---

## 1. Goals & Non-Goals

### Goals

- Visualise the state of all running agents at a glance (kanban columns by status).
- Jump directly into an agent's tmux session from a selected card.
- Persist agent metadata locally (SQLite file in the repo).
- Allow agents themselves to update their own status via a small CLI helper.
- Single static binary, cross-platform (macOS + Linux), no runtime dependencies.

### Non-Goals (v1)

- Multi-user / remote sync.
- Web UI or HTTP API.
- Spawning agents from within the board (assume tmux sessions are created externally for now; revisit in v2).
- Authentication or access control.

---

## 2. Tech Stack


| Concern          | Choice                                                                   | Why                                                               |
| ---------------- | ------------------------------------------------------------------------ | ----------------------------------------------------------------- |
| Language         | Go 1.23+                                                                 | Single binary, fast, easy to learn coming from TS                 |
| TUI framework    | [Bubble Tea](https://github.com/charmbracelet/bubbletea)                 | Mature, Elm architecture, great kanban examples                   |
| Styling          | [Lip Gloss](https://github.com/charmbracelet/lipgloss)                   | Pairs natively with Bubble Tea                                    |
| Components       | [Bubbles](https://github.com/charmbracelet/bubbles)                      | List, viewport, textinput primitives                              |
| Database         | SQLite via `[modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)` | Pure Go driver — no CGO, keeps cross-compilation trivial          |
| Query layer      | [sqlc](https://sqlc.dev/)                                                | Generates type-safe Go from SQL — feels like Kysely               |
| Migrations       | [goose](https://github.com/pressly/goose)                                | Versioned migrations as plain `.sql` files                        |
| tmux integration | Shell out to `tmux` binary                                               | No library needed; tmux's CLI is the API                          |
| CLI flags        | [cobra](https://github.com/spf13/cobra)                                  | Standard for Go CLIs; supports `board status`, `board list`, etc. |


---

## 3. Go Toolchain Management

This is the equivalent of how you'd use `nvm` / `volta` / `pnpm` in TS-land.

### 3.1 Pinning the Go version

Go has built-in version management since 1.21. Declare the required version in `go.mod`:

```go
module github.com/kaikucrew/agent-board

go 1.23
```

If a developer runs `go build` with an older toolchain, Go will automatically download the correct version and use it. This is the closest thing to `engines` in `package.json` and it actually works.

For reproducibility across machines, pin a specific patch version:

```go
go 1.23.4

toolchain go1.23.4
```

### 3.2 Installing Go itself

Three sensible options:

1. **Homebrew (macOS)** — `brew install go`. Quick but you'll have one global Go.
2. **[mise](https://mise.jdx.dev/)** — recommended for Kaiku. Already a polyglot version manager, handles Go, Node, Python in one tool. Add a `.mise.toml` to the repo:
  ```toml
   [tools]
   go = "1.23.4"
  ```
   Then `mise install` in the repo root pulls the right version.
3. **Official installer** — download from go.dev. Fine for one-off machines.

The repo should include `.mise.toml` so anyone cloning gets the right version automatically.

### 3.3 Dependencies

Go modules work like npm but simpler:

```bash
go mod init github.com/kaikucrew/agent-board   # like npm init
go get github.com/charmbracelet/bubbletea       # like npm install
go mod tidy                                      # like npm prune + install
```

`go.mod` and `go.sum` are committed (equivalent to `package.json` + lockfile). There is no `node_modules` — modules are cached globally in `~/go/pkg/mod/`.

### 3.4 Tooling: install via `tools.go`

The Go convention for dev tools (sqlc, goose, golangci-lint) is a `tools/tools.go` file that imports them, so `go mod tidy` keeps them pinned:

```go
//go:build tools
// +build tools

package tools

import (
    _ "github.com/pressly/goose/v3/cmd/goose"
    _ "github.com/sqlc-dev/sqlc/cmd/sqlc"
    _ "github.com/golangci/golangci-lint/cmd/golangci-lint"
)
```

Then a Makefile target installs them:

```makefile
tools:
	go install github.com/pressly/goose/v3/cmd/goose
	go install github.com/sqlc-dev/sqlc/cmd/sqlc
	go install github.com/golangci/golangci-lint/cmd/golangci-lint
```

---

## 4. Project Structure

```
agent-board/
├── .mise.toml                  # Go version pin
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── tools/tools.go              # Dev tool pins
├── cmd/
│   └── board/
│       └── main.go             # Entry point — `board` command
├── internal/
│   ├── tui/
│   │   ├── model.go            # Bubble Tea model
│   │   ├── update.go           # Message handling
│   │   ├── view.go             # Lip Gloss rendering
│   │   ├── columns.go          # Column-specific logic
│   │   └── keys.go             # Keymap definitions
│   ├── db/
│   │   ├── schema/             # goose migrations (.sql files)
│   │   │   └── 001_init.sql
│   │   ├── queries/            # sqlc input (.sql files)
│   │   │   └── agents.sql
│   │   ├── generated/          # sqlc output (gitignored or committed)
│   │   └── db.go               # Connection setup
│   ├── tmux/
│   │   └── tmux.go             # exec wrappers around tmux binary
│   └── agent/
│       └── agent.go            # Domain model
├── sqlc.yaml                   # sqlc config
└── agent-board.db              # Local DB (gitignored)
```

`cmd/` and `internal/` are Go conventions: `cmd/<name>/main.go` is each binary's entry point, `internal/` is private packages that can't be imported by other modules.

---

## 5. Data Model

### 5.1 Schema (initial migration)

```sql
-- internal/db/schema/001_init.sql

-- +goose Up
CREATE TABLE agents (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    tmux_session    TEXT NOT NULL UNIQUE,
    status          TEXT NOT NULL CHECK (status IN ('idle', 'running', 'blocked', 'review', 'done')),
    task            TEXT,
    worktree_path   TEXT,
    branch          TEXT,
    last_activity   TEXT,                            -- free-form: "editing main.go", "running tests"
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_agents_status ON agents(status);

-- +goose Down
DROP TABLE agents;
```

### 5.2 Status lifecycle

```
idle ──▶ running ──▶ blocked ──▶ running ──▶ review ──▶ done
                  └─▶ review                  └─▶ running (kicked back)
```

`idle` = session exists but agent isn't actively working.
`running` = agent is processing.
`blocked` = waiting on input/decision.
`review` = work complete, awaiting human review.
`done` = merged/closed.

### 5.3 Source of truth

- **DB** = what agents *should* exist and their declared status.
- **tmux** = what sessions are *actually* alive.
- On every refresh tick, reconcile: mark agents whose `tmux_session` doesn't appear in `tmux list-sessions` with a visual "dead" indicator (don't auto-delete — let the user prune).

---

## 6. TUI Specification

### 6.1 Layout

```
┌─ Agent Board ───────────────────────────── 7 agents · 3 running ──┐
│                                                                    │
│  ┌─ Idle (1) ──┐ ┌─ Running (3) ─┐ ┌─ Blocked (1) ┐ ┌─ Review (1)│
│  │             │ │ ▸ refactor-db │ │   add-tests   │ │ docs-pass  │
│  │  spike-ui   │ │   feat-auth   │ │               │ │            │
│  │             │ │   bug-1234    │ │               │ │            │
│  │             │ │               │ │               │ │            │
│  └─────────────┘ └───────────────┘ └───────────────┘ └────────────│
│                                                                    │
│  ▸ refactor-db                                                     │
│    Session: agent-refactor-db   Branch: feat/db-cleanup            │
│    Last:    editing internal/db/queries.sql                        │
│                                                                    │
├────────────────────────────────────────────────────────────────────┤
│ ↵ attach  n new  e edit  d delete  s status  r refresh  q quit     │
└────────────────────────────────────────────────────────────────────┘
```

- **Top bar**: title + summary stats.
- **Columns**: one per status, equal width, scrollable internally.
- **Detail panel**: shows full details of currently selected card.
- **Footer**: keybind help.

### 6.2 Keymap


| Key            | Action                                      |
| -------------- | ------------------------------------------- |
| `←` / `h`      | Move selection to column on the left        |
| `→` / `l`      | Move selection to column on the right       |
| `↑` / `k`      | Move selection up within column             |
| `↓` / `j`      | Move selection down within column           |
| `Enter`        | Attach to selected agent's tmux session     |
| `n`            | New agent (modal form)                      |
| `e`            | Edit selected agent (modal form)            |
| `d`            | Delete selected agent (with confirm)        |
| `s`            | Cycle status forward (idle → running → ...) |
| `S`            | Cycle status backward                       |
| `r`            | Force refresh from DB + tmux                |
| `/`            | Filter / search                             |
| `?`            | Toggle help overlay                         |
| `q` / `Ctrl+C` | Quit                                        |


### 6.3 Bubble Tea model shape

```go
type Model struct {
    db          *db.Queries
    agents      []agent.Agent          // last loaded snapshot
    liveSessions map[string]bool       // tmux sessions currently alive

    columns     []Column               // one per status
    focused     int                    // index of focused column

    width, height int

    mode        Mode                   // Normal | Form | Confirm | Help
    form        *FormState
    confirm     *ConfirmState

    err         error
}

type Mode int
const (
    ModeNormal Mode = iota
    ModeForm
    ModeConfirm
    ModeHelp
)
```

### 6.4 Messages

```go
type tickMsg time.Time              // periodic refresh
type agentsLoadedMsg []agent.Agent  // DB query result
type tmuxSessionsMsg map[string]bool
type attachDoneMsg struct{ err error }
type agentSavedMsg agent.Agent
type errMsg error
```

### 6.5 Update loop sketch

```go
func (m Model) Init() tea.Cmd {
    return tea.Batch(loadAgents(m.db), listTmuxSessions(), tick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tickMsg:
        return m, tea.Batch(loadAgents(m.db), listTmuxSessions(), tick())
    case agentsLoadedMsg:
        m.agents = msg
        m.rebuildColumns()
        return m, nil
    case tea.KeyMsg:
        return m.handleKey(msg)
    case attachDoneMsg:
        // Returned from tmux session; refresh state
        return m, loadAgents(m.db)
    }
    return m, nil
}
```

### 6.6 Refresh strategy

- Tick every **2 seconds** to reload agents + tmux sessions.
- On any user-initiated DB change, reload immediately rather than waiting for tick.
- Reconciliation between DB and tmux happens client-side after both queries return.

---

## 7. tmux Integration

### 7.1 Detecting tmux context

```go
func InsideTmux() bool {
    return os.Getenv("TMUX") != ""
}
```

### 7.2 Listing sessions

```go
func ListSessions(ctx context.Context) (map[string]bool, error) {
    out, err := exec.CommandContext(ctx, "tmux", "list-sessions", "-F", "#{session_name}").Output()
    if err != nil {
        // Exit code 1 with no sessions is normal — return empty map.
        var exitErr *exec.ExitError
        if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
            return map[string]bool{}, nil
        }
        return nil, err
    }
    sessions := map[string]bool{}
    for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
        if line != "" {
            sessions[line] = true
        }
    }
    return sessions, nil
}
```

### 7.3 Attaching

```go
func AttachCmd(session string) tea.Cmd {
    var cmd *exec.Cmd
    if InsideTmux() {
        cmd = exec.Command("tmux", "switch-client", "-t", session)
    } else {
        cmd = exec.Command("tmux", "attach-session", "-t", session)
    }
    return tea.ExecProcess(cmd, func(err error) tea.Msg {
        return attachDoneMsg{err: err}
    })
}
```

`tea.ExecProcess` is the critical primitive: it suspends the Bubble Tea renderer, hands the terminal over to tmux, and resumes when the user detaches.

### 7.4 v2 ideas (not in scope)

- Create new tmux session from the board (`tmux new-session -d -s <name>`).
- Send keys into a session (`tmux send-keys`) — could be used to deliver instructions to an agent.
- Capture pane content (`tmux capture-pane`) for live preview in the detail panel.

---

## 8. Status Update CLI

A subcommand of the same binary so agents can update their own status:

```bash
board status running --task "implementing CREW-301"
board status blocked --task "needs review on auth approach"
board status review
```

Resolution rule for "which agent": read `$TMUX_PANE` or `tmux display -p '#S'` to get the session name, then look up the agent by `tmux_session`. This means agents don't need to know their own DB ID.

```go
// cmd/board/status.go (cobra subcommand)
func runStatus(cmd *cobra.Command, args []string) error {
    session, err := tmux.CurrentSession()
    if err != nil {
        return fmt.Errorf("not inside tmux: %w", err)
    }
    return queries.UpdateStatusBySession(ctx, db.UpdateStatusBySessionParams{
        TmuxSession: session,
        Status:      args[0],
        Task:        taskFlag,
    })
}
```

---

## 9. Configuration

Single config file at `~/.config/agent-board/config.toml` with sensible defaults:

```toml
[database]
path = "~/.local/share/agent-board/agents.db"   # default; overridable per-repo

[ui]
refresh_interval_seconds = 2
column_order = ["idle", "running", "blocked", "review", "done"]

[tmux]
attach_mode = "auto"   # auto | switch | attach
```

For per-repo DBs (your "lives in the repo" requirement), allow `--db ./agents.db` flag or env var `AGENT_BOARD_DB`. Recommend keeping the DB out of git (`*.db` in `.gitignore`) but commit the schema migrations.

---

## 10. Build & Distribution

### 10.1 Makefile

```makefile
.PHONY: build run test lint migrate generate tools

build:
	go build -o bin/board ./cmd/board

run: build
	./bin/board

test:
	go test ./...

lint:
	golangci-lint run

generate:
	sqlc generate

migrate:
	goose -dir internal/db/schema sqlite3 ./agent-board.db up

tools:
	go install github.com/pressly/goose/v3/cmd/goose
	go install github.com/sqlc-dev/sqlc/cmd/sqlc
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

cross-compile:
	GOOS=darwin GOARCH=arm64 go build -o bin/board-darwin-arm64 ./cmd/board
	GOOS=darwin GOARCH=amd64 go build -o bin/board-darwin-amd64 ./cmd/board
	GOOS=linux  GOARCH=amd64 go build -o bin/board-linux-amd64  ./cmd/board
```

Because `modernc.org/sqlite` is pure Go, `cross-compile` works without setting up CGO toolchains. This is a big quality-of-life win.

### 10.2 CI

Minimal GitHub Actions workflow:

```yaml
name: ci
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go test ./...
      - run: go vet ./...
```

`go-version-file: go.mod` reuses the version pin — no duplication.

---

## 11. Implementation Plan

Suggested order (roughly half-day chunks):

1. **Scaffold** — `go mod init`, install Bubble Tea, render a static three-column layout with hardcoded cards. *Goal: see something on screen.*
2. **DB layer** — goose migration, sqlc setup, replace hardcoded data with DB queries. *Goal: cards come from SQLite.*
3. **Navigation** — implement keymap for moving selection between columns and within columns. *Goal: focused card is highlighted.*
4. **tmux attach** — wire up Enter to `tea.ExecProcess`. *Goal: pressing Enter jumps into a session.*
5. **CRUD** — modal form for new/edit, confirm dialog for delete. *Goal: full board management from inside the TUI.*
6. **Live refresh** — tick + tmux reconciliation, "dead session" indicator. *Goal: board reflects reality without manual refresh.*
7. `**board status` subcommand** — cobra subcommand for agents to update their own status. *Goal: agents can self-report.*
8. **Polish** — help overlay, search/filter, color theming, error toasts.

A working v0 (steps 1–4) should be achievable in a day if you lean on the Charm examples.

---

## 12. Open Questions

- **Should agents auto-register?** When a new tmux session matching a naming convention (e.g. `agent-`*) appears, should it create a DB row automatically, or only show up after `board status` is called from inside it?
- **Worktree integration?** Given heavy git-worktree usage in your workflow, should the board know about worktrees and offer "open worktree in editor" alongside "attach to session"?
- **MCP tool?** A future iteration could expose `board.list`, `board.update_status` etc. as MCP tools so KES agents can interact with it programmatically — fits naturally with the existing `mcp-engineering` work.
- **Logs / activity feed?** Worth adding an `agent_events` table for an audit trail, or out of scope for v1?

---

## 13. References

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — main framework
- [Bubble Tea examples](https://github.com/charmbracelet/bubbletea/tree/main/examples) — especially `list-default`, `list-fancy`, `split-editors`
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — styling
- `[modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)` — pure-Go SQLite
- [sqlc](https://docs.sqlc.dev/en/latest/) — type-safe SQL → Go
- [goose](https://github.com/pressly/goose) — migrations
- [Effective Go](https://go.dev/doc/effective_go) — read this once before writing real code
- [A Tour of Go](https://go.dev/tour/) — language fundamentals (~2 hours)

