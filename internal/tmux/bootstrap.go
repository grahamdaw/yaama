package tmux

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type BootstrapSpec struct {
	SessionName   string
	WorkingDir    string
	AgentWindow   string
	LayoutFile    string
	StartupWindow string
	BeforeStart   []string
	AfterStart    []string
	AgentCommand  []string
	Windows       []BootstrapWindow
}

type BootstrapWindow struct {
	Name  string
	Focus bool
	Panes []BootstrapPane
}

type BootstrapPane struct {
	Split string
	Size  string
	Cwd   string
	Run   string
}

var (
	tmuxAvailableFn     = IsAvailable
	runTmuxFn           = runTmux
	sendCommandToPaneFn = sendCommandToPane
)

func BootstrapSession(ctx context.Context, spec BootstrapSpec) error {
	if !tmuxAvailableFn() {
		return ErrTmuxUnavailable
	}

	if strings.TrimSpace(spec.SessionName) == "" {
		return fmt.Errorf("bootstrap tmux session: session name is required")
	}
	if strings.TrimSpace(spec.WorkingDir) == "" {
		return fmt.Errorf("bootstrap tmux session: working directory is required")
	}

	for _, hook := range spec.BeforeStart {
		if err := RunShellHook(ctx, spec.WorkingDir, spec.SessionName, hook); err != nil {
			return fmt.Errorf("bootstrap tmux session: before_start hook failed: %w", err)
		}
	}

	if err := runTmuxFn(ctx, createDetachedSessionArgs(spec.SessionName, spec.WorkingDir)...); err != nil {
		return fmt.Errorf("bootstrap tmux session: create session: %w", err)
	}

	if err := applyWindowsAndPanes(ctx, spec); err != nil {
		return err
	}

	if strings.TrimSpace(spec.LayoutFile) != "" {
		layoutTarget := fmt.Sprintf("%s:%s.0", spec.SessionName, focusedWindowName(spec))
		if err := runTmuxFn(ctx, "source-file", "-t", layoutTarget, spec.LayoutFile); err != nil {
			return fmt.Errorf("bootstrap tmux session: source layout file: %w", err)
		}
	}

	for _, hook := range spec.AfterStart {
		if err := RunShellHook(ctx, spec.WorkingDir, spec.SessionName, hook); err != nil {
			return fmt.Errorf("bootstrap tmux session: after_start hook failed: %w", err)
		}
	}

	if len(spec.AgentCommand) > 0 {
		targetPane := fmt.Sprintf("%s:0.0", spec.SessionName)
		if err := sendCommandToPaneFn(ctx, targetPane, strings.Join(spec.AgentCommand, " ")); err != nil {
			return fmt.Errorf("bootstrap tmux session: start agent command: %w", err)
		}
	}

	if targetWindow := focusedWindowName(spec); targetWindow != "" {
		if err := runTmuxFn(ctx, "select-window", "-t", windowTarget(spec, targetWindow)); err != nil {
			return fmt.Errorf("bootstrap tmux session: select startup window: %w", err)
		}
	}

	return nil
}

func applyWindowsAndPanes(ctx context.Context, spec BootstrapSpec) error {
	agentWindowName := defaultAgentWindowName(spec)
	if err := runTmuxFn(ctx, "rename-window", "-t", fmt.Sprintf("%s:0", spec.SessionName), agentWindowName); err != nil {
		return fmt.Errorf("bootstrap tmux session: rename initial window: %w", err)
	}
	if err := initializePane(ctx, spec.WorkingDir, fmt.Sprintf("%s:0.0", spec.SessionName), BootstrapPane{Cwd: spec.WorkingDir}); err != nil {
		return fmt.Errorf("bootstrap tmux session: initialize pane %s: %w", fmt.Sprintf("%s:0.0", spec.SessionName), err)
	}

	for windowIdx, window := range spec.Windows {
		windowName := strings.TrimSpace(window.Name)
		if windowName == "" {
			windowName = fmt.Sprintf("window-%d", windowIdx+1)
		}

		if err := runTmuxFn(ctx, "new-window", "-d", "-t", spec.SessionName, "-n", windowName, "-c", spec.WorkingDir); err != nil {
			return fmt.Errorf("bootstrap tmux session: create window %q: %w", windowName, err)
		}

		panes := window.Panes
		if len(panes) == 0 {
			panes = []BootstrapPane{{Cwd: spec.WorkingDir}}
		}

		primaryPaneTarget := fmt.Sprintf("%s:%s.0", spec.SessionName, windowName)
		if err := initializePane(ctx, spec.WorkingDir, primaryPaneTarget, panes[0]); err != nil {
			return fmt.Errorf("bootstrap tmux session: initialize pane %s: %w", primaryPaneTarget, err)
		}

		for paneIdx := 1; paneIdx < len(panes); paneIdx++ {
			pane := panes[paneIdx]
			splitArgs := []string{"split-window", "-t", primaryPaneTarget}
			switch strings.ToLower(strings.TrimSpace(pane.Split)) {
			case "horizontal":
				splitArgs = append(splitArgs, "-h")
			default:
				splitArgs = append(splitArgs, "-v")
			}
			if size := strings.TrimSpace(pane.Size); size != "" {
				splitArgs = append(splitArgs, "-l", size)
			}
			paneCwd := resolvePaneCwd(spec.WorkingDir, pane.Cwd)
			splitArgs = append(splitArgs, "-c", paneCwd)
			if err := runTmuxFn(ctx, splitArgs...); err != nil {
				return fmt.Errorf("bootstrap tmux session: create pane %d in %s: %w", paneIdx, windowName, err)
			}

			paneTarget := fmt.Sprintf("%s:%s.%d", spec.SessionName, windowName, paneIdx)
			if err := initializePane(ctx, spec.WorkingDir, paneTarget, pane); err != nil {
				return fmt.Errorf("bootstrap tmux session: initialize pane %s: %w", paneTarget, err)
			}
		}
	}

	return nil
}

func initializePane(ctx context.Context, workingDir, paneTarget string, pane BootstrapPane) error {
	paneCwd := resolvePaneCwd(workingDir, pane.Cwd)
	if err := sendCommandToPaneFn(ctx, paneTarget, "cd "+shellQuote(paneCwd)); err != nil {
		return err
	}
	if strings.TrimSpace(pane.Run) != "" {
		if err := sendCommandToPaneFn(ctx, paneTarget, pane.Run); err != nil {
			return err
		}
	}
	return nil
}

func focusedWindowName(spec BootstrapSpec) string {
	defaultWindow := defaultAgentWindowName(spec)
	for _, window := range spec.Windows {
		if window.Focus && strings.TrimSpace(window.Name) != "" {
			return window.Name
		}
	}
	if strings.TrimSpace(spec.StartupWindow) != "" {
		startup := strings.TrimSpace(spec.StartupWindow)
		if startup == "agent" {
			return defaultWindow
		}
		return startup
	}
	return defaultWindow
}

func resolvePaneCwd(workingDir, paneCwd string) string {
	cwd := strings.TrimSpace(paneCwd)
	if cwd == "" {
		return workingDir
	}
	if filepath.IsAbs(cwd) {
		return filepath.Clean(cwd)
	}
	return filepath.Clean(filepath.Join(workingDir, cwd))
}

func sendCommandToPane(ctx context.Context, paneTarget, command string) error {
	return runTmuxFn(ctx, "send-keys", "-t", paneTarget, command, "C-m")
}

func runTmux(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "tmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func RunShellHook(ctx context.Context, workingDir, sessionName, command string) error {
	cmd := exec.CommandContext(ctx, "sh", "-lc", command)
	cmd.Dir = workingDir
	cmd.Env = append(
		os.Environ(),
		"YAAMA_TMUX_SESSION="+strings.TrimSpace(sessionName),
		"YAAMA_WORKING_DIR="+strings.TrimSpace(workingDir),
		"TMUX=",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func shellQuote(value string) string {
	if strings.TrimSpace(value) == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func defaultAgentWindowName(spec BootstrapSpec) string {
	return sanitizeWindowName(spec.AgentWindow, "agent")
}

func windowTarget(spec BootstrapSpec, windowName string) string {
	if strings.TrimSpace(windowName) == defaultAgentWindowName(spec) {
		return fmt.Sprintf("%s:0", spec.SessionName)
	}
	return fmt.Sprintf("%s:%s", spec.SessionName, windowName)
}

var unsafeWindowNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeWindowName(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	sanitized := unsafeWindowNameChars.ReplaceAllString(trimmed, "-")
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		return fallback
	}
	return sanitized
}
