package tmux

import (
	"context"
	"reflect"
	"testing"
)

func TestBootstrapSessionBuildsDefaultAgentWindowFirst(t *testing.T) {
	originalAvailable := tmuxAvailableFn
	originalRunTmux := runTmuxFn
	originalSend := sendCommandToPaneFn
	t.Cleanup(func() {
		tmuxAvailableFn = originalAvailable
		runTmuxFn = originalRunTmux
		sendCommandToPaneFn = originalSend
	})

	var runCalls [][]string
	var sendCalls []struct {
		target  string
		command string
	}
	tmuxAvailableFn = func() bool { return true }
	runTmuxFn = func(_ context.Context, args ...string) error {
		runCalls = append(runCalls, append([]string(nil), args...))
		return nil
	}
	sendCommandToPaneFn = func(_ context.Context, paneTarget, command string) error {
		sendCalls = append(sendCalls, struct {
			target  string
			command string
		}{target: paneTarget, command: command})
		return nil
	}

	spec := BootstrapSpec{
		SessionName:   "kai-123-dev",
		WorkingDir:    "/tmp/worktree",
		AgentWindow:   "KAI 123/dev",
		StartupWindow: "ops",
		AgentCommand:  []string{"codex", "--model", "gpt-5.3-codex"},
		Windows: []BootstrapWindow{
			{Name: "ops", Focus: true},
		},
	}

	if err := BootstrapSession(context.Background(), spec); err != nil {
		t.Fatalf("BootstrapSession returned error: %v", err)
	}

	if len(runCalls) < 4 {
		t.Fatalf("expected tmux command calls, got %d", len(runCalls))
	}
	if got := runCalls[0]; !reflect.DeepEqual(got, []string{
		"new-session", "-d", "-s", "kai-123-dev", "-c", "/tmp/worktree",
		";",
		"set-option", "-t", "kai-123-dev", "destroy-unattached", "off",
	}) {
		t.Fatalf("unexpected first tmux call: %#v", got)
	}
	if got := runCalls[1]; !reflect.DeepEqual(got, []string{"rename-window", "-t", "kai-123-dev:0", "KAI-123-dev"}) {
		t.Fatalf("expected default window rename second, got %#v", got)
	}
	if got := runCalls[2]; !reflect.DeepEqual(got, []string{"new-window", "-d", "-t", "kai-123-dev", "-n", "ops", "-c", "/tmp/worktree"}) {
		t.Fatalf("expected additional profile window creation third, got %#v", got)
	}
	if got := runCalls[len(runCalls)-1]; !reflect.DeepEqual(got, []string{"select-window", "-t", "kai-123-dev:ops"}) {
		t.Fatalf("expected startup focus to select ops, got %#v", got)
	}

	if len(sendCalls) != 3 {
		t.Fatalf("expected three pane commands (default cd, ops cd, agent command), got %d", len(sendCalls))
	}
	if sendCalls[0].target != "kai-123-dev:0.0" {
		t.Fatalf("expected default pane init target first, got %q", sendCalls[0].target)
	}
	if sendCalls[1].target != "kai-123-dev:ops.0" {
		t.Fatalf("expected profile pane init target second, got %q", sendCalls[1].target)
	}
	if sendCalls[2].target != "kai-123-dev:0.0" {
		t.Fatalf("expected agent command to target default pane, got %q", sendCalls[2].target)
	}
}

func TestFocusedWindowNameMapsAgentAliasToDefaultWindow(t *testing.T) {
	spec := BootstrapSpec{
		SessionName:   "crew-42-dev",
		AgentWindow:   "crew-42-dev",
		StartupWindow: "agent",
		Windows: []BootstrapWindow{
			{Name: "ops"},
		},
	}

	windowName := focusedWindowName(spec)
	if windowName != "crew-42-dev" {
		t.Fatalf("expected startup_window=agent to map to default agent window, got %q", windowName)
	}
	if got := windowTarget(spec, windowName); got != "crew-42-dev:0" {
		t.Fatalf("expected default startup target to resolve to index 0, got %q", got)
	}
}

func TestFocusedWindowNamePrefersFocusedProfileWindow(t *testing.T) {
	spec := BootstrapSpec{
		AgentWindow:   "crew-77-dev",
		StartupWindow: "agent",
		Windows: []BootstrapWindow{
			{Name: "ops"},
			{Name: "review", Focus: true},
		},
	}

	if got := focusedWindowName(spec); got != "review" {
		t.Fatalf("expected focused profile window to win, got %q", got)
	}
}
