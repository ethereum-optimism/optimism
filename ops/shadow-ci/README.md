# Shadow CI

A parallel CI system that proves itself against the existing CI before replacing it.

## Architecture

```
Layer 5: Agents           — Flake Investigator, Graph Maintainer, Config Verifier, Report Analyst
Layer 4: Data Layer       — Unified Event Store, Aggregator, Dashboard
Layer 3: Platform Adapter — CircleCI (renders TestPlans to YAML, fetches results via API)
Layer 2: Core Engine      — AffectedComputer, Planner, Executor, Classifier, Fingerprinter, ComparisonEngine
Layer 1: Language Adapters — Go (go list), Solidity (import parsing), Rust (cargo metadata)
```

## CLI Tools

| Binary | Purpose | When |
|--------|---------|------|
| `affected` | Compute dependency graphs, output affected targets | Setup phase |
| `planner` | Take affected targets, produce a test plan | Setup phase |
| `render` | Take test plan, produce CircleCI YAML | Setup phase |
| `runner` | Wrap test execution with retry + classification | Test jobs |
| `compare` | Compare shadow vs main CI, emit events | After all test jobs |
| `aggregate` | Read events, produce reports + dashboard | Daily cron |

## Observability

Every component emits structured events to the unified event store. Events are the
primary observability mechanism — they answer:

- **Is it catching everything?** `comparison.complete` events track catch rate per pipeline.
  A false negative emits `false_negative.detected` AND `graph.gap_detected`.
- **Is it faster?** Every `comparison.complete` includes wall time and compute reduction.
- **Are flakes being handled?** `flake.detected` events include fingerprints for clustering.
  The aggregator builds a flake leaderboard ranked by frequency.
- **Is the graph accurate?** `targets.computed` events show skip rate per language.
  `confidence.changed` tracks graph confidence over time.
- **What did each pipeline do?** `plan.created`, `job.started`, `job.completed`,
  `test.passed`, `test.failed`, `test.retried` — full audit trail.

### Key Metrics (from events)

| Metric | Source Event | Target |
|--------|-------------|--------|
| Catch rate | `comparison.complete` | 100% always |
| False negatives | `false_negative.detected` | 0 per month |
| Flake rate | `flake.detected` / total tests | < 0.5% |
| Skip rate (Go) | `targets.computed` | > 90% |
| Skip rate (Sol) | `targets.computed` | > 70% |
| PR feedback p50 | `job.completed` timestamps | < 5 min |
| PR feedback p95 | `job.completed` timestamps | < 10 min |
| Compute reduction | `comparison.complete` | > 70% |

## Configuration

All config in `config/`:

- `adapters.yaml` — Language adapter settings (paths, features, special paths)
- `scoping.yaml` — Always-run lists, confidence thresholds, activation controls
- `platform.yaml` — CircleCI runner mapping, concurrency limits, event store

## Activation Phases

The architecture is complete. Activation is controlled by `scoping.yaml`:

```yaml
activation:
  mode: shadow  # shadow → belt-and-suspenders → primary
  languages:
    go: true
    sol: true
    rust: true
  agents:
    flake_investigator: true
    graph_maintainer: true
  comparison:
    required: false  # true in belt-and-suspenders mode
```

## Development

```bash
cd ops/shadow-ci
go test ./...     # run all tests
go build ./...    # build all packages
go vet ./...      # static analysis
```
