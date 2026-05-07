package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/grahamdaw/yaama/internal/db/generated"
	"github.com/grahamdaw/yaama/internal/startup"
)

type column struct {
	key   string
	title string
	cards []generated.Agent
}

type model struct {
	width     int
	height    int
	mode      mode
	columns   []column
	agents    []generated.Agent
	focused   int
	selected  []int
	search    string
	formDirty bool
	confirm   confirmState
	notices   []string
	showEmpty bool
}

type mode int

const (
	modeNormal mode = iota
	modeSearch
	modeForm
	modeConfirm
	modeHelp
)

type confirmState struct {
	returnMode mode
	kind       string
}

const (
	confirmKindNone          = ""
	confirmKindDelete        = "delete"
	confirmKindDiscardEdits  = "discard"
	headerSelectionRow       = -1
)

func NewModel(state startup.State) tea.Model {
	agents := []generated.Agent{}
	if state.DB.Queries != nil {
		rows, err := state.DB.Queries.ListActiveAgents(context.Background())
		if err == nil {
			agents = rows
		} else {
			state.Notices = append(state.Notices, "Unable to load agents from DB; showing empty board.")
		}
	}

	showEmpty := len(agents) == 0
	columns := buildColumns(agents, "")
	selected := make([]int, len(columns))
	for i, col := range columns {
		selected[i] = defaultSelectedRow(col.cards)
	}

	return model{
		mode:      modeNormal,
		columns:   columns,
		agents:    agents,
		focused:   0,
		selected:  selected,
		notices:   state.Notices,
		showEmpty: showEmpty,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func buildColumns(agents []generated.Agent, search string) []column {
	columns := newStatusColumns()
	filtered := filterAgents(agents, search)
	for _, agent := range filtered {
		for i := range columns {
			if columns[i].key == agent.Status {
				columns[i].cards = append(columns[i].cards, agent)
				break
			}
		}
	}

	return columns
}

func filterAgents(agents []generated.Agent, search string) []generated.Agent {
	normalized := strings.TrimSpace(strings.ToLower(search))
	if normalized == "" {
		return agents
	}

	out := make([]generated.Agent, 0, len(agents))
	for _, agent := range agents {
		if containsAny(normalized,
			agent.Name,
			agent.TmuxSession,
			agent.Task.String,
			agent.Branch.String,
		) {
			out = append(out, agent)
		}
	}

	return out
}

func containsAny(needle string, fields ...string) bool {
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), needle) {
			return true
		}
	}
	return false
}

func defaultSelectedRow(cards []generated.Agent) int {
	if len(cards) == 0 {
		return headerSelectionRow
	}
	return 0
}

func (m model) rebuildColumns() model {
	oldColumns := m.columns
	oldSelected := m.selected

	m.columns = buildColumns(m.agents, m.search)
	m.selected = make([]int, len(m.columns))
	for i, col := range m.columns {
		prev := headerSelectionRow
		if i < len(oldSelected) {
			prev = oldSelected[i]
		}
		if i < len(oldColumns) && oldColumns[i].key != col.key {
			prev = defaultSelectedRow(col.cards)
		}
		m.selected[i] = clampRow(prev, len(col.cards))
	}

	if len(m.columns) == 0 {
		m.focused = 0
		return m
	}

	if m.focused < 0 {
		m.focused = 0
	}
	if m.focused >= len(m.columns) {
		m.focused = len(m.columns) - 1
	}

	return m
}

func clampRow(row int, cardCount int) int {
	if cardCount <= 0 {
		return headerSelectionRow
	}
	if row < 0 {
		return 0
	}
	if row >= cardCount {
		return cardCount - 1
	}
	return row
}

