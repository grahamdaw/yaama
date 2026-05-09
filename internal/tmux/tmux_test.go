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
