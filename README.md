# yaama

Terminal kanban board for managing AI coding agents running in tmux sessions.
This repository is currently in specification-first bootstrap mode.

## overview

The project target is a single Go binary backed by SQLite, with a Bubble Tea
TUI for observing agent status, attaching to tmux sessions, and managing agent
metadata. Implementation is organized as incremental work items under
`.plans/work/`.

## current status

- Specs and phased work items are available in `.plans/`.
- Runtime code is not scaffolded yet.
- Repository foundation files are in place so contributors can start work
  consistently.

## quick start

```bash
git clone <repo-url>
cd yaama
```

Read the project specs and start from the first implementation work item:

```bash
ls .plans
ls .plans/work
```

## implementation baseline (from specs)

The initial implementation direction is defined in
`.plans/000_INITIAL_SPEC.md` and `.plans/work/01-bootstrap-and-foundation.md`.
Expected baseline toolchain and commands:

- Go `1.23.x`
- `mise` (recommended) for local tool version pinning
- `make build`, `make run`, `make test`, `make lint` once scaffolding exists

## repository layout

- `.plans/`: product/technical specs and phased work items
- `.github/`: collaboration templates (PR template)
- Root docs: `README.md`, `CONTRIBUTING.md`, `AGENTS.md`

## contributing

Use short-lived branches from `main`, conventional commits, and the PR template
in `.github/PULL_REQUEST_TEMPLATE.md`. Full guidance is in
`CONTRIBUTING.md`.
