package tmux

import (
	"context"
	"reflect"
	"strings"
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

	if len(runCalls) < 6 {
		t.Fatalf("expected tmux command calls, got %d", len(runCalls))
	}
	if got := runCalls[0]; !reflect.DeepEqual(got, []string{
		"new-session", "-d", "-s", "kai-123-dev", "-c", "/tmp/worktree",
		";",
		"set-option", "-t", "kai-123-dev", "destroy-unattached", "off",
	}) {
		t.Fatalf("unexpected first tmux call: %#v", got)
	}
	if got := runCalls[1]; !reflect.DeepEqual(got, []string{"set-environment", "-t", "kai-123-dev", "YAAMA_TMUX_SESSION", "kai-123-dev"}) {
		t.Fatalf("expected session env injection second, got %#v", got)
	}
	if got := runCalls[2]; !reflect.DeepEqual(got, []string{"set-environment", "-t", "kai-123-dev", "YAAMA_WORKING_DIR", "/tmp/worktree"}) {
		t.Fatalf("expected working_dir env injection third, got %#v", got)
	}
	if got := runCalls[3]; !reflect.DeepEqual(got, []string{"rename-window", "-t", "kai-123-dev:0", "KAI-123-dev"}) {
		t.Fatalf("expected default window rename fourth, got %#v", got)
	}
	if got := runCalls[4]; !reflect.DeepEqual(got, []string{"new-window", "-d", "-t", "kai-123-dev", "-n", "ops", "-c", "/tmp/worktree"}) {
		t.Fatalf("expected additional profile window creation fifth, got %#v", got)
	}
	if got := runCalls[len(runCalls)-1]; !reflect.DeepEqual(got, []string{"select-window", "-t", "kai-123-dev:ops"}) {
		t.Fatalf("expected startup focus to select ops, got %#v", got)
	}

	if len(sendCalls) != 4 {
		t.Fatalf("expected four pane commands (pane0 export, pane0 cd, ops cd, agent command), got %d", len(sendCalls))
	}
	if sendCalls[0].target != "kai-123-dev:0.0" || !strings.Contains(sendCalls[0].command, "export YAAMA_TMUX_SESSION") {
		t.Fatalf("expected pane0 env export first, got %+v", sendCalls[0])
	}
	if sendCalls[1].target != "kai-123-dev:0.0" {
		t.Fatalf("expected default pane init target second, got %q", sendCalls[1].target)
	}
	if sendCalls[2].target != "kai-123-dev:ops.0" {
		t.Fatalf("expected profile pane init target third, got %q", sendCalls[2].target)
	}
	if sendCalls[3].target != "kai-123-dev:0.0" {
		t.Fatalf("expected agent command to target default pane, got %q", sendCalls[3].target)
	}
}

func TestBootstrapSessionWithoutAgentCommandSkipsAgentLaunch(t *testing.T) {
	originalAvailable := tmuxAvailableFn
	originalRunTmux := runTmuxFn
	originalSend := sendCommandToPaneFn
	t.Cleanup(func() {
		tmuxAvailableFn = originalAvailable
		runTmuxFn = originalRunTmux
		sendCommandToPaneFn = originalSend
	})

	tmuxAvailableFn = func() bool { return true }
	runTmuxFn = func(_ context.Context, args ...string) error { return nil }
	var sendCalls []struct {
		target  string
		command string
	}
	sendCommandToPaneFn = func(_ context.Context, paneTarget, command string) error {
		sendCalls = append(sendCalls, struct {
			target  string
			command string
		}{target: paneTarget, command: command})
		return nil
	}

	spec := BootstrapSpec{
		SessionName: "recovered-session",
		WorkingDir:  "/tmp/wd",
		AgentWindow: "recovered-session",
	}

	if err := BootstrapSession(context.Background(), spec); err != nil {
		t.Fatalf("BootstrapSession returned error: %v", err)
	}

	for _, call := range sendCalls {
		if strings.Contains(call.command, "codex") || strings.Contains(call.command, "claude ") {
			t.Fatalf("recovery should not launch agent command, got %+v", call)
		}
	}
	// Expect only pane0 export + pane0 cd
	if len(sendCalls) != 2 {
		t.Fatalf("expected exactly two pane commands when AgentCommand is nil, got %d (%+v)", len(sendCalls), sendCalls)
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
