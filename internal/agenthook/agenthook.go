// Package agenthook describes the agent harnesses yaama knows how to launch
// and (where supported) parse hook payloads from. Each harness is a single
// file under this package that registers itself in init().
package agenthook

import (
	"errors"
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

// StatusUpdate is the name used in the harness registry surface for what
// ParseHook returns. It is an alias of Event so existing call sites keep
// working.
type StatusUpdate = Event

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

// AgentDefaults describes the launch defaults a harness contributes when
// the operator omits agent.command / agent.args / agent.env in the profile.
// Operator-set values always win.
type AgentDefaults struct {
	Command string
	Args    []string
	Env     map[string]string
}

// ErrHarnessHasNoHook is returned by harnesses that have launch defaults
// but no hook-parser implementation yet. Callers (yaama hook) translate
// this into a friendly operator-facing message.
var ErrHarnessHasNoHook = errors.New("harness has no hook integration yet")

// Harness is the contract a registered agent harness implements.
// Implementations are expected to be stateless and safe to share.
type Harness interface {
	// ID returns the harness identifier (lowercase, hyphenated). Used as the
	// agent.harness value in profiles and as the yaama hook <id> argument.
	ID() string

	// Defaults returns launch defaults that fill agent.command / args / env
	// when the operator leaves them empty in the profile.
	Defaults() AgentDefaults

	// ParseHook decodes a hook payload into a StatusUpdate. Harnesses that
	// have no hook integration yet return ErrHarnessHasNoHook.
	ParseHook(payload []byte) (StatusUpdate, error)
}

var registry = map[string]Harness{}

// Register makes a harness available by id. Duplicate ids panic at init
// time to surface wiring mistakes early.
func Register(h Harness) {
	id := strings.ToLower(strings.TrimSpace(h.ID()))
	if id == "" {
		panic("agenthook: harness id must not be empty")
	}
	if _, exists := registry[id]; exists {
		panic(fmt.Sprintf("agenthook: harness %q already registered", id))
	}
	registry[id] = h
}

// Get returns the harness registered for id (case-insensitive) and a bool
// indicating whether it was found.
func Get(id string) (Harness, bool) {
	h, ok := registry[strings.ToLower(strings.TrimSpace(id))]
	return h, ok
}

// IDs returns the sorted list of registered harness ids; useful for help
// text and error hints.
func IDs() []string {
	out := make([]string, 0, len(registry))
	for id := range registry {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
