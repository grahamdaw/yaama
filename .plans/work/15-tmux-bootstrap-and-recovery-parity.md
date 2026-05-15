# 15 - Tmux Bootstrap And Recovery Parity

## Goal
Make tmux session setup consistent and predictable across both **new agent
creation** and **dead-session recovery**, with a single bootstrap pipeline that
- always opens a default agent window named after the session,
- always lays out the profile's additional windows/panes and runs the profile's
  tmux setup script,
- always injects the session name into every shell in the session,
- and runs the agent command **only** on new creation (never on recovery).

## Motivation
Today new-session bootstrap (`form.persistCreateForm` →
`tmux.BootstrapSession`) does the full profile setup (windows, panes, before/
after hooks, agent command launch), but recovery
(`update.recreateSelectedSession` → `tmux.CreateDetachedSessionCommand`) only
runs a bare `tmux new-session -d -s <name> -c <wd>`. That means a recovered
session has no extra windows/panes, no exported `YAAMA_TMUX_SESSION`, and no
profile-supplied tmux layout — operators land in a stripped-down shell that
looks nothing like the original. We want recovery to "rebuild the room" minus
the agent process.

## Scope
- Refactor the tmux bootstrap entry point so the same code path serves both
  flows, parameterized by whether the agent command should be launched.
- Inject the session name (and working dir) into the tmux session environment
  so every pane shell sees `YAAMA_TMUX_SESSION` / `YAAMA_WORKING_DIR` without
  per-pane export shims.
- Wire recovery to load the agent's profile and run the same bootstrap with
  `AgentCommand` suppressed.
- Update profile schema docs if a dedicated `tmux_setup` hook is added.

## Non-goals
- Re-launching the agent process on recovery (explicitly out of scope per
  operator decision — recovery rebuilds the shell environment only).
- Changing cleanup/prune semantics.
- Changing the create-form UX (steps, fields, validation).
- Changing existing profile TOML field names that operators already depend on.

## Current State (verified 2026-05-15)
- `internal/tui/form.go:382-389` builds a `tmux.BootstrapSpec` and calls
  `tmux.BootstrapSession` on create. The spec carries
  `AgentCommand`, `BeforeStart`, `AfterStart`, `LayoutFile`, `Windows`, etc.
- `internal/tmux/bootstrap.go:44` does, in order:
  before_start hooks → `new-session -d` → rename initial window to
  `AgentWindow` → apply extra windows/panes → optional `source-file` of
  `LayoutFile` → after_start hooks → `send-keys` agent command into `:0.0` →
  `select-window` startup window.
- `internal/tui/update.go:849 recreateSelectedSession` only validates
  `working_dir` and runs `tmux.CreateDetachedSessionCommand`
  (`internal/tmux/tmux.go:89`), which issues `new-session -d -s X -c WD ; set
  destroy-unattached off`. No profile is loaded, no windows applied, no env
  injected, no scripts run.
- `RunShellHook` (`internal/tmux/bootstrap.go:210`) already exports
  `YAAMA_TMUX_SESSION` and `YAAMA_WORKING_DIR` to before/after-start scripts,
  but those vars are not set inside the tmux session itself — pane shells
  spawned by `new-session`/`split-window` do not inherit them.
- Profile schema (`internal/profile/profile.go:46-64`) currently has
  `Scripts.BeforeStart` / `AfterStart` / `Cleanup`. Profiles in
  `profiles/profiles/*.toml` conventionally point `after_start` at a
  `scripts/tmux/*.sh` setup script (e.g. `simple.toml:11`,
  `hyrskytin-full.toml:11`). We will reuse this convention rather than invent
  a new hook field — see "Open Question 1".

## Target Behavior

### New session create (unchanged outcomes, refactored path)
1. Resolve profile + runtime (working_dir, branch, agent command).
2. Create worktree, persist agent row.
3. Run unified `BootstrapSession` with:
   - default agent window named after the session (currently the session name
     itself doubles as `AgentWindow`),
   - session-scoped env injection (`set-environment -t <session>
     YAAMA_TMUX_SESSION=<name>` and same for `YAAMA_WORKING_DIR`),
   - before_start scripts,
   - additional windows/panes from profile,
   - layout file `source-file`,
   - after_start scripts (the "tmux setup script" today),
   - `AgentCommand` sent to the default window's pane 0,
   - select startup window.

### Dead-session recovery (new behavior)
1. Validate working_dir as today.
2. Load the profile recorded on the agent row (`selected.ProfileName`). If the
   profile no longer exists, fall back to a minimal spec (no extra windows, no
   scripts) and surface a warning toast so the operator knows the layout is
   incomplete.
3. Run unified `BootstrapSession` with the **same spec as create except
   `AgentCommand = nil`**. The default agent window is still named after the
   session, env is still injected, before/after-start scripts still run,
   extra windows/panes still get laid out.
4. Mark recovered (DB heartbeat, `cleanup_state=active`, etc. — keep current
   `markRecovered`).
5. Attach.

## Design

### Single bootstrap entry point
Keep `tmux.BootstrapSession(ctx, spec)`; treat `spec.AgentCommand == nil` as
"do not send agent command." (Already mostly true via the
`len(spec.AgentCommand) > 0` guard at `internal/tmux/bootstrap.go:83`.) Verify
that with `nil` the rest still runs cleanly.

### Session-scoped env injection
After `new-session -d`, before any `send-keys` to panes, run:

```
tmux set-environment -t <session> YAAMA_TMUX_SESSION <session>
tmux set-environment -t <session> YAAMA_WORKING_DIR  <working_dir>
```

This makes the vars available to every shell spawned by subsequent
`new-window` / `split-window` calls in that session. (Existing pane-0 shell
that was created by `new-session` won't pick them up retroactively — but we
already issue a `cd <wd>` send-keys to pane 0, so we can also export them
there explicitly to cover that single pane. Open Question 2.)

### Recovery wiring
In `update.recreateSelectedSession`:
- Replace the call to `createDetachedCmd` with a call to a new injectable
  `recoverSession func(ctx, BootstrapSpec) error` (defaulting to
  `tmux.BootstrapSession`).
- Build the spec from the **same** code that `form.toBootstrapSpec` uses —
  extract `toBootstrapSpec` (or a shared helper) into a place both files can
  call. Pass `AgentCommand = nil`.
- If profile load fails, log + toast + fall through with a minimal spec
  (`SessionName`, `WorkingDir`, `AgentWindow = selected.TmuxSession`, no
  windows, no scripts).

### Naming / window defaults
`AgentWindow` already defaults to the session name on the create path
(`form.go:411 AgentWindow: sessionName`). Recovery must do the same — pass
`selected.TmuxSession` as `AgentWindow` so the default window is named after
the session, matching requirement #1.

### Tmux setup script
Today the convention is `[scripts].after_start = ["scripts/tmux/*.sh"]`. We
reuse this; recovery just runs the same `AfterStart` list. No new TOML key
required — see Open Question 1 for the alternative.

## Phases

### Phase 1: Shared spec builder & env injection
1. Move `toBootstrapSpec` (currently `internal/tui/form.go:407`) into a shared
   location callable from both `form.go` and `update.go`. Likely
   `internal/tui/bootstrap_spec.go` or a method on `model`.
2. Add `set-environment -t <session> YAAMA_TMUX_SESSION` / `YAAMA_WORKING_DIR`
   inside `tmux.BootstrapSession` immediately after `new-session -d`, before
   `applyWindowsAndPanes`.
3. Also export the same vars in pane 0's first send-keys so the pre-existing
   shell picks them up.
4. Tests: extend `internal/tmux/bootstrap_test.go` to assert
   `set-environment` is issued in order, and that with `AgentCommand=nil`
   no `send-keys` for the agent command is emitted.

### Phase 2: Recovery uses unified bootstrap
1. Add `bootstrapSession` injection point reuse to `update.go` (or a new
   `recoverSession` field) so recovery can call the same function the create
   path calls.
2. In `recreateSelectedSession`:
   - Resolve profile via `m.loadProfileFn` (or `profile.Load`) using
     `selected.ProfileName`. Trim/validate; on miss, build minimal spec and
     `pushWarning` "Profile X not found; recovered with minimal layout."
   - Build spec via the shared builder with `AgentCommand = nil`,
     `AgentWindow = selected.TmuxSession`, `WorkingDir = selected.WorkingDir`.
   - Call unified bootstrap.
3. Keep current preflight checks (tmux available, isDead, working_dir exists
   and is dir).
4. Keep `markRecovered` + attach behavior unchanged.

### Phase 3: Failure handling & tests
1. If profile load fails for a *non-missing* reason (parse error), record via
   `recordRecoveryError` and stop — do **not** silently fall back, because
   that would mask a corrupted profile.
2. If bootstrap fails partway (e.g. after-start script error), the tmux
   session may still exist; surface a warning that explains the operator can
   attach manually but layout is incomplete. Do not auto-kill — let the user
   inspect.
3. Tests in `internal/tui/update_test.go`:
   - recovery with valid profile applies extra windows + skips agent command,
   - recovery with missing profile falls back to minimal spec with warning,
   - recovery with parse-error profile records `last_error` and aborts,
   - env injection set-environment calls observed in fake tmux runner.

### Phase 4: Docs & profile examples
1. Update `README.md` recovery section to describe new behavior: "Recovery
   recreates the full tmux layout (windows, panes, after-start setup script)
   but does not relaunch the agent process. Restart the agent manually inside
   the agent window if needed."
2. Update `AGENTS.md` if it documents the recovery flow.
3. Confirm `examples/profiles/*.toml` still work end-to-end on recovery; add
   a short note in `examples/profiles/README.md` clarifying that
   `after_start` scripts run on both create and recovery.

## Definition of Done
- New and recovered sessions produce visually identical tmux layouts (same
  windows, panes, env vars) except that recovered sessions have no agent
  process running in window 0.
- `YAAMA_TMUX_SESSION` is set in every shell spawned inside the session
  (verifiable via `tmux send-keys ... "echo $YAAMA_TMUX_SESSION"`).
- Profile after-start ("tmux setup") scripts run on both create and recovery.
- Recovery does not launch the agent command under any circumstances.
- Missing/corrupt profiles on recovery degrade safely with operator-visible
  warnings, not silent stripped sessions.
- Tests cover: shared spec build, env injection, recovery skips agent
  command, recovery missing-profile fallback, recovery parse-error abort.

## Open Questions
1. **Dedicated `[scripts].tmux_setup` field vs. reusing `after_start`?**
   Pro dedicated: clearer intent, lets `after_start` mean "after agent
   launched." Pro reuse: zero migration, current profiles already work.
   Default: reuse `after_start` and document it; revisit if operators ask.
2. **Env injection for pane 0:** `set-environment` only affects shells
   spawned *after* it runs, so pane 0 (created by `new-session`) won't see
   the vars unless we also send-keys an export. Acceptable? Or kill+recreate
   pane 0? Default: send-keys export — simpler and idempotent.
3. **Recovery of agents created with no profile recorded** (e.g. legacy
   rows). Plan: treat null `profile_name` as the minimal-spec path with no
   warning (it's the expected state for those rows).
