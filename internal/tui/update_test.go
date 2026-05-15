package tui

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/grahamdaw/yaama/internal/db/generated"
	"github.com/grahamdaw/yaama/internal/profile"
	"github.com/grahamdaw/yaama/internal/tmux"
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
		loadProfileFn: func(string) (profile.Config, error) {
			return profile.Config{
				Agent: profile.AgentConfig{Command: "codex"},
			}, nil
		},
		resolveRuntimeFn: func(profile.Config, string, string, string) (profile.RuntimeValues, error) {
			return profile.RuntimeValues{
				WorkingDir:   "/tmp/work",
				Branch:       "feat/kai-123",
				AgentCommand: []string{"codex"},
			}, nil
		},
		ensureWorktreeFn: func(context.Context, string, string, string) (string, error) {
			return "/tmp/worktree", nil
		},
		bootstrapSession: func(context.Context, tmux.BootstrapSpec) error {
			return nil
		},
	}

	m = m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.mode != modeForm {
		t.Fatalf("expected form mode, got %v", m.mode)
	}

	profile := m.form.profileOptions[0]
	m.setFormFieldValue("profile_name", profile)
	m = m.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	m.setFormFieldValue("task", "KAI-123")
	m = m.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	m.setFormFieldValue("branch", "feat/kai-123")
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

func TestCreateFormPersistsResolvedRuntimeMetadata(t *testing.T) {
	loadedProfile := profile.Config{
		Name: "dev",
		Agent: profile.AgentConfig{
			Command: "codex",
			Args:    []string{"--model", "gpt-5.3-codex"},
		},
		Tmux: profile.TmuxConfig{
			StartupWindow: "agent",
			Windows: []profile.TmuxWindow{
				{
					Name:  "ops",
					Focus: true,
					Panes: []profile.TmuxPane{{Cwd: "."}},
				},
			},
		},
	}
	bootstrapCalls := 0
	var bootstrapSpec tmux.BootstrapSpec

	m := model{
		mode:     modeNormal,
		agents:   []generated.Agent{},
		columns:  buildColumns(nil, ""),
		focused:  0,
		selected: []int{headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		loadProfileFn: func(string) (profile.Config, error) {
			return loadedProfile, nil
		},
		resolveRuntimeFn: func(profile.Config, string, string, string) (profile.RuntimeValues, error) {
			return profile.RuntimeValues{
				WorkingDir:   "/tmp/runtime/work",
				Branch:       "feat/kai-123",
				AgentCommand: []string{"codex", "--model", "gpt-5.3-codex"},
			}, nil
		},
		ensureWorktreeFn: func(context.Context, string, string, string) (string, error) {
			return "/tmp/runtime/worktree", nil
		},
		bootstrapSession: func(_ context.Context, spec tmux.BootstrapSpec) error {
			bootstrapCalls++
			bootstrapSpec = spec
			return nil
		},
	}

	m = m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	profileName := m.form.profileOptions[0]
	m.setFormFieldValue("profile_name", profileName)
	m = m.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	m.setFormFieldValue("task", "KAI-123")
	m = m.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	m.setFormFieldValue("branch", "feat/kai-123")
	m.formDirty = true

	saved := m.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	if saved.mode != modeNormal {
		t.Fatalf("expected mode normal after save, got %v", saved.mode)
	}
	if len(saved.agents) != 1 {
		t.Fatalf("expected one saved agent, got %d", len(saved.agents))
	}
	created := saved.agents[0]
	if !created.WorkingDir.Valid || created.WorkingDir.String != "/tmp/runtime/worktree" {
		t.Fatalf("expected persisted working_dir, got %#v", created.WorkingDir)
	}
	if !created.Branch.Valid || created.Branch.String != "feat/kai-123" {
		t.Fatalf("expected persisted branch, got %#v", created.Branch)
	}
	if bootstrapCalls != 1 {
		t.Fatalf("expected one bootstrap call, got %d", bootstrapCalls)
	}
	if bootstrapSpec.SessionName != created.TmuxSession {
		t.Fatalf("expected bootstrap session %q, got %q", created.TmuxSession, bootstrapSpec.SessionName)
	}
	if bootstrapSpec.WorkingDir != "/tmp/runtime/worktree" {
		t.Fatalf("expected bootstrap working dir /tmp/runtime/worktree, got %q", bootstrapSpec.WorkingDir)
	}
	if bootstrapSpec.AgentWindow != created.TmuxSession {
		t.Fatalf("expected bootstrap default window %q, got %q", created.TmuxSession, bootstrapSpec.AgentWindow)
	}
	if len(bootstrapSpec.Windows) != 1 || bootstrapSpec.Windows[0].Name != "ops" {
		t.Fatalf("expected one additional window named ops, got %#v", bootstrapSpec.Windows)
	}
	if want := []string{"codex", "--model", "gpt-5.3-codex"}; !reflect.DeepEqual(bootstrapSpec.AgentCommand, want) {
		t.Fatalf("unexpected bootstrap command: %#v", bootstrapSpec.AgentCommand)
	}
}

func TestCreateFormShowsErrorWhenProfileLoadFails(t *testing.T) {
	m := model{
		mode:     modeNormal,
		agents:   []generated.Agent{},
		columns:  buildColumns(nil, ""),
		focused:  0,
		selected: []int{headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		loadProfileFn: func(string) (profile.Config, error) {
			return profile.Config{}, errors.New("profile file is invalid")
		},
	}

	m = m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	profileName := m.form.profileOptions[0]
	m.setFormFieldValue("profile_name", profileName)
	m = m.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	m.setFormFieldValue("task", "KAI-123")
	m = m.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	m.setFormFieldValue("branch", "feat/kai-123")
	m.formDirty = true

	next := m.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	if next.mode != modeForm {
		t.Fatalf("expected modeForm after profile load failure, got %v", next.mode)
	}
	if len(next.agents) != 0 {
		t.Fatalf("expected no created agents, got %d", len(next.agents))
	}
	if len(next.toasts) == 0 {
		t.Fatalf("expected validation toast after profile load failure")
	}
	last := next.toasts[len(next.toasts)-1]
	if !containsAny("profile file is invalid", last.message) {
		t.Fatalf("expected profile load failure message, got %q", last.message)
	}
}

func TestCreateWizardUpDownMovesFieldsWithoutChangingProfile(t *testing.T) {
	m := model{
		mode: modeForm,
		form: formState{
			purpose: formPurposeCreateProfile,
			fields: []formField{
				{key: "profile_name", value: "default"},
				{key: "task", value: ""},
				{key: "branch", value: ""},
			},
			active:         0,
			profileOptions: []string{"default", "dev"},
		},
	}

	afterDown := m.handleFormMode(tea.KeyMsg{Type: tea.KeyDown})
	if afterDown.form.active != 1 {
		t.Fatalf("expected down to move to next field, got active=%d", afterDown.form.active)
	}
	if got := afterDown.form.fields[0].value; got != "default" {
		t.Fatalf("expected profile unchanged on down, got %q", got)
	}

	afterUp := afterDown.handleFormMode(tea.KeyMsg{Type: tea.KeyUp})
	if afterUp.form.active != 0 {
		t.Fatalf("expected up to move back to profile field, got active=%d", afterUp.form.active)
	}
	if got := afterUp.form.fields[0].value; got != "default" {
		t.Fatalf("expected profile unchanged on up, got %q", got)
	}
}

func TestCreateWizardJKCyclesProfileOnlyOnProfileField(t *testing.T) {
	m := model{
		mode: modeForm,
		form: formState{
			purpose: formPurposeCreateProfile,
			fields: []formField{
				{key: "profile_name", value: "default"},
				{key: "task", value: ""},
				{key: "branch", value: ""},
			},
			active:         0,
			profileOptions: []string{"default", "dev"},
		},
	}

	afterJ := m.handleFormMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if got := afterJ.form.fields[0].value; got != "dev" {
		t.Fatalf("expected j to cycle profile forward, got %q", got)
	}
	if afterJ.form.active != 0 {
		t.Fatalf("expected active field to stay on profile, got %d", afterJ.form.active)
	}

	afterK := afterJ.handleFormMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if got := afterK.form.fields[0].value; got != "default" {
		t.Fatalf("expected k to cycle profile backward, got %q", got)
	}

	afterK.form.active = 1
	typedJ := afterK.handleFormMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if typedJ.form.active != 1 {
		t.Fatalf("expected non-profile j to not change active field, got %d", typedJ.form.active)
	}
	if got := typedJ.form.fields[1].value; got != "j" {
		t.Fatalf("expected non-profile j to append rune, got %q", got)
	}
}

func TestFormModeSlashAppendsRune(t *testing.T) {
	m := model{
		mode: modeForm,
		form: formState{
			purpose: formPurposeCreateProfile,
			fields: []formField{
				{key: "profile_name", value: "default"},
				{key: "task", value: "feat"},
				{key: "branch", value: ""},
			},
			active:         1,
			profileOptions: []string{"default", "dev"},
		},
	}

	next := m.handleFormMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	if next.form.active != 1 {
		t.Fatalf("expected / not to move active field, got %d", next.form.active)
	}
	if got := next.form.fields[1].value; got != "feat/" {
		t.Fatalf("expected / to append to active field, got %q", got)
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
	form = form.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	form.setFormFieldValue("branch", "feat/kai-123")
	form.formDirty = true

	next := form.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	if next.mode != modeForm {
		t.Fatalf("expected to remain in form mode on validation error")
	}
	if next.form.errors["task"] == "" {
		t.Fatalf("expected duplicate inferred session error")
	}
}

func TestCreateWizardRequiresBranchInput(t *testing.T) {
	m := model{
		mode:     modeNormal,
		agents:   []generated.Agent{},
		columns:  buildColumns(nil, ""),
		focused:  0,
		selected: []int{headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
	}

	form := m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	form.setFormFieldValue("profile_name", availableProfiles()[0])
	form = form.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	form.setFormFieldValue("task", "KAI-123")
	form = form.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	form.formDirty = true

	next := form.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	if next.mode != modeForm {
		t.Fatalf("expected to remain in form mode on validation error")
	}
	if next.form.errors["branch"] == "" {
		t.Fatalf("expected branch required validation error")
	}
}

func TestCreateWizardRejectsUnsafeBranchInput(t *testing.T) {
	m := model{
		mode:     modeNormal,
		agents:   []generated.Agent{},
		columns:  buildColumns(nil, ""),
		focused:  0,
		selected: []int{headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
	}

	form := m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	form.setFormFieldValue("profile_name", availableProfiles()[0])
	form = form.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	form.setFormFieldValue("task", "KAI-123")
	form = form.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	form.setFormFieldValue("branch", "bad branch")
	form.formDirty = true

	next := form.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	if next.mode != modeForm {
		t.Fatalf("expected to remain in form mode on validation error")
	}
	if next.form.errors["branch"] == "" {
		t.Fatalf("expected branch safety validation error")
	}
}

func TestCreateFormShowsErrorWhenWorktreeProvisionFails(t *testing.T) {
	m := model{
		mode:     modeNormal,
		agents:   []generated.Agent{},
		columns:  buildColumns(nil, ""),
		focused:  0,
		selected: []int{headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		loadProfileFn: func(string) (profile.Config, error) {
			return profile.Config{
				Agent: profile.AgentConfig{Command: "codex"},
			}, nil
		},
		resolveRuntimeFn: func(profile.Config, string, string, string) (profile.RuntimeValues, error) {
			return profile.RuntimeValues{
				WorkingDir:   "/tmp/repo",
				Branch:       "feat/kai-123",
				AgentCommand: []string{"codex"},
			}, nil
		},
		ensureWorktreeFn: func(context.Context, string, string, string) (string, error) {
			return "", errors.New("resolved path \"/tmp/repo\" is not a git repository")
		},
	}

	m = m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	profileName := m.form.profileOptions[0]
	m.setFormFieldValue("profile_name", profileName)
	m = m.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	m.setFormFieldValue("task", "KAI-123")
	m = m.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	m.setFormFieldValue("branch", "feat/kai-123")
	m.formDirty = true

	next := m.handleFormMode(tea.KeyMsg{Type: tea.KeyEnter})
	if next.mode != modeForm {
		t.Fatalf("expected modeForm after worktree provisioning failure, got %v", next.mode)
	}
	if len(next.agents) != 0 {
		t.Fatalf("expected no created agents, got %d", len(next.agents))
	}
	if len(next.toasts) == 0 {
		t.Fatalf("expected validation toast after git repository failure")
	}
	last := next.toasts[len(next.toasts)-1]
	if !containsAny("not a git repository", last.message) {
		t.Fatalf("expected git repository validation error, got %q", last.message)
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
		mode:          modeNormal,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       0,
		selected:      []int{0, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		killSessionFn: func(context.Context, string) error { return nil },
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
		mode:          modeNormal,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       0,
		selected:      []int{0, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		killSessionFn: func(context.Context, string) error { return nil },
		removeWorktreeFn: func(context.Context, string) error {
			return nil
		},
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

func TestPruneStopsWhenSessionKillFails(t *testing.T) {
	agents := []generated.Agent{
		{
			ID:           12,
			Name:         "agent-1",
			Status:       "running",
			TmuxSession:  "agent-1",
			CleanupState: "active",
		},
	}
	m := model{
		mode:          modeConfirm,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       1,
		selected:      []int{headerSelectionRow, 0, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		confirm: confirmState{
			kind:      confirmKindPrune,
			agentID:   12,
			agentName: "agent-1",
		},
		killSessionFn: func(context.Context, string) error {
			return errors.New("kill failed")
		},
	}

	next := m.applyPrune()
	if len(next.agents) != 1 {
		t.Fatalf("expected prune to stop without removing row")
	}
	if next.agents[0].CleanupState != "active" {
		t.Fatalf("expected cleanup_state to stay active, got %q", next.agents[0].CleanupState)
	}
	if !next.agents[0].LastError.Valid || !containsAny("tmux cleanup failed", next.agents[0].LastError.String) {
		t.Fatalf("expected last_error to persist tmux cleanup failure, got %#v", next.agents[0].LastError)
	}
}

func TestPruneStopsWhenWorkDirPruneFails(t *testing.T) {
	agents := []generated.Agent{
		{
			ID:           13,
			Name:         "agent-2",
			Status:       "running",
			TmuxSession:  "agent-2",
			CleanupState: "active",
			Branch:       sql.NullString{String: "feat/a", Valid: true},
			WorkingDir:   sql.NullString{String: "/tmp/work", Valid: true},
		},
	}
	m := model{
		mode:          modeConfirm,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       1,
		selected:      []int{headerSelectionRow, 0, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		confirm: confirmState{
			kind:      confirmKindPrune,
			agentID:   13,
			agentName: "agent-2",
			force:     true,
		},
		killSessionFn: func(context.Context, string) error {
			return nil
		},
		removeWorktreeFn: func(context.Context, string) error {
			return errors.New("adapter failed")
		},
	}

	next := m.applyPrune()
	if len(next.agents) != 1 {
		t.Fatalf("expected prune to stop before final transition")
	}
	if next.agents[0].CleanupState != "active" {
		t.Fatalf("expected cleanup_state active on git worktree remove failure, got %q", next.agents[0].CleanupState)
	}
	if !next.agents[0].LastError.Valid || !containsAny("git worktree remove failed", next.agents[0].LastError.String) {
		t.Fatalf("expected persisted git worktree remove failure in last_error, got %#v", next.agents[0].LastError)
	}
}

func TestArchivePersistsScriptFailureAndStillArchives(t *testing.T) {
	workingDir := t.TempDir()
	agents := []generated.Agent{
		{
			ID:           14,
			Name:         "agent-3",
			Status:       "idle",
			TmuxSession:  "agent-3",
			CleanupState: "active",
			ProfileName:  sql.NullString{String: "dev", Valid: true},
			WorkingDir:   sql.NullString{String: workingDir, Valid: true},
		},
	}
	m := model{
		mode:          modeConfirm,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       0,
		selected:      []int{0, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		confirm: confirmState{
			kind:      confirmKindArchive,
			agentID:   14,
			agentName: "agent-3",
		},
		killSessionFn: func(context.Context, string) error {
			return nil
		},
		loadProfileFn: func(string) (profile.Config, error) {
			return profile.Config{
				Scripts: profile.ScriptsConfig{Cleanup: []string{"cleanup-one", "cleanup-two"}},
			}, nil
		},
		runCleanupHookFn: func(context.Context, string, string, string) error {
			return errors.New("hook failed")
		},
	}

	next := m.applyArchive()
	if len(next.agents) != 0 {
		t.Fatalf("expected archived row to leave active board")
	}
	if len(next.toasts) == 0 {
		t.Fatalf("expected warning toast for cleanup hook failure")
	}
	last := next.toasts[len(next.toasts)-1]
	if !containsAny("script hooks failed", last.message) {
		t.Fatalf("expected cleanup script failure warning, got %q", last.message)
	}
}

func TestPruneRetrySucceedsAfterFailure(t *testing.T) {
	agents := []generated.Agent{
		{
			ID:           15,
			Name:         "agent-4",
			Status:       "running",
			TmuxSession:  "agent-4",
			CleanupState: "active",
			Branch:       sql.NullString{String: "feat/retry", Valid: true},
			WorkingDir:   sql.NullString{String: "/tmp/retry", Valid: true},
		},
	}
	m := model{
		mode:          modeConfirm,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       1,
		selected:      []int{headerSelectionRow, 0, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		confirm: confirmState{
			kind:      confirmKindPrune,
			agentID:   15,
			agentName: "agent-4",
			force:     true,
		},
		killSessionFn: func(context.Context, string) error {
			return nil
		},
		removeWorktreeFn: func(context.Context, string) error {
			return errors.New("transient failure")
		},
	}

	first := m.applyPrune()
	if len(first.agents) != 1 {
		t.Fatalf("expected first prune attempt to keep row for retry")
	}
	if !first.agents[0].LastError.Valid {
		t.Fatalf("expected first prune attempt to persist last_error")
	}

	first.removeWorktreeFn = func(context.Context, string) error { return nil }
	first.mode = modeConfirm
	first.confirm = confirmState{
		kind:      confirmKindPrune,
		agentID:   15,
		agentName: "agent-4",
		force:     true,
	}
	second := first.applyPrune()
	if len(second.agents) != 0 {
		t.Fatalf("expected successful retry to mark pruned and remove from active board")
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
	bootstrapCalled := false
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
		bootstrapSession: func(_ context.Context, spec tmux.BootstrapSpec) error {
			bootstrapCalled = true
			if spec.SessionName != "ghost-session" {
				t.Fatalf("expected session name passthrough, got %q", spec.SessionName)
			}
			if spec.AgentCommand != nil {
				t.Fatalf("recovery must not relaunch agent command, got %v", spec.AgentCommand)
			}
			return nil
		},
		attachOrSwitchCmd: func(context.Context, string) (*exec.Cmd, error) {
			attachCalled = true
			return exec.Command("true"), nil
		},
	}

	next, cmd := m.recreateSelectedSession()
	if !bootstrapCalled {
		t.Fatalf("expected tmux bootstrap to be invoked during recovery")
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

func TestRecoverDeadSessionAppliesProfileLayoutWithoutAgentCommand(t *testing.T) {
	workingDir := t.TempDir()
	agents := []generated.Agent{
		{
			ID:          301,
			Name:        "ghost",
			Status:      "blocked",
			TmuxSession: "ghost-session",
			WorkingDir:  sql.NullString{String: workingDir, Valid: true},
			ProfileName: sql.NullString{String: "dev", Valid: true},
		},
	}
	var captured tmux.BootstrapSpec
	m := model{
		mode:          modeNormal,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       2,
		selected:      []int{headerSelectionRow, headerSelectionRow, 0, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		liveSessions:  map[string]struct{}{},
		nowFn:         time.Now,
		loadProfileFn: func(name string) (profile.Config, error) {
			cfg := profile.Config{Name: name}
			cfg.Tmux.Windows = []profile.TmuxWindow{{Name: "ops", Focus: true}}
			cfg.Scripts.AfterStart = []string{"echo setup"}
			cfg.Agent.Command = "codex"
			cfg.Agent.Args = []string{"--model", "gpt-5.3-codex"}
			return cfg, nil
		},
		bootstrapSession: func(_ context.Context, spec tmux.BootstrapSpec) error {
			captured = spec
			return nil
		},
		attachOrSwitchCmd: func(context.Context, string) (*exec.Cmd, error) {
			return exec.Command("true"), nil
		},
	}

	if _, cmd := m.recreateSelectedSession(); cmd == nil {
		t.Fatalf("expected attach command after successful recovery")
	}
	if captured.SessionName != "ghost-session" {
		t.Fatalf("expected session name passthrough, got %q", captured.SessionName)
	}
	if captured.AgentCommand != nil {
		t.Fatalf("recovery must not relaunch agent command, got %v", captured.AgentCommand)
	}
	if len(captured.Windows) != 1 || captured.Windows[0].Name != "ops" {
		t.Fatalf("expected profile windows applied to recovery spec, got %+v", captured.Windows)
	}
	if len(captured.AfterStart) != 1 || captured.AfterStart[0] != "echo setup" {
		t.Fatalf("expected after_start hooks applied to recovery spec, got %+v", captured.AfterStart)
	}
}

func TestRecoverDeadSessionFallsBackWhenProfileMissing(t *testing.T) {
	workingDir := t.TempDir()
	agents := []generated.Agent{
		{
			ID:          302,
			Name:        "ghost",
			Status:      "blocked",
			TmuxSession: "ghost-session",
			WorkingDir:  sql.NullString{String: workingDir, Valid: true},
			ProfileName: sql.NullString{String: "removed", Valid: true},
		},
	}
	var captured tmux.BootstrapSpec
	m := model{
		mode:          modeNormal,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       2,
		selected:      []int{headerSelectionRow, headerSelectionRow, 0, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		liveSessions:  map[string]struct{}{},
		nowFn:         time.Now,
		loadProfileFn: func(string) (profile.Config, error) {
			return profile.Config{}, os.ErrNotExist
		},
		bootstrapSession: func(_ context.Context, spec tmux.BootstrapSpec) error {
			captured = spec
			return nil
		},
		attachOrSwitchCmd: func(context.Context, string) (*exec.Cmd, error) {
			return exec.Command("true"), nil
		},
	}

	next, cmd := m.recreateSelectedSession()
	if cmd == nil {
		t.Fatalf("expected attach command after fallback recovery")
	}
	if len(captured.Windows) != 0 {
		t.Fatalf("expected minimal spec when profile is missing, got %+v", captured.Windows)
	}
	if captured.AgentWindow != "ghost-session" {
		t.Fatalf("expected default agent window named after session, got %q", captured.AgentWindow)
	}
	var sawWarning bool
	for _, toast := range next.toasts {
		if toast.severity == toastWarning && containsAny("minimal layout", toast.message) {
			sawWarning = true
		}
	}
	if !sawWarning {
		t.Fatalf("expected minimal-layout warning toast, got %+v", next.toasts)
	}
}

func TestRecoverDeadSessionAbortsOnProfileParseError(t *testing.T) {
	workingDir := t.TempDir()
	agents := []generated.Agent{
		{
			ID:          303,
			Name:        "ghost",
			Status:      "blocked",
			TmuxSession: "ghost-session",
			WorkingDir:  sql.NullString{String: workingDir, Valid: true},
			ProfileName: sql.NullString{String: "broken", Valid: true},
		},
	}
	bootstrapInvoked := false
	parseErr := errors.New("load profile \"broken\": toml: line 1: invalid token")
	m := model{
		mode:          modeNormal,
		agents:        agents,
		columns:       buildColumns(agents, ""),
		focused:       2,
		selected:      []int{headerSelectionRow, headerSelectionRow, 0, headerSelectionRow, headerSelectionRow},
		tmuxAvailable: true,
		liveSessions:  map[string]struct{}{},
		nowFn:         time.Now,
		loadProfileFn: func(string) (profile.Config, error) {
			return profile.Config{}, parseErr
		},
		bootstrapSession: func(context.Context, tmux.BootstrapSpec) error {
			bootstrapInvoked = true
			return nil
		},
	}

	next, cmd := m.recreateSelectedSession()
	if cmd != nil {
		t.Fatalf("expected no attach command when profile parse fails")
	}
	if bootstrapInvoked {
		t.Fatalf("bootstrap should not run when profile load fails fatally")
	}
	if !next.agents[0].LastError.Valid || !containsAny("load profile for recovery", next.agents[0].LastError.String) {
		t.Fatalf("expected last_error to record parse failure, got %+v", next.agents[0].LastError)
	}
}

func TestLKeyToastsLogPath(t *testing.T) {
	m := model{
		mode:    modeNormal,
		columns: newStatusColumns(),
		logPath: "/tmp/yaama-test.log",
	}
	after := m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	if len(after.toasts) != 1 {
		t.Fatalf("expected one toast, got %d", len(after.toasts))
	}
	if want := "Log: /tmp/yaama-test.log"; after.toasts[0].message != want {
		t.Fatalf("expected toast %q, got %q", want, after.toasts[0].message)
	}
}

func TestLKeyWarnsWhenLogPathMissing(t *testing.T) {
	m := model{
		mode:    modeNormal,
		columns: newStatusColumns(),
	}
	after := m.handleNormalMode(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	if len(after.toasts) != 1 {
		t.Fatalf("expected one toast, got %d", len(after.toasts))
	}
	if after.toasts[0].severity != toastWarning {
		t.Fatalf("expected warning severity, got %v", after.toasts[0].severity)
	}
}
