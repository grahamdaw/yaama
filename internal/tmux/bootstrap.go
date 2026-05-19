package tmux

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grahamdaw/yaama/internal/logging"
)

type BootstrapSpec struct {
	SessionName   string
	WorkingDir    string
	LayoutFile    string
	StartupWindow string
	Setup         string
	SkipAgentRun  bool
	Windows       []BootstrapWindow
	Logger        *slog.Logger
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
	Agent bool
}

func specLogger(spec BootstrapSpec) *slog.Logger {
	if spec.Logger != nil {
		return spec.Logger.With("session", spec.SessionName, "working_dir", spec.WorkingDir)
	}
	return logging.Discard()
}

var (
	tmuxAvailableFn     = IsAvailable
	runTmuxFn           = runTmux
	sendCommandToPaneFn = sendCommandToPane
)

func BootstrapSession(ctx context.Context, spec BootstrapSpec) error {
	log := specLogger(spec)
	if !tmuxAvailableFn() {
		log.Error("tmux.bootstrap.unavailable")
		return ErrTmuxUnavailable
	}

	if strings.TrimSpace(spec.SessionName) == "" {
		return fmt.Errorf("bootstrap tmux session: session name is required")
	}
	if strings.TrimSpace(spec.WorkingDir) == "" {
		return fmt.Errorf("bootstrap tmux session: working directory is required")
	}
	if len(spec.Windows) == 0 {
		return fmt.Errorf("bootstrap tmux session: at least one window is required")
	}

	log.Info("tmux.bootstrap.begin",
		"windows", len(spec.Windows),
		"skip_agent_run", spec.SkipAgentRun)

	if setup := strings.TrimSpace(spec.Setup); setup != "" {
		if err := RunShellHook(ctx, spec.WorkingDir, spec.SessionName, setup); err != nil {
			log.Error("tmux.bootstrap.setup", "err", logging.Truncate(err.Error(), 512))
			return fmt.Errorf("bootstrap tmux session: setup script failed: %w", err)
		}
		log.Debug("tmux.bootstrap.setup.done")
	}

	if err := runTmuxFn(ctx, createDetachedSessionArgs(spec.SessionName, spec.WorkingDir)...); err != nil {
		log.Error("tmux.bootstrap.new_session", "err", logging.Truncate(err.Error(), 512))
		return fmt.Errorf("bootstrap tmux session: create session: %w", err)
	}
	log.Debug("tmux.bootstrap.new_session.done")

	if err := injectSessionEnv(ctx, spec); err != nil {
		log.Error("tmux.bootstrap.set_environment", "err", logging.Truncate(err.Error(), 512))
		return err
	}

	if err := applyWindowsAndPanes(ctx, spec); err != nil {
		log.Error("tmux.bootstrap.apply_windows", "err", logging.Truncate(err.Error(), 512))
		return err
	}

	if strings.TrimSpace(spec.LayoutFile) != "" {
		layoutTarget := fmt.Sprintf("%s:%s.0", spec.SessionName, focusedWindowName(spec))
		if err := runTmuxFn(ctx, "source-file", "-t", layoutTarget, spec.LayoutFile); err != nil {
			log.Error("tmux.bootstrap.source_layout", "layout", spec.LayoutFile, "err", logging.Truncate(err.Error(), 512))
			return fmt.Errorf("bootstrap tmux session: source layout file: %w", err)
		}
	}

	if targetWindow := focusedWindowName(spec); targetWindow != "" {
		if err := runTmuxFn(ctx, "select-window", "-t", fmt.Sprintf("%s:%s", spec.SessionName, targetWindow)); err != nil {
			log.Error("tmux.bootstrap.select_window", "window", targetWindow, "err", logging.Truncate(err.Error(), 512))
			return fmt.Errorf("bootstrap tmux session: select startup window: %w", err)
		}
	}

	log.Info("tmux.bootstrap.ready")
	return nil
}

func injectSessionEnv(ctx context.Context, spec BootstrapSpec) error {
	session := strings.TrimSpace(spec.SessionName)
	workingDir := strings.TrimSpace(spec.WorkingDir)
	if err := runTmuxFn(ctx, "set-environment", "-t", session, "YAAMA_TMUX_SESSION", session); err != nil {
		return fmt.Errorf("bootstrap tmux session: set-environment YAAMA_TMUX_SESSION: %w", err)
	}
	if err := runTmuxFn(ctx, "set-environment", "-t", session, "YAAMA_WORKING_DIR", workingDir); err != nil {
		return fmt.Errorf("bootstrap tmux session: set-environment YAAMA_WORKING_DIR: %w", err)
	}
	return nil
}

func sessionEnvExportCommand(spec BootstrapSpec) string {
	session := strings.TrimSpace(spec.SessionName)
	workingDir := strings.TrimSpace(spec.WorkingDir)
	return fmt.Sprintf("export YAAMA_TMUX_SESSION=%s YAAMA_WORKING_DIR=%s",
		shellQuote(session), shellQuote(workingDir))
}

func applyWindowsAndPanes(ctx context.Context, spec BootstrapSpec) error {
	for windowIdx, window := range spec.Windows {
		windowName := strings.TrimSpace(window.Name)
		if windowName == "" {
			windowName = fmt.Sprintf("window-%d", windowIdx+1)
		}

		if windowIdx == 0 {
			if err := runTmuxFn(ctx, "rename-window", "-t", fmt.Sprintf("%s:0", spec.SessionName), windowName); err != nil {
				return fmt.Errorf("bootstrap tmux session: rename initial window: %w", err)
			}
		} else {
			if err := runTmuxFn(ctx, "new-window", "-d", "-t", spec.SessionName, "-n", windowName, "-c", spec.WorkingDir); err != nil {
				return fmt.Errorf("bootstrap tmux session: create window %q: %w", windowName, err)
			}
		}

		panes := window.Panes
		if len(panes) == 0 {
			panes = []BootstrapPane{{Cwd: spec.WorkingDir}}
		}

		primaryPaneTarget := fmt.Sprintf("%s:%s.0", spec.SessionName, windowName)
		if windowIdx == 0 {
			if err := sendCommandToPaneFn(ctx, primaryPaneTarget, sessionEnvExportCommand(spec)); err != nil {
				return fmt.Errorf("bootstrap tmux session: export session env in pane %s: %w", primaryPaneTarget, err)
			}
		}
		if err := initializePane(ctx, spec.WorkingDir, primaryPaneTarget, panes[0], spec.SkipAgentRun); err != nil {
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
			if err := initializePane(ctx, spec.WorkingDir, paneTarget, pane, spec.SkipAgentRun); err != nil {
				return fmt.Errorf("bootstrap tmux session: initialize pane %s: %w", paneTarget, err)
			}
		}
	}

	return nil
}

func initializePane(ctx context.Context, workingDir, paneTarget string, pane BootstrapPane, skipAgentRun bool) error {
	paneCwd := resolvePaneCwd(workingDir, pane.Cwd)
	if err := sendCommandToPaneFn(ctx, paneTarget, "cd "+shellQuote(paneCwd)); err != nil {
		return err
	}
	if pane.Agent && skipAgentRun {
		return nil
	}
	if strings.TrimSpace(pane.Run) != "" {
		if err := sendCommandToPaneFn(ctx, paneTarget, pane.Run); err != nil {
			return err
		}
	}
	return nil
}

func focusedWindowName(spec BootstrapSpec) string {
	for _, window := range spec.Windows {
		if window.Focus && strings.TrimSpace(window.Name) != "" {
			return window.Name
		}
	}
	if startup := strings.TrimSpace(spec.StartupWindow); startup != "" {
		return startup
	}
	if len(spec.Windows) > 0 {
		return strings.TrimSpace(spec.Windows[0].Name)
	}
	return ""
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
