# 03 - Board Layout, Navigation, and Modes

## Goal
Deliver a keyboard-first operator console with predictable focus behavior, global modes, and clear context visibility.

## Scope
- Kanban board rendering by status
- Detail panel and header counters
- Mode/state machine and key semantics

## Phase 1: Layout and Visual Structure
### Steps
1. Render status columns (`idle`, `running`, `blocked`, `review`, `done`) with per-column counts.
2. Add detail panel bound to current selection, including expanded metadata fields.
3. Implement header stats: total, running, blocked, dead.
4. Add footer key hints and help overlay entrypoint.

## Phase 2: Navigation Semantics
### Steps
1. Implement vertical/horizontal navigation with row-index preservation.
2. Clamp selection when moving into shorter columns.
3. Support empty-column focus on header row without panic/invalid cursor.
4. Add explicit focus styling for selected card and active column.

## Phase 3: Global Modes and Escape Rules
### Steps
1. Implement mode switching for `Normal`, `Search`, `Form`, `Confirm`, and `Help`.
2. Add `Esc` behavior hierarchy (close overlay/modal, then discard prompt for dirty forms).
3. Ensure mode-specific key handling is isolated and testable.
4. Add regression tests for focus/mode transitions.

## Definition of Done
- Operators can move through board content quickly without mouse usage.
- Focus and mode behavior are deterministic and match UX rules.
- Detail panel and summary counters stay in sync with selection/state.
