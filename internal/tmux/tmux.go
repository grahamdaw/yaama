package tmux

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var ErrTmuxUnavailable = errors.New("tmux binary not found in PATH")

func IsAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func ListSessions(ctx context.Context) ([]string, error) {
	if !IsAvailable() {
		return nil, ErrTmuxUnavailable
	}

	out, err := exec.CommandContext(ctx, "tmux", "list-sessions", "-F", "#{session_name}").CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(out))
		if isNoTmuxServerOutput(trimmed) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("list tmux sessions: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	sessions := make([]string, 0, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name != "" {
			sessions = append(sessions, name)
		}
	}
	return sessions, nil
}

func isNoTmuxServerOutput(output string) bool {
	normalized := strings.ToLower(strings.TrimSpace(output))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "no server running") ||
		strings.Contains(normalized, "failed to connect to server") ||
		strings.Contains(normalized, "error connecting to") ||
		strings.Contains(normalized, "no such file or directory")
}

func CurrentSession(ctx context.Context) (string, error) {
	if !IsAvailable() {
		return "", ErrTmuxUnavailable
	}
	if strings.TrimSpace(os.Getenv("TMUX")) == "" {
		return "", nil
	}

	out, err := exec.CommandContext(ctx, "tmux", "display-message", "-p", "#S").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resolve current tmux session: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func AttachOrSwitchCommand(ctx context.Context, targetSession string) (*exec.Cmd, error) {
	if !IsAvailable() {
		return nil, ErrTmuxUnavailable
	}
	targetSession = strings.TrimSpace(targetSession)
	if targetSession == "" {
		return nil, errors.New("target tmux session is empty")
	}

	currentSession, err := CurrentSession(ctx)
	if err != nil {
		return nil, err
	}
	if currentSession != "" {
		return exec.CommandContext(ctx, "tmux", "switch-client", "-t", targetSession), nil
	}
	return exec.CommandContext(ctx, "tmux", "attach-session", "-t", targetSession), nil
}

func CreateDetachedSessionCommand(ctx context.Context, targetSession string, workingDir string) (*exec.Cmd, error) {
	if !IsAvailable() {
		return nil, ErrTmuxUnavailable
	}
	targetSession = strings.TrimSpace(targetSession)
	if targetSession == "" {
		return nil, errors.New("target tmux session is empty")
	}
	workingDir = strings.TrimSpace(workingDir)
	if workingDir == "" {
		return nil, errors.New("working directory is empty")
	}

	return exec.CommandContext(ctx, "tmux", "new-session", "-d", "-s", targetSession, "-c", workingDir), nil
}

func KillSession(ctx context.Context, targetSession string) error {
	if !IsAvailable() {
		return ErrTmuxUnavailable
	}
	targetSession = strings.TrimSpace(targetSession)
	if targetSession == "" {
		return errors.New("target tmux session is empty")
	}

	out, err := exec.CommandContext(ctx, "tmux", "kill-session", "-t", targetSession).CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(out))
		if isNoTmuxServerOutput(trimmed) || isMissingTmuxSessionOutput(trimmed) {
			// Already gone should be treated as idempotent success.
			return nil
		}
		return fmt.Errorf("kill tmux session %q: %w (%s)", targetSession, err, trimmed)
	}
	return nil
}

func isMissingTmuxSessionOutput(output string) bool {
	normalized := strings.ToLower(strings.TrimSpace(output))
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, "can't find session") ||
		strings.Contains(normalized, "no such session")
}
