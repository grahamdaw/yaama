package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/grahamdaw/yaama/internal/db/generated"
)

func TestInitCreatesDBAndAppliesMigrations(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "state", "yaama.db")

	result, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	t.Cleanup(func() { _ = result.Conn.Close() })

	if !result.Created {
		t.Fatalf("expected Created to be true for a new DB")
	}

	if result.Path != dbPath {
		t.Fatalf("expected path %q, got %q", dbPath, result.Path)
	}

	var tableCount int
	if err := result.Conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'agents'").Scan(&tableCount); err != nil {
		t.Fatalf("querying sqlite_master failed: %v", err)
	}
	if tableCount != 1 {
		t.Fatalf("expected agents table to exist")
	}

	var indexCount int
	if err := result.Conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_agents_status'").Scan(&indexCount); err != nil {
		t.Fatalf("querying index metadata failed: %v", err)
	}
	if indexCount != 1 {
		t.Fatalf("expected idx_agents_status index to exist")
	}
}

func TestAgentQueriesCRUDAndLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "yaama.db")

	result, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	t.Cleanup(func() { _ = result.Conn.Close() })

	created, err := result.Queries.CreateAgent(ctx, createAgentParams())
	if err != nil {
		t.Fatalf("CreateAgent returned error: %v", err)
	}

	bySession, err := result.Queries.GetAgentByTmuxSession(ctx, "agent-1")
	if err != nil {
		t.Fatalf("GetAgentByTmuxSession returned error: %v", err)
	}
	if bySession.ID != created.ID {
		t.Fatalf("expected lookup by session to return created row")
	}

	if err := result.Queries.UpdateAgentStatusByTmuxSession(ctx, updateStatusParams("agent-1")); err != nil {
		t.Fatalf("UpdateAgentStatusByTmuxSession returned error: %v", err)
	}

	updated, err := result.Queries.GetAgentByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetAgentByID returned error: %v", err)
	}
	if updated.Status != "running" {
		t.Fatalf("expected status to be running, got %q", updated.Status)
	}
	if !updated.LastHeartbeatAt.Valid {
		t.Fatalf("expected heartbeat timestamp to be set after status update")
	}

	if err := result.Queries.UpdateAgentCleanupState(ctx, updateCleanupStateParams(created.ID, "archived")); err != nil {
		t.Fatalf("UpdateAgentCleanupState returned error: %v", err)
	}

	activeAgents, err := result.Queries.ListActiveAgents(ctx)
	if err != nil {
		t.Fatalf("ListActiveAgents returned error: %v", err)
	}
	if len(activeAgents) != 0 {
		t.Fatalf("expected archived agent to be hidden from active list")
	}

	if err := result.Queries.DeleteAgent(ctx, created.ID); err != nil {
		t.Fatalf("DeleteAgent returned error: %v", err)
	}

	allAgents, err := result.Queries.ListAgents(ctx)
	if err != nil {
		t.Fatalf("ListAgents returned error: %v", err)
	}
	if len(allAgents) != 0 {
		t.Fatalf("expected no agents after delete, got %d", len(allAgents))
	}
}

func TestSchemaConstraints(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "yaama.db")

	result, err := Init(dbPath)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	t.Cleanup(func() { _ = result.Conn.Close() })

	badCreate := createAgentParams()
	badCreate.Status = "invalid"
	if _, err := result.Queries.CreateAgent(ctx, badCreate); err == nil {
		t.Fatalf("expected status CHECK constraint to reject invalid value")
	}

	created, err := result.Queries.CreateAgent(ctx, createAgentParams())
	if err != nil {
		t.Fatalf("CreateAgent returned error: %v", err)
	}

	if err := result.Queries.UpdateAgentCleanupState(ctx, updateCleanupStateParams(created.ID, "deleted")); err == nil {
		t.Fatalf("expected cleanup_state CHECK constraint to reject invalid value")
	}
}

func createAgentParams() generatedCreateAgentParams {
	return generatedCreateAgentParams{
		Name:          "agent 1",
		TmuxSession:   "agent-1",
		Status:        "idle",
		Task:          sql.NullString{String: "implement db task", Valid: true},
		LastActivity:  sql.NullString{String: "bootstrapping", Valid: true},
		Branch:        sql.NullString{String: "feat/db", Valid: true},
		WorkingDir:    sql.NullString{String: "/tmp/work", Valid: true},
		ProfileName:   sql.NullString{String: "default", Valid: true},
		TicketID:      sql.NullString{String: "YAAMA-2", Valid: true},
		InitialPrompt: sql.NullString{String: "Add migration runner", Valid: true},
	}
}

func updateStatusParams(session string) generatedUpdateStatusParams {
	return generatedUpdateStatusParams{
		TmuxSession:  session,
		Status:       "running",
		Task:         sql.NullString{String: "running migrations", Valid: true},
		LastActivity: sql.NullString{String: "executing sqlc queries", Valid: true},
		Branch:       sql.NullString{String: "feat/data-layer", Valid: true},
	}
}

func updateCleanupStateParams(id int64, state string) generatedUpdateCleanupStateParams {
	return generatedUpdateCleanupStateParams{
		ID:           id,
		CleanupState: state,
	}
}

type generatedCreateAgentParams = generated.CreateAgentParams
type generatedUpdateStatusParams = generated.UpdateAgentStatusByTmuxSessionParams
type generatedUpdateCleanupStateParams = generated.UpdateAgentCleanupStateParams
