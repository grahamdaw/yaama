package logging

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLevelFromEnv(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		want    slog.Level
		unknown string
	}{
		{"", slog.LevelInfo, ""},
		{"debug", slog.LevelDebug, ""},
		{"INFO", slog.LevelInfo, ""},
		{"warn", slog.LevelWarn, ""},
		{"warning", slog.LevelWarn, ""},
		{"error", slog.LevelError, ""},
		{"trace", slog.LevelInfo, "trace"},
	}
	for _, tc := range cases {
		got, unknown := LevelFromEnv(tc.in)
		if got != tc.want || unknown != tc.unknown {
			t.Errorf("LevelFromEnv(%q) = (%v,%q); want (%v,%q)", tc.in, got, unknown, tc.want, tc.unknown)
		}
	}
}

func TestDefaultPathHonorsOverride(t *testing.T) {
	dir := t.TempDir()
	override := filepath.Join(dir, "custom.log")
	t.Setenv("YAAMA_LOG_FILE", override)
	if got := DefaultPath(); got != override {
		t.Fatalf("expected override %q, got %q", override, got)
	}
}

func TestDefaultPathUsesXDGStateHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YAAMA_LOG_FILE", "")
	t.Setenv("XDG_STATE_HOME", dir)
	want := filepath.Join(dir, "yaama", "yaama.log")
	if got := DefaultPath(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestDefaultPathFallsBackToHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("YAAMA_LOG_FILE", "")
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", dir)
	want := filepath.Join(dir, ".local", "state", "yaama", "yaama.log")
	if got := DefaultPath(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestRotateIfLargeMovesOversizedFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "yaama.log")
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), 6*1024*1024), 0o644); err != nil {
		t.Fatalf("seed log: %v", err)
	}
	if err := rotateIfLarge(path, maxLogBytes); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected original file removed, stat err=%v", err)
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("expected backup file, stat err=%v", err)
	}
}

func TestRotateIfLargeLeavesSmallFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "yaama.log")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("seed log: %v", err)
	}
	if err := rotateIfLarge(path, maxLogBytes); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected original kept, stat err=%v", err)
	}
}

func TestNewEmitsWarnForUnknownLevel(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "y.log")
	res, err := New(Options{Path: path, LevelEnv: "loud"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = res.Closer.Close() })

	res.Logger.Info("after-warn")
	_ = res.Closer.Close()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "logging.level.unknown") {
		t.Fatalf("expected unknown-level warn line, got %q", text)
	}
	if !strings.Contains(text, "requested=loud") {
		t.Fatalf("expected requested=loud attr, got %q", text)
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()
	if got := Truncate("hello", 5); got != "hello" {
		t.Errorf("no-truncate got %q", got)
	}
	if got := Truncate("hello world", 5); got != "hello…" {
		t.Errorf("truncate got %q", got)
	}
	if got := Truncate("hi", 0); got != "" {
		t.Errorf("zero-n got %q", got)
	}
}
