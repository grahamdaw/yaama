# 16 - Rename `board` Binary to `yaama` — Review

## Plan vs Actual

- Renamed `cmd/board/` to `cmd/yaama/` (all files preserved; `package main`
  unchanged); `go.mod` required no edits since no code path imported
  `cmd/board` by package path.
- Updated `Makefile` `build`, `run`, and `release-check` targets to produce
  `bin/yaama` and `bin/release-check/yaama-{darwin-arm64,linux-amd64}`. No
  `bin/board` artifact is emitted now.
- Updated user-facing strings inside the CLI: `status.go` usage line,
  `hook.go` usage line, and `main.go` exit-error prefix all now say `yaama`.
- Updated the TUI empty-state CLI hint constant (`internal/tui/keys.go`)
  and the matching assertion in `internal/tui/view_test.go` so the
  rendered hint reads `yaama status running --task "..."`.
- Updated `README.md` (build instructions, repo layout, operator runbook,
  CLI examples, Claude Code hook JSON snippet, post-v1 candidates) and
  `AGENTS.md` (repository structure line) to reference `yaama`.
- Updated forward-looking plan files: `.plans/work/01-bootstrap-and-foundation.md`,
  `.plans/work/10-cli-status-parity.md`, and
  `.plans/work/19-action-logger.md`.

## Deviations / Notes

- Per skill guidance, historical narrative in `*-review.md` files and the
  upstream `.plans/000_INITIAL_SPEC.md` / `.plans/001_SPEC_ITERATION_UX.md`
  were left untouched — these capture the project's original wording and
  are not forward-looking instructions.
- `examples/WALKTHROUGH.md`'s "Start board" line and `CONTRIBUTING.md`'s
  example branch / commit names (`feat/board-shell`, `feat: add board
  shell startup model`) refer to the TUI as a concept and to illustrative
  branch naming rather than the binary, so they were intentionally not
  rewritten.
- No backwards-compatibility shim or `board` symlink was added (explicit
  out-of-scope item).

## Verification

- `make build` → produced `bin/yaama` only.
- `make test` → all packages pass (including the renamed `cmd/yaama`
  package tests and `internal/tui` empty-state assertion).
- `make vet` → clean.
- `make release-check` → produced `bin/release-check/yaama-darwin-arm64`
  and `bin/release-check/yaama-linux-amd64`.
