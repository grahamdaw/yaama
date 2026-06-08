package agenthook

func init() {
	Register(CopilotHarness{})
}

// CopilotHarness ships launch defaults for the GitHub Copilot CLI. Hook
// parsing is not yet implemented; ParseHook returns ErrHarnessHasNoHook.
type CopilotHarness struct{}

func (CopilotHarness) ID() string { return "copilot" }

func (CopilotHarness) Defaults() AgentDefaults {
	return AgentDefaults{Command: "gh", Args: []string{"copilot"}}
}

func (CopilotHarness) ParseHook([]byte) (StatusUpdate, error) {
	return StatusUpdate{}, ErrHarnessHasNoHook
}
