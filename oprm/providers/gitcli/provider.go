package gitcli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/oprm/providers/shell"
)

type Provider interface {
	Installed(context.Context) error
	ConfigGet(context.Context, string) (string, error)
	Remotes(context.Context, string) (map[string]string, error)
	CurrentBranch(context.Context, string) (string, error)
	HeadSHA(context.Context, string) (string, error)
	ResolveRef(context.Context, string, string) (string, error)
	ResolveCommit(context.Context, string, string) (string, error)
	IsAncestor(context.Context, string, string, string) (bool, error)
	WorkingTreeClean(context.Context, string) (bool, string, error)
	FetchTags(context.Context, string, string) error
	TagExists(context.Context, string, string) (bool, error)
	CreateAnnotatedTag(context.Context, string, string, string) error
	CreateAnnotatedTagAtRef(context.Context, string, string, string, string) error
	PushTag(context.Context, string, string, string) error
}

type CLI struct {
	Runner shell.Runner
}

func New(runner shell.Runner) *CLI {
	return &CLI{Runner: runner}
}

func (c *CLI) Installed(context.Context) error {
	if _, err := c.Runner.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not on PATH: %w", err)
	}
	return nil
}

func (c *CLI) ConfigGet(ctx context.Context, key string) (string, error) {
	result, err := c.Runner.Run(ctx, "git", "config", "--get", key)
	if err != nil {
		return "", fmt.Errorf("read git config %q: %w", key, err)
	}
	value := strings.TrimSpace(result.Stdout)
	if value == "" {
		return "", fmt.Errorf("git config %q is empty", key)
	}
	return value, nil
}

func (c *CLI) Remotes(ctx context.Context, repoPath string) (map[string]string, error) {
	result, err := c.Runner.Run(ctx, "git", c.repoArgs(repoPath, "remote")...)
	if err != nil {
		return nil, fmt.Errorf("list git remotes in %q: %w", repoPath, err)
	}
	names := strings.Fields(result.Stdout)
	remotes := make(map[string]string, len(names))
	for _, name := range names {
		urlResult, err := c.Runner.Run(ctx, "git", c.repoArgs(repoPath, "remote", "get-url", name)...)
		if err != nil {
			return nil, fmt.Errorf("get git remote url for %q in %q: %w", name, repoPath, err)
		}
		remotes[name] = strings.TrimSpace(urlResult.Stdout)
	}
	return remotes, nil
}

func (c *CLI) CurrentBranch(ctx context.Context, repoPath string) (string, error) {
	result, err := c.Runner.Run(ctx, "git", c.repoArgs(repoPath, "branch", "--show-current")...)
	if err != nil {
		return "", fmt.Errorf("read current git branch in %q: %w", repoPath, err)
	}
	value := strings.TrimSpace(result.Stdout)
	if value == "" {
		return "", fmt.Errorf("current git branch is empty in %q", repoPath)
	}
	return value, nil
}

func (c *CLI) HeadSHA(ctx context.Context, repoPath string) (string, error) {
	result, err := c.Runner.Run(ctx, "git", c.repoArgs(repoPath, "rev-parse", "HEAD")...)
	if err != nil {
		return "", fmt.Errorf("read git HEAD sha in %q: %w", repoPath, err)
	}
	value := strings.TrimSpace(result.Stdout)
	if value == "" {
		return "", fmt.Errorf("git HEAD sha is empty in %q", repoPath)
	}
	return value, nil
}

func (c *CLI) ResolveRef(ctx context.Context, repoPath string, ref string) (string, error) {
	result, err := c.Runner.Run(ctx, "git", c.repoArgs(repoPath, "rev-parse", ref)...)
	if err != nil {
		return "", fmt.Errorf("resolve git ref %q in %q: %w", ref, repoPath, err)
	}
	value := strings.TrimSpace(result.Stdout)
	if value == "" {
		return "", fmt.Errorf("git ref %q resolved to an empty sha in %q", ref, repoPath)
	}
	return value, nil
}

func (c *CLI) ResolveCommit(ctx context.Context, repoPath string, ref string) (string, error) {
	result, err := c.Runner.Run(ctx, "git", c.repoArgs(repoPath, "rev-parse", ref+"^{}")...)
	if err != nil {
		return "", fmt.Errorf("resolve git commit for ref %q in %q: %w", ref, repoPath, err)
	}
	value := strings.TrimSpace(result.Stdout)
	if value == "" {
		return "", fmt.Errorf("git commit for ref %q resolved to an empty sha in %q", ref, repoPath)
	}
	return value, nil
}

func (c *CLI) IsAncestor(ctx context.Context, repoPath string, older string, newer string) (bool, error) {
	_, err := c.Runner.Run(ctx, "git", c.repoArgs(repoPath, "merge-base", "--is-ancestor", older, newer)...)
	if err == nil {
		return true, nil
	}
	var runErr *shell.RunError
	if errors.As(err, &runErr) && runErr.Result.ExitCode == 1 {
		return false, nil
	}
	return false, fmt.Errorf("check whether %s is an ancestor of %s in %q: %w", older, newer, repoPath, err)
}

func (c *CLI) WorkingTreeClean(ctx context.Context, repoPath string) (bool, string, error) {
	result, err := c.Runner.Run(ctx, "git", c.repoArgs(repoPath, "status", "--porcelain")...)
	if err != nil {
		return false, "", fmt.Errorf("read git working tree status in %q: %w", repoPath, err)
	}
	value := strings.TrimSpace(result.Stdout)
	return value == "", value, nil
}

func (c *CLI) FetchTags(ctx context.Context, repoPath string, remote string) error {
	if _, err := c.Runner.Run(ctx, "git", c.repoArgs(repoPath, "fetch", remote, "--tags")...); err != nil {
		return fmt.Errorf("fetch tags from %q in %q: %w", remote, repoPath, err)
	}
	return nil
}

func (c *CLI) TagExists(ctx context.Context, repoPath string, tag string) (bool, error) {
	result, err := c.Runner.Run(ctx, "git", c.repoArgs(repoPath, "tag", "-l", tag)...)
	if err != nil {
		return false, fmt.Errorf("check local git tag %q in %q: %w", tag, repoPath, err)
	}
	return strings.TrimSpace(result.Stdout) != "", nil
}

func (c *CLI) CreateAnnotatedTag(ctx context.Context, repoPath string, tag string, message string) error {
	if _, err := c.Runner.Run(ctx, "git", c.repoArgs(repoPath, "tag", "-a", tag, "-m", message)...); err != nil {
		return fmt.Errorf("create annotated git tag %q in %q: %w", tag, repoPath, err)
	}
	return nil
}

func (c *CLI) CreateAnnotatedTagAtRef(ctx context.Context, repoPath string, tag string, ref string, message string) error {
	if _, err := c.Runner.Run(ctx, "git", c.repoArgs(repoPath, "tag", "-a", tag, ref, "-m", message)...); err != nil {
		return fmt.Errorf("create annotated git tag %q at %q in %q: %w", tag, ref, repoPath, err)
	}
	return nil
}

func (c *CLI) PushTag(ctx context.Context, repoPath string, remote string, tag string) error {
	if _, err := c.Runner.Run(ctx, "git", c.repoArgs(repoPath, "push", remote, tag)...); err != nil {
		return fmt.Errorf("push git tag %q to %q from %q: %w", tag, remote, repoPath, err)
	}
	return nil
}

func (c *CLI) repoArgs(repoPath string, args ...string) []string {
	if strings.TrimSpace(repoPath) == "" {
		return args
	}
	out := make([]string, 0, len(args)+2)
	out = append(out, "-C", repoPath)
	out = append(out, args...)
	return out
}
