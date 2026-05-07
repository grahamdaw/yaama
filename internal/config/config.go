package config

import (
	"os"
	"path/filepath"
)

const defaultDBPath = "./yaama.db"

type Config struct {
	DBPath string
}

func Load() (Config, error) {
	if envPath := os.Getenv("YAAMA_DB"); envPath != "" {
		return Config{DBPath: filepath.Clean(envPath)}, nil
	}

	return Config{DBPath: defaultDBPath}, nil
}
