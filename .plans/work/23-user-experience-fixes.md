# 23 - User Experience Fixes

## Goal
A grab-bag of small, operator-facing UX fixes discovered while using
Yaama in anger. Each item is scoped narrowly — no architectural
changes, no new subsystems. The spec is built up interactively as
issues surface; items are appended below in the order they are agreed.

## Branch Name
`feat/23-user-experience-fixes`

## Scope

Items are added incrementally. Each item should be small enough to
land as its own commit and tick off independently.

### Items

#### 1. Errors and error log render below the key hints
- **Problem:** The error banner and toasts currently render *above*
  the board, pushing it down on every error or status update. The
  operator's eye is on the board / footer, so error feedback would be
  better placed at the very bottom of the view where it doesn't
  reflow the board.
- **Fix:** In `View()` (`internal/tui/view.go:29-45`), move the
  `banner` and `toasts` sections so they are appended *after* the
  footer rather than before the board. Order becomes: topBar →
  search/status bar → emptyState → board → footer → banner → toasts.
  Overlays (form/help/confirm) still render last.
- **Files:** `internal/tui/view.go`.
- **Done when:** A toast or banner appears below the key hints line
  and the board no longer reflows vertically when one is shown.

#### 2. Each toast renders on its own line
- **Problem:** Multiple toasts are joined with ` · ` on a single line,
  which makes a burst of errors hard to read.
- **Fix:** Join toast lines with `\n` in `renderToasts`
  (`internal/tui/view.go`).
- **Files:** `internal/tui/view.go`.
- **Done when:** Two simultaneous toasts render on separate lines.

#### 3. Session rows render as cards
- **Problem:** Sessions in each column are plain text lines; the
  selected row is signalled only by a colour change which is easy to
  lose track of, and dense names/tasks bleed together.
- **Fix:** In `renderColumns` give each session its own small
  rounded-border card (name + runtime badge on line 1, task on line 2,
  faint). The selected card uses border colour `12`, others `8`. New
  helper `renderSessionCard` and a small `truncate` util live next to
  `focusedStyle` in `internal/tui/view.go`.
- **Files:** `internal/tui/view.go`, `internal/tui/view_test.go`
  (existing badge test bumped to width 200 so the substring assertion
  still holds against the narrower card content area).
- **Done when:** Each session renders inside a bordered box; the
  highlighted session's border is visibly the column-focus colour;
  the badge-rendering test still passes.

#### 4. Trim board columns to Idle / Running / Blocked
- **Problem:** Review and Done columns are not yet earning their
  screen space; three columns gives more horizontal room per card.
- **Fix:** Remove `review` and `done` entries from `boardStatuses`
  (`internal/tui/columns.go`) and update the form-validation message
  in `internal/tui/form.go` accordingly. Status cycling and the
  status picker pick up the trimmed list automatically because they
  iterate over `statusKeys()`. Existing agents persisted with status
  `review` or `done` simply don't appear on the board (acceptable
  "for now"; can be revisited if needed).
- **Files:** `internal/tui/columns.go`, `internal/tui/form.go`,
  `internal/tui/update_test.go` (reverse-cycle test adjusted to
  expect `idle → blocked` on the trimmed 3-column board).
- **Done when:** Board renders three columns; `S` from idle cycles to
  blocked; full tui test suite passes.

## Non-goals
- Anything that requires a schema migration or new dependency.
- Refactors beyond what a given fix strictly requires.
- New features. This work item is fixes only.

## Done Criteria
- Every item in the Items list is implemented, reviewed, and ticked.
- `.plans/work/23-user-experience-fixes-review.md` summarises plan vs
  actual in the commit that ticks the box in `.plans/INDEX.md`.
