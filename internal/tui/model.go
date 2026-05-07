package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/grahamdaw/yaama/internal/startup"
)

type column struct {
	title string
	cards []string
}

type model struct {
	width     int
	height    int
	columns   []column
	notices   []string
	showEmpty bool
}

func NewModel(state startup.State) tea.Model {
	showEmpty := state.DB.Created

	return model{
		columns:   seedColumns(showEmpty),
		notices:   state.Notices,
		showEmpty: showEmpty,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}
