// Package logging configures the yaama process-wide slog logger and
// resolves the on-disk log file path. Action paths (tmux bootstrap,
// recovery, cleanup, profile load) write to this logger; the TUI keeps
// stdout/stderr ownership.
package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxLogBytes = 5 * 1024 * 1024
	logDirName  = "yaama"
	logFileName = "yaama.log"
)

// Options configures New.
type Options struct {
	// Path overrides the resolved log file path. When empty DefaultPath
	// is used.
	Path string
	// LevelEnv is the value of YAAMA_LOG_LEVEL (or empty).
	LevelEnv string
	// PID and Version are added as handler-level attrs.
	PID     int
	Version string
}

// Result bundles the configured logger with the writer to close on
// shutdown and the resolved log path (for operator surfacing).
type Result struct {
	Logger *slog.Logger
	Closer io.Closer
	Path   string
}

// Discard returns a logger that writes nowhere. Useful in tests and as
// a safe default for libraries that take an optional logger.
func Discard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// New opens the log file (rotating if it exceeds 5 MiB) and returns a
// configured slog.Logger.
func New(opts Options) (Result, error) {
	path := strings.TrimSpace(opts.Path)
	if path == "" {
		path = DefaultPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Result{}, fmt.Errorf("logging: ensure log dir: %w", err)
	}
	if err := rotateIfLarge(path, maxLogBytes); err != nil {
		return Result{}, fmt.Errorf("logging: rotate: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return Result{}, fmt.Errorf("logging: open log file: %w", err)
	}

	level, unknown := LevelFromEnv(opts.LevelEnv)
	handler := slog.NewTextHandler(f, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)
	if opts.PID != 0 {
		logger = logger.With("pid", opts.PID)
	}
	if strings.TrimSpace(opts.Version) != "" {
		logger = logger.With("version", strings.TrimSpace(opts.Version))
	}
	if unknown != "" {
		logger.Warn("logging.level.unknown",
			"requested", unknown,
			"using", level.String())
	}
	return Result{Logger: logger, Closer: f, Path: path}, nil
}

// DefaultPath resolves the log file path using the documented order:
// YAAMA_LOG_FILE -> XDG_STATE_HOME/yaama/yaama.log -> $HOME/.local/state/yaama/yaama.log.
func DefaultPath() string {
	if override := strings.TrimSpace(os.Getenv("YAAMA_LOG_FILE")); override != "" {
		return override
	}
	if state := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); state != "" {
		return filepath.Join(state, logDirName, logFileName)
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(os.TempDir(), logDirName, logFileName)
	}
	return filepath.Join(home, ".local", "state", logDirName, logFileName)
}

// LevelFromEnv parses YAAMA_LOG_LEVEL. The second return value contains
// the original unrecognized value (or empty when accepted/unset).
func LevelFromEnv(value string) (slog.Level, string) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return slog.LevelInfo, ""
	case "debug":
		return slog.LevelDebug, ""
	case "info":
		return slog.LevelInfo, ""
	case "warn", "warning":
		return slog.LevelWarn, ""
	case "error":
		return slog.LevelError, ""
	default:
		return slog.LevelInfo, value
	}
}

func rotateIfLarge(path string, maxBytes int64) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Size() <= maxBytes {
		return nil
	}
	backup := path + ".1"
	_ = os.Remove(backup)
	return os.Rename(path, backup)
}

// Truncate clips s to n runes, appending an ellipsis when shortened. Use
// before logging captured stderr or other unbounded strings.
func Truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
