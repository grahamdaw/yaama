# 19 - Action Logger

## Goal
Give operators a real log file for debugging yaama-triggered actions
(bootstrap, recovery, cleanup, profile load, DB retries) without
breaking the TUI's stdout/stderr ownership. The existing
`last_error` field and toasts stay — the log is the underlying detail
they point to.

## Branch Name
`feat/19-action-logger`

## Scope
- `internal/logging` package wrapping stdlib `log/slog` with a
  `TextHandler`, level driven by `YAAMA_LOG_LEVEL`.
- One rolling log file at the XDG state path
  (`$XDG_STATE_HOME/yaama/yaama.log`, fallback
  `~/.local/state/yaama/yaama.log`); naive `>5 MiB → rename to .1`
  rotation at startup, no goroutines, no extra dependencies.
- Logger plumbed through `startup.State` and into the action-path
  packages (`tmux`, `profile`, `tui` recovery/cleanup/refresh).
- Operator surface: README documents the path and `YAAMA_LOG_LEVEL`;
  TUI help screen prints the resolved log path; a key (default `L`)
  pushes a toast with the path for quick copy.
- Tests for level/path resolution and rotation; logger injection point
  in tmux/tui tests uses a discard logger so existing assertions stay.

## Non-goals
- Shipping logs anywhere (no JSON, no stdout duplication, no syslog).
- Per-action structured event schemas. Free-form `slog` attributes are
  enough; we are not building an analytics pipeline.
- Logging every keypress / refresh tick. Action paths only.
- Multi-file logs per session or per agent. One file per yaama process
  is enough.
- Secret redaction beyond truncation. We won't log script contents, but
  we won't add a regex-based redactor either.

## Design

### Format and level
- `slog.NewTextHandler` with `time=…  level=…  msg=…  key=value` lines.
- Default level: `INFO`. `YAAMA_LOG_LEVEL` accepts `debug|info|warn|error`
  (case-insensitive); unknown values fall back to `INFO` and emit one
  `warn` line about it.

### Path resolution
- Order:
  1. `$YAAMA_LOG_FILE` (absolute path; honored verbatim).
  2. `$XDG_STATE_HOME/yaama/yaama.log` when set.
  3. `$HOME/.local/state/yaama/yaama.log` otherwise.
- `MkdirAll(dirname, 0o755)` before opening. Open with
  `O_APPEND|O_CREATE|O_WRONLY`, mode `0o644`.

### Rotation
- Before opening, `os.Stat` the target. If size > 5 MiB, rename to
  `<path>.1` (overwriting any existing `.1`). One backup, no compression.
- Document in README that operators wanting more history should
  configure `YAAMA_LOG_FILE` to point at a directory they manage.

### What gets logged (info unless noted)
- `internal/startup`: `startup.begin`, `startup.ready`, db open, tmux
  availability detection. `error` for DB open failures.
- `internal/tmux/bootstrap.go`: one line per step
  (`before_start`, `new-session`, `set-environment`, `apply-windows`,
  `source-layout`, `after_start`, `agent-command`, `select-window`)
  with `session`, `working_dir`. On step failure, log `error` with the
  trimmed stderr captured by `RunShellHook` / `runTmux`.
- `internal/tmux/cleanup` paths: stage transitions + partial failures
  (`error`).
- `internal/tui/update.go`:
  - `recreateSelectedSession`: outcome line with
    `profile=<name>`, `fallback=minimal|missing|parse-error|none`.
  - Refresh-tick DB errors (currently only banner-surfaced) at `warn`.
- `internal/profile/profile.go`: `profile.load` (debug on success,
  warn on missing-file, error on parse failure).

### TUI surface
- `model.logPath string` populated from `startup.State`.
- Help screen renders `Log: <path>` near the keybinding table.
- Add `L` key (normal mode) → `pushNotice("Log: " + m.logPath)`.

### Truncation
- Truncate `stderr`/`last_error` style strings to 512 chars before
  logging. Add `…` suffix when truncated. No regex scrubbing.

## Phases

### Phase 1: Logging package + startup wiring
1. Create `internal/logging/logging.go` with `New`, `DefaultPath`,
   `LevelFromEnv`, and a `rotateIfLarge` helper. No tests yet beyond
   compile.
2. Add `Logger *slog.Logger` and `LogClose io.Closer` to
   `startup.State`; open in `startup.Init`.
3. `cmd/board/main.go`: `defer state.LogClose.Close()`; emit
   `startup.ready` after `tea.NewProgram` builds.
4. Tests in `internal/logging/logging_test.go`:
   - level fallback (unknown env value → INFO + warn line),
   - path resolution honoring `YAAMA_LOG_FILE` vs `XDG_STATE_HOME` vs
     home fallback,
   - rotation: pre-seed a >5 MiB file, assert it moves to `.1`.

### Phase 2: tmux + profile instrumentation
1. Add an optional `*slog.Logger` to `tmux.BootstrapSpec` (no logger →
   `slog.New(slog.NewTextHandler(io.Discard, nil))` default).
2. Wire log lines per step in `BootstrapSession` and the cleanup paths.
   Truncate stderr captured by `RunShellHook` / `runTmux` to 512 chars.
3. `internal/profile.Load`: accept a logger parameter (default discard)
   or read one via context — pick the path that requires the fewest
   call-site changes; document the choice in the PR.
4. Update existing tests only enough to compile; do not assert log
   contents.

### Phase 3: TUI instrumentation + operator surface
1. Pass the logger into the `model` and into `recreateSelectedSession`,
   `runCleanupScripts`, `refreshData` failure branches.
2. Add `L` key handler in normal mode: toast the log path.
3. Update the help screen renderer to display `Log: <path>`.
4. README: add a "Logs" subsection under Troubleshooting with the
   default path, `YAAMA_LOG_LEVEL` examples, and a `tail -f` snippet.

### Phase 4: Polish + tests
1. Confirm `go test ./...` still passes; add one TUI test that drives
   `L` and asserts the toast contains the configured path.
2. AGENTS.md: note that any new action-path code should add an
   `slog` line at info on the happy path and error on failure.
3. Add a `.plans/work/19-action-logger-review.md` and tick INDEX.

## Definition of Done
- Default `./bin/board` run creates the XDG log file and writes
  startup + bootstrap step lines.
- Triggering a recovery with a missing/invalid profile produces a log
  line at `warn` or `error` containing enough detail to diagnose
  without re-running.
- `YAAMA_LOG_LEVEL=debug` increases verbosity without changing the
  set of files written.
- README documents the path, level env var, and `L` shortcut.
- Existing tests stay green; new tests cover path/level/rotation.

## Open Questions
1. **Logger injection style for profile package.** Constructor arg
   (`profile.LoadWithLogger`) vs. context-carried logger. Default:
   context, since `profile.Load` is called from multiple sites and we
   don't want to fork the API.
2. **Help/`L` toast vs. banner.** Toast is transient and easy to miss.
   Default: toast (cheap) plus help-screen line (durable). Promote to
   banner only if the file fails to open at startup.
3. **Process identity in log lines.** Add `pid` and yaama version as
   handler-level attrs? Default: yes — cheap, makes multi-run logs
   self-explanatory.
