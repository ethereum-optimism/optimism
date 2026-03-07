package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters"
	goAdapter "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters/golang"
	rustAdapter "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters/rust"
	solAdapter "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters/sol"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/engine"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/platform/circleci"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/state"
)

func main() {
	configDir := flag.String("config", "ops/shadow-ci/config", "Path to config directory")
	repoRoot := flag.String("repo", ".", "Repository root")
	base := flag.String("base", "origin/develop", "Base ref for diff")
	head := flag.String("head", "HEAD", "Head ref for diff")
	branch := flag.String("branch", os.Getenv("CIRCLE_BRANCH"), "Current branch")
	schedule := flag.Bool("schedule", false, "Whether this is a scheduled run")
	output := flag.String("output", "/tmp/shadow-ci/affected.json", "Affected output path")
	decisionOutput := flag.String("decision-output", "/tmp/shadow-ci/decision.json", "Pipeline decision output path")
	eventsDir := flag.String("events-dir", "/tmp/shadow-ci/events", "Events directory")
	pipelineID := flag.String("pipeline-id", os.Getenv("CIRCLE_PIPELINE_ID"), "Pipeline ID")
	flakeDBPath := flag.String("flake-db", "", "Path to flake database JSON (optional)")
	continueMode := flag.Bool("continue", false, "Generate continuation YAML for CircleCI dynamic config")
	continuationOutput := flag.String("continuation-output", "/tmp/shadow-ci-continuation.yml", "Continuation YAML output path")
	flag.Parse()

	cfg, err := model.LoadConfig(*configDir)
	if err != nil {
		fatal("loading config: %v", err)
	}

	changedFiles, err := getChangedFiles(*repoRoot, *base, *head)
	if err != nil {
		fatal("getting changed files: %v", err)
	}

	fmt.Printf("Changed files: %d\n", len(changedFiles))
	for _, f := range changedFiles {
		fmt.Printf("  %s\n", f)
	}

	registry := buildRegistry(cfg)
	store := events.NewLocalStore(*eventsDir)
	emitter := events.NewEmitter(store, *pipelineID, 0, *head, *branch)

	// Phase 1: compute affected targets per language (graph-based).
	computer := engine.NewAffectedComputer(registry, cfg.Scoping, *repoRoot)
	affected, err := computer.Compute(changedFiles, emitter)
	if err != nil {
		fatal("computing affected: %v", err)
	}

	for lang, lr := range affected.ByLanguage {
		fmt.Printf("[%s] selected=%d total=%d skip=%.1f%% always_run=%d\n",
			lang, lr.SelectedTargets, lr.TotalTargets, lr.SkipRate*100, lr.AlwaysRunCount)
	}

	// Load flake database via state store (or direct path as fallback).
	stateStore, err := state.FromConfig(cfg.Platform.State, *branch)
	if err != nil {
		fatal("creating state store: %v", err)
	}

	var flakeDB *model.FlakeDB
	if *flakeDBPath != "" {
		flakeDB, err = model.LoadFlakeDB(*flakeDBPath)
		if err != nil {
			fatal("loading flake db: %v", err)
		}
	} else {
		flakeDB, err = model.LoadFlakeDBFromStore(stateStore, "flake-db")
		if err != nil {
			fatal("loading flake db from state store: %v", err)
		}
	}

	// Phase 2: produce pipeline decision covering ALL job categories.
	de := engine.NewDecisionEngine(cfg.Scoping, cfg.Placement, flakeDB, affected, emitter)
	decision := de.Decide(changedFiles, *branch, *schedule)

	// Print decision summary.
	fmt.Printf("\n=== Pipeline Decision (stage: %s) ===\n", decision.Stage)
	needed, skipped := 0, 0
	for name, cd := range decision.Categories {
		if cd.Needed {
			needed++
			detail := cd.Reason
			if len(cd.Targets) > 0 {
				detail += fmt.Sprintf(" (%d targets)", len(cd.Targets))
			}
			if len(cd.Packages) > 0 {
				detail += fmt.Sprintf(" [%s]", strings.Join(cd.Packages, ", "))
			}
			fmt.Printf("  RUN  %-30s %s\n", name, detail)
		} else if cd.Skipped {
			skipped++
			fmt.Printf("  SKIP %-30s %s\n", name, cd.SkipWhy)
		}
	}
	fmt.Printf("\nTotal: %d run, %d skip, %d categories\n", needed, skipped, len(decision.Categories))

	// Write outputs.
	os.MkdirAll(filepath.Dir(*output), 0o755)
	os.MkdirAll(filepath.Dir(*decisionOutput), 0o755)

	writeJSON(*output, affected)
	writeJSON(*decisionOutput, decision)

	fmt.Printf("\nWrote affected targets to %s\n", *output)
	fmt.Printf("Wrote pipeline decision to %s\n", *decisionOutput)

	// Continuation mode: render CircleCI YAML from decision + plan.
	if *continueMode {
		trigger := model.Trigger{
			Type:   string(decision.Stage),
			Branch: *branch,
			Base:   *base,
			Head:   *head,
		}
		planner := engine.NewPlanner()
		plan := planner.Plan(trigger, changedFiles, affected, emitter)

		renderer := circleci.NewRenderer(cfg.Platform.CircleCI.Runners)
		yamlData, err := renderer.RenderFromDecision(decision, plan)
		if err != nil {
			fatal("rendering continuation: %v", err)
		}

		os.MkdirAll(filepath.Dir(*continuationOutput), 0o755)
		if err := os.WriteFile(*continuationOutput, yamlData, 0o644); err != nil {
			fatal("writing continuation YAML: %v", err)
		}
		fmt.Printf("Rendered %d bytes of continuation YAML to %s\n", len(yamlData), *continuationOutput)
	}
}

func writeJSON(path string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal("marshaling: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fatal("writing %s: %v", path, err)
	}
}

func getChangedFiles(repoRoot, base, head string) ([]string, error) {
	if strings.HasPrefix(base, "-") || strings.HasPrefix(head, "-") {
		return nil, fmt.Errorf("invalid ref: must not start with '-'")
	}
	cmd := exec.Command("git", "diff", "--name-only", base+"..."+head)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("git", "diff", "--name-only", base, head)
		cmd.Dir = repoRoot
		out, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git diff: %w", err)
		}
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

func buildRegistry(cfg *model.Config) *adapters.Registry {
	registry := adapters.NewRegistry()

	if cfg.Adapters.Go != nil && cfg.Adapters.Go.Enabled {
		g := goAdapter.NewGraph(cfg.Adapters.Go.Root, cfg.Adapters.Go.SpecialPaths)
		r := goAdapter.NewRunner(cfg.Adapters.Go.Root)
		registry.Register("go", adapters.Adapter{Graph: g, Runner: r})
	}

	if cfg.Adapters.Sol != nil && cfg.Adapters.Sol.Enabled {
		g := solAdapter.NewGraph(
			cfg.Adapters.Sol.Root,
			cfg.Adapters.Sol.SourceDirs,
			cfg.Adapters.Sol.RemappingsFile,
			cfg.Adapters.Sol.SpecialPaths,
			cfg.Adapters.Sol.Features,
		)
		r := solAdapter.NewRunner(cfg.Adapters.Sol.Root)
		registry.Register("sol", adapters.Adapter{Graph: g, Runner: r})
	}

	if cfg.Adapters.Rust != nil && cfg.Adapters.Rust.Enabled {
		g := rustAdapter.NewGraph(cfg.Adapters.Rust.Root, cfg.Adapters.Rust.SpecialPaths)
		r := rustAdapter.NewRunner(cfg.Adapters.Rust.Root)
		registry.Register("rust", adapters.Adapter{Graph: g, Runner: r})
	}

	return registry
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
