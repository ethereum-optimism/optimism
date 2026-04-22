package cihistory

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
)

// CircleCIConfig holds everything CircleCIFetcher needs.
//
// Token is optional — public projects (ethereum-optimism/optimism)
// don't require auth, and passing an empty token omits the Circle-Token
// header entirely, matching the CI operations rule in CLAUDE.md.
//
// Branch semantics:
//   - A non-empty Branch constrains the fetch to that specific branch
//     (the typical "develop" case for post-merge outcomes).
//   - An empty Branch fetches pipelines from all branches. Combine
//     with ExcludeBranchPrefixes to filter out the branches whose
//     outcomes you already have (e.g. develop) — the remainder are
//     pre-merge pipelines where failures actually happen, which is
//     the data the replay harness needs for recall measurement.
//
// FilesFor is the callback that returns the files changed in a given
// revision. Default is ResolveFilesViaGit, which shells to `git diff-
// tree` in RepoRoot; tests can substitute a deterministic function.
type CircleCIConfig struct {
	Org                   string        // e.g. "ethereum-optimism"
	Repo                  string        // e.g. "optimism"
	Branch                string        // non-empty = single branch; empty = all branches
	ExcludeBranchPrefixes []string      // optional; skip pipelines whose branch starts with any of these
	Token                 string        // $CITOKEN; empty for public projects
	BaseURL               string        // override for tests; default "https://circleci.com/api/v2"
	HTTPClient            *http.Client  // override for tests; default http.DefaultClient
	MaxPages              int           // safety cap on pipeline pagination; 0 = 10
	RepoRoot              string        // local checkout used by FilesFor's default
	FilesFor              func(revision string) ([]string, error)

	// PRFor attributes a PR number to a pipeline's commit. The
	// default parses `(#NNNN)` out of the commit subject via
	// ResolvePRViaGit — correct for squash-merges on develop and
	// noop (returns 0) for pre-merge PR-branch commits whose
	// subjects don't yet embed a PR reference. Replay uses the
	// result only for display/filtering; a 0 means "unknown" and
	// the event is still replayed.
	PRFor                 func(revision string) int
}

// CircleCIFetcher implements cihistory.Fetcher against CircleCI's v2 API.
type CircleCIFetcher struct {
	cfg        CircleCIConfig
	jobToCheck map[string]string // ci_job_name → catalog CheckType.ID
}

// NewCircleCIFetcher creates a fetcher. jobToCheck maps CircleCI job
// names to catalog check IDs; jobs outside the map are ignored.
// JobMapFromCatalog builds this from a parsed catalog.
func NewCircleCIFetcher(cfg CircleCIConfig, jobToCheck map[string]string) *CircleCIFetcher {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://circleci.com/api/v2"
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	if cfg.MaxPages <= 0 {
		cfg.MaxPages = 10
	}
	if cfg.FilesFor == nil {
		cfg.FilesFor = func(rev string) ([]string, error) {
			return ResolveFilesViaGit(cfg.RepoRoot, rev)
		}
	}
	if cfg.PRFor == nil {
		cfg.PRFor = func(rev string) int {
			return ResolvePRViaGit(cfg.RepoRoot, rev)
		}
	}
	return &CircleCIFetcher{cfg: cfg, jobToCheck: jobToCheck}
}

// JobMapFromCatalog reads the catalog's CIJobNames fields and returns
// a job_name → check_id lookup. Duplicate job names map to whichever
// check defined them last; callers should avoid overlapping names.
func JobMapFromCatalog(cat *catalog.Catalog) map[string]string {
	m := make(map[string]string)
	for _, ct := range cat.CheckTypes {
		for _, name := range ct.CIJobNames {
			m[name] = ct.ID
		}
	}
	return m
}

// Fetch walks pipelines on cfg.Branch newer than `since`, collects
// workflow→job status, and emits one Event per pipeline. Pipelines
// whose revision is unknown to the local git repo are skipped (the
// file list can't be resolved); pipelines with no mapped jobs are
// also skipped.
func (f *CircleCIFetcher) Fetch(since time.Time) ([]Event, error) {
	pipelines, err := f.walkPipelines(since)
	if err != nil {
		return nil, err
	}

	var events []Event
	for _, p := range pipelines {
		// Branch filter: if a specific branch is configured, only
		// include its pipelines and don't apply the exclude-prefix
		// list (the explicit request wins — otherwise the default
		// exclude "develop,main,master,release/" silently drops
		// `--circleci-branch develop`, since develop is on both
		// sides). Empty Branch means "all branches," where the
		// exclude filter is what keeps post-merge trunks and the
		// operator's opt-out prefixes out.
		if f.cfg.Branch != "" {
			if p.VCS.Branch != f.cfg.Branch {
				continue
			}
		} else if hasBranchPrefix(p.VCS.Branch, f.cfg.ExcludeBranchPrefixes) {
			continue
		}
		checks, err := f.fetchPipelineChecks(p.ID)
		if err != nil {
			return nil, fmt.Errorf("pipeline %s: %w", p.ID, err)
		}
		if len(checks) == 0 {
			continue
		}

		files, err := f.cfg.FilesFor(p.VCS.Revision)
		if err != nil {
			// Revision not present locally, or git failure. Skip —
			// without the file list we can't produce useful correlations.
			continue
		}
		if len(files) == 0 {
			continue
		}

		events = append(events, Event{
			PR:       f.cfg.PRFor(p.VCS.Revision),
			MergedAt: p.CreatedAt,
			Files:    files,
			Checks:   checks,
		})
	}
	return events, nil
}

// --- pipeline / workflow / job walks ---

type ccPipeline struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	VCS       struct {
		Revision string `json:"revision"`
		Branch   string `json:"branch"`
	} `json:"vcs"`
}

type ccPipelinePage struct {
	Items         []ccPipeline `json:"items"`
	NextPageToken string       `json:"next_page_token"`
}

type ccWorkflowPage struct {
	Items []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"items"`
	NextPageToken string `json:"next_page_token"`
}

type ccJobPage struct {
	Items []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"items"`
	NextPageToken string `json:"next_page_token"`
}

// walkPipelines paginates /project/.../pipeline, optionally filtered
// by branch at the API layer. Stops when page contains a pipeline
// older than `since` (pipelines are returned newest-first).
func (f *CircleCIFetcher) walkPipelines(since time.Time) ([]ccPipeline, error) {
	path := fmt.Sprintf("/project/github/%s/%s/pipeline", f.cfg.Org, f.cfg.Repo)
	params := url.Values{}
	if f.cfg.Branch != "" {
		params.Set("branch", f.cfg.Branch)
	}

	var out []ccPipeline
	for page := 0; page < f.cfg.MaxPages; page++ {
		var resp ccPipelinePage
		if err := f.get(path, params, &resp); err != nil {
			return nil, fmt.Errorf("pipelines page %d: %w", page, err)
		}
		stop := false
		for _, p := range resp.Items {
			if !since.IsZero() && p.CreatedAt.Before(since) {
				stop = true
				break
			}
			out = append(out, p)
		}
		if stop || resp.NextPageToken == "" {
			break
		}
		params.Set("page-token", resp.NextPageToken)
	}
	return out, nil
}

// fetchPipelineChecks collects all jobs across all workflows of a
// pipeline, maps their names to check IDs via jobToCheck, and
// aggregates statuses: any "failed"-class result on any workflow for
// that check name flags the check as failed.
func (f *CircleCIFetcher) fetchPipelineChecks(pipelineID string) ([]CheckRun, error) {
	wfs, err := f.workflowsForPipeline(pipelineID)
	if err != nil {
		return nil, err
	}

	ran := make(map[string]bool)    // checkID → any workflow ran this check
	failed := make(map[string]bool) // checkID → any workflow failed this check

	for _, wfID := range wfs {
		jobs, err := f.jobsForWorkflow(wfID)
		if err != nil {
			return nil, fmt.Errorf("workflow %s: %w", wfID, err)
		}
		for jobName, status := range jobs {
			checkID, ok := f.jobToCheck[jobName]
			if !ok {
				continue
			}
			kind := classifyStatus(status)
			if kind == statusInFlight || kind == statusSkip {
				continue
			}
			ran[checkID] = true
			if kind == statusFailed {
				failed[checkID] = true
			}
		}
	}

	var out []CheckRun
	for checkID := range ran {
		out = append(out, CheckRun{ID: checkID, Failed: failed[checkID]})
	}
	return out, nil
}

func (f *CircleCIFetcher) workflowsForPipeline(pipelineID string) ([]string, error) {
	path := fmt.Sprintf("/pipeline/%s/workflow", pipelineID)
	var ids []string
	params := url.Values{}
	for page := 0; page < f.cfg.MaxPages; page++ {
		var resp ccWorkflowPage
		if err := f.get(path, params, &resp); err != nil {
			return nil, err
		}
		for _, w := range resp.Items {
			ids = append(ids, w.ID)
		}
		if resp.NextPageToken == "" {
			break
		}
		params.Set("page-token", resp.NextPageToken)
	}
	return ids, nil
}

// jobsForWorkflow returns job name → raw status string. If a job appears
// twice (retry), the later entry wins in map insertion order, which is
// acceptable since classifyStatus handles all terminal states.
func (f *CircleCIFetcher) jobsForWorkflow(workflowID string) (map[string]string, error) {
	path := fmt.Sprintf("/workflow/%s/job", workflowID)
	out := make(map[string]string)
	params := url.Values{}
	for page := 0; page < f.cfg.MaxPages; page++ {
		var resp ccJobPage
		if err := f.get(path, params, &resp); err != nil {
			return nil, err
		}
		for _, j := range resp.Items {
			out[j.Name] = j.Status
		}
		if resp.NextPageToken == "" {
			break
		}
		params.Set("page-token", resp.NextPageToken)
	}
	return out, nil
}

// --- HTTP plumbing ---

func (f *CircleCIFetcher) get(path string, params url.Values, into interface{}) error {
	// CircleCI's edge occasionally RSTs long polling sessions mid-page.
	// Retry a handful of times on transient network / 5xx errors so an
	// hour-long ingest doesn't bomb on a single connection reset.
	const maxAttempts = 4
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}
		err := f.getOnce(path, params, into)
		if err == nil {
			return nil
		}
		if !isTransientHTTPError(err) {
			return err
		}
		lastErr = err
	}
	return fmt.Errorf("after %d attempts: %w", maxAttempts, lastErr)
}

func (f *CircleCIFetcher) getOnce(path string, params url.Values, into interface{}) error {
	u := f.cfg.BaseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	if f.cfg.Token != "" {
		req.Header.Set("Circle-Token", f.cfg.Token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("CircleCI rate limited (HTTP 429); retry after %q", resp.Header.Get("Retry-After"))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("CircleCI %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

// isTransientHTTPError reports whether err is worth retrying. Covers
// connection resets, broken pipes, EOFs mid-response, request timeouts,
// and 5xx responses wrapped into errors by getOnce.
func isTransientHTTPError(err error) bool {
	s := err.Error()
	for _, marker := range []string{
		"connection reset by peer",
		"broken pipe",
		"EOF",
		"i/o timeout",
		"Client.Timeout",
		"returned 5",
	} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

// --- status classification ---

type statusKind int

const (
	statusPassed statusKind = iota
	statusFailed
	statusInFlight
	statusSkip // canceled / blocked / not_run — didn't contribute a result
)

// classifyStatus groups CircleCI job statuses into selector-relevant buckets.
// Authoritative enum: https://circleci.com/docs/api/v2/index.html
func classifyStatus(s string) statusKind {
	switch s {
	case "success":
		return statusPassed
	case "failed", "infrastructure_fail", "timedout":
		return statusFailed
	case "queued", "running":
		return statusInFlight
	case "canceled", "on_hold", "blocked", "not_run", "not_running",
		"unauthorized", "retried", "terminated-unknown":
		return statusSkip
	}
	return statusSkip
}

// hasBranchPrefix reports whether any prefix in the list is a prefix
// of branch. Empty prefix list returns false.
func hasBranchPrefix(branch string, prefixes []string) bool {
	for _, p := range prefixes {
		if p == "" {
			continue
		}
		if strings.HasPrefix(branch, p) {
			return true
		}
	}
	return false
}

// ResolveFilesViaGit returns the list of repo-relative paths changed
// in the given revision, by shelling to `git diff-tree`. Works for
// squash-merge and rebase-merge commits; for true merges the combined
// diff is returned. Missing revisions return an error so the caller
// can skip those pipelines rather than fail the whole walk.
func ResolveFilesViaGit(rootDir, revision string) ([]string, error) {
	if rootDir == "" {
		return nil, fmt.Errorf("RepoRoot not set; cannot resolve files for %s", revision)
	}
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", revision)
	cmd.Dir = rootDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff-tree %s: %w", revision, err)
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	files := strings.Split(trimmed, "\n")
	// Filter empty lines defensively.
	filtered := files[:0]
	for _, f := range files {
		if f != "" {
			filtered = append(filtered, f)
		}
	}
	return filtered, nil
}

// prSubjectRef matches squash-merge commit subjects like
// "feat(op-node): fooize bar (#20164)" — the conventional format
// GitHub's squash-merge button produces on ethereum-optimism/optimism.
// Not brittle because squash-merge is the only supported merge mode
// on the repo.
var prSubjectRef = regexp.MustCompile(`\(#(\d+)\)\s*$`)

// ResolvePRViaGit parses the PR number out of a commit's subject
// line. Returns 0 when the commit message doesn't reference a PR
// (pre-merge PR-branch commits, merge commits from other repos) or
// when git fails. The returned integer is display/filter metadata
// only — replay doesn't gate on it.
func ResolvePRViaGit(rootDir, revision string) int {
	if rootDir == "" {
		return 0
	}
	cmd := exec.Command("git", "log", "-1", "--format=%s", revision)
	cmd.Dir = rootDir
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	m := prSubjectRef.FindStringSubmatch(strings.TrimSpace(string(out)))
	if len(m) < 2 {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return n
}
