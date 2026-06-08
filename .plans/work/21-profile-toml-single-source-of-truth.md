# 21 - Profile TOML As The Single, Inspectable Source Of Truth (Multi-Harness)

## Goal
Make the profile TOML the **only** place an operator configures an
agent, and give them a way to *see* what a profile resolves to before
launching a session. Three problems today:

1. **Layout is described twice.** `[[tmux.windows]]` blocks and an
   external `tmux.layout_file` both describe tmux state, and the
   external file is sourced *after* windows are created, so it can
   silently rewrite the layout. `internal/tmux/bootstrap.go` has to
   merge two languages and the README needs a callout explaining the
   ordering.
2. **The `[agent]` block is harness-blind.** yaama is converging on
   supporting Codex, Kiro CLI, Kiro, Copilot, and VS Code Copilot.
   Today `[agent]` is a generic `command`/`args` pair with no way to
   declare which harness's launch defaults and hook contract are in
   play — every operator has to know the right command line.
3. **Profiles are unverifiable until you launch them.** The only way
   to find out whether a profile is correctly wired is to create an
   agent and watch tmux bootstrap succeed or fail.

After this work item:
1. `tmux.layout_file` is gone. Layout lives inside the profile via
   either a built-in preset (`[tmux] preset = "agent+logs"`) or
   longhand `[[tmux.windows]]` blocks. One file, one language.
2. `[agent].harness = "<id>"` names the harness explicitly and is the
   key that a small harness registry uses to supply launch defaults
   and (where supported) parse hook payloads. Registry ships with
   entries for `codex`, `kiro-cli`, `kiro`, `copilot`, `vscode-copilot`,
   and `claude-code`.
3. `yaama profile check <name>` resolves the profile against the
   current repo and prints the exact sequence of git + tmux commands it
   would run, plus the resolved working directory, branch, and agent
   command line — without launching anything.

## Branch Name
`refactor/21-profile-toml-single-source-of-truth`

## Scope
- Profile schema:
  - Remove `tmux.layout_file` from the TOML schema and from
    `profile.TmuxConfig`.
  - Add `tmux.preset` (optional string). Accepted values come from a
    small built-in catalog: `solo`, `agent+logs`, `agent+tests`,
    `agent+git+tests`. `preset` and `[[tmux.windows]]` are mutually
    exclusive — set one or the other, not both.
  - Add `agent.harness` (string, required). Accepted values come from
    the harness registry. Unknown values fail load with the sorted
    list of registered ids.
  - Add optional `agent.env` (`map[string]string`) so harness-specific
    env vars live in the profile rather than wrapper scripts.
- Bootstrap:
  - Delete the `source-file` step in `internal/tmux/bootstrap.go`.
    `BootstrapSpec.LayoutFile` is removed.
  - Presets expand into the same `BootstrapWindow` / `BootstrapPane`
    structures the longhand form produces; bootstrap is agnostic to
    which form the operator wrote.
  - Add `BootstrapWindow.Layout string` (mapped from
    `[[tmux.windows]].layout` in TOML, or set by a preset). After a
    window's panes are created, run `tmux select-layout -t
    <session>:<window> <layout>` if non-empty. This covers the one
    thing `examples/tmux/dev-layout.tmux` was actually doing.
- Harness registry:
  - Promote `internal/agenthook` from "hook parser registry" to a
    small "harness registry". Each harness declares `ID()`,
    `Defaults() AgentDefaults` (command + args + env), and
    `ParseHook(payload) (StatusUpdate, error)`.
  - Initial registry entries:
    | id | hook parser | notes |
    |---|---|---|
    | `claude-code` | yes (existing) | full hook integration |
    | `codex` | not yet | launch defaults only |
    | `kiro-cli` | not yet | launch defaults only |
    | `kiro` | not yet | launch defaults only |
    | `copilot` | not yet | launch defaults only |
    | `vscode-copilot` | not yet | launch defaults only |
  - "Not yet" parsers return a sentinel `ErrHarnessHasNoHook` so
    `yaama hook <id>` exits with a clear message rather than crashing.
    Adding hook support for a harness later is a self-contained
    follow-up: drop a parser into the harness's file, no schema
    change.
  - `internal/profile.Load` consults the registry to validate
    `agent.harness` and fill `agent.command`/`args`/`env` from
    harness defaults when the operator omits them. Operator-set
    values always win.
- `yaama profile check <name>` CLI:
  - Resolves the profile against the current working directory and
    git repo.
  - Prints, in order: resolved working dir, branch, agent command
    line (with env), the full sequence of git worktree commands, the
    full sequence of tmux commands (`new-session`, `set-environment`,
    `new-window`, `split-window`, `select-layout`, `send-keys`),
    and the `before_start` / `after_start` shell hooks.
  - Exit 0 if the profile resolves cleanly, non-zero on validation
    failure with the same error messages `profile.Load` would emit
    during a real launch.
  - Does **not** execute anything. No tmux session is created, no
    worktree is created, no hooks run.
- Migration:
  - On profile load, if `tmux.layout_file` is present, fail with:
    `profile <name>: tmux.layout_file is no longer supported;
    declare layout inline via [tmux] preset = "<name>" or
    [[tmux.windows]].layout, or move arbitrary tmux commands into
    scripts.before_start`.
  - No silent default for `agent.harness` — fail with the registered
    id list.
- Examples + docs:
  - `examples/profiles/default.toml`: `harness = "claude-code"`,
    `[tmux] preset = "solo"`.
  - `examples/profiles/dev.toml`: `harness = "codex"`,
    `[tmux] preset = "agent+tests"`, plus `scripts.before_start`.
  - `examples/profiles/kiro.toml`: new example showing
    `harness = "kiro-cli"` with `[[tmux.windows]]` longhand for an
    operator who wants the escape hatch.
  - Delete `examples/tmux/` and its README references.
  - README: rewrite the "Profiles" section, add a "Harnesses"
    subsection listing supported ids and which have hook
    integration, add a "Presets" subsection listing the catalog, add
    a "Verify a profile" subsection covering `yaama profile check`.

## Non-goals
- Shipping hook parsers for `codex`, `kiro-cli`, `kiro`, `copilot`,
  or `vscode-copilot` in this work item. We're declaring the seam and
  the registry entries so launch works; hook integration per harness
  lands as separate work items keyed to each harness's hook contract.
- Operator-defined presets. The preset catalog is a closed set in
  this work item; the longhand `[[tmux.windows]]` form is the escape
  hatch.
- Profile inheritance, includes, or composition. One file per profile.
- Supporting raw tmux script files as a fallback. Arbitrary tmux
  commands go in `scripts.before_start` / `scripts.after_start`.
- Auto-migrating existing profiles. Fail loudly with a one-line
  migration recipe in the error message.
- A "dry run" mode for `yaama new` that goes through bootstrap but
  rolls back. `profile check` prints the plan; that's enough.

## Design

### TOML schema (post change)
```toml
[agent]
harness = "codex"             # required; one of the registered ids
command = "codex"             # optional; harness default if omitted
args    = ["--model", "..."]  # optional; harness default if omitted

[agent.env]
OPENAI_LOG = "debug"

[repo]
path           = "/abs/path/to/repo"
default_branch = "main"

[tmux]
preset         = "agent+tests"   # OR use [[tmux.windows]] below
startup_window = "agent"
# layout_file removed

[scripts]
before_start = ["scripts/init.sh"]
cleanup      = ["scripts/cleanup.sh"]
```

Longhand form (mutually exclusive with `preset`):
```toml
[tmux]
startup_window = "agent"

[[tmux.windows]]
name   = "ops"
focus  = true
layout = "even-vertical"

[[tmux.windows.panes]]
cwd = "."
run = "git status -sb"

[[tmux.windows.panes]]
split = "vertical"
size  = "30%"
cwd   = "."
run   = "make test"
```

### Preset catalog (initial)
| Preset | Expansion |
|---|---|
| `solo` | No extra windows. Agent runs in window 0 only. |
| `agent+logs` | Adds window `logs` (focus=false), single pane running `tail -F` against the action log path. |
| `agent+tests` | Adds window `tests` (focus=false), single pane in repo root, no auto-run command. |
| `agent+git+tests` | Adds window `ops` (focus=false) with two panes: `git status -sb` and `make test` (vertical split, 30%). |

Presets expand at profile load time into `[]BootstrapWindow`. The
operator can `yaama profile check` to see exactly what they unrolled
into. Catalog lives in `internal/profile/presets.go` as a single
`map[string]func() []TmuxWindow`. Adding a preset is one entry.

### Harness registry surface
```go
// internal/agenthook/registry.go
type Harness interface {
    ID() string                                // "codex", "kiro-cli", ...
    Defaults() AgentDefaults                   // command/args/env when profile omits them
    ParseHook(payload []byte) (StatusUpdate, error)
}

type AgentDefaults struct {
    Command string
    Args    []string
    Env     map[string]string
}

var ErrHarnessHasNoHook = errors.New("harness has no hook integration yet")

func Register(h Harness)
func Get(id string) (Harness, bool)
func IDs() []string             // sorted, for error messages
```
- Each harness gets one file under `internal/agenthook/` named after
  its id (`codex.go`, `kiro_cli.go`, etc.). The file declares the
  struct, registers it in `init()`, and exports defaults + (where
  available) a hook parser.
- `internal/profile.Load` calls `agenthook.Get(cfg.Agent.Harness)`,
  errors with `IDs()` if missing.
- Defaults are applied only where the operator left the field empty.
  Operator-set values always win.

### `yaama profile check` design
- Subcommand: `yaama profile check <name>` (alias `yaama check <name>`
  if that reads cleaner — decide in PR).
- Reuses the same `profile.Load` and `tmux.BuildBootstrapSpec` code
  paths the launch flow uses, then prints the plan instead of
  executing it.
- Output format: human-readable sections (`Resolved`, `Git`, `Tmux`,
  `Hooks`), each one a numbered list of commands or values. Plain
  text, no colors. One section per phase of the launch.
- Exits non-zero on any validation error encountered during load or
  plan construction; prints the same error message the launch flow
  would surface.
- Lives in `cmd/yaama/profile_check.go`. No new internal package —
  the planner just calls existing functions with a side-effect-free
  `Renderer` instead of a real executor.

### Bootstrap changes
- `BootstrapSpec.LayoutFile` removed. `BootstrapWindow` gains
  `Layout string`. After `applyWindowsAndPanes` finishes creating a
  window's panes, if `Layout` is non-empty, run
  `tmux select-layout -t <session>:<window> <layout>`.
- The "window 0 is the agent; declared windows come after" rule
  stays — it's real and useful; it just no longer has the surprise
  of a layout file rewriting things after the fact.
- README's "TMUX bootstrap behavior" callout shrinks to one bullet.

### Failure modes (load-time)
| Condition | Behavior |
|---|---|
| `agent.harness` missing | `profile <name>: agent.harness is required; one of [claude-code codex copilot kiro kiro-cli vscode-copilot]` |
| `agent.harness` unknown | `profile <name>: agent.harness "<x>" is not registered; one of [...]` |
| `tmux.layout_file` set | `profile <name>: tmux.layout_file is no longer supported; declare layout inline via [tmux] preset = "<name>" or [[tmux.windows]].layout, or move tmux commands into scripts.before_start` |
| `tmux.preset` and `[[tmux.windows]]` both set | `profile <name>: tmux.preset and [[tmux.windows]] are mutually exclusive` |
| `tmux.preset` unknown | `profile <name>: tmux.preset "<x>" is not in the catalog; one of [agent+git+tests agent+logs agent+tests solo]` |
| `[[tmux.windows]].layout` invalid | Deferred to tmux itself — fail at bootstrap, log line emitted with the tmux stderr. (Caught earlier by `profile check` against an obvious typo? No — we don't validate tmux layout names ourselves.) |

### What moves where
- `internal/profile/profile.go`: drop `LayoutFile`, add `Harness`,
  `Env`, `Preset`, `[[TmuxWindow]].Layout`. Call into `agenthook.Get`
  during load for validation + defaults. Call `presets.Expand` when
  `Preset` is set.
- `internal/profile/presets.go`: new file, preset catalog.
- `internal/agenthook/`: existing `claudecode.go` becomes a `Harness`.
  Add `codex.go`, `kiro_cli.go`, `kiro.go`, `copilot.go`,
  `vscode_copilot.go` — each registers a harness with defaults and a
  `ParseHook` that returns `ErrHarnessHasNoHook`.
- `internal/tmux/bootstrap.go`: delete `LayoutFile`, add per-window
  `select-layout` step.
- `cmd/yaama/profile_check.go`: new subcommand.
- `cmd/yaama/hook.go`: surface `ErrHarnessHasNoHook` as a clean
  message instead of a stack trace.

## Phases

### Phase 1: Harness registry + profile schema
1. Refactor `internal/agenthook` from a parser registry into a harness
   registry per the surface above. Existing `Register(parser)` calls
   become `Register(harness)`. Update `cmd/yaama hook` to look up the
   harness by id and call `ParseHook`; format
   `ErrHarnessHasNoHook` as a one-line operator message.
2. Add registry entries: `claude-code` (full), `codex`, `kiro-cli`,
   `kiro`, `copilot`, `vscode-copilot` (launch defaults + sentinel
   hook parser).
3. Add `Harness`, `Env`, `Preset` to `profile.AgentConfig` /
   `TmuxConfig`. Add `Layout` to `profile.TmuxWindow`. Wire
   validation: required harness, accepted ids, mutually-exclusive
   preset/longhand, defaults filled from registry.
4. Tests:
   - `profile_test.go`: missing/unknown harness, preset+longhand
     collision, unknown preset, defaults applied only when operator
     omits them, operator values always win.
   - `agenthook` registry test: `IDs()` sorted; unknown id returns
     `false`; `ErrHarnessHasNoHook` returned for the five new
     harnesses.

### Phase 2: Preset expansion + bootstrap migration
1. Add `internal/profile/presets.go` with the four-entry catalog.
   Expansion is pure (no I/O) so it's easy to unit test.
2. Remove `LayoutFile` from `tmux.BootstrapSpec` and all
   constructors. Update `internal/tui` create + recovery paths.
3. Implement per-window `select-layout` after pane creation.
4. Refuse profiles that still set `tmux.layout_file` with the load
   error described above.
5. Tests:
   - `presets_test.go`: each preset expands to the documented
     `[]TmuxWindow` structure.
   - `internal/tmux/bootstrap_test.go`: replace the `source-file`
     assertion with the new `select-layout` assertion; regression
     test that no `source-file` command is ever issued.

### Phase 3: `yaama profile check` + examples + docs
1. Add `cmd/yaama/profile_check.go` and wire it into the CLI router.
   Implement by extracting the existing bootstrap-plan construction
   into a function the launch path also uses.
2. Update example profiles:
   - `examples/profiles/default.toml`: harness `claude-code`, preset
     `solo`.
   - `examples/profiles/dev.toml`: harness `codex`, preset
     `agent+tests`.
   - Add `examples/profiles/kiro.toml`: harness `kiro-cli`,
     longhand `[[tmux.windows]]` form.
3. Delete `examples/tmux/` and its README references.
4. README rewrite:
   - "Profiles" section: one-file rule, harness selection, preset vs
     longhand, example.
   - New "Harnesses" subsection: id table with hook-support column.
   - New "Presets" subsection: catalog table.
   - New "Verify a profile" subsection: `yaama profile check`
     example output.
   - Remove the "TMUX bootstrap behavior" three-bullet callout; keep
     one line about window ordering.
5. AGENTS.md: update the profile-load + harness-registry pointers so
   future agents know where to add a new harness or preset.

## Done Criteria
- `grep -r 'layout_file\|LayoutFile' --include='*.go' --include='*.toml' --include='*.md'`
  returns nothing outside historical `*-review.md` files.
- A profile without `agent.harness` fails to load with the listed
  ids in the message.
- A profile with `agent.harness = "claude-code"` and no
  `agent.command` launches the harness default command.
- A profile with `[tmux] preset = "agent+tests"` and no
  `[[tmux.windows]]` blocks produces the documented two-window
  layout at bootstrap time.
- A profile setting both `preset` and `[[tmux.windows]]` fails load
  with the mutual-exclusion error.
- `yaama profile check default` on a fresh checkout prints a
  multi-section plan, executes nothing, and exits 0.
- `yaama profile check` on a broken profile prints the same error
  message a launch would and exits non-zero.
- `yaama hook claude-code` still works end-to-end against a live
  tmux session. `yaama hook codex` (or any other launch-only
  harness) exits with the `ErrHarnessHasNoHook` message and
  non-zero status.
- `examples/tmux/` does not exist; `examples/profiles/` contains
  `default.toml`, `dev.toml`, `kiro.toml`.
- README has "Harnesses", "Presets", and "Verify a profile"
  subsections.

## Dependencies / Sequencing
- Independent of #20.
- Hook-parser work items for individual harnesses (Codex hook,
  Kiro hook, etc.) follow this one and are self-contained — each is
  a single file in `internal/agenthook/` replacing the sentinel
  parser.
