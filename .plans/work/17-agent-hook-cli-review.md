# 17 - Agent Hook CLI — Review

## Plan vs Actual

- **Phase 1 — Normalized event model & registry** (`internal/agenthook/agenthook.go`):
  - `Event` carries `EventName`, `Status`, `LastActivity`, `LastError` as
    `Optional{Value, Set}` so unset fields are distinguishable from
    explicitly-empty values (lets the writer preserve existing columns).
  - `Parser` interface (`Name()`, `Parse([]byte) (Event, error)`) with a
    package-level `registry`. `Register` panics on duplicate or empty names;
    `Lookup` and `Names` are case-insensitive / sorted respectively.

- **Phase 2 — Claude Code parser** (`internal/agenthook/claudecode.go`):
  - Self-registers via `init()` as `claude-code`.
  - Decodes only the small payload subset needed (`hook_event_name`,
    `tool_name`, `message`, `prompt`, `reason`, `tool_response.error`);
    unknown JSON fields are ignored so future Claude Code additions don't
    break older yaama binaries.
  - Status mapping matches the plan: `SessionStart` / `UserPromptSubmit` /
    `PreToolUse` → `running`; `Notification` → `blocked`; `Stop` /
    `SessionEnd` → `idle`; `PostToolUse` records `LastError` when
    `tool_response.error` is present and otherwise just updates activity.
    `SubagentStop`, `PreCompact`, and unknown well-formed events leave
    status untouched and just record an activity line.
  - Strings are normalized via `singleLine` and truncated (200 chars for
    activity, 500 for `last_error`).

- **Phase 3 — `yaama hook` subcommand** (`cmd/yaama/hook.go`,
  `cmd/yaama/main.go`):
  - `runHookCommand` parses args (`<agent>` positional + optional
    `--db`), reads the payload from stdin, looks up the parser, and
    dispatches the event.
  - Tmux session resolution reuses `tmux.CurrentSession` (the same helper
    `status` uses); errors surface actionable messages for outside-tmux,
    `tmux.ErrTmuxUnavailable`, and missing-agent (`missingAgentError`)
    cases.
  - Updates flow through the existing
    `UpdateAgentStatusByTmuxSession` query — the `agents` schema is
    unchanged. Status is preserved when the event leaves it unset
    (re-passing `existing.Status`), and `Task` / `Branch` are sent as
    null `sql.NullString{}` so the existing COALESCE-style query
    preserves them.
  - Dispatch wired in `cmd/yaama/main.go` alongside `status`.

- **Phase 4 — Tests & docs**:
  - Parser unit tests (`internal/agenthook/claudecode_test.go`) cover
    known events, unknown event, empty/malformed payloads, missing
    `hook_event_name`, and case-insensitive registry lookup.
  - Command-level tests (`cmd/yaama/hook_test.go`) cover stdin parsing,
    outside-tmux, missing-agent, status-applied, and status-preserved
    paths.
  - README operator runbook documents `yaama hook claude-code` plus a
    `~/.claude/settings.json` snippet wiring `PreToolUse`, `PostToolUse`,
    `Notification`, and `Stop` to the binary, and notes that adding a
    new agent only requires a new parser file under `internal/agenthook/`.

## Deviations / Notes

- The implementation landed on `main` as a single squash-merged commit
  (`8a487fa — feat: add board hook CLI for agent state updates from
  hooks`) prior to the `board → yaama` rename in work item 16; the
  rename PR (#20) re-pointed the files to `cmd/yaama/...` and updated
  the README/help-text strings to use `yaama hook ...`.
- This review file and the INDEX tick are the only artifacts added by
  the work-item-17 PR itself — the code, tests, and docs were already
  in place on `main`.
- `Optional` (rather than `*string`) is used for tri-state fields. This
  matches the existing `optionalString` pattern from the `status`
  command and keeps JSON-style "absent vs explicitly empty" semantics
  visible at call sites without relying on pointer aliasing.

## Verification

- `make vet` → clean.
- `make test` → all packages pass, including `internal/agenthook` and
  `cmd/yaama` hook tests.
