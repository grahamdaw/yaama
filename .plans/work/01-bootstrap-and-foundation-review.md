# 01 Bootstrap and Foundation - Review

## Planned Scope
- Pin Go toolchain and scaffold module/tooling foundations.
- Add baseline Bubble Tea shell app wired from `cmd/board/main.go`.
- Add startup entry points for config load, DB init, and first render state.
- Add empty-state hints and startup notice surface.
- Add CI smoke checks (`go test`, `go vet`) and baseline developer commands.

## Actual Implementation
- Added toolchain + tooling foundations: `go.mod`, `.mise.toml`, `Makefile`, `tools/tools.go`, and `sqlc.yaml`.
- Scaffolded repository layout under `cmd/` and `internal/`:
  - `cmd/board/main.go`
  - `internal/{tui,config,startup,db,agent,tmux}`
- Implemented startup bootstrap path:
  - config load (`internal/config`)
  - DB initialization entrypoint (`internal/db`)
  - startup notices passed into first render (`internal/startup`)
- Implemented baseline Bubble Tea shell:
  - minimal model/update/view cycle
  - placeholder status columns/cards
  - top bar and footer hints
  - resize-safe rendering behavior
  - empty-state hints for first run (`n`, `board status ...`)
- Added initial DB scaffolding placeholders:
  - `internal/db/schema/001_init.sql`
  - `internal/db/queries/agents.sql`
  - `internal/db/generated/.gitkeep`
- Added CI smoke checks in `.github/workflows/ci.yml` (`go test`, `go vet`).
- Updated `README.md` and `AGENTS.md` to reflect bootstrap commands and structure.
- Updated `Makefile` tooling targets to use `$(go env GOPATH)/bin` so `make lint`, `make generate`, and `make migrate` do not depend on shell PATH.

## Plan vs Actual Notes
- No functional scope deviations from the work item.
- Additional robustness improvement: Make targets now resolve tool binaries directly from Go tool install location to reduce machine-specific PATH issues.

## Validation Evidence
- `go mod tidy` completed successfully.
- `make build` passed.
- `make test` passed.
- `make vet` passed.
- `make lint` passed.
