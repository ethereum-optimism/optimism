package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/engine"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

func main() {
	affectedPath := flag.String("affected", "/tmp/shadow-ci-affected.json", "Path to affected targets JSON")
	output := flag.String("output", "/tmp/shadow-ci-plan.json", "Output path for test plan")
	triggerType := flag.String("trigger", "pr", "Trigger type (pr, push, nightly)")
	branch := flag.String("branch", os.Getenv("CIRCLE_BRANCH"), "Branch name")
	baseBranch := flag.String("base", "develop", "Base branch")
	head := flag.String("head", os.Getenv("CIRCLE_SHA1"), "Head commit")
	pr := flag.Int("pr", 0, "PR number")
	eventsDir := flag.String("events-dir", "/tmp/shadow-ci-events", "Events directory")
	pipelineID := flag.String("pipeline-id", os.Getenv("CIRCLE_PIPELINE_ID"), "Pipeline ID")
	flag.Parse()

	// Load affected result.
	data, err := os.ReadFile(*affectedPath)
	if err != nil {
		fatal("reading affected: %v", err)
	}

	var affected engine.AffectedResult
	if err := json.Unmarshal(data, &affected); err != nil {
		fatal("parsing affected: %v", err)
	}

	trigger := model.Trigger{
		Type:   *triggerType,
		PR:     *pr,
		Branch: *branch,
		Base:   *baseBranch,
		Head:   *head,
	}

	store := events.NewLocalStore(*eventsDir)
	emitter := events.NewEmitter(store, *pipelineID, *pr, *head, *branch)

	planner := engine.NewPlanner()
	plan := planner.Plan(trigger, nil, &affected, emitter)

	planData, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		fatal("marshaling plan: %v", err)
	}

	if err := os.WriteFile(*output, planData, 0o644); err != nil {
		fatal("writing plan: %v", err)
	}

	fmt.Printf("Plan %s: %d jobs\n", plan.ID, len(plan.Jobs))
	for _, job := range plan.Jobs {
		fmt.Printf("  [%s] %s: %d targets, %d configs, parallelism=%d\n",
			job.Language, job.Name, len(job.Targets), len(job.Configurations), job.Resources.Parallelism)
	}
	fmt.Printf("Wrote plan to %s\n", *output)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
