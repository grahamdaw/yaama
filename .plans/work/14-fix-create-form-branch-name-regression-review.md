# 14 Fix Create Form Branch Name Regression - Review

## Planned Scope
- Reproduce the create-form branch regression where branch name was missing.
- Ensure branch is visible/editable in profile-backed create wizard.
- Preserve branch validation and persistence semantics.
- Add regression coverage for branch-step behavior.

## Actual Implementation
- Identified root cause in `internal/tui/view.go`: create wizard rendered only `profile` and `task` stages even though `branch` remained required in state/validation.
- Updated create wizard overlay rendering to include explicit third stage:
  - `3) Branch: <value>`
  - branch field error rendering (`m.form.errors["branch"]`).
- Updated step guidance copy to match real flow:
  - Step 1 profile select -> Enter continue
  - Step 2 task entry -> Enter continue
  - Step 3 branch entry -> Enter create
- Added regression test `TestRenderCreateWizardIncludesBranchStep` in `internal/tui/view_test.go` to lock branch-step visibility and guidance copy.

## Plan vs Actual Notes
- Scope matched planned fix intent exactly.
- No create-flow runtime or persistence logic changes were required; regression was isolated to UI rendering and instructional copy.

## Validation Evidence
- `go test ./internal/tui` passed.
- `make test` passed.
