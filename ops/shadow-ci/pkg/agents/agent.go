package agents

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/events"
	"github.com/ethereum-optimism/optimism/ops/shadow-ci/pkg/model"
)

// Agent is the interface that all shadow CI agents implement.
type Agent interface {
	Name() string
	SubscribedEvents() []model.EventType
	Handle(event model.Event, store events.Store) error
}

// Runner processes events and dispatches them to subscribed agents.
type Runner struct {
	agents []Agent
	store  events.Store
}

// NewRunner creates an agent runner.
func NewRunner(store events.Store, agents ...Agent) *Runner {
	return &Runner{
		agents: agents,
		store:  store,
	}
}

// ProcessEvents queries recent events and dispatches them to agents.
func (r *Runner) ProcessEvents(since time.Time) error {
	for _, agent := range r.agents {
		evts, err := r.store.Query(events.EventFilter{
			Types: agent.SubscribedEvents(),
			After: since,
		})
		if err != nil {
			return fmt.Errorf("querying events for agent %s: %w", agent.Name(), err)
		}

		for _, evt := range evts {
			if err := agent.Handle(evt, r.store); err != nil {
				fmt.Printf("agent %s error handling %s: %v\n", agent.Name(), evt.Type, err)
			}
		}
	}
	return nil
}

// FlakeInvestigator identifies and escalates recurring flakes.
type FlakeInvestigator struct {
	threshold int // minimum occurrences in lookback window to escalate
	lookback  time.Duration
}

// NewFlakeInvestigator creates a FlakeInvestigator.
func NewFlakeInvestigator(threshold int, lookback time.Duration) *FlakeInvestigator {
	return &FlakeInvestigator{threshold: threshold, lookback: lookback}
}

func (f *FlakeInvestigator) Name() string { return "flake-investigator" }
func (f *FlakeInvestigator) SubscribedEvents() []model.EventType {
	return []model.EventType{model.EventFlakeDetected}
}

func (f *FlakeInvestigator) Handle(event model.Event, store events.Store) error {
	var payload model.FlakePayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	// Query how many times this fingerprint has appeared recently.
	since := time.Now().Add(-f.lookback)
	fp := payload.Fingerprint

	recentFlakes, err := store.Query(events.EventFilter{
		Types: []model.EventType{model.EventFlakeDetected},
		After: since,
	})
	if err != nil {
		return err
	}

	count := 0
	for _, evt := range recentFlakes {
		var p model.FlakePayload
		json.Unmarshal(evt.Payload, &p)
		if p.Fingerprint == fp {
			count++
		}
	}

	if count >= f.threshold {
		// Emit escalation event.
		emitter := events.NewEmitter(store, event.PipelineID, event.PR, event.Commit, event.Branch)
		emitter.Emit(model.EventGraphGap, map[string]any{
			"type":        "flake_escalation",
			"fingerprint": fp,
			"count":       count,
			"test":        payload.Result.Test.Key(),
			"message":     fmt.Sprintf("flake %s has occurred %d times in %s — needs investigation", fp, count, f.lookback),
		})
	}

	return nil
}

// GraphMaintainer responds to false negatives by updating the graph.
type GraphMaintainer struct{}

func NewGraphMaintainer() *GraphMaintainer { return &GraphMaintainer{} }

func (g *GraphMaintainer) Name() string { return "graph-maintainer" }
func (g *GraphMaintainer) SubscribedEvents() []model.EventType {
	return []model.EventType{model.EventFalseNegative}
}

func (g *GraphMaintainer) Handle(event model.Event, store events.Store) error {
	var detail model.FalseNegativeDetail
	if err := json.Unmarshal(event.Payload, &detail); err != nil {
		return err
	}

	// Emit a graph update recommendation event.
	emitter := events.NewEmitter(store, event.PipelineID, event.PR, event.Commit, event.Branch)
	emitter.Emit(model.EventGraphUpdated, map[string]any{
		"type":     "always_run_addition",
		"test":     detail.Test.Key(),
		"language": detail.Language,
		"reason":   detail.MissedBecause,
		"action":   fmt.Sprintf("add %s/%s to always-run list for %s", detail.Test.Package, detail.Test.Name, detail.Language),
	})

	return nil
}

// ConfigVerifier checks test plans for suspicious patterns.
type ConfigVerifier struct{}

func NewConfigVerifier() *ConfigVerifier { return &ConfigVerifier{} }

func (c *ConfigVerifier) Name() string { return "config-verifier" }
func (c *ConfigVerifier) SubscribedEvents() []model.EventType {
	return []model.EventType{model.EventPlanCreated}
}

func (c *ConfigVerifier) Handle(event model.Event, store events.Store) error {
	var plan model.TestPlan
	if err := json.Unmarshal(event.Payload, &plan); err != nil {
		return err
	}

	emitter := events.NewEmitter(store, event.PipelineID, event.PR, event.Commit, event.Branch)

	// Check: Solidity files changed but no Solidity jobs?
	hasSolFiles := false
	hasSolJob := false
	for _, f := range plan.ChangedFiles {
		if len(f) > 4 && f[len(f)-4:] == ".sol" {
			hasSolFiles = true
			break
		}
	}
	for _, job := range plan.Jobs {
		if job.Language == "sol" {
			hasSolJob = true
			break
		}
	}
	if hasSolFiles && !hasSolJob {
		emitter.Emit(model.EventGraphGap, map[string]any{
			"type":    "plan_suspicious",
			"plan_id": plan.ID,
			"message": "Solidity files changed but no Solidity jobs in plan",
		})
	}

	// Check: too many targets with low parallelism?
	for _, job := range plan.Jobs {
		if len(job.Targets) > 100 && job.Resources.Parallelism < 2 {
			emitter.Emit(model.EventGraphGap, map[string]any{
				"type":    "plan_suspicious",
				"plan_id": plan.ID,
				"job":     job.Name,
				"message": fmt.Sprintf("job %s has %d targets but parallelism=%d", job.Name, len(job.Targets), job.Resources.Parallelism),
			})
		}
	}

	return nil
}

// ReportAnalyst produces human-readable analysis from weekly reports.
type ReportAnalyst struct{}

func NewReportAnalyst() *ReportAnalyst { return &ReportAnalyst{} }

func (r *ReportAnalyst) Name() string { return "report-analyst" }
func (r *ReportAnalyst) SubscribedEvents() []model.EventType {
	return []model.EventType{model.EventWeeklyReport}
}

func (r *ReportAnalyst) Handle(event model.Event, store events.Store) error {
	// The report analyst would generate human-readable summaries.
	// In a full implementation, this would post to Slack or write to a file.
	// For now, emit an event with the analysis.
	emitter := events.NewEmitter(store, event.PipelineID, event.PR, event.Commit, event.Branch)
	emitter.Emit(model.EventGraphUpdated, map[string]any{
		"type":    "report_analysis",
		"message": "Weekly report processed by report-analyst agent",
	})
	return nil
}
