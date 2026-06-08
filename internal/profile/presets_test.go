package profile

import (
	"reflect"
	"sort"
	"testing"
)

func TestPresetIDsAreSortedAndComplete(t *testing.T) {
	t.Parallel()
	want := []string{"agent+git+tests", "agent+logs", "agent+tests", "solo"}
	got := PresetIDs()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PresetIDs = %v, want %v", got, want)
	}
	sorted := append([]string(nil), got...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(got, sorted) {
		t.Fatalf("PresetIDs() should be sorted; got %v", got)
	}
}

func TestExpandPresetSolo(t *testing.T) {
	t.Parallel()
	w, ok := expandPreset("solo")
	if !ok || len(w) != 0 {
		t.Fatalf("expected solo to expand to zero windows, got ok=%v windows=%v", ok, w)
	}
}

func TestExpandPresetAgentGitTests(t *testing.T) {
	t.Parallel()
	w, ok := expandPreset("agent+git+tests")
	if !ok {
		t.Fatalf("expected agent+git+tests to expand")
	}
	if len(w) != 1 || w[0].Name != "ops" || w[0].Layout != "even-vertical" {
		t.Fatalf("unexpected expansion: %+v", w)
	}
	if len(w[0].Panes) != 2 || w[0].Panes[1].Split != "vertical" || w[0].Panes[1].Size != "30%" {
		t.Fatalf("unexpected panes: %+v", w[0].Panes)
	}
}

func TestExpandPresetUnknownReturnsFalse(t *testing.T) {
	t.Parallel()
	if _, ok := expandPreset("nope"); ok {
		t.Fatalf("expected unknown preset to return false")
	}
}
