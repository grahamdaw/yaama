# 16 - Rename `board` Binary to `yaama`

## Goal
Rename the operator-facing CLI from `board` to `yaama` so the command users invoke matches the product/repo name. The hooks CLI (work item 17) lands on top of the renamed binary so no downstream renames are needed afterwards.

## Branch Name
`feat/16-rename-board-binary-to-yaama`

## Scope
- rename `cmd/board/` package directory and binary output to `yaama`
- update Makefile, README, examples, and any internal references that hard-code `board` or `bin/board`
- preserve existing subcommand contracts (`status`, future `hook`) so behavior is identical apart from the executable name
- update tests/docs that reference the old binary path
- explicitly out of scope: backwards-compatibility `board` shim or symlink (clean rename)

## Phase 1: Inventory References
### Steps
1. Grep the repo for `board`, `bin/board`, and `cmd/board` to enumerate every site that must move or change.
2. Confirm no external scripts in `examples/` rely on the literal `board` name without override.
3. Note CI/release-check targets that produce or test `bin/board` artifacts.

## Phase 2: Rename Package and Build Output
### Steps
1. Move `cmd/board/` to `cmd/yaama/` (preserving file contents) and keep `package main`.
2. Update `Makefile` `build`/`run`/`release-check` targets to produce `bin/yaama`.
3. Update `go.mod`/import paths only if any test or tool references `cmd/board` directly.

## Phase 3: Update Docs and Examples
### Steps
1. Update `README.md`: build instructions, operator runbook, troubleshooting, and `board status` examples to use `yaama status`.
2. Update `examples/WALKTHROUGH.md` and any profile example references.
3. Update `AGENTS.md` / `CONTRIBUTING.md` if either mentions `board`.
4. Sweep `.plans/work/*.md` (including completed items, in-flight item 15, and follow-up item 17) for any references to `board`, `bin/board`, or `board status`/`board hook` invocations and rewrite them to `yaama …`. Note: do not rewrite historical narrative in `*-review.md` files where the original wording is part of the record — only update forward-looking instructions, command examples, and definition-of-done checklists.

## Phase 4: Validation
### Steps
1. `make build` produces `bin/yaama`; `make run` launches the TUI.
2. `make test` and `make vet` pass; `make release-check` succeeds for macOS/Linux artifacts.
3. `yaama status running --task "..." --activity "..."` works from inside a tmux session bound to an existing agent record.
4. Add a `16-rename-board-binary-to-yaama-review.md` summarizing plan vs actual when complete.

## Definition of Done
- The built binary is `bin/yaama`; `bin/board` is no longer produced.
- All in-repo docs, examples, and Makefile targets reference `yaama` (not `board`).
- Existing CLI subcommand behavior and exit codes are unchanged.
- Tests, vet, and release-check pass on the renamed layout.
