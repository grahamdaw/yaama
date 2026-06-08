package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/grahamdaw/yaama/internal/agenthook"
	"github.com/grahamdaw/yaama/internal/db"
	"github.com/grahamdaw/yaama/internal/db/generated"
	"github.com/grahamdaw/yaama/internal/profile"
	"github.com/grahamdaw/yaama/internal/tmux"
)

type hookCommandInput struct {
	agent string
	raw   []byte
}

func runHookCommand(ctx context.Context, args []string, stdin io.Reader, stderr io.Writer) int {
	input, dbPath, err := parseHookArgs(args, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "hook command error: %v\n", err)
		return 1
	}

	harness, ok := agenthook.Get(input.agent)
	if !ok {
		fmt.Fprintf(stderr, "hook command error: unknown harness %q (registered: %s)\n",
			input.agent, strings.Join(agenthook.IDs(), ", "))
		return 1
	}

	event, err := harness.ParseHook(input.raw)
	if err != nil {
		if errors.Is(err, agenthook.ErrHarnessHasNoHook) {
			fmt.Fprintf(stderr, "hook command error: harness %q has no hook integration yet; only launch defaults are supported\n", input.agent)
			return 1
		}
		fmt.Fprintf(stderr, "hook command error: %v\n", err)
		return 1
	}

	cfg, err := profile.LoadConfig(profile.ConfigOptions{DBPathOverride: dbPath})
	if err != nil {
		fmt.Fprintf(stderr, "hook command error: %v\n", err)
		return 1
	}
	dbState, err := db.Init(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(stderr, "hook command error: %v\n", err)
		return 1
	}
	defer func() { _ = dbState.Conn.Close() }()

	err = executeHookUpdate(ctx, dbState.Queries, tmux.CurrentSession, event)
	if err != nil {
		switch {
		case errors.Is(err, errOutsideTmux):
			fmt.Fprintln(stderr, "hook command error: run this from inside the agent tmux session (TMUX is not set)")
		case errors.Is(err, tmux.ErrTmuxUnavailable):
			fmt.Fprintln(stderr, "hook command error: tmux is unavailable in PATH")
		default:
			var missingErr missingAgentError
			if errors.As(err, &missingErr) {
				fmt.Fprintf(stderr, "hook command error: no agent record matches tmux session %q; create/register the agent first\n", missingErr.session)
				break
			}
			fmt.Fprintf(stderr, "hook command error: %v\n", err)
		}
		return 1
	}
	return 0
}

func parseHookArgs(args []string, stdin io.Reader) (hookCommandInput, string, error) {
	fs := flag.NewFlagSet("hook", flag.ContinueOnError)
	var dbPath string
	fs.StringVar(&dbPath, "db", "", "path to SQLite DB file")
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return hookCommandInput{}, "", err
	}

	positionals := fs.Args()
	if len(positionals) != 1 {
		return hookCommandInput{}, "", fmt.Errorf(
			"usage: yaama hook <harness> [--db <path>]  (payload read from stdin; registered: %s)",
			strings.Join(agenthook.IDs(), ", "),
		)
	}
	agent := strings.TrimSpace(positionals[0])
	if agent == "" {
		return hookCommandInput{}, "", fmt.Errorf("agent name must not be empty")
	}

	raw, err := io.ReadAll(stdin)
	if err != nil {
		return hookCommandInput{}, "", fmt.Errorf("read hook payload from stdin: %w", err)
	}
	if len(raw) == 0 {
		return hookCommandInput{}, "", fmt.Errorf("hook payload on stdin is empty")
	}

	return hookCommandInput{agent: agent, raw: raw}, dbPath, nil
}

func executeHookUpdate(
	ctx context.Context,
	queries *generated.Queries,
	resolveSession currentSessionFn,
	event agenthook.Event,
) error {
	session, err := resolveSession(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(session) == "" {
		return errOutsideTmux
	}

	existing, err := queries.GetAgentByTmuxSession(ctx, session)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return missingAgentError{session: session}
		}
		return fmt.Errorf("load agent by tmux session %q: %w", session, err)
	}

	status := existing.Status
	if event.Status.Set && event.Status.Value != "" {
		status = event.Status.Value
	}

	params := generated.UpdateAgentStatusByTmuxSessionParams{
		Status:       status,
		Task:         sql.NullString{},
		LastActivity: optionalToSQL(event.LastActivity),
		Branch:       sql.NullString{},
		LastError:    optionalToSQL(event.LastError),
		TmuxSession:  session,
	}
	if err := queries.UpdateAgentStatusByTmuxSession(ctx, params); err != nil {
		return fmt.Errorf("update agent state for tmux session %q: %w", session, err)
	}
	return nil
}

func optionalToSQL(o agenthook.Optional) sql.NullString {
	if !o.Set {
		return sql.NullString{}
	}
	return sql.NullString{String: o.Value, Valid: true}
}
