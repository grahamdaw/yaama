package tmux

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

type sendCall struct {
	target  string
	command string
}

func stubTmux(t *testing.T) (*[][]string, *[]sendCall) {
	t.Helper()
	originalAvailable := tmuxAvailableFn
	originalRunTmux := runTmuxFn
	originalSend := sendCommandToPaneFn
	t.Cleanup(func() {
		tmuxAvailableFn = originalAvailable
		runTmuxFn = originalRunTmux
		sendCommandToPaneFn = originalSend
	})

	runCalls := &[][]string{}
	sendCalls := &[]sendCall{}

	tmuxAvailableFn = func() bool { return true }
	runTmuxFn = func(_ context.Context, args ...string) error {
		*runCalls = append(*runCalls, append([]string(nil), args...))
		return nil
	}
	sendCommandToPaneFn = func(_ context.Context, target, command string) error {
		*sendCalls = append(*sendCalls, sendCall{target: target, command: command})
		return nil
	}
	return runCalls, sendCalls
}

func TestBootstrapSessionRunsSetupBeforeNewSession(t *testing.T) {
	runCalls, _ := stubTmux(t)

	spec := BootstrapSpec{
		SessionName: "kai-123-dev",
		WorkingDir:  "/tmp/worktree",
		Windows: []BootstrapWindow{
			{Name: "agent", Focus: true, Panes: []BootstrapPane{{Agent: true, Run: "codex"}}},
		},
	}
	if err := BootstrapSession(context.Background(), spec); err != nil {
		t.Fatalf("BootstrapSession returned error: %v", err)
	}
	if len(*runCalls) == 0 {
		t.Fatalf("expected tmux calls")
	}
	first := (*runCalls)[0]
	if !reflect.DeepEqual(first, []string{
		"new-session", "-d", "-s", "kai-123-dev", "-c", "/tmp/worktree",
		";",
		"set-option", "-t", "kai-123-dev", "destroy-unattached", "off",
	}) {
		t.Fatalf("expected new-session as first call, got %#v", first)
	}
}

func TestBootstrapSessionDrivesAllWindowsFromSpec(t *testing.T) {
	runCalls, sendCalls := stubTmux(t)

	spec := BootstrapSpec{
		SessionName:   "kai-123-dev",
		WorkingDir:    "/tmp/worktree",
		StartupWindow: "ops",
		Windows: []BootstrapWindow{
			{Name: "agent", Panes: []BootstrapPane{{Agent: true, Run: "codex"}}},
			{Name: "ops", Focus: true, Panes: []BootstrapPane{{Run: "git status -sb"}}},
		},
	}

	if err := BootstrapSession(context.Background(), spec); err != nil {
		t.Fatalf("BootstrapSession returned error: %v", err)
	}

	hasRename := false
	hasNewOps := false
	hasSelectOps := false
	for _, call := range *runCalls {
		if reflect.DeepEqual(call, []string{"rename-window", "-t", "kai-123-dev:0", "agent"}) {
			hasRename = true
		}
		if reflect.DeepEqual(call, []string{"new-window", "-d", "-t", "kai-123-dev", "-n", "ops", "-c", "/tmp/worktree"}) {
			hasNewOps = true
		}
		if reflect.DeepEqual(call, []string{"select-window", "-t", "kai-123-dev:ops"}) {
			hasSelectOps = true
		}
	}
	if !hasRename {
		t.Fatalf("expected first window rename to 'agent', got %#v", *runCalls)
	}
	if !hasNewOps {
		t.Fatalf("expected ops window creation, got %#v", *runCalls)
	}
	if !hasSelectOps {
		t.Fatalf("expected select-window to focus ops, got %#v", *runCalls)
	}

	codexSent := false
	for _, c := range *sendCalls {
		if c.target == "kai-123-dev:agent.0" && strings.Contains(c.command, "codex") {
			codexSent = true
		}
	}
	if !codexSent {
		t.Fatalf("expected agent pane to receive its Run command, got %+v", *sendCalls)
	}
}

func TestBootstrapSessionSkipAgentRunSuppressesAgentPaneCommand(t *testing.T) {
	_, sendCalls := stubTmux(t)

	spec := BootstrapSpec{
		SessionName:  "recovered",
		WorkingDir:   "/tmp/wd",
		SkipAgentRun: true,
		Windows: []BootstrapWindow{
			{Name: "agent", Panes: []BootstrapPane{{Agent: true, Run: "codex"}}},
			{Name: "ops", Panes: []BootstrapPane{{Run: "echo non-agent"}}},
		},
	}
	if err := BootstrapSession(context.Background(), spec); err != nil {
		t.Fatalf("BootstrapSession returned error: %v", err)
	}

	for _, c := range *sendCalls {
		if strings.Contains(c.command, "codex") {
			t.Fatalf("recovery should skip agent pane Run, got %+v", c)
		}
	}
	sawNonAgent := false
	for _, c := range *sendCalls {
		if strings.Contains(c.command, "echo non-agent") {
			sawNonAgent = true
		}
	}
	if !sawNonAgent {
		t.Fatalf("expected non-agent pane Run to still execute on recovery, got %+v", *sendCalls)
	}
}

func TestFocusedWindowNamePrefersFocusFlag(t *testing.T) {
	spec := BootstrapSpec{
		Windows: []BootstrapWindow{
			{Name: "agent"},
			{Name: "ops", Focus: true},
		},
	}
	if got := focusedWindowName(spec); got != "ops" {
		t.Fatalf("expected ops, got %q", got)
	}
}

func TestFocusedWindowNameFallsBackToStartupWindowThenFirst(t *testing.T) {
	spec := BootstrapSpec{
		StartupWindow: "ops",
		Windows: []BootstrapWindow{
			{Name: "agent"},
			{Name: "ops"},
		},
	}
	if got := focusedWindowName(spec); got != "ops" {
		t.Fatalf("expected startup_window ops, got %q", got)
	}

	noStartup := BootstrapSpec{
		Windows: []BootstrapWindow{
			{Name: "agent"},
			{Name: "ops"},
		},
	}
	if got := focusedWindowName(noStartup); got != "agent" {
		t.Fatalf("expected first-window fallback, got %q", got)
	}
}
