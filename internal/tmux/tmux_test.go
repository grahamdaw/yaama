package tmux

import "testing"

func TestIsNoTmuxServerOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "no server running output",
			output: "no server running on /private/tmp/tmux-501/default",
			want:   true,
		},
		{
			name:   "failed to connect output",
			output: "failed to connect to server",
			want:   true,
		},
		{
			name:   "missing socket output",
			output: "error connecting to /private/tmp/tmux-501/default (No such file or directory)",
			want:   true,
		},
		{
			name:   "other tmux error output",
			output: "unknown option -- Z",
			want:   false,
		},
		{
			name:   "empty output",
			output: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNoTmuxServerOutput(tt.output)
			if got != tt.want {
				t.Fatalf("isNoTmuxServerOutput(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

func TestIsMissingTmuxSessionOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "cannot find session",
			output: "can't find session: demo",
			want:   true,
		},
		{
			name:   "no such session",
			output: "no such session",
			want:   true,
		},
		{
			name:   "other output",
			output: "unknown option -- x",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMissingTmuxSessionOutput(tt.output)
			if got != tt.want {
				t.Fatalf("isMissingTmuxSessionOutput(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

func TestCreateDetachedSessionArgsDisablesDestroyUnattached(t *testing.T) {
	got := createDetachedSessionArgs("demo-session", "/tmp/worktree")
	want := []string{
		"new-session", "-d", "-s", "demo-session", "-c", "/tmp/worktree",
		";",
		"set-option", "-t", "demo-session", "destroy-unattached", "off",
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected args length: got %d, want %d", len(got), len(want))
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("arg[%d] = %q, want %q (full got: %#v)", idx, got[idx], want[idx], got)
		}
	}
}
