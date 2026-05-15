# Project Overview

`yaama` is a terminal-first agent operations board planned as a single Go
binary with a Bubble Tea UI and SQLite persistence. The repository is currently
spec-driven: implementation should follow the phased work items in `.plans/`.
Track remaining work in `.plans/INDEX.md`, which is the checklist of work
items to be done and completed.

**Important! ALWAYS maintain this file with any changes.**

## Repository Structure

- `.plans/`: source of truth for product behavior, architecture, and work item
  sequencing.
- `.plans/INDEX.md`: canonical tracker for work items to be done/completed.
- `.plans/work/`: ordered implementation tasks; each file defines goal, scope,
  and done criteria.
- `.agents/skills/`: project-specific agent skills for repeatable delivery
  workflows.
- `cmd/board/`: board binary entrypoint.
- `internal/`: app packages (`tui`, `startup`, `config`, `db`, `agent`, `tmux`).
- `.github/`: collaboration automation and templates, including PR template.
- Root docs (`README.md`, `CONTRIBUTING.md`, `AGENTS.md`): contributor and
  agent operating guidance.

## Build & Development Commands

Current repository phase:

```bash
ls .plans
ls .plans/work
```

Baseline commands after work item `01` scaffolding:

```bash
make build
make run
make test
make vet
make lint
```

## Code Style & Conventions

- Respect the architecture and behavior in `.plans/000_INITIAL_SPEC.md` and
  `.plans/001_SPEC_ITERATION_UX.md`.
- Keep changes scoped to the active work item; avoid speculative refactors.
- Branch naming: `<type>/<optional-issue>-<short-kebab-name>`.
- Commit style: Conventional Commits (`feat:`, `fix:`, `docs:`, `refactor:`,
  `test:`, `chore:`).

## Architecture Notes

```text
tmux sessions <-> board TUI (Bubble Tea) <-> SQLite state
                    ^
                    |
                operator + agent CLI updates
```

The board reconciles declared state (SQLite) with live tmux session reality,
and supports operator-driven lifecycle updates through keyboard-first flows.
For profile-backed create, tmux bootstrapping always creates the default agent
window first (window index `0`, named from the agent/session), then applies
profile-defined tmux windows as additional windows.

## Testing Strategy

- For docs-only changes: verify cross-file consistency and command accuracy.
- For implementation changes: run the smallest relevant check set first, then
  full project checks.
- Do not claim success without command output evidence when checks exist.

## Security & Compliance

- Never commit secrets or machine-local credentials.
- Keep local runtime artifacts (SQLite DBs, logs, tmp files) out of git.
- Avoid destructive git operations unless explicitly requested by a human.

## Logging Conventions

- Action-path code (tmux bootstrap, recovery, cleanup, profile load,
  refresh failures, DB retries) emits a single `slog` line at `info` on
  the happy path and `error` on failure. Use the logger threaded
  through `startup.State` / `tmux.BootstrapSpec.Logger` / the TUI model.
- Log file lives at `$YAAMA_LOG_FILE`, falling back to
  `$XDG_STATE_HOME/yaama/yaama.log`, then
  `~/.local/state/yaama/yaama.log`. Level is `YAAMA_LOG_LEVEL`. The
  `L` key on the board toasts the resolved path.
- Truncate captured stderr / `last_error`-style strings with
  `logging.Truncate(s, 512)` before logging. Do not log script bodies or
  secrets.

## Agent Guardrails

- Do not rewrite work item intent; implement within the active item's scope.
- Do not silently skip required validation; call out blockers and gaps.
- Do not revert unrelated user changes in a dirty working tree.
- Prefer incremental, reviewable edits over broad repo-wide churn.
- If conventions conflict, follow `.plans/` plus root docs and report mismatch.

## Extensibility Hooks

- Work items can introduce new commands and structure incrementally.
- Planned configuration roots include `~/.config/yaama/` for profiles/scripts.
- Planned optional integration points include richer native git-worktree lifecycle automation.
