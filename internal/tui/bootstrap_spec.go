package tui

import (
	"github.com/grahamdaw/yaama/internal/profile"
	"github.com/grahamdaw/yaama/internal/tmux"
)

// toBootstrapSpec builds a tmux.BootstrapSpec from a resolved profile plus
// runtime values. When agentCommand is nil/empty, the resulting bootstrap
// will lay out the session and run profile hooks without launching the
// agent process — used by the dead-session recovery flow.
func toBootstrapSpec(sessionName string, workingDir string, agentCommand []string, cfg profile.Config) tmux.BootstrapSpec {
	spec := tmux.BootstrapSpec{
		SessionName:   sessionName,
		WorkingDir:    workingDir,
		AgentWindow:   sessionName,
		LayoutFile:    cfg.Tmux.LayoutFile,
		StartupWindow: cfg.Tmux.StartupWindow,
		BeforeStart:   cfg.Scripts.BeforeStart,
		AfterStart:    cfg.Scripts.AfterStart,
		AgentCommand:  agentCommand,
	}

	for _, window := range cfg.Tmux.Windows {
		nextWindow := tmux.BootstrapWindow{
			Name:  window.Name,
			Focus: window.Focus,
			Panes: make([]tmux.BootstrapPane, 0, len(window.Panes)),
		}
		for _, pane := range window.Panes {
			nextWindow.Panes = append(nextWindow.Panes, tmux.BootstrapPane{
				Split: pane.Split,
				Size:  pane.Size,
				Cwd:   pane.Cwd,
				Run:   pane.Run,
			})
		}
		spec.Windows = append(spec.Windows, nextWindow)
	}

	return spec
}

// minimalBootstrapSpec is used when no profile is available — recovery still
// creates the session and names the default agent window, but skips windows
// and hooks.
func minimalBootstrapSpec(sessionName, workingDir string) tmux.BootstrapSpec {
	return tmux.BootstrapSpec{
		SessionName: sessionName,
		WorkingDir:  workingDir,
		AgentWindow: sessionName,
	}
}
