package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters"
	goAdapter "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters/golang"
	rustAdapter "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters/rust"
	solAdapter "github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/adapters/sol"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/engine"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

func main() {
	configDir := flag.String("config", "ops/shadow-ci/config", "Path to config directory")
	repoRoot := flag.String("repo", ".", "Repository root")
	base := flag.String("base", "origin/develop", "Base ref for diff")
	head := flag.String("head", "HEAD", "Head ref for diff")
	output := flag.String("output", "/tmp/shadow-ci-affected.json", "Output path")
	eventsDir := flag.String("events-dir", "/tmp/shadow-ci-events", "Events directory")
	pipelineID := flag.String("pipeline-id", os.Getenv("CIRCLE_PIPELINE_ID"), "Pipeline ID")
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
	emitter := events.NewEmitter(store, *pipelineID, 0, *head, "")

	computer := engine.NewAffectedComputer(registry, cfg.Scoping, *repoRoot)
	result, err := computer.Compute(changedFiles, emitter)
	if err != nil {
		fatal("computing affected: %v", err)
	}

	for lang, lr := range result.ByLanguage {
		fmt.Printf("[%s] selected=%d total=%d skip=%.1f%% always_run=%d\n",
			lang, lr.SelectedTargets, lr.TotalTargets, lr.SkipRate*100, lr.AlwaysRunCount)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fatal("marshaling: %v", err)
	}

	if err := os.WriteFile(*output, data, 0o644); err != nil {
		fatal("writing output: %v", err)
	}

	fmt.Printf("Wrote affected targets to %s\n", *output)
}

func getChangedFiles(repoRoot, base, head string) ([]string, error) {
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
