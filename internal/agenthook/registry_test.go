package agenthook

import (
	"errors"
	"reflect"
	"sort"
	"testing"
)

func TestIDsAreSortedAndIncludeAllRegistered(t *testing.T) {
	t.Parallel()

	got := IDs()
	if len(got) == 0 {
		t.Fatalf("expected at least one registered harness")
	}
	sorted := append([]string(nil), got...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(got, sorted) {
		t.Fatalf("IDs() should be sorted; got %v", got)
	}

	want := []string{"claude-code", "codex", "copilot", "kiro", "kiro-cli", "vscode-copilot"}
	for _, id := range want {
		if _, ok := Get(id); !ok {
			t.Fatalf("expected harness %q to be registered; have %v", id, got)
		}
	}
}

func TestGetReturnsFalseForUnknownID(t *testing.T) {
	t.Parallel()

	if _, ok := Get("nope"); ok {
		t.Fatalf("expected Get(nope) to return ok=false")
	}
}

func TestLaunchOnlyHarnessesReturnSentinelHook(t *testing.T) {
	t.Parallel()

	launchOnly := []string{"codex", "kiro-cli", "kiro", "copilot", "vscode-copilot"}
	for _, id := range launchOnly {
		h, ok := Get(id)
		if !ok {
			t.Fatalf("harness %q not registered", id)
		}
		if _, err := h.ParseHook([]byte(`{}`)); !errors.Is(err, ErrHarnessHasNoHook) {
			t.Fatalf("expected %q.ParseHook to return ErrHarnessHasNoHook, got %v", id, err)
		}
		if def := h.Defaults(); def.Command == "" {
			t.Fatalf("harness %q is missing a default command", id)
		}
	}
}
