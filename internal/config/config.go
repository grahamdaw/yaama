package config

import (
	"os"
	"path/filepath"
)

const defaultDBPath = "./yaama.db"

type Config struct {
	DBPath string
}

type LoadOptions struct {
	DBPathOverride string
}

func Load(opts LoadOptions) (Config, error) {
	if opts.DBPathOverride != "" {
		return Config{DBPath: filepath.Clean(opts.DBPathOverride)}, nil
	}

	if envPath := os.Getenv("YAAMA_DB"); envPath != "" {
		return Config{DBPath: filepath.Clean(envPath)}, nil
	}

	return Config{DBPath: defaultDBPath}, nil
}
