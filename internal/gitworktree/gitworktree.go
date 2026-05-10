package gitworktree

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const worktreesDirName = ".yaama-worktrees"

var gitCommandOutput = gitOutput

func ValidateBranch(branch string) error {
	name := strings.TrimSpace(branch)
	switch {
	case name == "":
		return errors.New("branch is required")
	case strings.HasPrefix(name, "/"), strings.HasSuffix(name, "/"):
		return errors.New("branch cannot start or end with /")
	case strings.HasPrefix(name, "."), strings.HasSuffix(name, "."):
		return errors.New("branch cannot start or end with .")
	case strings.HasSuffix(name, ".lock"):
		return errors.New("branch cannot end with .lock")
	case strings.Contains(name, ".."):
		return errors.New("branch cannot contain ..")
	case strings.Contains(name, "@{"):
		return errors.New("branch cannot contain @{")
	case strings.ContainsAny(name, " ~^:?*[\\"):
		return errors.New("branch contains invalid characters")
	case name == "@":
		return errors.New("branch cannot be @")
	default:
		return nil
	}
}

func Ensure(ctx context.Context, repoPath, branch, slug string) (string, error) {
	if err := ValidateBranch(branch); err != nil {
		return "", err
	}
	repoRoot, err := repoRootFor(ctx, repoPath)
	if err != nil {
		return "", err
	}
	identifier := strings.TrimSpace(slug)
	if identifier == "" {
		identifier = "session"
	}
	worktreePath := filepath.Join(filepath.Dir(repoRoot), worktreesDirName, identifier)
	worktreePath = filepath.Clean(worktreePath)

	if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
		return "", fmt.Errorf("create worktree root: %w", err)
	}

	existing, err := listWorktrees(ctx, repoRoot)
	if err != nil {
		return "", err
	}
	branchRef := "refs/heads/" + strings.TrimSpace(branch)
	for path, ref := range existing {
		if ref == branchRef && path != worktreePath {
			return "", fmt.Errorf("branch %q already checked out at %q", branch, path)
		}
		if path == worktreePath && ref != "" && ref != branchRef {
			return "", fmt.Errorf("worktree path %q already exists for %q", worktreePath, ref)
		}
	}
	if ref, ok := existing[worktreePath]; ok && ref == branchRef {
		return worktreePath, nil
	}

	baseRef, err := resolveBaseRef(ctx, repoRoot)
	if err != nil {
		return "", err
	}
	if hasLocal, err := hasRef(ctx, repoRoot, "refs/heads/"+branch); err != nil {
		return "", err
	} else if hasLocal {
		if _, err := gitCommandOutput(ctx, repoRoot, "worktree", "add", "--force", worktreePath, branch); err != nil {
			return "", err
		}
	} else {
		if _, err := gitCommandOutput(ctx, repoRoot, "worktree", "add", "--force", "-b", branch, worktreePath, baseRef); err != nil {
			return "", err
		}
	}

	return worktreePath, nil
}

func Remove(ctx context.Context, worktreePath string) error {
	target := strings.TrimSpace(worktreePath)
	if target == "" {
		return nil
	}
	if _, err := os.Stat(target); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat worktree: %w", err)
	}

	repoRoot, err := repoRootFor(ctx, target)
	if err != nil {
		return err
	}
	if _, err := gitCommandOutput(ctx, repoRoot, "worktree", "remove", "--force", target); err != nil {
		return err
	}
	return nil
}

func resolveBaseRef(ctx context.Context, repoRoot string) (string, error) {
	if hasOriginMain, err := hasRef(ctx, repoRoot, "refs/remotes/origin/main"); err != nil {
		return "", err
	} else if hasOriginMain {
		if _, err := gitCommandOutput(ctx, repoRoot, "update-ref", "refs/heads/main", "refs/remotes/origin/main"); err != nil {
			return "", err
		}
		return "main", nil
	}
	if hasLocalMain, err := hasRef(ctx, repoRoot, "refs/heads/main"); err != nil {
		return "", err
	} else if hasLocalMain {
		return "main", nil
	}
	return "HEAD", nil
}

func repoRootFor(ctx context.Context, path string) (string, error) {
	target := strings.TrimSpace(path)
	if target == "" {
		return "", errors.New("repository path is required")
	}
	out, err := gitCommandOutput(ctx, target, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("resolved path %q is not a git repository", target)
	}
	root := strings.TrimSpace(out)
	if root == "" {
		return "", fmt.Errorf("resolved path %q is not a git repository", target)
	}
	return filepath.Clean(root), nil
}

func listWorktrees(ctx context.Context, repoRoot string) (map[string]string, error) {
	out, err := gitCommandOutput(ctx, repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	result := map[string]string{}
	var currentPath string
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			currentPath = filepath.Clean(strings.TrimSpace(strings.TrimPrefix(line, "worktree ")))
			if _, ok := result[currentPath]; !ok {
				result[currentPath] = ""
			}
			continue
		}
		if strings.HasPrefix(line, "branch ") && currentPath != "" {
			result[currentPath] = strings.TrimSpace(strings.TrimPrefix(line, "branch "))
		}
	}
	return result, nil
}

func hasRef(ctx context.Context, repoRoot string, ref string) (bool, error) {
	_, err := gitCommandOutput(ctx, repoRoot, "show-ref", "--verify", "--quiet", ref)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func gitOutput(ctx context.Context, repoRoot string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoRoot}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
