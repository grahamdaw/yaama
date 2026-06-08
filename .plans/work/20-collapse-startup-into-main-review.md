# 20 - Collapse Startup Into Main — Review

## Summary
Work item 20 landed as a pure shape change with two commits: (1) fold
`internal/config` into `internal/profile` as a new `RuntimeConfig` /
`LoadConfig` / `ConfigOptions` triple; (2) inline `startup.Bootstrap`
into `cmd/yaama/main.go` and delete the `internal/startup` package.

## Plan vs Actual

### Followed plan
- `internal/config/config.go` moved into `internal/profile/` as
  `runtime_config.go`. Types renamed to `RuntimeConfig`,
  `ConfigOptions`, and `LoadConfig` so they do not clash with the
  pre-existing `profile.Config` (tmux session profile) type. The
  `LoadOptions` override semantics (`DBPathOverride` first, then
  `YAAMA_DB`, then default `./yaama.db`) are unchanged.
- `internal/startup` deleted along with `startup_test.go`.
- `tui.NewModel` now takes a `tui.Params` struct exposing only the
  fields the model reads (`Queries`, `Notices`, `TmuxAvailable`,
  `Logger`, `LogPath`). `startup.State`'s broader surface is gone.
- The board bootstrap sequence now lives in `cmd/yaama/main.go`'s
  `runBoard` and `bootstrapBoard` helpers — `profile.LoadConfig` →
  `logging.New` → `db.Init` → `tmux.IsAvailable`, in that order, with
  the same notices that previously surfaced (`Initialized DB at …`,
  `log file unavailable: …`, `tmux unavailable in PATH …`).
- The `hook` and `status` subcommands dropped their `startup` import
  and now call `profile.LoadConfig` + `db.Init` directly. They do not
  need logging or tmux probing on their hot paths, so the indirection
  was removable without behavior loss.
- README "Repository Layout" and AGENTS layout/logging sections
  rewritten to point at the new packages.

### Deviations from plan
- `bootstrapBoard` was extracted from `runBoard` (returning `(Params,
  cleanup, error)`) so the smoke test can exercise the bootstrap
  sequence without spinning up the Bubble Tea program. The plan
  described it as a single linear `main` body; the helper keeps the
  body linear but allows the test to run in-process without a TTY.
- The plan's illustrative `main` shape referenced post-#21 names
  (`store.Open`, `cfg.StatePath`, `StatePathOverride`). Per the
  plan's own non-goals ("Renaming flags / env vars / paths"), this
  refactor kept `DBPath` / `DBPathOverride` and the existing
  `db.Init` API. The shape change is on the bootstrap path only.
- `noopCloser` was replaced with a `func()` cleanup closure (`closeLog`
  / cleanup combined into one closer returned by `bootstrapBoard`)
  rather than `io.NopCloser`, since the log close path needed to be a
  no-op zero-value-friendly callback rather than an `io.Closer`.

## Done Criteria
- `internal/startup` and `internal/config` directories no longer exist.
- `cmd/yaama/main.go` contains the full bootstrap sequence inline
  (split between `runBoard` and `bootstrapBoard`, ~85 lines combined).
- All previously surfaced notices (DB created, log unavailable, tmux
  unavailable) still flow into `tui.Params.Notices` and render on
  first paint.
- `go test ./...` is green, including the new
  `cmd/yaama.TestBootstrapBoardInitializesFreshDBAndEmitsNotice`.

## Tests Run
- `go build ./...`
- `go test ./...`
- `make test`

## Files Touched
- New: `internal/profile/runtime_config.go`, `cmd/yaama/main_test.go`,
  `.plans/work/20-collapse-startup-into-main-review.md`.
- Removed: `internal/config/config.go`, `internal/startup/startup.go`,
  `internal/startup/startup_test.go`.
- Modified: `cmd/yaama/main.go`, `cmd/yaama/hook.go`,
  `cmd/yaama/status.go`, `internal/tui/model.go`, `README.md`,
  `AGENTS.md`, `.plans/INDEX.md`.
