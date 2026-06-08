package agenthook

func init() {
	Register(KiroCLIHarness{})
}

// KiroCLIHarness ships launch defaults for the Kiro CLI. Hook parsing is
// not yet implemented; ParseHook returns ErrHarnessHasNoHook.
type KiroCLIHarness struct{}

func (KiroCLIHarness) ID() string { return "kiro-cli" }

func (KiroCLIHarness) Defaults() AgentDefaults {
	return AgentDefaults{Command: "kiro-cli"}
}

func (KiroCLIHarness) ParseHook([]byte) (StatusUpdate, error) {
	return StatusUpdate{}, ErrHarnessHasNoHook
}
