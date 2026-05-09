package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/grahamdaw/yaama/internal/db/generated"
	"github.com/grahamdaw/yaama/internal/startup"
	"github.com/grahamdaw/yaama/internal/tmux"
)

var acceptedStatuses = []string{"idle", "running", "blocked", "review", "done"}

var (
	errOutsideTmux = errors.New("status command requires an active tmux session")
)

type statusCommandInput struct {
	status   string
	task     optionalString
	activity optionalString
	branch   optionalString
}

type optionalString struct {
	value string
	set   bool
}

func (s *optionalString) Set(value string) error {
	s.value = value
	s.set = true
	return nil
}

func (s *optionalString) String() string {
	return s.value
}

func runStatusCommand(ctx context.Context, args []string, stderr io.Writer) int {
	input, dbPath, err := parseStatusArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "status command error: %v\n", err)
		return 1
	}

	state, err := startup.Bootstrap(ctx, startup.Options{DBPathOverride: dbPath})
	if err != nil {
		fmt.Fprintf(stderr, "status command error: startup failed: %v\n", err)
		return 1
	}
	defer func() {
		_ = state.DB.Conn.Close()
	}()

	err = executeStatusUpdate(ctx, state.DB.Queries, tmux.CurrentSession, input)
	if err != nil {
		switch {
		case errors.Is(err, errOutsideTmux):
			fmt.Fprintln(stderr, "status command error: run this from inside the agent tmux session (TMUX is not set)")
		case errors.Is(err, tmux.ErrTmuxUnavailable):
			fmt.Fprintln(stderr, "status command error: tmux is unavailable in PATH")
		default:
			var missingErr missingAgentError
			if errors.As(err, &missingErr) {
				fmt.Fprintf(stderr, "status command error: no agent record matches tmux session %q; create/register the agent first\n", missingErr.session)
				break
			}
			fmt.Fprintf(stderr, "status command error: %v\n", err)
		}
		return 1
	}

	return 0
}

type currentSessionFn func(context.Context) (string, error)

func executeStatusUpdate(
	ctx context.Context,
	queries *generated.Queries,
	resolveSession currentSessionFn,
	input statusCommandInput,
) error {
	session, err := resolveSession(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(session) == "" {
		return errOutsideTmux
	}

	if _, err := queries.GetAgentByTmuxSession(ctx, session); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return missingAgentError{session: session}
		}
		return fmt.Errorf("load agent by tmux session %q: %w", session, err)
	}

	updateParams := generated.UpdateAgentStatusByTmuxSessionParams{
		Status:       input.status,
		Task:         optionalSQLString(input.task),
		LastActivity: optionalSQLString(input.activity),
		Branch:       optionalSQLString(input.branch),
		LastError:    sql.NullString{},
		TmuxSession:  session,
	}
	if err := queries.UpdateAgentStatusByTmuxSession(ctx, updateParams); err != nil {
		return fmt.Errorf("update agent status for tmux session %q: %w", session, err)
	}
	return nil
}

type missingAgentError struct {
	session string
}

func (e missingAgentError) Error() string {
	return fmt.Sprintf("no agent record for tmux session %q", e.session)
}

func parseStatusArgs(args []string) (statusCommandInput, string, error) {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)

	var dbPath string
	var task optionalString
	var activity optionalString
	var branch optionalString

	fs.StringVar(&dbPath, "db", "", "path to SQLite DB file")
	fs.Var(&task, "task", "current task text")
	fs.Var(&activity, "activity", "last activity text")
	fs.Var(&branch, "branch", "git branch")
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return statusCommandInput{}, "", err
	}

	positionals := fs.Args()
	if len(positionals) != 1 {
		return statusCommandInput{}, "", fmt.Errorf("usage: board status <status> [--task <text>] [--activity <text>] [--branch <name>]")
	}

	status := strings.TrimSpace(positionals[0])
	if !isAcceptedStatus(status) {
		return statusCommandInput{}, "", fmt.Errorf(
			"invalid status %q (accepted: %s)",
			status,
			strings.Join(acceptedStatuses, ", "),
		)
	}

	return statusCommandInput{
		status:   status,
		task:     task,
		activity: activity,
		branch:   branch,
	}, dbPath, nil
}

func isAcceptedStatus(status string) bool {
	for _, candidate := range acceptedStatuses {
		if status == candidate {
			return true
		}
	}
	return false
}

func optionalSQLString(value optionalString) sql.NullString {
	if !value.set {
		return sql.NullString{}
	}
	return sql.NullString{
		String: value.value,
		Valid:  true,
	}
}
