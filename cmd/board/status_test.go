package main

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grahamdaw/yaama/internal/db"
	"github.com/grahamdaw/yaama/internal/db/generated"
)

func TestParseStatusArgsRejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	_, _, err := parseStatusArgs([]string{"invalid"})
	if err == nil {
		t.Fatalf("expected invalid status error")
	}
	if !strings.Contains(err.Error(), "accepted: idle, running, blocked, review, done") {
		t.Fatalf("expected accepted status hint, got %q", err.Error())
	}
}

func TestExecuteStatusUpdateFailsOutsideTmux(t *testing.T) {
	t.Parallel()

	result := openTestDB(t)
	_, err := result.Queries.CreateAgent(context.Background(), createAgentParams())
	if err != nil {
		t.Fatalf("CreateAgent returned error: %v", err)
	}

	err = executeStatusUpdate(
		context.Background(),
		result.Queries,
		func(context.Context) (string, error) { return "", nil },
		statusCommandInput{status: "running"},
	)
	if !errors.Is(err, errOutsideTmux) {
		t.Fatalf("expected errOutsideTmux, got %v", err)
	}
}

func TestExecuteStatusUpdateFailsWhenSessionMissing(t *testing.T) {
	t.Parallel()

	result := openTestDB(t)

	err := executeStatusUpdate(
		context.Background(),
		result.Queries,
		func(context.Context) (string, error) { return "ghost-session", nil },
		statusCommandInput{status: "running"},
	)
	var missingErr missingAgentError
	if !errors.As(err, &missingErr) {
		t.Fatalf("expected missingAgentError, got %v", err)
	}
}

func TestExecuteStatusUpdateWritesStatusAndMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	result := openTestDB(t)

	created, err := result.Queries.CreateAgent(ctx, createAgentParams())
	if err != nil {
		t.Fatalf("CreateAgent returned error: %v", err)
	}
	if created.LastHeartbeatAt.Valid {
		t.Fatalf("expected new record to start without heartbeat in this test fixture")
	}

	err = executeStatusUpdate(
		ctx,
		result.Queries,
		func(context.Context) (string, error) { return "agent-1", nil },
		statusCommandInput{
			status:   "blocked",
			task:     optionalString{value: "investigate CI failure", set: true},
			activity: optionalString{value: "waiting on flaky integration test", set: true},
			branch:   optionalString{value: "feat/10-cli-status-parity", set: true},
		},
	)
	if err != nil {
		t.Fatalf("executeStatusUpdate returned error: %v", err)
	}

	updated, err := result.Queries.GetAgentByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetAgentByID returned error: %v", err)
	}

	if updated.Status != "blocked" {
		t.Fatalf("expected blocked status, got %q", updated.Status)
	}
	if !updated.Task.Valid || updated.Task.String != "investigate CI failure" {
		t.Fatalf("expected task update to persist, got %+v", updated.Task)
	}
	if !updated.LastActivity.Valid || updated.LastActivity.String != "waiting on flaky integration test" {
		t.Fatalf("expected activity update to persist, got %+v", updated.LastActivity)
	}
	if !updated.Branch.Valid || updated.Branch.String != "feat/10-cli-status-parity" {
		t.Fatalf("expected branch update to persist, got %+v", updated.Branch)
	}
	if !updated.LastHeartbeatAt.Valid {
		t.Fatalf("expected heartbeat timestamp to be set")
	}
}

func TestExecuteStatusUpdatePreservesUnsetOptionalFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	result := openTestDB(t)

	created, err := result.Queries.CreateAgent(ctx, createAgentParams())
	if err != nil {
		t.Fatalf("CreateAgent returned error: %v", err)
	}

	err = executeStatusUpdate(
		ctx,
		result.Queries,
		func(context.Context) (string, error) { return "agent-1", nil },
		statusCommandInput{status: "running"},
	)
	if err != nil {
		t.Fatalf("executeStatusUpdate returned error: %v", err)
	}

	updated, err := result.Queries.GetAgentByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetAgentByID returned error: %v", err)
	}

	if !updated.Task.Valid || updated.Task.String != created.Task.String {
		t.Fatalf("expected task to remain unchanged, got %+v", updated.Task)
	}
	if !updated.LastActivity.Valid || updated.LastActivity.String != created.LastActivity.String {
		t.Fatalf("expected activity to remain unchanged, got %+v", updated.LastActivity)
	}
	if !updated.Branch.Valid || updated.Branch.String != created.Branch.String {
		t.Fatalf("expected branch to remain unchanged, got %+v", updated.Branch)
	}
	if !updated.LastHeartbeatAt.Valid {
		t.Fatalf("expected heartbeat to be set on status update")
	}
}

func openTestDB(t *testing.T) db.InitResult {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "yaama.db")
	result, err := db.Init(dbPath)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	t.Cleanup(func() { _ = result.Conn.Close() })
	return result
}

func createAgentParams() generated.CreateAgentParams {
	return generated.CreateAgentParams{
		Name:          "agent 1",
		TmuxSession:   "agent-1",
		Status:        "idle",
		Task:          sql.NullString{String: "initial task", Valid: true},
		LastActivity:  sql.NullString{String: "initial activity", Valid: true},
		Branch:        sql.NullString{String: "feat/old", Valid: true},
		WorkingDir:    sql.NullString{String: "/tmp/work", Valid: true},
		ProfileName:   sql.NullString{String: "default", Valid: true},
		TicketID:      sql.NullString{String: "YAAMA-10", Valid: true},
		InitialPrompt: sql.NullString{String: "Initial prompt", Valid: true},
	}
}
