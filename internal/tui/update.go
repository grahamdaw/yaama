package tui

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/grahamdaw/yaama/internal/db/generated"
	"github.com/grahamdaw/yaama/internal/profile"
	"github.com/grahamdaw/yaama/internal/tmux"
)

import tea "github.com/charmbracelet/bubbletea"

type refreshTickMsg struct{}
type refreshNowMsg struct{}

type refreshResultMsg struct {
	agents      []generated.Agent
	liveSession map[string]struct{}
	err         error
}

type attachCompleteMsg struct {
	err error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m = m.pruneToasts()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case refreshTickMsg:
		return m, tea.Batch(refreshTickCmd(m.refreshEvery), m.refreshCmd())
	case refreshNowMsg:
		return m, m.refreshCmd()
	case refreshResultMsg:
		if msg.agents != nil {
			m.agents = msg.agents
			m.showEmpty = len(m.agents) == 0
			m = m.rebuildColumns()
		}
		if msg.liveSession != nil {
			m.liveSessions = msg.liveSession
		}
		if msg.err != nil {
			m = m.pushError(fmt.Sprintf("Refresh failed: %v. Retrying in background.", msg.err))
			m.banner = m.runtimeBannerForError(msg.err)
			return m, nil
		}
		if !m.tmuxAvailable {
			m.banner = initialBanner(false)
		} else {
			m.banner = ""
		}
		return m, nil
	case attachCompleteMsg:
		if msg.err != nil {
			return m.pushError(fmt.Sprintf("tmux attach failed: %v", msg.err)), refreshNowCmd()
		}
		return m.pushSuccess("Returned from tmux session."), refreshNowCmd()
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "enter" && m.mode == modeNormal {
			next, cmd := m.attachSelectedSession()
			return next, cmd
		}
		if msg.String() == "r" && m.mode == modeNormal {
			next, cmd := m.recreateSelectedSession()
			return next, cmd
		}
		return m.handleModeKey(msg), nil
	}

	return m, nil
}

func refreshTickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func refreshNowCmd() tea.Cmd {
	return func() tea.Msg {
		return refreshNowMsg{}
	}
}

func (m model) refreshData() tea.Msg {
	ctx := context.Background()
	if m.loadAgentsFn == nil {
		return refreshResultMsg{}
	}

	agents, err := m.loadAgentsFn(ctx)
	if err != nil {
		return refreshResultMsg{err: err}
	}

	live := map[string]struct{}{}
	if m.tmuxAvailable && m.listSessionsFn != nil {
		sessions, sessionErr := m.listSessionsFn(ctx)
		if sessionErr != nil {
			return refreshResultMsg{
				agents:      agents,
				liveSession: live,
				err:         sessionErr,
			}
		}
		for _, session := range sessions {
			live[session] = struct{}{}
		}
	}

	return refreshResultMsg{
		agents:      agents,
		liveSession: live,
	}
}

func (m model) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		return m.refreshData()
	}
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
	case "r":
		return m
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
		kind:       confirmKindArchive,
		returnMode: modeNormal,
		agentID:    selected.ID,
		agentName:  selected.Name,
	}
	return m
}

func (m model) openPruneConfirm() model {
	selected, ok := m.currentSelection()
	if !ok {
		return m.pushNotice("No agent selected; choose a row before pruning.")
	}
	kind := confirmKindPrune
	if m.pruneRequiresForce(selected) {
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

	switch msg.Type {
	case tea.KeyEsc:
		if m.formDirty {
			m.mode = modeConfirm
			m.confirm = confirmState{kind: confirmKindDiscardEdits, returnMode: modeForm}
			return m
		}
		m.mode = modeNormal
		m.form = formState{}
		return m
	case tea.KeyUp:
		if len(m.form.fields) > 0 {
			m.form.active = (m.form.active - 1 + len(m.form.fields)) % len(m.form.fields)
		}
		return m
	case tea.KeyDown, tea.KeyTab:
		if len(m.form.fields) > 0 {
			m.form.active = (m.form.active + 1) % len(m.form.fields)
		}
		return m
	case tea.KeyShiftTab:
		if len(m.form.fields) > 0 {
			m.form.active = (m.form.active - 1 + len(m.form.fields)) % len(m.form.fields)
		}
		return m
	case tea.KeyLeft:
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
	case tea.KeyRight:
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
	case tea.KeyBackspace:
		if isCreateWizard && m.form.active == 0 {
			return m
		}
		return m.editActiveFormField(func(current string) string {
			if len(current) == 0 {
				return current
			}
			return current[:len(current)-1]
		})
	case tea.KeyEnter:
		if isCreateWizard && m.form.active < len(m.form.fields)-1 {
			m.form.active++
			return m
		}
		return m.submitForm()
	case tea.KeyRunes:
		if len(m.form.fields) > 0 && m.form.active >= 0 && m.form.active < len(m.form.fields) {
			activeKey := m.form.fields[m.form.active].key
			if isCreateWizard && activeKey == "profile_name" {
				switch msg.String() {
				case "j", "l":
					return m.cycleCreateProfile(1)
				case "k", "h":
					return m.cycleCreateProfile(-1)
				default:
					return m
				}
			}
		}
		return m.editActiveFormField(func(current string) string {
			return current + msg.String()
		})
	default:
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
	return m.applyCleanup("archived", false)
}

func (m model) applyPrune() model {
	return m.applyCleanup("pruned", true)
}

func (m model) applyCleanup(finalState string, pruneWorkingDir bool) model {
	target, ok := m.findAgentByID(m.confirm.agentID)
	if !ok {
		m.mode = modeNormal
		m.confirm = confirmState{}
		return m.pushNotice("Cleanup target no longer exists.")
	}
	if pruneWorkingDir && m.pruneRequiresForce(target) && !m.confirm.force {
		return m.pushNotice("Working directory is non-empty; force prune required.")
	}

	if err := m.killCleanupSession(target); err != nil {
		next, persistErr := m.persistCleanupFailure(target, fmt.Sprintf("tmux cleanup failed: %v", err))
		next.mode = modeNormal
		next.confirm = confirmState{}
		if persistErr != nil {
			return next.pushError(fmt.Sprintf("Cleanup failed for %s: %v (also failed to persist error: %v)", target.Name, err, persistErr))
		}
		return next.pushWarning(
			fmt.Sprintf(
				"Cleanup stopped before archive/prune: could not kill tmux session (%v). Remediation: verify session name or tmux server health, then retry.",
				err,
			),
		)
	}

	if pruneWorkingDir {
		if err := m.removeCleanupWorktree(target); err != nil {
			next, persistErr := m.persistCleanupFailure(target, fmt.Sprintf("git worktree remove failed: %v", err))
			next.mode = modeNormal
			next.confirm = confirmState{}
			if persistErr != nil {
				return next.pushError(fmt.Sprintf("Cleanup failed for %s: %v (also failed to persist error: %v)", target.Name, err, persistErr))
			}
			return next.pushWarning(
				fmt.Sprintf(
					"Cleanup stopped before final prune: git worktree remove failed (%v). Remediation: fix git worktree state, then retry.",
					err,
				),
			)
		}
	}

	scriptErr := m.runCleanupScripts(target)

	activity := fmt.Sprintf("cleanup %s", finalState)
	lastErr := sql.NullString{}
	if scriptErr != nil {
		lastErr = sql.NullString{String: fmt.Sprintf("cleanup script failed: %v", scriptErr), Valid: true}
		activity = fmt.Sprintf("cleanup %s with script errors", finalState)
	}
	next, err := m.persistCleanupState(target, finalState, activity, lastErr)
	next.mode = modeNormal
	next.confirm = confirmState{}
	if err != nil {
		return next.pushError(fmt.Sprintf("Cleanup state transition failed for %s: %v", target.Name, err))
	}

	message := fmt.Sprintf("Cleaned up %s.", target.Name)
	if finalState == "pruned" {
		message = fmt.Sprintf("Pruned %s.", target.Name)
	}
	if finalState == "archived" {
		message = fmt.Sprintf("Archived %s.", target.Name)
	}
	if scriptErr != nil {
		return next.pushWarning(
			fmt.Sprintf(
				"%s Runtime cleanup completed but script hooks failed (%v). Remediation: inspect profile cleanup hooks and rerun if needed.",
				message,
				scriptErr,
			),
		)
	}
	return next.pushNotice(message)
}

func (m model) pruneRequiresForce(target generated.Agent) bool {
	if m.removeWorktreeFn == nil {
		return false
	}
	return strings.TrimSpace(nullStringRaw(target.WorkingDir)) != ""
}

func (m model) killCleanupSession(target generated.Agent) error {
	session := strings.TrimSpace(target.TmuxSession)
	if session == "" {
		return nil
	}
	if !m.tmuxAvailable {
		return errors.New("tmux unavailable in PATH")
	}
	killSession := m.killSessionFn
	if killSession == nil {
		killSession = tmux.KillSession
	}
	return killSession(context.Background(), session)
}

func (m model) removeCleanupWorktree(target generated.Agent) error {
	if m.removeWorktreeFn == nil {
		return nil
	}
	workingDir := strings.TrimSpace(nullStringRaw(target.WorkingDir))
	if workingDir == "" {
		return nil
	}
	return m.removeWorktreeFn(context.Background(), workingDir)
}

func (m model) runCleanupScripts(target generated.Agent) error {
	profileName := strings.TrimSpace(nullStringRaw(target.ProfileName))
	if profileName == "" {
		return nil
	}
	loadProfile := m.loadProfileFn
	if loadProfile == nil {
		loadProfile = profile.Load
	}
	cfg, err := loadProfile(profileName)
	if err != nil {
		return fmt.Errorf("load profile %q: %w", profileName, err)
	}
	if len(cfg.Scripts.Cleanup) == 0 {
		return nil
	}

	runHook := m.runCleanupHookFn
	if runHook == nil {
		runHook = tmux.RunShellHook
	}
	workingDir := strings.TrimSpace(nullStringRaw(target.WorkingDir))
	if workingDir == "" {
		return errors.New("working_dir is empty; cannot run cleanup hooks")
	}
	failures := make([]string, 0)
	for _, hook := range cfg.Scripts.Cleanup {
		if err := runHook(context.Background(), workingDir, target.TmuxSession, hook); err != nil {
			failures = append(failures, fmt.Sprintf("%q: %v", hook, err))
		}
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

func (m model) persistCleanupFailure(target generated.Agent, failure string) (model, error) {
	msg := strings.TrimSpace(failure)
	if msg == "" {
		msg = "cleanup failed"
	}
	return m.persistCleanupState(target, target.CleanupState, "cleanup failed", sql.NullString{String: msg, Valid: true})
}

func (m model) persistCleanupState(target generated.Agent, cleanupState string, activity string, lastErr sql.NullString) (model, error) {
	if strings.TrimSpace(cleanupState) == "" {
		cleanupState = "active"
	}
	cleanupStateParam := sql.NullString{String: cleanupState, Valid: true}
	lastActivity := toNullString(activity)

	if m.queries != nil {
		_, err := m.queries.UpdateAgent(context.Background(), generated.UpdateAgentParams{
			Name:            target.Name,
			TmuxSession:     target.TmuxSession,
			Status:          target.Status,
			Task:            target.Task,
			LastActivity:    lastActivity,
			Branch:          target.Branch,
			WorkingDir:      target.WorkingDir,
			ProfileName:     target.ProfileName,
			TicketID:        target.TicketID,
			InitialPrompt:   target.InitialPrompt,
			LastHeartbeatAt: target.LastHeartbeatAt,
			LastError:       lastErr,
			CleanupState:    cleanupStateParam,
			ID:              target.ID,
		})
		if err != nil {
			return m, err
		}
		rows, err := m.queries.ListActiveAgents(context.Background())
		if err != nil {
			return m, err
		}
		m.agents = rows
	} else {
		for i := range m.agents {
			if m.agents[i].ID == target.ID {
				m.agents[i].CleanupState = cleanupState
				m.agents[i].LastError = lastErr
				m.agents[i].LastActivity = lastActivity
				break
			}
		}
		m.agents = filterActiveAgents(m.agents)
	}
	m.showEmpty = len(m.agents) == 0
	m = m.rebuildColumns()
	return m, nil
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
	return m.pushWarning(notice)
}

func (m model) pushSuccess(message string) model {
	return m.pushToast(message, toastSuccess)
}

func (m model) pushWarning(message string) model {
	return m.pushToast(message, toastWarning)
}

func (m model) pushError(message string) model {
	return m.pushToast(message, toastError)
}

func (m model) pushToast(message string, severity toastSeverity) model {
	const maxToasts = 4
	nowFn := m.nowFn
	if nowFn == nil {
		nowFn = time.Now
	}
	m.toasts = append(m.toasts, toast{
		message:   message,
		severity:  severity,
		createdAt: nowFn(),
	})
	if len(m.toasts) > maxToasts {
		m.toasts = m.toasts[len(m.toasts)-maxToasts:]
	}
	return m
}

func (m model) pruneToasts() model {
	nowFn := m.nowFn
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn()
	next := make([]toast, 0, len(m.toasts))
	for _, t := range m.toasts {
		maxAge := 4 * time.Second
		switch t.severity {
		case toastSuccess:
			maxAge = 2 * time.Second
		case toastWarning:
			maxAge = 4 * time.Second
		case toastError:
			maxAge = 6 * time.Second
		}
		if now.Sub(t.createdAt) <= maxAge {
			next = append(next, t)
		}
	}
	m.toasts = next
	return m
}

func (m model) runtimeBannerForError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "database is locked") {
		return "DB is locked; waiting for lock release and retrying every refresh tick."
	}
	if strings.Contains(msg, "database") || strings.Contains(msg, "sqlite") {
		return "DB unavailable; check path/permissions and keep board open for automatic retries."
	}
	if errors.Is(err, tmux.ErrTmuxUnavailable) {
		return "tmux not found in PATH; attach actions are disabled."
	}
	if strings.Contains(msg, "tmux") {
		return "tmux query failed; verify tmux server/session health."
	}
	return ""
}

func (m model) attachSelectedSession() (model, tea.Cmd) {
	selected, ok := m.currentSelection()
	if !ok {
		return m.pushWarning("No agent selected; choose a row before attaching."), nil
	}
	if !m.tmuxAvailable {
		return m.pushError("tmux unavailable in PATH. Install tmux or update PATH, then retry attach."), nil
	}
	if !m.sessionIsLive(selected.TmuxSession) {
		return m.pushWarning("Session not found. Press r to recreate, e to update mapping, or d to remove this item."), refreshNowCmd()
	}
	if m.attachOrSwitchCmd == nil {
		return m.pushError("tmux attach is not configured."), nil
	}

	cmd, err := m.attachOrSwitchCmd(context.Background(), selected.TmuxSession)
	if err != nil {
		return m.pushError(fmt.Sprintf("Cannot attach to %s: %v", selected.TmuxSession, err)), nil
	}

	return m, tea.ExecProcess(cmd, func(execErr error) tea.Msg {
		return attachCompleteMsg{err: execErr}
	})
}

func (m model) recreateSelectedSession() (model, tea.Cmd) {
	selected, ok := m.currentSelection()
	if !ok {
		return m.pushWarning("No agent selected; choose a row before recovering a session."), nil
	}
	if !m.tmuxAvailable {
		return m.pushError("tmux unavailable in PATH. Install tmux or update PATH, then retry recovery."), nil
	}
	if !m.isDead(selected) {
		return m.pushWarning("Selected agent is live. Use Enter to attach to the existing session."), nil
	}
	workingDir := strings.TrimSpace(nullStringRaw(selected.WorkingDir))
	if workingDir == "" {
		return m.pushWarning("Cannot recover dead session: working_dir is missing. Press e to set it, then retry with r."), nil
	}

	info, statErr := os.Stat(workingDir)
	if statErr != nil {
		return m.pushWarning(fmt.Sprintf("Cannot recover dead session: working_dir %q is not accessible (%v). Press e to fix mapping.", workingDir, statErr)), nil
	}
	if !info.IsDir() {
		return m.pushWarning(fmt.Sprintf("Cannot recover dead session: working_dir %q is not a directory. Press e to fix mapping.", workingDir)), nil
	}
	if m.createDetachedCmd == nil {
		return m.pushError("tmux session creation is not configured."), nil
	}

	createCmd, err := m.createDetachedCmd(context.Background(), selected.TmuxSession, workingDir)
	if err != nil {
		return m.recordRecoveryError(selected, fmt.Sprintf("build tmux recreate command: %v", err)).
			pushWarning("Session recreation failed. Press e to edit mapping, then retry with r."), nil
	}

	out, runErr := createCmd.CombinedOutput()
	if runErr != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			runErr = fmt.Errorf("%w (%s)", runErr, detail)
		}
		return m.recordRecoveryError(selected, fmt.Sprintf("create tmux session: %v", runErr)).
			pushWarning("Session recreation failed. Press e to edit mapping, then retry with r."), nil
	}

	m = m.markRecovered(selected)
	return m.pushSuccess(fmt.Sprintf("Recreated tmux session %s. Attaching...", selected.TmuxSession)).attachSelectedSession()
}

func (m model) markRecovered(selected generated.Agent) model {
	nowFn := m.nowFn
	if nowFn == nil {
		nowFn = time.Now
	}
	heartbeat := sql.NullTime{Time: nowFn(), Valid: true}
	lastError := sql.NullString{}
	lastActivity := selected.LastActivity
	if !lastActivity.Valid || strings.TrimSpace(lastActivity.String) == "" {
		lastActivity = sql.NullString{String: "session recreated", Valid: true}
	}

	if m.queries != nil {
		_, err := m.queries.UpdateAgent(context.Background(), generated.UpdateAgentParams{
			Name:            selected.Name,
			TmuxSession:     selected.TmuxSession,
			Status:          "running",
			Task:            selected.Task,
			LastActivity:    lastActivity,
			Branch:          selected.Branch,
			WorkingDir:      selected.WorkingDir,
			ProfileName:     selected.ProfileName,
			TicketID:        selected.TicketID,
			InitialPrompt:   selected.InitialPrompt,
			LastHeartbeatAt: heartbeat,
			LastError:       lastError,
			CleanupState:    sql.NullString{String: "active", Valid: true},
			ID:              selected.ID,
		})
		if err != nil {
			return m.pushWarning(fmt.Sprintf("Session recreated, but state update failed: %v", err))
		}
		rows, err := m.queries.ListActiveAgents(context.Background())
		if err != nil {
			return m.pushWarning(fmt.Sprintf("Session recreated, but refresh failed: %v", err))
		}
		m.agents = rows
	} else {
		for i := range m.agents {
			if m.agents[i].ID == selected.ID {
				m.agents[i].Status = "running"
				m.agents[i].LastHeartbeatAt = heartbeat
				m.agents[i].LastError = lastError
				m.agents[i].CleanupState = "active"
				if !m.agents[i].LastActivity.Valid || strings.TrimSpace(m.agents[i].LastActivity.String) == "" {
					m.agents[i].LastActivity = lastActivity
				}
				m.agents[i].UpdatedAt = heartbeat.Time
				break
			}
		}
	}

	if m.liveSessions == nil {
		m.liveSessions = map[string]struct{}{}
	}
	m.liveSessions[selected.TmuxSession] = struct{}{}
	m.showEmpty = len(m.agents) == 0
	m = m.rebuildColumns()
	if colIdx, rowIdx, found := m.findSelectionByID(selected.ID); found {
		m.focused = colIdx
		m.selected[colIdx] = rowIdx
	}
	return m
}

func (m model) recordRecoveryError(selected generated.Agent, message string) model {
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = "session recovery failed"
	}
	lastError := sql.NullString{String: msg, Valid: true}

	if m.queries != nil {
		_, err := m.queries.UpdateAgent(context.Background(), generated.UpdateAgentParams{
			Name:            selected.Name,
			TmuxSession:     selected.TmuxSession,
			Status:          selected.Status,
			Task:            selected.Task,
			LastActivity:    selected.LastActivity,
			Branch:          selected.Branch,
			WorkingDir:      selected.WorkingDir,
			ProfileName:     selected.ProfileName,
			TicketID:        selected.TicketID,
			InitialPrompt:   selected.InitialPrompt,
			LastHeartbeatAt: selected.LastHeartbeatAt,
			LastError:       lastError,
			CleanupState:    sql.NullString{String: selected.CleanupState, Valid: selected.CleanupState != ""},
			ID:              selected.ID,
		})
		if err != nil {
			return m.pushWarning(fmt.Sprintf("Recovery failed and last_error could not be stored: %v", err))
		}
		rows, err := m.queries.ListActiveAgents(context.Background())
		if err != nil {
			return m.pushWarning(fmt.Sprintf("Recovery failed and refresh failed: %v", err))
		}
		m.agents = rows
		m.showEmpty = len(m.agents) == 0
		return m.rebuildColumns()
	}

	for i := range m.agents {
		if m.agents[i].ID == selected.ID {
			m.agents[i].LastError = lastError
			break
		}
	}
	return m.rebuildColumns()
}

func (m model) sessionIsLive(session string) bool {
	_, ok := m.liveSessions[session]
	return ok
}

func (m model) isDead(agent generated.Agent) bool {
	if !m.tmuxAvailable {
		return false
	}
	if strings.TrimSpace(agent.TmuxSession) == "" {
		return true
	}
	return !m.sessionIsLive(agent.TmuxSession)
}

func (m model) isStale(agent generated.Agent) bool {
	if agent.Status != "running" {
		return false
	}
	nowFn := m.nowFn
	if nowFn == nil {
		nowFn = time.Now
	}
	staleAfter := m.staleAfter
	if staleAfter <= 0 {
		staleAfter = defaultStaleAfter
	}
	return nowFn().Sub(agent.UpdatedAt) > staleAfter
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
