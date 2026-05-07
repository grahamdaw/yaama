# 04 - Search, Status Picker, and Operator Speed

## Goal
Improve operator throughput with fast filtering and safer status transitions.

## Scope
- Live search/filter flow
- Inline status picker + optional reverse cycle shortcut
- Keyboard shortcuts tuned for <=2 keypress core actions

## Phase 1: Search Mode
### Steps
1. Implement `/` to enter Search mode with focused input.
2. Apply filter live while typing across `name`, `task`, `branch`, and `tmux_session`.
3. Support `Enter` to keep filter active while returning to Normal mode.
4. Support `Esc` to clear filter and fully exit Search mode.

## Phase 2: Status Change UX
### Steps
1. Replace blind forward cycling on `s` with inline status picker.
2. Map keys `1..5` to configured status order and apply on `Enter`.
3. Keep `S` as optional backward quick cycle for power users.
4. Add visual confirmation/toast after transition and immediate DB refresh.

## Phase 3: Usability Hardening
### Steps
1. Validate state transitions from any mode do not corrupt selection/focus.
2. Ensure filtered view preserves expected keyboard navigation.
3. Add tests for search matching and status picker key handling.

## Definition of Done
- Search reliably narrows results in real time and exits predictably.
- Status transitions are explicit, low-error, and fast.
- Speed features materially reduce keystrokes for common operator loops.
