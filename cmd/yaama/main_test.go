package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapBoardInitializesFreshDBAndEmitsNotice(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "fresh", "yaama.db")

	params, cleanup, err := bootstrapBoard(dbPath)
	if err != nil {
		t.Fatalf("bootstrapBoard returned error: %v", err)
	}
	t.Cleanup(cleanup)

	if params.Queries == nil {
		t.Fatalf("expected non-nil queries after bootstrap")
	}

	sawInitNotice := false
	for _, notice := range params.Notices {
		if strings.Contains(notice, "Initialized DB at "+dbPath) {
			sawInitNotice = true
			break
		}
	}
	if !sawInitNotice {
		t.Fatalf("expected startup notice with initialized DB path, got %#v", params.Notices)
	}
}
