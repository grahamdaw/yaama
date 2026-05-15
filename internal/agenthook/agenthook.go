// Package agenthook normalizes hook payloads emitted by different AI agent
// runtimes (Claude Code today, others later) into a small set of fields that
// map onto an agent's row in the yaama database.
package agenthook

import (
	"fmt"
	"sort"
	"strings"
)

// Event is the normalized result of parsing a raw hook payload. Fields that
// the parser does not populate are left as zero-value optionals so the caller
// can preserve existing column values via COALESCE-style updates.
type Event struct {
	// EventName is the agent-native hook event identifier (e.g. "PreToolUse").
	// Mainly used for logging and diagnostics.
	EventName string

	// Status, when non-empty, is one of the accepted agent status values
	// ("idle", "running", "blocked", "review", "done"). An empty value means
	// "leave the existing status unchanged".
	Status Optional

	// LastActivity is a short human-readable summary of what the agent is
	// doing as of this hook firing.
	LastActivity Optional

	// LastError is set when the hook represents a failure surface (e.g. a
	// tool error). Empty value means "leave existing error untouched".
	LastError Optional
}

// Optional carries a value plus a flag distinguishing "unset" from
// "explicitly empty". This mirrors the optionalString pattern used by the
// status command.
type Optional struct {
	Value string
	Set   bool
}

// SetValue returns an Optional with Set=true.
func SetValue(v string) Optional {
	return Optional{Value: v, Set: true}
}

// Parser turns raw bytes from stdin into a normalized Event. Implementations
// are expected to be stateless and safe to share.
type Parser interface {
	// Name returns the agent identifier the parser handles (lowercase,
	// hyphenated). Used as the CLI subcommand argument.
	Name() string

	// Parse decodes raw and returns the derived event. An error indicates
	// the payload is malformed for this parser; callers should not write to
	// the database in that case.
	Parse(raw []byte) (Event, error)
}

var registry = map[string]Parser{}

// Register makes a parser available by name. Duplicate names panic at init
// time to surface wiring mistakes early.
func Register(p Parser) {
	name := strings.ToLower(strings.TrimSpace(p.Name()))
	if name == "" {
		panic("agenthook: parser name must not be empty")
	}
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("agenthook: parser %q already registered", name))
	}
	registry[name] = p
}

// Lookup returns the parser registered for name (case-insensitive) and a
// boolean indicating whether it was found.
func Lookup(name string) (Parser, bool) {
	p, ok := registry[strings.ToLower(strings.TrimSpace(name))]
	return p, ok
}

// Names returns the sorted list of registered parser names; useful for help
// text and error hints.
func Names() []string {
	out := make([]string, 0, len(registry))
	for name := range registry {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
