package tui

import (
	"context"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/grahamdaw/yaama/internal/db/generated"
	"github.com/grahamdaw/yaama/internal/profile"
	"github.com/grahamdaw/yaama/internal/startup"
	"github.com/grahamdaw/yaama/internal/tmux"
)

type column struct {
	key   string
	title string
	cards []generated.Agent
}

type model struct {
	width        int
	height       int
	mode         mode
	columns      []column
	agents       []generated.Agent
	queries      *generated.Queries
	focused      int
	selected     []int
	search       string
	formDirty    bool
	form         formState
	confirm      confirmState
	statusPicker statusPickerState
	toasts       []toast
	banner       string
	showEmpty    bool

	liveSessions map[string]struct{}

	refreshEvery time.Duration
	staleAfter   time.Duration
	nowFn        func() time.Time

	loadAgentsFn      func(context.Context) ([]generated.Agent, error)
	listSessionsFn    func(context.Context) ([]string, error)
	attachOrSwitchCmd func(context.Context, string) (*exec.Cmd, error)
	createDetachedCmd func(context.Context, string, string) (*exec.Cmd, error)
	loadProfileFn     func(string) (profile.Config, error)
	resolveRuntimeFn  func(profile.Config, string, string) (profile.RuntimeValues, error)
	bootstrapSession  func(context.Context, tmux.BootstrapSpec) error
	tmuxAvailable     bool
}

type mode int

const (
	modeNormal mode = iota
	modeSearch
	modeForm
	modeConfirm
	modeHelp
	modeStatusPicker
)

type confirmState struct {
	returnMode mode
	kind       string
	agentID    int64
	agentName  string
	force      bool
	workingDir string
}

type statusPickerState struct {
	selected int
}

type toastSeverity int

const (
	toastSuccess toastSeverity = iota
	toastWarning
	toastError
)

type toast struct {
	message   string
	severity  toastSeverity
	createdAt time.Time
}

const (
	confirmKindNone         = ""
	confirmKindArchive      = "archive"
	confirmKindPrune        = "prune"
	confirmKindPruneForce   = "prune-force-required"
	confirmKindDiscardEdits = "discard"
	headerSelectionRow      = -1
	defaultRefreshInterval  = 5 * time.Second
	defaultStaleAfter       = 15 * time.Minute
)

type formPurpose string

const (
	formPurposeCreateGeneric formPurpose = "create-generic"
	formPurposeCreateProfile formPurpose = "create-profile"
	formPurposeEdit          formPurpose = "edit"
)

type formField struct {
	key      string
	label    string
	value    string
	required bool
}

type formState struct {
	purpose        formPurpose
	targetID       int64
	fields         []formField
	active         int
	errors         map[string]string
	profileOptions []string
}

func NewModel(state startup.State) tea.Model {
	agents := []generated.Agent{}
	loadAgentsFn := func(context.Context) ([]generated.Agent, error) {
		return agents, nil
	}
	if state.DB.Queries != nil {
		loadAgentsFn = state.DB.Queries.ListActiveAgents
		rows, err := state.DB.Queries.ListActiveAgents(context.Background())
		if err == nil {
			agents = rows
		} else {
			state.Notices = append(state.Notices, "Unable to load agents from DB; showing empty board and retrying in background.")
		}
	}

	showEmpty := len(agents) == 0
	columns := buildColumns(agents, "")
	selected := make([]int, len(columns))
	for i, col := range columns {
		selected[i] = defaultSelectedRow(col.cards)
	}

	return model{
		mode:         modeNormal,
		columns:      columns,
		agents:       agents,
		queries:      state.DB.Queries,
		focused:      0,
		selected:     selected,
		statusPicker: statusPickerState{selected: 0},
		toasts:       initialToasts(state.Notices),
		banner:       initialBanner(state.TmuxAvailable),
		showEmpty:    showEmpty,
		liveSessions: map[string]struct{}{},
		refreshEvery: defaultRefreshInterval,
		staleAfter:   defaultStaleAfter,
		nowFn:        time.Now,
		loadAgentsFn: loadAgentsFn,
		listSessionsFn: func(ctx context.Context) ([]string, error) {
			return tmux.ListSessions(ctx)
		},
		attachOrSwitchCmd: tmux.AttachOrSwitchCommand,
		createDetachedCmd: tmux.CreateDetachedSessionCommand,
		loadProfileFn:     profile.Load,
		resolveRuntimeFn:  profile.ResolveRuntimeValues,
		bootstrapSession:  tmux.BootstrapSession,
		tmuxAvailable:     state.TmuxAvailable,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(refreshTickCmd(m.refreshEvery), refreshNowCmd())
}

func buildColumns(agents []generated.Agent, search string) []column {
	columns := newStatusColumns()
	filtered := filterAgents(agents, search)
	for _, agent := range filtered {
		for i := range columns {
			if columns[i].key == agent.Status {
				columns[i].cards = append(columns[i].cards, agent)
				break
			}
		}
	}

	return columns
}

func filterAgents(agents []generated.Agent, search string) []generated.Agent {
	normalized := strings.TrimSpace(strings.ToLower(search))
	if normalized == "" {
		return agents
	}

	out := make([]generated.Agent, 0, len(agents))
	for _, agent := range agents {
		if containsAny(normalized,
			agent.Name,
			agent.TmuxSession,
			agent.Task.String,
			agent.Branch.String,
		) {
			out = append(out, agent)
		}
	}

	return out
}

func containsAny(needle string, fields ...string) bool {
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), needle) {
			return true
		}
	}
	return false
}

func defaultSelectedRow(cards []generated.Agent) int {
	if len(cards) == 0 {
		return headerSelectionRow
	}
	return 0
}

func initialToasts(notices []string) []toast {
	now := time.Now()
	out := make([]toast, 0, len(notices))
	for _, notice := range notices {
		if strings.TrimSpace(notice) == "" {
			continue
		}
		out = append(out, toast{
			message:   notice,
			severity:  toastSuccess,
			createdAt: now,
		})
	}
	return out
}

func initialBanner(tmuxAvailable bool) string {
	if tmuxAvailable {
		return ""
	}
	return "tmux binary not found in PATH; attach actions are disabled."
}

func (m model) rebuildColumns() model {
	oldColumns := m.columns
	oldSelected := m.selected

	m.columns = buildColumns(m.agents, m.search)
	m.selected = make([]int, len(m.columns))
	for i, col := range m.columns {
		prev := headerSelectionRow
		if i < len(oldSelected) {
			prev = oldSelected[i]
		}
		if i < len(oldColumns) && oldColumns[i].key != col.key {
			prev = defaultSelectedRow(col.cards)
		}
		m.selected[i] = clampRow(prev, len(col.cards))
	}

	if len(m.columns) == 0 {
		m.focused = 0
		return m
	}

	if m.focused < 0 {
		m.focused = 0
	}
	if m.focused >= len(m.columns) {
		m.focused = len(m.columns) - 1
	}

	return m
}

func clampRow(row int, cardCount int) int {
	if cardCount <= 0 {
		return headerSelectionRow
	}
	if row < 0 {
		return 0
	}
	if row >= cardCount {
		return cardCount - 1
	}
	return row
}
