package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/grahamdaw/yaama/internal/startup"
	"github.com/grahamdaw/yaama/internal/tui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "status" {
		exitCode := runStatusCommand(context.Background(), os.Args[2:], os.Stderr)
		os.Exit(exitCode)
	}

	if len(os.Args) > 1 && os.Args[1] == "hook" {
		exitCode := runHookCommand(context.Background(), os.Args[2:], os.Stdin, os.Stderr)
		os.Exit(exitCode)
	}

	dbPath := flag.String("db", "", "path to SQLite DB file")
	flag.Parse()

	ctx := context.Background()

	state, err := startup.Bootstrap(ctx, startup.Options{
		DBPathOverride: *dbPath,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "startup failed: %v\n", err)
		os.Exit(1)
	}
	if state.LogClose != nil {
		defer state.LogClose.Close()
	}

	program := tea.NewProgram(tui.NewModel(state), tea.WithAltScreen())
	if state.Logger != nil {
		state.Logger.Info("startup.ready", "log_path", state.LogPath)
	}
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "yaama exited with error: %v\n", err)
		os.Exit(1)
	}
}
