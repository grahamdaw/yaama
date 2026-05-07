package tui

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/grahamdaw/yaama/internal/db/generated"
)

import tea "github.com/charmbracelet/bubbletea"

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m.handleModeKey(msg), nil
	}

	return m, nil
}

func (m model) handleModeKey(msg tea.KeyMsg) model {
	switch m.mode {
	case modeNormal:
		return m.handleNormalMode(msg)
	case modeSearch:
		return m.handleSearchMode(msg)
	case modeForm:
		return m.handleFormMode(msg)
	case modeConfirm:
		return m.handleConfirmMode(msg)
	case modeHelp:
		return m.handleHelpMode(msg)
	case modeStatusPicker:
		return m.handleStatusPickerMode(msg)
	default:
		return m
	}
}

func (m model) handleNormalMode(msg tea.KeyMsg) model {
	switch msg.String() {
	case "left", "h":
		return m.moveHorizontal(-1)
	case "right", "l":
		return m.moveHorizontal(1)
	case "up", "k":
		return m.moveVertical(-1)
	case "down", "j":
		return m.moveVertical(1)
	case "/":
		m.mode = modeSearch
		return m
	case "?":
		m.mode = modeHelp
		return m
	case "n", "e":
		m.mode = modeForm
		m.formDirty = false
		return m
	case "s":
		return m.openStatusPicker()
	case "S":
		return m.quickCycleStatus(-1)
	case "d":
		m.mode = modeConfirm
		m.confirm = confirmState{kind: confirmKindDelete, returnMode: modeNormal}
		return m
	case "esc":
		return m
	default:
		return m
	}
}

func (m model) handleStatusPickerMode(msg tea.KeyMsg) model {
	statuses := statusKeys()
	if len(statuses) == 0 {
		m.mode = modeNormal
		return m
	}

	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		return m
	case "left", "h", "up", "k":
		m.statusPicker.selected = (m.statusPicker.selected - 1 + len(statuses)) % len(statuses)
		return m
	case "right", "l", "down", "j":
		m.statusPicker.selected = (m.statusPicker.selected + 1) % len(statuses)
		return m
	case "1", "2", "3", "4", "5":
		index := int(msg.String()[0] - '1')
		if index >= 0 && index < len(statuses) {
			m.statusPicker.selected = index
		}
		return m
	case "enter":
		target := statuses[m.statusPicker.selected]
		m.mode = modeNormal
		return m.applyStatusTransition(target)
	default:
		return m
	}
}

func (m model) handleSearchMode(msg tea.KeyMsg) model {
	switch msg.String() {
	case "enter":
		m.mode = modeNormal
		return m
	case "esc":
		m.search = ""
		m.mode = modeNormal
		return m.rebuildColumns()
	case "backspace":
		if len(m.search) > 0 {
			m.search = m.search[:len(m.search)-1]
			return m.rebuildColumns()
		}
		return m
	default:
		if msg.Type == tea.KeyRunes {
			m.search += msg.String()
			return m.rebuildColumns()
		}
		return m
	}
}

func (m model) handleFormMode(msg tea.KeyMsg) model {
	switch msg.String() {
	case "esc":
		if m.formDirty {
			m.mode = modeConfirm
			m.confirm = confirmState{kind: confirmKindDiscardEdits, returnMode: modeForm}
			return m
		}
		m.mode = modeNormal
		return m
	default:
		if msg.Type == tea.KeyRunes || msg.String() == "backspace" {
			m.formDirty = true
		}
		return m
	}
}

func (m model) handleConfirmMode(msg tea.KeyMsg) model {
	switch msg.String() {
	case "esc":
		m.mode = m.confirm.returnMode
		m.confirm = confirmState{}
		return m
	case "enter":
		if m.confirm.kind == confirmKindDiscardEdits {
			m.mode = modeNormal
			m.formDirty = false
			m.confirm = confirmState{}
			return m
		}
		m.mode = modeNormal
		m.confirm = confirmState{}
		return m
	default:
		return m
	}
}

func (m model) handleHelpMode(msg tea.KeyMsg) model {
	switch msg.String() {
	case "esc", "?":
		m.mode = modeNormal
		return m
	default:
		return m
	}
}

func (m model) openStatusPicker() model {
	selected, ok := m.currentSelection()
	if !ok {
		return m.pushNotice("No agent selected; choose a row before changing status.")
	}

	index := statusIndex(selected.Status)
	if index < 0 {
		index = 0
	}
	m.statusPicker.selected = index
	m.mode = modeStatusPicker
	return m
}

func (m model) quickCycleStatus(delta int) model {
	selected, ok := m.currentSelection()
	if !ok {
		return m.pushNotice("No agent selected; choose a row before changing status.")
	}

	statuses := statusKeys()
	if len(statuses) == 0 {
		return m
	}
	current := statusIndex(selected.Status)
	if current < 0 {
		current = 0
	}
	target := statuses[(current+delta+len(statuses))%len(statuses)]
	return m.applyStatusTransition(target)
}

func (m model) applyStatusTransition(targetStatus string) model {
	selected, ok := m.currentSelection()
	if !ok {
		return m.pushNotice("No agent selected; choose a row before changing status.")
	}
	if selected.Status == targetStatus {
		return m.pushNotice(fmt.Sprintf("%s already %s.", selected.Name, statusTitle(targetStatus)))
	}

	if m.queries != nil {
		err := m.queries.UpdateAgentStatusByID(context.Background(), generated.UpdateAgentStatusByIDParams{
			Status:       targetStatus,
			Task:         selected.Task,
			LastActivity: selected.LastActivity,
			Branch:       selected.Branch,
			LastError:    selected.LastError,
			ID:           selected.ID,
		})
		if err != nil {
			return m.pushNotice(fmt.Sprintf("Status update failed for %s: %v", selected.Name, err))
		}

		rows, err := m.queries.ListActiveAgents(context.Background())
		if err != nil {
			return m.pushNotice(fmt.Sprintf("Status updated, but refresh failed: %v", err))
		}
		m.agents = rows
	} else {
		for i := range m.agents {
			if m.agents[i].ID == selected.ID {
				m.agents[i].Status = targetStatus
				m.agents[i].LastHeartbeatAt = sql.NullTime{}
				break
			}
		}
	}

	m.showEmpty = len(m.agents) == 0
	m = m.rebuildColumns()
	if colIdx, rowIdx, found := m.findSelectionByID(selected.ID); found {
		m.focused = colIdx
		m.selected[colIdx] = rowIdx
	}

	return m.pushNotice(fmt.Sprintf("Updated %s to %s.", selected.Name, statusTitle(targetStatus)))
}

func (m model) findSelectionByID(agentID int64) (int, int, bool) {
	for colIdx, col := range m.columns {
		for rowIdx, card := range col.cards {
			if card.ID == agentID {
				return colIdx, rowIdx, true
			}
		}
	}
	return 0, 0, false
}

func (m model) pushNotice(notice string) model {
	const maxNotices = 4
	m.notices = append(m.notices, notice)
	if len(m.notices) > maxNotices {
		m.notices = m.notices[len(m.notices)-maxNotices:]
	}
	return m
}

func statusIndex(status string) int {
	for idx, value := range statusKeys() {
		if value == status {
			return idx
		}
	}
	return -1
}

func (m model) moveHorizontal(delta int) model {
	if len(m.columns) == 0 {
		return m
	}

	target := m.focused + delta
	if target < 0 || target >= len(m.columns) {
		return m
	}

	currentRow := m.selected[m.focused]
	m.focused = target
	m.selected[m.focused] = clampRow(currentRow, len(m.columns[m.focused].cards))
	return m
}

func (m model) moveVertical(delta int) model {
	if len(m.columns) == 0 {
		return m
	}

	col := m.columns[m.focused]
	row := m.selected[m.focused]
	if len(col.cards) == 0 {
		m.selected[m.focused] = headerSelectionRow
		return m
	}

	switch {
	case delta < 0 && row == 0:
		m.selected[m.focused] = headerSelectionRow
	case delta < 0 && row > 0:
		m.selected[m.focused] = row - 1
	case delta > 0 && row < 0:
		m.selected[m.focused] = 0
	case delta > 0 && row < len(col.cards)-1:
		m.selected[m.focused] = row + 1
	}

	return m
}
