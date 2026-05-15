# 18 - Tmux Bootstrap System Tests

## Goal
Cover the real-tmux contract that work item 15 unified — session env
injection, profile layout application, and create-vs-recovery parity
(agent command suppressed on recovery) — with an integration test suite
that drives an actual `tmux` server and runs in GitHub Actions.

## Branch Name
`test/18-tmux-bootstrap-system-tests`

## Motivation
Existing unit tests stub `tmuxAvailableFn` / `runTmuxFn` /
`sendCommandToPaneFn`, so they prove call ordering against a fake but not
that the real `tmux` server interprets our argument flow the way we
expect. The risk surface is small but concentrated: `set-environment`
ordering, `new-session -d` race against subsequent commands, and the
fact that recovery must rebuild the same shape as create. A single
system-tagged test file plus a CI job is enough to lock this in.

## Scope
- New build-tagged test file `internal/tmux/bootstrap_system_test.go`
  (`//go:build system`) that calls `tmux.BootstrapSession` against the
  real `tmux` binary and asserts observable state via `tmux show-
  environment`, `tmux list-windows`, and a `send-keys` + `capture-pane`
  round-trip for env-var visibility.
- Coverage for both flows:
  - **Create:** profile with one extra window + an `after_start`
    sentinel; assert the extra window exists, both env vars are set
    on the session, pane 0 sees `$YAAMA_TMUX_SESSION` in its shell, and
    the `after_start` sentinel ran.
  - **Recovery:** kill the session, call `BootstrapSession` again with
    `AgentCommand=nil`; assert the layout is rebuilt, env vars are
    reinjected, and no agent send-keys ran (sentinel file for the agent
    command should not appear on the recovery pass).
- One GitHub Actions workflow (`.github/workflows/system-tests.yml`)
  that installs `tmux`, runs `go test -tags=system ./internal/tmux/...`,
  and uploads tmux server logs on failure.

## Non-goals
- Driving the TUI end-to-end (teatest). Recovery branching in the model
  is already covered by unit tests; this work item only validates the
  tmux contract.
- Cross-platform tmux (macOS-on-runners). Linux runner is enough for v1
  — the binary is the same.
- Replacing existing fake-runner unit tests. System tests run on the
  `system` build tag only, so default `go test ./...` stays fast and
  hermetic.

## Design

### Test helpers (internal to the test file)
- `tmuxBinary(t)` — skips the test if `tmux` is not on PATH.
- `uniqueSessionName(t)` — derives a session name from the test name +
  PID + nano timestamp to avoid collisions across parallel runs.
- `withTmuxServer(t)` — points `$TMUX_TMPDIR` at `t.TempDir()` so the
  test owns its own tmux server, then registers `t.Cleanup` that runs
  `tmux kill-server` on the isolated socket. This both isolates parallel
  tests and guarantees no stray sessions outlive the run.
- `assertSessionEnv(t, session, key, want)` — runs
  `tmux show-environment -t <session> <key>` and compares.
- `captureShellEnv(t, paneTarget, varName)` — sends
  `printf '\\nYAAMA_PROBE:%s\\n' "$<var>"` via `send-keys`, polls
  `capture-pane -p` for the `YAAMA_PROBE:` marker (with timeout), and
  returns the captured value.

### Test cases
1. `TestSystemBootstrapCreateAppliesFullLayout`
   - Build `BootstrapSpec` with one extra window (`ops`), one
     `after_start` script that `touch`es a sentinel under `t.TempDir()`,
     and an `AgentCommand` that writes a different sentinel.
   - After `BootstrapSession`, assert:
     - `list-windows` shows the named agent window + `ops`.
     - `show-environment` returns both `YAAMA_TMUX_SESSION` and
       `YAAMA_WORKING_DIR`.
     - Pane 0 shell sees `$YAAMA_TMUX_SESSION` (via the capture helper).
     - The `after_start` sentinel exists.
     - The agent-command sentinel exists (poll with timeout — the
       send-keys + shell exec is async).
2. `TestSystemBootstrapRecoveryRebuildsLayoutWithoutAgentCommand`
   - Run the create case once.
   - `tmux kill-session -t <name>`.
   - Re-call `BootstrapSession` with `AgentCommand=nil` and a *fresh*
     agent-sentinel path that did not exist before.
   - Assert:
     - `list-windows` again shows the agent window + `ops`.
     - Env vars are re-injected (new session, fresh tmux env).
     - `after_start` sentinel is touched a second time (idempotency
       assumption — script appends or uses `mkdir -p` + timestamp).
     - The new agent-sentinel path does **not** exist (recovery did not
       run the agent command).

## Phases

### Phase 1: Helpers + create-path test
1. Add `internal/tmux/bootstrap_system_test.go` with the `//go:build
   system` constraint and the four helpers above.
2. Implement `TestSystemBootstrapCreateAppliesFullLayout` end-to-end on
   a real tmux server. Iterate locally with `go test -tags=system
   -run TestSystemBootstrapCreate ./internal/tmux/...`.

### Phase 2: Recovery-path test
1. Add `TestSystemBootstrapRecoveryRebuildsLayoutWithoutAgentCommand`
   using the same helpers, asserting the no-agent-command invariant by
   sentinel-file absence.
2. Make the after-start script idempotent (timestamped sentinel file or
   `mkdir -p`) so it can run on both passes without false failures.

### Phase 3: CI workflow
1. Add `.github/workflows/system-tests.yml`:
   - Triggers: `push` to `main` and `pull_request`.
   - Job on `ubuntu-latest`:
     - `actions/setup-go@v5` pinned to repo `go.mod` version.
     - `sudo apt-get update && sudo apt-get install -y tmux`.
     - `tmux -V` (logged for postmortem clarity).
     - `go test -tags=system -count=1 -timeout=120s ./internal/tmux/...`.
   - On failure, upload `t.TempDir()` contents — or at minimum
     `tmux server logs` if we add a `-vvv` mode — as an artifact.
2. Reference the workflow in `README.md` (Testing section) and call out
   that default `go test ./...` does **not** include system tests.

### Phase 4: Stability hardening
1. Add a `pollUntil(t, timeout, fn)` helper used wherever the test
   reads tmux state right after a write (pane capture, sentinel-file
   existence). Default timeout 2s with 50ms steps.
2. In `withTmuxServer`, also `t.Setenv("TMUX", "")` so any inherited
   tmux env on a dev machine cannot leak into the test's session.

## Definition of Done
- Running `go test -tags=system ./internal/tmux/...` locally with `tmux`
  installed passes both new tests in under ~10 seconds.
- Running `go test ./...` (no tag) still completes without invoking the
  system tests.
- GitHub Actions workflow runs on every PR and main push; both tests
  green on `ubuntu-latest`.
- README testing section documents how to run system tests locally and
  in CI.

## Open Questions
1. **Workflow scope.** Run system tests on `push` + `pull_request`, or
   only on `pull_request` to keep main fast? Default: both — system
   tests are short and the signal on main is worth the seconds.
2. **macOS runner.** Skip for v1 (tmux behavior is the same on
   linux/darwin for our usage), or add a parallel job? Default: skip,
   revisit only if we ship a macOS-specific tmux flag.
3. **Sentinel for "agent command did not run."** Asserting absence is
   inherently a timing question. Default: write the sentinel from a
   command that is fast-to-exec on create (so the create case can poll
   for presence with timeout), and on recovery poll for *absence* using
   the same poll budget — i.e. if we still don't see the sentinel after
   the create-case timeout window, recovery did not run it. Document
   this assumption inline.
