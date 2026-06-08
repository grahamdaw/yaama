package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestProfile(t *testing.T, name, contents string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	profilesDir := filepath.Join(home, ".config", "yaama", "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatalf("mkdir profilesDir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profilesDir, name+".toml"), []byte(contents), 0o600); err != nil {
		t.Fatalf("write profile: %v", err)
	}
	return profilesDir
}

func TestProfileCheckPrintsPlanForValidProfile(t *testing.T) {
	writeTestProfile(t, "default", `
[agent]
harness = "claude-code"

[repo]
path = "/tmp/project"
default_branch = "main"

[tmux]
preset = "solo"
startup_window = "agent"
`)

	var stdout, stderr bytes.Buffer
	code := runProfileCheckCommand([]string{"default"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d (stderr=%s)", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"Profile: default",
		"Harness: claude-code",
		"Resolved:",
		"working_dir = /tmp/project",
		"branch      = main",
		"agent       = claude",
		"Git:",
		"git -C /tmp/project worktree add",
		"Tmux:",
		"tmux new-session -d -s <session>",
		"Hooks:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("plan output missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestProfileCheckReportsValidationErrors(t *testing.T) {
	writeTestProfile(t, "bad", `
[agent]
command = "codex"

[repo]
path = "/tmp/project"

[tmux]
startup_window = "agent"
`)

	var stdout, stderr bytes.Buffer
	code := runProfileCheckCommand([]string{"bad"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit for invalid profile; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "harness is required") {
		t.Fatalf("expected harness error on stderr, got %q", stderr.String())
	}
}

func TestProfileCheckRejectsMissingPositional(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runProfileCheckCommand(nil, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit when name is missing")
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage hint, got %q", stderr.String())
	}
}
