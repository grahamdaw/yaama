# 21 — Profile TOML As The Single, Inspectable Source Of Truth — Review

## Summary
All three phases shipped on `refactor/21-profile-toml-single-source-of-truth`:
1. `internal/agenthook` refactored from parser registry to Harness registry;
   profile schema gains required `[agent].harness`, optional `[agent.env]`,
   and harness-default fill.
2. `[tmux].preset` catalog (`solo`, `agent+logs`, `agent+tests`,
   `agent+git+tests`) with mutual-exclusion against `[[tmux.windows]]`, and
   per-window `layout` applied via `tmux select-layout`.
3. New `yaama profile check <name>` subcommand, refreshed example profiles
   (`default.toml`, `dev.toml`, `kiro.toml`), `examples/tmux/` deleted, and
   README + AGENTS.md docs.

## Deviations from the plan

- **Done-criteria grep — `layout_file` still appears outside `*-review.md`.**
  The plan's grep target excludes only historical `*-review.md` files. The
  remaining hits are intentional:
  - `internal/profile/profile.go` and `internal/profile/profile_test.go`:
    the rejection logic + its tests.
  - `.plans/001_SPEC_ITERATION_UX.md` and `.plans/work/15-...md`: historical
    spec narrative left untouched per the convention used by prior items.
  Reviewer should treat these as expected.

- **No `yaama check <name>` alias added.** The plan said "decide in PR";
  we opted for one canonical command (`yaama profile check <name>`) to keep
  the CLI router simple.

- **`agent+logs` preset uses a generic shell expansion.** The plan said
  `tail -F` against the action log path. Since the bootstrap does not
  inject `YAAMA_LOG_PATH` into the tmux env today, the preset tails
  `${YAAMA_LOG_PATH:-$HOME/.local/state/yaama/yaama.log}`. Operators on a
  non-standard XDG state path can either set `YAAMA_LOG_PATH` in
  `[agent.env]` (which will reach the agent process but not the `logs`
  pane shell) or fall back to longhand `[[tmux.windows]]`. A follow-up
  could plumb the resolved log path into `tmux set-environment`; out of
  scope here.

- **`profile check` output is structured per-spec but does not re-use a
  shared `tmux.BuildBootstrapSpec`.** There is no such helper today; the
  launch flow fuses spec construction with execution inside
  `BootstrapSession`. The plan called this out as a sequencing concern.
  Rather than extracting a planner-only helper in this work item (which
  would touch the create/recover paths), the new
  `cmd/yaama/profile_check.go` derives an equivalent plan from
  `profile.Config` + `RuntimeValues`. The risk is drift between the
  printed plan and the actual launch; mitigated by deriving from the same
  config + the small surface area of the bootstrap. A separate refactor
  to extract a real `BuildBootstrapSpec`/`Plan` would be a clean
  follow-up.

- **Harness default commands are best-effort guesses.** For harnesses we
  haven't integrated yet (`kiro-cli`, `kiro`, `copilot`,
  `vscode-copilot`), `Defaults()` returns a plausible command + args
  pair. Operators will likely override these in real profiles until each
  harness ships proper integration.

## Done criteria check

- ✅ Profile without `agent.harness` fails to load listing the registered ids.
- ✅ `harness = "claude-code"` + no `command` launches `claude` (default).
- ✅ `preset = "agent+tests"` + no `[[tmux.windows]]` expands to one extra
   window (`tests`) at bootstrap time; with the implicit default agent
   window that's two windows total.
- ✅ Both `preset` and `[[tmux.windows]]` set → load fails with mutual
   exclusion error.
- ✅ `yaama profile check default` on a fresh checkout prints the plan
   and exits 0; tested via `cmd/yaama/profile_check_test.go`.
- ✅ Broken profile → `yaama profile check` prints the same error as a
   launch would and exits non-zero.
- ✅ `yaama hook claude-code` unchanged. `yaama hook codex` (and the
   other launch-only ids) surface `ErrHarnessHasNoHook` cleanly and
   exit non-zero.
- ✅ `examples/tmux/` deleted; `examples/profiles/` contains
   `default.toml`, `dev.toml`, `kiro.toml`.
- ✅ README has Harnesses, Presets, and Verify a profile subsections.

## Tests added

- `internal/agenthook/registry_test.go`: registry shape (sorted IDs,
   unknown lookup) and sentinel-hook coverage for the five launch-only
   harnesses.
- `internal/profile/profile_test.go`: missing/unknown harness, defaults
   applied, operator-set values win, `layout_file` rejection,
   preset/longhand mutual exclusion, unknown preset, preset expansion.
- `internal/profile/presets_test.go`: catalog sortedness and the
   `agent+git+tests` expansion shape.
- `internal/tmux/bootstrap_test.go`: per-window `select-layout`
   assertion and a regression that `source-file` never runs.
- `cmd/yaama/profile_check_test.go`: valid-profile plan output,
   validation-error path, missing-positional usage.

## Verification

- `go vet ./...` — clean.
- `go test ./...` — passes.
- `gofmt -l` — clean (the unrelated `internal/tui/view_test.go` formatting
  drift predates this branch).
- System tests (`go test -tags=system ./internal/tmux/...`) were not run
  locally; CI workflow runs them on PR.
