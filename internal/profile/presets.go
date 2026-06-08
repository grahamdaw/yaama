package profile

import (
	"sort"
)

// preset names operators can set via [tmux] preset = "<name>". Adding a
// preset is one entry: a function that returns the windows it expands to.
var tmuxPresets = map[string]func() []TmuxWindow{
	"solo": func() []TmuxWindow { return nil },
	"agent+logs": func() []TmuxWindow {
		return []TmuxWindow{{
			Name: "logs",
			Panes: []TmuxPane{{
				Cwd: ".",
				Run: `tail -F "${YAAMA_LOG_PATH:-$HOME/.local/state/yaama/yaama.log}"`,
			}},
		}}
	},
	"agent+tests": func() []TmuxWindow {
		return []TmuxWindow{{
			Name:  "tests",
			Panes: []TmuxPane{{Cwd: "."}},
		}}
	},
	"agent+git+tests": func() []TmuxWindow {
		return []TmuxWindow{{
			Name:   "ops",
			Layout: "even-vertical",
			Panes: []TmuxPane{
				{Cwd: ".", Run: "git status -sb"},
				{Split: "vertical", Size: "30%", Cwd: ".", Run: "make test"},
			},
		}}
	},
}

// PresetIDs returns the sorted catalog of preset names.
func PresetIDs() []string {
	out := make([]string, 0, len(tmuxPresets))
	for id := range tmuxPresets {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// expandPreset returns the windows produced by the named preset and a bool
// indicating whether it exists.
func expandPreset(name string) ([]TmuxWindow, bool) {
	fn, ok := tmuxPresets[name]
	if !ok {
		return nil, false
	}
	return fn(), true
}
