# 19 - Action Logger - Plan vs Actual

## Outcome

Shipped per plan. Operators now get an `slog` text-handler log file at
the XDG state path covering startup, tmux bootstrap, recovery, refresh
failures, and profile load. The TUI retains stdout/stderr ownership;
`L` toasts the resolved path and the help overlay shows it durably.

## Deltas vs plan

- **Profile logger injection**: chose the constructor-arg path
  (`profile.LoadWithLogger`) rather than context-carried logger. The
  plan listed context as the tentative default but flagged the choice
  as open. The constructor variant keeps `Load(name)` callable from
  cleanup paths that already pass nothing else, and avoids threading a
  context just to carry diagnostics.
- **Startup events**: emit `startup.begin`, `startup.db_open`,
  `startup.tmux_detect`, `startup.ready` (plan listed `startup.begin`
  + `startup.ready` + DB/tmux availability — net same coverage with
  explicit event names).
- **Truncation helper**: lives in `internal/logging.Truncate` so action
  paths share one definition; plan implied inline truncation.
- **Process attrs**: `pid` is added at handler level. `version` is
  available as an option but not yet wired (no constant exists in the
  binary today); add when a release tag is introduced.
- **`L` shortcut**: emits a warning toast when the log path is empty
  (e.g. logger fell back to discard). Plan only described the happy
  path.

## Tests added

- `internal/logging/logging_test.go`: level parsing, path resolution,
  rotation (oversize -> `.1`, small -> kept), unknown-level warn line.
- `internal/tui/update_test.go`: `L` key toasts the path; warns when
  unavailable.

## Follow-ups (not in scope)

- Add a `--print-log-path` flag once a non-TUI consumer needs it (the
  README mentions a snippet that would benefit, currently using the
  documented default path).
- Wire `version` once the binary carries a build-time version constant.
- Cleanup-path stage logging is currently covered via per-step tmux
  events; a dedicated cleanup stage logger could come with work item
  18 (tmux bootstrap system tests).
