package ghcli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/ethereum-optimism/optimism/oprm/providers/shell"
)

type User struct {
	Login string `json:"login"`
}

type Release struct {
	ID         int    `json:"id"`
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	URL        string `json:"html_url"`
	Draft      bool   `json:"draft"`
	PreRelease bool   `json:"prerelease"`
}

type ChangedFile struct {
	Filename string `json:"filename"`
}

type CompareCommitDetails struct {
	Message string `json:"message"`
}

type CompareCommit struct {
	SHA    string               `json:"sha"`
	Commit CompareCommitDetails `json:"commit"`
}

type CompareResult struct {
	HTMLURL string          `json:"html_url"`
	Files   []ChangedFile   `json:"files"`
	Commits []CompareCommit `json:"commits"`
}

type RepoPermissions struct {
	Admin    bool `json:"admin"`
	Maintain bool `json:"maintain"`
	Push     bool `json:"push"`
	Triage   bool `json:"triage"`
	Pull     bool `json:"pull"`
}

type Repo struct {
	FullName    string          `json:"full_name"`
	Permissions RepoPermissions `json:"permissions"`
}

type Provider interface {
	Installed(context.Context) error
	Authenticated(context.Context) error
	CurrentUser(context.Context) (*User, error)
	ListReleases(context.Context, string, string) ([]Release, error)
	CompareCommits(context.Context, string, string, string, string) (*CompareResult, error)
	GetRepo(context.Context, string, string) (*Repo, error)
	TagExists(context.Context, string, string, string) (bool, error)
	TagCommit(context.Context, string, string, string) (string, error)
	CreateDraftRelease(context.Context, string, string, string, string, string, string) (*Release, error)
	UpdateDraftRelease(context.Context, string, string, int, string, string) (*Release, error)
	PublishRelease(context.Context, string, string, int, string) (*Release, error)
}

type CLI struct {
	Runner shell.Runner
}

func New(runner shell.Runner) *CLI {
	return &CLI{Runner: runner}
}

func (c *CLI) Installed(context.Context) error {
	if _, err := c.Runner.LookPath("gh"); err != nil {
		return fmt.Errorf("gh is not installed or not on PATH: %w", err)
	}
	return nil
}

func (c *CLI) Authenticated(ctx context.Context) error {
	if _, err := c.Runner.Run(ctx, "gh", "auth", "status"); err != nil {
		return fmt.Errorf("gh is not authenticated: %w", err)
	}
	return nil
}

func (c *CLI) CurrentUser(ctx context.Context) (*User, error) {
	result, err := c.Runner.Run(ctx, "gh", "api", "user")
	if err != nil {
		return nil, fmt.Errorf("query current GitHub user: %w", err)
	}
	var user User
	if err := json.Unmarshal([]byte(result.Stdout), &user); err != nil {
		return nil, fmt.Errorf("decode current GitHub user: %w", err)
	}
	user.Login = strings.TrimSpace(user.Login)
	if user.Login == "" {
		return nil, fmt.Errorf("current GitHub user response did not include a login")
	}
	return &user, nil
}

func (c *CLI) ListReleases(ctx context.Context, owner string, repo string) ([]Release, error) {
	result, err := c.Runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/%s/releases?per_page=100", owner, repo))
	if err != nil {
		return nil, fmt.Errorf("list releases for %s/%s: %w", owner, repo, err)
	}
	var releases []Release
	if err := json.Unmarshal([]byte(result.Stdout), &releases); err != nil {
		return nil, fmt.Errorf("decode releases for %s/%s: %w", owner, repo, err)
	}
	return releases, nil
}

func (c *CLI) CompareCommits(ctx context.Context, owner string, repo string, base string, head string) (*CompareResult, error) {
	comparison := url.PathEscape(base + "..." + head)
	result, err := c.Runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/%s/compare/%s", owner, repo, comparison))
	if err != nil {
		return nil, fmt.Errorf("compare %s/%s %s...%s: %w", owner, repo, base, head, err)
	}
	var compare CompareResult
	if err := json.Unmarshal([]byte(result.Stdout), &compare); err != nil {
		return nil, fmt.Errorf("decode compare result for %s/%s %s...%s: %w", owner, repo, base, head, err)
	}
	return &compare, nil
}

func (c *CLI) GetRepo(ctx context.Context, owner string, repo string) (*Repo, error) {
	result, err := c.Runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/%s", owner, repo))
	if err != nil {
		return nil, fmt.Errorf("get repo %s/%s: %w", owner, repo, err)
	}
	var out Repo
	if err := json.Unmarshal([]byte(result.Stdout), &out); err != nil {
		return nil, fmt.Errorf("decode repo %s/%s: %w", owner, repo, err)
	}
	return &out, nil
}

func (c *CLI) TagExists(ctx context.Context, owner string, repo string, tag string) (bool, error) {
	result, err := c.Runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/%s/git/matching-refs/tags/%s", owner, repo, url.PathEscape(tag)))
	if err != nil {
		return false, fmt.Errorf("query tag %s for %s/%s: %w", tag, owner, repo, err)
	}
	var refs []struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &refs); err != nil {
		return false, fmt.Errorf("decode matching refs for tag %s in %s/%s: %w", tag, owner, repo, err)
	}
	expected := "refs/tags/" + tag
	for _, ref := range refs {
		if ref.Ref == expected {
			return true, nil
		}
	}
	return false, nil
}

func (c *CLI) TagCommit(ctx context.Context, owner string, repo string, tag string) (string, error) {
	result, err := c.Runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/%s/git/ref/tags/%s", owner, repo, url.PathEscape(tag)))
	if err != nil {
		return "", fmt.Errorf("query tag ref %s for %s/%s: %w", tag, owner, repo, err)
	}
	var ref struct {
		Object struct {
			Type string `json:"type"`
			SHA  string `json:"sha"`
		} `json:"object"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &ref); err != nil {
		return "", fmt.Errorf("decode tag ref %s for %s/%s: %w", tag, owner, repo, err)
	}
	typeName := strings.TrimSpace(ref.Object.Type)
	sha := strings.TrimSpace(ref.Object.SHA)
	if sha == "" {
		return "", fmt.Errorf("tag ref %s for %s/%s did not include an object sha", tag, owner, repo)
	}
	if typeName == "commit" {
		return sha, nil
	}
	if typeName != "tag" {
		return "", fmt.Errorf("tag ref %s for %s/%s has unsupported object type %q", tag, owner, repo, typeName)
	}
	result, err = c.Runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/%s/git/tags/%s", owner, repo, sha))
	if err != nil {
		return "", fmt.Errorf("query tag object %s for %s/%s: %w", tag, owner, repo, err)
	}
	var tagObject struct {
		Object struct {
			Type string `json:"type"`
			SHA  string `json:"sha"`
		} `json:"object"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &tagObject); err != nil {
		return "", fmt.Errorf("decode tag object %s for %s/%s: %w", tag, owner, repo, err)
	}
	if strings.TrimSpace(tagObject.Object.Type) != "commit" || strings.TrimSpace(tagObject.Object.SHA) == "" {
		return "", fmt.Errorf("tag object %s for %s/%s does not point to a commit", tag, owner, repo)
	}
	return strings.TrimSpace(tagObject.Object.SHA), nil
}

func (c *CLI) CreateDraftRelease(ctx context.Context, owner string, repo string, tag string, target string, title string, body string) (*Release, error) {
	result, err := c.Runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/%s/releases", owner, repo), "--method", "POST", "-f", "tag_name="+tag, "-f", "target_commitish="+target, "-f", "name="+title, "-F", "draft=true", "-F", "prerelease=false", "-f", "body="+body)
	if err != nil {
		return nil, fmt.Errorf("create draft release %s for %s/%s: %w", tag, owner, repo, err)
	}
	var out Release
	if err := json.Unmarshal([]byte(result.Stdout), &out); err != nil {
		return nil, fmt.Errorf("decode created draft release %s for %s/%s: %w", tag, owner, repo, err)
	}
	return &out, nil
}

func (c *CLI) UpdateDraftRelease(ctx context.Context, owner string, repo string, releaseID int, title string, body string) (*Release, error) {
	result, err := c.Runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/%s/releases/%d", owner, repo, releaseID), "--method", "PATCH", "-f", "name="+title, "-F", "draft=true", "-F", "prerelease=false", "-f", "body="+body)
	if err != nil {
		return nil, fmt.Errorf("update draft release %d for %s/%s: %w", releaseID, owner, repo, err)
	}
	var out Release
	if err := json.Unmarshal([]byte(result.Stdout), &out); err != nil {
		return nil, fmt.Errorf("decode updated draft release %d for %s/%s: %w", releaseID, owner, repo, err)
	}
	return &out, nil
}

func (c *CLI) PublishRelease(ctx context.Context, owner string, repo string, releaseID int, tag string) (*Release, error) {
	result, err := c.Runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/%s/releases/%d", owner, repo, releaseID), "--method", "PATCH", "-f", "tag_name="+tag, "-F", "draft=false", "-F", "prerelease=false")
	if err != nil {
		return nil, fmt.Errorf("publish release %d for %s/%s as %s: %w", releaseID, owner, repo, tag, err)
	}
	var out Release
	if err := json.Unmarshal([]byte(result.Stdout), &out); err != nil {
		return nil, fmt.Errorf("decode published release %d for %s/%s: %w", releaseID, owner, repo, err)
	}
	return &out, nil
}
