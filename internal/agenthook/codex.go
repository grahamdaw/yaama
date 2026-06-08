package agenthook

func init() {
	Register(CodexHarness{})
}

// CodexHarness ships launch defaults for OpenAI's Codex CLI. Hook parsing
// is not yet implemented; ParseHook returns ErrHarnessHasNoHook so
// yaama hook codex exits with a clean operator-facing message.
type CodexHarness struct{}

func (CodexHarness) ID() string { return "codex" }

func (CodexHarness) Defaults() AgentDefaults {
	return AgentDefaults{Command: "codex"}
}

func (CodexHarness) ParseHook([]byte) (StatusUpdate, error) {
	return StatusUpdate{}, ErrHarnessHasNoHook
}
