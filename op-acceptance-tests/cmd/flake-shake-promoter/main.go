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

// Constants used across the promoter
const (
	flakeShakeGateID         = "flake-shake"
	flakeShakeWorkflowName   = "scheduled-flake-shake"
	flakeShakeReportJobName  = "op-acceptance-tests-flake-shake-report"
	flakeShakePRTitle        = "chore(op-acceptance-tests): flake-shake; test promotions"
	flakeShakePRBranchPrefix = "ci/flake-shake-promote/"
	flakeShakeLabel          = "M-ci"
	flakeShakeBotAuthor      = "opgitgovernance"
	flakeShakeSupersedeDays  = 2 // lookback window (days) when closing older PRs as superseded
)

// CircleCI API models
type pipelineList struct {
	Items         []pipeline `json:"items"`
	NextPageToken string     `json:"next_page_token"`
}

type pipeline struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Number    int       `json:"number"`
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
	WebURL    string `json:"web_url"`
}

type artifactsList struct {
	Items []artifact `json:"items"`
}

type artifact struct {
	URL  string `json:"url"`
	Path string `json:"path"`
}

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
	Owner    string                 `yaml:"owner,omitempty"`
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
	Timeout      string  `json:"timeout"`
	FirstSeenDay string  `json:"first_seen_day"`
	Owner        string  `json:"owner,omitempty"`
}

// Map tests in flake-shake: key -> (timeout, name)
type testInfo struct {
	Timeout   string
	Name      string
	Meta      map[string]interface{}
	Owner     string
	GateIndex int
	TestIndex int
}

func main() {
	opts := parsePromoterFlags()

	logger = log.New(os.Stdout, "[flake-shake-promoter] ", log.LstdFlags)
	if opts.verbose {
		logger.Printf("Flags: org=%s repo=%s branch=%s workflow=%s report_job=%s days=%d gate=%s min_runs=%d max_failure_rate=%.4f min_age_days=%d require_clean_24h=%t out=%s dry_run=%t",
			opts.org, opts.repo, opts.branch, opts.workflowName, opts.reportJobName, opts.daysBack, opts.gateID, opts.minRuns, opts.maxFailureRate, opts.minAgeDays, opts.requireClean24h, opts.outDir, opts.dryRun,
		)
	}

	token := requireEnv("CIRCLE_API_TOKEN")
	if err := ensureDirExists(opts.outDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create out dir: %v\n", err)
		os.Exit(1)
	}

	now := time.Now().UTC()
	since := now.AddDate(0, 0, -opts.daysBack)

	client := &http.Client{Timeout: 30 * time.Second}
	ctx := &apiCtx{client: client, token: token}

	dailyReports, err := collectReports(ctx, opts.org, opts.repo, opts.branch, opts.workflowName, opts.reportJobName, since, opts.verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "collection failed: %v\n", err)
		os.Exit(1)
	}

	agg := aggregate(dailyReports)

	logDailyReportSummary(dailyReports, opts.verbose)

	// Load acceptance-tests.yaml
	yamlPath := filepath.Join("op-acceptance-tests", "acceptance-tests.yaml")
	cfg, err := readAcceptanceYAML(yamlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed reading %s: %v\n", yamlPath, err)
		os.Exit(1)
	}

	// Build indices for flake-shake tests and target gates
	flakeTests, flakeGate, gateIndex := buildFlakeTests(&cfg, opts.gateID, yamlPath)
	_ = gateIndex

	// Select promotion candidates
	candidates, reasons := selectPromotionCandidates(agg, flakeTests, opts.minRuns, opts.maxFailureRate, opts.requireClean24h, opts.minAgeDays, now)

	// Write outputs
	if err := writeJSON(filepath.Join(opts.outDir, "aggregate.json"), agg); err != nil {
		fmt.Fprintf(os.Stderr, "failed writing aggregate: %v\n", err)
		os.Exit(1)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Package == candidates[j].Package {
			return candidates[i].TestName < candidates[j].TestName
		}
		return candidates[i].Package < candidates[j].Package
	})
	if err := writeJSON(filepath.Join(opts.outDir, "promotion-ready.json"), map[string]interface{}{"candidates": candidates, "skipped": reasons}); err != nil {
		fmt.Fprintf(os.Stderr, "failed writing promotion-ready: %v\n", err)
		os.Exit(1)
	}

	if opts.verbose {
		fmt.Printf("Promotion candidates: %d\n", len(candidates))
		for _, c := range candidates {
			name := c.TestName
			if strings.TrimSpace(name) == "" {
				name = "(package)"
			}
			fmt.Printf("  - %s %s (runs=%d pass=%.2f%%)\n", c.Package, name, c.TotalRuns, c.PassRate)
		}
	}

	// Write metadata for downstream consumers (e.g., Slack)
	meta := map[string]interface{}{
		"date":             now.Format("2006-01-02"),
		"candidates":       len(candidates),
		"flake_gate_tests": len(flakeGate.Tests),
	}
	if err := writeJSON(filepath.Join(opts.outDir, "metadata.json"), meta); err != nil {
		fmt.Fprintf(os.Stderr, "failed writing metadata: %v\n", err)
		os.Exit(1)
	}

	// Generate updated YAML (proposal)
	updated := computeUpdatedConfig(cfg, opts.gateID, candidates)

	// Write proposed YAML
	outYAML := filepath.Join(opts.outDir, "promotion.yaml")
	if err := writeYAML(outYAML, &updated); err != nil {
		fmt.Fprintf(os.Stderr, "failed writing promotion.yaml: %v\n", err)
		os.Exit(1)
	}

	// Print short summary
	if len(candidates) == 0 {
		reason := buildNoCandidatesSummary(agg, flakeTests, opts.minAgeDays, opts.requireClean24h)
		_ = os.WriteFile(filepath.Join(opts.outDir, "SUMMARY.txt"), []byte(reason+"\n"), 0o644)
		logger.Println(reason)
		return
	}
	var b bytes.Buffer
	b.WriteString("Promotion candidates (dry-run):\n")
	for _, c := range candidates {
		b.WriteString(fmt.Sprintf("- %s %s (runs=%d, pass=%.2f%%)\n", c.Package, c.TestName, c.TotalRuns, c.PassRate))
	}
	_ = os.WriteFile(filepath.Join(opts.outDir, "SUMMARY.txt"), b.Bytes(), 0o644)
	logger.Print(b.String())

	if opts.dryRun {
		logger.Println("Dry-run enabled; skipping branch creation, file update, and PR creation.")
		return
	}

	// Prepare updated YAML content for PR by editing only the flake-shake gate in-place to preserve comments
	var updatedYAMLBytes []byte

	prBranch := fmt.Sprintf("%s%s", flakeShakePRBranchPrefix, time.Now().UTC().Format("2006-01-02-150405"))

	// Prepare commit message and PR body
	title := flakeShakePRTitle
	var body bytes.Buffer
	body.WriteString("## 🤖 Automated Flake-Shake Test Promotion\n\n")

	// Attempt to resolve the CircleCI report job web URL for artifacts page
	reportArtifactsURL := resolveReportArtifactsURL(opts, ctx)

	body.WriteString(fmt.Sprintf("Promoting %d test(s) from gate `"+opts.gateID+"` based on stability criteria.\n\n", len(candidates)))
	if reportArtifactsURL != "" {
		body.WriteString(fmt.Sprintf("Artifacts: %s\n\n", reportArtifactsURL))
	}
	body.WriteString("### Tests Being Promoted\n\n")
	body.WriteString("| Test | Package | Total Runs | Pass Rate |\n|---|---|---:|---:|\n")
	for _, c := range candidates {
		name := c.TestName
		if strings.TrimSpace(name) == "" {
			name = "(package)"
		}
		body.WriteString(fmt.Sprintf("| %s | %s | %d | %.2f%% |\n", name, c.Package, c.TotalRuns, c.PassRate))
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

	if opts.verbose {
		logger.Printf("PR: starting creation process (base_branch=%s candidates=%d)", opts.branch, len(candidates))
	}

	// 1) Get base branch ref
	baseRef, _, err := ghc.Git.GetRef(ghCtx, opts.org, opts.repo, "refs/heads/"+opts.branch)
	if err != nil || baseRef.Object == nil || baseRef.Object.SHA == nil {
		fmt.Fprintf(os.Stderr, "failed to get base ref: %v\n", err)
		os.Exit(1)
	}
	if opts.verbose {
		logger.Printf("PR: base ref resolved sha=%s", baseRef.GetObject().GetSHA())
	}

	// 2) Create new branch ref
	newRef := &github.Reference{
		Ref:    github.String("refs/heads/" + prBranch),
		Object: &github.GitObject{SHA: baseRef.Object.SHA},
	}
	if _, _, err := ghc.Git.CreateRef(ghCtx, opts.org, opts.repo, newRef); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create ref: %v\n", err)
		os.Exit(1)
	}
	if opts.verbose {
		logger.Printf("PR: created branch %s", prBranch)
	}

	// 3) Read current file to fetch SHA (if exists) on base branch
	path := yamlPath
	var sha *string
	var originalYAML []byte
	if fileContent, _, resp, err := ghc.Repositories.GetContents(ghCtx, opts.org, opts.repo, path, &github.RepositoryContentGetOptions{Ref: opts.branch}); err == nil && fileContent != nil {
		sha = fileContent.SHA
		// Retrieve decoded file content via client helper
		rawContent, gcErr := fileContent.GetContent()
		if gcErr == nil && rawContent != "" {
			originalYAML = []byte(rawContent)
		}
	} else if resp != nil && resp.StatusCode == 404 {
		sha = nil
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get contents: %v\n", err)
		os.Exit(1)
	}

	// Build updated YAML by removing promoted tests only from flake-shake gate, preserving comments
	promoteKeys := map[string]promoteCandidate{}
	for _, c := range candidates {
		promoteKeys[keyFor(c.Package, c.TestName)] = c
	}
	updatedYAMLBytes, err = updateFlakeShakeGateOnly(originalYAML, opts.gateID, promoteKeys)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to update YAML: %v\n", err)
		os.Exit(1)
	}

	// 4) Update file in new branch
	commitMsg := title
	if _, _, err = ghc.Repositories.UpdateFile(ghCtx, opts.org, opts.repo, path, &github.RepositoryContentFileOptions{
		Message: github.String(commitMsg),
		Content: updatedYAMLBytes,
		Branch:  github.String(prBranch),
		SHA:     sha,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to update file: %v\n", err)
		os.Exit(1)
	}
	if opts.verbose {
		logger.Printf("PR: updated file %s on branch %s", path, prBranch)
	}

	// 5) Create PR
	prReq := &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(prBranch),
		Base:  github.String(opts.branch),
		Body:  github.String(body.String()),
	}
	pr, _, err := ghc.PullRequests.Create(ghCtx, opts.org, opts.repo, prReq)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create PR: %v\n", err)
		os.Exit(1)
	}
	logger.Printf("PR created: %s (number=%d)", pr.GetHTMLURL(), pr.GetNumber())

	// Update metadata with PR details for downstream Slack notification
	meta["pr_url"] = pr.GetHTMLURL()
	meta["pr_number"] = pr.GetNumber()
	if err := writeJSON(filepath.Join(opts.outDir, "metadata.json"), meta); err != nil {
		fmt.Fprintf(os.Stderr, "failed updating metadata with PR info: %v\n", err)
	}

	// 6) Add labels
	if _, _, err := ghc.Issues.AddLabelsToIssue(ghCtx, opts.org, opts.repo, pr.GetNumber(), []string{flakeShakeLabel, "A-acceptance-tests"}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to add labels: %v\n", err)
	}

	// 7) Request reviewers (user and team slug)
	if _, _, err := ghc.PullRequests.RequestReviewers(ghCtx, opts.org, opts.repo, pr.GetNumber(), github.ReviewersRequest{
		Reviewers:     []string{"scharissis"},
		TeamReviewers: []string{"platforms-team"},
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to request reviewers: %v\n", err)
	}

	// 8) Close any older open flake-shake PRs by this bot as superseded
	if err := closeSupersededFlakeShakePRs(ghCtx, ghc, opts.org, opts.repo, pr, title, opts.verbose); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to close superseded PRs: %v\n", err)
	}
}

// promoterOpts holds command-line options for the promoter tool.
type promoterOpts struct {
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
}

func parsePromoterFlags() promoterOpts {
	var opts promoterOpts
	flag.StringVar(&opts.org, "org", "ethereum-optimism", "GitHub org")
	flag.StringVar(&opts.repo, "repo", "optimism", "GitHub repo")
	flag.StringVar(&opts.branch, "branch", "develop", "Branch to scan")
	flag.StringVar(&opts.workflowName, "workflow", flakeShakeWorkflowName, "Workflow name")
	flag.StringVar(&opts.reportJobName, "report-job", flakeShakeReportJobName, "Report job name")
	flag.IntVar(&opts.daysBack, "days", 3, "Number of days to aggregate")
	flag.StringVar(&opts.gateID, "gate", flakeShakeGateID, "Gate id in acceptance-tests.yaml")
	flag.IntVar(&opts.minRuns, "min-runs", 300, "Minimum total runs required")
	flag.Float64Var(&opts.maxFailureRate, "max-failure-rate", 0.01, "Maximum allowed failure rate")
	flag.IntVar(&opts.minAgeDays, "min-age-days", 2, "Minimum age in days in flake-shake")
	flag.StringVar(&opts.outDir, "out", "./promotion-output", "Output directory")
	flag.BoolVar(&opts.dryRun, "dry-run", true, "Do not modify repo or open PRs")
	flag.BoolVar(&opts.requireClean24h, "require-clean-24h", false, "Require no failures in the last 24 hours")
	flag.BoolVar(&opts.verbose, "verbose", false, "Enable verbose debug logging")
	flag.Parse()
	// Validate interdependent options early to avoid confusing outcomes later
	if opts.daysBack < opts.minAgeDays {
		fmt.Fprintf(os.Stderr, "invalid flags: --days (%d) must be >= --min-age-days (%d)\n", opts.daysBack, opts.minAgeDays)
		os.Exit(2)
	}
	if opts.requireClean24h && opts.daysBack < 2 {
		fmt.Fprintf(os.Stderr, "invalid flags: --days (%d) must be >= 2 when --require-clean-24h is set to ensure >24h coverage\n", opts.daysBack)
		os.Exit(2)
	}
	return opts
}

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		fmt.Fprintf(os.Stderr, "%s is not set\n", name)
		os.Exit(1)
	}
	return v
}

func ensureDirExists(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func logDailyReportSummary(dailyReports map[string]DailySummary, verbose bool) {
	if !verbose {
		return
	}
	logger.Printf("Collected %d day(s) of summaries.", len(dailyReports))
	totalTests := 0
	for date, ds := range dailyReports {
		n := len(ds.StableTests) + len(ds.UnstableTests)
		totalTests += n
		logger.Printf("  - %s: %d tests (stable=%d unstable=%d)", date, n, len(ds.StableTests), len(ds.UnstableTests))
	}
	logger.Printf("Total tests across days: %d", totalTests)
}

// buildFlakeTests returns a map of tests in the flake-shake gate and also returns
// the flake gate reference and a gate index map for potential future use.
func buildFlakeTests(cfg *acceptanceYAML, gateID, yamlPath string) (map[string]testInfo, *gateYAML, map[string]*gateYAML) {
	flakeGate := findGate(cfg, gateID)
	if flakeGate == nil {
		fmt.Fprintf(os.Stderr, "gate %s not found in %s\n", gateID, yamlPath)
		os.Exit(1)
	}
	gateIndex := map[string]*gateYAML{}
	for i := range cfg.Gates {
		gateIndex[cfg.Gates[i].ID] = &cfg.Gates[i]
	}
	flakeTests := map[string]testInfo{}
	for ti, t := range flakeGate.Tests {
		key := keyFor(t.Package, t.Name)
		// Prefer explicit YAML field owner; fallback to metadata.owner
		owner := t.Owner
		if owner == "" && t.Metadata != nil {
			if v, ok := t.Metadata["owner"]; ok {
				owner = fmt.Sprintf("%v", v)
			}
		}
		flakeTests[key] = testInfo{Timeout: t.Timeout, Name: t.Name, Meta: t.Metadata, Owner: owner, GateIndex: indexOfGate(cfg, gateID), TestIndex: ti}
	}
	return flakeTests, flakeGate, gateIndex
}

func selectPromotionCandidates(agg map[string]*aggStats, flakeTests map[string]testInfo, minRuns int, maxFailureRate float64, requireClean24h bool, minAgeDays int, now time.Time) ([]promoteCandidate, map[string]string) {
	candidates := []promoteCandidate{}
	reasons := map[string]string{}
	// Identify wildcard package entries (tests with empty name in the flake-shake gate)
	wildcardPkgs := map[string]testInfo{}
	for k, info := range flakeTests {
		if strings.TrimSpace(info.Name) == "" && strings.HasSuffix(k, "::") {
			pkg := strings.TrimSuffix(k, "::")
			if pkg != "" {
				wildcardPkgs[pkg] = info
			}
		}
	}

	// Produce package-level candidates for wildcard entries by aggregating all tests in the package
	for pkg, info := range wildcardPkgs {
		totalRuns := 0
		totalPasses := 0
		totalFailures := 0
		earliest := ""
		var lastFailureAt *time.Time
		for _, s := range agg {
			if s.Package != pkg {
				continue
			}
			totalRuns += s.TotalRuns
			totalPasses += s.Passes
			totalFailures += s.Failures
			if earliest == "" || (s.FirstSeenDay != "" && s.FirstSeenDay < earliest) {
				earliest = s.FirstSeenDay
			}
			if s.LastFailureAt != nil {
				if lastFailureAt == nil || s.LastFailureAt.After(*lastFailureAt) {
					lastFailureAt = s.LastFailureAt
				}
			}
		}
		if totalRuns == 0 {
			reasons[keyFor(pkg, "")] = "no runs observed for package"
			continue
		}
		if totalRuns < minRuns {
			reasons[keyFor(pkg, "")] = fmt.Sprintf("insufficient runs: %d < %d (pkg)", totalRuns, minRuns)
			continue
		}
		failureRate := 0.0
		if totalRuns > 0 {
			failureRate = float64(totalFailures) / float64(totalRuns)
		}
		if !requireClean24h {
			if failureRate > maxFailureRate {
				reasons[keyFor(pkg, "")] = fmt.Sprintf("failure rate %.4f exceeds max %.4f (pkg)", failureRate, maxFailureRate)
				continue
			}
		}
		if requireClean24h && lastFailureAt != nil {
			if time.Since(*lastFailureAt) < 24*time.Hour {
				reasons[keyFor(pkg, "")] = "failure within last 24h (pkg)"
				continue
			}
		}
		if earliest == "" {
			reasons[keyFor(pkg, "")] = "no age information (pkg)"
			continue
		}
		firstDay, _ := time.Parse("2006-01-02", earliest)
		daysInGate := int(now.Sub(firstDay).Hours()/24) + 1
		if daysInGate < minAgeDays {
			reasons[keyFor(pkg, "")] = fmt.Sprintf("min age %dd not met (have %dd) (pkg)", minAgeDays, daysInGate)
			continue
		}
		passRate := 0.0
		if totalRuns > 0 {
			passRate = float64(totalPasses) / float64(totalRuns)
		}
		owner := info.Owner
		if owner == "" {
			if info.Meta != nil {
				if v, ok := info.Meta["owner"]; ok {
					owner = fmt.Sprintf("%v", v)
				}
			}
		}
		candidates = append(candidates, promoteCandidate{
			Package:      pkg,
			TestName:     "",
			TotalRuns:    totalRuns,
			PassRate:     passRate * 100.0,
			Timeout:      info.Timeout,
			FirstSeenDay: earliest,
			Owner:        owner,
		})
	}
	for key, s := range agg {
		// Skip per-test candidates for any package that is handled via wildcard aggregation
		if _, hasWildcard := wildcardPkgs[s.Package]; hasWildcard {
			continue
		}
		info, ok := flakeTests[key]
		if !ok {
			// Support wildcard package entries in the flake-shake gate where name is omitted.
			// Treat a gate entry with empty name as a wildcard that matches all tests in that package.
			if wi, wok := flakeTests[keyFor(s.Package, "")]; wok {
				info = wi
			} else {
				continue
			}
		}
		if s.TotalRuns < minRuns {
			reasons[key] = fmt.Sprintf("insufficient runs: %d < %d", s.TotalRuns, minRuns)
			continue
		}
		failureRate := 0.0
		if s.TotalRuns > 0 {
			failureRate = float64(s.Failures) / float64(s.TotalRuns)
		}
		if !requireClean24h {
			if failureRate > maxFailureRate {
				reasons[key] = fmt.Sprintf("failure rate %.4f exceeds max %.4f", failureRate, maxFailureRate)
				continue
			}
		}
		if requireClean24h && s.LastFailureAt != nil {
			if time.Since(*s.LastFailureAt) < 24*time.Hour {
				reasons[key] = "failure within last 24h"
				continue
			}
		}
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
		owner := info.Owner
		if owner == "" {
			if info.Meta != nil {
				if v, ok := info.Meta["owner"]; ok {
					owner = fmt.Sprintf("%v", v)
				}
			}
		}
		candidates = append(candidates, promoteCandidate{
			Package:      s.Package,
			TestName:     s.TestName,
			TotalRuns:    s.TotalRuns,
			PassRate:     passRate * 100.0,
			Timeout:      info.Timeout,
			FirstSeenDay: s.FirstSeenDay,
			Owner:        owner,
		})
	}
	return candidates, reasons
}

func computeUpdatedConfig(cfg acceptanceYAML, gateID string, candidates []promoteCandidate) acceptanceYAML {
	updated := cfg
	flakeIdx := indexOfGate(&updated, gateID)
	if flakeIdx < 0 {
		fmt.Fprintf(os.Stderr, "gate %s not found when updating\n", gateID)
		os.Exit(1)
	}
	promoteKeys := map[string]promoteCandidate{}
	for _, c := range candidates {
		promoteKeys[keyFor(c.Package, c.TestName)] = c
	}
	newFlakeTests := make([]testEntry, 0, len(updated.Gates[flakeIdx].Tests))
	for _, t := range updated.Gates[flakeIdx].Tests {
		k := keyFor(t.Package, t.Name)
		if _, ok := promoteKeys[k]; !ok {
			newFlakeTests = append(newFlakeTests, t)
		}
	}
	updated.Gates[flakeIdx].Tests = newFlakeTests
	return updated
}

func buildNoCandidatesSummary(agg map[string]*aggStats, flakeTests map[string]testInfo, minAgeDays int, requireClean24h bool) string {
	earliest := ""
	totalRuns := 0
	totalPass := 0
	totalFail := 0
	daySet := map[string]struct{}{}
	for key, s := range agg {
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
	return fmt.Sprintf(
		"No promotion candidates. Reason: min_age_days=%d; earliest_observation=%s; days_observed=%d; require_clean_24h=%t; total_runs=%d; passes=%d; failures=%d.",
		minAgeDays, earliest, daysObserved, requireClean24h, totalRuns, totalPass, totalFail,
	)
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

// collectReports scans CircleCI pipelines for the given GitHub repo/branch,
// locates the specified workflow and report job, and downloads/merges the
// daily-summary.json artifacts into a map keyed by date (YYYY-MM-DD).
// Only runs created at or after 'since' are considered. When multiple
// summaries exist for the same day, totals are summed and test lists merged.
func collectReports(ctx *apiCtx, org, repo, branch, workflowName, reportJobName string, since time.Time, verbose bool) (map[string]DailySummary, error) {
	dailyByDay := map[string]DailySummary{}

	basePipelines := fmt.Sprintf("https://circleci.com/api/v2/project/gh/%s/%s/pipeline?branch=%s", url.PathEscape(org), url.PathEscape(repo), url.QueryEscape(branch))
	pageURL := basePipelines

	for {
		pl, nextToken, err := getPipelinesPage(ctx, pageURL)
		if err != nil {
			return nil, err
		}
		if verbose {
			logger.Printf("Scanning pipelines page: %s", pageURL)
		}

		stop, err := processPipelines(ctx, pl, org, repo, workflowName, reportJobName, since, verbose, dailyByDay)
		if err != nil {
			return nil, err
		}
		if stop {
			break
		}
		if nextToken == "" {
			break
		}
		pageURL = basePipelines + "&page-token=" + url.QueryEscape(nextToken)
	}
	return dailyByDay, nil
}

// getPipelinesPage fetches a page of pipelines and returns the list along with the next page token.
func getPipelinesPage(ctx *apiCtx, pageURL string) (pipelineList, string, error) {
	var pl pipelineList
	if err := ctx.getJSON(pageURL, &pl); err != nil {
		return pipelineList{}, "", err
	}
	return pl, pl.NextPageToken, nil
}

// processPipelines iterates pipelines, filters by date/window, and merges daily summaries.
// It returns stop=true when it encounters pipelines older than the provided 'since' time.
func processPipelines(ctx *apiCtx, pl pipelineList, org, repo, workflowName, reportJobName string, since time.Time, verbose bool, dailyByDay map[string]DailySummary) (bool, error) {
	for _, p := range pl.Items {
		if verbose {
			logger.Printf("  pipeline %s created_at=%s", p.ID, p.CreatedAt.Format(time.RFC3339))
		}
		if p.CreatedAt.Before(since) {
			return true, nil
		}

		wfl, err := listWorkflows(ctx, p.ID)
		if err != nil {
			return false, err
		}
		for _, w := range wfl.Items {
			if w.Name != workflowName {
				continue
			}
			jl, err := listJobs(ctx, w.ID)
			if err != nil {
				return false, err
			}
			for _, j := range jl.Items {
				if j.Name != reportJobName {
					continue
				}
				al, err := listArtifacts(ctx, org, repo, j.JobNumber, verbose)
				if err != nil {
					return false, err
				}
				dailyURL := findDailySummaryArtifactURL(al)
				if dailyURL == "" {
					continue
				}
				if err := loadAndMergeDailySummary(ctx, dailyURL, dailyByDay, verbose); err != nil {
					return false, err
				}
			}
		}
	}
	return false, nil
}

func listWorkflows(ctx *apiCtx, pipelineID string) (workflowList, error) {
	wfURL := fmt.Sprintf("https://circleci.com/api/v2/pipeline/%s/workflow", pipelineID)
	var wfl workflowList
	if err := ctx.getJSON(wfURL, &wfl); err != nil {
		return workflowList{}, err
	}
	return wfl, nil
}

func listJobs(ctx *apiCtx, workflowID string) (jobList, error) {
	jobsURL := fmt.Sprintf("https://circleci.com/api/v2/workflow/%s/job", workflowID)
	var jl jobList
	if err := ctx.getJSON(jobsURL, &jl); err != nil {
		return jobList{}, err
	}
	return jl, nil
}

// closeSupersededFlakeShakePRs finds any open flake-shake promotion PRs created by the bot
// and closes them with a comment pointing to the newly created PR.
func closeSupersededFlakeShakePRs(ctx context.Context, ghc *github.Client, org, repo string, newPR *github.PullRequest, title string, verbose bool) error {
	// Search open PRs in this repo that match our title and bot author
	// Using Issues.ListByRepo with filters
	opt := &github.IssueListByRepoOptions{
		State:       "open",
		Labels:      []string{flakeShakeLabel},
		Since:       time.Now().AddDate(0, 0, -flakeShakeSupersedeDays),
		ListOptions: github.ListOptions{PerPage: 50},
	}
	for {
		issues, resp, err := ghc.Issues.ListByRepo(ctx, org, repo, opt)
		if err != nil {
			return err
		}
		for _, is := range issues {
			if is.IsPullRequest() && is.GetNumber() != newPR.GetNumber() {
				// Check title contains our flake-shake marker; be robust to minor variations
				// Use the provided title to derive a stable prefix (before the first ';') for matching
				titlePrefix := strings.TrimSpace(strings.TrimSuffix(title, "; test promotions"))
				if strings.Contains(strings.ToLower(is.GetTitle()), strings.ToLower(flakeShakeGateID)) && strings.Contains(is.GetTitle(), titlePrefix) {
					// Author check
					if is.User != nil && is.User.GetLogin() != flakeShakeBotAuthor {
						continue
					}
					// Comment and close
					msg := fmt.Sprintf("superseded by #%d", newPR.GetNumber())
					_, _, _ = ghc.Issues.CreateComment(ctx, org, repo, is.GetNumber(), &github.IssueComment{Body: github.String(msg)})
					state := "closed"
					_, _, _ = ghc.PullRequests.Edit(ctx, org, repo, is.GetNumber(), &github.PullRequest{State: &state})
					if verbose {
						logger.Printf("Closed superseded PR #%d: %s", is.GetNumber(), is.GetTitle())
					}
				}
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return nil
}

// resolveReportArtifactsURL attempts to find the web URL to the report job's artifacts page
// by scanning recent pipelines/workflows for the configured workflow/report job names.
// Returns an empty string if not found.
func resolveReportArtifactsURL(opts promoterOpts, ctx *apiCtx) string {
	// Scan the latest pipelines on the given branch; reuse collectReports traversal but short-circuit on first match
	basePipelines := fmt.Sprintf("https://circleci.com/api/v2/project/gh/%s/%s/pipeline?branch=%s", url.PathEscape(opts.org), url.PathEscape(opts.repo), url.QueryEscape(opts.branch))
	pageURL := basePipelines
	now := time.Now().UTC()
	since := now.AddDate(0, 0, -opts.daysBack)
	for {
		pl, nextToken, err := getPipelinesPage(ctx, pageURL)
		if err != nil {
			return ""
		}
		for _, p := range pl.Items {
			if p.CreatedAt.Before(since) {
				return ""
			}
			wfl, err := listWorkflows(ctx, p.ID)
			if err != nil {
				return ""
			}
			for _, w := range wfl.Items {
				if w.Name != opts.workflowName {
					continue
				}
				jl, err := listJobs(ctx, w.ID)
				if err != nil {
					return ""
				}
				for _, j := range jl.Items {
					if j.Name != opts.reportJobName {
						continue
					}
					// Prefer constructing the app.circleci.com artifacts URL from pipeline number + workflow id + job number
					if p.Number != 0 && j.JobNumber != 0 {
						u := fmt.Sprintf("https://app.circleci.com/pipelines/github/%s/%s/%d/workflows/%s/jobs/%d/artifacts", opts.org, opts.repo, p.Number, w.ID, j.JobNumber)
						return u
					}
					return ""
				}
			}
		}
		if nextToken == "" {
			break
		}
		pageURL = basePipelines + "&page-token=" + url.QueryEscape(nextToken)
	}
	return ""
}

func listArtifacts(ctx *apiCtx, org, repo string, jobNumber int, verbose bool) (artifactsList, error) {
	artsURL := fmt.Sprintf("https://circleci.com/api/v2/project/gh/%s/%s/%d/artifacts", url.PathEscape(org), url.PathEscape(repo), jobNumber)
	var al artifactsList
	if err := ctx.getJSON(artsURL, &al); err != nil {
		return artifactsList{}, err
	}
	if verbose {
		logger.Printf("    job %d artifacts: %d", jobNumber, len(al.Items))
		for _, a := range al.Items {
			logger.Printf("      - %s", a.Path)
		}
	}
	return al, nil
}

func findDailySummaryArtifactURL(al artifactsList) string {
	for _, a := range al.Items {
		// Accept any artifact path that ends with the filename, regardless of destination prefix
		if strings.HasSuffix(a.Path, "daily-summary.json") {
			return a.URL
		}
	}
	return ""
}

func loadAndMergeDailySummary(ctx *apiCtx, dailyURL string, dailyByDay map[string]DailySummary, verbose bool) error {
	data, err := ctx.getBytes(dailyURL)
	if err != nil {
		return err
	}
	var ds DailySummary
	if json.Unmarshal(data, &ds) != nil || ds.Date == "" {
		return nil
	}
	if prev, seen := dailyByDay[ds.Date]; !seen {
		dailyByDay[ds.Date] = ds
		if verbose {
			logger.Printf("    loaded daily summary for %s (runs=%d iterations=%d)", ds.Date, ds.TotalRuns, ds.Iterations)
		}
		return nil
	} else {
		merged := prev
		merged.TotalRuns += ds.TotalRuns
		merged.Iterations += ds.Iterations
		merged.StableTests = append(merged.StableTests, ds.StableTests...)
		merged.UnstableTests = append(merged.UnstableTests, ds.UnstableTests...)
		dailyByDay[ds.Date] = merged
		if verbose {
			logger.Printf("    merged another run for %s (+runs=%d +iters=%d) now runs=%d iters=%d", ds.Date, ds.TotalRuns, ds.Iterations, merged.TotalRuns, merged.Iterations)
		}
		return nil
	}
}

// aggregate reduces per-day test summaries into a single map keyed by test,
// summing runs/passes/failures and tracking which days each test appeared.
// It also computes first/last seen day boundaries for each test.
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

// ensureAgg returns the aggregated stats bucket for the given test key, creating it
// if it does not exist. It also records the provided day in DaysObserved (without
// duplicates) and updates FirstSeenDay/LastSeenDay bounds accordingly.
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

// parseDayEnd returns the exclusive end-of-day bound for the given date
// (YYYY-MM-DD) in UTC. This is the start of the next day, suitable for
// half-open intervals: [start, end).
func parseDayEnd(day string) time.Time {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return time.Now().UTC()
	}
	return t.UTC().Add(24 * time.Hour)
}

func keyFor(pkg, name string) string {
	return pkg + "::" + strings.TrimSpace(name)
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
	for i, gate := range acc.Gates {
		if gate.ID == id {
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

// updateFlakeShakeGateOnly updates only the flake-shake gate tests list in the original YAML bytes,
// preserving all comments and formatting elsewhere. It removes any test entries matching promoteKeys.
func updateFlakeShakeGateOnly(original []byte, gateID string, promoteKeys map[string]promoteCandidate) ([]byte, error) {
	if len(original) == 0 {
		// Fallback to structured marshal if original missing (should not happen in CI)
		return yaml.Marshal(nil)
	}
	lines := strings.Split(string(original), "\n")
	var out []string
	inGates := false
	inFlake := false
	indentGate := ""
	indentTests := ""
	// simple state machine: copy all lines except tests under flake-shake that match promoteKeys
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		// Detect top-level 'gates:'
		if strings.HasPrefix(trimmed, "gates:") {
			inGates = true
			out = append(out, line)
			continue
		}
		if inGates && strings.HasPrefix(trimmed, "- id:") {
			// Entering a gate block
			// Determine indentation
			indentGate = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			// Gate id value
			id := strings.TrimSpace(strings.TrimPrefix(trimmed, "- id:"))
			id = strings.Trim(id, "\"')")
			inFlake = (id == gateID)
			out = append(out, line)
			continue
		}
		if !inFlake {
			out = append(out, line)
			continue
		}
		// Within flake-shake gate only
		if strings.HasPrefix(strings.TrimSpace(line), "tests:") && indentTests == "" {
			// Capture tests indent from next line if present
			out = append(out, line)
			// From here, filter list items until we leave tests list (deduce by indentation decrease or new key at gate level)
			pos := i + 1
			for ; pos < len(lines); pos++ {
				cur := lines[pos]
				curTrim := strings.TrimSpace(cur)
				if curTrim == "" {
					out = append(out, cur)
					continue
				}
				// end of this gate block if next key aligns to indentGate and not list item
				if !strings.HasPrefix(cur, indentGate+"  ") || strings.HasPrefix(strings.TrimSpace(cur), "- id:") {
					// set i so that outer loop reprocesses this line next
					i = pos - 1
					break
				}
				// If this is a list item under tests: starts with indentGate + two spaces + two more (tests indent) + '- '
				// We can't rely on exact spaces; detect package line to gather block
				if strings.HasPrefix(strings.TrimSpace(cur), "- package:") {
					// start of a test block; buffer until next '- package:' or gate-level boundary
					block := []string{cur}
					pkg := strings.TrimSpace(strings.TrimPrefix(curTrim, "- package:"))
					name := ""
					j := pos + 1
					for ; j < len(lines); j++ {
						nt := strings.TrimSpace(lines[j])
						if nt == "" {
							block = append(block, lines[j])
							continue
						}
						// next test or end of tests
						if strings.HasPrefix(nt, "- package:") || (!strings.HasPrefix(lines[j], indentGate+"  ")) {
							j--
							break
						}
						block = append(block, lines[j])
						if strings.HasPrefix(nt, "name:") {
							name = strings.TrimSpace(strings.TrimPrefix(nt, "name:"))
						}
					}
					// Decide keep or drop
					k := keyFor(pkg, name)
					if _, toPromote := promoteKeys[k]; !toPromote {
						out = append(out, block...)
					}
					pos = j
					continue
				}
				// Non-test line under tests; keep
				out = append(out, cur)
			}
			continue
		}
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n")), nil
}
