# 09 - Cleanup Lifecycle and Prune

## Goal
Implement explicit, reliable cleanup that tears down runtime resources safely and leaves auditable state.

## Scope
- ordered cleanup pipeline
- archive vs prune semantics
- resilient error handling across tmux/work-dir/scripts

## Phase 1: Cleanup Action Design
### Steps
1. Expose cleanup actions from selected item with clear archive/prune choices.
2. Keep archive as default non-destructive cleanup state transition.
3. Gate hard prune behind explicit confirmation prompts.

## Phase 2: Ordered Cleanup Pipeline
### Steps
1. Kill tmux session for selected item; stop flow if this fails.
2. Prune branch working directory via adapter when applicable.
3. Execute configured profile cleanup script hooks.
4. Mark resulting cleanup state (`archived` or `pruned`) and persist completion/error info.

## Phase 3: Error and Recovery Semantics
### Steps
1. Prevent final hard-delete transitions when work-dir prune fails.
2. Record script failures in `last_error` while preserving cleanup visibility.
3. Add operator-facing remediation guidance per failure stage.
4. Add integration tests for partial failures and idempotent retries.

## Definition of Done
- Cleanup runs in deterministic order with safe stop conditions.
- Archive/prune semantics are visible and reversible where intended.
- Partial failures are persisted and actionable rather than silent.
