package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	github "github.com/google/go-github/v55/github" // newer version of Go is needed for the latest GitHub API
	"golang.org/x/oauth2"
	yaml "gopkg.in/yaml.v3"
)

var logger *log.Logger

// CircleCI API models
type pipelineList struct {
	Items         []pipeline `json:"items"`
	NextPageToken string     `json:"next_page_token"`
}

type pipeline struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type workflowList struct {
	Items         []workflow `json:"items"`
	NextPageToken string     `json:"next_page_token"`
}

type workflow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type jobList struct {
	Items         []job  `json:"items"`
	NextPageToken string `json:"next_page_token"`
}

type job struct {
	Name      string `json:"name"`
	JobNumber int    `json:"job_number"`
}

type artifactsList struct {
	Items []artifact `json:"items"`
}

type artifact struct {
	URL  string `json:"url"`
	Path string `json:"path"`
}

// We rely solely on daily-summary.json artifacts for aggregation.

// Daily summary (as produced in CI job)
type DailySummary struct {
	Date       string `json:"date"`
	Gate       string `json:"gate"`
	TotalRuns  int    `json:"total_runs"`
	Iterations int    `json:"iterations"`
	Totals     struct {
		Stable   int `json:"stable"`
		Unstable int `json:"unstable"`
	} `json:"totals"`
	StableTests []struct {
		TestName  string  `json:"test_name"`
		Package   string  `json:"package"`
		TotalRuns int     `json:"total_runs"`
		PassRate  float64 `json:"pass_rate"`
	} `json:"stable_tests"`
	UnstableTests []struct {
		TestName  string  `json:"test_name"`
		Package   string  `json:"package"`
		TotalRuns int     `json:"total_runs"`
		Passes    int     `json:"passes"`
		Failures  int     `json:"failures"`
		PassRate  float64 `json:"pass_rate"`
	} `json:"unstable_tests"`
}

// Acceptance tests YAML models
type acceptanceYAML struct {
	Gates []gateYAML `yaml:"gates"`
}

type gateYAML struct {
	ID          string      `yaml:"id"`
	Description string      `yaml:"description,omitempty"`
	Inherits    []string    `yaml:"inherits,omitempty"`
	Tests       []testEntry `yaml:"tests,omitempty"`
}

type testEntry struct {
	Name     string                 `yaml:"name,omitempty"`
	Package  string                 `yaml:"package"`
	Timeout  string                 `yaml:"timeout,omitempty"`
	Metadata map[string]interface{} `yaml:"metadata,omitempty"`
}

// Aggregated per test across days
type aggStats struct {
	Package       string     `json:"package"`
	TestName      string     `json:"test_name"`
	TotalRuns     int        `json:"total_runs"`
	Passes        int        `json:"passes"`
	Failures      int        `json:"failures"`
	FirstSeenDay  string     `json:"first_seen_day"`
	LastSeenDay   string     `json:"last_seen_day"`
	LastFailureAt *time.Time `json:"last_failure_at,omitempty"`
	DaysObserved  []string   `json:"days_observed"`
}

type promoteCandidate struct {
	Package      string  `json:"package"`
	TestName     string  `json:"test_name"`
	TotalRuns    int     `json:"total_runs"`
	PassRate     float64 `json:"pass_rate"`
	TargetGate   string  `json:"target_gate"`
	Timeout      string  `json:"timeout"`
	FirstSeenDay string  `json:"first_seen_day"`
}

func main() {
	var (
		org             string
		repo            string
		branch          string
		workflowName    string
		reportJobName   string
		daysBack        int
		gateID          string
		minRuns         int
		maxFailureRate  float64
		minAgeDays      int
		outDir          string
		dryRun          bool
		requireClean24h bool
		verbose         bool
	)

	flag.StringVar(&org, "org", "ethereum-optimism", "GitHub org")
	flag.StringVar(&repo, "repo", "optimism", "GitHub repo")
	flag.StringVar(&branch, "branch", "develop", "Branch to scan")
	flag.StringVar(&workflowName, "workflow", "scheduled-flake-shake", "Workflow name")
	flag.StringVar(&reportJobName, "report-job", "op-acceptance-tests-flake-shake-report", "Report job name")
	flag.IntVar(&daysBack, "days", 3, "Number of days to aggregate")
	flag.StringVar(&gateID, "gate", "flake-shake", "Gate id in acceptance-tests.yaml")
	flag.IntVar(&minRuns, "min-runs", 300, "Minimum total runs required")
	flag.Float64Var(&maxFailureRate, "max-failure-rate", 0.01, "Maximum allowed failure rate")
	flag.IntVar(&minAgeDays, "min-age-days", 3, "Minimum age in days in flake-shake")
	flag.StringVar(&outDir, "out", "./promotion-output", "Output directory")
	flag.BoolVar(&dryRun, "dry-run", true, "Do not modify repo or open PRs")
	flag.BoolVar(&requireClean24h, "require-clean-24h", false, "Require no failures in the last 24 hours")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose debug logging")
	flag.Parse()

	logger = log.New(os.Stdout, "[flake-shake-promoter] ", log.LstdFlags)
	if verbose {
		logger.Printf("Flags: org=%s repo=%s branch=%s workflow=%s report_job=%s days=%d gate=%s min_runs=%d max_failure_rate=%.4f min_age_days=%d require_clean_24h=%t out=%s dry_run=%t",
			org, repo, branch, workflowName, reportJobName, daysBack, gateID, minRuns, maxFailureRate, minAgeDays, requireClean24h, outDir, dryRun,
		)
	}

	token := os.Getenv("CIRCLE_API_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "CIRCLE_API_TOKEN is not set")
		os.Exit(1)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create out dir: %v\n", err)
		os.Exit(1)
	}

	now := time.Now().UTC()
	since := now.AddDate(0, 0, -daysBack)

	client := &http.Client{Timeout: 30 * time.Second}
	ctx := &apiCtx{client: client, token: token}

	dailyReports, err := collectReports(ctx, org, repo, branch, workflowName, reportJobName, since, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "collection failed: %v\n", err)
		os.Exit(1)
	}

	agg := aggregate(dailyReports)

	if verbose {
		logger.Printf("Collected %d day(s) of summaries.", len(dailyReports))
		totalTests := 0
		for date, ds := range dailyReports {
			n := len(ds.StableTests) + len(ds.UnstableTests)
			totalTests += n
			logger.Printf("  - %s: %d tests (stable=%d unstable=%d)", date, n, len(ds.StableTests), len(ds.UnstableTests))
		}
		logger.Printf("Total tests across days: %d", totalTests)
	}

	// Load acceptance-tests.yaml
	yamlPath := filepath.Join("op-acceptance-tests", "acceptance-tests.yaml")
	cfg, err := readAcceptanceYAML(yamlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed reading %s: %v\n", yamlPath, err)
		os.Exit(1)
	}

	// Build indices for flake-shake tests and target gates
	flakeGate := findGate(&cfg, gateID)
	if flakeGate == nil {
		fmt.Fprintf(os.Stderr, "gate %s not found in %s\n", gateID, yamlPath)
		os.Exit(1)
	}
	gateIndex := map[string]*gateYAML{}
	for i := range cfg.Gates {
		gateIndex[cfg.Gates[i].ID] = &cfg.Gates[i]
	}

	// Map tests in flake-shake: key -> (timeout, target_gate, name)
	type testInfo struct {
		Timeout   string
		Target    string
		Name      string
		Meta      map[string]interface{}
		GateIndex int
		TestIndex int
	}
	flakeTests := map[string]testInfo{}
	for ti := range flakeGate.Tests {
		t := flakeGate.Tests[ti]
		var target string
		if t.Metadata != nil {
			if v, ok := t.Metadata["target_gate"].(string); ok {
				target = v
			}
		}
		key := keyFor(t.Package, t.Name)
		flakeTests[key] = testInfo{Timeout: t.Timeout, Target: target, Name: t.Name, Meta: t.Metadata, GateIndex: indexOfGate(&cfg, gateID), TestIndex: ti}
	}

	// Select promotion candidates
	candidates := []promoteCandidate{}
	reasons := map[string]string{}
	for key, s := range agg {
		info, ok := flakeTests[key]
		if !ok {
			continue // only promote tests currently in flake-shake
		}
		if info.Target == "" {
			reasons[key] = "missing target_gate metadata"
			continue
		}
		if s.TotalRuns < minRuns {
			reasons[key] = fmt.Sprintf("insufficient runs: %d < %d", s.TotalRuns, minRuns)
			continue
		}
		failureRate := 0.0
		if s.TotalRuns > 0 {
			failureRate = float64(s.Failures) / float64(s.TotalRuns)
		}
		if failureRate > maxFailureRate {
			reasons[key] = fmt.Sprintf("failure rate %.4f exceeds max %.4f", failureRate, maxFailureRate)
			continue
		}
		if requireClean24h && s.LastFailureAt != nil {
			if time.Since(*s.LastFailureAt) < 24*time.Hour {
				reasons[key] = "failure within last 24h"
				continue
			}
		}
		// Age criterion: days from first observed day to now >= minAgeDays
		if s.FirstSeenDay == "" {
			reasons[key] = "no age information"
			continue
		}
		firstDay, _ := time.Parse("2006-01-02", s.FirstSeenDay)
		daysInGate := int(now.Sub(firstDay).Hours()/24) + 1
		if daysInGate < minAgeDays {
			reasons[key] = fmt.Sprintf("min age %dd not met (have %dd)", minAgeDays, daysInGate)
			continue
		}

		passRate := 0.0
		if s.TotalRuns > 0 {
			passRate = float64(s.Passes) / float64(s.TotalRuns)
		}
		candidates = append(candidates, promoteCandidate{
			Package:      s.Package,
			TestName:     s.TestName,
			TotalRuns:    s.TotalRuns,
			PassRate:     passRate * 100.0,
			TargetGate:   info.Target,
			Timeout:      info.Timeout,
			FirstSeenDay: s.FirstSeenDay,
		})
	}

	// Write outputs
	if err := writeJSON(filepath.Join(outDir, "aggregate.json"), agg); err != nil {
		fmt.Fprintf(os.Stderr, "failed writing aggregate: %v\n", err)
		os.Exit(1)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].TargetGate == candidates[j].TargetGate {
			if candidates[i].Package == candidates[j].Package {
				return candidates[i].TestName < candidates[j].TestName
			}
			return candidates[i].Package < candidates[j].Package
		}
		return candidates[i].TargetGate < candidates[j].TargetGate
	})
	if err := writeJSON(filepath.Join(outDir, "promotion-ready.json"), map[string]interface{}{"candidates": candidates, "skipped": reasons}); err != nil {
		fmt.Fprintf(os.Stderr, "failed writing promotion-ready: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Promotion candidates: %d\n", len(candidates))
		for _, c := range candidates {
			fmt.Printf("  - %s %s -> %s (runs=%d pass=%.2f%%)\n", c.Package, c.TestName, c.TargetGate, c.TotalRuns, c.PassRate)
		}
	}

	// Write metadata for downstream consumers (e.g., Slack)
	meta := map[string]interface{}{
		"date":             now.Format("2006-01-02"),
		"gate":             gateID,
		"candidates":       len(candidates),
		"flake_gate_tests": len(flakeGate.Tests),
	}
	if err := writeJSON(filepath.Join(outDir, "metadata.json"), meta); err != nil {
		fmt.Fprintf(os.Stderr, "failed writing metadata: %v\n", err)
		os.Exit(1)
	}

	// Generate updated YAML (proposal)
	updated := cfg // copy
	// Build map for quick removal
	flakeIdx := indexOfGate(&updated, gateID)
	if flakeIdx < 0 {
		fmt.Fprintf(os.Stderr, "gate %s not found when updating\n", gateID)
		os.Exit(1)
	}

	// Determine which tests to promote by key
	promoteKeys := map[string]promoteCandidate{}
	for _, c := range candidates {
		promoteKeys[keyFor(c.Package, c.TestName)] = c
	}

	// Remove from flake-shake and add to target gates
	newFlakeTests := make([]testEntry, 0, len(updated.Gates[flakeIdx].Tests))
	for _, t := range updated.Gates[flakeIdx].Tests {
		k := keyFor(t.Package, t.Name)
		c, ok := promoteKeys[k]
		if !ok {
			newFlakeTests = append(newFlakeTests, t)
			continue
		}
		// Add to target gate
		tgt := findGate(&updated, c.TargetGate)
		if tgt == nil {
			// If target gate is missing, skip moving but keep in flake-shake
			newFlakeTests = append(newFlakeTests, t)
			reasons[k] = fmt.Sprintf("target gate %s not found", c.TargetGate)
			continue
		}
		promoted := t
		// Preserve package/name/timeout; drop metadata.target_gate
		if promoted.Metadata != nil {
			delete(promoted.Metadata, "target_gate")
			if len(promoted.Metadata) == 0 {
				promoted.Metadata = nil
			}
		}
		// Ensure timeout preserved; if empty, leave as-is
		if c.Timeout != "" {
			promoted.Timeout = c.Timeout
		}
		tgt.Tests = append(tgt.Tests, promoted)
	}
	updated.Gates[flakeIdx].Tests = newFlakeTests

	// Write proposed YAML
	outYAML := filepath.Join(outDir, "promotion.yaml")
	if err := writeYAML(outYAML, &updated); err != nil {
		fmt.Fprintf(os.Stderr, "failed writing promotion.yaml: %v\n", err)
		os.Exit(1)
	}

	// Print short summary
	if len(candidates) == 0 {
		// Build an explanatory summary of why there are no candidates
		earliest := ""
		totalRuns := 0
		totalPass := 0
		totalFail := 0
		daySet := map[string]struct{}{}
		for key, s := range agg {
			// Only consider tests currently in flake-shake
			if _, ok := flakeTests[key]; !ok {
				continue
			}
			totalRuns += s.TotalRuns
			totalPass += s.Passes
			totalFail += s.Failures
			if earliest == "" || (s.FirstSeenDay != "" && s.FirstSeenDay < earliest) {
				earliest = s.FirstSeenDay
			}
			for _, d := range s.DaysObserved {
				daySet[d] = struct{}{}
			}
		}
		daysObserved := len(daySet)
		reason := fmt.Sprintf(
			"No promotion candidates. Reason: min_age_days=%d; earliest_observation=%s; days_observed=%d; require_clean_24h=%t; total_runs=%d; passes=%d; failures=%d.",
			minAgeDays, earliest, daysObserved, requireClean24h, totalRuns, totalPass, totalFail,
		)
		_ = os.WriteFile(filepath.Join(outDir, "SUMMARY.txt"), []byte(reason+"\n"), 0o644)
		logger.Println(reason)
		return
	}
	var b bytes.Buffer
	b.WriteString("Promotion candidates (dry-run):\n")
	for _, c := range candidates {
		b.WriteString(fmt.Sprintf("- %s %s -> %s (runs=%d, pass=%.2f%%)\n", c.Package, c.TestName, c.TargetGate, c.TotalRuns, c.PassRate))
	}
	_ = os.WriteFile(filepath.Join(outDir, "SUMMARY.txt"), b.Bytes(), 0o644)
	logger.Print(b.String())

	if dryRun {
		logger.Println("Dry-run enabled; skipping branch creation, file update, and PR creation.")
		return
	}

	// Prepare updated YAML content for PR (do not modify working tree)
	updatedYAMLBytes, err := yaml.Marshal(&updated)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal updated YAML: %v\n", err)
		os.Exit(1)
	}

	prBranch := fmt.Sprintf("ci/flake-shake-promote/%s", time.Now().UTC().Format("2006-01-02-150405"))

	// Prepare commit message and PR body
	title := "chore(op-acceptance-tests): flake-shake; test promotions"
	var body bytes.Buffer
	body.WriteString("## 🤖 Automated Flake-Shake Test Promotion\n\n")
	body.WriteString(fmt.Sprintf("Promoting %d test(s) from gate `"+gateID+"` based on stability criteria.\n\n", len(candidates)))
	body.WriteString("### Tests Being Promoted\n\n")
	body.WriteString("| Test | Package | Target Gate | Total Runs | Pass Rate |\n|---|---|---|---:|---:|\n")
	for _, c := range candidates {
		body.WriteString(fmt.Sprintf("| %s | %s | %s | %d | %.2f%% |\n", c.TestName, c.Package, c.TargetGate, c.TotalRuns, c.PassRate))
	}
	body.WriteString("\nThis PR was auto-generated by flake-shake promoter.\n")

	// Use GitHub API to create branch, update file, and open PR
	ghToken := os.Getenv("GH_TOKEN")
	if ghToken == "" {
		fmt.Fprintln(os.Stderr, "GH_TOKEN is required for PR creation but not set")
		os.Exit(1)
	}
	ghCtx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ghCtx, ts)
	ghc := github.NewClient(tc)

	if verbose {
		logger.Printf("PR: starting creation process (base_branch=%s candidates=%d)", branch, len(candidates))
	}

	// 1) Get base branch ref
	baseRef, _, err := ghc.Git.GetRef(ghCtx, org, repo, "refs/heads/"+branch)
	if err != nil || baseRef.Object == nil || baseRef.Object.SHA == nil {
		fmt.Fprintf(os.Stderr, "failed to get base ref: %v\n", err)
		os.Exit(1)
	}
	if verbose {
		logger.Printf("PR: base ref resolved sha=%s", baseRef.GetObject().GetSHA())
	}

	// 2) Create new branch ref
	newRef := &github.Reference{
		Ref:    github.String("refs/heads/" + prBranch),
		Object: &github.GitObject{SHA: baseRef.Object.SHA},
	}
	if _, _, err := ghc.Git.CreateRef(ghCtx, org, repo, newRef); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create ref: %v\n", err)
		os.Exit(1)
	}
	if verbose {
		logger.Printf("PR: created branch %s", prBranch)
	}

	// 3) Read current file to fetch SHA (if exists) on base branch
	path := yamlPath
	var sha *string
	if fileContent, _, resp, err := ghc.Repositories.GetContents(ghCtx, org, repo, path, &github.RepositoryContentGetOptions{Ref: branch}); err == nil && fileContent != nil {
		sha = fileContent.SHA
	} else if resp != nil && resp.StatusCode == 404 {
		sha = nil
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get contents: %v\n", err)
		os.Exit(1)
	}

	// 4) Update file in new branch
	commitMsg := title
	if _, _, err := ghc.Repositories.UpdateFile(ghCtx, org, repo, path, &github.RepositoryContentFileOptions{
		Message: github.String(commitMsg),
		Content: updatedYAMLBytes,
		Branch:  github.String(prBranch),
		SHA:     sha,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to update file: %v\n", err)
		os.Exit(1)
	}
	if verbose {
		logger.Printf("PR: updated file %s on branch %s", path, prBranch)
	}

	// 5) Create PR
	prReq := &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(prBranch),
		Base:  github.String(branch),
		Body:  github.String(body.String()),
	}
	pr, _, err := ghc.PullRequests.Create(ghCtx, org, repo, prReq)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create PR: %v\n", err)
		os.Exit(1)
	}
	logger.Printf("PR created: %s (number=%d)", pr.GetHTMLURL(), pr.GetNumber())

	// Update metadata with PR details for downstream Slack notification
	meta["pr_url"] = pr.GetHTMLURL()
	meta["pr_number"] = pr.GetNumber()
	if err := writeJSON(filepath.Join(outDir, "metadata.json"), meta); err != nil {
		fmt.Fprintf(os.Stderr, "failed updating metadata with PR info: %v\n", err)
	}

	// 6) Add labels
	if _, _, err := ghc.Issues.AddLabelsToIssue(ghCtx, org, repo, pr.GetNumber(), []string{"M-ci", "A-acceptance-tests"}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to add labels: %v\n", err)
	}

	// 7) Request reviewers (user and team slug)
	if _, _, err := ghc.PullRequests.RequestReviewers(ghCtx, org, repo, pr.GetNumber(), github.ReviewersRequest{
		Reviewers:     []string{"scharissis"},
		TeamReviewers: []string{"platforms-team"},
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to request reviewers: %v\n", err)
	}
}

// HTTP helper context
type apiCtx struct {
	client *http.Client
	token  string
}

func (c *apiCtx) getJSON(u string, v interface{}) error {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Circle-Token", c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s: status %d body=%s", u, resp.StatusCode, string(body))
	}
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(v)
}

func (c *apiCtx) getBytes(u string) ([]byte, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Circle-Token", c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET %s: status %d body=%s", u, resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

// runCmd executes a command and streams output; returns error on non-zero exit
// (no external command execution required; GitHub API handles branch, commit, PR)

func collectReports(ctx *apiCtx, org, repo, branch, workflowName, reportJobName string, since time.Time, verbose bool) (map[string]DailySummary, error) {
	dailyByDay := map[string]DailySummary{}

	basePipelines := fmt.Sprintf("https://circleci.com/api/v2/project/gh/%s/%s/pipeline?branch=%s", url.PathEscape(org), url.PathEscape(repo), url.QueryEscape(branch))

	pageURL := basePipelines
	for {
		var pl pipelineList
		if err := ctx.getJSON(pageURL, &pl); err != nil {
			return nil, err
		}
		if verbose {
			logger.Printf("Scanning pipelines page: %s", pageURL)
		}
		// Process newest first; CircleCI already returns newest-first order
		for _, p := range pl.Items {
			if verbose {
				logger.Printf("  pipeline %s created_at=%s", p.ID, p.CreatedAt.Format(time.RFC3339))
			}
			if p.CreatedAt.Before(since) {
				// Current page includes items older than our window; stop scanning further pages
				return dailyByDay, nil
			}
			// Workflows for this pipeline
			wfURL := fmt.Sprintf("https://circleci.com/api/v2/pipeline/%s/workflow", p.ID)
			var wfl workflowList
			if err := ctx.getJSON(wfURL, &wfl); err != nil {
				return nil, err
			}
			for _, w := range wfl.Items {
				if w.Name != workflowName {
					continue
				}
				// Jobs for workflow
				jobsURL := fmt.Sprintf("https://circleci.com/api/v2/workflow/%s/job", w.ID)
				var jl jobList
				if err := ctx.getJSON(jobsURL, &jl); err != nil {
					return nil, err
				}
				for _, j := range jl.Items {
					if j.Name != reportJobName {
						continue
					}
					// Artifacts for job number
					artsURL := fmt.Sprintf("https://circleci.com/api/v2/project/gh/%s/%s/%d/artifacts", url.PathEscape(org), url.PathEscape(repo), j.JobNumber)
					var al artifactsList
					if err := ctx.getJSON(artsURL, &al); err != nil {
						return nil, err
					}
					if verbose {
						logger.Printf("    job %d artifacts: %d", j.JobNumber, len(al.Items))
						for _, a := range al.Items {
							logger.Printf("      - %s", a.Path)
						}
					}
					var dailyURL string
					for _, a := range al.Items {
						// Accept any artifact path that ends with the filename, regardless of destination prefix
						if strings.HasSuffix(a.Path, "daily-summary.json") {
							dailyURL = a.URL
						}
					}
					if dailyURL == "" {
						continue
					}
					data, err := ctx.getBytes(dailyURL)
					if err == nil {
						var ds DailySummary
						if json.Unmarshal(data, &ds) == nil && ds.Date != "" {
							if prev, seen := dailyByDay[ds.Date]; !seen {
								dailyByDay[ds.Date] = ds
								if verbose {
									logger.Printf("    loaded daily summary for %s (runs=%d iterations=%d)", ds.Date, ds.TotalRuns, ds.Iterations)
								}
							} else {
								// Merge multiple runs on the same day by summing totals and appending per-test entries.
								merged := prev
								merged.TotalRuns += ds.TotalRuns
								merged.Iterations += ds.Iterations
								merged.StableTests = append(merged.StableTests, ds.StableTests...)
								merged.UnstableTests = append(merged.UnstableTests, ds.UnstableTests...)
								dailyByDay[ds.Date] = merged
								if verbose {
									logger.Printf("    merged another run for %s (+runs=%d +iters=%d) now runs=%d iters=%d", ds.Date, ds.TotalRuns, ds.Iterations, merged.TotalRuns, merged.Iterations)
								}
							}
						}
					}
				}
			}
		}
		if pl.NextPageToken == "" {
			break
		}
		pageURL = basePipelines + "&page-token=" + url.QueryEscape(pl.NextPageToken)
	}
	return dailyByDay, nil
}

func aggregate(daily map[string]DailySummary) map[string]*aggStats {
	result := map[string]*aggStats{}
	// Collect all days
	days := make([]string, 0, len(daily))
	for d := range daily {
		days = append(days, d)
	}
	sort.Strings(days)

	for _, day := range days {
		if ds, ok := daily[day]; ok {
			for _, t := range ds.StableTests {
				k := keyFor(t.Package, t.TestName)
				s := ensureAgg(result, k, t.Package, t.TestName, day)
				s.TotalRuns += t.TotalRuns
				s.Passes += t.TotalRuns
			}
			for _, t := range ds.UnstableTests {
				k := keyFor(t.Package, t.TestName)
				s := ensureAgg(result, k, t.Package, t.TestName, day)
				s.TotalRuns += t.TotalRuns
				s.Passes += t.Passes
				s.Failures += t.Failures
				approx := parseDayEnd(day)
				if s.LastFailureAt == nil || approx.After(*s.LastFailureAt) {
					s.LastFailureAt = &approx
				}
			}
		}
	}
	return result
}

func ensureAgg(m map[string]*aggStats, key, pkg, name, day string) *aggStats {
	s, ok := m[key]
	if !ok {
		s = &aggStats{Package: pkg, TestName: name, DaysObserved: []string{}, FirstSeenDay: day, LastSeenDay: day}
		m[key] = s
	}
	// Append day if new
	found := false
	for _, d := range s.DaysObserved {
		if d == day {
			found = true
			break
		}
	}
	if !found {
		s.DaysObserved = append(s.DaysObserved, day)
		if s.FirstSeenDay == "" || day < s.FirstSeenDay {
			s.FirstSeenDay = day
		}
		if s.LastSeenDay == "" || day > s.LastSeenDay {
			s.LastSeenDay = day
		}
	}
	return s
}

func parseDayEnd(day string) time.Time {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return time.Now().UTC()
	}
	return t.Add(24*time.Hour - time.Nanosecond).UTC()
}

func keyFor(pkg, name string) string {
	if strings.TrimSpace(name) == "" {
		return pkg + "::"
	}
	return pkg + "::" + name
}

func readAcceptanceYAML(path string) (acceptanceYAML, error) {
	var acc acceptanceYAML
	data, err := os.ReadFile(path)
	if err != nil {
		return acc, err
	}
	if err := yaml.Unmarshal(data, &acc); err != nil {
		return acc, err
	}
	if len(acc.Gates) == 0 {
		return acc, errors.New("no gates found")
	}
	return acc, nil
}

func findGate(acc *acceptanceYAML, id string) *gateYAML {
	for i := range acc.Gates {
		if acc.Gates[i].ID == id {
			return &acc.Gates[i]
		}
	}
	return nil
}

func indexOfGate(acc *acceptanceYAML, id string) int {
	for i := range acc.Gates {
		if acc.Gates[i].ID == id {
			return i
		}
	}
	return -1
}

func writeJSON(path string, v interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeYAML(path string, v interface{}) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	// Normalize line endings
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	return os.WriteFile(path, data, 0o644)
}
