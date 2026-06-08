package agenthook

func init() {
	Register(VSCodeCopilotHarness{})
}

// VSCodeCopilotHarness ships launch defaults for the VS Code Copilot chat
// CLI bridge. Hook parsing is not yet implemented; ParseHook returns
// ErrHarnessHasNoHook.
type VSCodeCopilotHarness struct{}

func (VSCodeCopilotHarness) ID() string { return "vscode-copilot" }

func (VSCodeCopilotHarness) Defaults() AgentDefaults {
	return AgentDefaults{Command: "code", Args: []string{"--copilot"}}
}

func (VSCodeCopilotHarness) ParseHook([]byte) (StatusUpdate, error) {
	return StatusUpdate{}, ErrHarnessHasNoHook
}
