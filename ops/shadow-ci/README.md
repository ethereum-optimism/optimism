# Shadow CI

An adaptive CI system that runs alongside the existing CI, proves itself by
comparing results, and progressively takes over as confidence grows.

## How It Works

Shadow CI replaces the monolithic "run everything" approach with a targeted one:

1. **Decide** — `affected` computes what changed, walks dependency graphs per language,
   and produces a per-category run/skip decision.
2. **Execute** — `execute` runs needed categories in parallel within each group,
   with DAG-based dependency scheduling and content-addressed build caching.
3. **Compare** — `compare` checks shadow results against mainline CI to measure
   catch rate and surface false negatives.
4. **Learn** — Flake reactor tracks flaky tests through a lifecycle (suspected →
   quarantined → fixed). Placement optimizer uses historical data to move tests
   to the cheapest stage that still catches failures.

## Architecture

```
Layer 5: Feedback Loop    — Comparison Engine, Placement Optimizer, Flake Reactor
Layer 4: Data Layer       — Unified Event Store, Aggregator
Layer 3: Execution        — Parallel Executor (DAG scheduling), Build Cache (content-addressed)
Layer 2: Decision Engine  — Affected Computer, Scoping Rules, Stage Placement
Layer 1: Language Adapters — Go (go list), Solidity (import parsing), Rust (cargo metadata)
```

### Why groups, not per-target jobs

The original design (planner → render → runner) generated a CircleCI job per test
target. This hit practical limits: CircleCI has continuation YAML size caps, each
job has fixed overhead (environment spin-up, checkout, workspace attach), and a
pipeline with hundreds of tiny jobs is slower than a few large ones.

The current design uses **static group jobs** (build, go, sol, rust, misc) checked
into the repo. Each group job runs `execute`, which handles per-category parallelism
internally via goroutines and channel-based dependency resolution. CircleCI manages
8 jobs; the executor manages the categories within each.

The planner/render/runner binaries still exist for potential future use (e.g., if
CircleCI raises continuation limits or we move to a platform with cheaper job overhead).

## CI Pipeline

```
shadow-ci-setup          Build binaries, compute decision, update flake state
     │
     ├── shadow-ci-verify    Coherence check against mainline
     ├── shadow-ci-tests     Unit tests + YAML staleness check
     │
     └── shadow-ci-build     Build categories (contracts, cannon, go binaries, rust)
              │                 Content-addressed cache: restore → verify → use or rebuild
              │
              ├── shadow-ci-go      Go tests, fuzz, lint, acceptance tests
              ├── shadow-ci-sol     Solidity tests, upgrades, checks
              ├── shadow-ci-rust    Rust CI, e2e tests
              └── shadow-ci-misc   Semgrep, shellcheck, TODOs
```

Build artifacts pass downstream via CircleCI workspace. Build cache persists
across pipelines via CircleCI `save_cache`/`restore_cache`.

## CLI Tools

| Binary | Purpose | Pipeline Phase |
|--------|---------|----------------|
| `affected` | Dependency graphs → per-category run/skip decision | Setup |
| `execute` | Run categories for a group with DAG scheduling + caching | Group jobs |
| `flake-reactor` | Update flake state from recent events | Setup |
| `coherence` | Verify decision is consistent with mainline CI | Verify |
| `compare` | Compare shadow vs mainline results, measure catch rate | Post-test |
| `generate-ci` | Render shadow-ci.yml from scoping.yaml | Development |
| `validate` | Validate generated CircleCI YAML for structural errors | Development |
| `optimize` | Suggest stage placement changes from historical data | Periodic |
| `aggregate` | Read events, produce reports | Periodic |
| `auto-revert` | Evaluate whether to revert based on test failures | Future |

Legacy binaries (exist but not used in current pipeline):

| Binary | Original Purpose | Why Unused |
|--------|-----------------|------------|
| `planner` | Affected targets → test plan | Replaced by scoping.yaml categories |
| `render` | Test plan → per-target CircleCI YAML | Replaced by generate-ci groups |
| `runner` | Wrap single test with retry + classification | Replaced by execute |

## Build Cache

The build group uses content-addressed caching to skip rebuilds when inputs
haven't changed:

1. **Key**: `sha256(git tree hashes of cache_inputs + mise.toml)`
2. **Restore**: Copy cached artifacts from `/tmp/shadow-ci-cache` to repo
3. **Verify**: Run `verify_command` to confirm artifacts are valid post-restore
4. **Use or rebuild**: If verify passes, skip build. If not, full rebuild + save.

Each build category in `scoping.yaml` declares:
- `cache_inputs` — source paths that determine the build output
- `workspace_paths` — output paths to cache
- `verify_command` — cheap check that cached outputs are valid

The system is fail-open: any error falls through to a full build.

## Observability

Every component emits structured events to the unified event store:

- **Catch rate**: `comparison.complete` — are we catching everything mainline catches?
- **False negatives**: `false_negative.detected` — tests we missed
- **Flake lifecycle**: `flake.detected`, `flake.quarantined`, `flake.restored`
- **Cache effectiveness**: `cache.hit`, `cache.miss`, `cache.verify_failed`
- **Skip rate**: `targets.computed` — how much work are we avoiding?
- **Pipeline audit**: `plan.created`, `job.started`, `job.completed`

### Key Metrics

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

- `scoping.yaml` — Job categories, trigger paths, always-run lists, activation mode
- `adapters.yaml` — Language adapter settings (paths, features, special paths)
- `platform.yaml` — CircleCI runner mapping, concurrency limits, event store

## Activation Phases

Shadow CI progresses through three modes:

| Mode | Behavior | Exit Criteria |
|------|----------|---------------|
| **shadow** | Runs alongside mainline, compares results, no gate | 100% catch rate for 4 weeks |
| **belt-and-suspenders** | Both run, shadow failures block merge | 0 false negatives for 4 weeks |
| **primary** | Shadow becomes the CI, mainline decommissioned | — |

Current mode is controlled in `scoping.yaml`:

```yaml
activation:
  mode: shadow
  languages:
    go: true
    sol: true
    rust: true
```

## Development

```bash
cd ops/shadow-ci
go test ./...     # run all tests
go build ./...    # build all packages
go vet ./...      # static analysis

# Regenerate CI YAML after config changes:
go run ./cmd/generate-ci --config config --output ../../.circleci/continue/shadow-ci.yml

# Check YAML is not stale:
go run ./cmd/generate-ci --config config --check ../../.circleci/continue/shadow-ci.yml
```
