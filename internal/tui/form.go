package tui

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grahamdaw/yaama/internal/db/generated"
)

func (m model) openCreateForm(purpose formPurpose) model {
	m.mode = modeForm
	m.formDirty = false
	m.form = newFormState(purpose)
	return m
}

func (m model) openEditForm() model {
	selected, ok := m.currentSelection()
	if !ok {
		return m.pushNotice("No agent selected; choose a row before editing.")
	}

	m.mode = modeForm
	m.formDirty = false
	m.form = newFormState(formPurposeEdit)
	m.form.targetID = selected.ID
	m.setFormFieldValue("name", selected.Name)
	m.setFormFieldValue("tmux_session", selected.TmuxSession)
	m.setFormFieldValue("status", selected.Status)
	m.setFormFieldValue("task", nullStringRaw(selected.Task))
	m.setFormFieldValue("branch", nullStringRaw(selected.Branch))
	m.setFormFieldValue("working_dir", nullStringRaw(selected.WorkingDir))
	m.setFormFieldValue("profile_name", nullStringRaw(selected.ProfileName))
	m.setFormFieldValue("ticket_id", nullStringRaw(selected.TicketID))
	m.setFormFieldValue("initial_prompt", nullStringRaw(selected.InitialPrompt))
	return m
}

func newFormState(purpose formPurpose) formState {
	statusRequired := true
	fields := []formField{
		{key: "name", label: "Name", required: true},
		{key: "tmux_session", label: "Session", required: true},
		{key: "status", label: "Status", value: "idle", required: statusRequired},
		{key: "task", label: "Task"},
		{key: "branch", label: "Branch"},
		{key: "working_dir", label: "Working Dir"},
		{key: "profile_name", label: "Profile"},
		{key: "ticket_id", label: "Ticket"},
		{key: "initial_prompt", label: "Initial Prompt"},
	}
	if purpose == formPurposeCreateProfile {
		setRequired(&fields, "profile_name", true)
		setRequired(&fields, "ticket_id", true)
		setRequired(&fields, "initial_prompt", true)
	}

	return formState{
		purpose: purpose,
		fields:  fields,
		errors:  map[string]string{},
	}
}

func setRequired(fields *[]formField, key string, required bool) {
	for i := range *fields {
		if (*fields)[i].key == key {
			(*fields)[i].required = required
			return
		}
	}
}

func (m *model) setFormFieldValue(key, value string) {
	for idx := range m.form.fields {
		if m.form.fields[idx].key == key {
			m.form.fields[idx].value = value
			return
		}
	}
}

func (m model) formFieldValue(key string) string {
	for _, field := range m.form.fields {
		if field.key == key {
			return strings.TrimSpace(field.value)
		}
	}
	return ""
}

func (m model) editActiveFormField(mutator func(string) string) model {
	if len(m.form.fields) == 0 || m.form.active < 0 || m.form.active >= len(m.form.fields) {
		return m
	}
	field := &m.form.fields[m.form.active]
	field.value = mutator(field.value)
	delete(m.form.errors, field.key)
	m.formDirty = m.isFormDirty()
	return m
}

func (m model) isFormDirty() bool {
	if m.form.purpose == formPurposeEdit {
		selected, ok := m.findAgentByID(m.form.targetID)
		if !ok {
			return true
		}
		return strings.TrimSpace(m.formFieldValue("name")) != strings.TrimSpace(selected.Name) ||
			strings.TrimSpace(m.formFieldValue("tmux_session")) != strings.TrimSpace(selected.TmuxSession) ||
			strings.TrimSpace(m.formFieldValue("status")) != strings.TrimSpace(selected.Status) ||
			strings.TrimSpace(m.formFieldValue("task")) != strings.TrimSpace(nullStringRaw(selected.Task)) ||
			strings.TrimSpace(m.formFieldValue("branch")) != strings.TrimSpace(nullStringRaw(selected.Branch)) ||
			strings.TrimSpace(m.formFieldValue("working_dir")) != strings.TrimSpace(nullStringRaw(selected.WorkingDir)) ||
			strings.TrimSpace(m.formFieldValue("profile_name")) != strings.TrimSpace(nullStringRaw(selected.ProfileName)) ||
			strings.TrimSpace(m.formFieldValue("ticket_id")) != strings.TrimSpace(nullStringRaw(selected.TicketID)) ||
			strings.TrimSpace(m.formFieldValue("initial_prompt")) != strings.TrimSpace(nullStringRaw(selected.InitialPrompt))
	}

	for _, field := range m.form.fields {
		if strings.TrimSpace(field.value) != "" {
			return true
		}
	}
	return false
}

func (m model) submitForm() model {
	errorsByField := m.validateForm()
	m.form.errors = errorsByField
	if len(errorsByField) > 0 {
		return m.pushNotice("Form has validation errors; fix highlighted fields.")
	}

	switch m.form.purpose {
	case formPurposeCreateGeneric, formPurposeCreateProfile:
		return m.persistCreateForm()
	case formPurposeEdit:
		return m.persistEditForm()
	default:
		return m.pushNotice("Unknown form mode; save cancelled.")
	}
}

func (m model) validateForm() map[string]string {
	errorsByField := map[string]string{}

	for _, field := range m.form.fields {
		if field.required && strings.TrimSpace(field.value) == "" {
			errorsByField[field.key] = "required"
		}
	}

	status := m.formFieldValue("status")
	if status != "" && statusIndex(status) < 0 {
		errorsByField["status"] = "must be one of: idle, running, blocked, review, done"
	}

	session := m.formFieldValue("tmux_session")
	if session != "" {
		existingID, exists, err := m.findSessionOwner(session)
		if err != nil {
			errorsByField["tmux_session"] = fmt.Sprintf("failed to validate uniqueness: %v", err)
		} else if exists && existingID != m.form.targetID {
			errorsByField["tmux_session"] = "session already in use"
		}
	}

	profile := m.formFieldValue("profile_name")
	if profile != "" {
		if err := validateProfileReference(profile); err != nil {
			errorsByField["profile_name"] = err.Error()
		}
	}

	return errorsByField
}

func validateProfileReference(profileName string) error {
	clean := strings.TrimSpace(profileName)
	if clean == "" {
		return nil
	}
	if strings.Contains(clean, "/") || strings.Contains(clean, `\`) || strings.Contains(clean, "..") {
		return errors.New("invalid profile name")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot resolve home directory")
	}
	path := filepath.Join(home, ".config", "yaam", "profiles", clean+".toml")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return errors.New("profile not found in ~/.config/yaam/profiles")
		}
		return fmt.Errorf("unable to verify profile file")
	}
	return nil
}

func (m model) findSessionOwner(session string) (int64, bool, error) {
	if m.queries != nil {
		agent, err := m.queries.GetAgentByTmuxSession(context.Background(), session)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, false, nil
			}
			return 0, false, err
		}
		return agent.ID, true, nil
	}

	for _, agent := range m.agents {
		if agent.TmuxSession == session {
			return agent.ID, true, nil
		}
	}
	return 0, false, nil
}

func (m model) persistCreateForm() model {
	params := generated.CreateAgentParams{
		Name:          m.formFieldValue("name"),
		TmuxSession:   m.formFieldValue("tmux_session"),
		Status:        m.formFieldValue("status"),
		Task:          toNullString(m.formFieldValue("task")),
		Branch:        toNullString(m.formFieldValue("branch")),
		WorkingDir:    toNullString(m.formFieldValue("working_dir")),
		ProfileName:   toNullString(m.formFieldValue("profile_name")),
		TicketID:      toNullString(m.formFieldValue("ticket_id")),
		InitialPrompt: toNullString(m.formFieldValue("initial_prompt")),
	}

	var saved generated.Agent
	if m.queries != nil {
		created, err := m.queries.CreateAgent(context.Background(), params)
		if err != nil {
			return m.pushNotice(fmt.Sprintf("Create failed: %v", err))
		}
		saved = created
		rows, err := m.queries.ListActiveAgents(context.Background())
		if err != nil {
			return m.pushNotice(fmt.Sprintf("Create succeeded, but refresh failed: %v", err))
		}
		m.agents = rows
	} else {
		saved = generated.Agent{
			ID:          m.nextLocalID(),
			Name:        params.Name,
			TmuxSession: params.TmuxSession,
			Status:      params.Status,
			Task:        params.Task,
			Branch:      params.Branch,
			WorkingDir:  params.WorkingDir,
			ProfileName: params.ProfileName,
			TicketID:    params.TicketID,
			InitialPrompt: params.InitialPrompt,
			CleanupState: "active",
		}
		m.agents = append(m.agents, saved)
	}

	m = m.rebuildColumns()
	if colIdx, rowIdx, found := m.findSelectionByID(saved.ID); found {
		m.focused = colIdx
		m.selected[colIdx] = rowIdx
	}
	m.mode = modeNormal
	m.form = formState{}
	m.formDirty = false
	m.showEmpty = len(m.agents) == 0
	return m.pushNotice(fmt.Sprintf("Created agent %s.", saved.Name))
}

func (m model) persistEditForm() model {
	target, ok := m.findAgentByID(m.form.targetID)
	if !ok {
		return m.pushNotice("Edit target no longer exists.")
	}

	if m.queries != nil {
		_, err := m.queries.UpdateAgent(context.Background(), generated.UpdateAgentParams{
			ID:              target.ID,
			Name:            m.formFieldValue("name"),
			TmuxSession:     m.formFieldValue("tmux_session"),
			Status:          m.formFieldValue("status"),
			Task:            toNullString(m.formFieldValue("task")),
			LastActivity:    target.LastActivity,
			Branch:          toNullString(m.formFieldValue("branch")),
			WorkingDir:      toNullString(m.formFieldValue("working_dir")),
			ProfileName:     toNullString(m.formFieldValue("profile_name")),
			TicketID:        toNullString(m.formFieldValue("ticket_id")),
			InitialPrompt:   toNullString(m.formFieldValue("initial_prompt")),
			LastHeartbeatAt: target.LastHeartbeatAt,
			LastError:       target.LastError,
		})
		if err != nil {
			return m.pushNotice(fmt.Sprintf("Update failed: %v", err))
		}
		rows, err := m.queries.ListActiveAgents(context.Background())
		if err != nil {
			return m.pushNotice(fmt.Sprintf("Update succeeded, but refresh failed: %v", err))
		}
		m.agents = rows
	} else {
		for i := range m.agents {
			if m.agents[i].ID == target.ID {
				m.agents[i].Name = m.formFieldValue("name")
				m.agents[i].TmuxSession = m.formFieldValue("tmux_session")
				m.agents[i].Status = m.formFieldValue("status")
				m.agents[i].Task = toNullString(m.formFieldValue("task"))
				m.agents[i].Branch = toNullString(m.formFieldValue("branch"))
				m.agents[i].WorkingDir = toNullString(m.formFieldValue("working_dir"))
				m.agents[i].ProfileName = toNullString(m.formFieldValue("profile_name"))
				m.agents[i].TicketID = toNullString(m.formFieldValue("ticket_id"))
				m.agents[i].InitialPrompt = toNullString(m.formFieldValue("initial_prompt"))
				break
			}
		}
	}

	m = m.rebuildColumns()
	if colIdx, rowIdx, found := m.findSelectionByID(target.ID); found {
		m.focused = colIdx
		m.selected[colIdx] = rowIdx
	}
	m.mode = modeNormal
	m.form = formState{}
	m.formDirty = false
	m.showEmpty = len(m.agents) == 0
	return m.pushNotice(fmt.Sprintf("Updated %s.", target.Name))
}

func (m model) findAgentByID(id int64) (generated.Agent, bool) {
	for _, agent := range m.agents {
		if agent.ID == id {
			return agent, true
		}
	}
	return generated.Agent{}, false
}

func (m model) nextLocalID() int64 {
	var maxID int64
	for _, agent := range m.agents {
		if agent.ID > maxID {
			maxID = agent.ID
		}
	}
	return maxID + 1
}

func toNullString(value string) sql.NullString {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func nullStringRaw(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

