# 05 - Agent CRUD, Forms, and Guardrails

## Goal
Enable robust create/edit/delete flows with strong validation and safe destructive actions.

## Scope
- New/edit modal forms
- Validation and dirty-form handling
- Soft archive and explicit destructive confirms

## Phase 1: Form Framework
### Steps
1. Implement reusable form state and field components for create/edit actions.
2. Define required fields for generic create path and profile-driven path.
3. Add field validation for status, session uniqueness, and profile references.
4. Persist successful form submissions via typed DB queries.

## Phase 2: Edit and Dirty-State Protection
### Steps
1. Preload selected agent fields into edit form.
2. Track dirty state and trigger discard confirmation on `Esc`.
3. Show inline validation errors and prevent invalid submissions.
4. Refresh board and keep selection on updated card after save.

## Phase 3: Delete/Cleanup-State UX
### Steps
1. Implement confirm dialog for destructive actions.
2. Default deletion to soft archive (`cleanup_state = archived`) rather than hard remove.
3. Add explicit hard prune path with separate confirmation.
4. Ensure non-empty working directory deletion requires explicit force flow.

## Definition of Done
- Operators can create/edit/archive/prune records entirely from keyboard flows.
- Data integrity and safety prompts prevent accidental destructive actions.
- Form UX handles cancel/dirty/validation cases without losing operator context.
