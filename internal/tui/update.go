package tui

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
