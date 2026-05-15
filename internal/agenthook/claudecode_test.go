package agenthook

import (
	"strings"
	"testing"
)

func TestClaudeCodeParserMapsKnownEvents(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		payload        string
		wantStatus     string
		wantActivityIn string
		wantErrorIn    string
	}{
		{
			name:           "session start runs",
			payload:        `{"hook_event_name":"SessionStart"}`,
			wantStatus:     "running",
			wantActivityIn: "session started",
		},
		{
			name:           "pretooluse names tool",
			payload:        `{"hook_event_name":"PreToolUse","tool_name":"Bash"}`,
			wantStatus:     "running",
			wantActivityIn: "running tool Bash",
		},
		{
			name:           "posttooluse with error records last_error",
			payload:        `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_response":{"error":"exit 1"}}`,
			wantActivityIn: "tool error Bash",
			wantErrorIn:    "exit 1",
		},
		{
			name:           "notification flips to blocked",
			payload:        `{"hook_event_name":"Notification","message":"awaiting approval"}`,
			wantStatus:     "blocked",
			wantActivityIn: "awaiting approval",
		},
		{
			name:           "stop returns to idle",
			payload:        `{"hook_event_name":"Stop"}`,
			wantStatus:     "idle",
			wantActivityIn: "stopped",
		},
		{
			name:           "unknown event records activity but leaves status alone",
			payload:        `{"hook_event_name":"MysteryEvent"}`,
			wantStatus:     "",
			wantActivityIn: "MysteryEvent",
		},
	}

	parser := ClaudeCodeParser{}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ev, err := parser.Parse([]byte(tc.payload))
			if err != nil {
				t.Fatalf("Parse returned error: %v", err)
			}
			if tc.wantStatus == "" {
				if ev.Status.Set {
					t.Fatalf("expected unset status, got %+v", ev.Status)
				}
			} else {
				if !ev.Status.Set || ev.Status.Value != tc.wantStatus {
					t.Fatalf("expected status %q, got %+v", tc.wantStatus, ev.Status)
				}
			}
			if !ev.LastActivity.Set || !strings.Contains(ev.LastActivity.Value, tc.wantActivityIn) {
				t.Fatalf("expected activity to contain %q, got %+v", tc.wantActivityIn, ev.LastActivity)
			}
			if tc.wantErrorIn != "" {
				if !ev.LastError.Set || !strings.Contains(ev.LastError.Value, tc.wantErrorIn) {
					t.Fatalf("expected last_error to contain %q, got %+v", tc.wantErrorIn, ev.LastError)
				}
			} else if ev.LastError.Set {
				t.Fatalf("expected last_error to be unset, got %+v", ev.LastError)
			}
		})
	}
}

func TestClaudeCodeParserRejectsEmptyAndMalformed(t *testing.T) {
	t.Parallel()

	parser := ClaudeCodeParser{}

	if _, err := parser.Parse(nil); err == nil {
		t.Fatalf("expected error for empty payload")
	}
	if _, err := parser.Parse([]byte("{")); err == nil {
		t.Fatalf("expected error for malformed json")
	}
	if _, err := parser.Parse([]byte(`{"tool_name":"Bash"}`)); err == nil {
		t.Fatalf("expected error when hook_event_name missing")
	}
}

func TestLookupAndNamesIncludeClaudeCode(t *testing.T) {
	t.Parallel()

	if _, ok := Lookup("claude-code"); !ok {
		t.Fatalf("expected claude-code parser to be registered")
	}
	if _, ok := Lookup("CLAUDE-CODE"); !ok {
		t.Fatalf("expected lookup to be case-insensitive")
	}
	found := false
	for _, n := range Names() {
		if n == "claude-code" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Names() to include claude-code; got %v", Names())
	}
}
