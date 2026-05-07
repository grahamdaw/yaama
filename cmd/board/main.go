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

	program := tea.NewProgram(tui.NewModel(state), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "board exited with error: %v\n", err)
		os.Exit(1)
	}
}
