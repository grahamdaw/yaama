# 14 - Fix Create Form Branch Name Regression

## Goal
Restore reliable branch-name capture in the create flow so profile-backed session creation cannot proceed with an empty, missing, or silently dropped branch value.

## Branch Name
`fix/14-create-form-branch-name-regression`

## Scope
- reproduce and isolate the current create-form regression where branch name is missing
- ensure the create flow always renders and persists the branch field for profile-backed sessions
- harden validation and state transitions so branch input is never lost between steps
- add regression coverage for keyboard-first create flow (`profile -> task -> branch`)

## Phase 1: Reproduce and Isolate
### Steps
1. Reproduce the bug in the current create flow and document exact trigger path.
2. Identify whether regression source is in form state, update routing, or runtime value resolution.
3. Capture expected vs actual behavior for branch field rendering, editing, and submit.

## Phase 2: Implement Fix
### Steps
1. Repair create-form state wiring so branch input is always initialized for profile-backed create.
2. Ensure branch value survives navigation and mode transitions until submit/cancel.
3. Preserve existing validation semantics for safe branch tokens and actionable errors.
4. Confirm runtime metadata persistence still records the branch used for worktree/session setup.

## Phase 3: Regression Tests and UX Guardrails
### Steps
1. Add/update tests in `internal/tui` for create flow branch-step behavior.
2. Add/update resolver tests where branch is required for profile-backed runtime values.
3. Verify no regressions in non-profile flows or archive/edit forms.

## Phase 4: Validation and Docs
### Steps
1. Run focused tests for changed packages first, then broader project checks.
2. Update user/operator docs only if create-flow behavior or wording changed.
3. Add a review file summarizing plan vs actual when the work item is completed.

## Definition of Done
- Branch field is visible and editable in the profile-backed create flow.
- Submitting profile-backed create without branch still fails with actionable validation.
- Branch value is persisted and available to downstream runtime/worktree wiring.
- Automated tests cover the regression path and pass in CI/local test runs.
