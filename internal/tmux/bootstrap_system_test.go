//go:build system

package tmux

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func tmuxBinary(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("tmux")
	if err != nil {
		t.Skipf("tmux not available on PATH: %v", err)
	}
	return path
}

func uniqueSessionName(t *testing.T) string {
	t.Helper()
	// Keep the name short — pane capture truncates at the default pane width
	// (~80 cols) and the env-probe assertion needs the full session string
	// to fit on one line of captured output.
	return fmt.Sprintf("yst-%d-%d", os.Getpid(), time.Now().UnixNano()%1_000_000)
}

func withTmuxServer(t *testing.T) {
	t.Helper()
	tmuxBinary(t)
	// tmux uses TMUX_TMPDIR/tmux-<uid>/default as a unix socket path; keep it
	// short so we don't hit the AF_UNIX sun_path limit (~104 on darwin) when
	// running under deeply nested t.TempDir() paths.
	dir, err := os.MkdirTemp("", "yst-")
	if err != nil {
		t.Fatalf("mkdir tmux tmpdir: %v", err)
	}
	// Guard: only ever remove a path we just created under os.TempDir() with
	// our well-known prefix. Belt-and-braces against accidental wider removal.
	parent := os.TempDir()
	safeToRemove := dir != "" &&
		dir != "/" &&
		strings.HasPrefix(filepath.Clean(dir), filepath.Clean(parent)) &&
		strings.HasPrefix(filepath.Base(dir), "yst-")
	t.Setenv("TMUX_TMPDIR", dir)
	t.Setenv("TMUX", "")
	t.Cleanup(func() {
		_ = exec.Command("tmux", "kill-server").Run()
		if safeToRemove {
			_ = os.RemoveAll(dir)
		}
	})
}

func runTmuxCapture(t *testing.T, args ...string) (string, error) {
	t.Helper()
	out, err := exec.Command("tmux", args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func assertSessionEnv(t *testing.T, session, key, want string) {
	t.Helper()
	out, err := runTmuxCapture(t, "show-environment", "-t", session, key)
	if err != nil {
		t.Fatalf("show-environment %s %s: %v (%s)", session, key, err, out)
	}
	expected := key + "=" + want
	if out != expected {
		t.Fatalf("env %s mismatch: got %q want %q", key, out, expected)
	}
}

func pollUntil(t *testing.T, timeout time.Duration, fn func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fn()
}

func captureShellEnv(t *testing.T, paneTarget, varName string) string {
	t.Helper()
	marker := "YAAMA_PROBE"
	cmd := fmt.Sprintf(`printf '\n%s:%%s\n' "$%s"`, marker, varName)
	if _, err := runTmuxCapture(t, "send-keys", "-t", paneTarget, cmd, "C-m"); err != nil {
		t.Fatalf("send-keys probe: %v", err)
	}
	var captured string
	ok := pollUntil(t, 3*time.Second, func() bool {
		out, err := runTmuxCapture(t, "capture-pane", "-p", "-t", paneTarget)
		if err != nil {
			return false
		}
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, marker+":") {
				captured = strings.TrimPrefix(line, marker+":")
				return captured != ""
			}
		}
		return false
	})
	if !ok {
		t.Fatalf("did not see %s marker in pane %s", marker, paneTarget)
	}
	return captured
}

func listWindowNames(t *testing.T, session string) []string {
	t.Helper()
	out, err := runTmuxCapture(t, "list-windows", "-t", session, "-F", "#{window_name}")
	if err != nil {
		t.Fatalf("list-windows: %v (%s)", err, out)
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	return names
}

func sessionExists(t *testing.T, session string) bool {
	t.Helper()
	err := exec.Command("tmux", "has-session", "-t", session).Run()
	return err == nil
}

func TestSystemBootstrapCreateAppliesFullLayout(t *testing.T) {
	withTmuxServer(t)

	workDir := t.TempDir()
	sentinelDir := t.TempDir()
	afterStartSentinel := filepath.Join(sentinelDir, "after_start.txt")
	agentSentinel := filepath.Join(sentinelDir, "agent.txt")

	session := uniqueSessionName(t)

	spec := BootstrapSpec{
		SessionName: session,
		WorkingDir:  workDir,
		AgentWindow: "agent",
		Windows: []BootstrapWindow{
			{Name: "ops"},
		},
		AfterStart: []string{
			fmt.Sprintf("date +%%s%%N >> %s", afterStartSentinel),
		},
		AgentCommand: []string{"touch", agentSentinel},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := BootstrapSession(ctx, spec); err != nil {
		t.Fatalf("BootstrapSession: %v", err)
	}
	t.Cleanup(func() {
		_ = exec.Command("tmux", "kill-session", "-t", session).Run()
	})

	names := listWindowNames(t, session)
	wantAgent, wantOps := false, false
	for _, n := range names {
		if n == "agent" {
			wantAgent = true
		}
		if n == "ops" {
			wantOps = true
		}
	}
	if !wantAgent || !wantOps {
		t.Fatalf("windows mismatch: got %v want agent + ops", names)
	}

	assertSessionEnv(t, session, "YAAMA_TMUX_SESSION", session)
	assertSessionEnv(t, session, "YAAMA_WORKING_DIR", workDir)

	got := captureShellEnv(t, session+":0.0", "YAAMA_TMUX_SESSION")
	if got != session {
		t.Fatalf("pane env YAAMA_TMUX_SESSION: got %q want %q", got, session)
	}

	if _, err := os.Stat(afterStartSentinel); err != nil {
		t.Fatalf("after_start sentinel missing: %v", err)
	}

	if !pollUntil(t, 5*time.Second, func() bool {
		_, err := os.Stat(agentSentinel)
		return err == nil
	}) {
		t.Fatalf("agent sentinel not created at %s", agentSentinel)
	}
}

func TestSystemBootstrapRecoveryRebuildsLayoutWithoutAgentCommand(t *testing.T) {
	withTmuxServer(t)

	workDir := t.TempDir()
	sentinelDir := t.TempDir()
	afterStartSentinel := filepath.Join(sentinelDir, "after_start.txt")
	createAgentSentinel := filepath.Join(sentinelDir, "agent_create.txt")
	recoveryAgentSentinel := filepath.Join(sentinelDir, "agent_recovery.txt")

	session := uniqueSessionName(t)

	makeSpec := func(agent []string) BootstrapSpec {
		return BootstrapSpec{
			SessionName: session,
			WorkingDir:  workDir,
			AgentWindow: "agent",
			Windows: []BootstrapWindow{
				{Name: "ops"},
			},
			AfterStart: []string{
				fmt.Sprintf("date +%%s%%N >> %s", afterStartSentinel),
			},
			AgentCommand: agent,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := BootstrapSession(ctx, makeSpec([]string{"touch", createAgentSentinel})); err != nil {
		t.Fatalf("create BootstrapSession: %v", err)
	}
	t.Cleanup(func() {
		_ = exec.Command("tmux", "kill-session", "-t", session).Run()
	})

	if !pollUntil(t, 5*time.Second, func() bool {
		_, err := os.Stat(createAgentSentinel)
		return err == nil
	}) {
		t.Fatalf("create agent sentinel never appeared at %s", createAgentSentinel)
	}

	if out, err := runTmuxCapture(t, "kill-session", "-t", session); err != nil {
		t.Fatalf("kill-session: %v (%s)", err, out)
	}
	if sessionExists(t, session) {
		t.Fatalf("session %s still exists after kill", session)
	}

	if err := BootstrapSession(ctx, makeSpec(nil)); err != nil {
		t.Fatalf("recovery BootstrapSession: %v", err)
	}

	names := listWindowNames(t, session)
	wantAgent, wantOps := false, false
	for _, n := range names {
		if n == "agent" {
			wantAgent = true
		}
		if n == "ops" {
			wantOps = true
		}
	}
	if !wantAgent || !wantOps {
		t.Fatalf("recovery windows mismatch: got %v want agent + ops", names)
	}

	assertSessionEnv(t, session, "YAAMA_TMUX_SESSION", session)
	assertSessionEnv(t, session, "YAAMA_WORKING_DIR", workDir)

	// Same poll budget the create case uses for sentinel-presence. If the agent
	// command had run, the sentinel would exist by now; treat absence after the
	// poll window as confirmation that recovery suppressed the agent command.
	if pollUntil(t, 5*time.Second, func() bool {
		_, err := os.Stat(recoveryAgentSentinel)
		return err == nil
	}) {
		t.Fatalf("recovery agent sentinel unexpectedly created at %s", recoveryAgentSentinel)
	}

	data, err := os.ReadFile(afterStartSentinel)
	if err != nil {
		t.Fatalf("read after_start sentinel: %v", err)
	}
	lines := 0
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) != "" {
			lines++
		}
	}
	if lines < 2 {
		t.Fatalf("after_start should have run twice; got %d entries: %q", lines, string(data))
	}
}
