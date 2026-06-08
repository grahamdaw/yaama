# 20 - Collapse Startup, Config, Logging Bootstrap Into `cmd/yaama`

## Goal
`internal/startup` exists to call `config.Load`, `logging.New`,
`db.Init`, and `tmux.IsAvailable` in order and return a struct. It is
used in exactly one place (`cmd/yaama/main.go`) and ships its own
`noopCloser`, `State`/`Options` types, and a notice slice. Inline it.
While we're there, merge `internal/config` into `internal/profile` —
both packages read TOML out of `~/.config/yaama` and the seam between
them is artificial. Net: one fewer package, ~80 lines deleted, the
bootstrap path becomes linear and readable.

## Branch Name
`refactor/20-collapse-startup-into-main`

## Scope
- Inline `startup.Bootstrap` into `cmd/yaama/main.go` as a sequence of
  calls. Replace the `State` struct with locals; pass them into
  `tui.New(...)` directly.
- Replace `noopCloser` with `io.NopCloser(nil)` or a `func() error`
  cleanup pattern. The TUI never closed the logger via the interface
  anyway; `defer logFile.Close()` in `main` is enough.
- Move `internal/config` content into `internal/profile` (config root
  resolution is already a profile concern). Keep the `LoadOptions`
  override for the storage path so tests can still redirect it.
- Delete the `internal/startup` package and its test file. The notices
  list becomes a `[]string` local in `main` that gets passed to the
  TUI model verbatim.

## Non-goals
- Changing what gets initialized or in what order. This is a pure
  shape change.
- Touching `internal/logging` internals. Only the call site moves.
- Re-homing `tmux.IsAvailable`. Stays where it is.
- Renaming flags / env vars / paths.

## Design

### Target `main.go` shape (~50 lines)
```go
func main() {
    opts := parseFlags()

    cfg, err := profile.LoadConfig(profile.ConfigOptions{
        StatePathOverride: opts.StatePath,
    })
    must(err)

    logResult, logErr := logging.New(logging.Options{
        LevelEnv: os.Getenv("YAAMA_LOG_LEVEL"),
        PID:      os.Getpid(),
    })
    logger := logResult.Logger
    if logErr != nil {
        logger = logging.Discard()
    }
    defer logResult.Closer.Close()

    logger.Info("startup.begin", "state", cfg.StatePath)

    st, err := store.Open(cfg.StatePath) // post #20
    if err != nil {
        logger.Error("startup.store_open_failed", "err", err)
        os.Exit(1)
    }
    defer st.Close()

    notices := []string{}
    if st.Created() {
        notices = append(notices, "Initialized state at "+cfg.StatePath)
    }
    if logErr != nil {
        notices = append(notices,
            fmt.Sprintf("log file unavailable: %v", logErr))
    }
    tmuxOk := tmux.IsAvailable()
    if !tmuxOk {
        notices = append(notices,
            "tmux unavailable in PATH; attach actions are disabled.")
    }

    model := tui.New(tui.Params{
        Config: cfg, Store: st, Logger: logger,
        LogPath: logResult.Path, Notices: notices,
        TmuxAvailable: tmuxOk,
    })
    if _, err := tea.NewProgram(model).Run(); err != nil {
        logger.Error("tui.run_failed", "err", err)
        os.Exit(1)
    }
}
```

### Package moves
- `internal/config/config.go` → `internal/profile/config.go` (rename
  types: `config.Config` → `profile.RuntimeConfig`, `config.LoadOptions`
  → `profile.ConfigOptions`). Keep test coverage; rename the test
  file along with it.
- Update imports in `cmd/yaama/**`, `internal/tui/**`, and anywhere
  else `config` was referenced.

### `tui.Params`
- Today `tui.New` takes a `startup.State`. After this change it takes a
  smaller `Params` struct defined in `internal/tui`. The fields are
  the same set the TUI actually uses — anything in `State` that the
  TUI doesn't reach for gets dropped from the call site rather than
  carried along.

### Notices
- The notice slice is constructed in `main` and passed in. The TUI's
  notice-rendering code does not change.

## Phases

### Phase 1: Move config into profile
1. Move `internal/config/config.go` and `_test.go` into
   `internal/profile/`. Adjust the package clause and type names.
2. Update all imports. Run `go test ./...`; expect green.
3. Delete the empty `internal/config` directory.

### Phase 2: Inline startup
1. Define `tui.Params` matching what `tui.New` currently reads from
   `startup.State`.
2. Rewrite `cmd/yaama/main.go` per the shape above.
3. Delete `internal/startup` and `internal/startup/startup_test.go`.
   Any assertions there that we still want (e.g., "notice on fresh
   DB") move into a `cmd/yaama/main_test.go` smoke test that runs
   `main` in-process with a temp `$HOME`.

### Phase 3: Cleanup
1. `grep -r 'internal/startup\|internal/config'` → expect zero hits.
2. Update `README.md` "Repository Layout" to remove `internal/startup`
   and `internal/config`.
3. Update `AGENTS.md` if it references the bootstrap flow.

## Done Criteria
- `internal/startup` and `internal/config` directories do not exist.
- `cmd/yaama/main.go` is under ~80 lines and contains the full
  bootstrap sequence inline.
- All notices that previously surfaced (DB created, log unavailable,
  tmux unavailable) still surface on first render — verified by the
  smoke test.
- `go test ./...` green.
- No behavior changes observable to an operator running `yaama`
  against an existing or fresh `$HOME`.

## Dependencies / Sequencing
- Independent of #21, but lower-cost if landed **before** #21 — you'd
  otherwise touch `main.go`'s bootstrap path twice. Suggested order:
  #20 → #21.
