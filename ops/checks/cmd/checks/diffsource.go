package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// diffSource enumerates where `checks run` pulls its diff from when
// the caller doesn't pipe one in explicitly.
type diffSource int

const (
	sourceAuto        diffSource = iota // branch vs base (default)
	sourceStaged                        // git diff --cached
	sourceUncommitted                   // git diff (unstaged) + --cached
	sourceCommit                        // a specific commit
	sourceStdin                         // explicit stdin/--diff file
)

// computeDiff runs git to produce the unified diff that matches the
// caller's requested source. Returns the raw diff text. `root` is the
// repo root; `base` is the base-branch ref (only used by sourceAuto).
func computeDiff(src diffSource, root, base, commitSHA string, stdin io.Reader) (string, error) {
	if src == sourceStdin {
		b, err := io.ReadAll(stdin)
		return string(b), err
	}
	var args []string
	switch src {
	case sourceAuto:
		// Branch diff vs the merge-base of the base branch. Same thing
		// `git log base..HEAD` covers, but in patch form so we get
		// hunks for line-level freshness analysis.
		args = []string{"diff", "--merge-base", base, "HEAD"}
	case sourceStaged:
		args = []string{"diff", "--cached"}
	case sourceUncommitted:
		args = []string{"diff", "HEAD"}
	case sourceCommit:
		if commitSHA == "" {
			return "", fmt.Errorf("--commit requires a SHA")
		}
		args = []string{"diff", commitSHA + "^", commitSHA}
	default:
		return "", fmt.Errorf("unknown diff source")
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// stdinIsPipe reports whether stdin is a pipe/redirect (vs a TTY or
// closed fd). Used by cmdRun to auto-detect `git diff | checks run`
// without requiring `--diff -`.
func stdinIsPipe() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Stdin is piped when it's a FIFO or regular file (redirect); it's
	// a char device when connected to a terminal.
	return (fi.Mode() & os.ModeCharDevice) == 0
}

// defaultBaseBranch returns the repo's conventional base branch.
// `develop` on the optimism monorepo, `main` everywhere else. Reads
// origin/HEAD when available; falls back to origin/develop.
func defaultBaseBranch(root string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "origin/HEAD")
	cmd.Dir = root
	if out, err := cmd.Output(); err == nil {
		ref := strings.TrimSpace(string(out))
		if ref != "" && ref != "origin/HEAD" {
			return ref
		}
	}
	// Check if origin/develop exists
	cmd = exec.Command("git", "rev-parse", "--verify", "origin/develop")
	cmd.Dir = root
	if err := cmd.Run(); err == nil {
		return "origin/develop"
	}
	return "origin/main"
}
