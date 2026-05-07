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

func Bootstrap(_ context.Context) (State, error) {
	cfg, err := config.Load()
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
