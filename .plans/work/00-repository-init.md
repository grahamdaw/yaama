# 00 - Repository Initialization

## Goal
Bootstrap the repository with core contributor and agent guidance files so all future work follows consistent standards and automation conventions.

## Scope
- Create foundational docs: `AGENTS.md`, `CONTRIBUTING.md`, `README.md`
- Create baseline `.gitignore` for Go/TUI/SQLite workflow
- Use available project assets and agent skills to produce aligned configurations/content

## Phase 1: Context and Asset Discovery
### Steps
1. Scan repository assets (existing scripts, folder structure, config files, tooling choices) to infer project conventions.
2. Identify available skills/rules that should inform agent behavior and contributor workflow.
3. Extract naming conventions, branch/PR expectations, test commands, and local setup requirements from current plans/specs.
4. Define document boundaries so each file has a clear purpose and minimal overlap.

## Phase 2: Author Foundation Files
### Steps
1. Create `README.md` with project intent, prerequisites, local setup, core commands, and high-level architecture.
2. Create `CONTRIBUTING.md` with contribution flow (branching, coding standards, tests/lint, commit/PR expectations, review checklist).
3. Create `AGENTS.md` with explicit instructions for coding agents (repo norms, execution guardrails, testing expectations, safe edit rules).
4. Create `.gitignore` covering Go build outputs, local DB files, temp artifacts, editor noise, and machine-local configuration.

## Phase 3: Validation and Alignment
### Steps
1. Verify all command references in docs are runnable and match Makefile/CLI targets.
2. Ensure no conflicting guidance exists between `AGENTS.md`, `CONTRIBUTING.md`, and `README.md`.
3. Confirm `.gitignore` does not exclude required source/config assets that must be versioned.
4. Run a final docs pass for clarity, onboarding readiness, and agent usability.

## Deliverables
- `AGENTS.md`
- `CONTRIBUTING.md`
- `README.md`
- `.gitignore`

## Definition of Done
- New contributor can clone the repo and follow `README.md` + `CONTRIBUTING.md` to run/build/test successfully.
- Agent behavior expectations are explicit in `AGENTS.md` and compatible with repository workflows.
- `.gitignore` prevents common local artifacts from polluting commits while preserving required project files.
