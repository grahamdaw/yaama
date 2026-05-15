# 17 - Agent Hook CLI

## Goal
Provide a CLI subcommand that lets agent hooks (Claude Code first, extensible to other agents) push state changes into the yaama database — keyed by the current tmux session — without opening the TUI.

## Branch Name
`feat/17-agent-hook-cli`

## Scope
- `yaama hook <agent>` subcommand that reads a hook payload from stdin
- agent-specific parser registry under `internal/agenthook/` (claude-code shipped; others pluggable)
- normalized event model (`Status`, `LastActivity`, `LastError`) mapped to existing `agents` columns
- reuse tmux session resolution and DB update path from the `status` command
- README + example hook wiring for Claude Code
- depends on work item 16 (rename to `yaama`) so the documented command is the final one

## Phase 1: Normalized Event Model and Registry
### Steps
1. Define `agenthook.Event` with optional Status/LastActivity/LastError fields.
2. Define `agenthook.Parser` interface (`Name()`, `Parse([]byte) (Event, error)`).
3. Implement `Register`/`Lookup`/`Names` registry; panic on duplicate registration.

## Phase 2: Claude Code Parser
### Steps
1. Decode `hook_event_name` plus the small subset of fields used per event.
2. Map known events to status transitions: SessionStart/UserPromptSubmit/PreToolUse → running; Notification → blocked; Stop/SessionEnd → idle; PostToolUse → record tool error if present.
3. Preserve status for events with no clear transition (e.g. SubagentStop, PreCompact, unknown).
4. Truncate long strings (prompts, error messages) before writing.

## Phase 3: `yaama hook` Subcommand
### Steps
1. Add `cmd/yaama/hook.go` reading stdin, looking up the parser, parsing the event.
2. Resolve current tmux session (reuse the helper used by `status`); fail with actionable errors for outside-tmux, missing-agent, and tmux-unavailable cases.
3. Apply the event via `UpdateAgentStatusByTmuxSession`, preserving the existing status when the event leaves it unset.
4. Wire dispatch in `cmd/yaama/main.go`.

## Phase 4: Tests and Docs
### Steps
1. Parser unit tests covering known events, unknown event, malformed/empty payload, case-insensitive registry lookup.
2. Command-level tests covering stdin parsing, outside-tmux, missing-agent, status-applied, status-preserved.
3. Update README operator runbook with a `yaama hook claude-code` example and a `~/.claude/settings.json` hooks snippet.
4. Add a `17-agent-hook-cli-review.md` summarizing plan vs actual when complete.

## Definition of Done
- `yaama hook claude-code` reads a Claude Code hook payload on stdin and updates the agent row keyed by the current tmux session.
- Adding a new agent only requires a new parser file with an `init()` `Register(...)`; no changes to `cmd/yaama/`.
- Existing `agents` schema is unchanged.
- Tests, vet, and release-check pass.
