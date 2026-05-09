package tui

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/grahamdaw/yaama/internal/db/generated"
)

func (m model) View() string {
	width := max(m.width, 80)
	if m.width > 0 {
		width = m.width
	}

	header := lipgloss.NewStyle().Bold(true).Padding(0, 1).
		Render("Agent Board")
	stats := lipgloss.NewStyle().Faint(true).Render(m.renderStats())

	topBar := lipgloss.JoinHorizontal(lipgloss.Left, header, "  ", stats)
	topBar = lipgloss.NewStyle().Width(width).Render(topBar)

	board := m.renderBoard(width)

	sections := []string{topBar}
	if m.mode == modeSearch {
		sections = append(sections, m.renderSearchBar(width))
	}
	if m.mode == modeStatusPicker {
		sections = append(sections, m.renderStatusPickerBar(width))
	}
	if strings.TrimSpace(m.banner) != "" {
		sections = append(sections, m.renderBanner(width))
	}
	if len(m.toasts) > 0 {
		sections = append(sections, m.renderToasts(width))
	}
	if m.showEmpty {
		sections = append(sections, m.renderEmptyState(width))
	}
	sections = append(sections, board, m.renderFooter(width))
	if m.mode == modeForm {
		sections = append(sections, m.renderFormOverlay(width))
	}
	if m.mode == modeHelp {
		sections = append(sections, m.renderHelpOverlay(width))
	}
	if m.mode == modeConfirm {
		sections = append(sections, m.renderConfirmOverlay(width))
	}

	return strings.Join(sections, "\n")
}

func (m model) renderBoard(totalWidth int) string {
	columnSection := m.renderColumns(totalWidth)
	detailSection := m.renderDetailPanel(totalWidth)
	return lipgloss.JoinVertical(lipgloss.Left, columnSection, detailSection)
}

func (m model) renderColumns(totalWidth int) string {
	if len(m.columns) == 0 {
		return ""
	}

	usableWidth := max(totalWidth-2, 20)
	columnCount := len(m.columns)
	columnGap := 0
	totalGapWidth := (columnCount - 1) * columnGap
	borderAndPaddingWidth := 4 // 2 border chars + 1 left + 1 right padding

	outerColumnWidth := (usableWidth - totalGapWidth) / columnCount
	columnWidth := max(outerColumnWidth-borderAndPaddingWidth, 6)

	baseColumnStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(columnWidth)

	rendered := make([]string, 0, len(m.columns))
	for idx, col := range m.columns {
		columnStyle := baseColumnStyle
		if idx == m.focused {
			columnStyle = columnStyle.BorderForeground(lipgloss.Color("12"))
		}

		title := fmt.Sprintf("%s (%d)", col.title, len(col.cards))
		bodyLines := []string{}
		if len(col.cards) == 0 {
			if m.selected[idx] == headerSelectionRow {
				bodyLines = append(bodyLines, focusedStyle().Render("(empty)"))
			} else {
				bodyLines = append(bodyLines, "(empty)")
			}
		} else {
			for cardIdx, card := range col.cards {
				label := card.Name
				if card.Task.Valid && card.Task.String != "" {
					label = fmt.Sprintf("%s — %s", label, card.Task.String)
				}
				label += m.runtimeBadge(card)
				if m.selected[idx] == cardIdx {
					bodyLines = append(bodyLines, focusedStyle().Render(label))
				} else {
					bodyLines = append(bodyLines, label)
				}
			}
		}

		rendered = append(rendered, columnStyle.Render(fmt.Sprintf("%s\n\n%s", title, strings.Join(bodyLines, "\n"))))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m model) renderDetailPanel(width int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(max(width-2, 20))

	title := "Details"
	if len(m.columns) > 0 {
		title = fmt.Sprintf("Details · %s", m.columns[m.focused].title)
	}

	lines := []string{title, ""}
	if agent, ok := m.currentSelection(); ok {
		lines = append(lines,
			fmt.Sprintf("Name: %s", agent.Name),
			fmt.Sprintf("Status: %s", agent.Status),
			fmt.Sprintf("Session: %s", agent.TmuxSession),
			fmt.Sprintf("Task: %s", nullStringValue(agent.Task)),
			fmt.Sprintf("Branch: %s", nullStringValue(agent.Branch)),
			fmt.Sprintf("Working Dir: %s", nullStringValue(agent.WorkingDir)),
			fmt.Sprintf("Profile: %s", nullStringValue(agent.ProfileName)),
			fmt.Sprintf("Ticket: %s", nullStringValue(agent.TicketID)),
			fmt.Sprintf("Activity: %s", nullStringValue(agent.LastActivity)),
			fmt.Sprintf("Heartbeat: %s", nullTimeValue(agent.LastHeartbeatAt)),
			fmt.Sprintf("Last Error: %s", nullStringValue(agent.LastError)),
			fmt.Sprintf("Runtime: %s", m.agentRuntimeState(agent)),
			fmt.Sprintf("Cleanup: %s", agent.CleanupState),
			fmt.Sprintf("Updated: %s", agent.UpdatedAt.Format(time.RFC3339)),
		)
		if m.isDead(agent) {
			recoveryHint := "Recovery: press r to recreate in working_dir, e to edit mapping, d archive cleanup, D hard prune cleanup."
			if strings.TrimSpace(nullStringRaw(agent.WorkingDir)) == "" {
				recoveryHint = "Recovery: working_dir is missing; press e to set it, then press r to recreate session."
			}
			lines = append(lines, recoveryHint)
		}
	} else {
		lines = append(lines, "No agent selected in this column.")
	}

	return style.Render(strings.Join(lines, "\n"))
}

func (m model) renderEmptyState(width int) string {
	copy := []string{
		"No agents yet.",
		fmt.Sprintf("Press %s to create your first agent.", keyNewAgentHint),
		fmt.Sprintf("From inside tmux, update status with: %s", keyStatusUpdateHint),
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(max(width-2, 20)).
		Render(strings.Join(copy, "\n"))
}

func (m model) renderBanner(width int) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Bold(true).
		Width(max(width-2, 20)).
		Render(m.banner)
}

func (m model) renderToasts(width int) string {
	lines := make([]string, 0, len(m.toasts))
	for _, toast := range m.toasts {
		color := lipgloss.Color("10")
		prefix := "OK"
		switch toast.severity {
		case toastWarning:
			color = lipgloss.Color("11")
			prefix = "WARN"
		case toastError:
			color = lipgloss.Color("9")
			prefix = "ERR"
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf("[%s] %s", prefix, toast.message)))
	}
	return lipgloss.NewStyle().
		Width(max(width-2, 20)).
		Render(strings.Join(lines, " · "))
}

func (m model) renderFooter(width int) string {
	footer := "h/l columns  j/k rows  Enter attach  r recover  / search  s status (1..5, S reverse)  n/e create/edit  d/D cleanup  Esc back  ? help  q quit"
	return lipgloss.NewStyle().
		Faint(true).
		Width(max(width-2, 20)).
		Render(footer)
}

func (m model) renderStats() string {
	total := 0
	running := 0
	blocked := 0
	dead := 0

	for _, col := range m.columns {
		total += len(col.cards)
		if col.key == "running" {
			running += len(col.cards)
		}
		if col.key == "blocked" {
			blocked += len(col.cards)
		}
		for _, card := range col.cards {
			if m.isDead(card) {
				dead++
			}
		}
	}

	return fmt.Sprintf("total %d · running %d · blocked %d · dead %d · mode %s", total, running, blocked, dead, m.modeLabel())
}

func (m model) renderSearchBar(width int) string {
	copy := fmt.Sprintf("Search: %s", m.search)
	if strings.TrimSpace(m.search) == "" {
		copy = "Search: (type to filter by name/task/branch/session)"
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")).
		Width(max(width-2, 20)).
		Render(copy)
}

func (m model) renderStatusPickerBar(width int) string {
	statuses := statusKeys()
	options := make([]string, 0, len(statuses))
	for idx, status := range statuses {
		label := fmt.Sprintf("%d:%s", idx+1, statusTitle(status))
		if idx == m.statusPicker.selected {
			label = focusedStyle().Render(label)
		}
		options = append(options, label)
	}

	copy := fmt.Sprintf("Set status: %s · Enter apply · Esc cancel", strings.Join(options, "  "))
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")).
		Width(max(width-2, 20)).
		Render(copy)
}

func (m model) renderHelpOverlay(width int) string {
	copy := []string{
		"Help",
		"",
		"Navigation: h/l or arrows move columns; j/k or arrows move rows.",
		"Attach: Enter attaches/switches into selected live tmux session.",
		"Dead session recovery: r recreates selected dead session in working_dir and immediately attaches.",
		"CRUD: n opens 2-step create wizard (profile -> task), e edit selected, d archive cleanup, D hard prune cleanup.",
		"Modes: / enters search, s opens status picker, ? toggles help.",
		"Status picker: press 1..5 to target a status, Enter to apply, Esc to cancel, S for reverse quick cycle.",
		"Create wizard infers name + tmux session as <lowercase-task-id>-<profile>.",
		"Esc: closes help/confirm, exits search, or opens discard confirm from dirty form.",
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("14")).
		Padding(0, 1).
		Width(max(width-2, 20)).
		Render(strings.Join(copy, "\n"))
}

func (m model) renderConfirmOverlay(width int) string {
	var body string
	switch m.confirm.kind {
	case confirmKindDiscardEdits:
		body = "Discard unsaved form edits?\nEnter confirms discard · Esc returns to form."
	case confirmKindArchive:
		body = fmt.Sprintf("Archive %s?\nEnter runs cleanup (kill session, cleanup hooks) and marks cleanup_state=archived · Esc cancels.", m.confirm.agentName)
	case confirmKindPrune:
		body = fmt.Sprintf("Hard prune %s?\nEnter runs cleanup, optional work-dir prune, and marks cleanup_state=pruned · Esc cancels.", m.confirm.agentName)
	case confirmKindPruneForce:
		body = fmt.Sprintf("Working dir is non-empty for %s (%s).\nPress f to enable force prune, then Enter. Esc cancels.", m.confirm.agentName, m.confirm.workingDir)
	default:
		body = "Confirm action?\nEnter accepts · Esc cancels."
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("11")).
		Padding(0, 1).
		Width(max(width-2, 20)).
		Render(body)
}

func (m model) renderFormOverlay(width int) string {
	isCreateWizard := m.form.purpose == formPurposeCreateGeneric || m.form.purpose == formPurposeCreateProfile
	title := "Create Agent"
	switch m.form.purpose {
	case formPurposeCreateProfile:
		title = "Create Agent · Profile Path"
	case formPurposeEdit:
		title = "Edit Agent"
	}

	lines := []string{
		title,
		"",
	}

	if isCreateWizard {
		profile := m.formFieldValue("profile_name")
		task := m.formFieldValue("task")
		inferred := inferNameAndSession(task, profile)

		stageProfile := "1) Profile: " + profile
		stageTask := "2) Task: " + task
		if m.form.active == 0 {
			stageProfile = focusedStyle().Render(stageProfile)
		}
		if m.form.active == 1 {
			stageTask = focusedStyle().Render(stageTask)
		}
		lines = append(lines, stageProfile)
		if errText, ok := m.form.errors["profile_name"]; ok {
			lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("  ! "+errText))
		}
		lines = append(lines, stageTask)
		if errText, ok := m.form.errors["task"]; ok {
			lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("  ! "+errText))
		}

		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Inferred name: %s", inferred))
		lines = append(lines, fmt.Sprintf("Inferred tmux session: %s", inferred))
		lines = append(lines, "")
		lines = append(lines, "Step 1: left/right or j/k select profile, Enter continue")
		lines = append(lines, "Step 2: type task, Enter create")
		lines = append(lines, "Esc cancel")
	} else {
		for idx, field := range m.form.fields {
			required := ""
			if field.required {
				required = " *"
			}
			label := fmt.Sprintf("%s%s: %s", field.label, required, field.value)
			if idx == m.form.active {
				label = focusedStyle().Render(label)
			}
			lines = append(lines, label)
			if errText, ok := m.form.errors[field.key]; ok {
				lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("  ! "+errText))
			}
		}
		lines = append(lines, "", "Enter save · Esc cancel · Tab/j/k move · left/right cycle status")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("13")).
		Padding(0, 1).
		Width(max(width-2, 20)).
		Render(strings.Join(lines, "\n"))
}

func (m model) currentSelection() (generated.Agent, bool) {
	if len(m.columns) == 0 || m.focused < 0 || m.focused >= len(m.columns) {
		return generated.Agent{}, false
	}
	row := m.selected[m.focused]
	if row < 0 || row >= len(m.columns[m.focused].cards) {
		return generated.Agent{}, false
	}
	return m.columns[m.focused].cards[row], true
}

func (m model) modeLabel() string {
	switch m.mode {
	case modeNormal:
		return "Normal"
	case modeSearch:
		return "Search"
	case modeForm:
		return "Form"
	case modeConfirm:
		return "Confirm"
	case modeHelp:
		return "Help"
	case modeStatusPicker:
		return "Status Picker"
	default:
		return strconv.Itoa(int(m.mode))
	}
}

func focusedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
}

func (m model) agentRuntimeState(agent generated.Agent) string {
	switch {
	case m.isDead(agent):
		return "DEAD (tmux session missing)"
	case m.isStale(agent):
		return "STALE (running with old update timestamp)"
	default:
		return "LIVE"
	}
}

func (m model) runtimeBadge(agent generated.Agent) string {
	switch {
	case m.isDead(agent):
		return " [DEAD]"
	case m.isStale(agent):
		return " [STALE]"
	default:
		return ""
	}
}

func nullStringValue(value sql.NullString) string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return "-"
	}
	return value.String
}

func nullTimeValue(value sql.NullTime) string {
	if !value.Valid {
		return "-"
	}
	return value.Time.Format(time.RFC3339)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
