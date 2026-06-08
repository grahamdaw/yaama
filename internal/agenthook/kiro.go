package agenthook

func init() {
	Register(KiroHarness{})
}

// KiroHarness ships launch defaults for the Kiro IDE agent. Hook parsing
// is not yet implemented; ParseHook returns ErrHarnessHasNoHook.
type KiroHarness struct{}

func (KiroHarness) ID() string { return "kiro" }

func (KiroHarness) Defaults() AgentDefaults {
	return AgentDefaults{Command: "kiro"}
}

func (KiroHarness) ParseHook([]byte) (StatusUpdate, error) {
	return StatusUpdate{}, ErrHarnessHasNoHook
}
