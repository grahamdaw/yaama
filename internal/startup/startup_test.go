package startup

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapInitializesFreshDBWithoutManualMigrations(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "fresh", "yaama.db")

	state, err := Bootstrap(context.Background(), Options{
		DBPathOverride: dbPath,
	})
	if err != nil {
		t.Fatalf("Bootstrap returned error: %v", err)
	}
	t.Cleanup(func() {
		if state.DB.Conn != nil {
			_ = state.DB.Conn.Close()
		}
	})

	if !state.DB.Created {
		t.Fatalf("expected bootstrap to create DB for fresh path")
	}
	if state.DB.Path != dbPath {
		t.Fatalf("expected DB path %q, got %q", dbPath, state.DB.Path)
	}
	if state.DB.Queries == nil {
		t.Fatalf("expected generated queries to be initialized")
	}

	sawInitNotice := false
	for _, notice := range state.Notices {
		if strings.Contains(notice, "Initialized DB at ") {
			sawInitNotice = true
			break
		}
	}
	if !sawInitNotice {
		t.Fatalf("expected startup notice with initialized DB path, got %#v", state.Notices)
	}
}
