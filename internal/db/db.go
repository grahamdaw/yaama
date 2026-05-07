package db

import (
	"os"
	"path/filepath"
)

type InitResult struct {
	Path    string
	Created bool
}

func Init(path string) (InitResult, error) {
	cleanPath := filepath.Clean(path)
	parent := filepath.Dir(cleanPath)
	if parent != "." {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return InitResult{}, err
		}
	}

	_, statErr := os.Stat(cleanPath)
	created := os.IsNotExist(statErr)
	if statErr != nil && !os.IsNotExist(statErr) {
		return InitResult{}, statErr
	}

	file, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return InitResult{}, err
	}
	if err := file.Close(); err != nil {
		return InitResult{}, err
	}

	return InitResult{
		Path:    cleanPath,
		Created: created,
	}, nil
}
