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
