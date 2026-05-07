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
	if len(m.notices) > 0 {
		sections = append(sections, m.renderNotices(width))
	}
	if m.showEmpty {
		sections = append(sections, m.renderEmptyState(width))
	}
	sections = append(sections, board, m.renderFooter(width))
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
			fmt.Sprintf("Cleanup: %s", agent.CleanupState),
			fmt.Sprintf("Updated: %s", agent.UpdatedAt.Format(time.RFC3339)),
		)
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

func (m model) renderNotices(width int) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Width(max(width-2, 20)).
		Render(strings.Join(m.notices, " · "))
}

func (m model) renderFooter(width int) string {
	footer := "h/l or arrows move columns  j/k or arrows move rows  / search  ? help  n form  d confirm  q quit"
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
			if card.LastError.Valid && strings.TrimSpace(card.LastError.String) != "" {
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

func (m model) renderHelpOverlay(width int) string {
	copy := []string{
		"Help",
		"",
		"Navigation: h/l or arrows move columns; j/k or arrows move rows.",
		"Modes: / enters search, n/e opens form, d opens confirm, ? toggles help.",
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
	case confirmKindDelete:
		body = "Delete selected agent?\nEnter confirms delete (not yet implemented) · Esc cancels."
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
	default:
		return strconv.Itoa(int(m.mode))
	}
}

func focusedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
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
