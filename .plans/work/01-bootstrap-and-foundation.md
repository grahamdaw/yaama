# 01 - Bootstrap and Foundation

## Goal
Establish the project skeleton, pinned tooling, and a runnable baseline TUI so implementation can proceed consistently across machines.

## Scope
- Go toolchain pinning and dependency setup
- Repository structure and developer workflow targets
- Initial static board render and startup path

## Phase 1: Toolchain and Project Scaffolding
### Steps
1. Create `go.mod` with pinned Go/toolchain versions and initialize module metadata.
2. Add `.mise.toml` to align local tool versions for contributors.
3. Add `tools/tools.go` and Makefile targets for `goose`, `sqlc`, and `golangci-lint`.
4. Scaffold folder layout under `cmd/` and `internal/` per spec conventions.
5. Add baseline `README.md` with build/run/test commands.

## Phase 2: Baseline TUI App Shell
### Steps
1. Wire `cmd/board/main.go` to launch a Bubble Tea program.
2. Implement a minimal model/update/view cycle with hardcoded columns and cards.
3. Add top bar and footer placeholders for summary stats and key hints.
4. Verify terminal resizing works and no panic occurs in narrow widths.

## Phase 3: Startup and Empty-State Skeleton
### Steps
1. Define startup flow entry points for config load, DB init, and first render.
2. Add empty-state UI copy with first-run hints (`n`, `board status ...`).
3. Add startup messaging surface for one-time DB initialization notices.
4. Add smoke checks in CI (`go test`, `go vet`) and ensure project builds on macOS/Linux.

## Definition of Done
- `make build` produces a working binary.
- Running `board` opens a stable TUI shell with placeholder columns.
- Tooling and repo scaffolding are committed and reproducible on a fresh machine.
