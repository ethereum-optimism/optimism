# Shadow CI — Gap Analysis

What works, what doesn't, and what to do about it.

Last updated: 2026-03-10

## What Works End-to-End

These run in CI on every pipeline and produce real outcomes:

- **Decision engine** computes per-category run/skip from git diff + dependency graphs
- **Build caching** skips rebuilds when source inputs haven't changed (content-addressed)
- **Parallel executor** runs categories within each group with DAG scheduling
- **Artifact distribution** passes build outputs to downstream groups via workspace
- **Flake reactor** runs lifecycle state machine (but has no input data yet — see below)
- **Coherence checker** validates decision consistency
- **YAML staleness check** ensures generated pipeline matches scoping.yaml

## What's Computed But Unused

These components produce correct results that nothing consumes:

### 1. Graph-based test selection (HIGH PRIORITY)

**The problem:** `affected` computes which Go/Sol/Rust targets are affected and
writes them to `decision.json`. The downstream execute commands ignore this and
run the full test suite.

Example: decision.json says `go_tests.targets = ["op-node/...", "op-batcher/..."]`
but the command is `make go-tests-short-ci` which runs everything.

**To close the gap:**
- Executor needs to read `cd.Targets` for graph-based categories
- Build a target-aware command for each language:
  - Go: `gotestsum -- -short <packages...>` instead of `make go-tests-short-ci`
  - Sol: `forge test --match-path <files...>` instead of `forge test`
  - Rust: `cargo nextest run -p <crates...>` instead of full workspace
- The `runner` binary already does this — evaluate whether to wire it in or
  inline the target filtering into `execute`
- Start with Go (highest skip rate potential, >90% of packages unaffected on
  most PRs)

**Risk:** False negatives. If the dependency graph misses an edge, we skip a test
that should have run. Mitigation: run in shadow mode first, compare against full
suite, only activate after proving 100% catch rate.

### 2. Comparison engine (HIGH PRIORITY)

**The problem:** `compare` binary is built but never called. Without comparison,
we can't measure catch rate or detect false negatives — which means we can't
prove the graph is correct, which means we can't activate test selection.

**To close the gap:**
- Add a `shadow-ci-compare` job that runs after all group jobs complete
- It needs: shadow results (from execute) + mainline results (from CircleCI API)
- Wire it as: `requires: [shadow-ci-go, shadow-ci-sol, shadow-ci-rust, shadow-ci-misc]`
- Output: catch rate, false negatives, speedup metrics
- Store comparison results as CircleCI artifacts for visibility

**Dependency:** Test selection (#1) must be active for comparison to be meaningful.
Without it, shadow runs the same tests as mainline and catch rate is trivially 100%.

### 3. Event pipeline (MEDIUM PRIORITY)

**The problem:** The event store exists but is empty. Flake-reactor runs but finds
nothing. The placement optimizer needs historical data to make recommendations.

**To close the gap:**
- Executor needs to emit events for test results (pass/fail/flake/retry)
- This requires test-level granularity, which requires either:
  - Using `runner` binary (already emits events per test), or
  - Parsing test output in `execute` (gotestsum JSON, forge JSON)
- Once events flow, flake-reactor starts detecting real flakes
- After weeks of data, placement optimizer can suggest stage changes

**Dependency:** Test selection (#1) should come first — no point emitting events
for a system that runs everything anyway.

### 4. Dynamic pipeline continuation (LOW PRIORITY)

**The problem:** `affected --continue` can render a minimal CircleCI YAML with
only needed jobs, but the flag is never passed. The pipeline uses static YAML.

**To close the gap:**
- In shadow-ci-setup, pass `--continue` to affected
- Use CircleCI's continuation API to trigger the rendered YAML
- This would skip entire group jobs when nothing in that group is affected

**Why low priority:** Group-level skipping is less impactful than test-level
selection. If go_tests runs 47 packages instead of 4000, the group job is already
fast — skipping it entirely saves only the job overhead (~2 min).

## Activation Sequence

The gaps should be closed in this order:

```
1. Wire test selection for Go          — biggest skip rate, most CI time
   └── Shadow mode: run selected AND full, compare results
2. Add comparison job                  — proves catch rate, detects false negatives
3. Wire test selection for Sol          — second biggest skip rate
4. Add event emission                  — enables flake detection, optimizer
5. Wire test selection for Rust         — smallest benefit (fewer packages)
6. Dynamic pipeline continuation       — skip entire group jobs
7. Belt-and-suspenders mode            — shadow failures block merge
8. Primary mode                        — decommission mainline CI
```

Each step requires proving itself before the next activates. The comparison job
is the gate — nothing progresses past shadow mode without measured catch rate.

## Legacy Binaries

These exist but aren't used in the current pipeline:

| Binary | Status | Recommendation |
|--------|--------|----------------|
| `planner` | Replaced by scoping.yaml categories | Delete or repurpose |
| `render` | Replaced by generate-ci groups | Delete or repurpose |
| `runner` | Has useful test-level execution logic | Evaluate for #1 |
