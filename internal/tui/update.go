package tui

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
		if msg.String() == "n" {
			return m.openCreateForm(formPurposeCreateGeneric)
		}
		return m.openEditForm()
	case "s":
		return m.openStatusPicker()
	case "S":
		return m.quickCycleStatus(-1)
	case "d":
		return m.openArchiveConfirm()
	case "D":
		return m.openPruneConfirm()
	case "esc":
		return m
	default:
		return m
	}
}

func (m model) openArchiveConfirm() model {
	selected, ok := m.currentSelection()
	if !ok {
		return m.pushNotice("No agent selected; choose a row before archiving.")
	}
	m.mode = modeConfirm
	m.confirm = confirmState{
		kind:      confirmKindArchive,
		returnMode: modeNormal,
		agentID:   selected.ID,
		agentName: selected.Name,
	}
	return m
}

func (m model) openPruneConfirm() model {
	selected, ok := m.currentSelection()
	if !ok {
		return m.pushNotice("No agent selected; choose a row before pruning.")
	}
	kind := confirmKindPrune
	if strings.TrimSpace(nullStringRaw(selected.WorkingDir)) != "" {
		kind = confirmKindPruneForce
	}
	m.mode = modeConfirm
	m.confirm = confirmState{
		kind:       kind,
		returnMode: modeNormal,
		agentID:    selected.ID,
		agentName:  selected.Name,
		workingDir: nullStringRaw(selected.WorkingDir),
	}
	return m
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
	isCreateWizard := m.form.purpose == formPurposeCreateGeneric || m.form.purpose == formPurposeCreateProfile

	switch msg.String() {
	case "esc":
		if m.formDirty {
			m.mode = modeConfirm
			m.confirm = confirmState{kind: confirmKindDiscardEdits, returnMode: modeForm}
			return m
		}
		m.mode = modeNormal
		m.form = formState{}
		return m
	case "up", "k":
		if isCreateWizard && m.form.active == 0 {
			return m.cycleCreateProfile(-1)
		}
		if len(m.form.fields) > 0 {
			m.form.active = (m.form.active - 1 + len(m.form.fields)) % len(m.form.fields)
		}
		return m
	case "down", "j", "tab":
		if isCreateWizard && m.form.active == 0 {
			return m.cycleCreateProfile(1)
		}
		if len(m.form.fields) > 0 {
			m.form.active = (m.form.active + 1) % len(m.form.fields)
		}
		return m
	case "shift+tab":
		if len(m.form.fields) > 0 {
			m.form.active = (m.form.active - 1 + len(m.form.fields)) % len(m.form.fields)
		}
		return m
	case "left", "h":
		if len(m.form.fields) == 0 || m.form.active < 0 || m.form.active >= len(m.form.fields) {
			return m
		}
		if isCreateWizard && m.form.fields[m.form.active].key == "profile_name" {
			return m.cycleCreateProfile(-1)
		}
		if m.form.fields[m.form.active].key == "status" {
			return m.editActiveFormField(func(current string) string {
				index := statusIndex(strings.TrimSpace(current))
				if index < 0 {
					index = 0
				}
				statuses := statusKeys()
				return statuses[(index-1+len(statuses))%len(statuses)]
			})
		}
		return m
	case "right", "l":
		if len(m.form.fields) == 0 || m.form.active < 0 || m.form.active >= len(m.form.fields) {
			return m
		}
		if isCreateWizard && m.form.fields[m.form.active].key == "profile_name" {
			return m.cycleCreateProfile(1)
		}
		if m.form.fields[m.form.active].key == "status" {
			return m.editActiveFormField(func(current string) string {
				index := statusIndex(strings.TrimSpace(current))
				if index < 0 {
					index = 0
				}
				statuses := statusKeys()
				return statuses[(index+1)%len(statuses)]
			})
		}
		return m
	case "backspace":
		if isCreateWizard && m.form.active == 0 {
			return m
		}
		return m.editActiveFormField(func(current string) string {
			if len(current) == 0 {
				return current
			}
			return current[:len(current)-1]
		})
	case "enter":
		if isCreateWizard && m.form.active < len(m.form.fields)-1 {
			m.form.active++
			return m
		}
		return m.submitForm()
	default:
		if msg.Type == tea.KeyRunes {
			if isCreateWizard && m.form.active == 0 {
				return m
			}
			return m.editActiveFormField(func(current string) string {
				return current + msg.String()
			})
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
	case "f":
		if m.confirm.kind == confirmKindPruneForce {
			m.confirm.kind = confirmKindPrune
			m.confirm.force = true
			return m.pushNotice("Force prune enabled; press Enter to confirm.")
		}
		return m
	case "enter":
		switch m.confirm.kind {
		case confirmKindDiscardEdits:
			m.mode = modeNormal
			m.formDirty = false
			m.form = formState{}
			m.confirm = confirmState{}
			return m
		case confirmKindArchive:
			return m.applyArchive()
		case confirmKindPrune:
			return m.applyPrune()
		case confirmKindPruneForce:
			return m.pushNotice("Working directory is non-empty; press f then Enter to force prune.")
		default:
			m.mode = modeNormal
			m.confirm = confirmState{}
			return m
		}
	default:
		return m
	}
}

func (m model) applyArchive() model {
	target, ok := m.findAgentByID(m.confirm.agentID)
	if !ok {
		m.mode = modeNormal
		m.confirm = confirmState{}
		return m.pushNotice("Archive target no longer exists.")
	}

	if m.queries != nil {
		err := m.queries.UpdateAgentCleanupState(context.Background(), generated.UpdateAgentCleanupStateParams{
			ID:           target.ID,
			CleanupState: "archived",
		})
		if err != nil {
			return m.pushNotice(fmt.Sprintf("Archive failed: %v", err))
		}
		rows, err := m.queries.ListActiveAgents(context.Background())
		if err != nil {
			return m.pushNotice(fmt.Sprintf("Archive succeeded, but refresh failed: %v", err))
		}
		m.agents = rows
	} else {
		for i := range m.agents {
			if m.agents[i].ID == target.ID {
				m.agents[i].CleanupState = "archived"
				break
			}
		}
		m.agents = filterActiveAgents(m.agents)
	}

	m.mode = modeNormal
	m.confirm = confirmState{}
	m.showEmpty = len(m.agents) == 0
	return m.rebuildColumns().pushNotice(fmt.Sprintf("Archived %s.", target.Name))
}

func (m model) applyPrune() model {
	target, ok := m.findAgentByID(m.confirm.agentID)
	if !ok {
		m.mode = modeNormal
		m.confirm = confirmState{}
		return m.pushNotice("Prune target no longer exists.")
	}
	if strings.TrimSpace(nullStringRaw(target.WorkingDir)) != "" && !m.confirm.force {
		return m.pushNotice("Working directory is non-empty; force prune required.")
	}

	if m.queries != nil {
		err := m.queries.DeleteAgent(context.Background(), target.ID)
		if err != nil {
			return m.pushNotice(fmt.Sprintf("Prune failed: %v", err))
		}
		rows, err := m.queries.ListActiveAgents(context.Background())
		if err != nil {
			return m.pushNotice(fmt.Sprintf("Prune succeeded, but refresh failed: %v", err))
		}
		m.agents = rows
	} else {
		next := make([]generated.Agent, 0, len(m.agents))
		for _, agent := range m.agents {
			if agent.ID != target.ID {
				next = append(next, agent)
			}
		}
		m.agents = next
	}

	m.mode = modeNormal
	m.confirm = confirmState{}
	m.showEmpty = len(m.agents) == 0
	return m.rebuildColumns().pushNotice(fmt.Sprintf("Pruned %s.", target.Name))
}

func filterActiveAgents(agents []generated.Agent) []generated.Agent {
	out := make([]generated.Agent, 0, len(agents))
	for _, agent := range agents {
		if agent.CleanupState == "active" || agent.CleanupState == "" {
			out = append(out, agent)
		}
	}
	return out
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
