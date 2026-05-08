package startup

import (
	"context"
	"fmt"

	"github.com/grahamdaw/yaama/internal/config"
	"github.com/grahamdaw/yaama/internal/db"
	"github.com/grahamdaw/yaama/internal/tmux"
)

type State struct {
	Config        config.Config
	DB            db.InitResult
	Notices       []string
	TmuxAvailable bool
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

	dbState, err := db.Init(cfg.DBPath)
	if err != nil {
		return State{}, err
	}

	notices := []string{}
	if dbState.Created {
		notices = append(notices, fmt.Sprintf("Initialized DB at %s", dbState.Path))
	}
	tmuxAvailable := tmux.IsAvailable()
	if !tmuxAvailable {
		notices = append(notices, "tmux unavailable in PATH; attach actions are disabled.")
	}

	return State{
		Config:        cfg,
		DB:            dbState,
		Notices:       notices,
		TmuxAvailable: tmuxAvailable,
	}, nil
}
