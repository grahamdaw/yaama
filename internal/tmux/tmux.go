package tmux

import "os/exec"

func IsAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}
