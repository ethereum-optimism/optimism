package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/platform/circleci"
)

func main() {
	planPath := flag.String("plan", "/tmp/shadow-ci-plan.json", "Path to test plan JSON")
	configDir := flag.String("config", "ops/shadow-ci/config", "Path to config directory")
	output := flag.String("output", "/tmp/shadow-ci.yml", "Output path for CircleCI YAML")
	flag.Parse()

	planData, err := os.ReadFile(*planPath)
	if err != nil {
		fatal("reading plan: %v", err)
	}

	var plan model.TestPlan
	if err := json.Unmarshal(planData, &plan); err != nil {
		fatal("parsing plan: %v", err)
	}

	cfg, err := model.LoadConfig(*configDir)
	if err != nil {
		fatal("loading config: %v", err)
	}

	renderer := circleci.NewRenderer(cfg.Platform.CircleCI.Runners)
	yamlData, err := renderer.Render(plan)
	if err != nil {
		fatal("rendering: %v", err)
	}

	if err := os.WriteFile(*output, yamlData, 0o644); err != nil {
		fatal("writing output: %v", err)
	}

	fmt.Printf("Rendered %d bytes of CircleCI YAML to %s\n", len(yamlData), *output)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
