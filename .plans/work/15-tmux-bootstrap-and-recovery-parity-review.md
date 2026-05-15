# 15 - Tmux Bootstrap And Recovery Parity (Review)

## Summary
Work item 15 completed. Dead-session recovery now goes through the same
`tmux.BootstrapSession` pipeline as new-session create, with the agent
command suppressed. Session-scoped env vars are injected on both flows.

## Plan vs Actual

- **Shared spec builder.** `toBootstrapSpec` moved to
  `internal/tui/bootstrap_spec.go`; both `form.persistCreateForm` and
  `update.recreateSelectedSession` use it. A `minimalBootstrapSpec` helper
  was added for the missing-profile fallback. Matches plan.
- **Env injection.** `tmux.BootstrapSession` issues
  `set-environment -t <session> YAAMA_TMUX_SESSION/YAAMA_WORKING_DIR`
  immediately after `new-session`, before windows/panes. Pane 0 also gets
  an explicit `export …` send-keys so the pre-existing shell sees the
  values (covers Open Question 2 as planned).
- **Recovery wiring.** `recreateSelectedSession` calls
  `m.bootstrapSession` with `AgentCommand=nil` and `AgentWindow` set to the
  session name. Profile-missing falls back to minimal spec + warning
  toast; parse/other load error records `last_error` and aborts.
- **Open Question 1.** Resolved by reuse — `after_start` continues to host
  the tmux setup script. No new TOML field was added. Documented in
  `examples/profiles/README.md` (after_start runs on both create and
  recovery; must be idempotent).
- **Tests.** New unit coverage in
  `internal/tmux/bootstrap_test.go` (env injection ordering,
  `AgentCommand=nil` no-op) and `internal/tui/update_test.go`
  (profile-applied recovery, missing-profile fallback warning,
  parse-error abort, agent command suppression).

## Deviations / Notes

- No dedicated `tmux_setup` profile field introduced — reuse of
  `after_start` was the documented preference. If operators later need a
  separate "runs only on recovery vs. only on create" semantics this is
  the place to add it.
- `createDetachedCmd` is still present on the model and used by the
  existing test for missing/invalid working_dir paths; it is no longer
  called by `recreateSelectedSession`. Left intact to minimize blast
  radius — can be removed in a follow-up sweep if no other caller appears.
- Profile-missing detection relies on `errors.Is(err, os.ErrNotExist)`,
  which works because `profile.Load` wraps the underlying
  `toml.DecodeFile` error with `%w`. If `profile.Load` is refactored to
  return a typed error, the recovery branch should be updated to match.

## Verification

- `go test ./...` — green.
- `go build ./...` — green.
