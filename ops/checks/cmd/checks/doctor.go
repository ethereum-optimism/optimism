package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/ops/checks/catalog"
	"github.com/ethereum-optimism/optimism/ops/checks/graph"
)

// cmdDoctor answers "is the checks engine healthy and configured?"
// in one shot. Each sub-check reports OK / WARN / ERROR with a
// short message. The command exits 0 if no ERROR-level problems
// were found (WARNs don't fail the check).
func cmdDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	graphPath := fs.String("graph", "ops/checks/graph.json", "path to graph file")
	catalogPath := fs.String("catalog", "ops/checks/checks.yaml", "path to checks catalog")
	coverageDir := fs.String("coverage-dir", "ops/checks/coverage-data", "coverage reports directory")
	policyPath := fs.String("policy", "", "optional policy override path")
	fs.Parse(args)

	root := findRepoRoot()

	results := []result{
		checkTool("go", "go", []string{"version"}),
		checkToolOptional("cargo", "cargo", []string{"--version"}, "Rust coverage / adapter disabled without cargo"),
		checkToolOptional("forge", "forge", []string{"--version"}, "Solidity coverage disabled without forge"),
		checkFile("go.mod", filepath.Join(root, "go.mod")),
		checkGraph(*graphPath, root),
		checkCatalog(*catalogPath, root),
		checkPolicy(*policyPath),
		checkCoverage(*coverageDir, root),
		checkCatalogCIJobNames(*catalogPath, root),
	}

	failed := 0
	for _, r := range results {
		fmt.Println(r.String())
		if r.status == statusError {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d diagnostic(s) failed", failed)
	}
	return nil
}

type status int

const (
	statusOK status = iota
	statusWarn
	statusError
)

type result struct {
	label   string
	status  status
	message string
}

func (r result) String() string {
	icon := "\u2713" // ✓
	switch r.status {
	case statusWarn:
		icon = "!"
	case statusError:
		icon = "\u2717" // ✗
	}
	if r.message == "" {
		return fmt.Sprintf("  %s %s", icon, r.label)
	}
	return fmt.Sprintf("  %s %s — %s", icon, r.label, r.message)
}

func checkTool(label, bin string, args []string) result {
	path, err := exec.LookPath(bin)
	if err != nil {
		return result{label, statusError, "not found in PATH"}
	}
	cmd := exec.Command(path, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return result{label, statusError, fmt.Sprintf("%s %s failed: %v", bin, strings.Join(args, " "), err)}
	}
	return result{label, statusOK, firstLine(string(out))}
}

func checkToolOptional(label, bin string, args []string, absentMessage string) result {
	if _, err := exec.LookPath(bin); err != nil {
		return result{label, statusWarn, absentMessage}
	}
	return checkTool(label, bin, args)
}

func checkFile(label, path string) result {
	info, err := os.Stat(path)
	if err != nil {
		return result{label, statusError, err.Error()}
	}
	return result{label, statusOK, fmt.Sprintf("%d bytes, modified %s ago", info.Size(), humanDuration(time.Since(info.ModTime())))}
}

func checkGraph(path, root string) result {
	absPath := path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(root, absPath)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return result{"graph", statusError, "not found — run `checks build`"}
	}
	g, err := graph.Load(absPath)
	if err != nil {
		return result{"graph", statusError, fmt.Sprintf("failed to load: %v", err)}
	}
	age := time.Since(info.ModTime())
	msg := fmt.Sprintf("%d nodes, %d edges, built %s ago", g.NodeCount(), g.EdgeCount(), humanDuration(age))
	if age > 7*24*time.Hour {
		return result{"graph", statusWarn, msg + " (consider rebuilding)"}
	}
	return result{"graph", statusOK, msg}
}

func checkCatalog(path, root string) result {
	absPath := path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(root, absPath)
	}
	cat, err := catalog.Load(absPath)
	if err != nil {
		return result{"catalog", statusError, fmt.Sprintf("failed to load: %v", err)}
	}
	if err := cat.Validate(); err != nil {
		return result{"catalog", statusError, fmt.Sprintf("validation: %v", err)}
	}
	return result{"catalog", statusOK, fmt.Sprintf("%d check types, %d profiles", len(cat.CheckTypes), len(cat.Profiles))}
}

func checkPolicy(explicit string) result {
	pol, err := loadPolicy(explicit)
	if err != nil {
		return result{"policy", statusError, err.Error()}
	}
	return result{"policy", statusOK, fmt.Sprintf("%d stages, %d tiers, %d knob-policies",
		len(pol.Stages), len(pol.Tiers), len(pol.KnobPolicies))}
}

func checkCoverage(dir, root string) result {
	absDir := dir
	if !filepath.IsAbs(absDir) {
		absDir = filepath.Join(root, absDir)
	}
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return result{"coverage", statusWarn, "no coverage data — run `checks coverage batch`"}
	}
	counts := map[string]int{"sol": 0, "go": 0, "rs": 0, "other": 0}
	var oldest time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		switch {
		case strings.HasPrefix(e.Name(), "test_") || strings.HasPrefix(e.Name(), "sol_"):
			counts["sol"]++
		case strings.HasPrefix(e.Name(), "go_"):
			counts["go"]++
		case strings.HasPrefix(e.Name(), "rs_"):
			counts["rs"]++
		default:
			counts["other"]++
		}
		if info, err := e.Info(); err == nil {
			if oldest.IsZero() || info.ModTime().Before(oldest) {
				oldest = info.ModTime()
			}
		}
	}
	total := counts["sol"] + counts["go"] + counts["rs"] + counts["other"]
	if total == 0 {
		return result{"coverage", statusWarn, "directory present but empty"}
	}
	ageMsg := ""
	if !oldest.IsZero() {
		ageMsg = fmt.Sprintf("; oldest %s old", humanDuration(time.Since(oldest)))
	}
	return result{"coverage", statusOK, fmt.Sprintf("%d reports (sol=%d, go=%d, rs=%d%s)",
		total, counts["sol"], counts["go"], counts["rs"], ageMsg)}
}

// checkCatalogCIJobNames reports how many catalog check types have
// CIJobNames populated. Without that, `ingest ci-history --source
// circleci` can't map jobs to checks.
func checkCatalogCIJobNames(catalogPath, root string) result {
	absPath := catalogPath
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(root, absPath)
	}
	cat, err := catalog.Load(absPath)
	if err != nil {
		return result{"catalog ci_job_names", statusError, err.Error()}
	}
	mapped, total := 0, len(cat.CheckTypes)
	for _, ct := range cat.CheckTypes {
		if len(ct.CIJobNames) > 0 {
			mapped++
		}
	}
	if mapped == 0 {
		return result{"catalog ci_job_names", statusWarn,
			fmt.Sprintf("0/%d checks mapped — CircleCI ingestion unavailable", total)}
	}
	if mapped < total {
		return result{"catalog ci_job_names", statusWarn,
			fmt.Sprintf("%d/%d checks mapped — remaining %d won't receive CI-history priors", mapped, total, total-mapped)}
	}
	return result{"catalog ci_job_names", statusOK,
		fmt.Sprintf("%d/%d checks mapped", mapped, total)}
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[:i]
	}
	return s
}

func humanDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "<1m"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

