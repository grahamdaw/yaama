# 10 - CLI Status Parity

## Goal
Ensure `yaama status` provides reliable agent self-reporting from within tmux and stays behaviorally aligned with board UX.

## Scope
- status command resolution by active tmux session
- task/activity/branch update flags
- clear failure behavior outside tmux or without matching record

## Phase 1: Command and Argument Contract
### Steps
1. Implement `yaama status <status>` with accepted status validation.
2. Add flags: `--task`, `--activity`, and optional `--branch`.
3. Return non-zero exit and accepted-value hints for invalid status inputs.

## Phase 2: Session Resolution and Updates
### Steps
1. Resolve current session via tmux (`$TMUX` context and `display -p '#S'`).
2. Update row by `tmux_session` with status + optional metadata fields.
3. Update heartbeat timestamps during successful status writes.
4. Trigger board refresh behavior compatibility for near-real-time visibility.

## Phase 3: Missing-Record and Environment Cases
### Steps
1. Return clear error when command is run outside tmux.
2. Return clear error when no matching agent exists for current session.
3. Include recommendation for creating/registering agent in error output.
4. Add tests for in-tmux success and all failure modes.

## Definition of Done
- Agents can self-report status/task/activity reliably from active sessions.
- CLI failures are explicit and actionable.
- Board and CLI state transitions remain consistent and conflict-free.
