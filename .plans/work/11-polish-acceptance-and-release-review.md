# 11 Polish, Acceptance, and Release Readiness - Review

## Planned Scope
- UI polish and help surfaces.
- End-to-end acceptance verification.
- CI/build/distribution readiness.

## Actual Implementation
- Normalized runtime emphasis in board and detail panel with consistent
  `[DEAD]` / `[STALE]` labels and explicit `LIVE|DEAD|STALE` detail state text.
- Tightened footer command hints to reduce ambiguity around status picker,
  cleanup modes, and `Esc` behavior.
- Added view-focused regression tests in `internal/tui/view_test.go` for:
  - runtime badge rendering in board columns,
  - actionable empty-state next steps,
  - dead-session detail recovery guidance.
- Added fresh-start acceptance coverage in `internal/startup/startup_test.go`
  to validate startup DB creation/migration path and first-run notices without
  manual migration steps.
- Added release-readiness cross-build validation:
  - new `make release-check` target building macOS arm64 + Linux amd64 binaries,
  - CI workflow now executes `make release-check` after test/vet.
- Expanded `README.md` with operator runbook, troubleshooting, and v1 scope
  freeze plus post-v1 candidate list.

## Plan vs Actual Notes
- Manual keyboard-only flow validation is represented through executable tests
  plus existing interaction tests under `internal/tui/update_test.go`; no
  additional ad-hoc script was introduced.
- Cross-platform readiness is validated by compile checks rather than packaging
  automation, which keeps scope aligned to release-readiness (not release
  publishing).

## Acceptance Checklist (Executable/Verifiable)
- [x] App starts on fresh machine state without manual migration (`TestBootstrapInitializesFreshDBWithoutManualMigrations`).
- [x] Keyboard-only create/edit/filter/attach/status flows covered (existing `internal/tui/update_test.go` suite).
- [x] Dead sessions remain visible and actionable (`TestRenderColumnsShowsRuntimeBadges`, `TestRenderDetailPanelShowsDeadRecoveryActions`).
- [x] Existing working-dir recovery guidance remains explicit in details and recovery flow tests.
- [x] Profile-driven create path already covered by existing update/profile tests.
- [x] Cleanup lifecycle behavior remains covered by existing cleanup/retry tests.
- [x] `board status` reliability and failure behavior covered by `cmd/board/status_test.go`.
- [x] Empty/error surfaces include next actions (`TestRenderEmptyStateIncludesNextActions` and runtime banner/toast coverage).
- [x] CI/build path includes cross-platform compile checks (`make release-check` in CI).

## Validation Evidence
- `go test ./internal/startup ./internal/tui` passed.
- `make test` passed.
- `make release-check` passed.
