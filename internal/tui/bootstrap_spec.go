package tui

import (
	"github.com/grahamdaw/yaama/internal/profile"
	"github.com/grahamdaw/yaama/internal/tmux"
)

// toBootstrapSpec builds a tmux.BootstrapSpec from a resolved profile.
// When skipAgentRun is true, the pane marked agent = true has its Run
// suppressed — used by dead-session recovery so the agent process is not
// relaunched.
func toBootstrapSpec(sessionName string, workingDir string, skipAgentRun bool, cfg profile.Config) tmux.BootstrapSpec {
	spec := tmux.BootstrapSpec{
		SessionName:   sessionName,
		WorkingDir:    workingDir,
		LayoutFile:    cfg.LayoutFile,
		StartupWindow: cfg.StartupWindow,
		Setup:         cfg.Setup,
		SkipAgentRun:  skipAgentRun,
	}

	for _, window := range cfg.Windows {
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
				Agent: pane.Agent,
			})
		}
		spec.Windows = append(spec.Windows, nextWindow)
	}

	return spec
}

// minimalBootstrapSpec is used during recovery when the persisted profile
// can no longer be loaded. It produces a single-window session with one
// empty pane so the operator can still re-enter the working directory.
func minimalBootstrapSpec(sessionName, workingDir string) tmux.BootstrapSpec {
	return tmux.BootstrapSpec{
		SessionName: sessionName,
		WorkingDir:  workingDir,
		Windows: []tmux.BootstrapWindow{
			{
				Name:  "shell",
				Focus: true,
				Panes: []tmux.BootstrapPane{{Cwd: workingDir}},
			},
		},
	}
}
