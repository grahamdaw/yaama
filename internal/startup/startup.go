package startup

import (
	"context"
	"fmt"

	"github.com/grahamdaw/yaama/internal/config"
	"github.com/grahamdaw/yaama/internal/db"
)

type State struct {
	Config  config.Config
	DB      db.InitResult
	Notices []string
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

	return State{
		Config:  cfg,
		DB:      dbState,
		Notices: notices,
	}, nil
}
