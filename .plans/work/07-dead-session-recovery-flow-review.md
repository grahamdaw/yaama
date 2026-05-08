# 07 Dead Session Recovery Flow - Review

## Planned Scope
- dead-session action menu.
- tmux recreation with working directory reuse.
- failure-safe recovery messaging.

## Actual Implementation
- Added explicit dead-session recovery action on `r` in Normal mode:
  - only triggers on selected dead agents,
  - keeps `Enter` attach behavior unchanged for live sessions.
- Expanded dead-session guidance in the UI:
  - footer now advertises `r recover dead`,
  - help overlay documents dead-session recovery flow,
  - detail panel adds actionable recovery hints for dead cards, including
    missing `working_dir` guidance.
- Implemented recreate execution path:
  - validates `working_dir` is present, accessible, and a directory,
  - creates detached tmux session in the existing directory using
    `tmux new-session -d -s <session> -c <working_dir>`,
  - transitions recovered agent state to `running`, refreshes heartbeat, clears
    `last_error`, and ensures cleanup state remains active,
  - immediately attaches/switches operator into recovered session.
- Implemented failure-safe recovery behavior:
  - missing/invalid `working_dir` stops with actionable warning and no state
    mutation,
  - tmux recreate command/build failures preserve status and persist `last_error`,
  - failure toast points operator to `e` for direct mapping edits.
- Added/updated regression coverage in `internal/tui/update_test.go`:
  - missing `working_dir`,
  - invalid `working_dir` (path is not a directory),
  - successful recreate + attach + metadata update.

## Plan vs Actual Notes
- The dead-session options are surfaced as contextual key hints and detail-panel
  recovery copy rather than a modal action menu; behavior remains keyboard-first
  and meets the planned recover/edit/archive decision flow.

## Validation Evidence
- `go test ./internal/tui` passed.
- `make test` passed.
