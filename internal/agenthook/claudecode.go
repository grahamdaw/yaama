package agenthook

import (
	"encoding/json"
	"fmt"
	"strings"
)

func init() {
	Register(ClaudeCodeParser{})
}

// ClaudeCodeParser maps Claude Code hook payloads
// (https://docs.claude.com/en/docs/claude-code/hooks) onto agent state.
//
// Only the small subset of fields needed for board state is decoded; unknown
// fields are ignored so future hook additions don't break older binaries.
type ClaudeCodeParser struct{}

func (ClaudeCodeParser) Name() string { return "claude-code" }

type claudeCodePayload struct {
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
	Message       string `json:"message"`
	Prompt        string `json:"prompt"`
	Reason        string `json:"reason"`
	ToolResponse  struct {
		Error string `json:"error"`
	} `json:"tool_response"`
}

func (ClaudeCodeParser) Parse(raw []byte) (Event, error) {
	if len(raw) == 0 {
		return Event{}, fmt.Errorf("claude-code hook: empty payload")
	}
	var p claudeCodePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return Event{}, fmt.Errorf("claude-code hook: decode json: %w", err)
	}

	event := strings.TrimSpace(p.HookEventName)
	if event == "" {
		return Event{}, fmt.Errorf("claude-code hook: missing hook_event_name")
	}

	out := Event{EventName: event}

	switch event {
	case "SessionStart":
		out.Status = SetValue("running")
		out.LastActivity = SetValue("claude-code: session started")
	case "UserPromptSubmit":
		out.Status = SetValue("running")
		out.LastActivity = SetValue(truncate("claude-code: prompt — "+singleLine(p.Prompt), 200))
	case "PreToolUse":
		out.Status = SetValue("running")
		out.LastActivity = SetValue(toolActivity("running tool", p.ToolName))
	case "PostToolUse":
		if errMsg := strings.TrimSpace(p.ToolResponse.Error); errMsg != "" {
			out.LastError = SetValue(truncate("claude-code "+p.ToolName+": "+errMsg, 500))
			out.LastActivity = SetValue(toolActivity("tool error", p.ToolName))
		} else {
			out.LastActivity = SetValue(toolActivity("tool done", p.ToolName))
		}
	case "Notification":
		out.Status = SetValue("blocked")
		msg := strings.TrimSpace(p.Message)
		if msg == "" {
			msg = "awaiting user input"
		}
		out.LastActivity = SetValue(truncate("claude-code: "+singleLine(msg), 200))
	case "Stop", "SessionEnd":
		out.Status = SetValue("idle")
		reason := strings.TrimSpace(p.Reason)
		if reason == "" {
			out.LastActivity = SetValue("claude-code: stopped")
		} else {
			out.LastActivity = SetValue(truncate("claude-code: stopped — "+singleLine(reason), 200))
		}
	case "SubagentStop":
		out.LastActivity = SetValue("claude-code: subagent stopped")
	case "PreCompact":
		out.LastActivity = SetValue("claude-code: compacting context")
	default:
		// Unknown but well-formed event — record it so operators see the
		// agent is alive, but don't guess at a status transition.
		out.LastActivity = SetValue("claude-code: " + event)
	}

	return out, nil
}

func toolActivity(verb, tool string) string {
	tool = strings.TrimSpace(tool)
	if tool == "" {
		tool = "(unknown tool)"
	}
	return truncate("claude-code: "+verb+" "+tool, 200)
}

func singleLine(s string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
