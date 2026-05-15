package startup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/grahamdaw/yaama/internal/config"
	"github.com/grahamdaw/yaama/internal/db"
	"github.com/grahamdaw/yaama/internal/logging"
	"github.com/grahamdaw/yaama/internal/tmux"
)

type noopCloser struct{}

func (noopCloser) Close() error { return nil }

type State struct {
	Config        config.Config
	DB            db.InitResult
	Notices       []string
	TmuxAvailable bool
	Logger        *slog.Logger
	LogPath       string
	LogClose      io.Closer
}

type Options struct {
	DBPathOverride string
}

func Bootstrap(_ context.Context, opts Options) (State, error) {
	cfg, err := config.Load(config.LoadOptions{
		DBPathOverride: opts.DBPathOverride,
	})
	if err != nil {
		return State{}, err
	}

	notices := []string{}
	logResult, logErr := logging.New(logging.Options{
		LevelEnv: os.Getenv("YAAMA_LOG_LEVEL"),
		PID:      os.Getpid(),
	})
	logger := logResult.Logger
	logPath := logResult.Path
	logCloser := logResult.Closer
	if logErr != nil {
		logger = logging.Discard()
		logPath = ""
		logCloser = noopCloser{}
		notices = append(notices, fmt.Sprintf("log file unavailable: %v", logErr))
	}
	logger.Info("startup.begin", "db_override", opts.DBPathOverride)

	dbState, err := db.Init(cfg.DBPath)
	if err != nil {
		logger.Error("startup.db_open_failed", "path", cfg.DBPath, "err", err.Error())
		_ = logCloser.Close()
		return State{}, err
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

	return State{
		Config:        cfg,
		DB:            dbState,
		Notices:       notices,
		TmuxAvailable: tmuxAvailable,
		Logger:        logger,
		LogPath:       logPath,
		LogClose:      logCloser,
	}, nil
}
