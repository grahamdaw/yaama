# 07 - Dead Session Recovery Flow

## Goal
Allow operators to recover dead agents quickly by recreating tmux sessions in the existing working directory.

## Scope
- dead-session action menu
- tmux recreation with working directory reuse
- failure-safe recovery messaging

## Phase 1: Recovery Interaction Model
### Steps
1. On dead selected card, present recovery options (`r` recreate, `e` edit, `d` prune/archive).
2. Keep default attach behavior for live sessions unchanged.
3. Add detail panel cues explaining why item is considered dead.

## Phase 2: Recreate Session Execution
### Steps
1. Validate `working_dir` is present and exists on disk.
2. Create detached tmux session bound to `working_dir`.
3. Update heartbeat/activity metadata and transition to active state.
4. Attach/switch operator into recreated session after successful creation.

## Phase 3: Recovery Failure Paths
### Steps
1. If `working_dir` missing, show actionable error and stop without side effects.
2. If tmux creation fails, preserve record state and capture `last_error`.
3. Offer direct path to edit mapping after failure.
4. Add tests for missing path, invalid path, and recreate success.

## Definition of Done
- Dead cards expose a clear and safe recovery path.
- Recreated sessions reuse existing working-directory bindings (no duplicates).
- Failures provide next actions instead of leaving operator in ambiguous state.
