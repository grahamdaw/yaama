package profile

import (
	"os"
	"path/filepath"
	"reflect"
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
command = "codex"
args = ["--model", "gpt-5.3-codex"]

[repo]
path = "/tmp/project"

[tmux]
layout_file = "tmux/default-layout.tmux"

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

	if cfg.Agent.PromptArg != defaultPromptArg {
		t.Fatalf("expected default prompt arg %q, got %q", defaultPromptArg, cfg.Agent.PromptArg)
	}
	if cfg.Agent.TicketArg != defaultTicketArg {
		t.Fatalf("expected default ticket arg %q, got %q", defaultTicketArg, cfg.Agent.TicketArg)
	}
	if cfg.Repo.DefaultBranch != defaultBranchName {
		t.Fatalf("expected default branch %q, got %q", defaultBranchName, cfg.Repo.DefaultBranch)
	}
	if want := filepath.Join(configRoot, "tmux", "default-layout.tmux"); cfg.Tmux.LayoutFile != want {
		t.Fatalf("expected resolved layout file %q, got %q", want, cfg.Tmux.LayoutFile)
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

func TestResolveRuntimeValuesUsesFallbackAndTaskArgs(t *testing.T) {
	cfg := Config{
		Agent: AgentConfig{
			Command:   "codex",
			Args:      []string{"--model", "gpt-5.3-codex"},
			TicketArg: "--ticket",
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
	if want := []string{"codex", "--model", "gpt-5.3-codex", "--ticket", "KAI-123"}; !reflect.DeepEqual(values.AgentCommand, want) {
		t.Fatalf("unexpected agent command: %#v", values.AgentCommand)
	}
}

func TestResolveRuntimeValuesRequiresBranchInput(t *testing.T) {
	cfg := Config{
		Agent: AgentConfig{
			Command:   "codex",
			Args:      []string{"--model", "gpt-5.3-codex"},
			TicketArg: "--ticket",
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
