package profile

import (
	"os"
	"path/filepath"
)

const defaultDBPath = "./yaama.db"

type RuntimeConfig struct {
	DBPath string
}

type ConfigOptions struct {
	DBPathOverride string
}

func LoadConfig(opts ConfigOptions) (RuntimeConfig, error) {
	if opts.DBPathOverride != "" {
		return RuntimeConfig{DBPath: filepath.Clean(opts.DBPathOverride)}, nil
	}

	if envPath := os.Getenv("YAAMA_DB"); envPath != "" {
		return RuntimeConfig{DBPath: filepath.Clean(envPath)}, nil
	}

	return RuntimeConfig{DBPath: defaultDBPath}, nil
}
