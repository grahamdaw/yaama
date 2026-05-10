package gitworktree

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestEnsureRejectsNonGitPaths(t *testing.T) {
	restore := stubGitCommandOutput(t, func(_ context.Context, _ string, args ...string) (string, error) {
		if slices.Equal(args, []string{"rev-parse", "--show-toplevel"}) {
			return "", exitStatusErr(t, 128)
		}
		return "", nil
	})
	defer restore()

	_, err := Ensure(context.Background(), "/tmp", "feat/kai-13", "kai-13-dev")
	if err == nil {
		t.Fatalf("expected non-git path to be rejected")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("expected git repository guidance, got %v", err)
	}
}

func TestEnsureCreatesDeterministicWorktreePathAndRemovesIt(t *testing.T) {
	repoParent := testTempDir(t)
	repoRoot := filepath.Join(repoParent, "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}

	var addedPath string
	var addedBranch string
	restore := stubGitCommandOutput(t, func(_ context.Context, root string, args ...string) (string, error) {
		if slices.Equal(args, []string{"rev-parse", "--show-toplevel"}) {
			switch root {
			case repoRoot, addedPath:
				return repoRoot, nil
			default:
				return "", exitStatusErr(t, 128)
			}
		}
		if slices.Equal(args, []string{"worktree", "list", "--porcelain"}) {
			if addedPath == "" {
				return fmt.Sprintf("worktree %s\nbranch refs/heads/main\n", repoRoot), nil
			}
			return fmt.Sprintf("worktree %s\nbranch refs/heads/main\n\nworktree %s\nbranch refs/heads/%s\n", repoRoot, addedPath, addedBranch), nil
		}
		if len(args) == 4 && args[0] == "show-ref" && args[1] == "--verify" && args[2] == "--quiet" {
			switch args[3] {
			case "refs/remotes/origin/main", "refs/heads/feat/kai-13":
				return "", exitStatusErr(t, 1)
			case "refs/heads/main":
				return "", nil
			default:
				return "", exitStatusErr(t, 1)
			}
		}
		if len(args) >= 6 && args[0] == "worktree" && args[1] == "add" {
			addedPath = args[5]
			addedBranch = args[4]
			if err := os.MkdirAll(addedPath, 0o755); err != nil {
				return "", err
			}
			return "", nil
		}
		if len(args) == 4 && slices.Equal(args[:3], []string{"worktree", "remove", "--force"}) {
			return "", os.RemoveAll(args[3])
		}
		return "", nil
	})
	defer restore()

	worktreePath, err := Ensure(context.Background(), repoRoot, "feat/kai-13", "kai-13-dev")
	if err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}
	wantPath := filepath.Join(repoParent, worktreesDirName, "kai-13-dev")
	if worktreePath != wantPath {
		t.Fatalf("expected worktree path %q, got %q", wantPath, worktreePath)
	}
	if info, err := os.Stat(worktreePath); err != nil || !info.IsDir() {
		t.Fatalf("expected created worktree dir, stat err=%v", err)
	}
	if addedBranch != "feat/kai-13" {
		t.Fatalf("expected branch feat/kai-13, got %q", addedBranch)
	}

	secondPath, err := Ensure(context.Background(), repoRoot, "feat/kai-13", "kai-13-dev")
	if err != nil {
		t.Fatalf("second Ensure returned error: %v", err)
	}
	if secondPath != worktreePath {
		t.Fatalf("expected second ensure to return same path, got %q", secondPath)
	}

	if err := Remove(context.Background(), worktreePath); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Fatalf("expected worktree to be removed, stat err=%v", err)
	}
}

func TestValidateBranchRejectsUnsafeValues(t *testing.T) {
	cases := []string{"", "bad branch", "feat..oops", "feat@{oops", "feature.lock", "/start", "end/"}
	for _, value := range cases {
		if err := ValidateBranch(value); err == nil {
			t.Fatalf("expected branch %q to be rejected", value)
		}
	}
	if err := ValidateBranch("feat/kai-13"); err != nil {
		t.Fatalf("expected valid branch to pass, got %v", err)
	}
}

func stubGitCommandOutput(t *testing.T, fn func(context.Context, string, ...string) (string, error)) func() {
	t.Helper()
	previous := gitCommandOutput
	gitCommandOutput = fn
	return func() {
		gitCommandOutput = previous
	}
}

func exitStatusErr(t *testing.T, code int) error {
	t.Helper()
	cmd := exec.Command("sh", "-c", fmt.Sprintf("exit %d", code))
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected exit status %d", code)
	}
	return err
}

func testTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", "tmp-gitworktree-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("failed to resolve temp dir: %v", err)
	}
	return abs
}
