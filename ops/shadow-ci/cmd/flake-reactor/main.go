package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/engine"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/state"
)

func main() {
	configDir := flag.String("config", "ops/shadow-ci/config", "Path to config directory")
	eventsDir := flag.String("events-dir", "/tmp/shadow-ci-events", "Events directory")
	flakeDBPath := flag.String("flake-db", "", "Path to flake DB JSON (overrides state store)")
	branch := flag.String("branch", os.Getenv("CIRCLE_BRANCH"), "Current branch")
	since := flag.Duration("since", 24*time.Hour, "Process events since this duration ago")
	flag.Parse()

	cfg, err := model.LoadConfig(*configDir)
	if err != nil {
		fatal("loading config: %v", err)
	}

	// Load flake DB from state store or direct path.
	var db *model.FlakeDB
	var stateStore state.Store
	if *flakeDBPath != "" {
		db, err = model.LoadFlakeDB(*flakeDBPath)
		if err != nil {
			fatal("loading flake db: %v", err)
		}
	} else {
		stateStore, err = state.FromConfig(cfg.Platform.State, *branch)
		if err != nil {
			fatal("creating state store: %v", err)
		}
		db, err = model.LoadFlakeDBFromStore(stateStore, "flake-db")
		if err != nil {
			fatal("loading flake db from state store: %v", err)
		}
	}

	store := events.NewLocalStore(*eventsDir)
	emitter := events.NewEmitter(store, "flake-reactor", 0, "", "")

	config := engine.DefaultFlakeReactorConfig()
	reactor := engine.NewFlakeReactor(db, store, emitter, config)

	now := time.Now()
	cutoff := now.Add(-*since)

	// Scan for flake events and process them.
	flakeEvents, err := store.Query(events.EventFilter{
		Types: []model.EventType{model.EventFlakeDetected},
		After: cutoff,
	})
	if err != nil {
		fatal("querying flake events: %v", err)
	}

	fmt.Printf("Processing %d flake events since %s\n", len(flakeEvents), cutoff.Format(time.RFC3339))

	for _, evt := range flakeEvents {
		var payload model.FlakePayload
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping event %s: %v\n", evt.ID, err)
			continue
		}
		testKey := payload.Result.Test.Key()
		reactor.ProcessFlake(testKey, payload.Result.Language, payload.Fingerprint, evt.Timestamp)
	}

	// Check for auto-recovery.
	reactor.CheckAutoRecovery(now)

	// Save updated flake DB.
	if *flakeDBPath != "" {
		if err := db.Save(*flakeDBPath); err != nil {
			fatal("saving flake db: %v", err)
		}
	} else {
		if err := db.SaveToStore(stateStore, "flake-db"); err != nil {
			fatal("saving flake db to state store: %v", err)
		}
	}

	quarantined := reactor.GetQuarantinedTests()
	fmt.Printf("Flake DB: %d records, %d quarantined\n", len(db.Records), len(quarantined))
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
