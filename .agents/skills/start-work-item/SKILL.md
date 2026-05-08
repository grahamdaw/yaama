---
name: start-work-item
description: Complete the next unfinished yaama work item end-to-end: pick the next item from .plans/INDEX.md, create a convention-compliant feature branch, implement in small logical commits, and open a PR using the repository PR template. Use when asked to continue roadmap implementation or complete the next work item.
disable-model-invocation: true
---

# Start Work Item

## Goal

Ship the next unfinished work item from `.plans/work/` in a clean, reviewable flow.

## Required Flow

1. **Identify the next work item**
   - Read `.plans/INDEX.md`.
   - Find the first unchecked entry in the work item checklist (`- [ ]`).
   - Open the corresponding `.plans/work/<item>.md` file.
   - Extract scope, constraints, and done criteria before coding.

2. **Create a feature branch**
   - Follow branch naming convention from `AGENTS.md`: `<type>/<optional-issue>-<short-kebab-name>`.
   - Default type for work item implementation: `feat`.
   - Example: `feat/06-tmux-attach-live-state-and-errors`.
   - Create and switch to the branch before code edits.

3. **Implement step by step**
   - Keep changes scoped to the active work item only.
   - Work in small vertical slices (one behavior at a time).
   - After each slice:
     - run the smallest relevant validation commands,
     - fix issues,
     - commit as one logical piece.
   - Use Conventional Commits (`feat:`, `fix:`, `test:`, `docs:`, `refactor:`, `chore:`).

4. **Use Kaiku Assets when needed**
   - If the work item needs assets (diagrams, supporting visuals, shared artifacts), use Kaiku Assets tools/workflows.
   - Keep generated assets versioned with the related implementation commit.
   - Prefer concise asset notes in commit body or PR notes when reviewer context is needed.

5. **Finish with full verification**
   - Run broader checks before opening the PR (at minimum `make test`; include others relevant to modified areas).
   - Confirm done criteria from the work item are satisfied.
   - Ensure no unrelated file churn is included.

6. **Open a PR with the repo template format**
   - Use `.github/PULL_REQUEST_TEMPLATE.md` sections:
     - `## Summary`
     - `## Context`
     - `## Testing`
     - `## Notes`
   - In `Testing`, include commands run and expected vs actual results.
   - Link any issue references (`Closes #`, `Fixes #`, `Resolves #`) when available.

## Commit Strategy

- Prefer multiple small commits over one large commit.
- Commit boundaries should match reviewer-understandable milestones:
  - data/model changes
  - core behavior
  - UI/UX wiring
  - tests
  - docs/plan tracking updates

## Completion Checklist

Copy and mark while executing:

```md
- [ ] Next unchecked work item identified from `.plans/INDEX.md`
- [ ] Feature branch created with naming convention
- [ ] Implementation completed in logical slices
- [ ] Each slice validated and committed with Conventional Commit message
- [ ] Kaiku Assets used when needed
- [ ] Full verification run and passing
- [ ] PR opened using repository template format
- [ ] Make sure the review of the plan is updated
```
