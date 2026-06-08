# 18 - Tmux Bootstrap System Tests — Review

## Outcome

Both required test cases land in `internal/tmux/bootstrap_system_test.go`
under the `//go:build system` tag, and a dedicated GitHub Actions
workflow runs them on `ubuntu-latest`. Local run:

```
go test -tags=system -count=1 -timeout=60s ./internal/tmux/...
ok  	github.com/grahamdaw/yaama/internal/tmux	6.323s
```

Default `go test ./...` continues to skip the system suite.

## Plan vs Actual

- **Helpers** — implemented as specified: `tmuxBinary`, `uniqueSessionName`,
  `withTmuxServer`, `assertSessionEnv`, `captureShellEnv`, plus the
  `pollUntil` helper called out in Phase 4. Added one un-planned helper
  (`listWindowNames`) for cleaner window assertions.
- **`uniqueSessionName`** — deviated from the plan: the spec called for
  `<test name>-<pid>-<nano>`, but the test-name prefix combined with the
  PID + full nanosecond stamp overflowed pane-capture line width and
  truncated the `YAAMA_PROBE:` marker. Shortened to `yst-<pid>-<nano mod 1e6>`
  which is still collision-safe for parallel runs and short enough to
  survive an 80-column pane capture.
- **`withTmuxServer`** — deviated from the plan's "point `TMUX_TMPDIR`
  at `t.TempDir()`". On darwin the nested `t.TempDir()` path exceeds
  the `AF_UNIX` `sun_path` (~104 byte) limit and tmux fails with
  `File name too long`. Switched to `os.MkdirTemp("", "yst-")` which
  lives directly under the OS temp root, and added an explicit
  safe-to-remove guard (prefix match + parent-dir match) before
  `os.RemoveAll`.
- **Sentinel "absence" assertion** — implemented exactly as the open
  question recommends: the recovery test uses the same 5s poll budget
  the create test uses to confirm sentinel presence, then asserts the
  recovery-only sentinel is still absent after that window. The
  rationale is documented inline at the assertion site.
- **CI workflow** — landed as `.github/workflows/system-tests.yml` on
  `push` to `main` and `pull_request`, matching the plan's default for
  Open Question 1. macOS runner intentionally skipped per Open
  Question 2. The failure-artifact upload globs `/tmp/yst-*/**` (where
  our isolated tmux tmpdirs live) rather than a `t.TempDir()` path the
  workflow cannot resolve.

## Non-goals respected

- No TUI / teatest coverage added — recovery branching in the TUI
  stays on its existing unit tests.
- Existing fake-runner unit tests under `internal/tmux/` are untouched.
- No macOS runner job; Linux is enough for v1.

## Follow-ups

- None required. If we ever need `-vvv` tmux server logs, the workflow
  is positioned to swap in a richer log-collection step without test
  changes.
