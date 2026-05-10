package tui

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/grahamdaw/yaama/internal/db/generated"
)

func TestRenderColumnsShowsRuntimeBadges(t *testing.T) {
	now := time.Date(2026, time.May, 9, 9, 0, 0, 0, time.UTC)
	dead := generated.Agent{
		Name:        "dead-agent",
		Status:      "running",
		TmuxSession: "dead-session",
		UpdatedAt:   now,
	}
	stale := generated.Agent{
		Name:        "stale-agent",
		Status:      "running",
		TmuxSession: "live-session",
		UpdatedAt:   now.Add(-30 * time.Minute),
	}

	m := model{
		columns:     buildColumns([]generated.Agent{dead, stale}, ""),
		selected:    []int{0, headerSelectionRow, headerSelectionRow, headerSelectionRow, headerSelectionRow},
		focused:     0,
		tmuxAvailable: true,
		liveSessions:  map[string]struct{}{"live-session": {}},
		staleAfter:    15 * time.Minute,
		nowFn: func() time.Time {
			return now
		},
	}

	rendered := m.renderColumns(120)
	if !strings.Contains(rendered, "dead-agent [DEAD]") {
		t.Fatalf("expected dead badge in render output, got: %s", rendered)
	}
	if !strings.Contains(rendered, "[STALE]") {
		t.Fatalf("expected stale badge in render output, got: %s", rendered)
	}
}

func TestRenderEmptyStateIncludesNextActions(t *testing.T) {
	m := model{}
	rendered := m.renderEmptyState(100)

	if !strings.Contains(rendered, "Press n to create your first agent.") {
		t.Fatalf("expected create action in empty-state copy, got: %s", rendered)
	}
	if !strings.Contains(rendered, "board status running") {
		t.Fatalf("expected cli action in empty-state copy, got: %s", rendered)
	}
}

func TestRenderDetailPanelShowsDeadRecoveryActions(t *testing.T) {
	now := time.Date(2026, time.May, 9, 9, 0, 0, 0, time.UTC)
	agent := generated.Agent{
		Name:        "broken",
		Status:      "running",
		TmuxSession: "missing",
		WorkingDir:  sql.NullString{String: "/tmp/work", Valid: true},
		UpdatedAt:   now,
	}
	m := model{
		columns: []column{
			{key: "running", title: "Running", cards: []generated.Agent{agent}},
		},
		selected:      []int{0},
		focused:       0,
		tmuxAvailable: true,
		liveSessions:  map[string]struct{}{},
		nowFn: func() time.Time {
			return now
		},
	}

	rendered := m.renderDetailPanel(120)
	if !strings.Contains(rendered, "Runtime: DEAD (tmux session missing)") {
		t.Fatalf("expected dead runtime in details, got: %s", rendered)
	}
	if !strings.Contains(rendered, "Recovery: press r to recreate in working_dir") {
		t.Fatalf("expected recovery next actions in details, got: %s", rendered)
	}
}

func TestRenderCreateWizardIncludesBranchStep(t *testing.T) {
	form := newFormState(formPurposeCreateGeneric)
	form.active = 2
	form.fields[1].value = "KAI-123"
	form.fields[2].value = "fix/kai-123"

	m := model{
		mode: modeForm,
		form: form,
	}

	rendered := m.renderFormOverlay(120)
	if !strings.Contains(rendered, "3) Branch: fix/kai-123") {
		t.Fatalf("expected branch step in create wizard, got: %s", rendered)
	}
	if !strings.Contains(rendered, "Step 3: type branch, Enter create") {
		t.Fatalf("expected branch step guidance in create wizard, got: %s", rendered)
	}
}
