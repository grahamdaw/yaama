package profile

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadResolvesRelativePathsAndDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configRoot := filepath.Join(os.Getenv("HOME"), ".config", "yaama")
	profilesDir := filepath.Join(configRoot, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}

	profileContents := `
[agent]
harness = "codex"
command = "codex"
args = ["--model", "gpt-5.3-codex"]

[repo]
path = "/tmp/project"

[tmux]
startup_window = "agent"

[scripts]
before_start = ["scripts/init.sh", "echo ready"]
after_start = ["./scripts/after.sh"]
`
	if err := os.WriteFile(filepath.Join(profilesDir, "dev.toml"), []byte(profileContents), 0o600); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}

	cfg, err := Load("dev")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Repo.DefaultBranch != defaultBranchName {
		t.Fatalf("expected default branch %q, got %q", defaultBranchName, cfg.Repo.DefaultBranch)
	}
	if want := []string{
		filepath.Join(configRoot, "scripts", "init.sh"),
		"echo ready",
	}; !reflect.DeepEqual(cfg.Scripts.BeforeStart, want) {
		t.Fatalf("unexpected before_start values: %#v", cfg.Scripts.BeforeStart)
	}
	if want := []string{filepath.Join(configRoot, "scripts", "after.sh")}; !reflect.DeepEqual(cfg.Scripts.AfterStart, want) {
		t.Fatalf("unexpected after_start values: %#v", cfg.Scripts.AfterStart)
	}
}

func TestResolveRuntimeValuesUsesFallbackDir(t *testing.T) {
	cfg := Config{
		Agent: AgentConfig{
			Command: "codex",
			Args:    []string{"--model", "gpt-5.3-codex"},
		},
		Repo: RepoConfig{
			DefaultBranch: "main",
		},
	}

	values, err := ResolveRuntimeValues(cfg, "/tmp/workspace", "KAI-123", "feat/kai-123")
	if err != nil {
		t.Fatalf("ResolveRuntimeValues returned error: %v", err)
	}
	if values.WorkingDir != "/tmp/workspace" {
		t.Fatalf("expected working dir /tmp/workspace, got %q", values.WorkingDir)
	}
	if values.Branch != "feat/kai-123" {
		t.Fatalf("expected branch feat/kai-123, got %q", values.Branch)
	}
	if want := []string{"codex", "--model", "gpt-5.3-codex"}; !reflect.DeepEqual(values.AgentCommand, want) {
		t.Fatalf("unexpected agent command: %#v", values.AgentCommand)
	}
}

func TestLoadRejectsLegacyPromptAndTicketArgs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configRoot := filepath.Join(os.Getenv("HOME"), ".config", "yaama")
	profilesDir := filepath.Join(configRoot, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}

	const profileContents = `
[agent]
harness = "codex"
command = "codex"
prompt_arg = "--prompt"
ticket_arg = "--ticket"

[repo]
path = "/tmp/project"

[tmux]
startup_window = "agent"
`
	if err := os.WriteFile(filepath.Join(profilesDir, "legacy.toml"), []byte(profileContents), 0o600); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}

	_, err := Load("legacy")
	if err == nil {
		t.Fatalf("expected error for legacy prompt/ticket args")
	}
	if got := err.Error(); got == "" {
		t.Fatalf("expected non-empty error")
	}
	if !strings.Contains(err.Error(), "prompt_arg is no longer supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsLayoutFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configRoot := filepath.Join(os.Getenv("HOME"), ".config", "yaama")
	profilesDir := filepath.Join(configRoot, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}

	const profileContents = `
[agent]
harness = "codex"
command = "codex"

[repo]
path = "/tmp/project"

[tmux]
layout_file = "tmux/dev-layout.tmux"
`
	if err := os.WriteFile(filepath.Join(profilesDir, "layout.toml"), []byte(profileContents), 0o600); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}

	_, err := Load("layout")
	if err == nil {
		t.Fatalf("expected error for layout_file")
	}
	if !strings.Contains(err.Error(), "layout_file is no longer supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveRuntimeValuesRequiresBranchInput(t *testing.T) {
	cfg := Config{
		Agent: AgentConfig{
			Command: "codex",
			Args:    []string{"--model", "gpt-5.3-codex"},
		},
		Repo: RepoConfig{
			DefaultBranch: "main",
		},
	}

	_, err := ResolveRuntimeValues(cfg, "/tmp/workspace", "KAI-123", "")
	if err == nil {
		t.Fatalf("expected error for missing branch input")
	}
}

func TestLoadRequiresHarness(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	profilesDir := filepath.Join(os.Getenv("HOME"), ".config", "yaama", "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}

	const noHarness = `
[agent]
command = "codex"

[repo]
path = "/tmp/project"

[tmux]
startup_window = "agent"
`
	if err := os.WriteFile(filepath.Join(profilesDir, "no-harness.toml"), []byte(noHarness), 0o600); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}
	_, err := Load("no-harness")
	if err == nil || !strings.Contains(err.Error(), "harness is required") {
		t.Fatalf("expected missing-harness error, got %v", err)
	}

	const unknownHarness = `
[agent]
harness = "made-up"
command = "codex"

[repo]
path = "/tmp/project"

[tmux]
startup_window = "agent"
`
	if err := os.WriteFile(filepath.Join(profilesDir, "unknown.toml"), []byte(unknownHarness), 0o600); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}
	_, err = Load("unknown")
	if err == nil || !strings.Contains(err.Error(), "is not registered") {
		t.Fatalf("expected unknown-harness error, got %v", err)
	}
}

func TestLoadAppliesHarnessDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	profilesDir := filepath.Join(os.Getenv("HOME"), ".config", "yaama", "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}

	const profileContents = `
[agent]
harness = "claude-code"

[repo]
path = "/tmp/project"

[tmux]
startup_window = "agent"
`
	if err := os.WriteFile(filepath.Join(profilesDir, "claude.toml"), []byte(profileContents), 0o600); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}
	cfg, err := Load("claude")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Agent.Command != "claude" {
		t.Fatalf("expected harness default command 'claude', got %q", cfg.Agent.Command)
	}
}

func TestLoadOperatorValueOverridesHarnessDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	profilesDir := filepath.Join(os.Getenv("HOME"), ".config", "yaama", "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}

	const profileContents = `
[agent]
harness = "claude-code"
command = "/usr/local/bin/my-claude"
args = ["--flag"]

[agent.env]
ANTHROPIC_LOG = "debug"

[repo]
path = "/tmp/project"

[tmux]
startup_window = "agent"
`
	if err := os.WriteFile(filepath.Join(profilesDir, "override.toml"), []byte(profileContents), 0o600); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}
	cfg, err := Load("override")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Agent.Command != "/usr/local/bin/my-claude" {
		t.Fatalf("expected operator command to win, got %q", cfg.Agent.Command)
	}
	if !reflect.DeepEqual(cfg.Agent.Args, []string{"--flag"}) {
		t.Fatalf("expected operator args, got %v", cfg.Agent.Args)
	}
	if cfg.Agent.Env["ANTHROPIC_LOG"] != "debug" {
		t.Fatalf("expected operator env, got %v", cfg.Agent.Env)
	}
}

func TestLoadRejectsPresetAndWindowsTogether(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	profilesDir := filepath.Join(os.Getenv("HOME"), ".config", "yaama", "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}
	const profileContents = `
[agent]
harness = "claude-code"

[repo]
path = "/tmp/project"

[tmux]
preset = "solo"

[[tmux.windows]]
name = "ops"
`
	if err := os.WriteFile(filepath.Join(profilesDir, "both.toml"), []byte(profileContents), 0o600); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}
	_, err := Load("both")
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutual-exclusion error, got %v", err)
	}
}

func TestLoadRejectsUnknownPreset(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	profilesDir := filepath.Join(os.Getenv("HOME"), ".config", "yaama", "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}
	const profileContents = `
[agent]
harness = "claude-code"

[repo]
path = "/tmp/project"

[tmux]
preset = "made-up"
`
	if err := os.WriteFile(filepath.Join(profilesDir, "bad-preset.toml"), []byte(profileContents), 0o600); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}
	_, err := Load("bad-preset")
	if err == nil || !strings.Contains(err.Error(), "not in the catalog") {
		t.Fatalf("expected unknown preset error, got %v", err)
	}
}

func TestLoadExpandsPresetIntoWindows(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	profilesDir := filepath.Join(os.Getenv("HOME"), ".config", "yaama", "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}
	const profileContents = `
[agent]
harness = "claude-code"

[repo]
path = "/tmp/project"

[tmux]
preset = "agent+git+tests"
`
	if err := os.WriteFile(filepath.Join(profilesDir, "preset.toml"), []byte(profileContents), 0o600); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}
	cfg, err := Load("preset")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(cfg.Tmux.Windows) != 1 || cfg.Tmux.Windows[0].Name != "ops" {
		t.Fatalf("expected preset to expand into ops window, got %+v", cfg.Tmux.Windows)
	}
	if cfg.Tmux.Windows[0].Layout != "even-vertical" {
		t.Fatalf("expected layout even-vertical, got %q", cfg.Tmux.Windows[0].Layout)
	}
}

func TestLoadDefaultProfileWithoutFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, err := Load("default")
	if err != nil {
		t.Fatalf("Load(default) returned error: %v", err)
	}
	if cfg.Agent.Command == "" {
		t.Fatalf("expected default profile to include an agent command")
	}
	if cfg.Repo.DefaultBranch != defaultBranchName {
		t.Fatalf("expected default branch %q, got %q", defaultBranchName, cfg.Repo.DefaultBranch)
	}
}
