package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/grahamdaw/yaama/internal/db/generated"
)

func TestHorizontalMovePreservesAndClampsRow(t *testing.T) {
	m := model{
		mode:    modeNormal,
		columns: newStatusColumns(),
		focused: 0,
		selected: []int{
			2, 0, headerSelectionRow, headerSelectionRow, headerSelectionRow,
		},
	}
	m.columns[0].cards = []generated.Agent{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	m.columns[1].cards = []generated.Agent{{Name: "x"}}

	next := m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRight})
	if next.focused != 1 {
		t.Fatalf("expected focused column 1, got %d", next.focused)
	}
	if next.selected[1] != 0 {
		t.Fatalf("expected clamped row 0 in shorter column, got %d", next.selected[1])
	}
}

func TestEmptyColumnSelectionLandsOnHeader(t *testing.T) {
	m := model{
		mode:    modeNormal,
		columns: newStatusColumns(),
		focused: 0,
		selected: []int{
			0, 0, headerSelectionRow, headerSelectionRow, headerSelectionRow,
		},
	}
	m.columns[0].cards = []generated.Agent{{Name: "a"}}
	m.columns[1].cards = nil

	next := m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRight})
	if next.focused != 1 {
		t.Fatalf("expected focused column 1, got %d", next.focused)
	}
	if next.selected[1] != headerSelectionRow {
		t.Fatalf("expected header selection for empty column, got %d", next.selected[1])
	}
}

func TestEscInDirtyFormOpensDiscardConfirm(t *testing.T) {
	m := model{
		mode:      modeForm,
		formDirty: true,
	}

	next := m.handleFormMode(tea.KeyMsg{Type: tea.KeyEsc})
	if next.mode != modeConfirm {
		t.Fatalf("expected confirm mode, got %v", next.mode)
	}
	if next.confirm.kind != confirmKindDiscardEdits {
		t.Fatalf("expected discard confirm kind, got %q", next.confirm.kind)
	}
	if next.confirm.returnMode != modeForm {
		t.Fatalf("expected return mode form, got %v", next.confirm.returnMode)
	}
}

func TestEscClosesHelpConfirmAndSearch(t *testing.T) {
	help := model{mode: modeHelp}
	afterHelp := help.handleHelpMode(tea.KeyMsg{Type: tea.KeyEsc})
	if afterHelp.mode != modeNormal {
		t.Fatalf("expected help esc to return normal mode, got %v", afterHelp.mode)
	}

	confirm := model{
		mode:    modeConfirm,
		confirm: confirmState{returnMode: modeNormal, kind: confirmKindDelete},
	}
	afterConfirm := confirm.handleConfirmMode(tea.KeyMsg{Type: tea.KeyEsc})
	if afterConfirm.mode != modeNormal {
		t.Fatalf("expected confirm esc to return normal mode, got %v", afterConfirm.mode)
	}

	search := model{
		mode:   modeSearch,
		search: "run",
		agents: []generated.Agent{
			{Name: "runner", Status: "running"},
			{Name: "idle-one", Status: "idle"},
		},
	}
	search = search.rebuildColumns()
	afterSearch := search.handleSearchMode(tea.KeyMsg{Type: tea.KeyEsc})
	if afterSearch.mode != modeNormal {
		t.Fatalf("expected search esc to return normal mode, got %v", afterSearch.mode)
	}
	if afterSearch.search != "" {
		t.Fatalf("expected search esc to clear search query, got %q", afterSearch.search)
	}
}

func TestSearchFiltersAcrossNameTaskBranchAndSession(t *testing.T) {
	agents := []generated.Agent{
		{ID: 1, Name: "alpha", Status: "idle"},
		{ID: 2, Name: "beta", Status: "running"},
		{ID: 3, Name: "gamma", Status: "blocked"},
		{ID: 4, Name: "delta", Status: "review"},
	}
	agents[1].Task.Valid = true
	agents[1].Task.String = "pipeline"
	agents[2].Branch.Valid = true
	agents[2].Branch.String = "feature/x"
	agents[3].TmuxSession = "tmux-dev"

	checks := []struct {
		query    string
		expected int64
	}{
		{query: "alpha", expected: 1},
		{query: "pipeline", expected: 2},
		{query: "feature/x", expected: 3},
		{query: "tmux-dev", expected: 4},
	}

	for _, check := range checks {
		filtered := filterAgents(agents, check.query)
		if len(filtered) != 1 {
			t.Fatalf("expected one match for %q, got %d", check.query, len(filtered))
		}
		if filtered[0].ID != check.expected {
			t.Fatalf("expected id %d for %q, got %d", check.expected, check.query, filtered[0].ID)
		}
	}
}

func TestStatusPickerAppliesSelectedStatus(t *testing.T) {
	agents := []generated.Agent{
		{ID: 11, Name: "agent-1", Status: "idle"},
	}
	m := model{
		mode:     modeNormal,
		agents:   agents,
		columns:  buildColumns(agents, ""),
		focused:  0,
		selected: []int{0, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
	}

	afterOpen := m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if afterOpen.mode != modeStatusPicker {
		t.Fatalf("expected status picker mode, got %v", afterOpen.mode)
	}

	afterPick := afterOpen.handleStatusPickerMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	afterApply := afterPick.handleStatusPickerMode(tea.KeyMsg{Type: tea.KeyEnter})
	if afterApply.mode != modeNormal {
		t.Fatalf("expected to return to normal mode, got %v", afterApply.mode)
	}

	if afterApply.agents[0].Status != "blocked" {
		t.Fatalf("expected status blocked after picker apply, got %q", afterApply.agents[0].Status)
	}
	if afterApply.focused != 2 {
		t.Fatalf("expected focus on blocked column index 2, got %d", afterApply.focused)
	}
	if afterApply.selected[2] != 0 {
		t.Fatalf("expected selected row 0 in blocked column, got %d", afterApply.selected[2])
	}
}

func TestReverseStatusCycleShortcut(t *testing.T) {
	agents := []generated.Agent{
		{ID: 29, Name: "agent-2", Status: "idle"},
	}
	m := model{
		mode:     modeNormal,
		agents:   agents,
		columns:  buildColumns(agents, ""),
		focused:  0,
		selected: []int{0, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
	}

	after := m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	if after.agents[0].Status != "done" {
		t.Fatalf("expected reverse cycle to move idle -> done, got %q", after.agents[0].Status)
	}
	if after.focused != 4 {
		t.Fatalf("expected focus to move to done column index 4, got %d", after.focused)
	}
}
