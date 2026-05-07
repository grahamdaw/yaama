package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	width := max(m.width, 80)
	if m.width > 0 {
		width = m.width
	}

	header := lipgloss.NewStyle().Bold(true).Padding(0, 1).
		Render("Agent Board")
	stats := lipgloss.NewStyle().Faint(true).Render("5 statuses · placeholder data")

	topBar := lipgloss.JoinHorizontal(lipgloss.Left, header, "  ", stats)
	topBar = lipgloss.NewStyle().Width(width).Render(topBar)

	board := m.renderColumns(width)

	sections := []string{topBar}
	if len(m.notices) > 0 {
		sections = append(sections, m.renderNotices(width))
	}
	if m.showEmpty {
		sections = append(sections, m.renderEmptyState(width))
	}
	sections = append(sections, board, m.renderFooter(width))

	return strings.Join(sections, "\n")
}

func (m model) renderColumns(totalWidth int) string {
	if len(m.columns) == 0 {
		return ""
	}

	usableWidth := max(totalWidth-2, 20)
	columnWidth := max((usableWidth/len(m.columns))-1, 12)

	columnStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Width(columnWidth)

	rendered := make([]string, 0, len(m.columns))
	for _, col := range m.columns {
		body := "(empty)"
		if len(col.cards) > 0 {
			body = strings.Join(col.cards, "\n")
		}

		rendered = append(rendered, columnStyle.Render(fmt.Sprintf("%s\n\n%s", col.title, body)))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
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
	footer := "Enter attach  n new  e edit  d delete  s status  r refresh  q quit"
	return lipgloss.NewStyle().
		Faint(true).
		Width(max(width-2, 20)).
		Render(footer)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
