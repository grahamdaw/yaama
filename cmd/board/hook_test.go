package main

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/grahamdaw/yaama/internal/agenthook"
	"github.com/grahamdaw/yaama/internal/db/generated"
)

func TestParseHookArgsRequiresAgentAndStdin(t *testing.T) {
	t.Parallel()

	if _, _, err := parseHookArgs(nil, strings.NewReader(`{"hook_event_name":"Stop"}`)); err == nil {
		t.Fatalf("expected usage error when no agent given")
	}
	if _, _, err := parseHookArgs([]string{"claude-code"}, strings.NewReader("")); err == nil {
		t.Fatalf("expected error when stdin payload empty")
	}
}

func TestParseHookArgsReadsPayloadFromStdin(t *testing.T) {
	t.Parallel()

	input, _, err := parseHookArgs([]string{"claude-code"}, strings.NewReader(`{"hook_event_name":"Stop"}`))
	if err != nil {
		t.Fatalf("parseHookArgs error: %v", err)
	}
	if input.agent != "claude-code" {
		t.Fatalf("expected agent claude-code, got %q", input.agent)
	}
	if !strings.Contains(string(input.raw), "Stop") {
		t.Fatalf("expected raw payload to contain Stop, got %s", string(input.raw))
	}
}

func TestExecuteHookUpdateFailsOutsideTmux(t *testing.T) {
	t.Parallel()

	result := openTestDB(t)
	if _, err := result.Queries.CreateAgent(context.Background(), createAgentParams()); err != nil {
		t.Fatalf("CreateAgent error: %v", err)
	}

	err := executeHookUpdate(
		context.Background(),
		result.Queries,
		func(context.Context) (string, error) { return "", nil },
		agenthook.Event{Status: agenthook.SetValue("running")},
	)
	if !errors.Is(err, errOutsideTmux) {
		t.Fatalf("expected errOutsideTmux, got %v", err)
	}
}

func TestExecuteHookUpdateFailsWhenSessionMissing(t *testing.T) {
	t.Parallel()

	result := openTestDB(t)
	err := executeHookUpdate(
		context.Background(),
		result.Queries,
		func(context.Context) (string, error) { return "ghost", nil },
		agenthook.Event{Status: agenthook.SetValue("running")},
	)
	var missingErr missingAgentError
	if !errors.As(err, &missingErr) {
		t.Fatalf("expected missingAgentError, got %v", err)
	}
}

func TestExecuteHookUpdateAppliesStatusAndActivity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	result := openTestDB(t)
	created, err := result.Queries.CreateAgent(ctx, createAgentParams())
	if err != nil {
		t.Fatalf("CreateAgent error: %v", err)
	}

	event := agenthook.Event{
		EventName:    "Notification",
		Status:       agenthook.SetValue("blocked"),
		LastActivity: agenthook.SetValue("awaiting approval"),
	}
	err = executeHookUpdate(
		ctx,
		result.Queries,
		func(context.Context) (string, error) { return "agent-1", nil },
		event,
	)
	if err != nil {
		t.Fatalf("executeHookUpdate error: %v", err)
	}

	updated, err := result.Queries.GetAgentByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetAgentByID error: %v", err)
	}
	if updated.Status != "blocked" {
		t.Fatalf("expected status blocked, got %q", updated.Status)
	}
	if !updated.LastActivity.Valid || updated.LastActivity.String != "awaiting approval" {
		t.Fatalf("expected activity update, got %+v", updated.LastActivity)
	}
	if !updated.LastHeartbeatAt.Valid {
		t.Fatalf("expected heartbeat to be set")
	}
}

func TestExecuteHookUpdatePreservesStatusWhenEventLeavesItUnset(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	result := openTestDB(t)
	created, err := result.Queries.CreateAgent(ctx, createAgentParams())
	if err != nil {
		t.Fatalf("CreateAgent error: %v", err)
	}

	// Seed status to "running" so the next hook (status-less) is observed
	// to leave it intact rather than reset it.
	if err := result.Queries.UpdateAgentStatusByTmuxSession(ctx, generated.UpdateAgentStatusByTmuxSessionParams{
		Status:      "running",
		TmuxSession: created.TmuxSession,
		LastError:   sql.NullString{},
	}); err != nil {
		t.Fatalf("seed running status error: %v", err)
	}

	event := agenthook.Event{
		EventName:    "SubagentStop",
		LastActivity: agenthook.SetValue("subagent stopped"),
	}
	if err := executeHookUpdate(
		ctx,
		result.Queries,
		func(context.Context) (string, error) { return created.TmuxSession, nil },
		event,
	); err != nil {
		t.Fatalf("executeHookUpdate error: %v", err)
	}

	updated, err := result.Queries.GetAgentByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetAgentByID error: %v", err)
	}
	if updated.Status != "running" {
		t.Fatalf("expected status to remain running, got %q", updated.Status)
	}
	if !updated.LastActivity.Valid || updated.LastActivity.String != "subagent stopped" {
		t.Fatalf("expected activity update, got %+v", updated.LastActivity)
	}
}
