package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/grahamdaw/yaama/internal/db"
	"github.com/grahamdaw/yaama/internal/logging"
	"github.com/grahamdaw/yaama/internal/profile"
	"github.com/grahamdaw/yaama/internal/tmux"
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

	if err := runBoard(*dbPath); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func runBoard(dbPathOverride string) error {
	params, cleanup, err := bootstrapBoard(dbPathOverride)
	if err != nil {
		return err
	}
	defer cleanup()

	params.Logger.Info("startup.ready", "log_path", params.LogPath)
	if _, err := tea.NewProgram(tui.NewModel(params), tea.WithAltScreen()).Run(); err != nil {
		return fmt.Errorf("yaama exited with error: %w", err)
	}
	return nil
}

func bootstrapBoard(dbPathOverride string) (tui.Params, func(), error) {
	cfg, err := profile.LoadConfig(profile.ConfigOptions{
		DBPathOverride: dbPathOverride,
	})
	if err != nil {
		return tui.Params{}, func() {}, fmt.Errorf("startup failed: %w", err)
	}

	logResult, logErr := logging.New(logging.Options{
		LevelEnv: os.Getenv("YAAMA_LOG_LEVEL"),
		PID:      os.Getpid(),
	})
	logger := logResult.Logger
	logPath := logResult.Path
	closeLog := func() { _ = logResult.Closer.Close() }
	if logErr != nil {
		logger = logging.Discard()
		logPath = ""
		closeLog = func() {}
	}
	logger.Info("startup.begin", "db_override", dbPathOverride)

	notices := []string{}
	if logErr != nil {
		notices = append(notices, fmt.Sprintf("log file unavailable: %v", logErr))
	}

	dbState, err := db.Init(cfg.DBPath)
	if err != nil {
		logger.Error("startup.db_open_failed", "path", cfg.DBPath, "err", err.Error())
		closeLog()
		return tui.Params{}, func() {}, fmt.Errorf("startup failed: %w", err)
	}
	logger.Info("startup.db_open", "path", dbState.Path, "created", dbState.Created)
	if dbState.Created {
		notices = append(notices, fmt.Sprintf("Initialized DB at %s", dbState.Path))
	}

	tmuxAvailable := tmux.IsAvailable()
	logger.Info("startup.tmux_detect", "available", tmuxAvailable)
	if !tmuxAvailable {
		notices = append(notices, "tmux unavailable in PATH; attach actions are disabled.")
	}

	cleanup := func() {
		_ = dbState.Conn.Close()
		closeLog()
	}
	return tui.Params{
		Queries:       dbState.Queries,
		Notices:       notices,
		TmuxAvailable: tmuxAvailable,
		Logger:        logger,
		LogPath:       logPath,
	}, cleanup, nil
}
