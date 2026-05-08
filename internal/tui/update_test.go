package tui

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

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
		confirm: confirmState{returnMode: modeNormal, kind: confirmKindArchive},
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

func TestCreateFormSubmitsAndFocusesNewCard(t *testing.T) {
	m := model{
		mode:     modeNormal,
		agents:   []generated.Agent{},
		columns:  buildColumns(nil, ""),
		focused:  0,
		selected: []int{headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
	}

	m = m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.mode != modeForm {
		t.Fatalf("expected form mode, got %v", m.mode)
	}

	profile := m.form.profileOptions[0]
	m.setFormFieldValue("profile_name", profile)
	m = m.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	m.setFormFieldValue("task", "KAI-123")
	m.formDirty = true

	saved := m.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	if saved.mode != modeNormal {
		t.Fatalf("expected mode normal after save, got %v", saved.mode)
	}
	if len(saved.agents) != 1 {
		t.Fatalf("expected one agent after create, got %d", len(saved.agents))
	}
	expected := inferNameAndSession("KAI-123", profile)
	if saved.agents[0].Name != expected {
		t.Fatalf("expected inferred name, got %q", saved.agents[0].Name)
	}
	if saved.agents[0].TmuxSession != expected {
		t.Fatalf("expected inferred tmux session, got %q", saved.agents[0].TmuxSession)
	}
	if saved.focused != 0 {
		t.Fatalf("expected focus on idle column, got %d", saved.focused)
	}
}

func TestEditFormPreloadsAndUpdatesSelection(t *testing.T) {
	agents := []generated.Agent{
		{ID: 1, Name: "alpha", Status: "idle", TmuxSession: "alpha"},
	}
	m := model{
		mode:     modeNormal,
		agents:   agents,
		columns:  buildColumns(agents, ""),
		focused:  0,
		selected: []int{0, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
	}

	edit := m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if edit.mode != modeForm {
		t.Fatalf("expected edit form mode, got %v", edit.mode)
	}
	if got := edit.formFieldValue("name"); got != "alpha" {
		t.Fatalf("expected preloaded name alpha, got %q", got)
	}

	edit.setFormFieldValue("name", "alpha-updated")
	edit.formDirty = true
	updated := edit.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.agents[0].Name != "alpha-updated" {
		t.Fatalf("expected updated name, got %q", updated.agents[0].Name)
	}
	if updated.focused != 0 || updated.selected[0] != 0 {
		t.Fatalf("expected selection to stay on updated card")
	}
}

func TestCreateWizardRejectsDuplicateInferredSession(t *testing.T) {
	profile := availableProfiles()[0]
	agents := []generated.Agent{
		{ID: 1, Name: "alpha", Status: "idle", TmuxSession: inferNameAndSession("KAI-123", profile)},
	}
	m := model{
		mode:     modeNormal,
		agents:   agents,
		columns:  buildColumns(agents, ""),
		focused:  0,
		selected: []int{0, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
	}

	form := m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	form.setFormFieldValue("profile_name", profile)
	form = form.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	form.setFormFieldValue("task", "KAI-123")
	form.formDirty = true

	next := form.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	if next.mode != modeForm {
		t.Fatalf("expected to remain in form mode on validation error")
	}
	if next.form.errors["task"] == "" {
		t.Fatalf("expected duplicate inferred session error")
	}
}

func TestArchiveAndPruneFlows(t *testing.T) {
	agents := []generated.Agent{
		{
			ID:           1,
			Name:         "agent-1",
			Status:       "idle",
			TmuxSession:  "agent-1",
			CleanupState: "active",
			WorkingDir:   sql.NullString{String: "/tmp/work", Valid: true},
		},
	}
	m := model{
		mode:     modeNormal,
		agents:   agents,
		columns:  buildColumns(agents, ""),
		focused:  0,
		selected: []int{0, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
	}

	archive := m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if archive.confirm.kind != confirmKindArchive {
		t.Fatalf("expected archive confirm kind, got %q", archive.confirm.kind)
	}
	afterArchive := archive.handleConfirmMode(tea.KeyMsg{Type: tea.KeyEnter})
	if len(afterArchive.agents) != 0 {
		t.Fatalf("expected archived agent to disappear from active board")
	}

	pruneModel := model{
		mode:     modeNormal,
		agents:   agents,
		columns:  buildColumns(agents, ""),
		focused:  0,
		selected: []int{0, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
	}
	pruneConfirm := pruneModel.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("D")})
	if pruneConfirm.confirm.kind != confirmKindPruneForce {
		t.Fatalf("expected force prune confirm kind, got %q", pruneConfirm.confirm.kind)
	}
	pruneConfirm = pruneConfirm.handleConfirmMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	afterPrune := pruneConfirm.handleConfirmMode(tea.KeyMsg{Type: tea.KeyEnter})
	if len(afterPrune.agents) != 0 {
		t.Fatalf("expected hard prune to remove agent row")
	}
}

func TestEnterAttachWarnsWhenSessionMissing(t *testing.T) {
	agents := []generated.Agent{
		{ID: 99, Name: "ghost", Status: "running", TmuxSession: "ghost-session"},
	}
	m := model{
		mode:          modeNormal,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       1,
		selected:      []int{headerSelectionRow, 0, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		liveSessions:  map[string]struct{}{},
		nowFn:         time.Now,
	}

	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := nextModel.(model)
	if len(next.toasts) == 0 {
		t.Fatalf("expected warning toast when session is missing")
	}
	last := next.toasts[len(next.toasts)-1]
	if last.severity != toastWarning {
		t.Fatalf("expected warning severity, got %v", last.severity)
	}
	if last.message == "" || !containsAny("session not found", last.message) {
		t.Fatalf("expected actionable missing-session message, got %q", last.message)
	}
}

func TestEnterAttachFailsWhenTmuxUnavailable(t *testing.T) {
	agents := []generated.Agent{
		{ID: 100, Name: "alpha", Status: "running", TmuxSession: "alpha"},
	}
	m := model{
		mode:          modeNormal,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       1,
		selected:      []int{headerSelectionRow, 0, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: false,
		nowFn:         time.Now,
	}

	nextModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next := nextModel.(model)
	if len(next.toasts) == 0 {
		t.Fatalf("expected error toast when tmux is unavailable")
	}
	last := next.toasts[len(next.toasts)-1]
	if last.severity != toastError {
		t.Fatalf("expected error severity, got %v", last.severity)
	}
	if !containsAny("tmux unavailable", last.message) {
		t.Fatalf("expected tmux unavailable message, got %q", last.message)
	}
}

func TestRecoverDeadSessionFailsWhenWorkingDirMissing(t *testing.T) {
	agents := []generated.Agent{
		{
			ID:          201,
			Name:        "ghost",
			Status:      "running",
			TmuxSession: "ghost-session",
			WorkingDir:  sql.NullString{},
		},
	}
	m := model{
		mode:          modeNormal,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       1,
		selected:      []int{headerSelectionRow, 0, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		liveSessions:  map[string]struct{}{},
		nowFn:         time.Now,
	}

	next, cmd := m.recreateSelectedSession()
	if cmd != nil {
		t.Fatalf("expected no command when working dir is missing")
	}
	if next.agents[0].LastError.Valid {
		t.Fatalf("expected no side effects when working dir is missing")
	}
	if len(next.toasts) == 0 {
		t.Fatalf("expected warning toast")
	}
	last := next.toasts[len(next.toasts)-1]
	if !containsAny("working_dir is missing", last.message) {
		t.Fatalf("expected actionable working_dir message, got %q", last.message)
	}
}

func TestRecoverDeadSessionFailsWhenWorkingDirInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := tmpDir + "/not-a-dir.txt"
	if err := os.WriteFile(filePath, []byte("x"), 0o600); err != nil {
		t.Fatalf("failed to create file fixture: %v", err)
	}

	agents := []generated.Agent{
		{
			ID:          202,
			Name:        "ghost",
			Status:      "running",
			TmuxSession: "ghost-session",
			WorkingDir:  sql.NullString{String: filePath, Valid: true},
		},
	}
	m := model{
		mode:          modeNormal,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       1,
		selected:      []int{headerSelectionRow, 0, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		liveSessions:  map[string]struct{}{},
		nowFn:         time.Now,
	}

	next, cmd := m.recreateSelectedSession()
	if cmd != nil {
		t.Fatalf("expected no command when working dir is invalid")
	}
	if next.agents[0].LastError.Valid {
		t.Fatalf("expected no side effects when working dir is invalid")
	}
	if len(next.toasts) == 0 {
		t.Fatalf("expected warning toast")
	}
	last := next.toasts[len(next.toasts)-1]
	if !containsAny("not a directory", last.message) {
		t.Fatalf("expected not-a-directory message, got %q", last.message)
	}
}

func TestRecoverDeadSessionRecreatesAndAttaches(t *testing.T) {
	workingDir := t.TempDir()
	agents := []generated.Agent{
		{
			ID:          203,
			Name:        "ghost",
			Status:      "blocked",
			TmuxSession: "ghost-session",
			WorkingDir:  sql.NullString{String: workingDir, Valid: true},
			LastError:   sql.NullString{String: "previous failure", Valid: true},
		},
	}
	createCalled := false
	attachCalled := false
	m := model{
		mode:          modeNormal,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       2,
		selected:      []int{headerSelectionRow, headerSelectionRow, 0, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		liveSessions:  map[string]struct{}{},
		nowFn:         time.Now,
		createDetachedCmd: func(context.Context, string, string) (*exec.Cmd, error) {
			createCalled = true
			return exec.Command("true"), nil
		},
		attachOrSwitchCmd: func(context.Context, string) (*exec.Cmd, error) {
			attachCalled = true
			return exec.Command("true"), nil
		},
	}

	next, cmd := m.recreateSelectedSession()
	if !createCalled {
		t.Fatalf("expected tmux recreation command to be built")
	}
	if !attachCalled {
		t.Fatalf("expected attach command after successful recreation")
	}
	if cmd == nil {
		t.Fatalf("expected attach command after successful recreation")
	}
	if next.agents[0].Status != "running" {
		t.Fatalf("expected status to move to running, got %q", next.agents[0].Status)
	}
	if next.agents[0].LastError.Valid {
		t.Fatalf("expected recovery to clear last_error")
	}
	if !next.agents[0].LastHeartbeatAt.Valid {
		t.Fatalf("expected recovery to set heartbeat")
	}
	if _, ok := next.liveSessions["ghost-session"]; !ok {
		t.Fatalf("expected recreated session to be marked live")
	}
}

func TestRefreshFailureSetsPersistentBanner(t *testing.T) {
	lockErr := errors.New("database is locked")
	m := model{
		mode:          modeNormal,
		tmuxAvailable: true,
		nowFn:         time.Now,
		loadAgentsFn: func(context.Context) ([]generated.Agent, error) {
			return nil, lockErr
		},
		listSessionsFn: func(context.Context) ([]string, error) {
			return []string{"alpha"}, nil
		},
	}

	msg := m.refreshData()
	nextModel, _ := m.Update(msg)
	next := nextModel.(model)
	if next.banner == "" {
		t.Fatalf("expected persistent banner on refresh failure")
	}
	if !containsAny("locked", next.banner) {
		t.Fatalf("expected lock-related banner message, got %q", next.banner)
	}
}

func TestEnterAttachBuildsExecCommandWhenSessionLive(t *testing.T) {
	agents := []generated.Agent{
		{ID: 101, Name: "live", Status: "running", TmuxSession: "live"},
	}
	m := model{
		mode:          modeNormal,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       1,
		selected:      []int{headerSelectionRow, 0, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		liveSessions:  map[string]struct{}{"live": {}},
		nowFn:         time.Now,
		attachOrSwitchCmd: func(context.Context, string) (*exec.Cmd, error) {
			return exec.Command("true"), nil
		},
	}

	_, cmd := m.attachSelectedSession()
	if cmd == nil {
		t.Fatalf("expected exec command when session is live")
	}
}
