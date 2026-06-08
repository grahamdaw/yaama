package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/grahamdaw/yaama/internal/profile"
)

// runProfileCheckCommand handles `yaama profile check <name>`. It resolves
// the named profile against the current working directory, prints a
// human-readable plan of what would happen on launch (working dir, branch,
// agent command + env, git worktree commands, tmux commands, hooks), and
// exits with status 0 if the profile is valid. Nothing is executed.
func runProfileCheckCommand(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("profile check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "profile check error: %v\n", err)
		return 1
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		fmt.Fprintln(stderr, "usage: yaama profile check <name>")
		return 1
	}
	name := strings.TrimSpace(positionals[0])
	if name == "" {
		fmt.Fprintln(stderr, "profile check error: profile name must not be empty")
		return 1
	}

	cfg, err := profile.Load(name)
	if err != nil {
		fmt.Fprintf(stderr, "profile check error: %v\n", err)
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "profile check error: %v\n", err)
		return 1
	}

	branch := strings.TrimSpace(cfg.Repo.DefaultBranch)
	if branch == "" {
		branch = "main"
	}
	runtime, err := profile.ResolveRuntimeValues(cfg, cwd, "", branch)
	if err != nil {
		fmt.Fprintf(stderr, "profile check error: %v\n", err)
		return 1
	}

	fmt.Fprint(stdout, renderProfilePlan(cfg, runtime))
	return 0
}

// renderProfilePlan formats a multi-section plan for a resolved profile.
// The output is plain text — no colors — and lists each phase the launch
// would execute.
func renderProfilePlan(cfg profile.Config, runtime profile.RuntimeValues) string {
	var b strings.Builder
	sessionPlaceholder := "<session>"

	fmt.Fprintf(&b, "Profile: %s\n", cfg.Name)
	fmt.Fprintf(&b, "Harness: %s\n\n", cfg.Agent.Harness)

	b.WriteString("Resolved:\n")
	fmt.Fprintf(&b, "  1. working_dir = %s\n", runtime.WorkingDir)
	fmt.Fprintf(&b, "  2. branch      = %s\n", runtime.Branch)
	fmt.Fprintf(&b, "  3. agent       = %s\n", strings.Join(runtime.AgentCommand, " "))
	if len(cfg.Agent.Env) > 0 {
		keys := sortedKeys(cfg.Agent.Env)
		fmt.Fprintf(&b, "  4. agent_env:\n")
		for _, k := range keys {
			fmt.Fprintf(&b, "       %s=%s\n", k, cfg.Agent.Env[k])
		}
	}
	b.WriteString("\n")

	b.WriteString("Git:\n")
	worktreePath := fmt.Sprintf("%s/.yaama-worktrees/%s", runtime.WorkingDir, sessionPlaceholder)
	fmt.Fprintf(&b, "  1. git -C %s worktree add %s %s\n", runtime.WorkingDir, worktreePath, runtime.Branch)
	b.WriteString("\n")

	b.WriteString("Tmux:\n")
	step := 1
	fmt.Fprintf(&b, "  %d. tmux new-session -d -s %s -c %s\n", step, sessionPlaceholder, worktreePath)
	step++
	fmt.Fprintf(&b, "  %d. tmux set-environment -t %s YAAMA_TMUX_SESSION %s\n", step, sessionPlaceholder, sessionPlaceholder)
	step++
	fmt.Fprintf(&b, "  %d. tmux set-environment -t %s YAAMA_WORKING_DIR %s\n", step, sessionPlaceholder, worktreePath)
	step++
	fmt.Fprintf(&b, "  %d. tmux rename-window -t %s:0 agent\n", step, sessionPlaceholder)
	step++
	for _, w := range cfg.Tmux.Windows {
		fmt.Fprintf(&b, "  %d. tmux new-window -d -t %s -n %s -c %s\n", step, sessionPlaceholder, w.Name, worktreePath)
		step++
		for i, p := range w.Panes {
			if i == 0 {
				continue
			}
			split := "-v"
			if strings.EqualFold(strings.TrimSpace(p.Split), "horizontal") {
				split = "-h"
			}
			args := fmt.Sprintf("tmux split-window -t %s:%s.0 %s", sessionPlaceholder, w.Name, split)
			if size := strings.TrimSpace(p.Size); size != "" {
				args += " -l " + size
			}
			fmt.Fprintf(&b, "  %d. %s -c %s\n", step, args, paneCwd(worktreePath, p.Cwd))
			step++
		}
		for i, p := range w.Panes {
			if strings.TrimSpace(p.Run) == "" {
				continue
			}
			fmt.Fprintf(&b, "  %d. tmux send-keys -t %s:%s.%d %q C-m\n", step, sessionPlaceholder, w.Name, i, p.Run)
			step++
		}
		if layout := strings.TrimSpace(w.Layout); layout != "" {
			fmt.Fprintf(&b, "  %d. tmux select-layout -t %s:%s %s\n", step, sessionPlaceholder, w.Name, layout)
			step++
		}
	}
	if len(runtime.AgentCommand) > 0 {
		fmt.Fprintf(&b, "  %d. tmux send-keys -t %s:0.0 %q C-m\n", step, sessionPlaceholder, strings.Join(runtime.AgentCommand, " "))
		step++
	}
	b.WriteString("\n")

	b.WriteString("Hooks:\n")
	if len(cfg.Scripts.BeforeStart) == 0 && len(cfg.Scripts.AfterStart) == 0 {
		b.WriteString("  (none)\n")
	} else {
		idx := 1
		for _, h := range cfg.Scripts.BeforeStart {
			fmt.Fprintf(&b, "  %d. before_start: %s\n", idx, h)
			idx++
		}
		for _, h := range cfg.Scripts.AfterStart {
			fmt.Fprintf(&b, "  %d. after_start:  %s\n", idx, h)
			idx++
		}
	}

	return b.String()
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func paneCwd(workingDir, cwd string) string {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" || cwd == "." {
		return workingDir
	}
	if strings.HasPrefix(cwd, "/") {
		return cwd
	}
	return workingDir + "/" + cwd
}
