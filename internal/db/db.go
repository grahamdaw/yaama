package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grahamdaw/yaama/internal/db/generated"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed schema/*.sql
var schemaFS embed.FS

type InitResult struct {
	Path    string
	Created bool
	Conn    *sql.DB
	Queries *generated.Queries
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

	file, err := os.OpenFile(cleanPath, os.O_CREATE, 0o644)
	if err != nil {
		return InitResult{}, err
	}
	if err := file.Close(); err != nil {
		return InitResult{}, err
	}

	conn, err := sql.Open("sqlite", cleanPath)
	if err != nil {
		return InitResult{}, err
	}

	if err := runMigrations(conn, cleanPath); err != nil {
		_ = conn.Close()
		return InitResult{}, err
	}

	return InitResult{
		Path:    cleanPath,
		Created: created,
		Conn:    conn,
		Queries: generated.New(conn),
	}, nil
}

func runMigrations(conn *sql.DB, dbPath string) error {
	goose.SetBaseFS(schemaFS)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	if err := goose.Up(conn, "schema"); err != nil {
		return fmt.Errorf(
			"failed to apply database migrations for %q: %w; run `make migrate` to inspect or re-apply migrations",
			dbPath,
			err,
		)
	}

	return nil
}
