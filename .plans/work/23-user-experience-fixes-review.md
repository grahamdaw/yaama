# 23 - User Experience Fixes · Review

Comparison of the shipped changes against
`.plans/work/23-user-experience-fixes.md` (as of the first PR cut —
this work item is intentionally open-ended and may grow further
items).

## Implementation summary

- **Item 1 — errors at the bottom.** `internal/tui/view.go` reorders
  the section list in `View()` so banner and toasts are appended
  *after* the footer rather than before the board. Order is now:
  top bar → search/status bar → empty state → board → footer →
  banner → toasts. Overlays (form / help / confirm) still render
  last.
- **Item 2 — one toast per line.** `renderToasts` joins toast lines
  with `\n` instead of ` · `.
- **Item 3 — session cards.** `renderColumns` now delegates each row
  to a new `renderSessionCard` helper which renders a small rounded
  border around the session, with the name + runtime badge on line 1
  (bold; focused colour when selected) and the task on line 2
  (faint, truncated). Border colour is `8` for unselected and `12`
  for the selected row (matching the focused-column border). A
  `truncate(s, width)` util sits next to it. Card width is
  `columnWidth-4` so the bordered card fits inside the column's
  inner content area without wrapping.
- **Item 4 — three columns.** `boardStatuses` in
  `internal/tui/columns.go` now contains only `idle`, `running`,
  `blocked`. The status-validation message in
  `internal/tui/form.go` was updated to match. Status cycling and
  the status picker pick up the trimmed list automatically through
  `statusKeys()`.

## Plan vs actual deviations

- **None of substance.** The spec entries above match what shipped
  one-for-one. Two small adjustments worth noting:
  - The existing badge-rendering test
    (`TestRenderColumnsShowsRuntimeBadges`) was bumped from width
    `120` to `200` so the substring assertion still finds
    `dead-agent [DEAD]` inside the narrower card content area. The
    test still meaningfully asserts that the badge reaches the
    rendered output.
  - `TestReverseStatusCycleShortcut` was updated to expect
    `idle → blocked` (instead of `idle → done`) and focus on column
    index `2` (instead of `4`) to reflect the trimmed 3-column
    board.

## Notes / follow-ups

- Agents that were previously persisted with status `review` or
  `done` will no longer appear on the board. Acceptable "for now"
  per the spec; reinstating those columns is a one-line change to
  `boardStatuses`.
- The card layout is currently two lines (name + task). Branch,
  profile, runtime state etc. remain in the detail panel below the
  board; a third card line was considered but not added.
- The work item is left **unticked** in `.plans/INDEX.md` because
  further UX fixes are expected to land on this branch before merge.
  Tick it (and append any new items to the spec) when the operator
  is satisfied that 23 is complete.
