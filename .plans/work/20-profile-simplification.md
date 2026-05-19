# 20 - Profile Simplification

## Goal
Replace the current four-section profile schema (`[agent]`, `[repo]`, `[tmux]`,
`[scripts]`) with a flat, minimal schema where (1) the worktree is an explicit
per-profile opt-in, (2) there is a single `setup` script run between worktree
creation and tmux session creation (plus a single `teardown` script run on
cleanup), and (3) tmux windows/panes are declared uniformly — the agent is
just a pane marked `agent = true`, not a magic auto-created window.

Clean break: no backwards compatibility with the existing TOML format.

## Branch Name
`feat/20-profile-simplification`

## Scope
- Rewrite `internal/profile/profile.go` with a flat `Config` type
- Drop the auto `<session>-agent` window from `internal/tmux/bootstrap.go`;
  drive every window/pane from `spec.Windows`
- Gate worktree create/remove on `cfg.Worktree` in the create and cleanup
  flows (`internal/tui/form.go`, `internal/tui/update.go`)
- Collapse `before_start`/`after_start` into a single `setup` and rename
  `cleanup` to `teardown`; run `setup` after worktree creation and before
  tmux bootstrap; recovery re-runs `setup` and all non-agent pane `run`s
- Rewrite `examples/profiles/default.toml`, `examples/profiles/dev.toml`,
  and `examples/profiles/README.md` to the new schema
- Update all profile/tmux/tui tests to the new schema

## New Schema

```toml
# ~/.config/yaama/profiles/<name>.toml
repo = "/abs/path/to/repo"        # optional; defaults to cwd
default_branch = "main"           # optional; defaults to "main"
worktree = true                   # opt-in; default false

setup    = "scripts/init.sh"      # optional; single command or script path
teardown = "scripts/cleanup.sh"   # optional

layout_file    = "tmux/dev.tmux"  # optional
startup_window = "agent"          # optional; else first window with focus=true

[[windows]]
name  = "agent"
focus = true

  [[windows.panes]]
  run   = "codex --model gpt-5.3-codex"
  agent = true        # initial focus; `run` NOT re-executed on recovery

[[windows]]
name = "ops"

  [[windows.panes]]
  run = "git status -sb"

  [[windows.panes]]
  split = "vertical"
  size  = "30%"
  run   = "make test"
```

Validation:
- At least one `[[windows]]` required.
- At most one pane across the profile may set `agent = true`.
- `setup`/`teardown` accept a single string (command or relative script path
  resolved from `~/.config/yaama/`).

## Phase 1: Schema and Loader
### Steps
1. Replace `Config`/`AgentConfig`/`RepoConfig`/`TmuxConfig`/`ScriptsConfig`
   in `internal/profile/profile.go` with the flat `Config { Repo,
   DefaultBranch, Worktree, Setup, Teardown, LayoutFile, StartupWindow,
   Windows []Window }`. Window/Pane gain `Agent bool`.
2. Rewrite `validateLoadedConfig` for the new shape (require ≥1 window;
   ≤1 agent pane; valid `split` values; non-empty pane `run` when `agent`).
3. Drop `RuntimeValues.AgentCommand`; keep `WorkingDir` and `Branch` only.
4. Update `defaultConfig` to return an empty windows slice with a single
   agent pane (`agent = true`, no `run`).

## Phase 2: Tmux Bootstrap
### Steps
1. Replace `BootstrapSpec.BeforeStart`/`AfterStart`/`Cleanup` and
   `AgentCommand` with `Setup string` and per-pane `Agent bool` in
   `internal/tmux/bootstrap.go`.
2. Remove the auto-create of the `<session>-agent` window and the special
   send-keys block for the agent command. The first window is whatever
   `spec.Windows[0]` is.
3. On bootstrap, run `setup` (if set) after env injection and before
   creating any windows beyond the first. Honour `startup_window` and
   per-window `focus`.
4. On recovery (`recover = true` flag on the spec), re-run `setup` and all
   non-agent pane `run`s; skip `run` for the agent pane.

## Phase 3: TUI Wiring
### Steps
1. `internal/tui/form.go` (`persistCreateForm`): call `gitworktree.Ensure`
   only when `cfg.Worktree`; otherwise use the resolved repo path directly.
2. `internal/tui/bootstrap_spec.go`: translate the new `profile.Config`
   into the new `tmux.BootstrapSpec` (no agent prepend logic).
3. `internal/tui/update.go` cleanup path: gate `gitworktree.Remove` on the
   persisted profile's `worktree` flag; if profile cannot be loaded, skip
   removal and surface a warning toast.
4. `internal/tui/update.go` `runCleanupScripts` → `runTeardownScript`:
   execute `cfg.Teardown` via `tmux.RunShellHook`.
5. `internal/tui/update.go` `buildRecoverySpec`: build the new spec, set
   `Recover = true`.

## Phase 4: Examples and Docs
### Steps
1. Rewrite `examples/profiles/default.toml` (minimal: single agent window,
   no worktree, no setup).
2. Rewrite `examples/profiles/dev.toml` (worktree on, setup/teardown,
   extra `ops` window with a split pane).
3. Rewrite `examples/profiles/README.md` field guide for the new schema.
4. Update `README.md` and `AGENTS.md` references to the old fields.

## Phase 5: Tests
### Steps
1. `internal/profile/*_test.go`: cover load/validation/path resolution
   for the new schema; assert at-most-one agent pane; assert worktree
   default false.
2. `internal/tmux/bootstrap_test.go`: assert no auto agent window;
   recovery skips agent pane `run`.
3. `internal/tui/bootstrap_spec_test.go`: assert mapping; agent pane
   propagation; setup/teardown population.
4. `internal/tui/update_test.go`: teardown invocation; worktree removal
   gated by `worktree` flag; recovery flow.

## Done Criteria
- `go build ./...` and `go test ./...` pass.
- Fresh `examples/profiles/dev.toml` produces a tmux session whose
  windows/panes exactly match the profile (no extra `<session>-agent`
  window) and where the `agent = true` pane is focused.
- Creating a session with `worktree = true` produces a directory under
  `<repo>/.yaama-worktrees/<slug>`; with `worktree = false` it does not.
- `setup` runs between worktree creation and tmux bootstrap; `teardown`
  runs on archive cleanup; both omitted when not set.
- Recovery (`r`) recreates windows/panes and re-runs `setup` and
  non-agent pane `run`s; agent pane `run` is not re-executed.
- `INDEX.md` updated with the new entry, and a `20-profile-simplification-review.md`
  is added in the same commit that marks the box ticked.
