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
	eventsDir := flag.String("events-dir", "/tmp/shadow-ci/events", "Events directory")
	flakeDBPath := flag.String("flake-db", "/tmp/shadow-ci/flake-db.json", "Path to flake DB JSON")
	pipelineID := flag.String("pipeline-id", "", "Pipeline to evaluate")
	commit := flag.String("commit", "", "Culprit commit SHA")
	pr := flag.Int("pr", 0, "Culprit PR number")
	dryRun := flag.Bool("dry-run", true, "Evaluate but don't create revert PR")
	flag.Parse()

	// The failed tests would normally come from event store queries.
	// For now, accept them as remaining args.
	failedTests := flag.Args()

	db, err := model.LoadFlakeDB(*flakeDBPath)
	if err != nil {
		fatal("loading flake db: %v", err)
	}

	store := events.NewLocalStore(*eventsDir)
	emitter := events.NewEmitter(store, *pipelineID, 0, *commit, "develop")

	config := engine.DefaultAutoRevertConfig()
	config.DryRun = *dryRun

	// Configure notifier: Slack if webhook is set, otherwise log-only.
	var notifier engine.Notifier
	if webhookURL := os.Getenv("SLACK_WEBHOOK_URL"); webhookURL != "" {
		notifier = engine.NewSlackNotifier(webhookURL, os.Getenv("SLACK_CHANNEL"))
	} else {
		notifier = &engine.LogNotifier{}
	}

	reverter := engine.NewAutoReverter(store, db, emitter, config, notifier)

	decision := reverter.Evaluate(failedTests, *commit, *pr)

	data, _ := json.MarshalIndent(decision, "", "  ")
	fmt.Println(string(data))

	if decision.ShouldRevert && !*dryRun {
		fmt.Println("Auto-revert would create PR (not implemented yet)")
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
