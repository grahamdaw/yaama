# 11 - Polish, Acceptance, and Release Readiness

## Goal
Finalize usability polish, verify acceptance criteria, and prepare v1 for reliable daily operator use.

## Scope
- UI polish and help surfaces
- end-to-end acceptance verification
- CI/build/distribution readiness

## Phase 1: UX Polish
### Steps
1. Finalize badges (`dead`, `stale`) and ensure visual consistency.
2. Implement/help-tune keymap overlay and footer hints.
3. Refine empty/error state copy to always include next actions.
4. Tune toast durations and non-blocking behavior for operational clarity.

## Phase 2: Acceptance Validation
### Steps
1. Convert v1 acceptance criteria into executable test checklist.
2. Run manual keyboard-only flow validation across all critical user journeys.
3. Add targeted tests for dead-session handling, recovery, create, and cleanup.
4. Validate first-run startup on fresh machine state (no preexisting DB).

## Phase 3: Release Readiness
### Steps
1. Confirm Makefile and CI pass on supported platforms.
2. Validate cross-compilation artifacts for macOS/Linux targets.
3. Document operator runbook and troubleshooting notes in `README.md`.
4. Freeze v1 scope and capture post-v1 candidates (auto-register, richer activity timeline, adapters).

## Definition of Done
- Core operator workflows are stable and keyboard-complete.
- Acceptance criteria pass with clear evidence.
- Binary build and CI paths are reliable for ongoing iteration.
