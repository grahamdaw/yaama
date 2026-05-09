# 10 CLI Status Parity - Review

## Planned Scope
- status command resolution by active tmux session.
- task/activity/branch update flags.
- clear failure behavior outside tmux or without matching record.

## Actual Implementation
- Added a dedicated `board status` subcommand in `cmd/board/main.go` that routes
  status updates without starting the Bubble Tea TUI.
- Implemented `board status <status>` contract in `cmd/board/status.go` with
  strict accepted-status validation (`idle`, `running`, `blocked`, `review`,
  `done`) and non-zero failure behavior for invalid input.
- Added optional metadata flags `--task`, `--activity`, and `--branch` and
  applied them as partial updates so omitted fields remain unchanged.
- Wired session resolution to tmux via `tmux.CurrentSession` (`$TMUX` context
  plus `display-message -p '#S'`) and used that session key to resolve and
  update the matching agent row.
- Reused DB status update query semantics so successful CLI writes also update
  heartbeat and `updated_at` timestamps for board visibility parity.
- Added explicit, actionable failures for:
  - running outside tmux,
  - tmux unavailable in PATH,
  - no matching agent row for current tmux session (with create/register
    recommendation).
- Added regression tests in `cmd/board/status_test.go` covering:
  - invalid status validation errors with accepted-value hints,
  - outside-tmux failure,
  - missing-record failure,
  - successful in-session status + metadata writes,
  - preservation of existing metadata when optional flags are omitted.

## Plan vs Actual Notes
- The implementation introduces a lightweight subcommand dispatch in `main`
  before TUI startup, which keeps existing board behavior intact while enabling
  CLI self-reporting from tmux sessions.
- "Board refresh compatibility" is achieved via existing DB update semantics:
  status writes update `updated_at`/heartbeat, which the board already consumes
  on refresh ticks.

## Validation Evidence
- `go test ./cmd/board` passed.
- `make test` passed.
