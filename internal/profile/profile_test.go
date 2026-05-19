package profile

import (
	"os"
	"path/filepath"
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
repo = "/tmp/project"
worktree = true
layout_file = "tmux/default-layout.tmux"
setup = "scripts/init.sh"
teardown = "echo bye"

[[windows]]
name = "agent"
focus = true

  [[windows.panes]]
  run = "codex"
  agent = true
`
	if err := os.WriteFile(filepath.Join(profilesDir, "dev.toml"), []byte(profileContents), 0o600); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}

	cfg, err := Load("dev")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.DefaultBranch != defaultBranchName {
		t.Fatalf("expected default branch %q, got %q", defaultBranchName, cfg.DefaultBranch)
	}
	if !cfg.Worktree {
		t.Fatalf("expected worktree=true")
	}
	if want := filepath.Join(configRoot, "tmux", "default-layout.tmux"); cfg.LayoutFile != want {
		t.Fatalf("expected resolved layout file %q, got %q", want, cfg.LayoutFile)
	}
	if want := filepath.Join(configRoot, "scripts", "init.sh"); cfg.Setup != want {
		t.Fatalf("expected resolved setup %q, got %q", want, cfg.Setup)
	}
	if cfg.Teardown != "echo bye" {
		t.Fatalf("expected literal teardown command, got %q", cfg.Teardown)
	}
	if len(cfg.Windows) != 1 || cfg.Windows[0].Name != "agent" {
		t.Fatalf("expected single agent window, got %#v", cfg.Windows)
	}
	if len(cfg.Windows[0].Panes) != 1 || !cfg.Windows[0].Panes[0].Agent {
		t.Fatalf("expected agent pane, got %#v", cfg.Windows[0].Panes)
	}
}

func TestResolveRuntimeValuesUsesFallbackDir(t *testing.T) {
	cfg := Config{}

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
}

func TestResolveRuntimeValuesRequiresBranchInput(t *testing.T) {
	cfg := Config{}

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
	if cfg.DefaultBranch != defaultBranchName {
		t.Fatalf("expected default branch %q, got %q", defaultBranchName, cfg.DefaultBranch)
	}
	if len(cfg.Windows) == 0 {
		t.Fatalf("expected default profile to declare at least one window")
	}
	if cfg.Worktree {
		t.Fatalf("expected default profile to have worktree=false")
	}
}

func TestValidateRejectsMissingWindows(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configRoot := filepath.Join(os.Getenv("HOME"), ".config", "yaama")
	profilesDir := filepath.Join(configRoot, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profilesDir, "empty.toml"), []byte(`repo = "/tmp"`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := Load("empty")
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("expected missing-windows error, got %v", err)
	}
}

func TestValidateRejectsMultipleAgentPanes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configRoot := filepath.Join(os.Getenv("HOME"), ".config", "yaama")
	profilesDir := filepath.Join(configRoot, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	contents := `
[[windows]]
name = "agent"
  [[windows.panes]]
  agent = true
  [[windows.panes]]
  agent = true
`
	if err := os.WriteFile(filepath.Join(profilesDir, "twoagent.toml"), []byte(contents), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := Load("twoagent")
	if err == nil || !strings.Contains(err.Error(), "at most one pane") {
		t.Fatalf("expected at-most-one-agent error, got %v", err)
	}
}
